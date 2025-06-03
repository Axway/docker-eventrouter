package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"axway.com/qlt-router/src/config"
	"axway.com/qlt-router/src/filters/qlt2json"
	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"github.com/esimov/gogu/cache"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	_ "github.com/jackc/pgx/v4/stdlib"
	_ "modernc.org/sqlite"
)

var (
	usageTable    = config.DeclareString("report.usage.generalTableName", "usagemetrics", "Database table name used for storing usage metrics")
	cycleIDTable  = config.DeclareString("report.usage.countingTableName", "cycleids", "Database table name used calculating metrics")
	lastSentTable = config.DeclareString("report.usage.lastSent", "lastsent", "Database table name used to keep date of last successful sent")
)

type UsageReportWriterConf struct {
	DatabaseUri    string
	DatabaseType   string
	DatabaseUser   string
	DatabaseSecret string

	DatabaseCacheRetentionPeriod int /* days: min 7? */
	DatabaseRetentionPeriod      int /* days: min 365 */

	EnvironmentId string

	PlatformAPIUrl             string `default:"https://platform.axway.com"`
	PlatformAuthenticationUrl  string `default:"https://login.axway.com"`
	ServiceAccountClientID     string
	ServiceAccountClientSecret string

	HttpProxyAddr     string
	HttpProxyUser     string
	HttpProxyPassword string

	ReportPath string

	ReportFrequency int /* minutes */
}

type UsageReportWriter struct {
	CtxS          string
	Conf          *UsageReportWriterConf
	UsageReporter *UsageReporter
	Program       *vm.Program
	dbConn        *sql.DB
	lastPurge     string
	cacheCycleID  *cache.Cache[string, *CycleIDObj]
	cacheUsage    *cache.Cache[string, *UsageObj]
	initialized   bool
	cacheLoaded   bool
}

func (conf *UsageReportWriterConf) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, inc chan processor.AckableEvent, out chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	var q UsageReportWriter

	q.Conf = conf
	q.OpenBase()
	q.UsageReporter = &UsageReporter{Conf: conf, Instance_id: p.Instance_id, Ctxs: p.Name + "-reporter"}

	err := q.UsageReporter.Report(true)
	if err != nil {
		log.Errorc(q.UsageReporter.Ctxs, "Error occurs while sending first report of usage. Please check your configuration.", "err", err)
		log.Warnc(q.UsageReporter.Ctxs, "Not Fatal, continuing. Storing in the database but skipping the submission of the usage report to Amplify platform.")
	} else {
		log.Infoc(q.UsageReporter.Ctxs, "First usage report recorded.")
	}

	return processor.GenProcessorHelperWriter(ctx, &q, p, ctl, inc, out)
}

