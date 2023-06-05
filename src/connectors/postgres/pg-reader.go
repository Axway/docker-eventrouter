package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
)

type PGReader struct {
	conf           *PGReaderConf
	conn           *sql.DB
	ctx            string
	offset         int64
	lastmsgid      int64
	lastackedmsgid int64
	processor      *processor.Processor
}

type PGReaderConf struct {
	Url        string
	ReaderName string
}

func (conf *PGReaderConf) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc chan processor.AckableEvent, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	var q PGReader

	q.conf = conf
	if conf.Url == "" {
		return nil, errors.New("Url field cannot be empty")
	}
	if conf.ReaderName == "" {
		return nil, errors.New("ReaderName field cannot be empty")
	}

	return processor.GenProcessorHelperReader(ctx, &q, p, ctl, inc, out)
}

func (c *PGReaderConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *PGReader) Init(p *processor.Processor) error {
	time.Sleep(1 * time.Second)
	log.Infoc(q.ctx, "Opening database", "url", "'"+q.conf.Url+"'")
	conn, err := sql.Open("pgx", q.conf.Url)
	if err != nil {
		log.Errorc(q.ctx, "Error opening file for appending", "err", err)
		return err
	}

	q.conn = conn

	offset, err := q.initializeReaderEntry()
	if err != nil {
	}

	q.offset = offset

	q.lastmsgid, _ = pgDBGetLast(conn)

	return nil
}

func (q *PGReader) Close() error {
	log.Infoc(q.ctx, "Closing...")
	err := q.conn.Close()
	if err != nil {
		log.Errorc(q.ctx, "close", "err", err)
	} else {
		log.Debugc(q.ctx, "close OK")
	}
	return err
}

func (q *PGReader) Read() ([]processor.AckableEvent, error) {
	rows, err := pgDBRead(q.conn, 1000, int(q.offset))
	if err != nil {
		log.Errorc(q.ctx, "error reading db", "err", err)
		return nil, err
	}

	q.lastmsgid, _ = pgDBGetLast(q.conn)

	msgs := make([]processor.AckableEvent, len(rows))

	for i, row := range rows {
		msgs[i] = processor.AckableEvent{q, row.id, row.text, nil}
		q.offset = row.id // keep last
	}
	return msgs, nil
}

func (q *PGReader) Ctx() string {
	return q.ctx
}

func (q *PGReader) AckMsg(ack processor.EventAck) {
	offset := ack.(int64)
	q.commitAck(offset)
}

func (q *PGReader) commitAck(offset int64) error {
	_, err := q.conn.Exec("UPDATE "+QLTTABLECONSUMER+" SET position = $2 WHERE name = $1", q.conf.ReaderName, offset)
	if err != nil {
		log.Errorc(q.ctx, "Error commiting Ack", "err", err)
		return err
	}
	q.lastackedmsgid = offset
	return nil
}

func (q *PGReader) initializeReaderEntry() (int64, error) {
	// FIXME: retrieve last
	_, err := q.conn.Exec("INSERT INTO "+QLTTABLECONSUMER+" (name, position) VALUES ($1, $2) ON CONFLICT DO NOTHING", q.conf.ReaderName, 0)
	if err != nil {
		log.Errorc(q.ctx, "Error creating consumer ", "readerName", q.conf.ReaderName, "err", err)
		return 0, err
	}
	return 0, nil
}
