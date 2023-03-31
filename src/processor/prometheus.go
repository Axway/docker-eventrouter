package processor

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	QltConnectionIn = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "qlt_cnx",
		Help: "The number of connections",
	})

	/*qltMessageIn = promauto.NewCounter(prometheus.CounterOpts{
		Name:        "qlt_in_message_total",
		Help:        "The total number of qlt messages",
		ConstLabels: prometheus.Labels{"zouz": "zz"},
	})*/

	QltMessageInAcked = promauto.NewCounter(prometheus.CounterOpts{
		Name: "qlt_acked_in_message_total",
		Help: "The total number of qlt messages acked",
	})

	QltMessageInSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "qlt_in_size_message",
		Help:    "Size of qlt in messages",
		Buckets: prometheus.LinearBuckets(0, 1000, 32),
	})

	QltMessageOut = promauto.NewCounter(prometheus.CounterOpts{
		Name: "qlt_out_message_total",
		Help: "The total number of out qlt messages",
	})

	QltMessageOutSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "qlt_out_size_message",
		Help:    "Sizes of qlt out messages",
		Buckets: prometheus.LinearBuckets(0, 1000, 32),
	})
)
