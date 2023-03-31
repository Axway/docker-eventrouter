package file

import (
	"encoding/binary"
	"os"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type BufferMessage struct {
	response int
	data     []byte
}

type ProcessI interface {
	processIn(s *Stream, inData []BufferMessage)
	processOut(s *Stream)
}

type Stream struct {
	inData       chan []BufferMessage
	inResponses  []chan bool
	outData      chan []BufferMessage
	outResponses []chan bool

	data []BufferMessage
}

func (s *Stream) run(p ProcessI) {
	for {
		if len(s.data) == 0 {
			msg := <-s.inData
			p.processIn(s, msg)
		} else if len(s.data) > 10000 {
			s.outData <- s.data
			p.processOut(s)
		} else {
			select {
			case msg := <-s.inData:
				p.processIn(s, msg)
			case s.outData <- s.data:
				p.processOut(s)
			}
		}
	}
}

type BufferProcess struct{}

func (p *BufferProcess) processIn(s *Stream, inData []BufferMessage) {
	s.data = append(s.data, inData...)
}

func (p *BufferProcess) processOut(s *Stream) {
	s.data = make([]BufferMessage, 0)
}

type fileWriteProcess struct {
	folder string
	f      *os.File
}

func bufferFileProcessCreate(folder string) *fileWriteProcess {
	var p fileWriteProcess

	filename := folder + "/dat-" + time.Now().Format(time.RFC3339)
	f, err := os.Create(filename)
	check(err)
	p.f = f
	return &p
}

func (p *fileWriteProcess) processIn(s *Stream, inData []BufferMessage) {
	var buf []byte
	for _, item := range inData {
		a := make([]byte, 4)
		binary.LittleEndian.PutUint32(a, uint32(len(item.data)))
		buf = append(buf, a[:]...)
		buf = append(buf, item.data[:]...)
	}
	_, err := p.f.Write(buf)
	check(err)

	err = p.f.Sync()
	check(err)

	s.data = append(s.data, inData...)
}

func (p *fileWriteProcess) processOut(s *Stream) {
	s.data = make([]BufferMessage, 0)
}

type fileReadProcess struct {
	folder string
	f      *os.File
}

func bufferFileReadNew(folder string) *fileReadProcess {
	var p fileReadProcess

	filename, err := readDirNextInt(folder, "dat-", "")
	check(err)
	f, err := os.Open(filename)
	check(err)
	p.f = f
	return &p
}

func (p *fileReadProcess) processIn(s *Stream, inData []BufferMessage) {
}

func (p *fileReadProcess) processOut(s *Stream) {
	s.data = make([]BufferMessage, 0)
}
