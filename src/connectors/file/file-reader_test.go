package file_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"axway.com/qlt-router/src/connectors/file"
	"axway.com/qlt-router/src/connectors/memtest"
	"axway.com/qlt-router/src/processor"
)

func TestFileStoreRawReaderStart(t *testing.T) {
	msgs, all_count := memtest.MessageGenerator(1, 1, 50, 100, 10, 100)

	input := strings.Join(msgs[0], "\n")

	targetFilename := "/tmp/zoup"
	err := os.WriteFile(targetFilename, []byte(input+"\n"), 0o666)
	if err != nil {
		t.Fatal(err)
	}
	channels := processor.NewChannels()
	// defer os.Remove(targetFilename)
	ctl := make(chan processor.ControlEvent, 100)

	conf := file.FileStoreRawReaderConfig{targetFilename, 113}
	out := channels.Create("file-reader-out", -1)
	p := processor.NewProcessor("file-reader", &conf, channels)

	p.Start(context.Background(), ctl, nil, out)
	defer p.Close()

	for i := 0; i < all_count; i++ {
		select {
		case msg := <-out.C:
			if msg.Msg != msgs[0][i] {
				t.Error("bad message received:", i, msg.Msg, msgs[0][i])
				return
			}
			msg.Src.AckMsg(msg.Msgid)
		case <-time.After(1000 * time.Millisecond):
			t.Error("reached timeout", all_count, i)
		}
	}

	select {
	case <-out.C:
		t.Error("unexpected message")
	default:
		// ok
	}
	// t.Error("==Success==")
}
