package log

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var (
	TraceLevel = LogLevel{1, "TRC"}
	DebugLevel = LogLevel{2, "DBG"}
	InfoLevel  = LogLevel{3, "INF"}
	WarnLevel  = LogLevel{4, "WRN"}
	ErrorLevel = LogLevel{5, "ERR"}
	FatalLevel = LogLevel{6, "FTL"}
)

type LogLevel struct {
	code int
	str  string
}

var TraceLevelFull = []LogLevel{
	TraceLevel,
	DebugLevel,
	InfoLevel,
	WarnLevel,
	ErrorLevel,
	FatalLevel,
}

type Logger struct {
	context  []interface{}
	parent   *Logger
	prefix   string
	SkipTime bool // for testing purposes
	w        io.Writer
}

var (
	root  = Logger{}
	level = TraceLevel
)

func SetLevel(lvl LogLevel) {
	level = lvl
}

var sb strings.Builder // FIXME: reuse

func (l *Logger) New(prefix string) *Logger {
	l2 := Logger{}
	l2.parent = l
	l2.prefix = l.prefix + prefix
	l2.SkipTime = l.SkipTime
	return &l2
}

func (l *Logger) Add(k string, v interface{}) {
	l.context = append(l.context, k, v)
}

func (l *Logger) _params(sb *strings.Builder, kv ...interface{}) {
	for i := 0; i < len(kv); i += 2 {
		sb.WriteString(" ")
		sb.WriteString(kv[i].(string))
		sb.WriteString("=")
		writeValue(sb, kv[i+1])
	}
	if l.parent != nil {
		l.parent._params(sb, l.context...)
	}
}

var logLock sync.Mutex

func (l *Logger) _logw(lvl LogLevel, msg string, kv ...interface{}) {
	if len(kv)%2 != 0 {
		panic("wrong number of arguments")
	}
	if lvl.code < level.code {
		return
	}
	logLock.Lock()
	sb.Reset()
	sb.Grow(1000)
	if !l.SkipTime {
		sb.WriteString(time.Now().Format(time.RFC3339)) // 1 alloc
		sb.WriteString(" ")
	}
	sb.WriteString(lvl.str)
	sb.WriteString(" ")
	sb.WriteString(l.prefix)
	sb.WriteString(msg)
	l._params(&sb, kv...)
	sb.WriteString("\n")
	if l.w != nil {
		// l.w.Write([]byte(sb.String()))
	} else {
		os.Stdout.WriteString(sb.String())
	}
	logLock.Unlock()
}

func (l *Logger) Trace(msg string, kv ...interface{}) {
	l._logw(TraceLevel, msg, kv...)
}

func (l *Logger) Debug(msg string, kv ...interface{}) {
	l._logw(DebugLevel, msg, kv...)
}

func (l *Logger) Info(msg string, kv ...interface{}) {
	l._logw(InfoLevel, msg, kv...)
}

func (l *Logger) Warn(msg string, kv ...interface{}) {
	l._logw(WarnLevel, msg, kv...)
}

func (l *Logger) Error(msg string, kv ...interface{}) {
	l._logw(ErrorLevel, msg, kv...)
}

/*
	func Trace(msg string, kv ...interface{}) {
		root._logw(TraceLevel, msg, kv...)
	}

	func Debug(msg string, kv ...interface{}) {
		root._logw(DebugLevel, msg, kv...)
	}

	func Info(msg string, kv ...interface{}) {
		root._logw(InfoLevel, msg, kv...)
	}

	func Warn(msg string, kv ...interface{}) {
		root._logw(WarnLevel, msg, kv...)
	}

	func Error(msg string, kv ...interface{}) {
		root._logw(ErrorLevel, msg, kv...)
	}

	func Fatal(msg string, kv ...interface{}) {
		root._logw(FatalLevel, msg, kv...)
		os.Exit(1)
	}
*/
func Tracec(ctx, msg string, kv ...interface{}) {
	root._logw(TraceLevel, ctx+" "+msg, kv...)
}

