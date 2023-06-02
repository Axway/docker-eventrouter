package file_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"axway.com/qlt-router/src/connectors/file"
	mem "axway.com/qlt-router/src/connectors/mem"
	"axway.com/qlt-router/src/connectors/memtest"
	"axway.com/qlt-router/src/processor"
)

func TestFileStoreRawWriterStart(t *testing.T) {
	targetFilename := "/tmp/write"
	os.Remove(targetFilename) // Ignore error

	msgs, all_count := memtest.MessageGenerator(2, 10, 50, 100, 10, 100)

	ctl := make(chan processor.ControlEvent, 100)

	channels := processor.NewChannels()
	cIn := channels.Create("file-store-in", -1)
	// cIn := make(chan processor.AckableEvent, 1000)
	wp := processor.NewProcessor("file-store", &file.FileStoreRawWriterConfig{targetFilename, 0, 0}, channels)
	rp := processor.NewProcessor("mem-reader", &mem.MemReadersConf{msgs}, channels)

	wp.Start(context.Background(), ctl, cIn, nil)
	rp.Start(context.Background(), ctl, nil, cIn)
	defer wp.Close()
	defer rp.Close()

	var err error
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
	if err != nil {
		t.Error("Oups: " + err.Error())
		return
	}

	file, err := os.Open(targetFilename)
	if err != nil {
		t.Error(err)
		return
	}

	b, err := io.ReadAll(file)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(string(b))
	wMessages := strings.Split(string(b), "\n")

	ackPos := make([]int64, len(rp.Runtimes)-1)
	for j := 0; j < len(rp.Runtimes)-1; j++ {
		memReaderSource := rp.Runtimes[j].(*mem.MemReadersSource)
		ackPos[j] = memReaderSource.AckPos
	}
	memtest.MemMessageCheck(t, msgs, ackPos, wMessages)

	// t.Error("===Success===")
}
