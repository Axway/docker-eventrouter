package mem

import (
	"context"
	_ "embed"
	"fmt"
	"math"
	"time"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
)

type MemGeneratorReader struct {
	Conf *MemGeneratorReaderConf
	CtxS string

	Current int
	SentPos int64
	AckPos  int64

	CurrentDelay int
	C            float64
}

type MemGeneratorReaderConf struct {
	MinDelay int
	MaxDelay int
	AvgDelay int
}

func (c *MemGeneratorReaderConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := MemGeneratorReader{Conf: c, CtxS: p.Name}

	if c.MinDelay == 0 {
		c.MinDelay = 1000
	}
	if c.AvgDelay == 0 {
		c.AvgDelay = c.MinDelay * 2
	}
	if c.MaxDelay == 0 {
		c.MaxDelay = c.AvgDelay * 2
	}
	q.CurrentDelay = c.AvgDelay

	r, err := processor.GenProcessorHelperReader(context, &q, p, ctl, inc, out)
	return r, err
}

func (c *MemGeneratorReaderConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func NewMemGeneratorReader(ctx string, messages []string) *MemGeneratorReader {
	m := MemGeneratorReader{}
	m.CtxS = ctx
	return &m
}

func (m *MemGeneratorReader) Init(p *processor.Processor) error {
	return nil
}

func (m *MemGeneratorReader) AckMsg(ack processor.EventAck) {
	msgid, ok := ack.(int64)
	if !ok {
		log.Fatalc(m.CtxS, "invalid ack type", "ack", ack)
	}
	m.AckPos = msgid
}

func (m *MemGeneratorReader) Ctx() string {
	return m.CtxS
}

func (m *MemGeneratorReader) IsServer() bool {
	return false
}

//go:embed qltEvent.xml
var qltSample string

func (m *MemGeneratorReader) Read() ([]processor.AckableEvent, error) {
	if m.Conf.AvgDelay != -1 {
		t := time.Now().UnixMilli()
		p := 5
		m.C = math.Cos(float64(t) / 1000 * math.Pi / float64(p))
		m.CurrentDelay = m.Conf.AvgDelay + int(float64(m.Conf.MinDelay-m.Conf.AvgDelay)*m.C)
		if m.CurrentDelay < m.Conf.MinDelay {
			m.CurrentDelay = m.Conf.MinDelay
		}
		if m.CurrentDelay > m.Conf.MaxDelay {
			m.CurrentDelay = m.Conf.MaxDelay
		}
		// log.Debugln(m.CtxS, "********** delay", m.CurrentDelay, m.Conf.AvgDelay, m.C)
		time.Sleep(time.Duration(m.CurrentDelay) * time.Microsecond)
	}
	msg := fmt.Sprint(qltSample)
	msgs := []processor.AckableEvent{{m, int64(m.Current), msg, nil}}
	m.Current++
	return msgs, nil
}

func (m *MemGeneratorReader) Close() error {
	return nil
}
