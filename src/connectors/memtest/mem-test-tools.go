package memtest

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"testing"

	"axway.com/qlt-router/src/connectors/mem"
	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
)

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

func RandStringBytesMaskImpr(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func randI(min, max int) int {
	n := min
	if max-min >= 1 {
		n = min + int(rand.Int31n(int32(max-min)))
	}
	return n
}

func MessageGenerator(minReaders, maxReaders, minMessages, maxMessages, minMsgSize, maxMsgSize int) ([][]string, int) {
	// rand.Seed(time.Now().UnixNano())
	n_readers := randI(minReaders, maxReaders)
	all_count := 0
	readers := make([][]string, n_readers)
	rall := RandStringBytesMaskImpr(maxMsgSize)

	for j := 0; j < n_readers; j++ {
		n := randI(minMessages, maxMessages)

		msgs := make([]string, n)
		for i := 0; i < n; i++ {
			r := RandStringBytesMaskImpr(maxMsgSize)
			size := randI(minMsgSize, maxMsgSize)
			msgs[i] = "{\"m\": \"" + fmt.Sprint("msg", "-", j, "-", i, "-", rall[:4], "-", r)[:size] + "\"}"
			all_count++
		}
		readers[j] = msgs
	}
	return readers, all_count
}

func MemMessageCheck(t *testing.T, readers [][]string, ackPos []int64, wMessages []string) {
	// Ensure that the right number of messages arrived
	count := 0
	for j := 0; j < len(readers); j++ {
		count += len(readers[j])
		//if memWriter.datas[i] != msgs[i] {
		//	t.Error("messages mismatch", memWriter.datas[i], msgs[i])
		//}
	}

	if count != len(wMessages) {
		t.Error("Wrong number of messages received", "mem_reader_count", count, "mem_writer_count", len(wMessages), wMessages)
		return
	}

	// Verify that all acks have been received
	for j := 0; j < len(readers); j++ {
		if ackPos[j] != int64(len(readers[j])-1) {
			t.Error("messages ack mismatch", j, ackPos[j], len(readers[j])-1)
		}
		k := 0
		for i := 0; i < len(readers[j]); i++ {
			msg := readers[j][i]

			for k < len(wMessages) && msg != wMessages[k] {
				t.Logf("msg %d %d %s %s", j, i, msg, wMessages[k])
				k++
			}
			if len(wMessages) <= k || msg != wMessages[k] {
				t.Error("missing message", msg)
				return
			}
		}
	}
}

// Test Connector generic test
func TestConnector(t *testing.T, writer, reader processor.Connector) {
	log.Infoc("testconnector", "Start")
	msgs, all_count := MessageGenerator(1, 1, 5, 5, 20, 20)

	ctl := make(chan processor.ControlEvent, 100)
	channels := processor.NewChannels()

	rp := processor.NewProcessor("mem-reader", &mem.MemReadersConf{msgs}, channels)
	c1 := channels.Create("writerStream", -1)

	connectorWriter := processor.NewProcessor("tested-writer", writer, channels)
	connectorReader := processor.NewProcessor("tested-reader", reader, channels)
	c2 := channels.Create("readerStream", -1)
	wp := processor.NewProcessor("mem-writer", &mem.MemWriterConf{-1}, channels)

	processors := []*processor.Processor{connectorReader, wp, rp, connectorWriter}
	c := []*processor.Channel{nil, c2, nil, c1, nil}
	rProcessors := []processor.ConnectorRuntime{}

	log.Infoc("testconnector", "Starting connectors...")
	errorCount := 0
	var wg sync.WaitGroup
	for i, p2 := range processors {
		wg.Add(1)
		go func(i int, p2 *processor.Processor) {
			defer wg.Done()
			p, err := p2.Start(context.Background(), ctl, c[i], c[i+1])
			if err != nil {
				t.Error("Error starting connector'"+p.Ctx()+"'", err)
				errorCount++
			}
			rProcessors = append(rProcessors, p)
		}(i, p2)
	}
	wg.Wait()
	if errorCount > 0 {
		return
	}
	log.Infoc("test", "All connectors started")
	cond1 := false
	cond2 := false
	for !cond1 || !cond2 {
		op := <-ctl
		op.Log()
		if op.From.Name == "mem-reader" && op.Id == "ACK_ALL_DONE" && rp.Out_ack >= int64(all_count) {
			cond1 = true
			t.Logf("op %+v", op.From)
		}
		if op.From.Name == connectorReader.Name && op.Id == "ACK_ALL_DONE" && connectorReader.Out_ack >= int64(all_count) {
			cond2 = true
			t.Logf("op %+v", op.From)
		}
		if op.Id == "ERROR" {
			t.Error("Error", op.Id, op.From.Name, op.Msg)
			return
		}

	}

	memWriter := wp.Runtime.(*mem.MemWriter)

	ackPos := make([]int64, len(rp.Runtimes)-1)
	for j := 0; j < len(rp.Runtimes)-1; j++ {
		memReaderSource := rp.Runtimes[j].(*mem.MemReadersSource)
		ackPos[j] = memReaderSource.AckPos
	}
	MemMessageCheck(t, msgs, ackPos, memWriter.Messages)

	/*for _, p := range rProcessors {
		p.Close()
	}*/
}
