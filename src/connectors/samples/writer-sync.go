package sample

import (
	"context"

	"axway.com/qlt-router/src/processor"
	log "github.com/sirupsen/logrus"
)

// replace SampleWriterSync

type SampleWriterSync struct {
	Conf     *SampleWriterSyncConf
	Messages []string
	CtxS     string
}

type SampleWriterSyncConf struct{}

func (c *SampleWriterSyncConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := SampleWriterSync{c, nil, p.Name}

	conn, err := processor.GenProcessorHelperWriter(context, &q, p, ctl, inc, out)
	return conn, err
}

func (c *SampleWriterSyncConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *SampleWriterSync) Ctx() string {
	return q.CtxS
}

func (q *SampleWriterSync) Init(p *processor.Processor) error {
	return nil
}

func (q *SampleWriterSync) Write(events []processor.AckableEvent) error {
	datas := make([]string, len(events))
	for i, e := range events {
		str, _ := e.Msg.(string)
		datas[i] = str
	}
	q.Messages = append(q.Messages, datas...)
	return nil
}

func (q *SampleWriterSync) IsAckAsync() bool {
	return false
}

func (q *SampleWriterSync) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent) {
	log.Fatal("Not supported")
}

func (q *SampleWriterSync) Close() error {
	return nil
}
