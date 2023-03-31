package memtest

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

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
	rand.Seed(time.Now().UnixNano())
	n_readers := randI(minReaders, maxReaders)
	all_count := 0
	readers := make([][]string, n_readers)
	r := RandStringBytesMaskImpr(maxMsgSize)

	for j := 0; j < n_readers; j++ {
		n := randI(minMessages, maxMessages)

		msgs := make([]string, n)
		for i := 0; i < n; i++ {
			size := randI(minMsgSize, maxMsgSize)
			msgs[i] = fmt.Sprint("msg", "-", j, "-", i, "-", r)[:size]
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
				k++
			}
			if msg != wMessages[k] {
				t.Error("missing message", msg)
			}
		}
	}
}
