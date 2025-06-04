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
	err = pgDBInit(ctx, conn, QLTTABLE, true)
	return err
}

func pgDBInit(ctx string, conn *sql.DB, tab string, reset bool) error {
	var err error
	var count int

	if reset {
		count, err = pgDBCount(conn, tab)
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
	}

	log.Infoc(ctx, "[DB-PG] create table", "table", tab)
	_, err = conn.Exec("CREATE TABLE IF NOT EXISTS " + tab +
		" ( id BIGSERIAL NOT NULL" +
		", inserted_at timestamptz NOT NULL DEFAULT now() " +
		", name TEXT NOT NULL " +
		", PRIMARY KEY ( id ))")
	if err != nil {
		log.Errorc(ctx, "[DB-PG] error initializing table: ", "err", err)
		return err
	}

	log.Infoc(ctx, "[DB-PG] create table", "table", tab+"Consumer")
	_, err = conn.Exec("CREATE TABLE IF NOT EXISTS " + tab + "Consumer" +
		" ( name TEXT PRIMARY KEY, position BIGINT )")
	if err != nil {
		log.Errorc(ctx, "[DB-PG] error initializing table: ", "err", err)
		return err
	}
	log.Infoc(ctx, "[DB-PG] initalization done")
	return nil
}

type PGWriter struct {
	ctx         string
	conn        *sql.DB
	conf        *PGWriterConf
	initialized bool
	processor   *processor.Processor
}

type PGWriterConf struct {
	Url            string
	User, Password string
	Table          string
	Initialize     bool
}

func (conf *PGWriterConf) Start(context context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc chan processor.AckableEvent, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	var q PGWriter

	q.ctx = p.Name
	q.conf = conf
	q.processor = p
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
	if q.conn == nil {
		return nil
	}
	err := q.conn.Close()
	if err != nil {
		log.Errorc(q.ctx, "close", "err", err)
	} else {
		log.Debugc(q.ctx, "close OK")
	}
	q.conn = nil
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

func PrepareUris(originalUri string, user string, password string) (string, string) {
	sanityzedUri := ""
	hostspec := originalUri
	hostspec = strings.TrimPrefix(hostspec, "postgres://")
	hostspec = strings.TrimPrefix(hostspec, "postgresql://")
	userspec := ""
	sanityzedUser := ""
	if strings.Contains(hostspec, "@") {
		userspec = strings.Split(hostspec, "@")[0]
		hostspec = strings.Split(hostspec, "@")[1]
		if strings.Contains(userspec, ":") {
			sanityzedUser = strings.Split(userspec, ":")[0] + ":" + strings.Repeat("*", 6)
		} else {
			sanityzedUser = userspec
		}
		sanityzedUser += "@"
		userspec += "@"
	} else if user != "" {
		userspec = user
		sanityzedUser = user
		if password != "" {
			userspec += ":" + password
			sanityzedUser += ":" + strings.Repeat("*", 6)
		}
		sanityzedUser += "@"
		userspec += "@"
	}
	completeUri := "postgres://" + userspec + hostspec
	sanityzedUri = "postgres://" + sanityzedUser + hostspec

	return completeUri, sanityzedUri
}

func (q *PGWriter) Init(p *processor.Processor) error {
	completeUri, sanityzedUri := PrepareUris(q.conf.Url, q.conf.User, q.conf.Password)

	log.Infoc(q.ctx, "Opening database", "url", "'"+sanityzedUri+"'")
	conn, err := sql.Open("pgx", completeUri)

	if err != nil {
		log.Errorc(q.ctx, "Error opening database", "err", err, "url", "'"+sanityzedUri+"'")
	}

	q.conn = conn

	if !q.initialized {
		err = pgDBInit(q.ctx, conn, q.conf.Table, q.conf.Initialize)
		if err != nil {
			q.Close()
		} else {
			q.initialized = true
		}
	}
	return nil
}

func (q *PGWriter) Write(msgs []processor.AckableEvent) (int, error) {
	if q.conn == nil {
		err := q.Init(q.processor)
		if err != nil {
			return 0, err
		}
	}
	if len(msgs) == 0 { /* Nothing to write */
		return 0, nil
	}

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
