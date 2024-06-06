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
	ReaderAckAllNotify           = config.DeclareDuration("processor.ReaderAckAllNotify", "10ms", "Duration to wait before waiting ack message")
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
		defer p.RemoveRuntime(p2)
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
				done = true
			case <-ctxz.Done():
				done = true
			case <-t.C:
				if acked != sent {
					log.Debugc(ctxp, "Waiting Ack Timeout...", "acked", acked, "sent", sent, "all_ack", p.Out_ack, "all_sent", p.Out)
				}
			}
			t.Stop()
			if acked == sent && lastAcked != acked {
				ctl <- ControlEvent{p, p2, "ACK_DONE", "" + fmt.Sprint(sent)}
				if p.Out_ack == p.Out {
					ctl <- ControlEvent{p, p2, "ACK_ALL_DONE", "" + fmt.Sprint(p.Out)}
				}
				lastAcked = acked
			}
		}
		log.Infoc(ctxp, "Stopping Reader Proxy Ack Loop...", "acked", acked, "sent", sent, "all_ack", p.Out_ack, "all_sent", p.Out)
	}()

	log.Infoc(ctxp, "Starting Reader Main Loop...")
	go func() {
		done := false
		retryFactor := 1
		ctl <- ControlEvent{p, p2, "RUNNING", ""}
		var lastAcked int64 = -1
		for {
			timeout := false
			// log.Debugc(ctxp, "Reading messages...")
			events, err := p2.Read()
			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					timeout = true
					// log.Debugc(ctxp, "IO Timeout")
				} else if errors.Is(err, io.EOF) {
					log.Infoc(ctxp, "No more event to read", "err", err)
					done = true
					ackDone <- 1
				} else {
					log.Errorc(ctxp, "Reading error", "err", err)
					ctl <- ControlEvent{p, p2, "ERROR", err.Error()}
					done = true
					ackDone <- 1
					break
				}
			}

			if len(events) == 0 && !timeout {
				delay := ReaderReadRetryDelay * time.Duration(retryFactor)
				if delay >= time.Minute {
					delay = time.Minute
				}

				t := time.NewTimer(delay)
				select {
				case <-ctxz.Done():
					done = true
				case <-t.C:
				}
				t.Stop()

				retryFactor = retryFactor * 2
			} else {
				retryFactor = 1
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
				ctl <- ControlEvent{p, p2, "STOPPING", ""}
				break
			}
		}
		log.Infoc(ctxp, "Stopping Reader Main Loop...")
		ctl <- ControlEvent{p, p2, "STOPPED", ""}
	}()
	return p2, nil
}
