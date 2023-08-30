package tools_test

import (
	"fmt"
	"os"
	"testing"

	"axway.com/qlt-router/src/tools"
	"github.com/esimov/gogu"
)

func CleanFiles(ctx, filenamePrefix string, filenameSuffix string) error {
	entries, err := tools.FileSwitchList(ctx, filenamePrefix, filenameSuffix)
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

func TestFileSwitch(t *testing.T) {
	t.Parallel()

	prefix := "/tmp/testSwitch"
	suffix := "txt"
	ctx := "test"
	data := "testdata"
	n := 4

	err := CleanFiles(ctx, prefix, suffix)
	if err != nil {
		return
	}

	// real test
	for i := 0; i < n+3; i++ {
		entries, err := tools.FileSwitchList(ctx, prefix, suffix)
		if err != nil {
			t.Error("FileSwitchList error", err)
			return
		}
		expectedLength := gogu.Min(i, n)

		if len(entries) != expectedLength {
			t.Error(fmt.Sprint("It ", i) + " expecting number of file " + fmt.Sprint(expectedLength, " got ", len(entries), " : ", entries))
			return
		}

		fullfilename, err := tools.FileSwitch(ctx, prefix, suffix, n)
		if err != nil {
			t.Error("unexpected file switch", err)
			return
		}

		err = os.WriteFile(fullfilename, []byte(data), 0o666)
		if err != nil {
			t.Error("unexpected write", err)
			return
		}
	}

	err = CleanFiles(ctx, prefix, suffix)
	if err != nil {
		return
	}

}
