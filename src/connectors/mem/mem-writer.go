package mem

import (
	"context"
	"errors"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
)

type MemWriter struct {
	Conf     *MemWriterConf
	Messages []string
	CtxS     string
}

type MemWriterConf struct {
	MaxSize int
}

func (c *MemWriterConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := MemWriter{c, nil, p.Name}

	conn, err := processor.GenProcessorHelperWriter(context, &q, p, ctl, inc, out)
	return conn, err
}

func (c *MemWriterConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *MemWriter) Ctx() string {
	return q.CtxS
}

func (q *MemWriter) Init(p *processor.Processor) error {
	return nil
}

func (q *MemWriter) PrepareEvent(event *processor.AckableEvent) (string, error) {
	str, b := event.Msg.(string)
	if !b {
		return "", errors.New("can't transform to string")
	}
	return str, nil
}

func (q *MemWriter) Write(events []processor.AckableEvent) (int, error) {
	i := 0
	datas := make([]string, len(events))
	log.Tracec(q.CtxS, "write msg count", "n", len(events))
	for _, e := range events {
		data, err := q.PrepareEvent(&e)
		if err != nil {
			continue
		}
		log.Tracec(q.CtxS, "write msg", "msg", data)
		datas[i] = data
		i++
	}
	datas = datas[:i]

	if q.Conf.MaxSize != 0 {
		q.Messages = append(q.Messages, datas...)
		n := len(q.Messages)
		if q.Conf.MaxSize > 0 && n > int(q.Conf.MaxSize) {
			q.Messages = q.Messages[n-q.Conf.MaxSize:]
		}
	}
	// log.Debugln(q.ctx, q.Messages)
	return len(events), nil
}

func (q *MemWriter) IsAckAsync() bool {
	return false
}

func (q *MemWriter) IsActive() bool {
	return true
}

func (q *MemWriter) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent, errs chan error) {
	log.Fatalc(q.CtxS, "Not supported")
}

func (q *MemWriter) Close() error {
	return nil
}