func (c *UsageReportWriterConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func (q *UsageReportWriter) OpenBase() error {
	var err error

	q.dbConn, err = OpenBase(q.CtxS, q.Conf, q.initialized)
	if err == nil {
		q.initialized = true
	}

	return err
}

func OpenBase(ctxs string, conf *UsageReportWriterConf, initialized bool) (*sql.DB, error) {
	completePath := "./usage-database.dbx"
	sanitizedPath := completePath
	basetype := "sqlite"
	var err error

	if conf.DatabaseUri != "" {
		completePath = conf.DatabaseUri
		sanitizedPath = conf.DatabaseUri
	}
	if conf.DatabaseType != "" {
		if strings.ToLower(conf.DatabaseType) == "postgres" || strings.HasPrefix(conf.DatabaseUri, "postgres://") || strings.HasPrefix(conf.DatabaseUri, "postgresql://") {
			basetype = "pgx"

			if conf.DatabaseUser != "" {
				hostspec := conf.DatabaseUri
				hostspec = strings.TrimPrefix(hostspec, "postgres://")
				hostspec = strings.TrimPrefix(hostspec, "postgresql://")

				userspec := conf.DatabaseUser
				if conf.DatabaseSecret != "" {
					userspec += ":" + conf.DatabaseSecret
				}
				userspec += "@"

				completePath = "postgres://" + userspec + hostspec
			}
		}

	}
	log.Debugc(ctxs, "Using database", "type", basetype, "url", sanitizedPath)

	dbConn, err := sql.Open(basetype, completePath)
	if err != nil {
		log.Fatalc(ctxs, "Error opening database", "err", err, "database", sanitizedPath)
		return nil, err
	}
	if !initialized {
		_, err = dbConn.Exec("CREATE TABLE IF NOT EXISTS " + cycleIDTable + " ( cycleid TEXT, monitor TEXT DEFAULT 'CFT', tabversion INTEGER DEFAULT 0, eventdate TEXT, ismainframe  INTEGER DEFAULT 0, PRIMARY KEY (cycleid, monitor) )")
		if err != nil {
			log.Errorc(ctxs, "Error initializing table: ", "table", cycleIDTable, "err", err)
			return dbConn, err
		}
		_, err = dbConn.Exec("CREATE TABLE IF NOT EXISTS " + usageTable + " ( datetime TEXT, monitor TEXT DEFAULT 'CFT', tabversion INTEGER DEFAULT 0, metrics TEXT DEFAULT '{}', PRIMARY KEY (datetime, monitor) )")
		if err != nil {
			log.Errorc(ctxs, "Error initializing table: ", "table", usageTable, "err", err)
			return dbConn, err
		}
		_, err = dbConn.Exec("CREATE TABLE IF NOT EXISTS " + lastSentTable + " ( instanceId TEXT, mode TEXT, datetime TEXT, tabversion INTEGER DEFAULT 0, PRIMARY KEY (instanceId, mode) )")
		if err != nil {
			log.Errorc(ctxs, "Error initializing table: ", "table", lastSentTable, "err", err)
			return dbConn, err
		}
	}

	return dbConn, err
}

func (q *UsageReportWriter) Init(p *processor.Processor) error {
	q.CtxS = p.Name

	if q.Conf.ReportPath == "" && q.Conf.ServiceAccountClientID == "" {
		log.Warnc(q.CtxS, "usage reporting activated, but no output configured.")
	} else {
		if q.Conf.EnvironmentId == "" {
			msg := "Environment ID cannot be empty"
			log.Fatalc(q.CtxS, msg)
			return errors.New(msg)
		}
		go q.UsageReporter.Reporting(p.Context)
	}

	if q.Conf.DatabaseRetentionPeriod < 365 {
		log.Warnc(q.CtxS, "retention period smaller than minimum. Using 365 days instead.", "RetentionPeriod", q.Conf.DatabaseRetentionPeriod)
		q.Conf.DatabaseRetentionPeriod = 365
	}
	if q.Conf.DatabaseCacheRetentionPeriod < 7 {
		log.Warnc(q.CtxS, "retention period for temporaty data smaller than minimum. Using 7 days instead.", "RetentionPeriodTmp", q.Conf.DatabaseCacheRetentionPeriod)
		q.Conf.DatabaseCacheRetentionPeriod = 7
	}

	env := map[string]map[string]string{}
	options := []expr.Option{
		expr.Env(env),
		expr.AllowUndefinedVariables(), // Allow the use of undefined variables.
		expr.AsBool(),
	}
	var err error
	// Counting completed (msg.state in ['SENT','RECEIVED','COMPLETED','ROUTED']) XFBTransfer messages (msg.qltname == 'XFBTransfer')
	// that are related to File transfers (msg.commandtype == 'F') and are sent by CFT (msg.monitor == 'CFT')
	// not generic transfers (msg.requesttype == 'S')
	q.Program, err = expr.Compile("msg.monitor == 'CFT' && msg.qltname == 'XFBTransfer' && "+
		"msg.commandtype == 'F' && msg.requesttype == 'S' && msg.state in ['SENT','RECEIVED','COMPLETED','ROUTED']", options...)
	if err != nil {
		log.Fatalc(q.CtxS, "usage report filtering expression compiling failed...", "error", err)
		return err
	}

	/* Declaring and loading cache */
	q.cacheCycleID = cache.New[string, *CycleIDObj](cache.NoExpiration, 24*time.Hour) /* No expiration/ No cleanup(24h as disable not possible) */
	q.cacheUsage = cache.New[string, *UsageObj]((48+1)*time.Hour, 1*time.Hour)        /* 2 days expiration, 1 hour cleanup */
	if q.dbConn == nil {
		q.OpenBase()
	}
	q.Purge()
	q.loadDBCache()

	return nil
}

func (q *UsageReportWriter) Ctx() string {
	return q.CtxS
}

func (q *UsageReportWriter) IsAckAsync() bool {
	return false
}

func (q *UsageReportWriter) ProcessAcks(ctx context.Context, acks chan processor.AckableEvent, errs chan error) {
	log.Fatalc(q.CtxS, "Not supported")
}

func (q *UsageReportWriter) IsActive() bool {
	return q.dbConn != nil
}

func (q *UsageReportWriter) Close() error {
	err := q.dbConn.Close()
	if err != nil {
		log.Errorc(q.CtxS, "close", "err", err)
	} else {
		log.Infoc(q.CtxS, "close OK")
	}
	q.dbConn = nil
	return nil
}

func (q *UsageReportWriter) Write(events []processor.AckableEvent) (int, error) {
	var err error
	var transact *sql.Tx
	invalidateCacheOnError := false
	n := 0
	mainframe := []string{"SYST_MVS", "SYST_VMS", "SYST_OS400", "SYST_HPNS"}
	cycleIdsToAdd := make(map[string]bool)    /* Map of cycle IDs to add */
	cycleIdsToUpdate := make(map[string]bool) /* Map of cycle IDs to update */
	usageToAdd := make(map[string]bool)       /* Map of usage items to add */
	usageToUpdate := make(map[string]bool)    /* Map of usage items to update */

	if q.dbConn == nil {
		q.OpenBase()
	}
	if !q.cacheLoaded {
		q.cacheCycleID.Flush()
		q.cacheUsage.Flush()
		q.loadDBCache()
	}
	current_time := time.Now().UTC().Format(time.RFC3339)
	for _, e := range events {
		/* FILTERING */
		if e.Msg == nil {
			n++
			continue
		}
		event := e.Msg.(string)
		/* Converting to json if needed */
		var eventMap map[string]map[string]string
		if strings.HasPrefix(event, "<?xml") {
			eventMapInter, err := qlt2json.ConvertToMap(event)
			if err != nil {
				log.Errorc(q.CtxS, "Converting to map failed...", "error", err)
			}
			eventMap = make(map[string]map[string]string)
			eventMap["msg"] = eventMapInter
		} else {
			err = json.Unmarshal([]byte(`{"msg": `+event+`}`), &eventMap)
			if err != nil {
				log.Errorc(q.CtxS, "Converting to map failed...", "error", err)
			}
		}

		/* Filtering */
		keep, err := expr.Run(q.Program, eventMap)
		if err != nil {
			log.Errorc(q.CtxS, "Running expression failed...", "error", err)
			break
		}
		/* not in 'SENT','RECEIVED' */
		if keep == false {
			n++
			continue
		}
		q.ensureEventDateTime(eventMap, current_time)

		/* RELEVANT DATA, COUNTING */
		dateHour := eventMap["msg"]["eventdatetime"][0:14] + "00:00Z"
		isMainsframe := 0
		if slices.Contains(mainframe, eventMap["msg"]["productos"]) {
			isMainsframe = 1
		}

		log.Tracec(q.CtxS, "Relevant data", "dateHour", dateHour)
		log.Tracec(q.CtxS, "Event", "cycleid", eventMap["msg"]["cycleid"], "monitor", eventMap["msg"]["monitor"],
			"eventdatetime", eventMap["msg"]["eventdatetime"], "productos", eventMap["msg"]["productos"])

		var usageItem *UsageObj
		u, err := q.cacheUsage.Get(eventMap["msg"]["monitor"] + dateHour)
		if err != nil {
			jsonmetrics := "{}"
			u := NewUsageObj(dateHour, eventMap["msg"]["monitor"], jsonmetrics)
			usageItem = &u
			// if date of event is older than 2 days, select in data base and add to cache
			// else create new entry into cache
			expiration := 2 * 24 * time.Hour
			if time.Now().UTC().Add(-expiration).Format("2006-01-02T15:04:05Z") > dateHour {
				err = q.dbConn.QueryRow("SELECT metrics FROM " + usageTable + " WHERE datetime='" + dateHour + "' AND monitor='" + eventMap["msg"]["monitor"] + "';").Scan(&jsonmetrics)
				if err != nil {
					usageToAdd[usageItem.Key()] = false
				} else {
					usageItem.Metrics = jsonmetrics
				}
				expiration = 1 * time.Hour /* old event: keeping for 1h only in cache */
			} else {
				usageToAdd[usageItem.Key()] = false
			}
			if err = q.cacheUsage.Set(usageItem.Key(), usageItem, expiration); err != nil {
				log.Errorc(q.CtxS, "Failed to add to cacheUsage", "key", usageItem.Key(), "err", err)
				goto error
			}
			invalidateCacheOnError = true
		} else {
			usageItem = u.Val()
		}

		var cycleIDItem *CycleIDObj
		addMainframe := 0
		addCount := 0
		c, err := q.cacheCycleID.Get(eventMap["msg"]["monitor"] + eventMap["msg"]["cycleid"])
		if err != nil {
			// add to cache + insert in transact
			c := NewCycleIDObj(eventMap["msg"]["cycleid"], eventMap["msg"]["monitor"], eventMap["msg"]["eventdatetime"], isMainsframe)
			cycleIDItem = &c
			if err = q.cacheCycleID.Set(cycleIDItem.Key(), cycleIDItem, cache.NoExpiration); err != nil {
				log.Errorc(q.CtxS, "Failed to add to cacheCycleID", "key", cycleIDItem.Key(), "err", err)
				goto error
			}
			invalidateCacheOnError = true
			cycleIdsToAdd[cycleIDItem.Key()] = false
			addCount = 1
			addMainframe = isMainsframe
		} else {
			cycleIDItem = c.Val()
			// if new is mainframe but old wasn't: update cache + update in transact
			if (isMainsframe == 1) && !cycleIDItem.IsMainframe {
				cycleIDItem.IsMainframe = true
				invalidateCacheOnError = true
				/* Add item to update map if not already present in the new map */
				if _, ok := cycleIdsToAdd[cycleIDItem.Key()]; !ok {
					cycleIdsToUpdate[cycleIDItem.Key()] = false
				}
				addMainframe = 1
			}
		}

		if addCount != 0 || addMainframe != 0 {
			if cycleIDItem.Monitor == "CFT" {
				var m CftMetrics
				m.FromJson(usageItem.Metrics)
				m.TransferCount += addCount
				m.Mainframe += addMainframe
				usageItem.Metrics = m.ToJson()
			} else {
				log.Errorc(q.CtxS, "Not yet supported", "monitor", cycleIDItem.Monitor == "CFT")
			}
			invalidateCacheOnError = true
			/* Add item to update map if not present in new */
			if _, ok := usageToAdd[usageItem.Key()]; !ok {
				usageToUpdate[usageItem.Key()] = false
			}
		}
		n++
	}

	/* Construct transact with lists. */
	if transact, err = q.dbConn.Begin(); err == nil {
		if len(cycleIdsToAdd) > 0 {
			first := true
			cmd := SQLInsertOpenCycleID()
			for k := range cycleIdsToAdd {
				if !first {
					cmd += SQLInsertSeparator()
				}
				e, _ := q.cacheCycleID.Get(k)
				item := e.Val()
				cmd += item.SQLInsertValues()
				first = false
			}
			cmd += SQLInsertClose()
			if _, err = transact.Exec(cmd); err != nil {
				log.Errorc(q.CtxS, "SQL error", "err", err)
				goto error
			}
		}
		for k := range cycleIdsToUpdate {
			e, _ := q.cacheCycleID.Get(k)
			item := e.Val()
			if _, err = transact.Exec(item.SQLUpdateCmd()); err != nil {
				log.Errorc(q.CtxS, "SQL error", "err", err)
				goto error
			}
		}
		if len(usageToAdd) > 0 {
			first := true
			cmd := SQLInsertOpenUsage()
			for k := range usageToAdd {
				if !first {
					cmd += SQLInsertSeparator()
				}
				e, _ := q.cacheUsage.Get(k)
				item := e.Val()
				cmd += item.SQLInsertValues()
				first = false
			}
			cmd += SQLInsertClose()
			if _, err = transact.Exec(cmd); err != nil {
				log.Errorc(q.CtxS, "SQL error", "err", err)
				goto error
			}
		}
		for k := range usageToUpdate {
			e, _ := q.cacheUsage.Get(k)
			item := e.Val()
			if _, err = transact.Exec(item.SQLUpdateCmd()); err != nil {
				log.Errorc(q.CtxS, "SQL error", "err", err)
				goto error
			}
		}

		if err = transact.Commit(); err != nil {
			log.Errorc(q.CtxS, "SQL error", "err", err)
			goto error
		}
	} else {
		log.Debugc(q.CtxS, "Failed to create transaction ")
	}
	if q.Conf.DatabaseRetentionPeriod != 0 {
		q.Purge()
	}

error:
	if err != nil {
		n = 0
		/* In case of error my cache is no longer correct */
		if invalidateCacheOnError {
			q.cacheLoaded = false
			q.cacheCycleID.Flush()
			q.cacheUsage.Flush()
		}
		if transact != nil {
			transact.Rollback()
		}
		q.dbConn.Close()
		q.dbConn = nil
	}

	return n, err
}

func (q *UsageReportWriter) ensureEventDateTime(eventMap map[string]map[string]string, currentTime string) {
	/* In case eventMap["msg"]["eventdatetime"] is empty, we should generate one using
	eventMap["msg"]["eventdate"] eventMap["msg"]["eventtime"] eventMap["msg"]["gmtdiff"]
	If this also fail, we should use the current time as eventMap["msg"]["eventdatetime"] */
	if eventMap["msg"]["eventdatetime"] == "" {
		if eventMap["msg"]["eventdate"] != "" && eventMap["msg"]["eventtime"] != "" {
			if eventMap["msg"]["gmtdiff"] == "" {
				eventMap["msg"]["eventdatetime"] = eventMap["msg"]["eventdate"] + "T" + eventMap["msg"]["eventtime"] + "Z"
			} else {
				gmtstr := ""
				gmt, _ := strconv.Atoi(eventMap["msg"]["gmtdiff"])
				if gmt < 0 {
					gmtstr = "-"
					gmt = -gmt
				}
				gmtstr += strconv.Itoa(gmt/60) + ":" + strconv.Itoa(gmt%60)

				eventdatetime := eventMap["msg"]["eventdate"] + "T" + eventMap["msg"]["eventtime"] + "Z" + gmtstr
				datetime, _ := time.Parse(time.RFC3339, eventdatetime)
				eventMap["msg"]["eventdatetime"] = datetime.UTC().Format(time.RFC3339)
			}
		} else {
			eventMap["msg"]["eventdatetime"] = currentTime
		}
	}
}

func (q *UsageReportWriter) loadDBCache() error {

	if q.dbConn == nil {
		q.OpenBase()
	}

	rows, err := q.dbConn.Query("SELECT cycleid, monitor, eventdate, ismainframe, tabversion FROM " + cycleIDTable + ";")
	if err != nil {
		log.Errorc(q.CtxS, "SQL error", "err", err)
		return err
	}
	for rows.Next() {
		var cycleid string
		var monitor string
		var eventdate string
		var ismainframe int
		var tabversion int
		err = rows.Scan(&cycleid, &monitor, &eventdate, &ismainframe, &tabversion)
		if err != nil {
			log.Errorc(q.CtxS, "SQL error (load cycleid)", "err", err)
			return err
		}
		cycleIDObj := NewCycleIDObj(cycleid, monitor, eventdate, ismainframe)
		q.cacheCycleID.Set(cycleIDObj.Key(), &cycleIDObj, cache.NoExpiration)
	}
	rows.Close()

	cacheTimeLimit := time.Now().UTC().Add(-2 * 24 * time.Hour)
	rows, err = q.dbConn.Query("SELECT datetime, monitor, metrics, tabversion FROM " + usageTable + " WHERE datetime > '" + cacheTimeLimit.Format("2006-01-02T15:04:05Z") + "';")
	if err != nil {
		log.Errorc(q.CtxS, "SQL error", "err", err)
		return err
	}
	for rows.Next() {
		var datetimeS string
		var monitor string
		var metrics string
		var tabversion int
		err = rows.Scan(&datetimeS, &monitor, &metrics, &tabversion)
		if err != nil {
			log.Errorc(q.CtxS, "SQL error (load usage)", "err", err)
			return err
		}
		datetime, _ := time.Parse(time.RFC3339, datetimeS)
		usageObj := NewUsageObj(datetimeS, monitor, metrics)
		q.cacheUsage.Set(usageObj.Key(), &usageObj, datetime.Sub(cacheTimeLimit))
	}
	rows.Close()
	q.cacheLoaded = true

	return nil
}

func (q *UsageReportWriter) Purge() error {
	if q.Conf.DatabaseRetentionPeriod == 0 {
		log.Debugc(q.CtxS, "No retention limit, exiting purge")
		return nil
	}

	today := strings.Split(time.Now().UTC().Format(time.RFC3339), "T")[0]
	if today == q.lastPurge {
		log.Tracec(q.CtxS, "Purge already done")
		return nil
	}

	timeLimit := strings.Split(time.Now().UTC().Add(-1*24*time.Duration(q.Conf.DatabaseRetentionPeriod)*time.Hour).Format(time.RFC3339), "T")[0]
	timeLimitTmp := strings.Split(time.Now().UTC().Add(-1*24*time.Duration(q.Conf.DatabaseCacheRetentionPeriod)*time.Hour).Format(time.RFC3339), "T")[0]

	_, err := q.dbConn.Exec("DELETE FROM " + cycleIDTable + " WHERE eventdate < '" + timeLimitTmp + "';")
	if err != nil {
		log.Errorc(q.CtxS, "Purge: SQL error (delete from "+cycleIDTable+")", "err", err)
	} else {
		_, err = q.dbConn.Exec("DELETE FROM  " + usageTable + " WHERE datetime < '" + timeLimit + "';")
		if err != nil {
			log.Errorc(q.CtxS, "Purge: SQL error (delete from "+usageTable+")", "err", err)
		}
	}
	log.Infoc(q.CtxS, "Purge done")
	q.lastPurge = today
	return err
}
