package mem

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"axway.com/qlt-router/src/processor"
)

func TestMemWriter(t *testing.T) {
	n := 10
	msgs := make([]string, n)
	for i := 0; i < n; i++ {
		msgs[i] = fmt.Sprint("msg", i)
	}

	channels := processor.NewChannels()
	cIn := channels.Create("mem-writer-in", -1)
	// ch := make(chan processor.AckableEvent, 10)
	ctl := make(chan processor.ControlEvent, 10)

	w := MemWriterConf{}
	r := MemReaderConf{msgs}

	wp := processor.NewProcessor("mem-writer", &w, channels)
	rp := processor.NewProcessor("mem-reader", &r, channels)

	go wp.Start(context.Background(), ctl, cIn, nil)
	go rp.Start(context.Background(), ctl, nil, cIn)

	for {
		op := <-ctl
		op.Log()
		if op.From == rp && op.Id == "STOPPED" {
			break
		}
	}
	for {
		op := <-ctl
		op.Log()
		if op.From == rp && op.Id == "ACK_DONE" {
			break
		}
	}

	memWriter := wp.Runtime.(*MemWriter)
	memReader := rp.Runtime.(*MemReader)
	for i := 0; i < len(msgs); i++ {
		if memWriter.Messages[i] != msgs[i] {
			t.Error("messages mismatch", memWriter.Messages[i], msgs[i])
		}
	}
	if memReader.AckPos != int64(len(msgs)-1) {
		t.Error("messages ack mismatch", memReader.AckPos, len(msgs)-1)
		return
	}

	// t.Error("== SUCCESS ==")
}

func TestMemWriters(t *testing.T) {
	// msgs := []string{"msg1", "msg2", "msg3"}

	n_readers := 5 + int(rand.Int31n(20))
	all_count := 0
	readers := make([][]string, n_readers)
	for j := 0; j < n_readers; j++ {
		n := 10 + int(rand.Int31n(100))
		msgs := make([]string, n)
		for i := 0; i < n; i++ {
			msgs[i] = fmt.Sprint("msg", "-", j, "-", i)
			all_count++
		}
		readers[j] = msgs
	}

	channels := processor.NewChannels()
	cIn := channels.Create("mem-writer-in", -1)
	// ch := make(chan processor.AckableEvent, 10)
	ctl := make(chan processor.ControlEvent, 10)

	w := MemWriterConf{}
	r := MemReadersConf{readers}

	wp := processor.NewProcessor("mem-writer", &w, channels)
	rp := processor.NewProcessor("mem-reader", &r, channels)

	wp.Start(context.Background(), ctl, cIn, nil)
	rp.Start(context.Background(), ctl, nil, cIn)
	defer wp.Close()
	defer rp.Close()

	for {
		op := <-ctl
		op.Log()
		if op.From == rp && op.Id == "STOPPED" {
			break
		}
	}
	for {
		op := <-ctl
		op.Log()
		if op.From == rp && op.Id == "ACK_ALL_DONE" && rp.Out_ack == int64(all_count) {
			break
		}
	}

	memWriter := wp.Runtime.(*MemWriter)
	// memReaders := rp.Runtime.(*MemReaders)

	// Ensure that the right number of messages arived
	count := 0
	for j := 0; j < len(readers); j++ {
		count += len(readers[j])
		//if memWriter.datas[i] != msgs[i] {
		//	t.Error("messages mismatch", memWriter.datas[i], msgs[i])
		//}
	}
	if count != len(memWriter.Messages) {
		t.Error("Wrong number of messages received", count, len(memWriter.Messages), memWriter.Messages)
	}

	// Verify that all acks have been received
	for j := 0; j < len(readers); j++ {
		memReaderSource := rp.Runtimes[j].(*MemReadersSource)
		if memReaderSource.AckPos != int64(len(readers[j])-1) {
			t.Error("messages ack mismatch", memReaderSource.AckPos, len(readers[j])-1)
		}
		k := 0
		for i := 0; i < len(readers[j]); i++ {
			msg := readers[j][i]
			for k < len(memWriter.Messages) && msg != memWriter.Messages[k] {
				k++
			}
			if msg != memWriter.Messages[k] {
				t.Error("missing message", msg)
			}
		}
	}

	// t.Error("== SUCCESS ==")
}
