package file

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
)

type FileStoreJsonConsumerConfig struct {
	Filename string
}

type FileStoreJsonConsumer struct {
	ctx string
}

func (conf *FileStoreJsonConsumerConfig) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, FileStoreQueue chan processor.AckableEvent, out chan processor.AckableEvent) {
	var q FileStoreJsonConsumer
	q.ctx = "[FS-JSON] " + p.Flow.Name
	filename := conf.Filename + "." + p.Flow.Name
	log.Println(q.ctx, "Opening file", filename, "...")
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Errorln(q.ctx, "Error opening file for appending", filename, err)
		log.Fatal(err)
	}
	defer f.Close()

	count := 0
	var lines []string
	var events []processor.AckableEvent

	done := ctx.Done()
	for {
		flush := false
		// log.Println("[FS] Waiting MessageMessage on FSQueue...")
		if len(lines) == 0 {
			select {
			case event := <-FileStoreQueue:
				buf, _ := json.Marshal(event.Msg.(map[string]string))
				lines = append(lines, string(buf))
				events = append(events, event)
			case <-done:
				log.Infoln(q.ctx, "done")
				return
			}
		} else {
			select {
			case event := <-FileStoreQueue:
				buf, _ := json.Marshal(event.Msg.(map[string]string))
				lines = append(lines, string(buf))
				events = append(events, event)
			default:
				flush = true
			}
		}

		// log.Println("[FS] Marshalling Message")

		// log.Println("[FS] Message", string(buf))
		if flush {
			count += len(lines)
			// log.Debugln("[FS] write messsage to file", len(lines), count)
			if _, err := f.Write([]byte(strings.Join(lines, "\n") + "\n")); err != nil {
				log.Errorln(q.ctx, "Error write message in file", filename, err)
				log.Fatal(err)
			}
			for _, event := range events {
				event.Src.AckMsg(event.Msgid)
			}
			lines = nil
			events = nil
		}
	}
}
