package qlt

import (
	"errors"
	"io"
	"net"
	"time"

	"axway.com/qlt-router/src/config"
	log "axway.com/qlt-router/src/log"
)

type QltClient struct {
	CtxS   string
	conn   net.Conn
	Addr   string
	RCount int
	RSize  int
	WCount int
	WSize  int
}

var (
	qltSendRetryInitialDelay = config.DeclareDuration("qlt.SendRetryInitialDelay", "1s", "On connection/stream error, how much to retry, before reconnecting and sending")
	qltSendRetryMaxDelay     = config.DeclareDuration("qlt.SendRetryMaxDelay", "30s", "Max delay before retrying a send")
	qltSendRetryFactorDelay  = config.DeclareFloat("qlt.SendRetryMaxDelay", 1.5, "Factor on delay between retry of a send")
)

// FIXME: a windows should be set to be able to resend unsent packet ???!!!
func NewQltClient(ctx string, addr string) (*QltClient, error) {
	var c QltClient
	c.Addr = addr
	c.CtxS = ctx

	if err := c.Connect(); err != nil {
		log.Errorc(c.CtxS, " Dial failed", "addr", c.Addr, "err", err)
		return nil, err
	}
	// go c._AckLoop()
	return &c, nil
}

func (c *QltClient) Connect() error {
	log.Infoc(c.CtxS, " connecting... ", "addr", c.Addr)
	// FIXME: timeout needed
	conn, err := net.Dial("tcp", c.Addr)
	if err != nil {
		log.Errorc(c.CtxS, " dial failed", "addr", c.Addr, "err", err)
		return err
	}
	log.Infoc(c.CtxS, " connected", "addr", c.Addr)
	c.conn = conn
	return nil
}

func (c *QltClient) Close() error {
	err := c.conn.Close()
	log.Infoc(c.CtxS, " close", "rcount", c.RCount, "rsize", c.RSize, "wcount", c.WCount, "wsize", c.WSize, "err", err)
	return err
}

func (c *QltClient) WaitAck() error {
	buf := make([]byte, 3)

	// FIXME: timeout needed
	rsize, err := c.conn.Read(buf)
	if err != nil {
		log.Errorc(c.CtxS, " Error reading ack", "err", err)
		return err
	}
	if rsize < 3 {
		// FIXME: in theory the packet can be split in small pieces: retry needed
		log.Errorc(c.CtxS, " Error reading:", "err", err)
		return err
	}
	c.RCount++
	c.RSize += rsize
	if !qltIsPacketType(buf, QLTPacketTypeAck) {
		err := errors.New("unexpected packet type")
		log.Errorc(c.CtxS, " Error reading:", "err", err)
		return err
	}

	return nil
}

func (c *QltClient) Send(_msg string) error {
	retry := 0
	delay := qltSendRetryInitialDelay
	// log.Debugln(c.CtxS, "Send", _msg)
	for {
		if err := c._Send(_msg); err == io.EOF {
			if retry != 0 {
				delay := delay * time.Duration(qltSendRetryFactorDelay)
				if delay > qltSendRetryMaxDelay {
					delay = qltSendRetryMaxDelay
				}
				log.Warnc(c.CtxS, "  retrying...", "delay", delay/1000)
				time.Sleep(delay)
			}
			retry++
			c.Connect()
		} else {
			break
		}
	}
	// log.Debugln(c.CtxS, "Sent", _msg)
	return nil
}

func (c *QltClient) _Send(_msg string) error {
	m := []byte(_msg)
	l := len(m)
	wbuf := make([]byte, l+3)
	copy(wbuf[3:l+3], m[:])

	wbuf[0] = (byte)((l + 1) / 256)
	wbuf[1] = (byte)((l + 1) % 256)
	wbuf[2] = '1'
	// FIXME: timeout needed
	if _, err := c.conn.Write(wbuf); err != nil {
		log.Errorc(c.CtxS, "write error", "err", err)
		return err
	}
	c.WCount++
	c.WSize += l

	return nil
}
