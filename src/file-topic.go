package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Simple system for queuing leveraging underlying OS cache
// Format:
//    data/{topicName}/data.{stamp}
//       [INT32]length [BYTEs]data...
//       [INT32]length [BYTES]data...
//    data/{topicNlae}/client.{clientId}
//       [INT32]length [BYTEs]filename...
//       [INT32]last-offset

const dataFolder = "./data"

func fileWriteInt32(f *os.File, l int) error {
	tmp := make([]byte, 4)
	tmp[0] = byte((l >> 24) & 0xFF)
	tmp[1] = byte(l >> 16 & 0xFF)
	tmp[2] = byte(l >> 8 & 0xFF)
	tmp[3] = byte(l & 0xFF)
	if _, err := f.Write(tmp); err != nil {
		return err
	}
	return nil
}

func fileWriteBuffer32(f *os.File, buf []byte) error {
	l := len(buf)
	tmp := make([]byte, 4+l)
	tmp[0] = byte((l >> 24) & 0xFF)
	tmp[1] = byte(l >> 16 & 0xFF)
	tmp[2] = byte(l >> 8 & 0xFF)
	tmp[3] = byte(l & 0xFF)

	copy(tmp[4:4+len(buf)], buf)
	if _, err := f.Write(tmp); err != nil {
		return err
	}

	return nil
}

func fileReadInt32(f *os.File) (int, error) {
	tmp := make([]byte, 4)
	if n, err := f.Read(tmp); err != nil {
		return -1, err
	} else if n != 4 {
		return -1, errors.New("File too short, missing header")
	}

	n := int(tmp[0])<<24 | int(tmp[1])<<16 | int(tmp[2])<<8 | int(tmp[3])

	return n, nil
}

func fileReadBuffer32(f *os.File) ([]byte, error) {
	n, err := fileReadInt32(f)
	if err != nil {
		return nil, err
	}

	data := make([]byte, n)
	if datan, err := f.Read(data); err != nil {
		return nil, err
	} else if datan != n {
		return nil, errors.New("File too short, missing header")
	}
	return data, nil

}

// ReadDir reads the directory named by dirname and returns
// a list of directory entries sorted by filename.
func ReadDir(dirname string, prefix string) ([]os.FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })

	flist := make([]os.FileInfo, 0)
	for _, s := range list {
		if strings.HasPrefix(s.Name(), prefix) {
			flist = append(flist, s)
		}
	}

	return list, nil
}

func fileTopicWriteInit(topic string, FileTopicQueue chan []byte) {
	topicFolder := dataFolder + "/" + topic + "/"
	topicFilename := "data." + fmt.Sprintf("%010x", time.Now().Unix())
	topicPath := topicFolder + topicFilename
	f, err := os.OpenFile(topicPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Errorln("[FileTopic] Failed to open data file...", topicPath, err)
		log.Fatal(err)
	}
	defer f.Close()

	for {
		log.Println("[FileTopic] Waiting MessageMessage on FSQueue...")
		event := <-FileTopicQueue
		log.Println("[FileTopic] Message for fs", len(event), string(event))

		if err := fileWriteBuffer32(f, event); err != nil {
			log.Fatal(err)
		}
	}
}

func fileTopicReadInit(topic string, clientID string, FileTopicQueue chan []byte) {
	topicFolder := dataFolder + "/" + topic + "/"
	offsetFilePath := topicFolder + "client." + clientID
	offsetFile, err := os.OpenFile(offsetFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Println("[FileTopic] Cannot open client file ", offsetFilePath, err)
		log.Fatal(err)
	}

	dataOffset := 0
	dataFilename, err := fileReadBuffer32(offsetFile)
	if err != nil {
		if err == io.EOF {
			log.Println("[FileTopic] client file is empty ", offsetFilePath)
			dataFilenames, err := ReadDir(topicFolder, "data.")
			if err != nil {
				log.Println("[FileTopic] Cannot read topic folder ", topicFolder, err)
				log.Fatal(err)
			}
			dataFilename = []byte(dataFilenames[0].Name())
			dataOffset = 0
			log.Println("[FileTopic] using datafile ", dataFilename)

		} else {
			log.Println("[FileTopic] Cannot read last topic filename ", offsetFilePath, err)
			log.Fatal(err)
		}
	} else {
		dataOffset, err = fileReadInt32(offsetFile)
		if err != nil {
			log.Println("[FileTopic] Cannot read last offset ", offsetFilePath, err)
			log.Fatal(err)
		}
	}

	dataFilepath := topicFolder + string(dataFilename)
	dataFile, err := os.OpenFile(dataFilepath, os.O_RDONLY, 0644)
	if err != nil {
		log.Println("[FileTopic] Cannot open data file ", dataFilepath, err)
		log.Fatal(err)
	}
	if _, err := dataFile.Seek(int64(dataOffset), 0); err != nil {
		log.Println("[FileTopic] Cannot move to offset ", dataFilepath, dataOffset, err)
		log.Fatal(err)
	}

	for {
		buf, err := fileReadBuffer32(dataFile)
		if err != nil {
			log.Println("[FileTopic] Cannot read next data ", dataFilepath, err)
			log.Fatal(err)
		}

		FileTopicQueue <- buf

		if _, err := offsetFile.Seek(int64(0), 0); err != nil {
			log.Println("[FileTopic] Cannot reset client file to offset 0", offsetFilePath, err)
			log.Fatal(err)
		}
		err = fileWriteBuffer32(offsetFile, []byte(dataFilename))
		if err != nil {
			log.Println("[FileTopic] Cannot write to client file (filename)", offsetFilePath, err)
			log.Fatal(err)
		}

		err = fileWriteInt32(offsetFile, dataOffset+len(buf)+4)
		if err != nil {
			log.Println("[FileTopic] Cannot write to client file offset (offset)", offsetFilePath, err)
			log.Fatal(err)
		}
	}
}
