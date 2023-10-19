package qlt

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"axway.com/qlt-router/src/config"
	"axway.com/qlt-router/src/connectors/mem"
	"axway.com/qlt-router/src/connectors/memtest"
	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
)

func TestQltConnectorPush(t *testing.T) {
	porti, _ := memtest.GetFreePort()
	port := fmt.Sprint(porti)

	writer := &QLTClientWriterConf{"localhost:" + port, "", "", "", 1}
	reader := &QLTServerReaderConf{"localhost", port, "", "", ""}
	memtest.TestConnector(t, writer, reader)
}

func TestQltConnectorPull(t *testing.T) {
	porti, _ := memtest.GetFreePort()
	port := fmt.Sprint(porti)

	writer := &QLTServerWriterConf{"Q1", "localhost", port, "", "", ""}
	reader := &QLTClientReaderConf{"Q1", "localhost:" + port, "", "", "", 1}
	memtest.TestConnector(t, writer, reader)
}

func testQltConnector(port string, disableQlt bool, minReaders, maxReaders, minMessages, maxMessages, minMsgSize, maxMsgSize int) ([][]string, *mem.MemWriter, *processor.Processor, string, error) {
	// msgs := []string{"msg1", "msg2", "msg3"}
	// port := "9999"
	// minReaders := 1
	// maxReaders := 1
	// minMessages := 50000
	// maxMessages := 50000
	// var coroutines []int
	// defer log.Println("test: defer coroutine", coroutines)
	if port == "" {
		porti, _ := memtest.GetFreePort()
		port = fmt.Sprint(porti)
	}
	readers, all_count := memtest.MessageGenerator(minReaders, maxReaders, minMessages, maxMessages, minMsgSize, maxMsgSize)

	ctl := make(chan processor.ControlEvent, 1000)

	var ch1 *processor.Channel
	var rp *processor.Processor
	var wp *processor.Processor

	channels := processor.NewChannels()

	// Consumer
	{
		// ch1 = make(chan processor.AckableEvent, 10)
		ch1 = channels.Create("reader", -1)
		qltServerConf := QLTServerReaderConf{"localhost", port, "", "", ""}
		w := mem.MemWriterConf{-1}
		qs := processor.NewProcessor("qlt-server-reader", &qltServerConf, channels)
		wp = processor.NewProcessor("mem-writer", &w, channels)

		wp.Start(context.Background(), ctl, ch1, nil)
		if !disableQlt {
			qs.Start(context.Background(), ctl, nil, ch1)
		}
		defer qs.Close()
		defer wp.Close()
	}

	// Producer
	{
		// ch2 := make(chan processor.AckableEvent, 10)
		ch2 := channels.Create("writer", -1)
		r := mem.MemReadersConf{readers}
		qltClientConf := QLTClientWriterConf{"localhost:" + port, "", "", "", 1}
		rp = processor.NewProcessor("mem-reader", &r, channels)
		qc := processor.NewProcessor("qlt-client-writer", &qltClientConf, channels)
		if !disableQlt {
			qc.Start(context.Background(), ctl, ch2, nil)
			rp.Start(context.Background(), ctl, nil, ch2)
		} else {
			rp.Start(context.Background(), ctl, nil, ch1)
		}
		defer rp.Close()
		defer qc.Close()
	}

	// coroutines = append(coroutines, runtime.NumGoroutine())
	// pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)

	var err error
	//FIXME: How to ensure all the message are properly sent!
	/*for {
		op := <-ctl
		op.Log()
		if op.From == rp && op.Id == "STOPPED" {
			break
		}
		if op.Id == "ERROR" {
			err = errors.New("connector Error: " + op.Msg)
			break
		}
	}*/

	// coroutines = append(coroutines, runtime.NumGoroutine())
	// pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)

	for err == nil {
		op := <-ctl
		op.Log()
		if op.From == rp && op.Id == "ACK_ALL_DONE" && rp.Out_ack == int64(all_count) {
			break
		}
		if op.Id == "ERROR" {
			err = errors.New("connector Error: " + op.Msg)
			break
		}
	}
	// coroutines = append(coroutines, runtime.NumGoroutine())
	// log.Println("test: coroutine", coroutines)

	memWriter := wp.Runtime.(*mem.MemWriter)
	// memReaders := rp.Runtime.(*MemReaders)
	return readers, memWriter, rp, port, err
}

