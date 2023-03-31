package mem

import (
	"context"
	"io"

	log "github.com/sirupsen/logrus"

	"axway.com/qlt-router/src/processor"
)

type MemReader struct {
	Conf *MemReaderConf
	CtxS string

	Current int
	SentPos int64
	AckPos  int64
}

type MemReaderConf struct {
	Messages []string
}

func (c *MemReaderConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := MemReader{Conf: c, CtxS: p.Name}

	r, err := processor.GenProcessorHelperReader(context, &q, p, ctl, inc, out)
	return r, err
}

func (c *MemReaderConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func NewMemReader(ctx string, messages []string) *MemReader {
	m := MemReader{}
	m.CtxS = ctx
	return &m
}

func (m *MemReader) Init(p *processor.Processor) error {
	return nil
}

func (m *MemReader) AckMsg(ack processor.EventAck) {
	msgid, ok := ack.(int64)
	if !ok {
		log.Fatalln(m.CtxS, "invalid ack type", ack)
	}
	m.AckPos = msgid
}

func (m *MemReader) Ctx() string {
	return m.CtxS
}

func (m *MemReader) Read() ([]processor.AckableEvent, error) {
	if m.Current >= len(m.Conf.Messages) {
		return nil, io.EOF
	}
	msgs := []processor.AckableEvent{{m, int64(m.Current), m.Conf.Messages[m.Current], nil}}
	m.Current++
	return msgs, nil
}

func (m *MemReader) Close() error {
	return nil
}
