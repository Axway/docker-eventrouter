package qlt

import (
	"context"
	"errors"
	"os"
	"strings"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"axway.com/qlt-router/src/protocols/qlt"
)

var (
	_ processor.Connector              = &QLTClientReaderConf{}
	_ processor.ConnectorRuntime       = &QLTClientReader{}
	_ processor.ConnectorRuntimeReader = &QLTClientReaderConnection{}
)

type QLTClientReader struct {
	Conf *QLTClientReaderConf
	CtxS string
}

func (q *QLTClientReader) Init(p *processor.Processor) error {
	return nil
}

func (q *QLTClientReader) Close() error {
	return nil
}

func (q *QLTClientReader) Ctx() string {
	return q.CtxS
}

type QLTClientReaderConf struct {
	QueueName string
	Addresses string
	Cnx       int
}

func (c *QLTClientReaderConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := &QLTClientReader{c, p.Name}

	addrs := strings.Split(q.Conf.Addresses, ",")
	if len(addrs) == 0 {
		return nil, errors.New("Not enough addresses")
	}
	for i := 0; i < q.Conf.Cnx; i++ {
		for _, addr := range addrs {
			log.Debugc(q.CtxS, "connection", "addr", addr)
			// TcpChaosInit(TCPChaosConf{Name: q.Ctx, Addr: addr})
			acks := make(chan int64, qltAckQueueSize) // FIXME: untracked queue
			c2 := &QLTClientReaderConnection{c, q.CtxS, addr, nil, 0, 0, acks}
			log.Debugc(q.CtxS, "AddReader!!!!*************************")
			p.AddReader(c2)
		}
	}

	return q, nil
}

func (c *QLTClientReaderConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

type QLTClientReaderConnection struct {
	Conf *QLTClientReaderConf
	CtxS string
	Addr string
	Qlt  *qlt.QltClientReader

	MsgId  int64
	AckPos int64
	ack    chan int64
}

func (c *QLTClientReaderConnection) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	conn, err := processor.GenProcessorHelperReader(context, c, p, ctl, inc, out)
	return conn, err
}

func (c *QLTClientReaderConnection) Clone() processor.Connector {
	return &*c
}

func (q *QLTClientReaderConnection) Ctx() string {
	return q.CtxS
}

func (q *QLTClientReaderConnection) Init(p *processor.Processor) error {
	/*log.Infoc(q.CtxS, "Connecting to ", "addr", q.Addr)
	for i := 0; i < 10; i++ {
		client := qlt.NewQltClientReader(q.CtxS, q.Addr, q.Conf.QueueName)

		err := client.Connect(qltClientConnectTimeout)
		if err != nil {
			log.Errorc(q.CtxS, "failed to connect", "addr", q.Addr, "err", err)
			// return err
		} else {
			q.Qlt = client
			return nil
		}
		time.Sleep(qltClientConnectionRetryDelay)
	}*/
	return nil
}

/*
func (q *QLTClientReaderConnection) PrepareEvent(event *processor.AckableEvent) (string, error) {
	str, _ := event.Msg.(string)
	return str, nil
}
*/

func (c *QLTClientReaderConnection) Read() ([]processor.AckableEvent, error) {
	if c.Qlt == nil {
		log.Infoc(c.CtxS, "Connecting to ", "addr", c.Addr)
		client := qlt.NewQltClientReader(c.CtxS, c.Addr, c.Conf.QueueName)
		err := client.Connect(qltClientConnectTimeout)
		if err != nil {
			log.Errorc(c.CtxS, "failed to connect", "addr", c.Addr, "queue", c.Conf.QueueName, "err", err)
			return nil, err
		} else {
			c.Qlt = client
		}
	}
	events := make([]processor.AckableEvent, 1)
	msg, err := c.Qlt.Read(qltReaderBlockTimeout)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return nil, err
		}

		return nil, err
	}
	c.MsgId += 1
	events[0] = processor.AckableEvent{c, c.MsgId, msg, nil}
	return events, nil
}

func (m *QLTClientReaderConnection) AckMsg(ack processor.EventAck) {
	msgid := ack.(int64)
	if m.AckPos+1 != msgid {
		panic("oups : already received or wrong")
	}
	err := m.Qlt.WriteAck()
	if err != nil {
		log.Errorc(m.CtxS, "error writing ack", "err", err)
		m.Close()
		return
	}
	m.AckPos = msgid
	// log.Debugln(m.CtxS, "ack", "msgid", m.MsgId, "ackPos", m.AckPos)
}

func (q *QLTClientReaderConnection) Close() error {
	if q.Qlt == nil {
		log.Warnc(q.CtxS, "close warn not opened")
		return nil
	}
	err := q.Qlt.Close()
	if err != nil {
		log.Errorc(q.CtxS, "close", "err", err)
	} else {
		log.Infoc(q.CtxS, "close OK")
	}
	q.Qlt = nil
	return err
}
