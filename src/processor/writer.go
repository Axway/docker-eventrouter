package processor

import (
	"context"
	"sync/atomic"

	"axway.com/qlt-router/src/config"
	log "axway.com/qlt-router/src/log"
)

var WriterBatchSize = config.DeclareInt("processor.writerBatchSize", 10, "Size of the Batch to writer connector")

type ConnectorRuntimeWriter interface {
	Ctx() string                                             // Context string mostly for logs
	Init(p *Processor) error                                 // Initialization before main runtime, when complet message are ready to be sent
	Write(event []AckableEvent) error                        // Flush Batched Data to be consumed
	IsAckAsync() bool                                        // Whether acks can be processed asynchronously
	ProcessAcks(ctx context.Context, acks chan AckableEvent) // When write acks are asynchronous, acked event are sent through this channel
	Close() error                                            // Close EventConnector (only when init is successful)
}

// GenProcessorHelper
func GenProcessorHelperWriter(ctx context.Context, p2 ConnectorRuntimeWriter, p *Processor, ctl chan ControlEvent, in chan AckableEvent, out chan AckableEvent) (ConnectorRuntime, error) {
	ctxp := p.Name + "-" + p2.Ctx()
	p.Name = ctxp
	sent := 0
	acked := 0

	log.Infoc(ctxp, "Initializing Writer...")
	ctl <- ControlEvent{p, p2, "STARTING", "Writer"}

	err := p2.Init(p)
	if err != nil {
		ctl <- ControlEvent{p, p2, "ERROR", err.Error()}
		return nil, err
	}

	if p2.IsAckAsync() {
		log.Infoc(ctxp, "Starting Writer Ack Loops (async writer)...")
		acks := p.Chans.Create(ctxp+"WriterAsyncAck", 1000)
		// acks := make(chan AckableEvent, 1000)

		go p2.ProcessAcks(ctx, acks.C)
		go func() {
			defer close(acks.C)
			for {
				select {
				case event := <-acks.C:

					// log.Debugln(ctxp, "Ack Event:", event)
					event.Src.AckMsg(event.Msgid)
					acked++
					atomic.AddInt64(&p.Out_ack, 1)
					// ctl <- ControlEvent{p, p2, "ACK", "" + fmt.Sprint(acked, sent)}
					// FIXME: not ALL_ALL_DONE on writer ?
					/*if acked == sent {
						ctl <- ControlEvent{p, p2, "ACK_DONE", ""}
						// FIXME: is this required ?
						if p.Out_ack == p.Out {
							ctl <- ControlEvent{p, p2, "ACK_ALL_DONE", ""}
						}
					}*/
				case <-ctx.Done():
					return
				}
			}
		}()
		// should not be required : time.Sleep(1 * time.Second)
	} else {
		log.Infoc(ctxp, "Not Starting Writer Proxy Ack Loop ! (sync writer)")
	}

	go func() {
		defer p2.Close()
		defer log.Infoc(ctxp, "Closing...(auto)")

		var events []AckableEvent
		// var toAckEvents []AckableEvent
		flush := false
		batchsize := WriterBatchSize
		donef := false
		log.Infoc(ctxp, "Running")
		ctl <- ControlEvent{p, p2, "RUNNING", ""}
		for !donef {
			flush = false
			if len(events) == 0 {
				// No event bached, wait for new events (or termination)
				// log.Debugln(ctxp, "Waiting messages...")
				select {
				case event := <-in:
					// data, _ := p2.PrepareEvent(&event)
					events = append(events, event)
					// toAckEvents = append(toAckEvents, event)
				case <-ctx.Done():
					flush = true
					log.Infoc(ctxp, "done")
					ctl <- ControlEvent{p, p2, "STOPPING", ""}
					donef = true
				}
			} else {
				// Some events in batched queue, try to enqueue more if available
				// log.Debugln(ctxp, "Waiting more messages...", "batch", len(events))
				select {
				case event := <-in:
					// data, _ := p2.PrepareEvent(&event)
					events = append(events, event)
					// toAckEvents = append(toAckEvents, event)
				case <-ctx.Done():
					log.Infoc(ctxp, "stppping")
					ctl <- ControlEvent{p, p2, "STOPPING", ""}
					donef = true
				default:
					flush = true
				}
			}

			// Send batched events
			if flush || len(events) > batchsize {
				/*if acked == sent {
					ctl <- ControlEvent{p, p2, "PROCESSING", fmt.Sprint(len(events), acked, acked+len(events))}
					if p.Out == p.Out_ack {
						// FIXME: is this required
						ctl <- ControlEvent{p, p2, "ALL_PROCESSING", fmt.Sprint(len(events), p.Out_ack, p.Out_ack+int64(len(events)))}
					}
				}*/
				log.Tracec(ctxp, "writer writing messages...", "batch", len(events))
				err := p2.Write(events)
				if err != nil {
					log.Errorc(ctxp, "error writing messages...", "batch", len(events), "err", err)
					ctl <- ControlEvent{p, p2, "ERROR", ""}
					return
				}
				sent += len(events)
				// FIXME: is this required ?
				atomic.AddInt64(&p.Out, int64(len(events)))

				if !p2.IsAckAsync() {
					// log.Debugln(ctxp, "Sending async acks...", "batch", len(events))
					for _, event := range events {
						event.Src.AckMsg(event.Msgid)
					}
					atomic.AddInt64(&p.Out_ack, int64(len(events)))
					// toAckEvents = nil
				}
				events = nil
			}
		}
		log.Infoc(ctxp, "Stopped")
		ctl <- ControlEvent{p, p2, "STOPPED", ""}
	}()

	return p2, nil
}
