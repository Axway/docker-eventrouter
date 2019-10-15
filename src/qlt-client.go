package main

import (
	"errors"
	"hash/fnv"
	"io"
	"net"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type qltClient struct {
	conn net.Conn
	addr string
}

func qltClientCreate(addr string) (*qltClient, error) {
	var c qltClient
	c.addr = addr

	if err := c.Connect(); err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *qltClient) Connect() error {
	log.Println("QLTClient connecting to ", c.addr, "...")
	conn, err := net.Dial("tcp", c.addr)
	if err != nil {

		log.Println("ERROR: QLTClient Dial failed :", c.addr, err)
		return err
	}
	c.conn = conn
	return nil
}

func (c *qltClient) Close() error {
	return c.conn.Close()
}

func (c *qltClient) Send(_msg string) error {
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

func (c *qltClient) _Send(_msg string) error {
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

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func qltClientInit(addresses string, cnx int, qltcQueue chan QLTMessage) {
	time.Sleep(10 * time.Millisecond)
	addrs := strings.Split(addresses, ",")
	clients := make([]*qltClient, 0)
	for i := 0; i < cnx; i++ {
		for _, addr := range addrs {
			log.Println("[QLTC] Connecting to ", addr, "...")
			client, err := qltClientCreate(addr)
			if err != nil {
				log.Errorln("[QLTC] failed to connect", addr, "...")
				panic(err)
			}
			defer client.Close()
			clients = append(clients, client)
		}
	}
	l := uint32(len(clients))
	count := 1
	for {
		log.Println("[QLTC] Waiting Message on QLTCQueue...", count)
		event := <-qltcQueue
		log.Println("[QLTC] Converting Message to xml...", count)
		msg := event.XMLOriginal
		n := hash(event.Fields["cycleid"]) % l
		log.Println("[QLTC] Sending Message to remote...", count, addrs[n], msg)
		if err := clients[n].Send(msg); err != nil {
			panic(err)
		}
		qltMessageOut.Inc()
		qltMessageOutSize.Observe(float64(len(msg)))
		log.Println("[QLTC] Sent", count, msg)
		count++
	}
}
