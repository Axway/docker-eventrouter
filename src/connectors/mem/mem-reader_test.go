package mem

import (
	"context"
	"fmt"
	"testing"
	"time"

	"axway.com/qlt-router/src/processor"
)

func testMemReader(n int) {
	msgs := make([]string, n)
	for i := 0; i < n; i++ {
		msgs[i] = fmt.Sprint("msg", i)
	}
	channels := processor.NewChannels()
	cIn := channels.Create("mem-reader-out", -1)
	// cIn := make(chan processor.AckableEvent, 1000)
	ctl := make(chan processor.ControlEvent, 10)

	cInc := MemReaderConf{msgs}
	pIn := processor.NewProcessor("test-reader", &cInc, channels)

	go processor.ControlEventLogAll(context.Background(), ctl)
	pIn.Start(context.Background(), ctl, nil, cIn)
	defer pIn.Close()

	for i := 0; i < len(msgs); i++ {
		msg := <-cIn.C
		if msg.Msg != msgs[i] {
			panic("Message mismatch " + msg.Msg.(string) + " " + msgs[i])
		}
	}
}

func TestMemReader(t *testing.T) {
	n := 10
	msgs := make([]string, n)
	for i := 0; i < n; i++ {
		msgs[i] = fmt.Sprint("msg", i)
	}
	channels := processor.NewChannels()
	cIn := channels.Create("mem-reader-out", -1)
	// cIn := make(chan processor.AckableEvent, 1000)
	ctl := make(chan processor.ControlEvent, 10)

	cInc := MemReaderConf{msgs}
	pIn := processor.NewProcessor("test-reader", &cInc, channels)

	go processor.ControlEventLogAll(context.Background(), ctl)

	pIn.Start(context.Background(), ctl, nil, cIn)
	defer pIn.Close()

	for i := 0; i < len(msgs); i++ {
		select {
		case msg := <-cIn.C:
			if msg.Msg != msgs[i] {
				t.Error("Message mismatch", msg, msgs[i])
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("timeout reached", i, len(msgs))
		}
	}
	// t.Error("== SUCCESS ==")
}

/*
func BenchmarkMemReader(b *testing.B) {
	locallog.InitLogSetLevelWarn()
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		testMemReader(1000000)
	}
}
*/
