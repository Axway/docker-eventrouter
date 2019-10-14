package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io/ioutil"
	"net"
	"strings"
	"time"

	v2 "github.com/elastic/go-lumber/client/v2"
	log "github.com/sirupsen/logrus"
)

func tlsDial(addr string, caFilename string, certFilename string, keyFilename string) (net.Conn, error) {
	// Load our TLS key pair to use for authentication
	cert, err := tls.LoadX509KeyPair(certFilename, keyFilename)
	if err != nil {
		log.Fatal("[LJ] Unable to load cert", certFilename, keyFilename, err)
	}

	// Load our CA certificate
	clientCertPool := x509.NewCertPool()
	insecureSkipVerify := true
	if caFilename != "" {
		clientCACert, err := ioutil.ReadFile(caFilename)
		if err != nil {
			log.Fatal("[LJ] Unable to open ca", caFilename, err)
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
		log.Errorln("[LJ] client: dial:", err)
		return nil, err
	}

	log.Println("[LJ] client: connected to: ", conn.RemoteAddr())
	state := conn.ConnectionState()
	for i, cert := range state.PeerCertificates {
		subject := cert.Subject
		issuer := cert.Issuer

		log.Printf("[LJ] client: %d s:/C=%s/ST=%v/L=%v/O=%v/OU=%v/CN=%s", i, strings.Join(subject.Country, ","), strings.Join(subject.Province, ","), strings.Join(subject.Locality, ","), strings.Join(subject.Organization, ","), strings.Join(subject.OrganizationalUnit, ","), subject.CommonName)
		log.Printf("[LJ] client:   i:/C=%s/ST=%v/L=%v/O=%v/OU=%v/CN=%s", strings.Join(issuer.Country, ","), strings.Join(issuer.Province, ","), strings.Join(issuer.Locality, ","), strings.Join(issuer.Organization, ","), strings.Join(issuer.OrganizationalUnit, ","), issuer.CommonName)

		der, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
		if err != nil {
			continue
		}
		block := pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: der,
		}
		p := pem.EncodeToMemory(&block)
		log.Println("\n" + string(p))
	}
	log.Println("[LJ] client: handshake: ", state.HandshakeComplete)
	log.Println("[LJ] client: mutual: ", state.NegotiatedProtocolIsMutual)

	return conn, nil
}

func lumberJackInit(addr, caFilename, certFilename, keyFilename string, Queue chan QLTMessage) {
	log.Println("[LJ] Initializing to ", addr, caFilename, certFilename, keyFilename)
	for {
		client, err := lumberJackConnect(addr, caFilename, certFilename, keyFilename)
		if err == nil {
			lumberJackSend(client, Queue)
		}
		log.Println("[LJ] Sleep...", 10)
		time.Sleep(10 * time.Second)
	}
}

func lumberJackConnect(addr, caFilename, certFilename, keyFilename string) (net.Conn, error) {
	if caFilename != "" || certFilename != "" || keyFilename != "" {
		conn, err := tlsDial(addr, caFilename, certFilename, keyFilename)
		if err != nil {
			log.Errorln("[LJ] Error establishing TLS connection to", addr, err)
			return nil, err
		}
		return conn, nil
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Errorln("[LJ] Error establishing connection to", addr, err)
		return nil, err
	}
	return conn, nil
}

func lumberJackSend(conn net.Conn, Queue chan QLTMessage) error {
	client, err := v2.NewWithConn(conn)
	if err != nil {
		log.Errorln("[LJ] Error opening lumberjack connection to", err)
		return err
	}

	for {
		log.Println("[LJ] Waiting Message on Queue...")
		event := <-Queue
		log.Println("[LJ] Got Message...")

		log.Println("[LJ] Sending Message...")
		count := 1

		m := make(map[string]interface{})
		fields := make(map[string]interface{})
		fields["axway-target-flow"] = "api-central-v8" // Condor
		fields["captureOrgID"] = "jda"                 // tenant

		m["fields"] = fields
		m["message"] = convertToJSON(event.Fields)

		messages := []interface{}{m}

		msgs, _ := json.Marshal(messages)
		log.Println("[LJ] Message", string(msgs))
		err := client.Send(messages)
		if err != nil {
			log.Errorln("[LJ] Error sending messages", count, err)
			return err
		}

		log.Println("[LJ] Waiting Ack..")
		acked, err := client.AwaitACK(1)
		if err != nil {
			log.Errorln("[LJ] Error awaiting ack", count, err)
			return err
		}
		log.Println("[LJ] Acked", acked)
	}
}
