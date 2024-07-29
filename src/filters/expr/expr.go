package expr

import (
	"context"
	"encoding/json"
	"strings"

	"axway.com/qlt-router/src/filters/qlt2json"
	"axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/processor"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

func filter(ctxS string, program *vm.Program, event string) (any, error) {
	var eventMap map[string]map[string]string
	var err error

	if strings.HasPrefix(event, "<?xml") {
		eventMapInter, err := qlt2json.ConvertToMap(event)
		if err != nil {
			log.Errorc(ctxS, "Converting to map failed...", "error", err)
		}
		eventMap = make(map[string]map[string]string)
		eventMap["msg"] = eventMapInter
	} else {
		err = json.Unmarshal([]byte(`{"msg": `+event+`}`), &eventMap)
		if err != nil {
			log.Errorc(ctxS, "Converting to map failed...", "error", err)
		}
	}
	if err != nil {
		return nil, err
	}

	output, err := expr.Run(program, eventMap)
	if err != nil {
		log.Errorc(ctxS, "Running expression failed...", "error", err)
	}
	return output, err
}

type ExprConf struct {
	Expression string
}

func (conf *ExprConf) Start(ctx context.Context, p *processor.Processor, ctl chan processor.ControlEvent, eventsIn, eventOut chan processor.AckableEvent) (processor.ConnectorRuntime, error) {
	ctxS := "[ExpressionFilter]"
	log.Infoc(ctxS, "start")

	env := map[string]map[string]string{}

	options := []expr.Option{
		expr.Env(env),
		expr.AllowUndefinedVariables(), // Allow the use of undefined variables.
		expr.AsBool(),
	}

	log.Tracec(ctxS, "Filtering messages", "Expression", conf.Expression)
	program, err := expr.Compile(conf.Expression, options...)
	if err != nil {
		log.Errorc(ctxS, "Expression compiling failed... ", "error", err)
		return nil, nil
	}

	done := ctx.Done()
	go func() {
		for {
			select {
			case event := <-eventsIn:
				output, err := filter(ctxS, program, event.Msg.(string))
				if err != nil {
					eventOut <- processor.AckableEvent{Src: event.Src, Msgid: event.Msgid, Msg: event.Msg.(string), Orig: &event}
					continue
				}
				if output == true {
					log.Tracec(ctxS, "Keeping message", "Msg", event.Msg.(string))
					eventOut <- processor.AckableEvent{Src: event.Src, Msgid: event.Msgid, Msg: event.Msg.(string), Orig: &event}
				} else {
					log.Tracec(ctxS, "Message filtered", "Msg", event.Msg.(string))
					eventOut <- processor.AckableEvent{Src: event.Src, Msgid: event.Msgid, Msg: nil, Orig: &event}
				}
			case <-done:
				log.Infoc(ctxS, "done")
				return
			}
		}
	}()
	return nil, nil
}

func (c *ExprConf) Clone() processor.Connector {
	c2 := *c
	return &c2
}
