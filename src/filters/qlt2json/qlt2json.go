package qlt2json

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"axway.com/qlt-router/src/processor"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html/charset"
)

// <TrkDescriptor>
// 	    <TrkXML VERSION="1.0"/>
// 		<TrkObject>
// 			<TrkIdentifier TYPE="Event" NAME="XFBTransfer" VERSION="1.0"/>
// 			<TrkAttr name="PRODUCTNAME" val="CFT"/>
// 		</TrkObject>
// </TrkDescriptor>

type trkDescriptor struct {
	XMLName   xml.Name `xml:"TrkDescriptor"`
	TrkObject trkObject
}

type trkObject struct {
	XMLName       xml.Name      `xml:"TrkObject"`
	TrkIdentifier trkIdentifier `xml:"TrkIdentifier"`
	TrkAttr       []trkAttr     `xml:"TrkAttr"`
}

type trkIdentifier struct {
	XMLName xml.Name `xml:"TrkIdentifier"`
	Type    string   `xml:"TYPE,attr"`
	Name    string   `xml:"NAME,attr"`
	Version string   `xml:"VERSION,attr"`
	// Attributes []xml.Attr `xml:",any,attr"`
}

func (mf *trkIdentifier) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// Attributes
	for _, attr := range start.Attr {
		name := strings.ToLower(attr.Name.Local)
		switch name {
		case "type":
			mf.Type = attr.Value
		case "name":
			mf.Name = attr.Value
		case "version":
			mf.Version = attr.Value
		}
	}
	return d.Skip()
}

type trkAttr struct {
	XMLName xml.Name `xml:"TrkAttr"`
	Name    string   `xml:"name,attr"`
	Val     string   `xml:"val,attr"`
}

func convert1(data string) (string, error) {
	var c trkDescriptor
	err := xml.Unmarshal([]byte(data), &c)
	if err != nil {
		return "", err
	}
	// fmt.Println(c.TrkObject)
	a := make([]string, 0) // FIXME: site is duplicated
	a = append(a, fmt.Sprintf(`"qlttype": "%s" `, c.TrkObject.TrkIdentifier.Type))
	a = append(a, fmt.Sprintf(`"qltname": "%s" `, c.TrkObject.TrkIdentifier.Name))
	for _, e := range c.TrkObject.TrkAttr {
		if strings.ToLower(e.Name) == "location" {
			// Noting
		} else if e.Val != "" && e.Val != " " {
			a = append(a, fmt.Sprintf(`"%s": "%s"`, strings.ToLower(e.Name), e.Val))
		}
	}
	return "{" + strings.Join(a, ", ") + "}", nil
}

func convertToMap(data string) (map[string]string, error) {
	var c trkDescriptor

	// err := xml.Unmarshal([]byte(data), &c)

	// Escape invalid '<' in attribute value
	r := regexp.MustCompile("=[ ]*\"([^\"]*)<([^\"]*)\"")
	din := data

	for {
		dout := r.ReplaceAllString(din, "=\"$1&lt;$2\"")
		if len(dout) == len(din) {
			break
		}
		din = dout
	}
	reader := strings.NewReader(din)
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = charset.NewReaderLabel
	err := decoder.Decode(&c)
	if err != nil {
		return nil, err
	}
	// fmt.Println(c.TrkObject)
	a := make(map[string]string) // FIXME: site is duplicated
	a["qlttype"] = c.TrkObject.TrkIdentifier.Type
	a["qltname"] = c.TrkObject.TrkIdentifier.Name
	for _, e := range c.TrkObject.TrkAttr {
		/*if strings.ToLower(e.Name) == "location" {
			// Noting
		} else */if e.Val != "" && e.Val != " " {
			a[strings.ToLower(e.Name)] = e.Val
		}
	}

	// b, err := json.Marshal(a)
	return a, err
}

func ConvertToJSON(msg map[string]string) string {
	a := make([]string, 0)
	for k, v := range msg {
		t := fields[k]
		if t == "i" {
			a = append(a, `"`+k+`": `+v)
		} else {
			a = append(a, `"`+k+`": "`+v+`"`)
		}
	}
	return "{" + strings.Join(a, ", ") + "}"
}

func convertToMapInterface(msg map[string]string) map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range msg {
		t := fields[k]
		if t == "i" {
			i, err := strconv.Atoi(v)
			if err != nil {
				m[k] = v
			} else {
				m[k] = i
			}
		} else {
			m[k] = v
		}
	}
	return m
}

func convertToQLTXML(msg map[string]string) string {
	header := `<?xml version="1.0" encoding="UTF-8"?>
	<TrkDescriptor> 
		<TrkXML VERSION="1.0"/> 
		<TrkObject>`
	tail := ` 
		</TrkObject> 
	</TrkDescriptor>`
	a := make([]string, 0)
	a = append(a, header)
	a = append(a, fmt.Sprintf(`<TrkIdentifier TYPE="%s" NAME="%s" VERSION="1.0"/>`, msg["qlttype"], msg["qltname"]))
	for k, v := range msg {
		v2 := strings.Replace(v, "<", "&lt;", -1)
		a = append(a, fmt.Sprintf(`<TrkAttr name="%s" val="%s"/>`, k, v2))
	}
	a = append(a, tail)
	return strings.Join(a, "\n")
}

