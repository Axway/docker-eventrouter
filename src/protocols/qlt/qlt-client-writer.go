package qlt

import (
	"net"
	"time"

	log "axway.com/qlt-router/src/log"
)

type QltClientWriter struct {
	CtxS string
	Addr string
	qlt  *QLT
}

func NewQltClientWriter(ctx string, addr string) *QltClientWriter {
	var c QltClientWriter
	c.Addr = addr
	c.CtxS = ctx

	return &c
}

func (c *QltClientWriter) Connect(timeout time.Duration) error {
	log.Infoc(c.CtxS, " connecting... ", "addr", c.Addr)
	// FIXME: timeout needed
	conn, err := net.DialTimeout("tcp", c.Addr, timeout)
	if err != nil {
		log.Errorc(c.CtxS, " dial failed", "addr", c.Addr, "err", err)
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
