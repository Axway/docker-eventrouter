package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
)

var counter = new(int64)

func getSessionId() string {
	val := atomic.AddInt64(counter, 1)
	return fmt.Sprint(val)
}

func tcpConnect(addr string, prefix string) (net.Conn, string, error) {
	ctx := fmt.Sprint("["+prefix+"-", getSessionId(), "]")

	log.Println(prefix+"- Dialing...", addr)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Errorln(prefix, "- Dial failed :", addr, err)
		return nil, "", err
	}
	return conn, ctx, nil
}

func tlsConnect(addr string, caFilename string, certFilename string, keyFilename string, prefix string) (net.Conn, string, error) {
	ctx := fmt.Sprint("["+prefix+"-", getSessionId(), "]")

	// Load our TLS key pair to use for authentication
	cert, err := tls.LoadX509KeyPair(certFilename, keyFilename)
	if err != nil {
		log.Fatal(prefix+" Unable to load cert", certFilename, keyFilename, err)
	}

	// Load our CA certificate
	clientCertPool := x509.NewCertPool()
	insecureSkipVerify := true
	if caFilename != "" {
		clientCACert, err := ioutil.ReadFile(caFilename)
		if err != nil {
			log.Fatal(prefix+"Unable to open ca", caFilename, err)
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
		log.Errorln(ctx+"client: dial:", err)
		return nil, "", err
	}

	log.Println(ctx+"client: connected to: ", conn.RemoteAddr())
	state := conn.ConnectionState()
	for i, cert := range state.PeerCertificates {
		subject := cert.Subject
		issuer := cert.Issuer

		log.Printf(ctx+"client: %d s:/C=%s/ST=%v/L=%v/O=%v/OU=%v/CN=%s", i, strings.Join(subject.Country, ","), strings.Join(subject.Province, ","), strings.Join(subject.Locality, ","), strings.Join(subject.Organization, ","), strings.Join(subject.OrganizationalUnit, ","), subject.CommonName)
		log.Printf(ctx+"client:   i:/C=%s/ST=%v/L=%v/O=%v/OU=%v/CN=%s", strings.Join(issuer.Country, ","), strings.Join(issuer.Province, ","), strings.Join(issuer.Locality, ","), strings.Join(issuer.Organization, ","), strings.Join(issuer.OrganizationalUnit, ","), issuer.CommonName)

		der, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
		if err != nil {
			continue
		}
		block := pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: der,
		}
		p := pem.EncodeToMemory(&block)
		log.Debugln("\n" + string(p))
	}
	log.Println(ctx+"client: handshake: ", state.HandshakeComplete)
	log.Println(ctx+"client: mutual: ", state.NegotiatedProtocolIsMutual)

	return conn, ctx, nil
}

func tcpServe(addr string, handleRequest func(net.Conn, string, interface{}), prefix string, data interface{}) {
	ctxInit := "[" + prefix + "]"
	// Listen for incoming connections.
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(ctxInit+" Error listening:", err.Error())
	}
	// Close the listener when the application closes.

	log.Println(ctxInit + " Listening on " + addr)
	// Detach the main accept loop
	go func() {
		defer l.Close()
		for {
			// Listen for an incoming connection.
			conn, err := l.Accept()
			if err != nil {
				log.Fatal(ctxInit+" Error accepting: ", err.Error())
			}
			ctx := fmt.Sprint("["+prefix+"-", getSessionId(), "]")
			// Handle connections in a new goroutine
			go handleRequest(conn, ctx, data)
		}

	}()
}

