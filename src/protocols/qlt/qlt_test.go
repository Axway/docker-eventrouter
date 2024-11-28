package qlt_test

import (
	"errors"
	"net"
	"os"
	"syscall"
	"testing"
	"time"

	log "axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/protocols/qlt"
	"axway.com/qlt-router/src/tools"
)

func TestQltPush(t *testing.T) {
	ctxS := "test-" + t.Name()
	port := "9899"
	timeout := 5 * time.Second

	ch := make(chan *qlt.QltServerReader)
	l, err := tools.TcpServe("localhost:"+port, func(conn net.Conn, ctx string) {
		qltServer := qlt.NewQltServerReader(ctx+"-conn", conn)
		ch <- qltServer
	}, "qlt-server-test")
	if err != nil {
		t.Error("error listening", err)
		return
	}
	defer l.Close()

	c := qlt.NewQltClientWriter("[qlt-client-test]", "localhost:"+port, "", "", "")

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
	qltServer := <-ch
	defer qltServer.Close()

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

	err = c.WaitAck(timeout)
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

	ch := make(chan *qlt.QltServerWriter)
	l, err := tools.TcpServe("localhost:"+port, func(conn net.Conn, ctx string) {
		q := qlt.NewQltServerWriter(ctx+"-conn", conn, queueName)
		err := q.WaitQueueName(timeout)
		if err != nil {
			t.Error("error waiting queue name", err)
			ch <- nil
			return
		}
		err = q.AckQueueName()
		if err != nil {
			t.Error("error ack queue name", err)
			ch <- nil
			return
		}

		ch <- q
	}, "qlt-server-test")
	if err != nil {
		t.Error("error listening", err)
		return
	}
	defer l.Close()

	c := qlt.NewQltClientReader("[qlt-client-test]", "localhost:"+port, queueName, "", "", "")

	err = c.Connect(timeout)
	if err != nil {
		t.Error("error connecting", err)
		return
	}
	defer c.Close()

	log.Infoc(ctxS, "waiting qltserver...")
	qltServer := <-ch
	if qltServer == nil {
		return
	}
	defer qltServer.Close()

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

func TestQltPullBadPort(t *testing.T) {
	c := qlt.NewQltClientReader("[qlt-client-test]", "localhost:1", "any", "", "", "")
	timeout := 1000 * time.Millisecond
	err := c.Connect(timeout)
	if err != nil {
		if !errors.Is(err, syscall.ECONNREFUSED) {
			t.Error("this test should fail with reject", err)
		}
		return
	}
	t.Error("this test should fail")
	c.Close()
}

func TestQltPullConnectTimeout(t *testing.T) {
	c := qlt.NewQltClientReader("[qlt-client-test]", "10.255.255.1:443", "any", "", "", "")
	timeout := 100 * time.Millisecond
	err := c.Connect(timeout)
	if err != nil {
		if !os.IsTimeout(err) {
			t.Error("this test should fail with timeout", err)
		}
		return
	}
	t.Error("this test should fail")
	c.Close()
}
