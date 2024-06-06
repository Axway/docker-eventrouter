package processor

import (
	"context"

	log "axway.com/qlt-router/src/log"
)

type ControlEvent struct {
	From  *Processor
	From2 ConnectorRuntime
	Id    string
	Msg   string
}

func (msg *ControlEvent) Log() {
	log.Debugc(msg.From.Name, "(CTL-LOG)", "id", msg.Id, "from", msg.From.Name, "ctx", msg.From2.Ctx(), "msg", msg.Msg)
}

func ControlEventDiscardAll(ctxS string, ctx context.Context, ctl chan ControlEvent) {
	log.Infoc(ctxS, "CTL-LOG DiscardAll")
	for {
		select {
		case <-ctl:

		case <-ctx.Done():
			log.Debugc(ctxS, "CTL-LOG Stopping")
			return
		}
	}
}

func ControlEventLogSome(ctxS string, ctx context.Context, ctl chan ControlEvent) {
	log.Infoc(ctxS, "CTL-LOG LogSome")
	for {
		select {
		case msg := <-ctl:
			if msg.Id != "PROCESSING" {
				msg.Log()
			}

		case <-ctx.Done():
			log.Debugc(ctxS, "CTL-LOG Stopping")
			return
		}
	}
}

func ControlEventLogAll(ctxS string, ctx context.Context, ctl chan ControlEvent) {
	log.Debugc(ctxS, "CTL-LOG Running")
	for {
		select {
		case msg := <-ctl:
			msg.Log()
		case <-ctx.Done():
			log.Debugc(ctxS, "CTL-LOG Stopping")
			return
		}
	}
}
