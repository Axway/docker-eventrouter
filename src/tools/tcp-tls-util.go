package tools

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"axway.com/qlt-router/src/log"
)

var counter = new(int64)

func getSessionId() string {
	val := atomic.AddInt64(counter, 1)
	return fmt.Sprint(val)
}

func TcpConnect(addr string, prefix string, timeout time.Duration) (net.Conn, string, error) {
	ctx := fmt.Sprint("["+prefix+"-", getSessionId(), "]")

	log.Infoc(prefix, "Dialing...", "addr", addr)

	var err error
	var conn net.Conn
	if timeout != 0 {
		conn, err = net.DialTimeout("tcp", addr, timeout)
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		log.Errorc(prefix, "Dial failed :", "addr", addr, "err", err)
		return nil, "", err
	}
	return conn, ctx, nil
}

func tlsCAConfig(ctx, caFilenames string) (clientCertPool *x509.CertPool, insecureSkipVerify bool, err error) {
	insecureSkipVerify = false
	if caFilenames == "-" {
		insecureSkipVerify = true
	} else if caFilenames != "" {
		clientCertPool = x509.NewCertPool()
		for _, caFilename := range strings.Split(caFilenames, ",") {
			clientCACert, err := ioutil.ReadFile(caFilename)
			if err != nil {
				log.Infoc(ctx, "Unable to open ca", caFilename, err)
				return nil, false, err
			}
			if !clientCertPool.AppendCertsFromPEM(clientCACert) {
				if err != nil {
					err = fmt.Errorf("cannot append '%s' to certpool", caFilename)
					return nil, false, err
				}
			}
		}
	}

	return clientCertPool, insecureSkipVerify, nil
}

func TlsConnect(
	addr string,
	caFilenames string,
	certFilename string,
	keyFilename string,
	prefix string,
) (net.Conn, string, error) {
	ctx := fmt.Sprint("["+prefix+"-", getSessionId(), "]")

	// Load our TLS key pair to use for mutual-authentication
	var certs []tls.Certificate
	if certFilename != "" {
		cert, err := tls.LoadX509KeyPair(certFilename, keyFilename)
		if err != nil {
			log.Fatalc(prefix, " Unable to load cert", "certFilename", certFilename, "keyFilename", keyFilename, "err", err)
		}
		certs = append(certs, cert)
	}

	// Load server CA certificates
	serverCertPool, insecureSkipVerify, err := tlsCAConfig(prefix, caFilenames)
	if err != nil {
		log.Fatalc(prefix, "Unable to open ca", caFilenames, err)
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12, // FIXME should we be able do force only 1.2 ? MaxVersion
		// Need to test if when we allow 1.3, can we runcheck with someone that doesn't understand 1.3, like ER?
		// when I allow 1.2, how do I runcheck with someone that uses 1.3? Will I only use 1.3?
		CipherSuites: []uint16{
			// TLS 1.2 cipher suites.
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			// TLS 1.3 cipher suites.
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
		},
		Certificates:       certs,
		RootCAs:            serverCertPool,
		InsecureSkipVerify: insecureSkipVerify,
	}

	tlsConfig.BuildNameToCertificate()

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		log.Errorc(ctx, "client: dial:", "err", err)
		return nil, "", err
	}

	log.Infoc(ctx, "client: connected to: ", "addr", conn.RemoteAddr())
	state := conn.ConnectionState()
	for i, cert := range state.PeerCertificates {
		subject := cert.Subject
		issuer := cert.Issuer

		log.Infoc(prefix, fmt.Sprintf("client: %d s:/C=%s/ST=%v/L=%v/O=%v/OU=%v/CN=%s", i, strings.Join(subject.Country, ","), strings.Join(subject.Province, ","), strings.Join(subject.Locality, ","), strings.Join(subject.Organization, ","), strings.Join(subject.OrganizationalUnit, ","), subject.CommonName))
		log.Infoc(prefix, fmt.Sprintf("client:   i:/C=%s/ST=%v/L=%v/O=%v/OU=%v/CN=%s", strings.Join(issuer.Country, ","), strings.Join(issuer.Province, ","), strings.Join(issuer.Locality, ","), strings.Join(issuer.Organization, ","), strings.Join(issuer.OrganizationalUnit, ","), issuer.CommonName))

		der, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
		if err != nil {
			continue
		}
		block := pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: der,
		}
		p := pem.EncodeToMemory(&block)
		log.Debugc(prefix, "pem", "data", string(p))
	}
	log.Infoc(ctx, "TLS - Client: conn: Handshake completed")

	log.Infoc(ctx, "TLS - Client info",
		"version", tls.VersionName(state.Version),
		"CipherSuite", state.CipherSuite,
		"CipherSuiteName", tls.CipherSuiteName(state.CipherSuite),
		"handshakecomplete", state.HandshakeComplete,
		"negotiatedProtocol", state.NegotiatedProtocol,
		"negotiatedProtocolIsMutual", state.NegotiatedProtocolIsMutual)

	return conn, ctx, nil
}

