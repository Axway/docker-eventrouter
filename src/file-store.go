package main

import (
	"encoding/json"
	"os"

	log "github.com/sirupsen/logrus"
)

func fileStoreInit(filename string, FileStoreQueue chan QLTMessage) {
	log.Println("[FS] Opening file", filename, "...")
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Errorln("[FS] Error opening file for appending", filename, err)
		log.Fatal(err)
	}
	defer f.Close()

	for {
		log.Println("[FS] Waiting MessageMessage on FSQueue...")
		event := <-FileStoreQueue
		log.Println("[FS] Marshalling Message")
		buf, _ := json.Marshal(event)
		log.Println("[FS] Message", string(buf))
		if _, err := f.Write([]byte(string(buf) + "\n")); err != nil {
			log.Errorln("[FS] Error write message in file", filename, err)
			log.Fatal(err)
		}
	}
}
