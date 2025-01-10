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
	QLTTABLE = "QLTBuffer"
)

func pgDBInitFromUrl(ctx, url string) error {
	log.Infoc(ctx, "Opening database", "url", url)
	conn, err := sql.Open("pgx", url)
	if err != nil {
		log.Errorc(ctx, "Error opening database", "err", err, "url", url)
		return err
	}
	err = pgDBInit(ctx, conn, QLTTABLE)
	return err
}

func pgDBInit(ctx string, conn *sql.DB, tab string) error {
	count, err := pgDBCount(conn, tab)
	if err != nil {
		log.Warnc(ctx, "[DB-PG] error fetching previous rows", "err", err)
	} else {
		log.Debugc(ctx, "[DB-PG] rows", "count", count)
	}

	rows, err := pgDBRead(conn, 10, 0, tab)
	if err != nil {
		log.Warnc(ctx, "[DB-PG] error fectching previous rows", "err", err)
	} else {
		log.Debugc(ctx, "[DB-PG] rows", "count", len(rows), "rows", rows)
	}

	log.Infoc(ctx, "[DB-PG] dropping table", "table", tab)
	_, err = conn.Exec("DROP TABLE IF EXISTS " + tab)
	if err != nil {
		log.Errorc(ctx, "[DB-PG] error dropping table", "err", err)
		return err
	}

	log.Infoc(ctx, "[DB-PG] dropping table", "table", tab+"Consumer")
	_, err = conn.Exec("DROP TABLE IF EXISTS " + tab + "Consumer")
	if err != nil {
		log.Errorc(ctx, "[DB-PG] error dropping table", "err", err)
		return err
	}

	log.Infoc(ctx, "[DB-PG] create table", "table", tab)
	_, err = conn.Exec("CREATE TABLE IF NOT EXISTS " + tab + " ( id BIGSERIAL PRIMARY KEY, name TEXT )")
	if err != nil {
		log.Errorc(ctx, "[DB-PG] error initializing table: ", "err", err)
		return err
	}

	log.Infoc(ctx, "[DB-PG] create table", "table", tab+"Consumer")
	_, err = conn.Exec("CREATE TABLE IF NOT EXISTS " + tab + "Consumer" + " ( name TEXT PRIMARY KEY, position BIGINT )")
	if err != nil {
		log.Errorc(ctx, "[DB-PG] error initializing table: ", "err", err)
		return err
	}
	log.Infoc(ctx, "[DB-PG] initalization done")
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
	Table      string
}

func (conf *PGWriterConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc chan processor.AckableEvent, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	var q PGWriter

	q.conf = conf
	if conf.Url == "" {
		return nil, errors.New("Url field cannot be empty")
	}
	if conf.Table == "" {
		q.conf.Table = QLTTABLE
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

func (q *PGWriter) IsActive() bool {
	return true
}

func (q *PGWriter) Init(p *processor.Processor) error {
	log.Infoc(q.ctx, "Opening database", "url", q.conf.Url)
	conn, err := sql.Open("pgx", q.conf.Url)
	if err != nil {
		log.Errorc(q.ctx, "Error opening database", "err", err, "url", q.conf.Url)
	}

	q.conn = conn

	if q.conf.Initialize {
		err := pgDBInit(q.ctx, conn, q.conf.Table)
		if err != nil {
			conn.Close()
			return err
		}
	}
	return nil
}

func (q *PGWriter) Write(msgs []processor.AckableEvent) (int, error) {
	i := 0
	valueStrings := make([]string, 0, len(msgs))
	valueArgs := make([]interface{}, 0, len(msgs))
	for _, msg := range msgs {
		if msg.Msg != nil {
			log.Tracec(q.ctx, "Write", "row", msg)
			valueStrings = append(valueStrings, "($"+fmt.Sprint(i+1)+")")
			valueArgs = append(valueArgs, msg.Msg)
			i++
		}
	}
	params := strings.Join(valueStrings, ",")
	stmt := fmt.Sprintf("INSERT INTO "+q.conf.Table+" (name) VALUES %s", params)

	// log.Debugln("[DB-PG]  rows", valueStrings, valueArgs, params)

	// log.Debugln(q.ctx, "rows", len(msgs))
	_, err := q.conn.Exec(stmt, valueArgs...)
	if err != nil {
		log.Errorc(q.ctx, "INSERT failed ", "n", i, "err", err)
		return 0, err
	}

	return len(msgs), nil
}

func (q *PGWriter) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent, errs chan error) {
	log.Fatalc(q.ctx, "Not supported")
}

func pgDBCount(conn *sql.DB, tab string) (int, error) {
	var count int
	err := conn.QueryRow("SELECT COUNT(*) FROM " + tab + ";").Scan(&count)
	return count, err
}
