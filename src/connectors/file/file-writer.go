package file

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"axway.com/qlt-router/src/processor"
	"axway.com/qlt-router/src/tools"

	log "axway.com/qlt-router/src/log"
)

type FileStoreRawWriterConfig struct {
	FilenamePrefix string
	FilenameSuffix string
	MaxFile        int
	MaxSize        int64 /* MB */
}

type FileStoreRawWriter struct {
	Conf *FileStoreRawWriterConfig

	CtxS     string
	Filename string
	file     *os.File
	Size     int64
}

var count = 0

func (c *FileStoreRawWriterConfig) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	q := &FileStoreRawWriter{}
	q.Conf = c
	count++
	q.CtxS = p.Name + fmt.Sprint("-", count)
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
	/* Find file to use */
	/* Looking for oldest file with prefix */
	filename := tools.FileToUse(q.CtxS, q.Conf.FilenamePrefix, q.Conf.FilenameSuffix)
	q.Filename = filename
	return q.Open()
}

func (q *FileStoreRawWriter) Open() error {
	if q.file != nil {
		panic("multiple open")
	}

	log.Infoc(q.CtxS, "opening file...", "filename", q.Filename)
	f, err := os.OpenFile(q.Filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Errorc(q.CtxS, "error opening file for appending", "filename", q.Filename, "err", err)
		return err
	}
	q.file = f

	/* go to current position */
	offset, err := q.file.Seek(0, io.SeekCurrent)
	if err != nil {
		log.Errorc(q.CtxS, "error get position", "filename", q.Filename, "err", err)
		return err
	}
	q.Size = offset

	return nil
}

func (q *FileStoreRawWriter) Switch() error {
	newfilename, err := tools.FileSwitch(q.CtxS, q.Conf.FilenamePrefix, q.Conf.FilenameSuffix, q.Conf.MaxFile)
	if err != nil {
		return err
	}

	err = q.file.Close()
	if err != nil {
		log.Warnc(q.CtxS, "error while switching : close error", "filename", q.Filename, "err", err)
	}
	q.Filename = newfilename

	q.file = nil
	err = q.Open()
	return err
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
	buf := []byte(strings.Join(datas, "\n") + "\n")

	q.Size += int64(len(buf))
	if q.Conf.MaxSize > 0 && q.Size > (q.Conf.MaxSize * 1048576) {
		log.Debugc(q.CtxS, "switching", "filename", q.Filename, "size", q.Size, "maxsize", q.Conf.MaxSize * 1048576)
		q.Switch()

		buf = []byte(strings.Join(datas, "\n") + "\n")
		q.Size = int64(len(buf))
	}

	if _, err := q.file.Write(buf); err != nil {
		log.Errorc(q.CtxS, "error write message in file", "filename", q.Filename, "err", err)
		return err
	}

	return nil
}

func (q *FileStoreRawWriter) IsAckAsync() bool {
	return false
}

func (q *FileStoreRawWriter) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent) {
	log.Fatalc(q.CtxS, "Not supported")
}

func (q *FileStoreRawWriter) Close() error {
	return q.file.Close()
}
