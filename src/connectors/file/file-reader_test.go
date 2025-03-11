package file_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"axway.com/qlt-router/src/connectors/file"
	"axway.com/qlt-router/src/connectors/memtest"
	"axway.com/qlt-router/src/processor"
	"axway.com/qlt-router/src/tools"
)

func CleanFiles(ctx, filenamePrefix string, filenameSuffix string) error {
	entries, err := tools.FileSwitchList(ctx, filenamePrefix, filenameSuffix, false)
	if err != nil {
		fmt.Println("FileSwitchList error ")
		return err
	}

	for _, entry := range entries {
		if entry[:len(filenamePrefix)] != filenamePrefix {
			return err
		}
		err := os.Remove(entry)
		if err != nil {
			return err
		}
	}
	return nil
}

func Test1FileStoreRawReaderStart(t *testing.T) {
	ctx := "test1"
	readerFilename := "/tmp/zoup-reader"
	targetFilenamePref := "/tmp/zoup"

	os.Remove(readerFilename)
	err := CleanFiles(ctx, targetFilenamePref, "")
	if err != nil {
		return
	}

	msgs, all_count := memtest.MessageGenerator(1, 1, 50, 100, 10, 100)
	msgs2, all_count2 := memtest.MessageGenerator(1, 1, 10, 20, 10, 100)

	input := strings.Join(msgs[0], "\n")
	input2 := strings.Join(msgs2[0], "\n")

	filename := tools.TimestampedFilename(ctx, targetFilenamePref, "")
	err = os.WriteFile(filename, []byte(input+"\n"), 0o666)
	if err != nil {
		t.Fatal(err)
	}
	filename = tools.TimestampedFilename(ctx, targetFilenamePref, "")
	err = os.WriteFile(filename, []byte(input2+"\n"), 0o666)
	if err != nil {
		t.Fatal(err)
	}

	channels := processor.NewChannels()
	ctl := make(chan processor.ControlEvent, 100)

	conf := file.FileStoreRawReaderConfig{targetFilenamePref, "", 113, readerFilename}
	out := channels.Create("file-reader-out", -1)
	p := processor.NewProcessor("file-reader", &conf, channels)

	p.Start(context.Background(), ctl, nil, out)
	defer p.Close()

	for i := 0; i < all_count; i++ {
		select {
		case msg := <-out.C:
			if msg.Msg != msgs[0][i] {
				t.Error("bad message received (1st for-loop):", i, msg.Msg, msgs[0][i])
				return
			}
			msg.Src.AckMsg(msg.Msgid)
		case <-time.After(2000 * time.Millisecond):
			t.Error("reached timeout", all_count, i)
		}
	}
	for i := 0; i < all_count2; i++ {
		select {
		case msg := <-out.C:
			if msg.Msg != msgs2[0][i] {
				t.Error("bad message received (2nd for-loop):", i, msg.Msg, msgs2[0][i])
				return
			}
			msg.Src.AckMsg(msg.Msgid)
		case <-time.After(2000 * time.Millisecond):
			t.Error("reached timeout", all_count, i)
		}
	}

	select {
	case <-out.C:
		t.Error("unexpected message")
	default:
		// ok
		defer CleanFiles(ctx, targetFilenamePref, "")
	}
	// t.Error("==Success==")
}

func Test2FileStoreRawReaderStart(t *testing.T) {
	ctx := "test1"
	readerFilename := "/tmp/zoup2-reader"
	targetFilenamePref := "/tmp/zoup"

	os.Remove(readerFilename)
	err := CleanFiles(ctx, targetFilenamePref, "")
	if err != nil {
		return
	}
	os.Create(readerFilename)

	msgs, all_count := memtest.MessageGenerator(1, 1, 50, 100, 10, 100)

	input := strings.Join(msgs[0], "\n")

	filename := tools.TimestampedFilename(ctx, targetFilenamePref, "")
	err = os.WriteFile(filename, []byte(input+"\n"), 0o666)
	if err != nil {
		t.Fatal(err)
	}

	channels := processor.NewChannels()
	ctl := make(chan processor.ControlEvent, 100)

	conf := file.FileStoreRawReaderConfig{targetFilenamePref, "", 113, readerFilename}
	out := channels.Create("file-reader-out", -1)
	p := processor.NewProcessor("file-reader", &conf, channels)

	p.Start(context.Background(), ctl, nil, out)
	defer p.Close()

	for i := 0; i < all_count; i++ {
		select {
		case msg := <-out.C:
			if msg.Msg != msgs[0][i] {
				t.Error("bad message received (1st for-loop):", i, msg.Msg, msgs[0][i])
				return
			}
			msg.Src.AckMsg(msg.Msgid)
		case <-time.After(2000 * time.Millisecond):
			t.Error("reached timeout", all_count, i)
		}
	}

	select {
	case <-out.C:
		t.Error("unexpected message")
	default:
		// ok
		defer CleanFiles(ctx, targetFilenamePref, "")
	}
	// t.Error("==Success==")
}