// numerical value
// key : lowcase  -
// values : lower
// date : + gmtdiff
//
// certificate - tenant
// SSO / logout : OUM / AxwayID
const xfbtransfer = `
{
	"blocksize" : "i",
	"creationdate" : "d",
	"creationtime" : "t",
	"earliestdate" : "*",
	"earliesttime" : "*",
	"enddate": "d",
	"eventdate": "d-",
	"eventtime": "d-",
	"filesize" : "i",
	"recordnumber" : "i",
	"recordsize" : "i",
	"requestcreationdate": "d",
	"requestcreationtime": "t",
	"retrymaxnumber" : "i",
	"retrynumber" : "i",
	"retrywaittime": "i",
	"senddate": "d",
	"sendtime": "t",
	"startdate": "d",
	"starttime": "t",
	"transmissionduration": "i",
	"transmittedbytes": "i" 
}`

var fields map[string]string

/*
func convertStreamDummy(eventsIn chan qlt.QLTEvent, eventOut chan AckableEvent) {
	for {
		qltEvent := <-eventsIn
		msg := make(map[string]string)
		eventOut <- AckableEvent{qltEvent.qlt, qltEvent.msgid, msg, nil}
	}
}*/

func convertStreamDummy2(eventsIn chan processor.AckableEvent, eventOut chan processor.AckableEvent) {
	for {
		qltEvent := <-eventsIn
		event, err := convertToMap(qltEvent.Msg.(string))
		if err != nil {
			log.Errorln(qltEvent.Src.Ctx(), "[", qltEvent.Msgid, "] XML Parsing failed '", err, "' Closing...")
			qltEvent.Src.AckMsg(qltEvent.Msgid)
			// qltEvent.Ack <- false
			continue
		}
		eventOut <- processor.AckableEvent{qltEvent.Src, qltEvent.Msgid, event, &qltEvent}
	}
}

func convertStreamNoTransform(p *processor.Processor, eventsIn chan processor.AckableEvent, eventOut chan processor.AckableEvent) {
	for {
		qltEvent := <-eventsIn
		// log.Debugln("[No Transform]", qltEvent.qlt.ctxMsg(), qltEvent.msgid)
		eventOut <- processor.AckableEvent{qltEvent.Src, qltEvent.Msgid, make(map[string]string), &qltEvent}

	}
}

type Convert2JsonConf struct{ _ string }

func (conf *Convert2JsonConf) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, eventsIn, eventOut chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	log.Infoln("[ConvertStreamXML2Dict] start")
	done := ctx.Done()
	go func() {
		for {
			select {
			case qltEvent := <-eventsIn:
				msg, err := convert1(qltEvent.Msg.(string))
				if err != nil {
					log.Errorln(qltEvent.Src.Ctx(), "convertStream [", qltEvent.Msgid, "] XML Parsing failed '", err, "' Closing...")
					eventOut <- processor.AckableEvent{qltEvent.Src, qltEvent.Msgid, nil, &qltEvent}
					continue
				}
				eventOut <- processor.AckableEvent{qltEvent.Src, qltEvent.Msgid, msg, &qltEvent}
			case <-done:
				log.Infoln("[ConvertStreamXML2Dict] done")
				return
			}
		}
	}()
	return nil, nil
}

func (c *Convert2JsonConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

type ConvertStreamConf struct{ _ string }

func (conf *ConvertStreamConf) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, eventsIn, eventOut chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	log.Infoln("[ConvertStreamXML2Dict] start")
	done := ctx.Done()
	go func() {
		for {
			select {
			case qltEvent := <-eventsIn:
				event, err := convertToMap(qltEvent.Msg.(string))
				if err != nil {
					log.Errorln(qltEvent.Src.Ctx(), "convertStream [", qltEvent.Msgid, "] XML Parsing failed '", err, "' Closing...")
					eventOut <- processor.AckableEvent{qltEvent.Src, qltEvent.Msgid, nil, &qltEvent}
					continue
				}

				// log.Println(qltEvent.qlt.ctx, "[", qltEvent.count, "] JSON :", event)
				// log.Println(q.ctx, "[", count, "] Pushing to ESQueue... ")

				// log.Println("", "Process Event... ")
				msg := processEvent(event)
				eventOut <- processor.AckableEvent{qltEvent.Src, qltEvent.Msgid, msg, &qltEvent}
			case <-done:
				log.Infoln("[ConvertStreamXML2Dict] done")
				return
			}
		}
	}()
	return nil, nil
}

func (c *ConvertStreamConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}

func processEvent(values map[string]string) map[string]string {
	n := make(map[string]string)

	for k, v := range values {
		t := fields[k]
		if t == "i" {
			n[k] = v
		} else if t == "t" {
			prefix := k[:len(k)-4]
			date := values[prefix+"date"]
			time := v
			d2 := strings.Replace(date, ".", "-", -1) + "T" + time + ".0000000Z"
			n[k] = d2

		} else if t == "d" {
			// ignore
		} else if t == "" {
			n[k] = v
		}
	}
	return n
}

func init() {
	fields = make(map[string]string)
	if err := json.Unmarshal([]byte(xfbtransfer), &fields); err != nil {
		panic(err)
	}
}
