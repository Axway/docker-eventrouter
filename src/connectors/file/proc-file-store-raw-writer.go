package file

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"axway.com/qlt-router/src/processor"
	log "github.com/sirupsen/logrus"
)

type FileStoreRawWriterConfig struct {
	Filename string
}

type FileStoreRawWriter struct {
	Conf *FileStoreRawWriterConfig

	CtxS     string
	Filename string
	file     *os.File
	lf       string
}

func (c *FileStoreRawWriterConfig) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := &FileStoreRawWriter{}
	q.Conf = c
	q.CtxS = p.Name
	return processor.GenProcessorHelperWriter(context, q, p, ctl, inc, out)
}

func (c *FileStoreRawWriterConfig) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *FileStoreRawWriter) Ctx() string {
	return q.CtxS
}

func (q *FileStoreRawWriter) Init(p *processor.Processor) error {
	q.Filename = q.Conf.Filename
	log.Println(q.CtxS, "opening file...", "filename", fmt.Sprintf("%p", q.Conf), q.Conf.Filename)
	f, err := os.OpenFile(q.Filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Errorln(q.CtxS, "error opening file for appending", "filename", q.Filename, "err", err)
		return err
	}
	q.file = f
	offset, err := q.file.Seek(0, io.SeekCurrent)
	if err != nil {
		log.Errorln(q.CtxS, "error get position", "filename", q.Filename, "err", err)
		return err
	}
	if offset != 0 {
		q.lf = "\n"
	}
	return nil
}

func (q *FileStoreRawWriter) PrepareEvent(event *processor.AckableEvent) (string, error) {
	str, b := event.Msg.(string)
	if !b {
		str = event.Orig.Msg.(string)
	}
	str = strings.ReplaceAll(str, "\n", "")
	return str, nil
}

func (q *FileStoreRawWriter) Write(events []processor.AckableEvent) error {
	datas := processor.PrepareEvents(q, events)

	// log.Debugln(q.CtxS, "writing", "count", len(datas))

	if _, err := q.file.Write([]byte(q.lf + strings.Join(datas, "\n"))); err != nil {
		log.Errorln(q.CtxS, "error write message in file", "filename", q.Filename, "err", err)
		return err
	}
	q.lf = "\n"

	return nil
}

func (q *FileStoreRawWriter) IsAckAsync() bool {
	return false
}

func (q *FileStoreRawWriter) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent) {
	log.Fatal("Not supported")
}

func (q *FileStoreRawWriter) Close() error {
	return q.file.Close()
}
