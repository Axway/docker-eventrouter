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
	"axway.com/qlt-router/src/connectors/qlt"
	"axway.com/qlt-router/src/filters/qlt2json"
	"axway.com/qlt-router/src/locallog"
	log "axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	locallog.InitLog()

	/*log.SetFormatter(&log.TextFormatter{
		//		DisableColors: true,
		FullTimestamp: true,
	})*/
	var configFile string
	var verbose bool
	// flag.String(flag.DefaultConfigFlagname, "", "path to config file")
	flag.StringVar(&configFile, "config", "./qlt-router.yml", "path to config file")
	flag.BoolVar(&verbose, "verbose", false, "be verbose")

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile := flag.String("memprofile", "", "write memory profile to this file")
	// var confTcpChaos tools.TCPChaosConf
	// processor.ParseConfig(&confTcpChaos, "chaos")
	// tools.TcpChaosInit(&confTcpChaos)

	connectors := &processor.RegisteredProcessors

	connectors.Register("event-generator", &mem.MemGeneratorReaderConf{})
	connectors.Register("qlt2dict", &qlt2json.ConvertStreamConf{})
	connectors.Register("qlt2json", &qlt2json.Convert2JsonConf{})
	connectors.Register("control", &processor.ControlConf{})
	connectors.Register("file-raw-writer", &file.FileStoreRawWriterConfig{})
	connectors.Register("file-raw-reader", &file.FileStoreRawReaderConfig{})
	connectors.Register("mem-writer", &mem.MemWriterConf{})
	connectors.Register("aws-sqs-writer", &awssqs.AwsSQSWriterConf{})
	// processors.Register("file_json_consumer", &file.FileStoreJsonConsumerConfig{})
	connectors.Register("qlt-client-writer", &qlt.QLTClientWriterConf{})
	connectors.Register("qlt-server-reader", &qlt.QLTServerReaderConf{})
	// processors.Register("pg_buffer_consumer", &postgres.PgDBConsumerConf{})
	// processors.Register("pg_buffer_producer", &postgres.PgDBProducerConf{})
	// processors.Register("es_json_consumer", &elasticsearch.EsConsumerConf{})
	// processors.Register("mongo_json_consumer", &mongo.MongoConsumerConf{})
	// processors.Register("lumberjack_json_consumer", &elasticsearch.LumberjackConsumerConf{})
	connectors.Register("kafka-writer", &kafka.KafkaWriterConf{})
	connectors.Register("kafka-reader", &kafka.KafkaReaderConf{})

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err.Error())
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

	log.Info("[MAIN]", "Version", Version, " Build", Build, " Date", Date)

	config.Print()

	conf, err := processor.ParseConfigFile(configFile)
	if err != nil {
		log.Fatal("Cannot open config file", "err", fmt.Sprint(err))
	}
	if len(conf.Streams) == 0 {
		log.Fatal("Not configured flows")
	}
	log.Info("config [internal]", "streams", conf.Streams)
	b, _ := yaml.Marshal(conf)

	log.Info("config [yaml]:", "marshall", b)

	// Verify that CLone is properly implemented
	for _, p := range connectors.All() {
		c1 := p.Conf
		c2 := p.Conf.Clone()
		if c1 == c2 {
			log.Fatal("Internal error: Badly Implemented Clone()", p.Name)
		}
	}

	// FIXME: comment
	all := false

	ctx := context.Background()
	ctl := make(chan processor.ControlEvent, 100)
	errors := 0
	if verbose {
		go processor.ControlEventLogAll(ctx, ctl)
	} else {
		go processor.ControlEventDiscardAll(ctx, ctl)
	}
	channels := processor.NewChannels()

	var runtimes []*processor.Processor

	for _, flow := range conf.Streams {
		if !flow.Disable {
			r, err := flow.Start(ctx, all, ctl, channels, connectors)
			if err != nil {
				errors++
			}
			runtimes = append(runtimes, r...)
		}
	}

	if errors > 0 {
		log.Fatal("error configuring flows")
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

	log.Info("[HTTP] Setting up /metrics (prometheus)...")
	http.Handle("/metrics", promhttp.Handler())

	log.Info("[HTTP] Setting up /api...")
	http.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		channels.Update()
		b, err := json.Marshal(state)
		if err != nil {
			log.Error("Error Marshalling state", "err", err)
			w.Write([]byte("Error Marshalling state"))
			w.WriteHeader(400)
			return
		}
		w.Write(b)
	})

	log.Info("[HTTP] Setting up / (static)...")
	live := true
	if live {
		fs := http.FileServer(http.Dir("./src/main/ui"))
		http.Handle("/", fs)
	} else {
		fs2, _ := fs.Sub(ui, "ui")
		fs := http.FileServer(http.FS(fs2))
		http.Handle("/", fs)
	}

	log.Info("[HTTP] Listening on localhost:9900")
	go http.ListenAndServe("localhost:9900", nil)

	time.Sleep(1 * time.Second)
	channels.Display()

	hup := make(chan os.Signal, 2)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		for range hup {
			log.Info("Got A HUP Signal! Now Reloading Conf....")
			channels.Display()
		}
	}()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Info("terminate")
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err.Error())
		}
		pprof.WriteHeapProfile(f)
		f.Close()
		return
	}
}