func tlsLogInfo(tlscon *tls.Conn, ctx string) {

	log.Print(ctx + " TLS - Server: conn: Handshake completed")
	state := tlscon.ConnectionState()

	log.Printf(ctx+" TLS - Server: Version %x", state.Version)
	log.Printf(ctx+" TLS - Server: HandshakeComplete: %t", state.HandshakeComplete)
	log.Printf(ctx+" TLS - Server: NegotiatedProtocol: %s", state.NegotiatedProtocol)
	log.Printf(ctx+" TLS - Server: NegotiatedProtocolIsMutual %t ", state.NegotiatedProtocolIsMutual)
	log.Printf(ctx+" TLS - Server: ServerName: %s", state.ServerName)
	log.Printf(ctx+" TLS - Server: CipherSuite %x", state.CipherSuite)
	log.Println(ctx+" TLS - Server: OCSPResponse", state.OCSPResponse)
	log.Println(ctx + " TLS - Server: client public key is:")
	for i, cert := range state.PeerCertificates {
		subject := cert.Subject
		issuer := cert.Issuer

		log.Printf(ctx+" TLS - Server: %d s:/C=%s/ST=%v/L=%v/O=%v/OU=%v/CN=%s", i, strings.Join(subject.Country, ","), strings.Join(subject.Province, ","), strings.Join(subject.Locality, ","), strings.Join(subject.Organization, ","), strings.Join(subject.OrganizationalUnit, ","), subject.CommonName)
		log.Printf(ctx+" TLS - Server:   i:/C=%s/ST=%v/L=%v/O=%v/OU=%v/CN=%s", strings.Join(issuer.Country, ","), strings.Join(issuer.Province, ","), strings.Join(issuer.Locality, ","), strings.Join(issuer.Organization, ","), strings.Join(issuer.OrganizationalUnit, ","), issuer.CommonName)

		der, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
		if err != nil {
			continue
		}
		block := pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: der,
		}
		p := pem.EncodeToMemory(&block)
		log.Debugln("\n" + string(p))
	}
}

func tlsServe(
	addr string,
	certFilename string,
	keyFilename string,
	caFilename string,
	handleRequest func(net.Conn, string, interface{}),
	prefix string,
	data interface{}) {
	ctxInit := "[" + prefix + "]"

	// Listen for incoming connections.
	cert, err := tls.LoadX509KeyPair(certFilename, keyFilename)
	if err != nil {
		log.Fatalf(ctxInit+" TLS - loadkeys: %s", err)
	}

	var certpool *x509.CertPool
	clientAuth := tls.RequireAnyClientCert
	if caFilename != "" {
		clientAuth = tls.RequireAndVerifyClientCert
		certpool = x509.NewCertPool()
		pem, err := ioutil.ReadFile(caFilename)
		if err != nil {
			log.Fatalf(ctxInit+" TLS - Error - Failed to read client certificate authority: %v", err)
		}
		if !certpool.AppendCertsFromPEM(pem) {
			log.Fatalf(ctxInit + " TLS - Error - Can't parse client certificate authority")
		}
	}

	config := tls.Config{
		Certificates: []tls.Certificate{cert},
		//ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientAuth: clientAuth,
		ClientCAs:  certpool,
	}

	l, err := tls.Listen("tcp", addr, &config)
	if err != nil {
		log.Fatal(ctxInit+" TLS - Error listening tls://"+addr, err.Error())
	}
	// Close the listener when the application closes.
	log.Println(ctxInit + " TLS - listening on tls://" + addr)
	go func() {
		defer l.Close()
		for {
			// Listen for an incoming connection.
			conn, err := l.Accept()
			if err != nil {
				log.Println(ctxInit+" TLS - Server: Error accepting: ", err.Error())
				continue
			}
			// Handle connections in a new goroutine.
			go func() {
				ctx := fmt.Sprint("["+prefix+"-", getSessionId(), "]")
				tlsconn, ok := conn.(*tls.Conn)
				if ok {
					//log.Print(ctx + " TLS - Server: conn: type assert to TLS succeedded")
					err := tlsconn.Handshake()
					if err != nil {
						log.Errorf(ctx+" TLS - Server: handshake failed: %s", err)
						return
					}

					tlsLogInfo(tlsconn, ctx)

					handleRequest(conn, ctx, data)
				} else {
					log.Println(ctx + " TLS - server: conn: closed")
				}
			}()
		}
	}()
}
