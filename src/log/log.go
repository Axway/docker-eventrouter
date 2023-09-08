package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/esimov/gogu"
)

var (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[90m"
	White  = "\033[97m"

	EffectBold      = "1"
	EffectUnderline = "4"
	EffectReset     = "21"
)

func colorEffect(color string, effect string) string {
	return color[:len(color)-1] + ";" + effect + "m"
}

func colorEffectReset(color string) string {
	return color[:len(color)-1] + ";22;24m"
}

const (
	TRC_LEVEL = 0
	DBG_LEVEL = 1
	INF_LEVEL = 2
	WRN_LEVEL = 3
	ERR_LEVEL = 4
	FTL_LEVEL = 5
)

var (
	TraceLevel = LogLevel{TRC_LEVEL, "TRC"}
	DebugLevel = LogLevel{DBG_LEVEL, "DBG"}
	InfoLevel  = LogLevel{INF_LEVEL, "INF"}
	WarnLevel  = LogLevel{WRN_LEVEL, "WRN"}
	ErrorLevel = LogLevel{ERR_LEVEL, "ERR"}
	FatalLevel = LogLevel{FTL_LEVEL, "FTL"}
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
}

var (
	root                   = Logger{}
	level                  = DebugLevel
	useLocalTime           = false
	output       io.Writer = nil
)

func ParseLevel(lvl string) (LogLevel, error) {
	switch strings.ToLower(lvl) {
	case "trace":
		return TraceLevel, nil
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	case "fatal":
		return FatalLevel, nil
	default:
		return TraceLevel, errors.New("invalid log level")
	}
}

func SetLevel(lvl LogLevel) {
	level = lvl
}

func SetOutput(w io.Writer) {
	output = w
}

func SetUseLocalTime(use bool) {
	useLocalTime = use
}

func Level(lvl LogLevel) bool {
	return (lvl.code < level.code)
}

var (
	sb       strings.Builder // FIXME: reuse
	logLock  sync.Mutex
	UseColor = false
)

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

func (l *Logger) _params(sb *strings.Builder, color string, kv ...interface{}) {
	b := colorEffect(color, EffectBold)
	ub := colorEffectReset(color)
	for i := 0; i < len(kv); i += 2 {
		sb.WriteString(" ")
		sb.WriteString(kv[i].(string))
		sb.WriteString("=")
		if UseColor {
			sb.WriteString(b)
		}
		writeValue(sb, kv[i+1])
		if UseColor {
			sb.WriteString(ub)
		}
	}
	if l.parent != nil {
		l.parent._params(sb, color, l.context...)
	}
}

func (l *Logger) _logw(lvl LogLevel, msg string, kv ...interface{}) {
	if len(kv)%2 != 0 {
		panic("wrong number of arguments")
	}
	if lvl.code < level.code {
		return
	}

	color := White
	if lvl.code == ERR_LEVEL {
		color = Red
	} else if lvl.code == DBG_LEVEL {
		color = Gray
	} else if lvl.code == WRN_LEVEL {
		color = Purple
	}

	b := colorEffect(color, EffectBold)
	// u := colorEffect(color, EffectUnderline)
	ub := colorEffectReset(color)

	logLock.Lock()
	sb.Reset()
	sb.Grow(1000)
	if UseColor {
		sb.WriteString(color)
	}
	if !l.SkipTime {
		// sb.WriteString(time.Now().UTC().Format(time.RFC3339Nano)) // 1 alloc

		if useLocalTime {
			sb.WriteString(time.Now().UTC().Format("2006-01-02_15:04:05.000000"))
		} else {
			sb.WriteString(time.Now().Format("2006-01-02T15:04:05.000000-07:00"))
		}
		sb.WriteString(" ")
	}
	sb.WriteString(lvl.str)
	sb.WriteString(" ")
	sb.WriteString("[")
	sb.WriteString(l.prefix)
	sb.WriteString("] ")
	if UseColor {
		sb.WriteString(b)
	}

	sb.WriteString(msg)
	if UseColor {
		sb.WriteString(ub)
	}

	sb.WriteString(" --")
	l._params(&sb, color, kv...)
	if UseColor {
		sb.WriteString(Reset)
	}
	sb.WriteString("\n")

	if output == nil {
		os.Stdout.Write([]byte(sb.String()))
	} else {
		output.Write([]byte(sb.String()))
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

func TraceW(ctx *Logger, msg string, kv ...interface{}) {
	ctx._logw(TraceLevel, msg, kv...)
}

func DebugW(ctx *Logger, msg string, kv ...interface{}) {
	ctx._logw(DebugLevel, msg, kv...)
}

func InfoW(ctx *Logger, msg string, kv ...interface{}) {
	ctx._logw(InfoLevel, msg, kv...)
}

func WarnW(ctx *Logger, msg string, kv ...interface{}) {
	ctx._logw(WarnLevel, msg, kv...)
}

func ErrorW(ctx *Logger, msg string, kv ...interface{}) {
	ctx._logw(ErrorLevel, msg, kv...)
}

func FatalW(ctx *Logger, msg string, kv ...interface{}) {
	ctx._logw(FatalLevel, msg, kv...)
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
	case bool:
		err = writeStringValue(w, fmt.Sprint(v), true)
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
		keys := gogu.Keys(v)
		sort.Strings(keys)
		for _, k := range keys {
			if !first {
				w.WriteByte(',')
			} else {
				first = false
			}
			writeValue(w, k)
			w.WriteString(":")
			writeValue(w, v[k])
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