func TestQltConnectorSimple(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	config.Print()

	readers, memWriter, rp, _, err := testQltConnector("", false, 1, 1, 100000, 100000, 100, 100)
	if err != nil {
		t.Error("error running the test" + err.Error())
		return
	}
	ackPos := make([]int64, len(readers))
	for j := 0; j < len(readers); j++ {
		memReaderSource := rp.Runtimes[j].(*mem.MemReadersSource)
		ackPos[j] = memReaderSource.AckPos
	}

	memtest.MemMessageCheck(t, readers, ackPos, memWriter.Messages)

	config.Print()
	// t.Error("===Success===")
}

func TestQltConnectorSimpleNoQlt(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	_, _, _, _, err := testQltConnector("", true, 1, 1, 10, 10, 100, 100)
	if err != nil {
		t.Error("error running the test:" + err.Error())
		return
	}
}

func TestQltConnector2Run(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	_, _, _, port, err := testQltConnector("", false, 1, 1, 10, 10, 100, 100)
	if err != nil {
		t.Error("error running the test:" + err.Error())
		return
	}
	_, _, _, _, err = testQltConnector(port, false, 1, 1, 10, 10, 100, 100)
	if err != nil {
		t.Error("error running the test2: " + err.Error())
		return
	}
}

const (
	benchSize = 100000
	msgSize1  = 100
	msgSize2  = 1000
	msgSize3  = 10000
	portBase  = 7000
)

func benchmarkQltConnector(b *testing.B, f int, msgSize int, qltDisable bool) {
	// log.SetLevel(log.WarnLevel)
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		_, _, _, _, err := testQltConnector(fmt.Sprint(portBase+n), qltDisable, f, f, benchSize/f, benchSize/f, msgSize, msgSize)
		if err != nil {
			panic("error running te test: " + err.Error())
		}
	}
}

func BenchmarkQltConnect1_size1(b *testing.B) {
	benchmarkQltConnector(b, 1, msgSize1, false)
}

/*func BenchmarkQltConnect2_size1(b *testing.B) {
	benchmarkQltConnector(b, 2, msgSize1, false)
}

func BenchmarkQltConnect4_size1(b *testing.B) {
	benchmarkQltConnector(b, 4, msgSize1, false)
}*/

func BenchmarkQltConnect1B_size1(b *testing.B) {
	benchmarkQltConnector(b, 1, msgSize1, true)
}

/*func BenchmarkQltConnect2B_size1(b *testing.B) {
	benchmarkQltConnector(b, 2, msgSize1, true)
}

func BenchmarkQltConnect4B_size1(b *testing.B) {
	benchmarkQltConnector(b, 4, msgSize1, true)
}*/

func BenchmarkQltConnect1_size2(b *testing.B) {
	benchmarkQltConnector(b, 1, msgSize2, false)
}

/*func BenchmarkQltConnect2_size2(b *testing.B) {
	benchmarkQltConnector(b, 2, msgSize2, false)
}

func BenchmarkQltConnect4_size2(b *testing.B) {
	benchmarkQltConnector(b, 4, msgSize2, false)
}*/

func BenchmarkQltConnect1B_size2(b *testing.B) {
	benchmarkQltConnector(b, 1, msgSize2, true)
}

/*func BenchmarkQltConnect2B_size2(b *testing.B) {
	benchmarkQltConnector(b, 2, msgSize2, true)
}

func BenchmarkQltConnect4B_size2(b *testing.B) {
	benchmarkQltConnector(b, 4, msgSize2, true)
}*/

func BenchmarkQltConnect1_size3(b *testing.B) {
	benchmarkQltConnector(b, 1, msgSize3, false)
}

/*func BenchmarkQltConnect2_size3(b *testing.B) {
	benchmarkQltConnector(b, 2, msgSize3, false)
}

func BenchmarkQltConnect4_size3(b *testing.B) {
	benchmarkQltConnector(b, 4, msgSize3, false)
}*/

func BenchmarkQltConnect1B_size3(b *testing.B) {
	benchmarkQltConnector(b, 1, msgSize3, true)
}

/*func BenchmarkQltConnect2B_size3(b *testing.B) {
	benchmarkQltConnector(b, 2, msgSize3, true)
}

func BenchmarkQltConnect4B_size3(b *testing.B) {
	benchmarkQltConnector(b, 4, msgSize3, true)
}*/
