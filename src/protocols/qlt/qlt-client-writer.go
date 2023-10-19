package qlt

import (
	"net"
	"time"

	"axway.com/qlt-router/src/tools"
	log "axway.com/qlt-router/src/log"
)

type QltClientWriter struct {
	CtxS string
	Addr string
	Cert, CertKey, Ca string
	qlt  *QLT
}

func NewQltClientWriter(ctx string, addr string, cert string, certKey string, ca string) *QltClientWriter {
	var c QltClientWriter
	c.Addr = addr
	c.CtxS = ctx
	c.Cert = cert
	c.CertKey = certKey
	c.Ca = ca

	return &c
}

func (c *QltClientWriter) Connect(timeout time.Duration) error {
	log.Infoc(c.CtxS, " connecting... ", "addr", c.Addr)

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
	log.Infoc(c.CtxS, " connected", "addr", c.Addr)
	c.qlt = newQltConnection(c.CtxS, conn)
	return nil
}

func (c *QltClientWriter) Close() error {
	err := c.qlt.Close()
	log.Infoc(c.CtxS, " close", "err", err)
	return err
}

func (c *QltClientWriter) WaitAck() error {
	err := c.qlt.WaitAck(0)
	return err
}

func (c *QltClientWriter) Send(_msg string) error {
	return c.qlt.Send(QLTPacketTypeData, _msg)
}
