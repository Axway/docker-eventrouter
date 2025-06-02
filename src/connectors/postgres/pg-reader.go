package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"axway.com/qlt-router/src/tools"
	"github.com/jackc/pgerrcode"
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
	Url            string
	User, Password string
	Table          string
	ReaderName     string
}

func (conf *PGReaderConf) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc chan processor.AckableEvent, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	var q PGReader

	q.ctx = p.Name
	q.conf = conf
	q.processor = p
	if conf.Url == "" {
		return nil, errors.New("Url field cannot be empty")
	}
	if conf.ReaderName == "" {
		return nil, errors.New("ReaderName field cannot be empty")
	}
	if conf.Table == "" {
		q.conf.Table = QLTTABLE
	}

	return processor.GenProcessorHelperReader(ctx, &q, p, ctl, inc, out)
}

func (c *PGReaderConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *PGReader) Init(p *processor.Processor) error {
	time.Sleep(1 * time.Second)

	completeUri, sanityzedUri := PrepareUris(q.conf.Url, q.conf.User, q.conf.Password)

	log.Infoc(q.ctx, "Opening database", "url", "'"+sanityzedUri+"'")
	conn, err := sql.Open("pgx", completeUri)
	if err != nil {
		log.Errorc(q.ctx, "Error opening database", "err", err, "url", "'"+sanityzedUri+"'")
		return err
	}

	q.conn = conn
	offset, err := q.initializeReaderEntry()
	if err != nil {
		// Handle connection refused error specifically
		if errors.Is(err, sql.ErrConnDone) || strings.Contains(err.Error(), "connection refused") {
			log.Infoc(q.ctx, "Connection refused when initializing reader entry", "err", err)
			return nil
		}
	} else {
		q.offset = offset
	}

	q.lastmsgid, err = pgDBGetLast(conn, q.conf.Table)
	if err == nil && q.lastmsgid < q.offset {
		q.offset = q.lastmsgid
	}
	return err
}

func (q *PGReader) Close() error {
	if q.conn != nil {
		log.Infoc(q.ctx, "Closing...")
		err := q.conn.Close()
		if err != nil {
			log.Errorc(q.ctx, "close", "err", err)
		} else {
			log.Debugc(q.ctx, "close OK")
		}
		q.conn = nil
		return err
	}
	log.Debugc(q.ctx, "Already closed...")
	return nil
}

func (q *PGReader) Read() ([]processor.AckableEvent, error) {
	if q.conn == nil {
		err := q.Init(q.processor)
		if err != nil {
			return nil, err
		}
	}

	rows, err := pgDBRead(q.conn, 1000, int(q.offset), q.conf.Table)
	if err != nil {
		if err == sql.ErrNoRows {
			err = nil
		} else if errors.Is(err, sql.ErrConnDone) || strings.Contains(err.Error(), "connection refused") {
			log.Errorc(q.ctx, "error reading db", "err", err)
			err = tools.ErrClosedConnection
			q.conn = nil
		} else {
			log.Errorc(q.ctx, "error reading db", "err", err)
			/* Translating errors type 57 into tools.ErrClosedConnection.
			* We might need to translate more errors, not doing it for now. */
			if pgErr, ok := err.(interface{ SQLState() string }); ok &&
				pgerrcode.IsOperatorIntervention(pgErr.SQLState()) {
				err = tools.ErrClosedConnection
				q.conn = nil
			}
		}
		return nil, err
	}

	msgs := make([]processor.AckableEvent, len(rows))

	for i, row := range rows {
		log.Tracec(q.ctx, "Read", "row", row)
		msgs[i] = processor.AckableEvent{q, row.id, row.text, nil}
		q.offset = row.id // keep last
	}
	return msgs, nil
}

func (q *PGReader) Ctx() string {
	return q.ctx
}

func (m *PGReader) IsServer() bool {
	return false
}

func (q *PGReader) AckMsg(ack processor.EventAck) {
	offset := ack.(int64)
	q.commitAck(offset)
}

func (q *PGReader) commitAck(offset int64) error {
	_, err := q.conn.Exec("UPDATE "+q.conf.Table+"Consumer"+" SET position = $1 WHERE name = $2", offset, q.conf.ReaderName)
	if err != nil {
		log.Errorc(q.ctx, "Error commiting Ack", "err", err)
		return err
	}
	q.lastackedmsgid = offset
	return nil
}

func (q *PGReader) initializeReaderEntry() (int64, error) {
	_, err := q.conn.Exec("INSERT INTO "+q.conf.Table+"Consumer"+" (name, position) VALUES ($1, $2) ON CONFLICT DO NOTHING", q.conf.ReaderName, q.lastackedmsgid)
	if err != nil {
		log.Errorc(q.ctx, "Error creating consumer ", "readerName", q.conf.ReaderName, "err", err)
		return 0, err
	}

	var lastackedmsgid sql.NullInt64
	err = q.conn.QueryRow("SELECT position FROM  "+q.conf.Table+"Consumer"+" WHERE name = $1", q.conf.ReaderName).Scan(&lastackedmsgid)
	if err == nil {
		if !lastackedmsgid.Valid {
			q.lastackedmsgid = 0
		} else {
			q.lastackedmsgid = lastackedmsgid.Int64
		}
	}
	return q.lastackedmsgid, nil
}
