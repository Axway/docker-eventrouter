package main

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"axway.com/qlt-router/src/connectors/file"
	"axway.com/qlt-router/src/connectors/kafka"
	"axway.com/qlt-router/src/connectors/mem"
	"axway.com/qlt-router/src/connectors/qlt"
	"axway.com/qlt-router/src/filters/qlt2json"
	"axway.com/qlt-router/src/locallog"
	"axway.com/qlt-router/src/processor"
	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func httpInit() {
}

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

	flag.String(flag.DefaultConfigFlagname, "", "path to config file")

	// var confTcpChaos tools.TCPChaosConf
	// processor.ParseConfig(&confTcpChaos, "chaos")
	// tools.TcpChaosInit(&confTcpChaos)

	processors := &processor.RegisteredProcessors

	processors.Register("event-generator", &mem.MemGeneratorReaderConf{})
	processors.Register("qlt2dict", &qlt2json.ConvertStreamConf{})
	processors.Register("qlt2json", &qlt2json.Convert2JsonConf{})
	processors.Register("control", &processor.ControlConf{})
	processors.Register("file-raw-writer", &file.FileStoreRawWriterConfig{})
	processors.Register("file-raw-reader", &file.FileStoreRawReaderConfig{})
	// processors.Register("file_json_consumer", &file.FileStoreJsonConsumerConfig{})
	processors.Register("qlt-client-writer", &qlt.QLTClientWriterConf{})
	processors.Register("qlt-server-reader", &qlt.QLTServerReaderConf{})
	// processors.Register("pg_buffer_consumer", &postgres.PgDBConsumerConf{})
	// processors.Register("pg_buffer_producer", &postgres.PgDBProducerConf{})
	// processors.Register("es_json_consumer", &elasticsearch.EsConsumerConf{})
	// processors.Register("mongo_json_consumer", &mongo.MongoConsumerConf{})
	// processors.Register("lumberjack_json_consumer", &elasticsearch.LumberjackConsumerConf{})
	processors.Register("kafka-writer", &kafka.KafkaWriterConf{})
	processors.Register("kafka-reader", &kafka.KafkaReaderConf{})

	flag.Parse()

	conf, err := processor.ParseConfigFile("./qlt-router.yml")
	if err != nil {
		log.Fatalln("Cannot open config file", "err", err)
	}
	if len(conf.Streams) == 0 {
		log.Fatalln("Not configured flows")
	}
	log.Printf("config [internal]: %+v", conf.Streams)
	b, _ := yaml.Marshal(conf)

	log.Printf("config [yaml]: %s", b)

	// Verify that CLone is properly implemented
	for _, p := range processors.All() {
		c1 := p.Conf
		c2 := p.Conf.Clone()
		if c1 == c2 {
			log.Fatal("Internal error: Badly Implemented Clone()", p.Name)
		}
	}

	log.Println("[MAIN] Version:", Version, " Build:", Build, " Date:", Date)

	all := false

	ctx := context.Background()
	ctl := make(chan processor.ControlEvent, 100)
	errors := 0
	go processor.ControlEventLogAll(ctx, ctl)

	channels := processor.NewChannels()

	var runtimes []*processor.Processor

	for _, flow := range conf.Streams {
		if !flow.Disable {
			r, err := flow.Start(ctx, all, ctl, channels, processors)
			if err != nil {
				errors++
			}
			runtimes = append(runtimes, r...)
		}
	}

	if errors > 0 {
		log.Fatalln("error configuring flows")
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

	log.Println("[HTTP] Setting up /metrics (prometheus)...")
	http.Handle("/metrics", promhttp.Handler())

	log.Println("[HTTP] Setting up /api...")
	http.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		channels.Update()
		b, err := json.Marshal(state)
		if err != nil {
			log.Errorln("Error Marshalling state", err)
			w.Write([]byte("Error Marshalling state"))
			w.WriteHeader(400)
			return
		}
		w.Write(b)
	})

	log.Println("[HTTP] Setting up / (static)...")
	live := true
	if live {
		fs := http.FileServer(http.Dir("./src/main/ui"))
		http.Handle("/", fs)
	} else {
		fs2, _ := fs.Sub(ui, "ui")
		fs := http.FileServer(http.FS(fs2))
		http.Handle("/", fs)
	}

	log.Println("[HTTP] Listening on localhost:9900")
	go http.ListenAndServe("localhost:9900", nil)

	time.Sleep(1 * time.Second)
	channels.Display()

	hup := make(chan os.Signal, 2)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		for range hup {
			log.Println("Got A HUP Signal! Now Reloading Conf....")
			channels.Display()
		}
	}()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
