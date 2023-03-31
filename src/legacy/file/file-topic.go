package file

import (
	"bufio"
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
//    data/{topicName}/client.{clientId}
//       [INT32]length [BYTEs]filename...
//       [INT32]last-offset

// const dataFolder = "./data"
func fileWriteInt32(f io.Writer, l int) error {
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

func fileWriteBuffer32(f io.Writer, buf []byte) error {
	l := len(buf)
	tmp := make([]byte, 4+l)
	tmp[0] = byte((l >> 24) & 0xFF)
	tmp[1] = byte(l >> 16 & 0xFF)
	tmp[2] = byte(l >> 8 & 0xFF)
	tmp[3] = byte(l & 0xFF)

	copy(tmp[4:4+len(buf)], buf)
	log.Println("[FileTopic] Buffer write", l, string(tmp))
	if _, err := f.Write(tmp); err != nil {
		return err
	}

	return nil
}

func fileReadInt32(f io.Reader) (int, error) {
	tmp := make([]byte, 4)
	if n, err := f.Read(tmp); err != nil {
		return -1, err
	} else if n != 4 {
		return -1, errors.New(fmt.Sprint("File too short, incomplete int32, got ", n, " bytes instead of ", 4))
	}

	n := int(tmp[0])<<24 | int(tmp[1])<<16 | int(tmp[2])<<8 | int(tmp[3])

	return n, nil
}

func fileReadBuffer32(f io.Reader) ([]byte, error) {
	n, err := fileReadInt32(f)
	if err != nil {
		return nil, err
	}

	data := make([]byte, n)
	if datan, err := f.Read(data); err != nil {
		return nil, err
	} else if datan != n {
		return nil, errors.New(fmt.Sprint("File too short, incomplete buffer, got ", datan, " bytes instead of ", n))
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

	return flist, nil
}

func readDirNextInt(dirname string, prefix string, current string) (string, error) {
	list, err := ReadDir(dirname, prefix)
	if err != nil {
		return "", err
	}
	log.Println("[FileTopic] List ", list)
	if current == "" {
		if len(list) > 0 {
			log.Println("[FileTopic] Readir return first", list[0].Name())
			return list[0].Name(), nil
		}
		log.Println("[FileTopic] Readir no data (none)")
		return "", nil // No available data
	}
	for i := 0; i < len(list); i++ {
		// log.Println("[FileTopic] Compare ", dirname, prefix, current, list[i].Name())
		if list[i].Name() == current {
			if i+1 >= len(list) {
				log.Println("[FileTopic] Readir no data (after last)")
				return "", nil // No available data
			}
			log.Println("[FileTopic] Readir (next)", list[i+1].Name())
			return list[i+1].Name(), nil
		}
	}
	log.Errorln("[FileTopic] Readir not found")
	return "", errors.New("Not Found")
}

// ReadDirNext get next available data amongs file
func ReadDirNext(dirname string, prefix string, current string) ([]byte, error) {
	s, err := readDirNextInt(dirname, prefix, current)
	return []byte(s), err
}

func fileTopicWriteInit(dataFolder, topic string, FileTopicQueue chan []byte, FileTopicAcks chan int, done chan interface{}) {
	topicFolder := dataFolder + "/" + topic + "/"
	topicFilename := "data." + fmt.Sprintf("%010x", time.Now().Unix())
	topicPath := topicFolder + topicFilename
	f, err := os.OpenFile(topicPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Errorln("[FileTopic] Failed to open data file...", topicPath, err)
		log.Fatal(err)
	}
	defer f.Close()
	bf := bufio.NewWriter(f)

	var event []byte
	for {
		log.Println("[FileTopic] Waiting Message...")

		select {
		case event = <-FileTopicQueue:
		case <-done:
			break
		}
		log.Println("[FileTopic] Message for fs", len(event), string(event))

		if err := fileWriteBuffer32(bf, event); err != nil {
			log.Fatal(err)
		}

		if err := bf.Flush(); err != nil {
			log.Fatal(err)
		}

		log.Println("[FileTopic] Message for fs written", len(event), string(event))
		FileTopicAcks <- 1
		log.Println("[FileTopic] Message acked")
	}
}

func fileOffsetWrite(offsetFile *os.File, offsetFilePath string, dataFilename string, dataOffset int) {
	_, err := offsetFile.Seek(int64(0), 0)
	if err != nil {
		log.Println("[FileTopic] Cannot reset client file to offset 0", offsetFilePath, err)
		log.Fatal(err)
	}
	err = fileWriteBuffer32(offsetFile, []byte(dataFilename))
	if err != nil {
		log.Errorln("[FileTopic] Cannot write to client file (filename)", offsetFilePath, err)
		log.Fatal(err)
	}

	err = fileWriteInt32(offsetFile, dataOffset)
	if err != nil {
		log.Errorln("[FileTopic] Cannot write to client file offset (offset)", offsetFilePath, err)
		log.Fatal(err)
	}
}

func fileTopicReadInit(dataFolder, topic string, clientID string, FileTopicQueue chan []byte, done chan interface{}) {
	topicFolder := dataFolder + "/" + topic + "/"
	offsetFilePath := topicFolder + "client." + clientID
	offsetFile, err := os.OpenFile(offsetFilePath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		log.Println("[FileTopic] Cannot open client file ", offsetFilePath, err)
		log.Fatal(err)
	}
	dataOffset := 0
	dataFilename, err := fileReadBuffer32(offsetFile)
	if err != nil {
		if err == io.EOF {
			log.Println("[FileTopic] client file is empty ", offsetFilePath)
			dataFilename, err = ReadDirNext(topicFolder, "data.", "")
			// dataFilenames, err := ReadDir(topicFolder, "data.")
			if err != nil {
				log.Println("[FileTopic] Cannot read topic folder ", topicFolder, err)
				log.Fatal(err)
			}
			// dataFilename = []byte(dataFilenames[0].Name())
			dataOffset = 0
			log.Println("[FileTopic] using datafile ", string(dataFilename))

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

loop:
	for {
		dataFilepath := topicFolder + string(dataFilename)
		dataFile, err := os.OpenFile(dataFilepath, os.O_RDONLY, 0o644)
		if err != nil {
			log.Println("[FileTopic] Cannot open data file ", dataFilepath, err)
			log.Fatal(err)
		}
		if _, err := dataFile.Seek(int64(dataOffset), 0); err != nil {
			log.Println("[FileTopic] Cannot move to offset ", dataFilepath, dataOffset, err)
			log.Fatal(err)
		}
		dataFileB := bufio.NewReader(dataFile)

		for {
			buf, err := fileReadBuffer32(dataFileB)
			if err == io.EOF {
				dataFilename, err = ReadDirNext(topicFolder, "data.", string(dataFilename))
				if err != nil {
					log.Errorln("[FileTopic] Cannot read topic folder ", topicFolder, err)
					log.Fatal(err)
				}
				dataOffset = 0
				log.Println("[FileTopic] using datafile ", string(dataFilename))
				if len(dataFilename) == 0 {
					close(FileTopicQueue)
					break loop
				}
				break
			} else if err != nil {
				log.Errorln("[FileTopic] Cannot read next data ", dataFilepath, err)
				log.Fatal(err)
			}

			log.Println("[FileTopic]", dataFilepath, "DATA", len(buf), string(buf))
			select {
			case FileTopicQueue <- buf:
			case <-done:
				close(FileTopicQueue)
				break loop
			}
			dataOffset = dataOffset + len(buf) + 4
			fileOffsetWrite(offsetFile, offsetFilePath, string(dataFilename), dataOffset)
			/*err = offsetFile.Sync()
			if err != nil {
				log.Errorln("[FileTopic] Cannot sync client file offset (offset)", offsetFilePath, err)
				log.Fatal(err)
			}*/

		}
	}
}
