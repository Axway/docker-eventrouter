package mem

import (
	"context"
	"fmt"
	"io"
	"time"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
)

type MemReaders struct {
	Conf *MemReadersConf
	CtxS string
	// sources   []*MemReadersSource
}

func (q *MemReaders) Init(p *processor.Processor) error {
	return nil
}

func (q *MemReaders) Close() error {
	return nil
}

func (q *MemReaders) Ctx() string {
	return q.CtxS
}

type MemReadersConf struct {
	Messages [][]string
}

/*
func (r *MemReaders) Add(messages []string) {
	reader := &MemReadersSource{Messages: messages}
	r.processor.AddRuntime(reader)
}*/

func (c *MemReadersConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := &MemReaders{c, p.Name}
	for i := 0; i < len(q.Conf.Messages); i++ {
		src := &MemReadersSource{CtxS: q.CtxS + "-" + fmt.Sprint(i), AckPos: -1}
		src.Messages = q.Conf.Messages[i]
		p.AddRuntime(src)
	}
	return q, nil
}

func (c *MemReadersConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

type MemReadersSource struct {
	Messages []string
	Current  int
	CtxS     string
	SentPos  int64
	AckPos   int64
}

func (m *MemReadersSource) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	c, e := processor.GenProcessorHelperReader(context, m, p, ctl, inc, out)
	return c, e
}

func (c *MemReadersSource) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (m *MemReadersSource) Init(p *processor.Processor) error {
	return nil
}

func (m *MemReadersSource) AckMsg(ack processor.EventAck) {
	msgid := ack.(int64)
	if m.AckPos >= msgid {
		panic("oups : already received or wrong")
	}
	m.AckPos = msgid
}

func (m *MemReadersSource) Ctx() string {
	return m.CtxS
}

func (m *MemReadersSource) Read() ([]processor.AckableEvent, error) {
	if m.Current >= len(m.Messages) {
		return nil, io.EOF
	}
	msgs := []processor.AckableEvent{{m, int64(m.Current), m.Messages[m.Current], nil}}
	log.Tracec(m.CtxS, "read 1 msg", "msg", m.Messages[m.Current])
	// n := time.Duration(int32(rand.Float32() * 100))
	time.Sleep(10 * time.Microsecond)
	m.Current++
	return msgs, nil
}

func (m *MemReadersSource) Close() error {
	return nil
}
