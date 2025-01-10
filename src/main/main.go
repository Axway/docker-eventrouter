package main

import (
	//"context"
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"
	"time"

	"axway.com/qlt-router/src/config"
	awssqs "axway.com/qlt-router/src/connectors/aws-sqs"
	"axway.com/qlt-router/src/connectors/file"
	"axway.com/qlt-router/src/connectors/kafka"
	"axway.com/qlt-router/src/connectors/mem"
	"axway.com/qlt-router/src/connectors/postgres"
	"axway.com/qlt-router/src/connectors/qlt"
	"axway.com/qlt-router/src/connectors/sqlite"
	"axway.com/qlt-router/src/filters/expr"
	"axway.com/qlt-router/src/filters/qlt2json"
	log "axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Version = ""
	Build   = ""
	Date    = ""
	Ready   = false
)

type Distribution struct {
	Version string
	Build   string
	Date    string
}

type RouterState struct {
	Distribution Distribution
	Config       *processor.Config
	Channels     *processor.Channels
	Processors   []*processor.Processor
}

// content holds our static web server content.
//
//go:embed ui/*
var ui embed.FS

func printHelp(rc int) {
	fmt.Println("Axway Event Router")
	fmt.Println("")
	fmt.Println("Usage: event-router [options] [arguments]")
	fmt.Println("If no argument is specified, the Event Router starts.")
	fmt.Println("")
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("")
	fmt.Println("Arguments:")
	fmt.Println("\thelp")
	fmt.Println("\tversion")
	fmt.Println("\tlist-connectors")
	fmt.Println("\tlist-config")
	os.Exit(rc)
}

