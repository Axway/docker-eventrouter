package tools_test

import (
	"fmt"
	"os"
	"testing"

	"axway.com/qlt-router/src/tools"
	"github.com/esimov/gogu"
)

func TestFileRotate(t *testing.T) {
	t.Parallel()

	filename := "/tmp/testRotate.txt"
	ctx := "test"
	data := "testdata"
	n := 3

	// cleanups
	os.Remove(filename) // Ignore

	entries, err := tools.FileRotateList(ctx, filename)
	if err != nil {
		t.Error("FileRotateList error", err)
		return
	}
	for _, entry := range entries {
		if entry[:len(filename)] != filename {
			t.Error("oups")
			return
		}
		err := os.Remove(entry)
		if err != nil {
			t.Error("delete error", err)
			return
		}
	}

	// realtest
	for i := 0; i < n+3; i++ {
		entries, err := tools.FileRotateList(ctx, filename)
		if err != nil {
			t.Error("FileRotateList error", err)
			return
		}
		fmt.Println("List", i, entries)
		expectedLength := gogu.Min(i, n)

		if len(entries) != expectedLength {
			t.Error("expecting number of file " + fmt.Sprint(expectedLength, " got ", len(entries), " : ", entries))
			return
		}

		err = os.WriteFile(filename, []byte(data), 0o666)
		if err != nil {
			t.Error("unexpected write", err)
			return
		}
		err = tools.FileRotate(ctx, filename, n)
		if err != nil {
			t.Error("unexpected file rotate", err)
			return
		}
	}
}
