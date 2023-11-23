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
	"axway.com/qlt-router/src/filters/qlt2json"
	log "axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
)

var (
	Version = ""
	Build   = ""
	Date    = ""
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

func main() {
	ctxS := "main"
	log.SetLevel(log.InfoLevel)

	configFile := flag.String("config", "./qlt-router.yml", "path to config file")
	port := flag.String("port", "8080", "http port")
	host := flag.String("host", "0.0.0.0", "http host")

	logFname := flag.String("log-file", "", "log file name")
	logMaxSize := flag.Int("log-max-size", 100, "log file max size (MB)")
	logMaxBackup := flag.Int("log-max-backups", 3, "log file backups")
	logMaxAge := flag.Int("log-max-age", 31, "log file max age (days)")
	// logExclude := flag.String("log-exclude", "", "regex to exclude log messages from output")
	logLocatime := flag.Bool("log-localtime", false, "log file max age (days)")
	logLevelStr := flag.String("log-level", "debug", "log level (trace, debug, info, warn)")

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile := flag.String("memprofile", "", "write memory profile to this file")
	// var confTcpChaos tools.TCPChaosConf
	// processor.ParseConfig(&confTcpChaos, "chaos")
	// tools.TcpChaosInit(&confTcpChaos)

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
			flag.PrintDefaults()
			os.Exit(0)
		default:
			flag.PrintDefaults()
			os.Exit(1)
		}
	}

	log.Infoc(ctxS, "info", "Version", Version, " Build", Build, " Date", Date)

	config.Print()

	conf, err := processor.ParseConfigFile(ctxS, *configFile)
	if err != nil {
		log.Fatalc(ctxS, "Cannot open config file", "err", fmt.Sprint(err))
	}
	if len(conf.Streams) == 0 {
		log.Fatalc(ctxS, "Not configured flows")
	}
	log.Infoc(ctxS, "config [internal]", "streams", conf.Streams)
	b, _ := yaml.Marshal(conf)

	log.Infoc(ctxS, "config [yaml]:", "marshall", string(b))

	// Verify that Clone is properly implemented
	for _, p := range connectors.All() {
		c1 := p.Conf
		c2 := p.Conf.Clone()
		if c1 == c2 {
			log.Fatalc(ctxS, "Internal error: Badly Implemented Clone()", p.Name)
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
					log.Errorc(ctxS, "Upstream flow not found", "flow", flow.Name, "upstream", flow.Upstream)
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
		log.Fatalc(ctxS, "error configuring flows")
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
		fmt.Fprintf(w, "Welcome to qlt-router!")
	})*/

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

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Infoc(ctxS, "terminate")
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
