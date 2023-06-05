package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	_ "github.com/jackc/pgx/v4/stdlib"
)

const (
	QLTTABLE         = "QLTBuffer"
	QLTTABLECONSUMER = "QLTBufferConsumer"
)

func pgDBInit(ctx string, conn *sql.DB) error {
	count, err := pgDBCount(conn)
	if err != nil {
		log.Warnc(ctx, "[DB-PG] error fectching previous rows", "err", err)
	} else {
		log.Debugc(ctx, "[DB-PG]  rows", count)
	}

	rows, err := pgDBRead(conn, 10, 0)
	if err != nil {
		log.Warnc(ctx, "[DB-PG] error fectching previous rows", "err", err)
	} else {
		log.Debugc(ctx, "[DB-PG]  rows", len(rows), rows)
	}

	_, err = conn.Exec("DROP TABLE IF EXISTS " + QLTTABLE)
	if err != nil {
		log.Errorc(ctx, "[DB-PG] error dropping table", "err", err)
		return err
	}

	_, err = conn.Exec("DROP TABLE IF EXISTS " + QLTTABLECONSUMER)
	if err != nil {
		log.Errorc(ctx, "[DB-PG] error dropping table", "err", err)
		return err
	}

	_, err = conn.Exec("CREATE TABLE IF NOT EXISTS " + QLTTABLE + " ( id BIGSERIAL PRIMARY KEY, name TEXT )")
	if err != nil {
		log.Errorc(ctx, "[DB-PG] error initializing table: ", "err", err)
		return err
	}

	_, err = conn.Exec("CREATE TABLE IF NOT EXISTS " + QLTTABLECONSUMER + " ( name TEXT PRIMARY KEY, position BIGINT )")
	if err != nil {
		log.Errorc(ctx, "[DB-PG] error initializing table: ", "err", err)
		return err
	}
	return nil
}

type PGWriter struct {
	ctx  string
	conn *sql.DB
	conf *PGWriterConf
}

type PGWriterConf struct {
	Url        string
	Initialize bool
}

func (conf *PGWriterConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc chan processor.AckableEvent, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	var q PGWriter

	q.conf = conf
	if conf.Url == "" {
		return nil, errors.New("Url field cannot be empty")
	}

	return processor.GenProcessorHelperWriter(context, processor.ConnectorRuntimeWriter(&q), p, ctl, inc, out)
}

func (c *PGWriterConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *PGWriter) Close() error {
	log.Infoc(q.ctx, "Closing...")
	err := q.conn.Close()
	if err != nil {
		log.Errorc(q.ctx, "close", "err", err)
	} else {
		log.Debugc(q.ctx, "close OK")
	}
	return err
}

func (q *PGWriter) Ctx() string {
	return q.ctx
}

func (q *PGWriter) IsAckAsync() bool {
	return false
}

func (q *PGWriter) Init(p *processor.Processor) error {
	log.Infoc(q.ctx, "Opening database", "url", q.conf.Url)
	conn, err := sql.Open("pgx", q.conf.Url)
	if err != nil {
		log.Fatalc(q.ctx, "Error opening file for appending", "err", err)
	}

	q.conn = conn

	if q.conf.Initialize {
		err := pgDBInit(q.ctx, conn)
		if err != nil {
			conn.Close()
			return err
		}
	}
	return nil
}

func (q *PGWriter) Write(msgs []processor.AckableEvent) error {
	valueStrings := make([]string, 0, len(msgs))
	valueArgs := make([]interface{}, 0, len(msgs))
	for i, msg := range msgs {
		valueStrings = append(valueStrings, "($"+fmt.Sprint(i+1)+")")
		valueArgs = append(valueArgs, msg.Msg)
	}
	params := strings.Join(valueStrings, ",")
	stmt := fmt.Sprintf("INSERT INTO "+QLTTABLE+" (name) VALUES %s", params)

	// log.Debugln("[DB-PG]  rows", valueStrings, valueArgs, params)

	// log.Debugln(q.ctx, "rows", len(msgs))
	_, err := q.conn.Exec(stmt, valueArgs...)
	if err != nil {
		log.Errorc(q.ctx, "INSERT failed ", "n", len(msgs), "err", err)
		return err
	}

	return nil
}

func (q *PGWriter) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent) {
	log.Fatalc(q.ctx, "Not supported")
}

func pgDBCount(conn *sql.DB) (int, error) {
	var count int
	err := conn.QueryRow("SELECT COUNT(*) FROM " + QLTTABLE + ";").Scan(&count)
	return count, err
}
