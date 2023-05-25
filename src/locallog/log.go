package locallog

import (
	log "axway.com/qlt-router/src/log"
)

func InitLog() {
	/*formatter := new(prefixed.TextFormatter)
	formatter.DisableTimestamp = false
	formatter.FullTimestamp = true
	formatter.TimestampFormat = "2006-01-02 15:04:05.000000000"
	log.SetFormatter(formatter)*/
	log.SetLevel(log.DebugLevel)
}

func InitLogSetLevelWarn() {
	log.SetLevel(log.WarnLevel)
}
