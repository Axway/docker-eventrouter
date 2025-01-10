package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	_ "modernc.org/sqlite"
)

const (
	QLTTABLE = "QLTBuffer"
)

func sqliteDBInitFromPath(ctx, path string) error {
	log.Infoc(ctx, "Opening database", "path", path)
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		log.Errorc(ctx, "Error opening database", "err", err, "path", path)
		return err
	}
	err = sqliteDBInit(ctx, conn, QLTTABLE)
	return err
}

func sqliteDBInit(ctx string, conn *sql.DB, tab string) error {
	count, err := sqliteDBCount(conn, tab)
	if err != nil {
		log.Warnc(ctx, "[DB-Sqlite] error fetching previous rows", "err", err)
	} else {
		log.Debugc(ctx, "[DB-Sqlite] rows", "count", count)
	}

	rows, err := sqliteDBRead(conn, 10, 0, tab)
	if err != nil {
		log.Warnc(ctx, "[DB-Sqlite] error fetching previous rows", "err", err)
	} else {
		log.Debugc(ctx, "[DB-Sqlite] rows", "count", len(rows), "rows", rows)
	}

	log.Infoc(ctx, "[DB-Sqlite] dropping table", "table", tab)
	_, err = conn.Exec("DROP TABLE IF EXISTS " + tab)
	if err != nil {
		log.Errorc(ctx, "[DB-Sqlite] error dropping table", "err", err)
		return err
	}

	log.Infoc(ctx, "[DB-Sqlite] dropping table", "table", tab+"Consumer")
	_, err = conn.Exec("DROP TABLE IF EXISTS " + tab + "Consumer")
	if err != nil {
		log.Errorc(ctx, "[DB-Sqlite] error dropping table", "err", err)
		return err
	}

	log.Infoc(ctx, "[DB-Sqlite] create table", "table", tab)
	_, err = conn.Exec("CREATE TABLE IF NOT EXISTS " + tab + " ( id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT )")
	if err != nil {
		log.Errorc(ctx, "[DB-Sqlite] error initializing table: ", "err", err)
		return err
	}

	log.Infoc(ctx, "[DB-Sqlite] create table", "table", tab+"Consumer")
	_, err = conn.Exec("CREATE TABLE IF NOT EXISTS " + tab + "Consumer" + " ( name TEXT PRIMARY KEY, position INTEGER )")
	if err != nil {
		log.Errorc(ctx, "[DB-Sqlite] error initializing table: ", "err", err)
		return err
	}
	log.Infoc(ctx, "[DB-Sqlite] initalization done")
	return nil
}

type SqliteWriter struct {
	ctx  string
	conn *sql.DB
	conf *SqliteWriterConf
}

type SqliteWriterConf struct {
	Path       string
	Initialize bool
	Table      string
}

func (conf *SqliteWriterConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc chan processor.AckableEvent, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	var q SqliteWriter

	q.conf = conf
	if conf.Path == "" {
		return nil, errors.New("Path field cannot be empty")
	}

	return processor.GenProcessorHelperWriter(context, processor.ConnectorRuntimeWriter(&q), p, ctl, inc, out)
}

func (c *SqliteWriterConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *SqliteWriter) Close() error {
	log.Infoc(q.ctx, "Closing...")
	err := q.conn.Close()
	if err != nil {
		log.Errorc(q.ctx, "close", "err", err)
	} else {
		log.Debugc(q.ctx, "close OK")
	}
	return err
}

func (q *SqliteWriter) Ctx() string {
	return q.ctx
}

func (q *SqliteWriter) IsAckAsync() bool {
	return false
}

func (q *SqliteWriter) IsActive() bool {
	return true
}

func (q *SqliteWriter) Init(p *processor.Processor) error {
	log.Infoc(q.ctx, "Opening database", "path", q.conf.Path)
	conn, err := sql.Open("sqlite", q.conf.Path)
	if err != nil {
		log.Fatalc(q.ctx, "Error opening database", "err", err, "path", q.conf.Path)
	}

	q.conn = conn

	if q.conf.Initialize {
		err := sqliteDBInit(q.ctx, conn, q.conf.Table)
		if err != nil {
			conn.Close()
			return err
		}
	}
	return nil
}

func (q *SqliteWriter) Write(msgs []processor.AckableEvent) (int, error) {
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

	// log.Debugln("[DB-Sqlite]  rows", valueStrings, valueArgs, params)

	// log.Debugln(q.ctx, "rows", len(msgs))
	_, err := q.conn.Exec(stmt, valueArgs...)
	if err != nil {
		log.Errorc(q.ctx, "INSERT failed ", "n", i, "err", err)
		return 0, err
	}

	return len(msgs), nil
}

func (q *SqliteWriter) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent, errs chan error) {
	log.Fatalc(q.ctx, "Not supported")
}

func sqliteDBCount(conn *sql.DB, tab string) (int, error) {
	var count int
	err := conn.QueryRow("SELECT COUNT(*) FROM " + tab + ";").Scan(&count)
	return count, err
}
