package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

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

var Version = ""
var Build = ""
var Date = ""

func main() {
	formatter := new(prefixed.TextFormatter)
	formatter.DisableTimestamp = false
	formatter.FullTimestamp = true
	formatter.TimestampFormat = "2006-01-02 15:04:05.000000000"
	log.SetFormatter(formatter)
	log.SetLevel(log.DebugLevel)

	/*log.SetFormatter(&log.TextFormatter{
		//		DisableColors: true,
		FullTimestamp: true,
	})*/

	var qltHost = ""
	var qltPort = ""

	var qltsHost = ""
	var qltsPort = ""
	var qltsCert = ""
	var qltsKey = ""
	var qltsCa = ""

	var ELASTICSEARCH = ""
	var SENTINEL = ""
	var SENTINEL_CONNECTIONS = 1
	var FILENAME = ""
	var LUMBERJACK = ""
	var LUMBERJACKCA = ""
	var LUMBERJACKCERT = ""
	var LUMBERJACKKEY = ""

	//var SENTINEL = "qlt:3333"

	flag.String(flag.DefaultConfigFlagname, "", "path to config file")

	flag.StringVar(&SENTINEL, "sentinel_addrs", "", "target sentinels (comma separated")
	flag.IntVar(&SENTINEL_CONNECTIONS, "sentinel_connections", 1, "number of connections per sentinel server")
	flag.StringVar(&ELASTICSEARCH, "es_url", "", "target elasticsearch")
	flag.StringVar(&FILENAME, "filename", "", "target file")
	flag.StringVar(&LUMBERJACK, "lumberjack_addr", "", "target lumberjack")
	flag.StringVar(&LUMBERJACKCA, "lumberjack_ca", "", "target lumberjack CA")
	flag.StringVar(&LUMBERJACKCERT, "lumberjack_cert", "", "target lumberjack CERT")
	flag.StringVar(&LUMBERJACKKEY, "lumberjack_key", "", "target lumberjack CERT KEY")

	flag.StringVar(&qltPort, "qlt_port", "", "QLT listening port (example: 3333")
	flag.StringVar(&qltHost, "qlt_host", "0.0.0.0", "QLT listening host")

	flag.StringVar(&qltsPort, "qlts_port", "", "QLT listening port (examplte: 3334")
	flag.StringVar(&qltsHost, "qlts_host", "0.0.0.0", "QLT listening host")
	flag.StringVar(&qltsCert, "qlts_cert", "./certs/server.pem", "QLT listening host")
	flag.StringVar(&qltsKey, "qlts_key", "./certs/server.key", "QLT listening host")
	flag.StringVar(&qltsCa, "qlts_ca", "", "QLT listening host")

	flag.Parse()

	log.Println("[MAIN] Version:", Version, " Build:", Build, " Date:", Date)

	convertInit()

	go httpInit()

	queues := []chan QLTMessage{}

	if SENTINEL != "" {
		QLTCQueue := make(chan QLTMessage)
		queues = append(queues, QLTCQueue)
		go qltClientInit(SENTINEL, SENTINEL_CONNECTIONS, QLTCQueue)
	}

	if ELASTICSEARCH != "" {
		ESQueue := make(chan QLTMessage)
		queues = append(queues, ESQueue)
		go esInit(ELASTICSEARCH, ESQueue)
	}

	if FILENAME != "" {
		FSQueue := make(chan QLTMessage)
		queues = append(queues, FSQueue)
		go fileStoreInit(FILENAME, FSQueue)
	}

	if LUMBERJACK != "" {
		LBQueue := make(chan QLTMessage)
		queues = append(queues, LBQueue)
		go lumberJackInit(LUMBERJACK, LUMBERJACKCA, LUMBERJACKCERT, LUMBERJACKKEY, LBQueue)
	}

	if qltPort != "" {
		tcpServe(qltHost+":"+qltPort, qltHandleRequest, "QLT-TCP", queues)
	}

	if qltsPort != "" {
		tlsServe(qltsHost+":"+qltsPort, qltsCert, qltsKey, qltsCa, qltHandleRequest, "QLT-TLS", queues)
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
