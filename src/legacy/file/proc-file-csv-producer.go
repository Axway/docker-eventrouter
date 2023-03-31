package file

import (
	"bufio"
	"context"
	"encoding/csv"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"axway.com/qlt-router/src/processor"
	"github.com/ulikunitz/xz/lzma"

	log "github.com/sirupsen/logrus"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var escapeMap = map[string]string{
	">": "&gt;",
	"<": "&lt;",
	"'": "&apos;",
	`"`: "&quot;",
	"&": "&amp;",
}

func escapeMapFunc(item string) string {
	t := escapeMap[item]
	return t
}

var m1 *regexp.Regexp

func xmlEscape(s string) string {
	if m1.MatchString(s) {
		return m1.ReplaceAllStringFunc(s, escapeMapFunc)
	} else {
		return s
	}
	// return s.replace(new RegExp("([&\"<>'])", "g"), (str, item) => escapeMap[item])
}

func anonymize(s string) string {
	pi := 0
	out := ""
	const password = "dksqjghkglkfjhgaezrohgdjsgjbdsklgfdgsdfgqdfl"
	const mod = len(password)
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			// out += string(int('A') + (int(s[i])+int(password[pi%mod]))%26)
			pi++
		} else if c >= 'a' && c <= 'z' {
			// out += string(int('a') + (int(s[i])+int(password[pi%mod]))%26)
			pi++
		} else if c >= '0' && c <= '9' {
			// out += string(int('0') + (int(s[i])+int(password[pi%mod]))%10)
			pi++
		} else {
			out += string(c)
		}
	}
	return out
}

func a2o(a []string) map[string]bool {
	o := make(map[string]bool)
	for _, a := range a {
		o[a] = true
	}
	return o
}

var (
	excludeFields    map[string]bool
	anonymizedFields map[string]bool
	dateFields       map[string]bool
	knownDates       map[string]string
)

func init() {
	m1 = regexp.MustCompile("([&\"<>'\u0001-\u001A])")
	excludeFields = a2o([]string{"EVENTID", "OBJECTID", "AGENTIPPORT", "AGENTIPADDR", "INTERNALSTATE"})
	dateFields = a2o([]string{"EVENTDATE", "CREATIONDATE", "ENDDATE", "REQUESTCREATIONDATE", "STARTDATE", "EARLIESTDATE", "LATESTDATE"})
	knownDates = make(map[string]string)
	anonymizedFields = a2o([]string{
		"AGENTIPADDR",
		"PRODUCTIPADDR",
		"APPLICATION",
		"ORIGINALSENDERID",
		"FINALRECEIVERID",
		"LOCALID",
		"PROTOCOLFILENAME",
		"FILENAME",
		"GROUPID",
		"RAPPL",
		"REQUESTUSERID",
		"SITE",
		"USERID",
		"RECEIVERID",
		"SENDERID",

		"PROTOCOLFILELABEL",
		"PROTOCOLPARAMETER",
	})
}

const (
	escape          = true
	anonymiseOption = false
)

func convertDate(k, item string) string {
	// layout := "2006-01-02T15:04:05.000Z"
	layout := "02-Jan-2006 00:00:00"
	// str := "2014-11-12T11:45:26.371Z"
	t, err := time.Parse(layout, item)

	if err != nil {
		log.Errorln(item, err)
	} else {
		date := t.Format("2006.01.02")
		if knownDates[item] == "" {
			newMap := make(map[string]string)
			for k, v := range knownDates {
				newMap[k] = v
			}
			newMap[item] = date

			knownDates = newMap
			log.Debugln("date", k, item, "-->", date)
			// console.log("Convert new date", k, "'" + item + "' -->", date)
		}
		item = date
	}
	return item
}

