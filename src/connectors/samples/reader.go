package sample

import (
	"context"
	"io"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
)

// Replace SampleReader* by your Connector Name
type SampleReader struct {
	Conf *SampleReaderConf
	CtxS string

	Current int   // Sample
	AckPos  int64 // Sample
}

type SampleReaderConf struct {
	Messages []string // Sample
}

func (c *SampleReaderConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := SampleReader{Conf: c, CtxS: p.Name}

	r, err := processor.GenProcessorHelperReader(context, &q, p, ctl, inc, out)
	return r, err
}

func (c *SampleReaderConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (m *SampleReader) Init(p *processor.Processor) error {
	return nil
}

func (m *SampleReader) AckMsg(ack processor.EventAck) {
	msgid, ok := ack.(int64)
	if !ok {
		log.Fatalc(m.CtxS, "invalid ack type", "ack", ack)
	}
	m.AckPos = msgid
}

func (m *SampleReader) Ctx() string {
	return m.CtxS
}

func (m *SampleReader) Read() ([]processor.AckableEvent, error) {
	if m.Current >= len(m.Conf.Messages) {
		return nil, io.EOF
	}
	msgs := []processor.AckableEvent{{m, int64(m.Current), m.Conf.Messages[m.Current], nil}}
	m.Current++
	return msgs, nil
}

func (m *SampleReader) Close() error {
	return nil
}
