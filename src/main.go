package main

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	qltConnectionIn = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "qlt_cnx",
		Help: "The number of connections",
	})

	qltMessageIn = promauto.NewCounter(prometheus.CounterOpts{
		Name: "qlt_in_message_total",
		Help: "The total number of qlt messages",
	})

	qltMessageInSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "qlt_in_size_message",
		Help:    "Size of qlt in messages",
		Buckets: prometheus.LinearBuckets(0, 1000, 32),
	})

	qltMessageOut = promauto.NewCounter(prometheus.CounterOpts{
		Name: "qlt_out_message_total",
		Help: "The total number of out qlt messages",
	})

	qltMessageOutSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "qlt_out_size_message",
		Help:    "Sizes of qlt out messages",
		Buckets: prometheus.LinearBuckets(0, 1000, 32),
	})
)

func httpInit() {
	log.Println("[HTTP] Setting up / welcome...")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to qlt-router!")
	})
	log.Println("[HTTP] Setting up /metrics (prometheus)...")
	http.Handle("/metrics", promhttp.Handler())

	log.Println("[HTTP] Listening on 0.0.0.0:80")
	http.ListenAndServe(":80", nil)
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		//		DisableColors: true,
		FullTimestamp: true,
	})

	var QLT_HOST = "0.0.0.0"
	var QLT_PORT = "3333"

	var ELASTICSEARCH = ""
	var SENTINEL = ""
	var FILENAME = ""
	var LUMBERJACK = ""

	//var SENTINEL = "qlt:3333"

	flag.String(flag.DefaultConfigFlagname, "", "path to config file")

	flag.StringVar(&SENTINEL, "sentinel_addr", "", "target sentinel")
	flag.StringVar(&ELASTICSEARCH, "es_url", "", "target elasticsearch")
	flag.StringVar(&FILENAME, "filename", "", "target file")
	flag.StringVar(&LUMBERJACK, "lumberjack_addr", "", "target lumberjack")

	flag.StringVar(&QLT_PORT, "qlt_port", "3333", "QLT listening port")
	flag.StringVar(&QLT_HOST, "qlt_host", "0.0.0.0", "QLT listening host")

	flag.Parse()

	ConvertInit()

	go httpInit()

	queues := []chan map[string]string{}

	if SENTINEL != "" {
		QLTCQueue := make(chan map[string]string)
		queues = append(queues, QLTCQueue)
		go qltClientInit(SENTINEL, QLTCQueue)
	}

	if ELASTICSEARCH != "" {
		ESQueue := make(chan map[string]string)
		queues = append(queues, ESQueue)
		go esInit(ELASTICSEARCH, ESQueue)
	}

	if FILENAME != "" {
		FSQueue := make(chan map[string]string)
		queues = append(queues, FSQueue)
		go fileStoreInit(FILENAME, FSQueue)
	}

	if LUMBERJACK != "" {
		LBQueue := make(chan map[string]string)
		queues = append(queues, LBQueue)
		go lumberJackInit(LUMBERJACK, LBQueue)
	}

	//queues := []chan map[string]string{ /*ESQueue, FSQueue,*/ QLTCQueue}
	QLTListen(QLT_HOST+":"+QLT_PORT, queues)
}
