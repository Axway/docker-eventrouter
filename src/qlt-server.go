package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// QLT Handles incoming requests.
// QLT protocol
//   offset [0-1] length
//   offset [2] type
// 1
//  Data to be processed
//  Next bytes are the message
//  Message will be accepted (ACK) or rejected (NACK)
// 2
//   ACK = Positive Acknowledgement
//   Sender can pass at the next message or cut the connection.
// 3
//   NACK = Negative Acknowledgement
//   Message cannot be processed by the receiver
//   If sender  use an overflow file, the message will be send again at next connection
// 4
//   Connection Requested
//   Only for ER mode QLTREQ/QLTSRV
//   Data to follow are the Target Name (in ascii)
// 5
//   Connection Rejected
//   Only for ER mode QLRREQ/QLTSRV
//   Data to follow is composed
//   1 byte for Indication that connection retry will be  accepted (2) or not (3)
//   If 3, the connection requester must stop.
//   Raison explaining the error on two bytes
//   Text  explaining the cause the reject code.
// 6
//   Ask to stop sending message
//   Only for ER mode QLRREQ/QLTSRV
//   The server ask to stop
//   The requester will do the connection later, following its retry parameters
//   The message is the same that for Connection rejected except
//   Code = 11
//   String text contain “no more message“
type QLT struct {
	conn net.Conn
	buf  []byte
	idx  int
	ctx  string
	ch   []chan map[string]string
}

func (q *QLT) readData() error {
	log.Println(q.ctx, "Reading... idx=", q.idx)
	rsize, err := q.conn.Read(q.buf[q.idx:])
	if err != nil {
		log.Println(q.ctx, "Error reading:", err.Error(), "Closing...")
		return err
	}
	log.Println(q.ctx, "Read:", "idx=", q.idx, "rsize=", rsize, "size=", q.idx+rsize)
	q.idx = rsize + q.idx
	return nil
}

func (q *QLT) writeQLTAck() error {
	_, err := q.conn.Write([]byte{0, 1, '2'})
	if err != nil {
		log.Println(q.ctx, "Error writing:", err.Error(), "Closing...")
		return err
	}
	return nil
}

func (q *QLT) readQLTPacket() (int, error) {
	var err error
	for {
		if q.idx < 3 {
			log.Println(q.ctx, "Incomplete packet (<3), reading...", q.idx)
			err = q.readData()
			if err != nil {
				return 0, err
			}
			continue
		}
		if q.buf[2] != '1' {
			log.Println(q.ctx, "Received unexpected message code", q.buf[2], "Closing...")
			code := fmt.Sprintf("%c (0x%x)", q.buf[2], q.buf[2])
			return 0, errors.New("Unexpected QLT code : " + code)
		}

		length := int(q.buf[0])*256 + int(q.buf[1])

		if length+2 <= q.idx {
			return length, nil
		}

		log.Println(q.ctx, "Incomplete packet, reading...", q.idx, length+2)
		err = q.readData()
		if err != nil {
			return 0, err
		}
	}
}

var qltID = 0

func qltListen(addr string, q []chan map[string]string) {
	// Listen for incoming connections.
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Println("[QLT] Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()

	log.Println("[QLT]+ Listening on " + addr)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			log.Println("[QLT] Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn, q)
	}
}
func qltListenTLS(addr string, q []chan map[string]string, certFilename string, keyFilename string, caFilename string) {
	// Listen for incoming connections.
	cert, err := tls.LoadX509KeyPair(certFilename, keyFilename)
	if err != nil {
		log.Fatalf("[QLT] TLS - loadkeys: %s", err)
		os.Exit(1)
	}

	var certpool *x509.CertPool
	clientAuth := tls.RequireAnyClientCert
	if caFilename != "" {
		clientAuth = tls.RequireAndVerifyClientCert
		certpool = x509.NewCertPool()
		pem, err := ioutil.ReadFile(caFilename)
		if err != nil {
			log.Fatalf("[QLT] TLS - Error - Failed to read client certificate authority: %v", err)
		}
		if !certpool.AppendCertsFromPEM(pem) {
			log.Fatalf("[QLT] TLS - Error - Can't parse client certificate authority")
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
		log.Println("[QLT] TLS - Error listening qlts://"+addr, err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	log.Println("[QLT] TLS - listening on qlts://" + addr)

	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			log.Println("[QLT] TLS - Server: Error accepting: ", err.Error())
			continue
		}
		// Handle connections in a new goroutine.
		go func() {
			ctx := fmt.Sprint("[QLT-", qltID+1, "]")
			tlscon, ok := conn.(*tls.Conn)
			if ok {
				//log.Print(ctx + " TLS - Server: conn: type assert to TLS succeedded")
				err = tlscon.Handshake()
				if err != nil {
					log.Errorf(ctx+" TLS - Server: handshake failed: %s", err)
					return
				}

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
					log.Println("\n" + string(p))
				}

				handleRequest(conn, q)
			} else {
				log.Println(ctx + " TLS - server: conn: closed")
			}
		}()
	}
}

func handleRequest(conn net.Conn, ESQueue []chan map[string]string) QLT {
	var qlt QLT
	qltID++
	qlt.conn = conn
	qlt.buf = make([]byte, 65000)
	qlt.idx = 0
	qlt.ctx = fmt.Sprint("[QLT-", qltID, "]")
	qlt.ch = ESQueue
	qlt.handle()
	return qlt
}

func (q *QLT) handle() {
	defer q.conn.Close()
	defer qltConnectionIn.Dec()
	defer log.Println(q.ctx, "Closing...")

	defer qltConnectionIn.Inc()

	log.Println(q.ctx, "New Connection ", q.conn.RemoteAddr())
	// Make a buffer to hold incoming data.

	// Read the incoming connection into the buffer.
	count := 0

	for {
		length, err := q.readQLTPacket()
		if err != nil {
			break
		}
		count++
		log.Println(q.ctx, "[", count, "] Message Length", length-1)
		log.Println(q.ctx, "[", count, "] Message ", string(q.buf[3:3+length-1]))

		//FIXME: a bit early to write the Ack
		log.Println(q.ctx, "[", count, "] Writing Ack... ")
		err = q.writeQLTAck()
		if err != nil {
			break
		}

		qltMessageIn.Inc()
		qltMessageInSize.Observe(float64(length))
		log.Println(q.ctx, "[", count, "] Converting to Map... ")
		event, err := convertToMap(string(q.buf[3 : 3+length-1]))
		if err != nil {
			log.Println(q.ctx, "[", count, "] XML Parsing failed", err, "Closing...")
			break
		}

		log.Println(q.ctx, "[", count, "] JSON :", event)
		log.Println(q.ctx, "[", count, "] Pushing to ESQueue... ")

		log.Println(q.ctx, "[", count, "] Converting to Map... ")
		msg := processEvent(event)
		if msg["broker"] == "" {
			msg["broker"] = "qlt"
			msg["timestamp"] = time.Now().Format(time.RFC3339Nano)
			//msg["axway-target-flow"] = "api-central-v8" // Condor
			//msg["captureOrgID"] = "trcblt-test"         // tenant
			for idx, ch := range q.ch {
				log.Println(q.ctx, "[", count, "] Pushing to Queue... ", idx)
				ch <- msg
			}
		} else {
			log.Println(q.ctx, "[", count, "] Skip Message...", msg["broker"])
		}

		log.Println(q.ctx, "[", count, "] Recycling Buffer... ")
		copy(q.buf[0:], q.buf[length+2:q.idx])
		q.idx = q.idx - length - 2
	}
}
