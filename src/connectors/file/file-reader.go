package file

import (
	"context"
	"os"
	"io"
	"strings"
	"strconv"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"axway.com/qlt-router/src/tools"
)

type FileStoreRawReaderConfig struct {
	FilenamePrefix string
	FilenameSuffix string
	Size           int
	ReaderFilename string
}

type FileStoreRawReader struct {
	conf *FileStoreRawReaderConfig

	CtxS     string
	Filename string
	AckFilename string
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
	/* Try to get filename + offset from conf file */
	log.Infoc(q.CtxS, "Opening file", "filename", q.conf.ReaderFilename)
	f2, err := os.OpenFile(q.conf.ReaderFilename, os.O_RDONLY, 0o644)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Errorc(q.CtxS, "Error opening file for reading last position", "filename", q.conf.ReaderFilename, "err", err)
		}
		q.Filename, err = tools.NextFile(q.CtxS, q.conf.FilenamePrefix, q.conf.FilenameSuffix, q.conf.FilenamePrefix)
		q.Offset = 0
	} else {
		/* Read info from file */
		b := make([]byte, 3072)
		n, err := f2.Read(b)
		f2.Close()
		s := string(b[0:n])
		arr := strings.Split(s, "\n")
		q.Offset, err = strconv.ParseInt(arr[0], 10, 64)
		if err != nil {
			q.Offset = 0
		}
		q.Filename = arr[1]
	}

	log.Infoc(q.CtxS, "Opening file", "filename", q.Filename)
	f, err := os.OpenFile(q.Filename, os.O_RDONLY, 0o644)
	if err != nil {
		log.Errorc(q.CtxS, "Error opening file for reading", "filename", q.Filename, "err", err)
		return err
	}
	q.file = f
	q.AckFilename = q.Filename

	/* Go to correct offset */
	//log.Infoc(q.CtxS, "Seeking position", "offset", strconv.FormatInt(q.Offset,10))
	q.file.Seek(q.Offset, io.SeekStart)

	return nil
}

func (q *FileStoreRawReader) Read() ([]processor.AckableEvent, error) {
	n, err := q.file.Read(q.b[q.Pos:q.Size])
	if err != nil && err != io.EOF {
		return nil, err
	}

	if n == 0 || err != nil {
		filename, _ := tools.NextFile(q.CtxS, q.conf.FilenamePrefix, q.conf.FilenameSuffix, q.Filename)
		if filename != q.Filename {
			q.file.Close()
			q.Filename = filename
			q.Offset = 0
			q.Pos = 0

			f, err := os.OpenFile(q.Filename, os.O_RDONLY, 0o644)
			if err != nil {
				log.Errorc(q.CtxS, "Error opening file for reading", "filename", q.Filename, "err", err)
				return nil, err
			}
			q.file = f
			n, err = q.file.Read(q.b[q.Pos:q.Size])
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
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
	changingFile := 0
	if offset < q.AckOffset {
		f, err := os.OpenFile(q.AckFilename, os.O_RDONLY, 0o644)
		/* Test err */
		f.Seek(q.AckOffset, io.SeekStart)
		b1 := make([]byte, 1)
		_, err = f.Read(b1)

		if err == io.EOF {
			changingFile = 1
		}
		f.Close()
	}

	if !ok || (offset <= q.AckOffset && changingFile == 0) {
		log.Fatalc(q.CtxS, "AckMsg", "offset", q.Offset, "msgid", msgid)
	}
	if changingFile == 1 {
		q.AckFilename, _ = tools.NextFile(q.CtxS, q.conf.FilenamePrefix, q.conf.FilenameSuffix, q.AckFilename)
		/* Test err */
	}
	q.AckOffset = offset

	/* Write in file */
	f2, err := os.OpenFile(q.conf.ReaderFilename, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		log.Errorc(q.CtxS, "Error opening file for writing last position", "filename", q.conf.ReaderFilename, "err", err)
	} else {
		defer f2.Close()
		b := []byte(strconv.FormatInt(q.AckOffset,10) + "\n" + q.AckFilename)
		_, err = f2.Write(b)
	}
}

func (q *FileStoreRawReader) Close() error {
	err := q.file.Close()
	if err != nil {
		log.Errorc(q.CtxS, "Close", "filename", q.Filename, "err", err)
		return err
	} else {
		log.Infoc(q.CtxS, "Close", "filename", q.Filename)
	}
	return nil
}
