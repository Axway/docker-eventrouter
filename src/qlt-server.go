package main

import (
	"fmt"
	"net"
	"os"

	log "github.com/sirupsen/logrus"
)

// Handles incoming requests.
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
				return -1, err
			}
			continue
		}
		if q.buf[2] != '1' {
			log.Println(q.ctx, "Received unexpected message code", q.buf[2], "Closing...")
			return -1, nil //FIXME: need error
		}

		length := int(q.buf[0])*256 + int(q.buf[1])

		if length+2 <= q.idx {
			return length, nil
		}

		log.Println(q.ctx, "Incomplete packet, reading...", q.idx, length+2)
		err = q.readData()
		if err != nil {
			return -1, err
		}
	}
}

var qltId = 0

func QLTListen(addr string, q []chan map[string]string) {
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

func handleRequest(conn net.Conn, ESQueue []chan map[string]string) QLT {
	var qlt QLT
	qltId++
	qlt.conn = conn
	qlt.buf = make([]byte, 32768)
	qlt.idx = 0
	qlt.ctx = fmt.Sprint("[QLT-", qltId, "]")
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
		event, err := ConvertToMap(string(q.buf[3 : 3+length-1]))
		if err != nil {
			log.Println(q.ctx, "[", count, "] XML Parsing failed", err, "Closing...")
			break
		}

		log.Println(q.ctx, "[", count, "] JSON :", event)
		log.Println(q.ctx, "[", count, "] Pushing to ESQueue... ")

		log.Println(q.ctx, "[", count, "] Converting to Map... ")
		msg := ProcessEvent(event)
		if msg["broker"] == "" {
			msg["broker"] = "qlt"
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
