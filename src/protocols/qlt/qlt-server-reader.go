package qlt

import (
	"net"
	"time"

	log "axway.com/qlt-router/src/log"
)

type QltServerReader struct {
	CtxS string
	qlt  *QLT
}

func NewQltServerReader(ctx string, conn net.Conn) *QltServerReader {
	var c QltServerReader
	c.qlt = newQltConnection(c.CtxS, conn)
	return &c
}

func (c *QltServerReader) Close() error {
	err := c.qlt.Close()
	log.Infoc(c.CtxS, " close", "err", err)
	return err
}

func (c *QltServerReader) Read(timeout time.Duration) (string, error) {
	return c.qlt.ReadQLTPacket(timeout)
}

func (c *QltServerReader) WriteAck() error {
	return c.qlt.WriteQLTAck()
}
