package file

import (
	"context"
	"os"
	"strings"

	"axway.com/qlt-router/src/processor"
	log "github.com/sirupsen/logrus"
)

type FileStoreRawConsumerConfig struct {
	Filename string
}

type FileStoreRawConsumer struct {
	ctx      string
	filename string
	conf     *FileStoreRawConsumerConfig
	file     *os.File
}

func (conf *FileStoreRawConsumerConfig) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, FileStoreQueue chan processor.AckableEvent, out chan processor.AckableEvent) {
	var q FileStoreRawConsumer
	q.ctx = "[FS-RAW] " + p.Flow.Name
	filename := conf.Filename + "." + p.Flow.Name
	log.Println("[FS-RAW] Opening file", filename, "...")
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Errorln("[FS-RAW] Error opening file for appending", filename, err)
		log.Fatal(err)
	}
	defer f.Close()
	count := 0
	var lines []string
	var events []processor.AckableEvent

	done := ctx.Done()

	for {
		flush := false
		// log.Debugln("[FS-RAW] Waiting MessageMessage on FSQueue...", count)
		if len(lines) == 0 {
			select {
			case event := <-FileStoreQueue:
				str, b := event.Msg.(string)
				if !b {
					str = event.Orig.Msg.(string)
				}
				str = strings.ReplaceAll(str, "\n", "") // remove all line return (one line per message)
				lines = append(lines, str)
				events = append(events, event)
			case <-done:
				log.Infoln(q.ctx, "done")
				return
			}
		} else {
			select {
			case event := <-FileStoreQueue:
				str, b := event.Msg.(string)
				if !b {
					str = event.Orig.Msg.(string)
				}
				str = strings.ReplaceAll(str, "\n", "") // remove all line return (one line per message)
				lines = append(lines, str)
				events = append(events, event)
			default:
				flush = true
			}
		}

		// log.Println("[FS-RAW] Marshalling Message", event.msgid)

		if flush {
			count += len(lines)
			// log.Println("[FS-RAW] Message", string(buf))
			if _, err := f.Write([]byte(strings.Join(lines, "\n") + "\n")); err != nil {
				log.Errorln("[FS-RAW] Error write message in file", filename, err)
				log.Fatal(err)
			}
			for _, event := range events {
				event.Src.AckMsg(event.Msgid)
			}
			lines = nil
			events = nil
		}
		// event.qltEvent.qlt.ackMsg(event.qltEvent.msgid)
	}
}
