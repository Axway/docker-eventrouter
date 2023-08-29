package qlt

import "axway.com/qlt-router/src/config"

var (
	qltClientConnectTimeout       = config.DeclareDuration("connectors.qlt.clientConnectTimeout", "1s", "Qlt client connection timeout")
	qltClientConnectionRetryDelay = config.DeclareDuration("connectors.qlt.clientRetryDelay", "100ms", "Qlt client intial retry delay")
	qltWriterAckTimeout           = config.DeclareDuration("connectors.qlt.writerAckTimeout", "10s", "Qlt writer ack reception timeout, ack being expected")
	qltReaderBlockTimeout         = config.DeclareDuration("connectors.qlt.readerBlockTimeout", "200ms", "Qlt read block timeout, new or partial")
	qltAckQueueSize               = config.DeclareInt("connectors.qlt.ackQueueSize", 100, "Qlt ack queue size")
)
