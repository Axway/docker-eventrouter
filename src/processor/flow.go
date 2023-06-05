package processor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	log "axway.com/qlt-router/src/log"
	"axway.com/qlt-router/src/tools"
	"github.com/esimov/gogu"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gopkg.in/yaml.v3"
)

type Flow struct {
	Name        string      `yaml:"name"`
	Upstream    string      `yaml:"upstream"`
	Description string      `yaml:"description"`
	Disable     bool        `yaml:"disable"`
	Flow        []*FlowStep `yaml:"flow"`
}

type FlowStep struct {
	Name           string    `yaml:"name"`
	ScaleUnordered int       `yaml:"scaleUnordered"`
	ScaleOrdered   int       `yaml:"scaleOrdered"`
	Conf           Connector `yaml:"conf,omitempty"`
}

func (f *FlowStep) UnmarshalYAML(n *yaml.Node) error {
	type Obj struct {
		Name           string    `yaml:"name"`
		ScaleUnordered int       `yaml:"scaleUnordered"`
		ScaleOrdered   int       `yaml:"scaleOrdered"`
		Conf           yaml.Node `yaml:"conf"`
	}
	obj := &Obj{}

	if err := tools.YamlParseVerify("connector", obj, n); err != nil {
		return err
	}
	ctx := "Flowstep"
	log.Debugc(ctx, "unmarshal", "obj", fmt.Sprintf("%+v", *obj))

	p := RegisteredProcessors.Get(obj.Name)
	if p == nil {
		names := gogu.Map(RegisteredProcessors, func(p *Processor) string { return p.Name })
		s := strings.Join(names, ",")
		// log.Debug("yaml: unknown processor '" + obj.Name + "' " + s)
		return errors.New("yaml: unknown processor '" + obj.Name + "' " + s)
	}
	// log.Debug("Unmarshall yaml: FlowStep default", "name", obj.Name, "v", fmt.Sprintf("%+v", p.Conf), "v2", fmt.Sprintf("%+v", obj.Conf.Content[0]))
	f.Conf = p.Conf.Clone()
	f.Name = obj.Name
	f.ScaleOrdered = obj.ScaleOrdered
	f.ScaleUnordered = obj.ScaleUnordered

	err := tools.YamlParseVerify(obj.Name, f.Conf, &obj.Conf)
	// err := obj.Conf.Decode(f.Conf)

	log.Debugc(ctx, "Unmarshal yaml: values", "name", obj.Name, "v", fmt.Sprintf("%+v", f.Conf))
	return err
}

func (flow *Flow) Start(ctx context.Context, all bool, ctl chan ControlEvent, channels *Channels, processors *Processors) ([]*Processor, error) {
	ctxS := "flow"
	if flow.Disable && !all {
		log.Warnc(ctxS, "flow disabled", "name", flow.Name)
		return nil, nil
	}
	log.Infoc(ctxS, "flow starting...", "name", flow.Name)
	var in *Channel
	var out *Channel
	flowtxt := ""

	var runtimeProcessor []*Processor
	for idx, step := range flow.Flow {
		channelName := flow.Name + "-" + fmt.Sprint(idx)
		// consumer := false
		producer := false
		if idx == len(flow.Flow)-1 {
			out = nil
			// consumer = true
		} else {
			if idx == 0 {
				channelName = flow.Name + "-producer"
				producer = true
			} else if idx == len(flow.Flow)-2 {
				channelName = flow.Name + "-consumer"
			}
			out = channels.Create(channelName, flowChannelSize)
		}
		flowtxt += fmt.Sprint(step.Name)
		connector := step.Conf
		p := NewProcessor(step.Name, connector, channels)

		if p == nil { // FIXME: cannot be nil : already checked when loading configuration
			log.Errorc(ctxS, "Processor not found", "name", flow.Name+"/"+step.Name)
			closest := (*processors)[0].Name
			closest_d := tools.Levenshtein(step.Name, closest)
			for _, p := range *processors {
				d := tools.Levenshtein(step.Name, p.Name)
				if d < closest_d {
					closest = p.Name
				}
			}
			log.Errorc(ctxS, "Processor not found", "name", flow.Name+"/"+step.Name, "closest", closest)
			return nil, fmt.Errorf("Processor " + flow.Name + "/" + step.Name + " not found, maybe " + closest)
		} else {
			p2 := *p // FIXME: really? Clone Processor
			p = &p2
			p.Flow = flow
			p.FlowStep = step
			p.Context = ctx // FIXME: quesako
			p.Ctl = ctl     // FIXME: quesako
			p.Cin = in      // FIXME: quesako
			p.Cout = out    // FIXME: quesako

			// p.FlowStep = &step //FIXME: ???????
			runtimeProcessor = append(runtimeProcessor, p)
			if producer {
				p.OutCounter = promauto.NewCounter(prometheus.CounterOpts{
					Name:        "qlt_in_message_total",
					Help:        "The total number of qlt messages for",
					ConstLabels: prometheus.Labels{"producer": step.Name, "flow": flow.Name},
				})
			}
			log.Infoc(ctxS, "flow", "name", flow.Name, "processorName", p.Name, "conf", fmt.Sprintf("%+v", p.Conf))
			if step.ScaleOrdered > 0 {
				ParallelOrdered(ctx, channelName+"-scale", step.ScaleOrdered, ctl, in.C, out.C, channels, p)
			} else if step.ScaleUnordered > 0 {
				ParallelUnordered(ctx, channelName+"-scale", step.ScaleUnordered, ctl, in.C, out.C, channels, p)
			} else {
				r, err := p.Conf.Start(ctx, p, ctl, in.GetC(), out.GetC())
				if err != nil {
					log.Fatalc(ctxS, "flow failed to start", "name", flow.Name+"/"+step.Name, "err", err)
					os.Exit(1)
				}
				p.Runtime = r
			}
		}
		if out != nil {
			flowtxt += fmt.Sprint(" -[", channelName, "]-> ")
		}
		in = out
	}
	log.Infoc(ctxS, "flow", "name", flow.Name, "desc", flowtxt)
	return runtimeProcessor, nil
}
