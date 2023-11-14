package processor

import (
	"context"
	"sync/atomic"
	"time"

	"axway.com/qlt-router/src/config"
	log "axway.com/qlt-router/src/log"
)

var WriterBatchSize = config.DeclareInt("processor.writerBatchSize", 10, "Size of the Batch to writer connector")

type ConnectorRuntimeWriter interface {
	Ctx() string                                             // Context string mostly for logs
	Init(p *Processor) error                                 // Initialization before main runtime, when complet message are ready to be sent
	Write(event []AckableEvent) (int, error)                 // Flush Batched Data to be consumed
	IsAckAsync() bool                                        // Whether acks can be processed asynchronously
	IsActive() bool                                          // Whether the connector is in an active state
	ProcessAcks(ctx context.Context, acks chan AckableEvent) // When write acks are asynchronous, acked event are sent through this channel
	Close() error                                            // Close EventConnector (only when init is successful)
}

func removeAckableFromList(l []AckableEvent, e AckableEvent) []AckableEvent {
	/* Remove element from the waiting for Ack list */
	for i := 0; i < len(l); i++ {
		if e.Msgid == l[i].Msgid {
			if i == len(l)-1 {
				l = l[:i]
			} else if i == 0 {
				l = l[i+1:]
			} else {
				l = append(l[:i], l[i+1:]...)
			}
			break
		}
	}
	return l
}

// GenProcessorHelper
func GenProcessorHelperWriter(ctx context.Context, p2 ConnectorRuntimeWriter, p *Processor, ctl chan ControlEvent, in chan AckableEvent, out chan AckableEvent) (ConnectorRuntime, error) {
	ctxp := p.Name + "-" + p2.Ctx()
	p.Name = ctxp
	sent := 0
	acked := 0
	var ackPendingEvents []AckableEvent

	acksReceived := p.Chans.Create(ctxp+"WriterAcks", 1000)

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
					acksReceived.C <- event
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
					log.Infoc(ctxp, "Done")
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
		retryFactor := 1
		log.Infoc(ctxp, "Running")
		ctl <- ControlEvent{p, p2, "RUNNING", ""}
		for !donef {
			flush = false

			// If not active and ackPendingEvents something went wrong
			// try to resend ackPendingEvents
			if len(ackPendingEvents) > 0 && !p2.IsActive() {
				log.Debugc(ctxp, "retry old events")
				events = append(ackPendingEvents, events...)
				ackPendingEvents = nil
			}

			if len(events) == 0 {
				// No event bached, wait for new events (or termination)
				select {
				case event := <-acksReceived.C:
					/* Remove element from the waiting for Ack list */
					ackPendingEvents = removeAckableFromList(ackPendingEvents, event)
				case event := <-in:
					// data, _ := p2.PrepareEvent(&event)
					events = append(events, event)
				case <-ctx.Done():
					flush = true
					log.Infoc(ctxp, "done")
					ctl <- ControlEvent{p, p2, "STOPPING", ""}
					donef = true
				}
			} else {
				// Some events in batched queue, try to enqueue more if available
				select {
				case event := <-acksReceived.C:
					/* Remove element from the waiting for Ack list */
					ackPendingEvents = removeAckableFromList(ackPendingEvents, event)
				case event := <-in:
					// data, _ := p2.PrepareEvent(&event)
					events = append(events, event)
				case <-ctx.Done():
					log.Infoc(ctxp, "stopping")
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

				n, err := p2.Write(events)
				sent += n
				if err != nil {
					log.Errorc(ctxp, "error writing messages...", "batch", n, "total", len(events), "err", err)
					ctl <- ControlEvent{p, p2, "ERROR", ""}

					delay := 10 * time.Millisecond * time.Duration(retryFactor)
					if delay >= time.Minute {
						delay = time.Minute
					}
					time.Sleep(delay)
					retryFactor = retryFactor * 2
				}

				/* add n written elements to ackPendingEvents */
				if p2.IsAckAsync() && n > 0 {
					ackPendingEvents = append(ackPendingEvents, events[:n]...)
				}
				// FIXME: is this required ?
				atomic.AddInt64(&p.Out, int64(n))

				if !p2.IsAckAsync() {
					// log.Debugln(ctxp, "Sending async acks...", "batch", len(events))
					for _, event := range events[:n] {
						event.Src.AckMsg(event.Msgid)
					}
					atomic.AddInt64(&p.Out_ack, int64(n))
					// toAckEvents = nil
				}

				if n != len(events) { /* error case */
					/* remove n already written elements from events */
					events = events[n:]

					continue // If we fail to send, we need to retry later
				}
				retryFactor = 1
				events = nil
			}
		}
		log.Infoc(ctxp, "Stopped")
		ctl <- ControlEvent{p, p2, "STOPPED", ""}
	}()

	return p2, nil
}
