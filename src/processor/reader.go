package processor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"axway.com/qlt-router/src/config"
	log "axway.com/qlt-router/src/log"
)

var (
	ReaderAckSourceProxyChanSize = config.DeclareInt("processor.readerAckSourceProxyChanSize", 10, "Size of the reader ack channel")
	ReaderAckAllNotify           = config.DeclareDuration("processor.readerAckSourceWait", "10ms", "Duration to wait before waiting ack message")
	ReaderAckSourceWait          = config.DeclareDuration("processor.readerAckSourceWait", "1s", "Duration to wait before waiting ack message")
	ReaderReadRetryDelay         = config.DeclareDuration("processor.readerReadRetryDelay", "200ms", "Duration to wait before retrying reading")
)

type ConnectorRuntimeReader interface {
	Ctx() string                   // Context string mostly for logs
	Init(p *Processor) error       // Initialization before main runtime, when complete message ready to be sent
	Read() ([]AckableEvent, error) // ReadMessage
	AckMsg(msgid EventAck)         // AckMessage
	Close() error                  // Close Connector (only when init is successful)
}

type SourceProxy struct {
	ack chan EventAck
	ctx string
}

func (c *SourceProxy) AckMsg(msgid EventAck) {
	c.ack <- msgid
}

func (c *SourceProxy) Ctx() string {
	return c.ctx
}

func GenProcessorHelperReader(ctxz context.Context, p2 ConnectorRuntimeReader, p *Processor, ctl chan ControlEvent, in chan AckableEvent, out chan AckableEvent) (ConnectorRuntime, error) {
	ctxp := p.Name + "-" + p2.Ctx()
	var sent int64
	var acked int64

	log.Infoc(ctxp, "Initializing Reader...")
	ctl <- ControlEvent{p, p2, "STARTING", ""}

	p.Chans.Create(ctxp+"ReaderAsyncAckProxy - FIXME/not tracked", 1000) // FIXME: not tracked
	acks := make(chan EventAck, ReaderAckSourceProxyChanSize)            // FIXME: not tracked
	src := &SourceProxy{acks, ctxp}

	ackDone := make(chan interface{})

	err := p2.Init(p)
	if err != nil {
		return nil, err
	}

	p.InitializePrometheusCounters()

	log.Infoc(ctxp, "Starting Reader Proxy Ack Loop...")
	go func() {
		defer p2.Close()
		defer log.Infoc(ctxp, "Closing Acks...", "acked", acked, "sent", sent, "all_ack", p.Out_ack, "all_sent", p.Out)
		done := false
		var lastAcked int64 = -1
		nextWait := ReaderAckSourceWait
		for !(done && acked == sent) {
			// log.Infoln(ctxp, "Waiting Acks...")
			t := time.NewTimer(nextWait)
			nextWait = ReaderAckSourceWait
			select {
			case msgid := <-src.ack:
				// log.Infoln(ctxp, "Ack2...", "msgId", msgid, "acked", acked, "sent", sent, "all_ack", p.Out_ack, "all_sent", p.Out)
				p2.AckMsg(msgid)
				atomic.AddInt64(&acked, 1)
				atomic.AddInt64(&p.Out_ack, 1)
				p.OutAckCounter.Inc()
				// log.Infoln(ctxp, "Ack...", "msgId", msgid, "acked", acked, "sent", sent, "all_ack", p.Out_ack, "all_sent", p.Out)
				// ctl <- ControlEvent{p, p2, "ACK", "" + fmt.Sprint(acked, sent)}
				if acked == sent {
					nextWait = ReaderAckAllNotify
				}
			case <-ackDone:
				log.Infoc(ctxp, "Closing Acks...", "acked", acked, "sent", sent)
				done = true
			case <-t.C:
				if acked != sent {
					log.Debugc(ctxp, "Waiting Ack Timeout...", "acked", acked, "sent", sent, "all_ack", p.Out_ack, "all_sent", p.Out)
				} else if lastAcked != acked {
					ctl <- ControlEvent{p, p2, "ACK_DONE", "" + fmt.Sprint(sent)}
					if p.Out_ack == p.Out {
						ctl <- ControlEvent{p, p2, "ACK_ALL_DONE", "" + fmt.Sprint(p.Out)}
					}
					lastAcked = acked
				}
			}
			t.Stop()
		}
		log.Infoc(ctxp, "*** Closed Acks...")
	}()

	// Trap reader cancellation in done variable
	done := false
	go func() {
		<-ctxz.Done()
		done = true
		log.Debugc(ctxp, "done")
	}()

	log.Infoc(ctxp, "Starting Reader Main Loop...")
	go func() {
		ctl <- ControlEvent{p, p2, "RUNNING", ""}
		var lastAcked int64 = -1
		for {
			timeout := false
			// log.Debugln(ctxp, "Reading messages...")
			events, err := p2.Read()
			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					timeout = true
					// log.Debugc(ctxp, "IO Timeout")
				} else if !errors.Is(err, io.EOF) {
					log.Errorc(ctxp, "Error reading", "err", err)
					ctl <- ControlEvent{p, p2, "WARN", err.Error()}
					// return Retry on error
				}
			}

			if len(events) == 0 && !timeout {
				// FIXME: should progressively increase from smaller value
				// time.Sleep(ReaderReadRetryDelay)
				t := time.NewTimer(ReaderReadRetryDelay)
				select {
				case <-ctxz.Done():
					done = true
				case <-t.C:
				}
				t.Stop()
			}
			for _, e := range events {
				log.Tracec(ctxp, "reader read", "msg", e.Msg.(string))
				out <- AckableEvent{src, e.Msgid, e.Msg, &e}
				sent++
				// FIXME: is this required ?
				atomic.AddInt64(&p.Out, 1)
				p.OutCounter.Inc()
				p.OutDataCounter.Add(float64(len(e.Msg.(string))))
			}
			// log.Debugln(ctxp, "Sending messages...", "batch", len(events), "acked", acked, "sent", sent, "all_ack", p.Out_ack, "all_sent", p.Out)
			if sent != 0 && lastAcked != acked && acked == sent {
				ctl <- ControlEvent{p, p2, "PROCESSING", fmt.Sprint(len(events), acked, acked+int64(len(events)))}
				if p.Out == p.Out_ack {
					// FIXME: is this required ?
					ctl <- ControlEvent{p, p2, "ALL_PROCESSING", fmt.Sprint(len(events), p.Out_ack, p.Out_ack+int64(len(events)))}
				}
			}
			lastAcked = acked
			if done {
				log.Infoc(ctxp, "stopping reading event")
				ctl <- ControlEvent{p, p2, "STOPPING", ""}
				break
			}
		}
		log.Infoc(ctxp, "stopped")
		ctl <- ControlEvent{p, p2, "STOPPED", ""}
	}()
	return p2, nil
}
