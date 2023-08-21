package elasticsearch

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"

	v2 "github.com/elastic/go-lumber/client/v2"
)

func (q *LumberjackConsumer) tlsDial(addr string, caFilename string, certFilename string, keyFilename string) (net.Conn, error) {
	// Load our TLS key pair to use for authentication
	cert, err := tls.LoadX509KeyPair(certFilename, keyFilename)
	if err != nil {
		log.Fatal(q.ctx, "Unable to load cert", certFilename, keyFilename, err)
	}

	// Load our CA certificate
	clientCertPool := x509.NewCertPool()
	insecureSkipVerify := true
	if caFilename != "" {
		clientCACert, err := ioutil.ReadFile(caFilename)
		if err != nil {
			log.Fatal(q.ctx, "Unable to open ca", caFilename, err)
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
		log.Errorln(q.ctx, "client: dial:", err)
		return nil, err
	}

	log.Println(q.ctx, "client: connected to: ", conn.RemoteAddr())
	state := conn.ConnectionState()
	for i, cert := range state.PeerCertificates {
		subject := cert.Subject
		issuer := cert.Issuer

		log.Printf(q.ctx, "client: %d s:/C=%s/ST=%v/L=%v/O=%v/OU=%v/CN=%s", i, strings.Join(subject.Country, ","), strings.Join(subject.Province, ","), strings.Join(subject.Locality, ","), strings.Join(subject.Organization, ","), strings.Join(subject.OrganizationalUnit, ","), subject.CommonName)
		log.Printf(q.ctx, "client:   i:/C=%s/ST=%v/L=%v/O=%v/OU=%v/CN=%s", strings.Join(issuer.Country, ","), strings.Join(issuer.Province, ","), strings.Join(issuer.Locality, ","), strings.Join(issuer.Organization, ","), strings.Join(issuer.OrganizationalUnit, ","), issuer.CommonName)

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
	log.Println(q.ctx, "client: handshake: ", state.HandshakeComplete)
	log.Println(q.ctx, "client: mutual: ", state.NegotiatedProtocolIsMutual)

	return conn, nil
}

type LumberjackConsumerConf struct {
	addr, caFilename, certFilename, keyFilename string
}

type LumberjackConsumer struct {
	ctx    string
	client *v2.Client
}

func (conf *LumberjackConsumerConf) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, Queue chan processor.AckableEvent, out chan processor.AckableEvent) {
	var q LumberjackConsumer
	q.ctx = "[LJ] " + p.Flow.Name
	log.Println(q.ctx, "Initializing to ", conf.addr, conf.caFilename, conf.certFilename, conf.keyFilename)
	for {
		conn, err := q.lumberJackConnect(conf.addr, conf.caFilename, conf.certFilename, conf.keyFilename)
		if err == nil {
			err := q.lumberJackSend(ctx, conn, Queue)
			if err == nil {
				return
			}
		}
		log.Println(q.ctx, "Sleep...", 10)
		time.Sleep(10 * time.Second)
	}
}

func (q *LumberjackConsumer) lumberJackConnect(addr, caFilename, certFilename, keyFilename string) (net.Conn, error) {
	if caFilename != "" || certFilename != "" || keyFilename != "" {
		conn, err := q.tlsDial(addr, caFilename, certFilename, keyFilename)
		if err != nil {
			log.Errorln(q.ctx, "Error establishing TLS connection to", addr, err)
			return nil, err
		}
		return conn, nil
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Errorln(q.ctx, "Error establishing connection to", addr, err)
		return nil, err
	}
	return conn, nil
}

func (q *LumberjackConsumer) processEvent(event *processor.AckableEvent) error {
	log.Println(q.ctx, "Got Message...")

	log.Println(q.ctx, "Sending Message...")
	count := 1

	m := make(map[string]interface{})
	fields := make(map[string]interface{})
	fields["axway-target-flow"] = "api-central-v8" // Condor
	fields["captureOrgID"] = "jda"                 // tenant

	m["fields"] = fields
	m["message"] = processor.ConvertToJSON(event.Msg.(map[string]string))

	messages := []interface{}{m}

	msgs, _ := json.Marshal(messages)
	log.Println(q.ctx, "Message", string(msgs))
	err := q.client.Send(messages)
	if err != nil {
		log.Errorln(q.ctx, "Error sending messages", count, err)
		return err
	}

	log.Println(q.ctx, "Waiting Ack..")
	acked, err := q.client.AwaitACK(1)
	if err != nil {
		log.Errorln(q.ctx, "Error awaiting ack", count, err)
		return err
	}
	log.Println(q.ctx, "Acked", acked)
	return nil
}

func (q *LumberjackConsumer) lumberJackSend(ctx context.Context, conn net.Conn, Queue chan processor.AckableEvent) error {
	client, err := v2.NewWithConn(conn)
	if err != nil {
		log.Errorln(q.ctx, "Error opening lumberjack connection to", err)
		return err
	}

	q.client = client

	done := ctx.Done()

	for {
		log.Println(q.ctx, "Waiting Message on Queue...")
		select {
		case event := <-Queue:
			err := q.processEvent(&event)
			if err != nil {
				return err
			}
		case <-done:
			log.Infoln(q.ctx, "done")
			return nil
		}
	}
}
