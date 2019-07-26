package main

import (
	"time"

	client "github.com/elastic/go-lumber/client/v2"
	log "github.com/sirupsen/logrus"
)

func lumberJackInit(addr string, Queue chan map[string]string) {
	for {
		lumberJackSend(addr, Queue)
		log.Println("[LJ] Sleep...", 10)
		time.Sleep(10 * time.Second)
	}
}

func lumberJackSend(addr string, Queue chan map[string]string) error {
	log.Println("[LJ] Connecting to", addr, "...")
	client, err := client.Dial(addr)
	if err != nil {
		log.Errorln("[LJ] Error opening lumberjack connection to", addr, err)
		return err
	}

	for {
		log.Println("[LJ] Waiting Message on Queue...")
		event := <-Queue
		log.Println("[LJ] Got Message...")

		log.Println("[LJ] Sending Message...")
		count := 1
		messages := []interface{}{event}
		err = client.Send(messages)
		if err != nil {
			log.Errorln("[LJ] Error sending messages", addr, count, err)
			return err
		}

		log.Println("[LJ] Waiting Ack..")
		acked, err := client.AwaitACK(1)
		if err != nil {
			log.Errorln("[LJ] Error awaiting ack", addr, count, err)
			return err
		}
		log.Println("[LJ] Acked", acked)
	}
}
