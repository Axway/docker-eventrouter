package qlt

import (
	"net"
	"time"

	log "axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/tools"
)

type QltClientReader struct {
	CtxS              string
	Addr              string
	QueueName         string
	Cert, CertKey, Ca string
	qlt               *QLT
}

func NewQltClientReader(ctx string, addr string, queueName string, cert string, certKey string, ca string) *QltClientReader {
	var c QltClientReader
	c.Addr = addr
	c.CtxS = ctx
	c.QueueName = queueName
	c.Cert = cert
	c.CertKey = certKey
	c.Ca = ca

	return &c
}

func (c *QltClientReader) Connect(timeout time.Duration) error {
	log.Infoc(c.CtxS, "connecting... ", "addr", c.Addr, "queue", c.QueueName)

	var err error
	var conn net.Conn
	// FIXME: timeout needed
	if c.Ca != "" {
		conn, _, err = tools.TlsConnect(c.Addr, c.Ca, c.Cert, c.CertKey, c.CtxS)
	} else {
		conn, _, err = tools.TcpConnect(c.Addr, c.CtxS, timeout)
	}
	if err != nil {
		return err
	}
	c.qlt = newQltConnection(c.CtxS, conn)

	log.Infoc(c.CtxS, "connected", "addr", c.Addr, "queue", c.QueueName)
	err = c.qlt.Send(QLTPacketTypeConnRequest, c.QueueName)
	if err != nil {
		log.Errorc(c.CtxS, "initialization failed (sending pull request)", "addr", c.Addr, "queue", c.QueueName, "err", err)
		return err
	}
	log.Infoc(c.CtxS, "wait connection ack", "addr", c.Addr, "queue", c.QueueName)
	// FIXME: '5' connection rejected can be sent as well
	err = c.qlt.WaitAck(timeout)
	if err != nil {
		log.Errorc(c.CtxS, "initialization failed (waiting ack pull request)", "addr", c.Addr, "queue", c.QueueName, "err", err)
		return err
	}
	log.Infoc(c.CtxS, "connected and ready", "addr", c.Addr, "queue", c.QueueName)
	return nil
}

func (c *QltClientReader) Close() error {
	err := c.qlt.Close()
	log.Infoc(c.CtxS, "close", "err", err)
	return err
}

func (c *QltClientReader) WriteAck() error {
	return c.qlt.WriteQLTAck()
}

func (c *QltClientReader) Read(timeout time.Duration) (string, error) {
	msg, err := c.qlt.ReadQLTPacket(timeout)
	return msg, err
}
