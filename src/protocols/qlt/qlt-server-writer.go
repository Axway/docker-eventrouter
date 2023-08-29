package qlt

import (
	"errors"
	"net"
	"time"

	log "axway.com/qlt-router/src/log"
)

type QltServerWriter struct {
	CtxS      string
	QueueName string
	qlt       *QLT
}

func NewQltServerWriter(ctx string, conn net.Conn, QueueName string) *QltServerWriter {
	var c QltServerWriter
	c.QueueName = QueueName
	c.qlt = newQltConnection(c.CtxS, conn)
	return &c
}

func (c *QltServerWriter) Close() error {
	err := c.qlt.Close()
	log.Infoc(c.CtxS, " close", "err", err)
	return err
}

func (c *QltServerWriter) WaitQueueName(timeout time.Duration) error {
	queueName, typ, err := c.qlt.ReadQLTPacketRaw(timeout)
	if typ != QLTPacketTypeConnRequest {
		return errors.New("Unexpected Packet Type")
	}
	if queueName != c.QueueName {
		return errors.New("Unexpected Queue Name")
	}
	return err
}

func (c *QltServerWriter) AckQueueName() error {
	return c.qlt.WriteQLTAck()
}

func (c *QltServerWriter) WaitAck(timeout time.Duration) error {
	err := c.qlt.WaitAck(timeout)
	return err
}

func (c *QltServerWriter) Send(_msg string) error {
	return c.qlt.Send(QLTPacketTypeData, _msg)
}
