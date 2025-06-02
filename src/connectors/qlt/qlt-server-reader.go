package qlt

import (
	"context"
	"errors"
	"net"
	"os"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"axway.com/qlt-router/src/protocols/qlt"
	"axway.com/qlt-router/src/tools"
)

var (
	_ processor.Connector              = &QLTServerReaderConf{}
	_ processor.ConnectorRuntime       = &QLTServerReader{}
	_ processor.ConnectorRuntimeReader = &QLTServerReaderConnection{}
)

type QLTServerReader struct {
	Conf     *QLTServerReaderConf
	ctx      string
	listener net.Listener
}

func (q *QLTServerReader) Init(p *processor.Processor) error {
	return nil
}

func (q *QLTServerReader) Ctx() string {
	return q.ctx
}

func (q *QLTServerReader) Close() error {
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

type QLTServerReaderConf struct {
	Host, Port, Cert, CertKey, Ca string
}

type QLTServerReaderConnection struct {
	CtxS   string
	Qlt    *qlt.QltServerReader
	MsgId  int64
	AckPos int64
	ack    chan int64
	From   string
}

func (conf *QLTServerReaderConf) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, in chan processor.AckableEvent, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	qltHandle := func(conn net.Conn, ctx2 string) {
		// qltHandleRequest(ctx, conn, ctx2+p.Flow.Name, p, ctl, out)
		qlt := qlt.NewQltServerReader(ctx2, conn)

		src := &QLTServerReaderConnection{CtxS: ctx2 + ".conn", Qlt: qlt, ack: make(chan int64), From: conn.RemoteAddr().String()}
		p.AddRuntime(src)
	}
	var listener net.Listener
	var err error
	if conf.Cert != "" {
		listener, err = tools.TlsServe(conf.Host+":"+conf.Port, conf.Cert, conf.CertKey, conf.Ca, qltHandle, false, "QLT-TLS")
	} else {
		listener, err = tools.TcpServe(conf.Host+":"+conf.Port, qltHandle, "QLT-TCP")
	}

	q := &QLTServerReader{conf, p.Name, listener}
	if err != nil {
		log.Errorc(q.ctx, "error starting listening server", "err", err)
	} else {
		log.Debugc(q.ctx, "listening server started", "host", conf.Host, "port", conf.Port)
	}
	return q, err
}

func (c *QLTServerReaderConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (m *QLTServerReaderConnection) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	c, e := processor.GenProcessorHelperReader(context, m, p, ctl, inc, out)
	return c, e
}

func (c *QLTServerReaderConnection) Clone() processor.Connector {
	return &*c
}

func (m *QLTServerReaderConnection) Init(p *processor.Processor) error {
	return nil
}

func (m *QLTServerReaderConnection) AckMsg(ack processor.EventAck) {
	msgid := ack.(int64)
	if m.AckPos+1 != msgid {
		panic("oups : already received or wrong")
	}
	if m.Qlt == nil {
		log.Errorc(m.CtxS, "closed")
		return
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

func (m *QLTServerReaderConnection) Ctx() string {
	return m.CtxS
}

func (m *QLTServerReaderConnection) IsServer() bool {
	return true
}

func (m *QLTServerReaderConnection) Read() ([]processor.AckableEvent, error) {
	events := make([]processor.AckableEvent, 1)
	if m.Qlt == nil {
		log.Warnc(m.CtxS, "closed")
		return nil, tools.ErrClosedConnection
	}
	msg, err := m.Qlt.Read(qltReaderBlockTimeout)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return nil, err
		}
		return nil, err
	}
	m.MsgId += 1
	events[0] = processor.AckableEvent{m, m.MsgId, msg, nil}
	return events, nil
}

func (m *QLTServerReaderConnection) Close() error {
	if m.Qlt == nil {
		log.Warnc(m.CtxS, "already closed")
		return nil
	}
	err := m.Qlt.Close()
	if err != nil {
		log.Errorc(m.CtxS, "close error", "err", err)
	} else {
		log.Debugc(m.CtxS, "close OK")
	}
	m.Qlt = nil
	return err
}
