package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
)

// <TrkDescriptor>
// 	    <TrkXML VERSION="1.0"/>
// 		<TrkObject>
// 			<TrkIdentifier TYPE="Event" NAME="XFBTransfer" VERSION="1.0"/>
// 			<TrkAttr name="PRODUCTNAME" val="CFT"/>
// 		</TrkObject>
// </TrkDescriptor>

type TrkDescriptor struct {
	XMLName   xml.Name `xml:"TrkDescriptor"`
	TrkObject TrkObject
}

type TrkObject struct {
	XMLName       xml.Name      `xml:"TrkObject"`
	TrkIdentifier TrkIdentifier `xml:"TrkIdentifier"`
	TrkAttr       []TrkAttr     `xml:"TrkAttr"`
}

type TrkIdentifier struct {
	XMLName xml.Name `xml:"TrkIdentifier"`
	Type    string   `xml:"TYPE,attr"`
	Name    string   `xml:"NAME,attr"`
	Version string   `xml:"VERSION,attr"`
}

type TrkAttr struct {
	XMLName xml.Name `xml:"TrkAttr"`
	Name    string   `xml:"name,attr"`
	Val     string   `xml:"val,attr"`
}

func Convert1(data string) (string, error) {
	var c TrkDescriptor
	err := xml.Unmarshal([]byte(data), &c)
	if err != nil {
		return "", err
	}
	//fmt.Println(c.TrkObject)
	a := make([]string, 0) //FIXME: site is duplicated
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

func ConvertToMap(data string) (map[string]string, error) {
	var c TrkDescriptor
	err := xml.Unmarshal([]byte(data), &c)
	if err != nil {
		return nil, err
	}
	//fmt.Println(c.TrkObject)
	a := make(map[string]string) //FIXME: site is duplicated
	a["qlttype"] = c.TrkObject.TrkIdentifier.Type
	a["qltname"] = c.TrkObject.TrkIdentifier.Name
	for _, e := range c.TrkObject.TrkAttr {
		/*if strings.ToLower(e.Name) == "location" {
			// Noting
		} else */if e.Val != "" && e.Val != " " {
			a[strings.ToLower(e.Name)] = e.Val
		}
	}

	//b, err := json.Marshal(a)
	return a, err
}

func ConcertToJSON(msg map[string]string) string {
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

func ConvertToQLTXML(msg map[string]string) string {
	var header = `<?xml version="1.0" encoding="UTF-8"?>
	<TrkDescriptor> 
		<TrkXML VERSION="1.0"/> 
		<TrkObject>`
	var tail = ` 
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

var indexName = "xfbtransfer"
var fields map[string]string

func ProcessEvent(values map[string]string) map[string]string {
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
			//ignore
		} else if t == "" {
			n[k] = v
		}
	}
	return n
}

func ConvertInit() {

	fields = make(map[string]string)
	if err := json.Unmarshal([]byte(xfbtransfer), &fields); err != nil {
		panic(err)
	}
}