func prepareXFBTransfer(headers []string, items []string) string {
	// var buf []string
	var sb strings.Builder
	sb.WriteString(
		`<?xml version="1.0" encoding="UTF-8"?>
    <TrkDescriptor>
                <TrkXML VERSION="1.0"/>
                <TrkObject>
                        <TrkIdentifier TYPE="Event" NAME="XFBTransfer" VERSION="1.0"/>`)

	n := min(len(items), len(headers))
	for i := 0; i < n; i++ {
		if items[i] != "" {
			k := headers[i]
			item := items[i]
			if excludeFields[k] {
				continue
			}
			/*if (!knownFields[k]) {
				console.log("Discover unknown field", k, "'" + item + "'")
				knownFields[k] = true
			  }*/
			if dateFields[k] {
				if knownDates[item] != "" {
					item = knownDates[item]
				} else {
					item = convertDate(k, item)
				}
			}
			if anonymiseOption && anonymizedFields[k] {
				anon := anonymize(item)
				// log.Debugln("anonymise", k, item, anon)
				item = anon
			}
			if escape {
				sb.WriteString(`<TrkAttr name="` + k + `" val="` + xmlEscape(item) + `"/>\n`)
			} else {
				sb.WriteString(`<TrkAttr name="` + k + `" val="` + item + `"/>\n`)
			}

		}
	}

	sb.WriteString(`</TrkObject></TrkDescriptor>`)

	return sb.String()
}

func forward(fileStoreQueueIn []chan processor.AckableEvent, fileStoreQueueOut chan processor.AckableEvent) {
	count := 0
	n := len(fileStoreQueueIn)

	for {
		count++
		msg := <-fileStoreQueueIn[count%n]
		fileStoreQueueOut <- processor.AckableEvent{msg.Src, msg.Msgid, msg.Msg, nil}
	}
}

type FileCSVEventSource struct {
	ctx       string
	ackedMsg  int64
	count     int64
	processor *processor.Processor
}

type FileCSVProducerConfig struct {
	Filename   string
	Offset     int
	Count      int
	FollowTime bool
	Scale      int
}

func (q *FileCSVEventSource) AckMsg(msgid int64) {
	q.ackedMsg = msgid
	q.processor.Out_ack++
	processor.QltMessageInAcked.Inc()
}

func (q *FileCSVEventSource) Ctx() string {
	return q.ctx
}

type EncodeConf struct {
	headers *[]string
}

func (c *EncodeConf) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, fileStoreQueueIn chan processor.AckableEvent, fileStoreQueueOut chan processor.AckableEvent) {
	for {
		select {
		case msg := <-fileStoreQueueIn:
			xml := prepareXFBTransfer(*c.headers, msg.Msg.([]string))
			fileStoreQueueOut <- processor.AckableEvent{msg.Src, msg.Msgid, xml, nil}
		case <-ctx.Done():
			return
		}
	}
}

