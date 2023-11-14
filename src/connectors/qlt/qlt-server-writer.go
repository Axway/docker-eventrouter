package qlt

import (
	"context"
	"net"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"axway.com/qlt-router/src/protocols/qlt"
	"axway.com/qlt-router/src/tools"
)

var (
	_ processor.Connector              = &QLTServerWriterConf{}
	_ processor.ConnectorRuntime       = &QLTServerWriter{}
	_ processor.ConnectorRuntimeWriter = &QLTServerWriterConnection{}
)

type QLTServerWriter struct {
	Conf     *QLTServerWriterConf
	ctx      string
	listener net.Listener
}

func (q *QLTServerWriter) Init(p *processor.Processor) error {
	return nil
}

func (q *QLTServerWriter) Ctx() string {
	return q.ctx
}

func (q *QLTServerWriter) Close() error {
	if q.listener == nil {
		log.Warnc(q.ctx, "listener closing: empty listener")
		return nil
	}
	log.Infoc(q.ctx, "listener closing")
	err := q.listener.Close()
	if err != nil {
		log.Errorc(q.ctx, "listener close error", "err", err)
	} else {
		log.Debugc(q.ctx, "listener closed")
	}
	return err
}

type QLTServerWriterConf struct {
	QueueName                     string
	Host, Port, Cert, CertKey, Ca string
}

type QLTServerWriterConnection struct {
	CtxS string
	Conf *QLTServerWriterConf
	Qlt  *qlt.QltServerWriter
	acks chan processor.AckableEvent
	From string
}

func (conf *QLTServerWriterConf) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, in chan processor.AckableEvent, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	qltHandle := func(conn net.Conn, ctx2 string) {
		// qltHandleRequest(ctx, conn, ctx2+p.Flow.Name, p, ctl, out)
		qlt := qlt.NewQltServerWriter(ctx2, conn, conf.QueueName)

		src := &QLTServerWriterConnection{CtxS: ctx2 + ".conn", Conf: conf, Qlt: qlt, acks: make(chan processor.AckableEvent, qltAckQueueSize), From: conn.RemoteAddr().String()}

		p.AddReader(src)
	}
	var listener net.Listener
	var err error
	if conf.Cert != "" {
		listener, err = tools.TlsServe(conf.Host+":"+conf.Port, conf.Cert, conf.CertKey, conf.Ca, qltHandle, false, "QLT-TLS")
	} else {
		listener, err = tools.TcpServe(conf.Host+":"+conf.Port, qltHandle, "QLT-TCP")
	}
	q := &QLTServerWriter{conf, p.Name, listener}
	log.Debugc(q.ctx, "listening server started", "host", conf.Host, "port", conf.Port)
	if err != nil {
		log.Errorc(q.ctx, "error starting listening server", "err", err)
	}
	return q, err
}

func (c *QLTServerWriterConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (m *QLTServerWriterConnection) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	c, e := processor.GenProcessorHelperWriter(context, m, p, ctl, inc, out)
	return c, e
}

func (c *QLTServerWriterConnection) Clone() processor.Connector {
	return &*c
}

func (m *QLTServerWriterConnection) Init(p *processor.Processor) error {
	log.Infoc(m.CtxS, "wait queue name", "queue", m.Conf.QueueName)
	err := m.Qlt.WaitQueueName(qltWriterAckTimeout)
	if err != nil {
		return err
	}
	err = m.Qlt.AckQueueName()
	if err != nil {
		return err
	}
	log.Infoc(m.CtxS, "ready to send message", "queue", m.Conf.QueueName)
	return nil
}

func (m *QLTServerWriterConnection) IsAckAsync() bool {
	return true
}

func (q *QLTServerWriterConnection) DrainAcks() {
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

func (q *QLTServerWriterConnection) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent) {
	for {
		// log.Debugln(q.CtxS, "waiting msg to ack")
		event, ok := <-q.acks
		if !ok {
			log.Infoc(q.CtxS, "close ack loop")
			return
		}

		if q.Qlt == nil {
			log.Warnc(q.CtxS, "close warn not opened: sleeping")
			q.DrainAcks()
			continue
		}

		err := q.Qlt.WaitAck(qltWriterAckTimeout)
		if err != nil {
			log.Errorc(q.CtxS, "error waiting ack: close ack loop", "err", err)
			q.DrainAcks()
			return
		}

		acks <- event
	}
}

func (m *QLTServerWriterConnection) Ctx() string {
	return m.CtxS
}

func (q *QLTServerWriterConnection) IsActive() bool {
	return true
}

func (q *QLTServerWriterConnection) Write(events []processor.AckableEvent) (int, error) {
	n := 0
	// log.Debugln(q.CtxS, "Write events", "events", events)
	for _, event := range events {
		str, _ := event.Msg.(string)
		if err := q.Qlt.Send(str); err != nil {
			return n, err
		}
		// log.Debugln(q.CtxS, "Wrote", str)
		q.acks <- event
		n++
	}

	return n, nil
}

func (m *QLTServerWriterConnection) Close() error {
	err := m.Qlt.Close()
	if err != nil {
		log.Errorc(m.CtxS, "close error", "err", err)
	} else {
		log.Debugc(m.CtxS, "close")
	}
	return err
}