func main() {
	ctxS := "main"
	log.SetLevel(log.InfoLevel)

	/* Parameters ENV VAR or command line */
	var configDefault string
	configEnvVar := os.Getenv("ER_CONFIG_FILE")
	if configEnvVar != "" {
		configDefault = configEnvVar
	} else {
		configDefault = "./event-router.yml"
	}
	configFile := flag.String("config", configDefault, "path to config file")

	var portDefault string
	portEnvVar := os.Getenv("ER_PORT")
	if portEnvVar != "" {
		portDefault = portEnvVar
	} else {
		portDefault = "8080"
	}
	port := flag.String("port", portDefault, "http port")

	var hostDefault string
	hostEnvVar := os.Getenv("ER_HOST")
	if hostEnvVar != "" {
		hostDefault = hostEnvVar
	} else {
		hostDefault = "0.0.0.0"
	}
	host := flag.String("host", hostDefault, "http host")

	var logFileDefault string
	logFileEnvVar := os.Getenv("ER_LOG_FILE")
	if logFileEnvVar != "" {
		logFileDefault = logFileEnvVar
	} else {
		logFileDefault = ""
	}
	logFname := flag.String("log-file", logFileDefault, "log file name")

	var logMaxSizeDefault int
	logMaxSizeEnvVar := os.Getenv("ER_LOG_FILE_MAX_SIZE")
	if logMaxSizeEnvVar != "" {
		var err error
		logMaxSizeDefault, err = strconv.Atoi(logMaxSizeEnvVar)
		if err != nil {
			logMaxSizeDefault = 100
			log.Infoc(ctxS, "Invalid log file max size (MB)", "ER_LOG_FILE_MAX_SIZE", logMaxSizeEnvVar, "using", logMaxSizeDefault)
		}
	} else {
		logMaxSizeDefault = 100
	}
	logMaxSize := flag.Int("log-max-size", logMaxSizeDefault, "log file max size (MB)")

	var logMaxBackupDefault int
	logMaxBackupEnvVar := os.Getenv("ER_LOG_FILE_MAX_BACKUP")
	if logMaxBackupEnvVar != "" {
		var err error
		logMaxBackupDefault, err = strconv.Atoi(logMaxBackupEnvVar)
		if err != nil {
			logMaxBackupDefault = 3
			log.Infoc(ctxS, "Invalid log file backups", "ER_LOG_FILE_MAX_BACKUP", logMaxBackupEnvVar, "using", logMaxBackupDefault)
		}
	} else {
		logMaxBackupDefault = 3
	}
	logMaxBackup := flag.Int("log-max-backups", logMaxBackupDefault, "log file backups")

	var logMaxAgeDefault int
	logMaxAgeEnvVar := os.Getenv("ER_LOG_FILE_MAX_AGE")
	if logMaxAgeEnvVar != "" {
		var err error
		logMaxAgeDefault, err = strconv.Atoi(logMaxAgeEnvVar)
		if err != nil {
			logMaxAgeDefault = 31
			log.Infoc(ctxS, "Invalid log file max age (days)", "ER_LOG_FILE_MAX_AGE", logMaxAgeEnvVar, "using", logMaxAgeDefault)
		}
	} else {
		logMaxAgeDefault = 31
	}
	logMaxAge := flag.Int("log-max-age", logMaxAgeDefault, "log file max age (days)")

	var logLocaltimeDefault bool
	logLocaltimeEnvVar := os.Getenv("ER_LOG_USE_LOCALTIME")
	if logLocaltimeEnvVar != "" {
		var err error
		logLocaltimeDefault, err = strconv.ParseBool(logLocaltimeEnvVar)
		if err != nil {
			logLocaltimeDefault = false
			log.Infoc(ctxS, "Invalid log file backups", "ER_LOG_MAX_BACKUP", logLocaltimeEnvVar, "using", logLocaltimeDefault)
		}
	} else {
		logLocaltimeDefault = false
	}
	logLocatime := flag.Bool("log-localtime", logLocaltimeDefault, "log uses local time")

	var cpuProfileFileDefault string
	cpuProfileFileEnvVar := os.Getenv("ER_CPU_PROFILE_FILE")
	if cpuProfileFileEnvVar != "" {
		cpuProfileFileDefault = cpuProfileFileEnvVar
	} else {
		cpuProfileFileDefault = ""
	}
	cpuprofile := flag.String("cpuprofile", cpuProfileFileDefault, "write cpu profile to this file")

	var memProfileFileDefault string
	memProfileFileEnvVar := os.Getenv("ER_MEM_PROFILE_FILE")
	if memProfileFileEnvVar != "" {
		memProfileFileDefault = memProfileFileEnvVar
	} else {
		memProfileFileDefault = ""
	}
	memprofile := flag.String("memprofile", memProfileFileDefault, "write memory profile to this file")

	var logLevelDefault string
	logLevelEnvVar := os.Getenv("ER_LOG_LEVEL")
	if logLevelEnvVar != "" {
		switch strings.ToLower(logLevelEnvVar) {
		case "trace":
			logLevelDefault = "trace"
		case "debug":
			logLevelDefault = "debug"
		case "info":
			logLevelDefault = "info"
		case "warn":
			logLevelDefault = "warn"
		case "error":
			logLevelDefault = "error"
		case "fatal":
			logLevelDefault = "fatal"
		default:
			logLevelDefault = "info"
			log.Infoc(ctxS, "Invalid log level", "ER_LOG_LEVEL", logLevelEnvVar, "using", logLevelDefault)
		}
	} else {
		logLevelDefault = "info"
	}
	logLevelStr := flag.String("log-level", logLevelDefault, "log level (trace, debug, info, warn, error, fatal)")

	flag.Parse()

	logLevel, err := log.ParseLevel(*logLevelStr)
	if err != nil {
		log.Fatalc(ctxS, "Invalid log level", "err", err)
	}

	log.SetLevel(logLevel)

	if *logLocatime {
		log.SetUseLocalTime(true)
	}

	var l *lumberjack.Logger = nil
	if *logFname != "" {
		l = &lumberjack.Logger{
			Filename:   *logFname,
			MaxSize:    *logMaxSize, // megabytes
			MaxBackups: *logMaxBackup,
			MaxAge:     *logMaxAge, // days
			Compress:   false,      // disabled by default
		}
		log.SetOutput(l)

		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP)

		go func() {
			for {
				<-c
				err := l.Rotate()
				if err != nil {
					log.Errorc(ctxS, "Log rotation failed", "err", err)
				}
			}
		}()
	}

	connectors := &processor.RegisteredProcessors

	connectors.Register("event-generator", &mem.MemGeneratorReaderConf{})
	connectors.Register("qlt2dict", &qlt2json.ConvertStreamConf{})
	connectors.Register("qlt2json", &qlt2json.Convert2JsonConf{})
	connectors.Register("filter", &expr.ExprConf{})
	connectors.Register("control", &processor.ControlConf{})
	connectors.Register("file-writer", &file.FileStoreRawWriterConfig{})
	connectors.Register("file-reader", &file.FileStoreRawReaderConfig{})
	// connectors.Register("mongo-writer", &mongo.MongoWriterConf{})
	connectors.Register("mem-writer", &mem.MemWriterConf{})
	connectors.Register("aws-sqs-writer", &awssqs.AwsSQSWriterConf{})
	// processors.Register("file_json_consumer", &file.FileStoreJsonConsumerConfig{})
	connectors.Register("qlt-client-writer", &qlt.QLTClientWriterConf{}) // Normal mode QLT
	connectors.Register("qlt-server-reader", &qlt.QLTServerReaderConf{}) // Normal mode QLT
	connectors.Register("qlt-client-reader", &qlt.QLTClientReaderConf{}) // Pull mode QLT
	connectors.Register("qlt-server-writer", &qlt.QLTServerWriterConf{}) // Pull mode QLT
	connectors.Register("pg-writer", &postgres.PGWriterConf{})
	connectors.Register("pg-reader", &postgres.PGReaderConf{})
	connectors.Register("sqlite-writer", &sqlite.SqliteWriterConf{})
	connectors.Register("sqlite-reader", &sqlite.SqliteReaderConf{})
	// processors.Register("es_json_consumer", &elasticsearch.EsConsumerConf{})
	// processors.Register("mongo_json_consumer", &mongo.MongoConsumerConf{})
	// processors.Register("lumberjack_json_consumer", &elasticsearch.LumberjackConsumerConf{})
	connectors.Register("kafka-writer", &kafka.KafkaWriterConf{})
	connectors.Register("kafka-reader", &kafka.KafkaReaderConf{})

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatalc(ctxS, "err", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	args := flag.Args()

	if len(args) > 0 {
		switch args[0] {
		case "version":
			fmt.Println("Version:", Version, " Build:", Build, " Date:", Date)
			os.Exit(0)
		case "list-connectors":
			for _, p := range connectors.All() {
				fmt.Println(p.Name)
				typ := reflect.TypeOf(p.Conf).Elem()
				for i := 0; i < typ.NumField(); i++ {
					tag := typ.Field(i).Tag.Get("yaml")
					name := typ.Field(i).Name
					name = strings.ToLower(string(name[0])) + name[1:] // Ack to ensure first letter as lowercase by default
					key := tag
					t := typ.Field(i).Type
					if key == "" {
						key = name
					}
					fmt.Println("  ", key, t)
				}
			}
			os.Exit(0)

		case "list-config":
			config.Print()
			os.Exit(0)
		case "help":
			printHelp(0)
		default:
			printHelp(1)
		}
	}

	log.Infoc(ctxS, "info", "Version", Version, "Build", Build, "Date", Date)

	config.Print()

	conf, err := processor.ParseConfigFile(ctxS, *configFile)
	if err != nil {
		log.Fatalc(ctxS, "Cannot open config file", "err", fmt.Sprint(err))
	}
	if len(conf.Streams) == 0 {
		log.Fatalc(ctxS, "No stream configured")
	}
	log.Infoc(ctxS, "config [internal]", "streams", conf.Streams)
	// Avoid pring paswords
	/*b, _ := yaml.Marshal(conf)

	log.Infoc(ctxS, "config [yaml]:", "marshall", string(b))*/

	// Verify that Clone is properly implemented
	for _, p := range connectors.All() {
		c1 := p.Conf
		c2 := p.Conf.Clone()
		if c1 == c2 {
			log.Fatalc(ctxS, "Internal error: Badly Implemented Clone()", p.Name)
		}
	}

	{ // Verify that all streams have reader and writer defined
		count := 0
		for _, flow := range conf.Streams {
			if flow.Reader == nil {
				log.Errorc(ctxS, "Missing mandatory argument reader", "stream", flow.Name)
				count++
			}
			if flow.Writer == nil {
				log.Errorc(ctxS, "Missing mandatory argument writer", "stream", flow.Name)
				count++
			}
		}
		if count != 0 {
			os.Exit(1)
		}
	}

	{ // Verify that all upstreams are defined
		count := 0
		for _, flow := range conf.Streams {
			if flow.Upstream != "" {
				found := false
				for _, flow2 := range conf.Streams {
					if flow2.Name == flow.Upstream {
						found = true
					}
				}
				if !found {
					count++
					log.Errorc(ctxS, "Upstream not found", "stream", flow.Name, "upstream", flow.Upstream)
				}
			}
		}
		if count != 0 {
			os.Exit(1)
		}
	}

	// Ensure to have an instance_id
	if conf.Instance_id == "" {
		hostname, _ := os.Hostname()
		conf.Instance_id = hostname
	}

	// FIXME: comment
	all := false

	ctx, cancel := context.WithCancel(context.Background())
	readerContext, readerCancel := context.WithCancel(context.Background())
	ctl := make(chan processor.ControlEvent, 100)
	errors := 0

	if log.Level(log.DebugLevel) {
		go processor.ControlEventLogSome(ctxS, ctx, ctl)
	} else {
		go processor.ControlEventDiscardAll(ctxS, ctx, ctl)
	}
	channels := processor.NewChannels()

	var runtimes []*processor.Processor

	for _, flow := range conf.Streams {
		if !flow.Disable {
			r, err := flow.Start(ctx, readerContext, conf.Instance_id, all, ctl, channels, connectors)
			if err != nil {
				errors++
			}
			runtimes = append(runtimes, r...)
		}
	}

	if errors > 0 {
		log.Fatalc(ctxS, "Error occurs while starting streams")
		os.Exit(1)
	}

	state := RouterState{
		Distribution: Distribution{Version, Build, Date},
		Config:       conf,
		Channels:     channels,
		Processors:   runtimes,
	}
	/*if fileCsvProducerConf.Filename != "" {
		go fileCSVProducer(&fileCsvProducerConf, nil, producer1)
	}*/

	/*log.Println("[HTTP] Setting up / welcome...")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to event-router!")
	})*/

	/* If I can answer, I'm alive */
	log.Infoc(ctxS, "[HTTP] Setting up /live...")
	http.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		channels.Update()
		log.Tracec(ctxS, "Status check: Alive")
		w.WriteHeader(http.StatusOK)
	})

	/* Verifies that all streams are ready */
	log.Infoc(ctxS, "[HTTP] Setting up /ready...")
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		channels.Update()
		if Ready {
			log.Tracec(ctxS, "Status check: Ready")
			w.WriteHeader(http.StatusOK)
		} else {
			log.Tracec(ctxS, "Status check: Not ready")
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})

	log.Infoc(ctxS, "[HTTP] Setting up /metrics (prometheus)...")
	http.Handle("/metrics", promhttp.Handler())

	log.Infoc(ctxS, "[HTTP] Setting up /api...")
	http.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		channels.Update()
		b, err := json.Marshal(state)
		if err != nil {
			log.Errorc(ctxS, "Error Marshalling state", "err", err)
			w.Write([]byte("Error Marshalling state"))
			w.WriteHeader(400)
			return
		}
		w.Write(b)
	})

	log.Infoc(ctxS, "[HTTP] Setting up / (static)...")
	live := true
	if live {
		fs := http.FileServer(http.Dir("./src/main/ui"))
		http.Handle("/", fs)
	} else {
		fs2, _ := fs.Sub(ui, "ui")
		fs := http.FileServer(http.FS(fs2))
		http.Handle("/", fs)
	}

	log.Infoc(ctxS, "[HTTP] Listening on "+*host+":"+*port)
	go http.ListenAndServe(*host+":"+*port, nil)

	time.Sleep(1 * time.Second)
	channels.Display(ctxS)

	hup := make(chan os.Signal, 2)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		for range hup {
			log.Infoc(ctxS, "Got A HUP Signal! Now Reloading Conf....")
			channels.Display(ctxS)
		}
	}()

	Ready = true
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Infoc(ctxS, "terminate")
	Ready = false
	channels.Display(ctxS)
	readerCancel()
	time.Sleep(1 * time.Second)
	cancel()
	time.Sleep(1 * time.Second)
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatalc(ctxS, err.Error())
		}
		pprof.WriteHeapProfile(f)
		f.Close()
		return
	}
}
