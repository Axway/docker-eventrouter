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

func PrepareEvents(q ConnectorWithPrepare, events []AckableEvent) ([]string, int) {
	n := 0
	datas := make([]string, len(events))
	for _, e := range events {
		data, err := q.PrepareEvent(&e)
		if err == nil {
			datas[n] = data
			n++
		}
	}
	return datas[:n], len(events) - n
}
