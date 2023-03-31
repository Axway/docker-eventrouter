package processor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"axway.com/qlt-router/src/config"
	log "github.com/sirupsen/logrus"
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

type ControlEvent struct {
	From  *Processor
	From2 ConnectorRuntime
	Id    string
	Msg   string
}

func (msg *ControlEvent) Log() {
	log.Debugln(msg.From.Name+msg.From2.Ctx(), "(CTL-LOG)", msg.Id, msg.Msg)
}

func ControlEventDiscardAll(ctx context.Context, ctl chan ControlEvent) {
	log.Infoln("CTL-LOG", "DiscardAll")
	for {
		select {
		case <-ctl:
		case <-ctx.Done():
			return
		}
	}
}

func ControlEventLogAll(ctx context.Context, ctl chan ControlEvent) {
	log.Debugln("CTL-LOG", "Running")
	for {
		select {
		case msg := <-ctl:
			msg.Log()
		case <-ctx.Done():
			log.Debugln("CTL-LOG", "Stopping")
			return
		}
	}
}

type ConnectorRuntimeWriter interface {
	Ctx() string             // Context string mostly for logs
	Init(p *Processor) error // Call before any run
	// PrepareEvent(event *AckableEvent) (T, error)            // Transform Event before being consumed or queued
	Write(event []AckableEvent) error                        // Flush Bachted Data to be consumed
	IsAckAsync() bool                                        // Whether ack can be processes asynchonously
	ProcessAcks(ctx context.Context, acks chan AckableEvent) //
	Close() error                                            // Close EventConnector
}

