package qlt

import (
	"context"
	"errors"
	"strings"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"axway.com/qlt-router/src/protocols/qlt"
)

var (
	_ processor.Connector              = &QLTClientWriterConf{}
	_ processor.ConnectorRuntime       = &QLTClientWriter{}
	_ processor.ConnectorRuntimeWriter = &QLTClientWriterConnection{}
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
	Addresses         string
	Cert, CertKey, Ca string
	Cnx               int
}

func (c *QLTClientWriterConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := &QLTClientWriter{c, p.Name}

	addrs := strings.Split(q.Conf.Addresses, ",")
	if len(addrs) == 0 {
		return nil, errors.New("Not enough addresses")
	}
	for i := 0; i < q.Conf.Cnx; i++ {
		for _, addr := range addrs {
			log.Debugc(q.CtxS, "connection", "addr", addr)
			acks := p.Chans.Create(q.CtxS+"-AckEvent (Not Tracked)", qltAckQueueSize)
			// TcpChaosInit(TCPChaosConf{Name: q.Ctx, Addr: addr})
			c2 := &QLTClientWriterConnection{c, q.CtxS, addr, nil, acks.C}
			log.Debugc(q.CtxS, "AddReader!!!!*************************")
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
	Conf *QLTClientWriterConf
	CtxS string
	Addr string
	Qlt  *qlt.QltClientWriter
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
	/*log.Infoc(q.CtxS, "Connecting to ", "addr", q.Addr)
	for i := 0; i < 10; i++ {
		client := qlt.NewQltClientWriter(q.CtxS, q.Addr)
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

func (q *QLTClientWriterConnection) Write(events []processor.AckableEvent) (int, error) {
	n := 0
	if q.Qlt == nil {
		log.Infoc(q.CtxS, "Connecting to ", "addr", q.Addr)
		client := qlt.NewQltClientWriter(q.CtxS, q.Addr, q.Conf.Cert, q.Conf.CertKey, q.Conf.Ca)
		err := client.Connect(qltClientConnectTimeout)
		if err != nil {
			log.Errorc(q.CtxS, "failed to connect", "addr", q.Addr, "err", err)
			return n, err
		} else {
			q.Qlt = client
		}
	}
	// log.Debugln(q.CtxS, "Write events", "events", events)
	for _, event := range events {
		str, _ := event.Msg.(string)

		if q.Qlt == nil {
			log.Warnc(q.CtxS, "")
			return n, nil
		}
		if err := q.Qlt.Send(str); err != nil {
			//log.Debugc(q.CtxS, "close")
			q.Close()
			return n, err
		}
		// log.Debugc(q.CtxS, "Wrote", "message", str)
		q.acks <- event
		n++
	}

	return n, nil
}

func (q *QLTClientWriterConnection) IsAckAsync() bool {
	return true
}

func (q *QLTClientWriterConnection) IsActive() bool {
	return q.Qlt != nil
}

func (q *QLTClientWriterConnection) DrainAcks() {
	for i := len(q.acks); i > 0; i-- {
		select {
		case _, ok := <-q.acks:
			if !ok {
				log.Debugc(q.CtxS, "acks channel closed while draining")
				return
			}
		default:
			log.Debugc(q.CtxS, "no event in acks channel while draining")
			return
		}
	}
}

func (q *QLTClientWriterConnection) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent) {
	for {
		select {
		case <-ctx.Done():
			log.Infoc(q.CtxS, "close ack loop")
			return
		default:
			event, ok := <-q.acks
			if !ok {
				log.Infoc(q.CtxS, "close ack loop")
				return
			}

			if q.Qlt == nil {
				log.Warnc(q.CtxS, "close warn not opened: draining")
				q.DrainAcks()
				continue
			}

			err := q.Qlt.WaitAck()
			if err != nil {
				log.Infoc(q.CtxS, "error waiting ack: draining", "err", err)
				q.DrainAcks()
				q.Close()
				continue
			}

			acks <- event
		}
	}
}

func (q *QLTClientWriterConnection) Close() error {
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
