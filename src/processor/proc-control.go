package processor

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type (
	ControlConf    struct{ DelayMS int }
	ControlRuntime struct {
		Conf   *ControlConf
		Count  int64
		HasMsg int
		Status string

		SizeIn  int
		SizeOut int
		Loop    int
		Speed   float64
	}
)

func (conf *ControlConf) Start(ctx context.Context, p *Processor, ctl chan ControlEvent, in chan AckableEvent, out chan AckableEvent) (ConnectorRuntime, error) {
	c := &ControlRuntime{
		Conf:   conf,
		HasMsg: -1,
	}
	avg := 0.1
	ctxname := "[control] " + p.Flow.Name
	ctl <- ControlEvent{p, c, "RUNNING", ""}
	c.Status = "RUNNING"

	go func() {
		lastTime := time.Now().UnixMilli()
		lastCount := int64(0)
		for {

			c.Loop++
			c.SizeIn = len(in)
			c.SizeOut = len(out)

			n := time.Now().UnixMilli()
			if n-lastTime > 500 {
				c.Speed = (1.0-avg)*float64(c.Count-lastCount)*1000.0/float64(n-lastTime) + avg*c.Speed

				lastTime = n
				lastCount = c.Count
			}
			select {
			case event := <-in:
				if c.HasMsg <= 0 {
					log.Warnln(ctxname, "restart message", c.Count, c.HasMsg)
				}
				c.Count++
				c.HasMsg++
				c.SizeIn = len(in)
				c.SizeOut = len(out)
				time.Sleep(time.Millisecond * time.Duration(conf.DelayMS))
			loopSend:
				for {
					select {
					case out <- event:
						break loopSend
					case <-time.After(1 * time.Second):
						ctl <- ControlEvent{p, c, "STUCK", ""}
						c.Status = "STUCK"
						log.Warnln(ctxname, "stuck")
					}
					c.SizeIn = len(in)
					c.SizeOut = len(out)
				}
			case <-ctx.Done():
				ctl <- ControlEvent{p, c, "STOPPED", ""}
				c.Status = "STOPPED"
				return
			case <-time.After(1 * time.Second):
				if c.HasMsg > 0 {
					log.Warnln(ctxname, "no message", c.Count, c.HasMsg)
					ctl <- ControlEvent{p, c, "NO_MESSAGE", ""}
					c.Status = "NO_MESSAGE"
					c.HasMsg = 0
				}
			}
		}
	}()
	return c, nil
}

func (c *ControlConf) Clone() Connector {
	c2 := *c
	return &c2
}

func (q *ControlRuntime) Init(p *Processor) error {
	return nil
}

func (r *ControlRuntime) Ctx() string {
	return "control"
}

func (r *ControlRuntime) Close() error { return nil }

func Dispatch(ctx context.Context, p *Processor, ctl chan ControlEvent, in chan AckableEvent, outs *[]chan AckableEvent) {
	count := 0
	hasMsg := -1
	ctl <- ControlEvent{p, nil, "RUNNING", ""}
	for {
		select {
		case qltMessage := <-in:
			if hasMsg <= 0 {
				log.Warnln("dispatch: restart message", count, hasMsg)
			}
			count++
			hasMsg++

			msg, b := qltMessage.Msg.(map[string]string)

			if b && msg["broker"] == "" {
				if msg != nil {
					msg["broker"] = "qlt"
					msg["timestamp"] = time.Now().Format(time.RFC3339Nano)
				}
				// msg["axway-target-flow"] = "api-central-v8" // Condor
				// msg["captureOrgID"] = "trcblt-test"         // tenant
				for idx, ch := range *outs {
					// log.Debugln(qltMessage.qltEvent.qlt.ctxMsg(), "[", qltMessage.qltEvent.msgid, "] Pushing to Queue... ", idx)
				loopSelect:
					for {
						select {
						case ch <- qltMessage:
							break loopSelect
						case <-time.After(1 * time.Second):
							ctl <- ControlEvent{p, nil, "STUCK", ""}
							log.Warnln("dispatch: stuck", idx)
						}
					}
				}
			} else {
				log.Debugln(qltMessage.Src.Ctx(), "[", qltMessage.Msgid, "] Skip Message...", msg["broker"])
			}

			// qltMessage.qltEvent.Ack <- true
		case <-ctx.Done():
			ctl <- ControlEvent{p, nil, "STOPPED", ""}
			return
		case <-time.After(1 * time.Second):
			if hasMsg > 0 {
				log.Warnln("dispatch: no message", count, hasMsg)
				ctl <- ControlEvent{p, nil, "NO_MESSAGE", ""}
				hasMsg = 0
			}
		}
	}
}

func fanInOrdered(ctx context.Context, name string, ctl chan ControlEvent, in []chan AckableEvent, out chan AckableEvent) {
	count := 0
	n := len(in)
	msgid := EventAck(int64(-1))
	unordered := 0
	ctl <- ControlEvent{nil, nil, "RUNNING", ""}
	for {
		event := <-in[count%n]
		out <- event

		offset1, ok1 := msgid.(int64)
		offset2, ok2 := msgid.(int64)
		if ok1 && ok2 && offset1 > offset2 {
			unordered++
			log.Warnln(name, "unordered message", msgid, event.Msgid, unordered, unordered*100.0/count)
		}
		msgid = event.Msgid

		count++
	}
}

func fanOutOrdered(ctx context.Context, name string, ctl chan ControlEvent, in chan AckableEvent, out []chan AckableEvent) {
	count := 0
	n := len(out)
	ctl <- ControlEvent{nil, nil, "RUNNING", ""}
	for {
		event := <-in
		out[count%n] <- event
		count++
	}
}

func ParallelOrdered(ctx context.Context, name string, n int, ctl chan ControlEvent, in, out chan AckableEvent, channels *Channels, p *Processor) {
	var ins []chan AckableEvent
	var outs []chan AckableEvent

	for i := 0; i < n; i++ {
		inchan := channels.Create(name+"-parallel-ordered-in-"+fmt.Sprint(i), flowChannelSize)
		ins = append(ins, inchan.C)

		outchan := channels.Create(name+"-parallel-ordered-out"+fmt.Sprint(i), flowChannelSize)
		outs = append(outs, outchan.C)

		go p.Conf.Start(ctx, p, ctl, inchan.C, outchan.C)
	}
	go fanOutOrdered(ctx, name+"-parallel-fanint-ordered", ctl, in, ins)
	go fanInOrdered(ctx, name+"-parallel-fanout-ordered", ctl, outs, out)
}

func ParallelUnordered(ctx context.Context, name string, n int, ctl chan ControlEvent, in, out chan AckableEvent, channels *Channels, p *Processor) {
	for i := 0; i < n; i++ {
		go p.Conf.Start(ctx, p, ctl, in, out)
	}
}