func (conf *FileCSVProducerConfig) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, in chan processor.AckableEvent, fileStoreQueue chan processor.AckableEvent) {
	// const ctx = "[FILE-CSV-PRODUCER] "
	q := FileCSVEventSource{}
	q.ctx = "[FILE-CSV-PRODUCER] " + p.Flow.Name
	q.ackedMsg = 0
	q.count = 0
	q.processor = p

	log.Println(q.ctx, "conf", conf)

	f, err := os.OpenFile(conf.Filename, os.O_RDONLY, 0o644)
	if err != nil {
		log.Errorln(q.ctx, "Error opening file readonly", conf.Filename, err)
		log.Fatal(err)
	}
	defer f.Close()

	r, err := lzma.NewReader(bufio.NewReader(f))
	if err != nil {
		log.Errorln(q.ctx, "Error creating lzma stream", conf.Filename, err)
		log.Fatal(err)
	}

	count := 0
	errorCount := 0
	headersM := make(map[string]int)
	headers := make([]string, 0)
	reader := csv.NewReader(r)
	start := time.Now()

	n := conf.Scale

	out := fileStoreQueue
	if n > 0 {
		out = processor.CreateChannel("file-csv-producer-fanout")
		proc := processor.NewProcessor("encode", &EncodeConf{&headers})
		go processor.ParallelOrdered(ctx, "file-csv-producer", n, ctl, out, fileStoreQueue, proc)
	}
	EVENTDATE := ""
	for {
		record, err := reader.Read()
		count++

		if err == io.EOF {
			break
		} else if err != nil {
			errorCount++
			// fmt.Println("Error:", err)
			if record == nil {
				return
			}
		}
		if count == 1 {
			for idx, header := range record {
				headersM[header] = idx
				headers = append(headers, header)
			}
			continue
		}

		if conf.Offset > count {
			continue
		}
		if conf.Offset == count {
			start = time.Now()
			log.Warnln(q.ctx, "offset reached", count, conf.Count+conf.Offset)
		}

		if false && conf.FollowTime {
			recordID := record[headersM["EVENTID"]]
			recordTime := record[headersM["EVENTTIME"]]
			recordDate := convertDate(EVENTDATE, record[headersM["EVENTDATE"]])
			recordGMTDIFF := record[headersM["GMTDIFF"]]
			recordDateTime := (recordDate + " " + recordTime[0:5])
			if EVENTDATE != recordDateTime {
				log.Debugln(q.ctx, "follow-time", recordID, recordDate, recordTime, recordGMTDIFF, recordDateTime, count)
				EVENTDATE = recordDateTime
			}
			// log.Debugln(q.ctx, "follow-time", recordDate)
			// log.Debugln(q.ctx, "follow-time", recordDate, recordTime)

			if false {
				tn := time.Now()
				tr, err := time.Parse("2006.01.02 15:04:05 -0700", tn.Format("2006.01.02 ")+recordTime+" +0100")
				if err != nil {
					log.Debugln(q.ctx, "follow-time", err)
				}

				sub := int(tr.UnixMilli()-tn.UnixMilli()) - 30*3600000
				// log.Debugln(q.ctx, "follow-time now", tn)
				// log.Debugln(q.ctx, "follow-time rec", tr)
				// log.Debugln(q.ctx, "follow-time diff", sub+1000)
				if sub > 0 {
					log.Debugln(q.ctx, "follow-time sleep", sub/1000.0, count, recordTime, recordDate)
					time.Sleep(time.Duration(sub) * time.Millisecond)
				}
			}
		}
		/*xml := prepareXFBTransfer(headers, record)
		if len(xml) == 0 {
			break
		}*/
		stuck := false
		done := ctx.Done()
	loop:
		for {
			if n > 0 {
				select {
				case out <- processor.AckableEvent{&q, int64(count), record, nil}:
					if stuck {
						log.Warnln(q.ctx, "unstuck", count)
						stuck = false
					}
					q.processor.Out++
					q.processor.OutCounter.Inc()
					processor.QltMessageInSize.Observe(float64(len(strings.Join(record, ","))))
					break loop
				case <-time.After(1 * time.Second):
					if !stuck {
						stuck = true
						log.Warnln(q.ctx, "Stuck?", count)
						processor.DisplayChannels()
					}
				case <-done:
					log.Infoln(q.ctx, "done")
					return
				}
			} else {
				xml := prepareXFBTransfer(headers, record)
				select {
				case fileStoreQueue <- processor.AckableEvent{&q, int64(count), xml, nil}:
					if stuck {
						log.Warnln(q.ctx, "unstuck", count)
						stuck = false
					}
					q.processor.Out++
					q.processor.OutCounter.Inc()
					processor.QltMessageInSize.Observe(float64(len(xml)))
					break loop
				case <-time.After(1 * time.Second):
					if !stuck {
						stuck = true
						log.Warnln(q.ctx, "Stuck?", count)
						processor.DisplayChannels()
					}
				case <-done:
					log.Infoln(q.ctx, "done")
					return
				}
			}
		}

		// log.Debugln(count, xml)
		if conf.Count > 0 && count >= conf.Count+conf.Offset {
			break
		}

		// fmt.Println(record) // record has the type []string
	}
	elapsed := time.Since(start)
	log.Printf(q.ctx, "time=%s msg=%d avg=%d msg/s errors=%d (%s)", elapsed, count, 1000*(count-conf.Offset)/int(float64(elapsed.Milliseconds())), errorCount, errorCount*100/count)
}
