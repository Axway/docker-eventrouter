package main

import (
	"errors"
	"io"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

type QLTClient struct {
	conn net.Conn
	addr string
}

func QLTClientCreate(addr string) (*QLTClient, error) {
	var c QLTClient
	c.addr = addr

	if err := c.Connect(); err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *QLTClient) Connect() error {
	log.Println("QLTClient connecting to ", c.addr, "...")
	if conn, err := net.Dial("tcp", c.addr); err != nil {
		log.Println("ERROR: QLTClient Dial failed :", c.addr, err)
		return err
	} else {
		c.conn = conn
	}
	return nil
}

func (c *QLTClient) Close() error {
	return c.conn.Close()
}

func (c *QLTClient) Send(_msg string) error {
	retry := 0
	delay := 100
	timemax := 30000
	for {
		if err := c._Send(_msg); err == io.EOF {
			if retry != 0 {
				delay := delay + delay/2
				if delay > timemax {
					delay = timemax
				}
				log.Println("WARN: QLTClient retrying...", delay/1000)
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
			retry++
			c.Connect()
		} else {
			break
		}
	}
	return nil
}

func (c *QLTClient) _Send(_msg string) error {
	m := []byte(_msg)
	l := len(m)
	wbuf := make([]byte, l+3)
	copy(wbuf[3:l+3], m[:])

	wbuf[0] = (byte)((l + 1) / 256)
	wbuf[1] = (byte)((l + 1) % 256)
	wbuf[2] = '1'
	if _, err := c.conn.Write(wbuf); err != nil {
		return err
	}

	rbuf := make([]byte, 3)

	if len, err := c.conn.Read(rbuf); err != nil {
		return err
	} else if len < 3 {
		return errors.New("Expecting 3 Ack response")
	}
	//FIXME: verify Ack message

	return nil
}

func qltClientInit(addr string, QLTCQueue chan map[string]string) {
	if c, err := QLTClientCreate(addr); err != nil {
		panic(err)
	} else {
		defer c.Close()
		count := 1
		for {
			log.Println("[QLTC] Waiting Message on QLTCQueue...", count)
			event := <-QLTCQueue
			log.Println("[QLTC] Converting Message to xml...", count)
			msg := ConvertToQLTXML(event)
			log.Println("[QLTC] Sending Message to remote...", count, msg)
			if err := c.Send(msg); err != nil {
				panic(err)
			}
			qltMessageOut.Inc()
			qltMessageOutSize.Observe(float64(len(msg)))
			log.Println("[QLTC] Sent", count, msg)
			count++
		}
	}
}
