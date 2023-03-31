package qlt_test

import (
	"net"
	"testing"
	"time"

	"axway.com/qlt-router/src/protocols/qlt"
	"axway.com/qlt-router/src/tools"
	log "github.com/sirupsen/logrus"
)

func TestQlt(t *testing.T) {
	port := "9899"
	var qltServer *qlt.QLT
	l, err := tools.TcpServe("localhost:"+port, func(conn net.Conn, ctx string) {
		qltServer = qlt.NewQltServer(ctx+"-conn", conn)
	}, "qlt-server-test")
	if err != nil {
		t.Error("error listening", err)
		return
	}
	defer l.Close()

	c, err := qlt.NewQltClient("[qlt-client-test]", "localhost:"+port)
	if err != nil {
		t.Error("error connecting", err)
		return
	}
	defer c.Close()

	msgSent := "my message"
	log.Println("send message", msgSent)
	err = c.Send(msgSent)
	if err != nil {
		t.Error("error sending message ", err)
		return
	}

	log.Println("waiting qltserver...")
	count := 0
	for qltServer == nil {
		time.Sleep(10 * time.Millisecond)
		count++
	}
	log.Println("qltserver wait count", count)

	msgReceived, err := qltServer.ReadQLTPacket()
	if err != nil {
		t.Error("error reading packet", err)
		return
	}
	log.Println("received message", msgReceived)

	if msgSent != msgReceived {
		t.Error("error different message", msgSent, msgReceived)
		return
	}
	err = qltServer.WriteQLTAck()
	if err != nil {
		t.Error("error send ack", err)
		return
	}

	err = c.WaitAck()
	if err != nil {
		t.Error("error waiting ack", err)
		return
	}

	// t.Error("==Success==")
}
