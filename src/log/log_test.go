package log

import (
	"errors"
	"log"
	"os"
	"testing"
)

func init() {
	root.SkipTime = true
}

func ExampleLogger_Info() {
	root.Info("test sssszou", "test", "testvalue")
	// Output: INF [] test sssszou -- test=testvalue
}

func ExampleLogger_Info_sub() {
	l2 := root.New("zip:")
	l2.Add("ctx1", 9999)
	l2.Info("test msg", "k1", "testvalue")
	l2.Info("test msg2", "k2", "testvalue2")
	// Output:
	// INF [zip:] test msg -- k1=testvalue ctx1=9999
	// INF [zip:] test msg2 -- k2=testvalue2 ctx1=9999
}

func ExampleLogger_Info_string() {
	root.Info("test sssszou2", "test", "te st value\nzo\ru")
	// Output: INF [] test sssszou2 -- test='te st value\nzo\ru'
}

func ExampleLogger_Info_number() {
	root.Info("test sssszou2", "test", 3)
	// Output: INF [] test sssszou2 -- test=3
}

func ExampleLogger_Info_bool() {
	root.Info("test sssszou2", "test", true)
	// Output: INF [] test sssszou2 -- test=true
}

func ExampleLogger_Info_nil() {
	root.Info("test sssszou2", "test", nil)
	// Output: INF [] test sssszou2 -- test=null
}

func ExampleLogger_Info_array() {
	root.Info("test sssszou2", "test", []string{"t1", "t2"})
	// Output: INF [] test sssszou2 -- test=[t1,t2]
}

func ExampleLogger_Info_map() {
	root.Error("test sssszou2", "test", map[string]string{"k1": "v1", "k2": "v2"})
	// Output: ERR [] test sssszou2 -- test={k1:v1,k2:v2}
}

func ExampleLogger_Error() {
	root.Error("test sssszou2", "err", errors.New("test error"))
	// Output: ERR [] test sssszou2 -- err='test error'
}

func BenchmarkLog(b *testing.B) {
	old := os.Stdout
	null, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0o755)
	os.Stdout = null
	log.SetOutput(null)
	for i := 0; i < b.N; i++ {
		log.Println("test sssszou2",
			"key1", "val1",
			"key2", "val2",
			"key3", "val3",
			"key4", "val4",
			"key5", "val5",
			"key6", "val6",
			"key7", "val7",
			"key8", "val8",
			"key9", "val9",
			"key10", "val10",
		)
	}
	log.SetOutput(old)
	os.Stdout = old
}

func BenchmarkLogfmt(b *testing.B) {
	null, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0o755)
	SetOutput(null)
	for i := 0; i < b.N; i++ {
		Infoc("pr", "test sssszou2",
			"key1", "val1",
			"key2", "val2",
			"key3", "val3",
			"key4", "val4",
			"key5", "val5",
			"key6", "val6",
			"key7", "val7",
			"key8", "val8",
			"key9", "val9",
			"key10", "val10",
		)
	}
}