func Debugc(ctx, msg string, kv ...interface{}) {
	root._logw(DebugLevel, ctx+" "+msg, kv...)
}

func Infoc(ctx, msg string, kv ...interface{}) {
	root._logw(InfoLevel, ctx+" "+msg, kv...)
}

func Warnc(ctx, msg string, kv ...interface{}) {
	root._logw(WarnLevel, ctx+" "+msg, kv...)
}

func Errorc(ctx, msg string, kv ...interface{}) {
	root._logw(ErrorLevel, ctx+" "+msg, kv...)
}

func Fatalc(ctx, msg string, kv ...interface{}) {
	root._logw(FatalLevel, ctx+" "+msg, kv...)
	os.Exit(1)
}

func needsQuotedValueRune(r rune) bool {
	return r <= ' ' || r == '=' || r == '"' || r == utf8.RuneError
}

var hex = "0123456789abcdef"

func writeValue(w *strings.Builder, val interface{}) error {
	var err error
	switch v := val.(type) {
	case string:
		err = writeStringValue(w, v, true)
	case int:
		err = writeStringValue(w, fmt.Sprintf("%d", v), true)
	case nil:
		err = writeStringValue(w, "null", true)
	case []string:
		w.WriteString("[")
		for i, item := range v {
			if i > 0 {
				w.WriteByte(',')
			}
			writeValue(w, item)
		}
		w.WriteString("]")
	case map[string]string:
		first := true
		w.WriteString("{")
		for k, item := range v {
			if !first {
				w.WriteByte(',')
			} else {
				first = false
			}
			writeValue(w, k)
			w.WriteString(":")
			writeValue(w, item)
		}
		w.WriteString("}")
	case error:
		w.WriteString("'")
		w.WriteString(v.Error())
		w.WriteString("'")
	default:
		/*rkey := reflect.ValueOf(key)
		switch rkey.Kind() {
		case reflect.Array, reflect.Chan, reflect.Func, reflect.Map, reflect.Slice, reflect.Struct:
			return ErrUnsupportedKeyType
		case reflect.Ptr:
			if rkey.IsNil() {
				return ErrNilKey
			}
			return writeKey(w, rkey.Elem().Interface())
		}*/
		//var b []byte
		//b, err = json.Marshal(val)
		err = writeStringValue(w, "@"+fmt.Sprint(val)+"@", true)
	}
	return err
}

func writeStringValue(w *strings.Builder, value string, ok bool) error {
	var err error
	/*if ok && value == "null" {
		_, err = io.WriteString(w, `"null"`)
	} else */
	if strings.IndexFunc(value, needsQuotedValueRune) != -1 {
		err = writeQuotedString(w, value)
	} else {
		_, err = io.WriteString(w, value)
	}
	return err
}

func writeQuotedString(w *strings.Builder, s string) error {
	w.WriteByte('\'')
	start := 0
	for i := 0; i < len(s); {
		b := s[i]
		if 0x20 <= b && b != '\\' /*&& b != '"'*/ {
			i++
			continue
		}
		if start < i {
			w.WriteString(s[start:i])
		}
		switch b {
		/*case '\\', '"':
		w.WriteByte('\\')
		w.WriteByte(b)*/
		case '\n':
			w.WriteByte('\\')
			w.WriteByte('n')
		case '\r':
			w.WriteByte('\\')
			w.WriteByte('r')
		case '\t':
			w.WriteByte('\\')
			w.WriteByte('t')
		default:
			// This encodes bytes < 0x20 except for \n, \r, and \t.
			w.WriteString(`\u00`)
			w.WriteByte(hex[b>>4])
			w.WriteByte(hex[b&0xF])
		}
		i++
		start = i
	}
	if start < len(s) {
		w.WriteString(s[start:])
	}
	w.WriteByte('\'')
	return nil
}