func TcpServe(addr string, handleRequest func(net.Conn, string), prefix string) (net.Listener, error) {
	ctxInit := "[" + prefix + "]"
	// Listen for incoming connections.
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Errorc(ctxInit, "error listening", "err", err)
		return nil, err
	}
	// Close the listener when the application closes.

	log.Infoc(ctxInit, "listening", "addr", addr)
	// Detach the main accept loop
	go func() {
		defer l.Close()
		for {
			// Listen for an incoming connection.
			conn, err := l.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				log.Errorc(ctxInit, "error accepting conenction", "err", err)
				return
			}
			ctx := fmt.Sprint("["+prefix+"-", getSessionId(), "]")
			// Handle connections in a new goroutine
			go handleRequest(conn, ctx)
		}
	}()
	return l, nil
}

func TlsLogInfo(tlscon *tls.Conn, ctx string) {
	log.Infoc(ctx, "TLS - Server: conn: Handshake completed")
	state := tlscon.ConnectionState()

	log.Infoc(ctx, "TLS - Server info",
		"version", tls.VersionName(state.Version),
		"serverName", state.ServerName,
		"CipherSuite", state.CipherSuite,
		"CipherSuiteName", tls.CipherSuiteName(state.CipherSuite),
		"OCSPResponse", state.OCSPResponse,
		"handshakecomplete", state.HandshakeComplete,
		"negotiatedProtocol", state.NegotiatedProtocol,
		"negotiatedProtocolIsMutual", state.NegotiatedProtocolIsMutual,
	)
	log.Infoc(ctx, "TLS - Server: client public key is:")
	for i, cert := range state.PeerCertificates {
		subject := cert.Subject
		issuer := cert.Issuer

		log.Infoc(ctx, fmt.Sprintf("TLS - Server: %d s:/C=%s/ST=%v/L=%v/O=%v/OU=%v/CN=%s", i, strings.Join(subject.Country, ","), strings.Join(subject.Province, ","), strings.Join(subject.Locality, ","), strings.Join(subject.Organization, ","), strings.Join(subject.OrganizationalUnit, ","), subject.CommonName))
		log.Infoc(ctx, fmt.Sprintf("TLS - Server:   i:/C=%s/ST=%v/L=%v/O=%v/OU=%v/CN=%s", strings.Join(issuer.Country, ","), strings.Join(issuer.Province, ","), strings.Join(issuer.Locality, ","), strings.Join(issuer.Organization, ","), strings.Join(issuer.OrganizationalUnit, ","), issuer.CommonName))

		der, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
		if err != nil {
			continue
		}
		block := pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: der,
		}
		p := pem.EncodeToMemory(&block)
		log.Debugc(ctx, "pem", "data", string(p))
	}
}

func TlsServe(
	addr string,
	certFilename string,
	keyFilename string,
	caFilenames string,
	handleRequest func(net.Conn, string),
	requireClientAuth bool,
	prefix string,
) (net.Listener, error) {
	ctxInit := "[" + prefix + "]"

	// Listen for incoming connections.
	cert, err := tls.LoadX509KeyPair(certFilename, keyFilename)
	if err != nil {
		log.Errorc(ctxInit, " TLS - loadkeys", "err", err)
		return nil, err

	}

	// Load server CA certificates
	certpool, _, err := tlsCAConfig(prefix, caFilenames)
	if err != nil {
		log.Fatalc(prefix, "Unable to open ca", caFilenames, err)
	}
	clientAuth := tls.RequireAnyClientCert
	if !requireClientAuth {
		clientAuth = tls.NoClientCert
	}

	config := tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			// TLS 1.2 cipher suites.
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			// TLS 1.3 cipher suites.
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
		},
		Certificates: []tls.Certificate{cert},
		// ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientAuth: clientAuth,
		ClientCAs:  certpool,
	}

	l, err := tls.Listen("tcp", addr, &config)
	if err != nil {
		log.Errorc(ctxInit, "TLS - Error listening", "addr", addr, "err", err)
		return nil, err
	}
	// Close the listener when the application closes.
	log.Infoc(ctxInit, "TLS - listening", "addr", addr)
	go func() {
		defer l.Close()
		for {
			// Listen for an incoming connection.
			conn, err := l.Accept()
			if err != nil {
				log.Infoc(ctxInit, "TLS - Server: Error accepting: ", "err", err)
				continue
			}
			// Handle connections in a new goroutine.
			go func() {
				ctx := fmt.Sprint("["+prefix+"-", getSessionId(), "]")
				tlsconn, ok := conn.(*tls.Conn)
				if ok {
					// log.Print(ctx + " TLS - Server: conn: type assert to TLS succeedded")
					err := tlsconn.Handshake()
					if err != nil {
						log.Errorc(ctx, "TLS - Server: handshake failed", "err", err)
						return
					}

					TlsLogInfo(tlsconn, ctx)

					handleRequest(conn, ctx)
				} else {
					log.Errorc(ctx, "TLS - server: conn: closed")
				}
			}()
		}
	}()
	return l, nil
}
