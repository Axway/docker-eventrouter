package main

import (
	"fmt"
	"log"
	"testing"
)

func ExampleConvert2() {
	json, _ := Convert1(data)
	fmt.Println(json)
	// Output:
	// {"_type" : "Event","_name" : "XFBTransfer","PRODUCTNAME" : "CFT","PRODUCTIPADDR" : "cft"}
}

func TestTopic(t *testing.T) {
	qw := make(chan []byte)
	fileTopicWriteInit("test-topic", qw)
	qw <- []byte("data1")
	qw <- []byte("data2")

	qr := make(chan []byte)
	fileTopicReadInit("test-topic", "test-client", qr)
	for {
		data := <-qr
		log.Println("data", string(data))
	}
}
