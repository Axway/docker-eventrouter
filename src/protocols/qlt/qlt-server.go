package qlt

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"axway.com/qlt-router/src/config"
	log "axway.com/qlt-router/src/log"
)

// QLTMessage to be passed to protocol
/*type QLTMessage struct {
	qltEvent *QLTEvent
	Fields   map[string]string
}*/

/*type QLTEvent struct {
	qlt   processor.QLTEventSource
	msgid int64
	msg   string
}*/

// QLT Handles incoming requests.
// QLT protocol
//
//	offset [0-1] length
//	offset [2] type
// 1
//
//	Data to be processed
//	Next bytes are the message
//	Message will be accepted (ACK) or rejected (NACK)
//
// 2
//
//	ACK = Positive Acknowledgement
//	Sender can pass at the next message or cut the connection.
//
// 3
//
//	NACK = Negative Acknowledgement
//	Message cannot be processed by the receiver
//	If sender  use an overflow file, the message will be send again at next connection
//
// 4
//
//	Connection Requested
//	Only for ER mode QLTREQ/QLTSRV
//	Data to follow are the Target Name (in ascii)
//
// 5
//
//	Connection Rejected
//	Only for ER mode QLRREQ/QLTSRV
//	Data to follow is composed
//	1 byte for Indication that connection retry will be  accepted (2) or not (3)
//	If 3, the connection requester must stop.
//	Raison explaining the error on two bytes
//	Text  explaining the cause the reject code.
//
// 6
//
//	Ask to stop sending message
//	Only for ER mode QLRREQ/QLTSRV
//	The server ask to stop
//	The requester will do the connection later, following its retry parameters
//	The message is the same that for Connection rejected except
//	Code = 11
//	String text contain “no more message“

const (
	QLTPacketTypeData         = '1' // Data : message
	QLTPacketTypeAck          = '2' // NoData
	QLTPacketTypeNAck         = '3' // NoData
	QLTPacketTypeConnRequest  = '4' // Data Target Name
	QLTPacketTypeConnRejected = '5' // Data [IsPermanent/1][ReasonCode/2][ResonText...]
	QLTPacketTypeConnStop     = '6' // Data="no more message"
)

var (
	RecvBufferSize            = config.DeclareSize("qlt.RecvBufferSize", "64kb", "Max QLT Packet Size (theory is 64k+1)")
	RecvBufferCompleteTimeout = config.DeclareDuration("qlt.RecvBufferCompleteTimeout", "10s", "(unsupported)")
)

func qltIsPacketType(buf []byte, typ byte) bool {
	return qltPacketType(buf) == typ
}

func qltPacketType(buf []byte) byte {
	return buf[2]
}

func qltPacketSize(buf []byte) int {
	return int(buf[0])*256 + int(buf[1])
}

func qltMakePacket(buf []byte, typ byte, size int) {
	buf[0] = (byte)((size + 1) / 256)
	buf[1] = (byte)((size + 1) % 256)
	buf[2] = typ
}

type QLT struct {
	Conn   net.Conn
	buf    []byte
	idx    int
	RCount int
	RSize  int
	WCount int
	WSize  int
	CtxS   string
}

func newQltConnection(ctx string, conn net.Conn) *QLT {
	qlt := QLT{}
	qlt.Conn = conn
	qlt.buf = make([]byte, RecvBufferSize)
	qlt.idx = 0
	qlt.CtxS = ctx
	log.Infoc(qlt.CtxS, "New QLT connection")
	return &qlt
}

func (q *QLT) Close() error {
	err := q.Conn.Close()
	log.Infoc(q.CtxS, "Close", "err", err, "rcount", q.RCount, "wcount", q.WCount)
	return err
}

func (q *QLT) readData() error {
	// log.Println(q.ctx, "Reading... idx=", q.idx)

	rsize, err := q.Conn.Read(q.buf[q.idx:])
	if err != nil {
		if !errors.Is(err, os.ErrDeadlineExceeded) && !errors.Is(err, io.EOF) {
			log.Errorc(q.CtxS, "Error reading closing...", "err", err.Error())
		}
		return err
	}
	// log.Println(q.ctx, "Read:", "idx=", q.idx, "rsize=", rsize, "size=", q.idx+rsize)
	q.idx = rsize + q.idx
	return nil
}

