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
	"axway.com/qlt-router/src/tools"
	mem "axway.com/qlt-router/src/connectors/mem"
	"axway.com/qlt-router/src/connectors/memtest"
	"axway.com/qlt-router/src/processor"
)

func Test1FileStoreRawWriterStart(t *testing.T) {
	targetFilename := "/tmp/write_test1"
	targetFilenameSuf := "zouzou"
	ctx := "test1"

	err := CleanFiles(ctx, targetFilename, targetFilenameSuf)
	if err != nil {
		return
	}

	msgs, all_count := memtest.MessageGenerator(2, 10, 50, 100, 10, 100)

	ctl := make(chan processor.ControlEvent, 100)

	channels := processor.NewChannels()
	cIn := channels.Create("file-store-in", -1)
	// cIn := make(chan processor.AckableEvent, 1000)
	wp := processor.NewProcessor("file-store", &file.FileStoreRawWriterConfig{targetFilename, targetFilenameSuf, 0, 0}, channels)
	rp := processor.NewProcessor("mem-reader", &mem.MemReadersConf{msgs}, channels)

	wp.Start(context.Background(), ctl, cIn, nil)
	rp.Start(context.Background(), ctl, nil, cIn)
	defer wp.Close()
	defer rp.Close()

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

	entries, err := tools.FileSwitchList(ctx, targetFilename, targetFilenameSuf)
	if err != nil {
		t.Error(err)
		return
	}
	if len(entries) == 0 {
		t.Error("Empty list of files")
		return
	}

	file, err := os.Open(entries[0])
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
	// remove empty lines from end of file
	if len(wMessages[len(wMessages)-1]) == 0 {
		wMessages = wMessages[:len(wMessages)-1]
	}

	ackPos := make([]int64, len(rp.Runtimes)-1)
	for j := 0; j < len(rp.Runtimes)-1; j++ {
		memReaderSource := rp.Runtimes[j].(*mem.MemReadersSource)
		ackPos[j] = memReaderSource.AckPos
	}
	memtest.MemMessageCheck(t, msgs, ackPos, wMessages)

	err = CleanFiles(ctx, targetFilename, targetFilenameSuf)
	if err != nil {
		return
	}
}

func Test2FileStoreMultipleFilesStart(t *testing.T) {
	targetFilename := "/tmp/write_test2"
	ctx := "test2"

	err := CleanFiles(ctx, targetFilename, "")
	if err != nil {
		return
	}

	msgs, all_count := memtest.MessageGenerator(1, 1, 1200, 1500, 800, 1000)

	ctl := make(chan processor.ControlEvent, 100)

	channels := processor.NewChannels()
	cIn := channels.Create("file-store-in2", -1)
	// cIn := make(chan processor.AckableEvent, 1000)
	wp := processor.NewProcessor("file-store", &file.FileStoreRawWriterConfig{targetFilename, "", 0, 1}, channels)
	rp := processor.NewProcessor("mem-reader", &mem.MemReadersConf{msgs}, channels)

	wp.Start(context.Background(), ctl, cIn, nil)
	rp.Start(context.Background(), ctl, nil, cIn)
	defer wp.Close()
	defer rp.Close()

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

	entries, err := tools.FileSwitchList(ctx, targetFilename, "")
	if err != nil {
		t.Error(err)
		return
	}
	if len(entries) == 0 {
		t.Error("Empty list of files")
		return
	}
	if len(entries) == 1 {
		t.Error("All messages in same file")
		return
	}

	// Read each file and count that number of messages is respected
	var wMessages []string
	for i := 0; i < len(entries); i++ {
		file, err := os.Open(entries[i])
		if err != nil {
			t.Error(err)
			return
		}

		b, err := io.ReadAll(file)
		if err != nil {
			t.Error(err)
			return
		}

		lines := strings.Split(string(b), "\n")
		// remove empty lines from end of file
		if len(lines[len(lines)-1]) == 0 {
			lines = lines[:len(lines)-1]
		}
		// concatenate all messages in one buffer
		wMessages = append(wMessages, lines...)
	}

	// Test that all messages are present
	ackPos := make([]int64, len(rp.Runtimes)-1)
	for j := 0; j < len(rp.Runtimes)-1; j++ {
		memReaderSource := rp.Runtimes[j].(*mem.MemReadersSource)
		ackPos[j] = memReaderSource.AckPos
	}
	memtest.MemMessageCheck(t, msgs, ackPos, wMessages)

	err = CleanFiles(ctx, targetFilename, "")
	if err != nil {
		return
	}
}
