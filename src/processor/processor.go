package processor

import (
	"context"
	"fmt"

	"axway.com/qlt-router/src/locallog"
	"github.com/prometheus/client_golang/prometheus"

	log "github.com/sirupsen/logrus"
)

func init() {
	// log.SetLevel(log.DebugLevel)
	locallog.InitLog()
}

type EventAck interface{}

type AckableEvent struct {
	Src   EventSource
	Msgid EventAck
	Msg   interface{}
	Orig  *AckableEvent
}

type EventSource interface {
	AckMsg(msgid EventAck)
	Ctx() string
}

type Processor struct {
	Name     string
	Conf     Connector
	Chans    *Channels
	Runtime  ConnectorRuntime
	Runtimes []ConnectorRuntime

	Flow     *Flow
	FlowStep *FlowStep
	// In       int64
	Out     int64
	Out_ack int64
	Context context.Context `json:"-"`

	OutCounter prometheus.Counter `json:"-"`

	Cin  *Channel
	Cout *Channel
	Ctl  chan ControlEvent `json:"-"`
}

/*func (p *Processor) Stop() {
	done := p.Context.Done()
	close(done)
}*/

func NewProcessor(name string, conf Connector, channels *Channels) *Processor {
	var p Processor
	p.Name = name
	p.Conf = conf
	p.Chans = channels

	return &p
}

func (p *Processor) AddReader(reader Connector) (ConnectorRuntime, error) {
	log.Debugln(p.Name, "Starting AddReader!!!!*************************")
	runtime, err := reader.Start(p.Context, p, p.Ctl, p.Cin.GetC(), p.Cout.GetC())
	if err != nil {
		p.Ctl <- ControlEvent{p, runtime, "ERROR", fmt.Sprint("connector start error: ", "err ", err)}
	}
	// log.Debugln(p.Name, "Started AddReader!!!!*************************")
	p.Runtimes = append(p.Runtimes, runtime)
	return runtime, err
}

func (p *Processor) Start(ctx context.Context, ctl chan ControlEvent, cin *Channel, cout *Channel) (ConnectorRuntime, error) {
	p.Ctl = ctl
	p.Cin = cin
	p.Cout = cout
	p.Context = ctx
	runtime, err := p.AddReader(p.Conf)
	p.Runtime = runtime
	return runtime, err
}

func (p *Processor) Close() error {
	if p.Runtime == nil {
		log.Debugln(p.Name, "processor closing: empty runtime")
		return nil
	}
	log.Debugln(p.Name, "processor closing", "runtime", p.Runtime.Ctx())
	return p.Runtime.Close()
}

/*
func ParseConfig(q interface{}, prefix string) {
	if reflect.ValueOf(q).Elem().Kind() == reflect.Struct {
		t := reflect.TypeOf(q).Elem()
		// typename := reflect.TypeOf(q).Name()
		// log.Debugln("name", typename)
		v := reflect.ValueOf(q).Elem()

		for i := 0; i < v.NumField(); i++ {
			name := t.Field(i).Name
			paramName := prefix + "_" + strings.ToLower(name)
			switch v.Field(i).Kind() {
			case reflect.Int:
				log.Println(paramName, "Int")
				flag.IntVar((*int)(unsafe.Pointer(v.Field(i).Addr().Pointer())), paramName, 0, "")

			case reflect.String:
				log.Println(paramName, "String")
				flag.StringVar((*string)(unsafe.Pointer(v.Field(i).Addr().Pointer())), paramName, "", "")

			case reflect.Bool:
				log.Println(paramName, "Bool")
				flag.BoolVar((*bool)(unsafe.Pointer(v.Field(i).Addr().Pointer())), paramName, false, "")
			case reflect.Slice:
			log.Warnln(paramName, "Slice")
			// flag.BoolVar((*bool)(unsafe.Pointer(v.Field(i).Addr().Pointer())), paramName, false, "")
			default:
				log.Fatalln("Unsupported type", v.Field(i).Kind().String())
				return
			}
		}
	} else {
		log.Fatal("unsupported type", reflect.ValueOf(q).Kind())
	}
}
*/

type Processors []*Processor

func (processors Processors) Get(name string) *Processor {
	for _, p := range processors {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func (processors *Processors) Register(name string, conf Connector) *Processor {
	p := NewProcessor(name, conf, nil)
	*processors = append(*processors, p)
	// ParseConfig(conf, name)
	return p
}

func (processors Processors) All() []*Processor {
	return processors[:]
}

// FIXME: required for config parsing, but awful !!!!
var RegisteredProcessors Processors