func (q *QLT) WriteQLTAck() error {
	// FIXME: timeout needed
	_, err := q.Conn.Write([]byte{0, 1, QLTPacketTypeAck})
	if err != nil {
		log.Errorc(q.CtxS, "Error writing (Closing...)", "err", err)
		return err
	}
	q.WCount++
	q.WSize += 3
	return nil
}

func (q *QLT) WaitAck(timeout time.Duration) error {
	_, typ, err := q.ReadQLTPacketRaw(timeout)
	if err != nil {
		return err
	}
	if typ != QLTPacketTypeAck {
		log.Errorc(q.CtxS, "Received unexpected message code (Closing...)", "code", typ)
		code := fmt.Sprintf("%c (0x%x)", q.buf[2], q.buf[2])
		return errors.New("Unexpected QLT code : " + code)
	}
	return nil
}

func (q *QLT) ReadQLTPacket(timeout time.Duration) (string, error) {
	msg, typ, err := q.ReadQLTPacketRaw(timeout)
	if err != nil {
		return "", err
	}
	if typ != QLTPacketTypeData {
		log.Errorc(q.CtxS, "Received unexpected message code (Closing...)", "code", typ)
		code := fmt.Sprintf("%c (0x%x)", q.buf[2], q.buf[2])
		return "", errors.New("Unexpected QLT code : " + code)
	}

	return msg, nil
}

func (q *QLT) ReadQLTPacketRaw(timeout time.Duration) (string, byte, error) {
	var err error

	log.Tracec(q.CtxS, "ReadQLTPacketRaw", "timeout", timeout)
	if timeout > 0 {
		err = q.Conn.SetReadDeadline(time.Now().Add(timeout))
		if err != nil {
			log.Errorc(q.CtxS, "Error setting deadline closing...", "err", err.Error())
			return "", 0xFF, err
		}
	} else {
		err = q.Conn.SetReadDeadline(time.Time{})
		if err != nil {
			log.Errorc(q.CtxS, "Error setting deadline closing...", "err", err.Error())
			return "", 0xFF, err
		}
	}
	for {
		if q.idx < 3 {
			// log.Println(q.ctx, "Incomplete packet (<3), reading...", q.idx)
			err = q.readData()
			if err != nil {
				return "", 0xFF, err
			}
			continue
		}
		typ := q.buf[2]
		/*if !qltIsPacketType(q.buf, QLTPacketTypeData) {
			log.Errorc(q.CtxS, "Received unexpected message code (Closing...)", "code", q.buf[2])
			code := fmt.Sprintf("%c (0x%x)", q.buf[2], q.buf[2])
			return "", q.buf[2], errors.New("Unexpected QLT code : " + code)
		}*/

		length := qltPacketSize(q.buf)
		if length == 0 {
			if q.Conn.RemoteAddr() != nil {
				return "", 0xFF, errors.New("unexpected QLT message size: 0 (RemoteAddr=" + q.Conn.RemoteAddr().String() + ")")
			}
			return "", 0xFF, errors.New("unexpected QLT message size: 0")
		}

		if length+2 <= q.idx {
			orig := string(q.buf[3 : 3+length-1])
			copy(q.buf[0:], q.buf[length+2:q.idx])
			q.idx = q.idx - length - 2
			q.RCount++
			q.RSize += length
			return orig, typ, nil
		}

		// log.Println(q.ctx, "Incomplete packet, reading...", q.idx, length+2)
		err = q.readData()
		if err != nil {
			return "", 0xFF, err
		}
	}
}

func (q *QLT) Send(typ byte, _msg string) error {
	m := []byte(_msg)
	l := len(m)
	wbuf := make([]byte, l+3)
	copy(wbuf[3:l+3], m[:])

	wbuf[0] = (byte)((l + 1) / 256)
	wbuf[1] = (byte)((l + 1) % 256)
	wbuf[2] = typ
	// FIXME: timeout needed
	if _, err := q.Conn.Write(wbuf); err != nil {
		log.Errorc(q.CtxS, "write error", "err", err)
		return err
	}
	q.WCount++
	q.WSize += l

	return nil
}
