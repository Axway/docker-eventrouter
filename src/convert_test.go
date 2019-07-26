package main

import (
	"fmt"
	"testing"
)

var data = `<?xml version="1.0" encoding="UTF-8"?>
    <TrkDescriptor> 
		<TrkXML VERSION="1.0"/> 
		<TrkObject> 
			<TrkIdentifier TYPE="Event" NAME="XFBTransfer" VERSION="1.0"/> 
			<TrkAttr name="PRODUCTNAME" val="CFT"/> 
			<TrkAttr name="PRODUCTIPADDR" val="cft"/> 
		</TrkObject> 
	</TrkDescriptor>`

func ExampleConvert1() {
	json, _ := Convert1(data)
	fmt.Println(json)
	// Output:
	// {"_type" : "Event","_name" : "XFBTransfer","PRODUCTNAME" : "CFT","PRODUCTIPADDR" : "cft"}
}

func TestXmlFailure(t *testing.T) {
	_, err := Convert1("data")
	if err == nil {
		t.Fail()
	}
}

func TestUnexpectedContent(t *testing.T) {
	_, err := Convert1("<data/>")
	if err == nil {
		//fmt.Println("error", err)
		t.Fail()
	}
}
