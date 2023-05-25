package qlt

import (
	"context"
	"errors"
	"strings"
	"time"

	"axway.com/qlt-router/src/processor"
	"axway.com/qlt-router/src/protocols/qlt"
	log "github.com/sirupsen/logrus"
)

type QLTClientWriter struct {
	Conf *QLTClientWriterConf
	CtxS string
}

func (q *QLTClientWriter) Init(p *processor.Processor) error {
	return nil
}

func (q *QLTClientWriter) Close() error {
	return nil
}

func (q *QLTClientWriter) Ctx() string {
	return q.CtxS
}

type QLTClientWriterConf struct {
	Addresses string
	Cnx       int
}

func (c *QLTClientWriterConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := &QLTClientWriter{c, p.Name}

	addrs := strings.Split(q.Conf.Addresses, ",")
	if len(addrs) == 0 {
		return nil, errors.New("Not enough addresses")
	}
	for i := 0; i < q.Conf.Cnx; i++ {
		for _, addr := range addrs {
			log.Debugln(q.CtxS, "connection", "addr", addr)
			p.Chans.Create(q.CtxS+"-AckEvent (Not Tracked)", 100)
			// TcpChaosInit(TCPChaosConf{Name: q.Ctx, Addr: addr})
			c2 := &QLTClientWriterConnection{q.CtxS, addr, nil, make(chan processor.AckableEvent, 100)}
			log.Debugln(q.CtxS, "AddReader!!!!*************************")
			p.AddReader(c2)
		}
	}

	return q, nil
}

func (c *QLTClientWriterConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

type QLTClientWriterConnection struct {
	CtxS string
	Addr string
	Qlt  *qlt.QltClient
	acks chan processor.AckableEvent
}

func (c *QLTClientWriterConnection) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	conn, err := processor.GenProcessorHelperWriter(context, c, p, ctl, inc, out)
	return conn, err
}

func (c *QLTClientWriterConnection) Clone() processor.Connector {
	return &*c
}

func (q *QLTClientWriterConnection) Ctx() string {
	return q.CtxS
}

func (q *QLTClientWriterConnection) Init(p *processor.Processor) error {
	log.Println(q.CtxS, "Connecting to ", q.Addr, "...")
	for i := 0; i < 10; i++ {
		client, err := qlt.NewQltClient(q.CtxS, q.Addr)
		if err != nil {
			log.Errorln(q.CtxS, "failed to connect", "addr", q.Addr, "err", err)
			// return err
		} else {
			q.Qlt = client
			return nil
		}
		time.Sleep(time.Millisecond * time.Duration(i*100))
	}
	return nil
}

func (q *QLTClientWriterConnection) PrepareEvent(event *processor.AckableEvent) (string, error) {
	str, _ := event.Msg.(string)
	return str, nil
}

func (q *QLTClientWriterConnection) Write(events []processor.AckableEvent) error {
	// log.Debugln(q.CtxS, "Write events", "events", events)
	for _, event := range events {
		str, _ := event.Msg.(string)
		if err := q.Qlt.Send(str); err != nil {
			panic(err)
		}
		// log.Debugln(q.CtxS, "Wrote", str)
		q.acks <- event
	}

	return nil
}

func (q *QLTClientWriterConnection) IsAckAsync() bool {
	return true
}

func (q *QLTClientWriterConnection) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent) {
	for {
		// log.Debugln(q.CtxS, "waiting msg to ack")
		event, ok := <-q.acks
		if !ok {
			log.Infoln(q.CtxS, "close ack loop")
			return
		}
		// log.Debugln(q.CtxS, "waiting ack from qlt", "msgId", event.Msgid)

		err := q.Qlt.WaitAck()
		if err != nil {
			log.Errorln(q.CtxS, "error waiting ack: close ack loop", "err", err)
			return
		}
		// log.Debugln(q.CtxS, "ack received", "msgId", event.Msgid)
		acks <- event
	}
}

func (q *QLTClientWriterConnection) Close() error {
	if q.Qlt == nil {
		log.Warnln(q.CtxS, "close", "warn", "not opened")
		return nil
	}
	err := q.Qlt.Close()
	if err != nil {
		log.Errorln(q.CtxS, "close", "err", err)
	} else {
		log.Infoln(q.CtxS, "close OK")
	}
	return err
}