type ConnectorRuntimeReader interface {
	Ctx() string             // Context string mostly for logs
	Init(p *Processor) error // Call before any run
	Read() ([]AckableEvent, error)
	AckMsg(msgid EventAck) //
	Close() error
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

var (
	ReaderAckSourceProxyChanSize = config.DeclareInt("processor.readerAckSourceProxyChanSize", 10, "Size of the reader ack channel")
	ReaderAckSourceWait          = config.DeclareDuration("processor.readerAckSourceWait", "1s", "Duration to wait before waiting ack message")
	WriterBatchSize              = config.DeclareInt("processor.writerBatchSize", 100, "Size of the Batch to writer connector")
)

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

func GenProcessorHelperReader(ctxz context.Context, p2 ConnectorRuntimeReader, p *Processor, ctl chan ControlEvent, in chan AckableEvent, out chan AckableEvent) (ConnectorRuntime, error) {
	ctxp := p.Name + "-" + p2.Ctx()
	sent := 0
	acked := 0

	log.Infoln(ctxp, "Initializing Reader...")
	ctl <- ControlEvent{p, p2, "STARTING", ""}

	src := &SourceProxy{make(chan EventAck, ReaderAckSourceProxyChanSize), ctxp}

	ackDone := make(chan interface{})

	err := p2.Init(p)
	if err != nil {
		return nil, err
	}

	log.Infoln(ctxp, "Starting Reader Proxy Ack Loop...")
	go func() {
		defer p2.Close()
		defer log.Infoln(ctxp, "Closing Acks...", "acked", acked, "sent", sent, "all_ack", p.Out_ack, "all_sent", p.Out)
		done := false
		for !done || acked != sent {
			// log.Infoln(ctxp, "Waiting Acks...")
			select {
			case msgid := <-src.ack:
				// log.Infoln(ctxp, "Ack2...", "msgId", msgid, "acked", acked, "sent", sent, "all_ack", p.Out_ack, "all_sent", p.Out)
				p2.AckMsg(msgid)
				acked++
				// FIXME: is this required
				atomic.AddInt64(&p.Out_ack, 1)
				// log.Infoln(ctxp, "Ack...", "msgId", msgid, "acked", acked, "sent", sent, "all_ack", p.Out_ack, "all_sent", p.Out)
				// ctl <- ControlEvent{p, p2, "ACK", "" + fmt.Sprint(acked, sent)}
				if acked == sent {
					ctl <- ControlEvent{p, p2, "ACK_DONE", ""}
					// FIXME: is this required
					if p.Out_ack == p.Out {
						ctl <- ControlEvent{p, p2, "ACK_ALL_DONE", ""}
					}
				}
			case <-ackDone:
				log.Infoln(ctxp, "Closing Acks...", fmt.Sprint(acked, sent))
				done = true
			case <-time.After(time.Duration(ReaderAckSourceWait) * time.Second):
				if acked != sent {
					log.Debugln(ctxp, "Waiting Ack Timeout...", "acked", acked, "sent", sent, "all_ack", p.Out_ack, "all_sent", p.Out)
				}
			}
		}
		log.Infoln(ctxp, "*** Closed Acks...")
	}()

	log.Infoln(ctxp, "Starting Reader Main Loop...")
	go func() {
		ctl <- ControlEvent{p, p2, "RUNNING", ""}
		lastAcked := -1
		for {
			// log.Debugln(ctxp, "Reading messages...")
			events, err := p2.Read()
			if err != nil {
				if errors.Is(err, io.EOF) {
					log.Infoln(ctxp, "Stopping Reader (EOF)...")
					ctl <- ControlEvent{p, p2, "STOPPED", "EOF"}
					return
				}
				log.Errorln(ctxp, "Error reading", "err", err)
				ctl <- ControlEvent{p, p2, "ERROR", err.Error()}
				return
			}
			// log.Debugln(ctxp, "Sending messages...", "batch", len(events), "acked", acked, "sent", sent, "all_ack", p.Out_ack, "all_sent", p.Out)
			if lastAcked != acked && acked == sent {
				ctl <- ControlEvent{p, p2, "PROCESSING", fmt.Sprint(len(events), acked, acked+len(events))}
				if p.Out == p.Out_ack {
					// FIXME: is this required ?
					ctl <- ControlEvent{p, p2, "ALL_PROCESSING", fmt.Sprint(len(events), p.Out_ack, p.Out_ack+int64(len(events)))}
				}
			}
			lastAcked = acked
			if len(events) == 0 {
				// FIXME: should progressively increase from smaller value
				time.Sleep(1 * time.Second)
			}
			for _, e := range events {
				out <- AckableEvent{src, e.Msgid, e.Msg, &e}
				sent++
				// FIXME: is this required ?
				atomic.AddInt64(&p.Out, 1)
			}
		}
	}()
	return p2, nil
}

// GenProcessorHelper
func GenProcessorHelperWriter(ctx context.Context, p2 ConnectorRuntimeWriter, p *Processor, ctl chan ControlEvent, in chan AckableEvent, out chan AckableEvent) (ConnectorRuntime, error) {
	ctxp := p.Name + "-" + p2.Ctx()
	p.Name = ctxp
	sent := 0
	acked := 0

	log.Infoln(ctxp, "Initializing Writer...")
	ctl <- ControlEvent{p, p2, "STARTING", "Writer"}

	err := p2.Init(p)
	if err != nil {
		ctl <- ControlEvent{p, p2, "ERROR", err.Error()}
		return nil, err
	}

	if p2.IsAckAsync() {
		log.Infoln(ctxp, "Starting Writer Ack Loops (async writer)...")
		acks := make(chan AckableEvent, 1000)

		go p2.ProcessAcks(ctx, acks)
		go func() {
			defer close(acks)
			for {
				select {
				case event := <-acks:

					// log.Debugln(ctxp, "Ack Event:", event)
					event.Src.AckMsg(event.Msgid)
					acked++
					atomic.AddInt64(&p.Out_ack, 1)
					// ctl <- ControlEvent{p, p2, "ACK", "" + fmt.Sprint(acked, sent)}
					if acked == sent {
						ctl <- ControlEvent{p, p2, "ACK_DONE", ""}
						// FIXME: is this required ?
						if p.Out_ack == p.Out {
							ctl <- ControlEvent{p, p2, "ACK_ALL_DONE", ""}
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()
		time.Sleep(1 * time.Second) // FIXME: beurk
	} else {
		log.Infoln(ctxp, "Not Starting Writer Proxy Ack Loop ! (sync writer)")
	}

	go func() {
		defer p2.Close()
		defer log.Infoln(ctxp, "Closing...")

		var events []AckableEvent
		// var toAckEvents []AckableEvent
		flush := false
		batchsize := WriterBatchSize
		donef := false

		ctl <- ControlEvent{p, p2, "RUNNING", ""}
		for !donef {
			flush = false
			if len(events) == 0 {
				// log.Debugln(ctxp, "Waiting messages...")
				select {
				case event := <-in:
					// data, _ := p2.PrepareEvent(&event)
					events = append(events, event)
					// toAckEvents = append(toAckEvents, event)
				case <-ctx.Done():
					flush = true
					log.Infoln(ctxp, "done")
					ctl <- ControlEvent{p, p2, "STOPPING", ""}
					donef = true
				}
			} else {
				// log.Debugln(ctxp, "Waiting more messages...", "batch", len(events))
				select {
				case event := <-in:
					// data, _ := p2.PrepareEvent(&event)
					events = append(events, event)
					// toAckEvents = append(toAckEvents, event)
				case <-ctx.Done():
					log.Infoln(ctxp, "done")
					ctl <- ControlEvent{p, p2, "STOPPING", ""}
					donef = true
				default:
					flush = true
				}
			}

			if flush || len(events) > batchsize {
				if acked == sent {
					ctl <- ControlEvent{p, p2, "PROCESSING", fmt.Sprint(len(events), acked, acked+len(events))}
					if p.Out == p.Out_ack {
						// FIXME: is this required
						ctl <- ControlEvent{p, p2, "ALL_PROCESSING", fmt.Sprint(len(events), p.Out_ack, p.Out_ack+int64(len(events)))}
					}
				}
				// log.Debugln(ctxp, "Writing messages...", "batch", len(events))
				err := p2.Write(events)
				if err != nil {
					log.Errorln(ctxp, "error writing messages...", "batch", len(events), "err", err)
					ctl <- ControlEvent{p, p2, "ERROR", ""}
					return
				}
				sent += len(events)
				// FIXME: is this required
				atomic.AddInt64(&p.Out, int64(len(events)))

				if !p2.IsAckAsync() {
					// log.Debugln(ctxp, "Sending async acks...", "batch", len(events))
					for _, event := range events {
						event.Src.AckMsg(event.Msgid)
					}
					// toAckEvents = nil
				}
				events = nil
			}
		}
		ctl <- ControlEvent{p, p2, "STOPPED", ""}
	}()

	return p2, nil
}
