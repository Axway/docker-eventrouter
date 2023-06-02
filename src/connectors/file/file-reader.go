package file

import (
	"context"
	"os"
	"strings"

	"axway.com/qlt-router/src/processor"
	log "github.com/sirupsen/logrus"
)

type FileStoreRawReaderConfig struct {
	Filename string
	Size     int
}

type FileStoreRawReader struct {
	conf *FileStoreRawReaderConfig

	CtxS     string
	Filename string
	file     *os.File

	b         []byte
	Pos       int
	Size      int
	Offset    int64
	AckOffset int64
}

func (c *FileStoreRawReaderConfig) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	var q FileStoreRawReader
	q.conf = c
	q.Size = c.Size
	if c.Size == 0 {
		q.Size = 10000
	}
	q.b = make([]byte, q.Size)
	return processor.GenProcessorHelperReader(context, &q, p, ctl, inc, out)
}

func (c *FileStoreRawReaderConfig) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *FileStoreRawReader) Ctx() string {
	return q.CtxS
}

func (q *FileStoreRawReader) Init(p *processor.Processor) error {
	q.Filename = q.conf.Filename //+ "." + p.Flow.Name
	log.Println(q.CtxS, "Opening file", q.Filename, "...")
	f, err := os.OpenFile(q.Filename, os.O_RDONLY, 0o644)
	if err != nil {
		log.Errorln(q.CtxS, "Error opening file for reading", q.Filename, err)
		return err
	}
	q.file = f
	return nil
}

func (q *FileStoreRawReader) Read() ([]processor.AckableEvent, error) {
	n, err := q.file.Read(q.b[q.Pos:q.Size])
	if err != nil {
		return nil, err
	}
	q.Pos += n

	// log.Debugln(q.ctx, "Buffer", "size", q.pos, "content", string(q.b[0:q.pos]))
	s := string(q.b[0:q.Pos])
	arr := strings.Split(s, "\n")
	msgCount := len(arr)
	// log.Debugln(q.ctx, "Buffer", "msgCount", msgCount, "msgs", arr)
	if q.b[q.Pos-1] != '\n' {
		lastSize := len(arr[msgCount-1])
		// log.Debugln(q.ctx, "Buffer", "lastsize", lastSize, "lastbuf", string(q.b[q.pos-lastSize:q.pos]))
		copy(q.b[0:], q.b[q.Pos-lastSize:q.Pos])
		// log.Debugln(q.ctx, "Buffer", "lastsize", lastSize, "lastbuf", string(q.b[0:lastSize]))
		q.Pos = lastSize
		msgCount -= 1
	} else {
		msgCount -= 1
		q.Pos = 0
	}

	events := make([]processor.AckableEvent, msgCount)

	for i := 0; i < msgCount; i++ {
		msg := arr[i]
		q.Offset += int64(len(msg)) + 1
		// log.Debugln(q.CtxS, "Event", q.Offset, msg)
		events[i] = processor.AckableEvent{q, q.Offset, msg, nil}
	}

	// log.Debugln(q.ctx, "Buffer", "size", q.pos, "content", string(q.b[0:q.pos]))
	// log.Debugf("%s %s %+v", q.ctx, "Events", events)
	return events, nil
}

func (q *FileStoreRawReader) AckMsg(msgid processor.EventAck) {
	// log.Debugln(q.CtxS, "Ackmsg", msgid)
	offset, ok := msgid.(int64)
	if !ok || offset <= q.AckOffset {
		log.Fatalln(q.CtxS, "AckMsg", q.Offset, msgid)
	}
	q.AckOffset = offset
}

func (q *FileStoreRawReader) Close() error {
	err := q.file.Close()
	if err != nil {
		log.Println(q.CtxS, "Close", "filename", q.Filename, "err", err)
		return err
	} else {
		log.Println(q.CtxS, "Close", q.Filename)
	}
	return nil
}
