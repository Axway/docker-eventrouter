package qlt_test

import (
	"net"
	"testing"
	"time"

	log "axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/protocols/qlt"
	"axway.com/qlt-router/src/tools"
)

func TestQltPush(t *testing.T) {
	ctxS := "test-" + t.Name()
	port := "9899"
	var qltServer *qlt.QltServerReader
	l, err := tools.TcpServe("localhost:"+port, func(conn net.Conn, ctx string) {
		qltServer = qlt.NewQltServerReader(ctx+"-conn", conn)
	}, "qlt-server-test")
	if err != nil {
		t.Error("error listening", err)
		return
	}
	defer l.Close()

	c := qlt.NewQltClientWriter("[qlt-client-test]", "localhost:"+port)

	err = c.Connect(200 * time.Millisecond)
	if err != nil {
		t.Error("error connecting", err)
		return
	}
	defer c.Close()

	msgSent := "my message"
	log.Infoc(ctxS, "send message", "msgSent", msgSent)
	err = c.Send(msgSent)
	if err != nil {
		t.Error("error sending message ", err)
		return
	}

	log.Infoc(ctxS, "waiting qltserver...")
	count := 0
	for qltServer == nil {
		time.Sleep(10 * time.Millisecond)
		count++
	}
	log.Infoc(ctxS, "qltserver wait count", "count", count)

	msgReceived, err := qltServer.Read(200 * time.Millisecond)
	if err != nil {
		t.Error("error reading packet", err)
		return
	}
	log.Infoc(ctxS, "received message", "msgReceiverd", msgReceived)

	if msgSent != msgReceived {
		t.Error("error different message", "msgSent", msgSent, "msgReceiverd", msgReceived)
		return
	}
	err = qltServer.WriteAck()
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

func TestQltPull(t *testing.T) {
	ctxS := "test-" + t.Name()
	port := "8399"
	queueName := "myqueue"
	timeout := 200 * time.Millisecond
	var qltServer *qlt.QltServerWriter

	l, err := tools.TcpServe("localhost:"+port, func(conn net.Conn, ctx string) {
		qltServer = qlt.NewQltServerWriter(ctx+"-conn", conn, queueName)
		qltServer.WaitQueueName(timeout)
		qltServer.AckQueueName()
	}, "qlt-server-test")
	if err != nil {
		t.Error("error listening", err)
		return
	}
	defer l.Close()

	c := qlt.NewQltClientReader("[qlt-client-test]", "localhost:"+port, queueName)

	err = c.Connect(timeout)
	if err != nil {
		t.Error("error connecting", err)
		return
	}
	defer c.Close()

	log.Infoc(ctxS, "waiting qltserver...")
	count := 0
	for qltServer == nil {
		time.Sleep(10 * time.Millisecond)
		count++
	}

	log.Infoc(ctxS, "qltserver wait count", "count", count)

	msgSent := "my message" + time.Now().String()
	log.Infoc(ctxS, "send message", "msgSent", msgSent)
	err = qltServer.Send(msgSent)
	if err != nil {
		t.Error("error sending message ", err)
		return
	}

	msgReceived, err := c.Read(timeout)
	if err != nil {
		t.Error("error reading packet", err)
		return
	}
	log.Infoc(ctxS, "received message", "msgReceiverd", msgReceived)

	if msgSent != msgReceived {
		t.Error("error different message", "msgSent", msgSent, "msgReceiverd", msgReceived)
		return
	}
	err = c.WriteAck()
	if err != nil {
		t.Error("error send ack", err)
		return
	}

	err = qltServer.WaitAck(timeout)
	if err != nil {
		t.Error("error waiting ack", err)
		return
	}

	// t.Error("==Success==")
}
