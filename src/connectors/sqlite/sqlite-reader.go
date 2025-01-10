package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
)

type SqliteReader struct {
	conf           *SqliteReaderConf
	conn           *sql.DB
	ctx            string
	offset         int64
	lastmsgid      int64
	lastackedmsgid int64
	processor      *processor.Processor
}

type SqliteReaderConf struct {
	Path       string
	ReaderName string
	Table      string
}

func (conf *SqliteReaderConf) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc chan processor.AckableEvent, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	var q SqliteReader

	q.conf = conf
	if conf.Path == "" {
		return nil, errors.New("Path field cannot be empty")
	}
	if conf.ReaderName == "" {
		return nil, errors.New("ReaderName field cannot be empty")
	}
	if conf.Table == "" {
		q.conf.Table = QLTTABLE
	}

	return processor.GenProcessorHelperReader(ctx, &q, p, ctl, inc, out)
}

func (c *SqliteReaderConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *SqliteReader) Init(p *processor.Processor) error {
	time.Sleep(1 * time.Second)
	log.Infoc(q.ctx, "Opening database", "path", "'"+q.conf.Path+"'")
	conn, err := sql.Open("sqlite", q.conf.Path)
	if err != nil {
		log.Errorc(q.ctx, "Error opening file for appending", "err", err)
		return err
	}

	q.conn = conn

	offset, err := q.initializeReaderEntry()
	if err != nil {
	}

	q.offset = offset

	q.lastmsgid, _ = sqliteDBGetLast(conn, q.conf.Table)

	return nil
}

func (q *SqliteReader) Close() error {
	log.Infoc(q.ctx, "Closing...")
	err := q.conn.Close()
	if err != nil {
		log.Errorc(q.ctx, "close", "err", err)
	} else {
		log.Debugc(q.ctx, "close OK")
	}
	return err
}

func (q *SqliteReader) Read() ([]processor.AckableEvent, error) {
	rows, err := sqliteDBRead(q.conn, 1000, int(q.offset), q.conf.Table)
	if err != nil {
		log.Errorc(q.ctx, "error reading db", "err", err)
		// FIXME the reader should reconnect
		return nil, err
	}

	q.lastmsgid, _ = sqliteDBGetLast(q.conn, q.conf.Table)

	msgs := make([]processor.AckableEvent, len(rows))

	for i, row := range rows {
		log.Tracec(q.ctx, "Read", "row", row)
		msgs[i] = processor.AckableEvent{q, row.id, row.text, nil}
		q.offset = row.id // keep last
	}
	return msgs, nil
}

func (q *SqliteReader) Ctx() string {
	return q.ctx
}

func (q *SqliteReader) AckMsg(ack processor.EventAck) {
	offset := ack.(int64)
	q.commitAck(offset)
}

func (q *SqliteReader) commitAck(offset int64) error {
	_, err := q.conn.Exec("UPDATE "+q.conf.Table+"Consumer"+" SET position = $2 WHERE name = $1", q.conf.ReaderName, offset)
	if err != nil {
		log.Errorc(q.ctx, "Error commiting Ack", "err", err)
		return err
	}
	q.lastackedmsgid = offset
	return nil
}

func (q *SqliteReader) initializeReaderEntry() (int64, error) {
	// FIXME: retrieve last
	_, err := q.conn.Exec("INSERT INTO "+q.conf.Table+"Consumer"+" (name, position) VALUES ($1, $2) ON CONFLICT DO NOTHING", q.conf.ReaderName, 0)
	if err != nil {
		log.Errorc(q.ctx, "Error creating consumer ", "readerName", q.conf.ReaderName, "err", err)
		return 0, err
	}
	return 0, nil
}
