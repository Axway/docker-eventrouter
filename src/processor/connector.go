package processor

import (
	"context"
)

type Connector interface {
	Start(ctx context.Context, p *Processor, ctl chan ControlEvent, cin chan AckableEvent, cout chan AckableEvent) (ConnectorRuntime, error)
	Clone() Connector
}

type ConnectorRuntime interface {
	Ctx() string
	Init(p *Processor) error
	Close() error
}

type ConnectorWithPrepare interface {
	PrepareEvent(event *AckableEvent) (string, error)
}

func PrepareEvents(q ConnectorWithPrepare, events []AckableEvent) []string {
	datas := make([]string, len(events))
	for i, e := range events {
		data, _ := q.PrepareEvent(&e)
		datas[i] = data
	}
	return datas
}
