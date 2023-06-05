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

	"axway.com/qlt-router/src/log"
)

var counter = new(int64)

func getSessionId() string {
	val := atomic.AddInt64(counter, 1)
	return fmt.Sprint(val)
}

func TcpConnect(addr string, prefix string) (net.Conn, string, error) {
	ctx := fmt.Sprint("["+prefix+"-", getSessionId(), "]")

	log.Infoc(prefix, "Dialing...", "addr", addr)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Errorc(prefix, "Dial failed :", "addr", addr, "err", err)
		return nil, "", err
	}
	return conn, ctx, nil
}

func TlsConnect(addr string, caFilename string, certFilename string, keyFilename string, prefix string) (net.Conn, string, error) {
	ctx := fmt.Sprint("["+prefix+"-", getSessionId(), "]")

	// Load our TLS key pair to use for authentication
	cert, err := tls.LoadX509KeyPair(certFilename, keyFilename)
	if err != nil {
		log.Fatalc(prefix, " Unable to load cert", "certFilename", certFilename, "keyFilename", keyFilename, "err", err)
	}

	// Load our CA certificate
	clientCertPool := x509.NewCertPool()
	insecureSkipVerify := true
	if caFilename != "" {
		clientCACert, err := ioutil.ReadFile(caFilename)
		if err != nil {
			log.Fatalc(prefix, "Unable to open ca", "caFilename", caFilename, "err", err)
		}
		insecureSkipVerify = false
		clientCertPool.AppendCertsFromPEM(clientCACert)
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            clientCertPool,
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
	log.Infoc(ctx, "client: handshake: ", "handshake", state.HandshakeComplete, "mutual", state.NegotiatedProtocolIsMutual)

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
	log.Infoc(ctx, " TLS - Server: conn: Handshake completed")
	state := tlscon.ConnectionState()

	log.Infoc(ctx, "TLS - Server info",
		"version", state.Version,
		"serverName", state.ServerName,
		"CipherSuite", state.CipherSuite,
		"OCSPResponse", state.OCSPResponse,
		"handshakecomplete", state.HandshakeComplete,
		"negotiatedProtocol", state.NegotiatedProtocol,
		"negotiatedProtocolIsMutual", state.NegotiatedProtocolIsMutual,
	)
	log.Infoc(ctx, "TLS - Server: client public key is:")
	for i, cert := range state.PeerCertificates {
		subject := cert.Subject
		issuer := cert.Issuer

		log.Infoc(ctx, fmt.Sprintf(" TLS - Server: %d s:/C=%s/ST=%v/L=%v/O=%v/OU=%v/CN=%s", i, strings.Join(subject.Country, ","), strings.Join(subject.Province, ","), strings.Join(subject.Locality, ","), strings.Join(subject.Organization, ","), strings.Join(subject.OrganizationalUnit, ","), subject.CommonName))
		log.Infoc(ctx, fmt.Sprintf(" TLS - Server:   i:/C=%s/ST=%v/L=%v/O=%v/OU=%v/CN=%s", strings.Join(issuer.Country, ","), strings.Join(issuer.Province, ","), strings.Join(issuer.Locality, ","), strings.Join(issuer.Organization, ","), strings.Join(issuer.OrganizationalUnit, ","), issuer.CommonName))

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
	caFilename string,
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

	var certpool *x509.CertPool
	clientAuth := tls.RequireAnyClientCert
	if !requireClientAuth {
		clientAuth = tls.NoClientCert
	}
	if caFilename != "" {
		clientAuth = tls.RequireAndVerifyClientCert
		certpool = x509.NewCertPool()
		pem, err := ioutil.ReadFile(caFilename)
		if err != nil {
			log.Errorc(ctxInit, " TLS - Error - Failed to read client certificate authority", "err", err)
			return nil, err
		}
		if !certpool.AppendCertsFromPEM(pem) {
			log.Errorc(ctxInit, " TLS - Error - Can't parse client certificate authority", "err", err)
			return nil, err
		}
	}

	config := tls.Config{
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
