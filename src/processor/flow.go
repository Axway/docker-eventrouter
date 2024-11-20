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
	Reader      *FlowStep   `yaml:"reader"`
	Transforms  []*FlowStep `yaml:"transforms"`
	Writer      *FlowStep   `yaml:"writer"`
}

type FlowStep struct {
	Type string `yaml:"type"`
	// ScaleUnordered int       `yaml:"scaleUnordered"`
	Scale int       `yaml:"scale"`
	Conf  Connector `yaml:"conf,omitempty"`
}

func (f *FlowStep) UnmarshalYAML(n *yaml.Node) error {
	type Obj struct {
		Type  string    `yaml:"type"`
		Scale int       `yaml:"scale"`
		Conf  yaml.Node `yaml:"conf"`
	}
	obj := &Obj{}

	if err := tools.YamlParseVerify("connector", obj, n); err != nil {
		return err
	}
	// Avoid pring paswords
	/*ctx := "Flowstep"
	log.Debugc(ctx, "unmarshal", "obj", fmt.Sprintf("%+v", *obj))*/

	p := RegisteredProcessors.Get(obj.Type)
	if p == nil {
		names := gogu.Map(RegisteredProcessors, func(p *Processor) string { return p.Name })
		s := strings.Join(names, ",")
		// log.Debug("yaml: unknown processor '" + obj.Name + "' " + s)
		return errors.New("yaml: unknown processor '" + obj.Type + "' " + s)
	}
	// log.Debug("Unmarshall yaml: FlowStep default", "name", obj.Name, "v", fmt.Sprintf("%+v", p.Conf), "v2", fmt.Sprintf("%+v", obj.Conf.Content[0]))
	f.Conf = p.Conf.Clone()
	f.Type = obj.Type
	f.Scale = obj.Scale

	err := tools.YamlParseVerify(obj.Type, f.Conf, &obj.Conf)
	// err := obj.Conf.Decode(f.Conf)

	// Avoid pring paswords
	/*log.Debugc(ctx, "Unmarshal yaml: values", "name", obj.Type, "v", fmt.Sprintf("%+v", f.Conf))*/
	return err
}

func (flow *Flow) Start(ctx context.Context, readerContext context.Context, instance_id string, all bool, ctl chan ControlEvent, channels *Channels, processors *Processors) ([]*Processor, error) {
	ctxS := "stream"
	if flow.Disable && !all {
		log.Warnc(ctxS, "disabled", "name", flow.Name)
		return nil, nil
	}
	log.Infoc(ctxS, "starting...", "name", flow.Name)
	var in *Channel
	var out *Channel
	flowtxt := ""

	steps := []*FlowStep{}
	steps = append(steps, flow.Reader)
	steps = append(steps, flow.Transforms...)
	steps = append(steps, flow.Writer)

	var runtimeProcessor []*Processor
	for idx, step := range steps {
		channelName := flow.Name + "-" + fmt.Sprint(idx)
		writer := false
		reader := false
		if idx == len(steps)-1 {
			out = nil
			writer = true
		} else {
			if idx == 0 {
				channelName = flow.Name + "-reader"
				reader = true
			} else if idx == len(steps)-2 {
				channelName = flow.Name + "-writer"
			}
			out = channels.Create(channelName, flowChannelSize)
		}
		flowtxt += step.Type
		connector := step.Conf
		p := NewProcessor(step.Type, connector, channels)
		p.Instance_id = instance_id

		if p == nil { // FIXME: cannot be nil : already checked when loading configuration
			log.Errorc(ctxS, "processor not found", "name", flow.Name+"/"+step.Type)
			closest := (*processors)[0].Name
			closest_d := tools.Levenshtein(step.Type, closest)
			for _, p := range *processors {
				d := tools.Levenshtein(step.Type, p.Name)
				if d < closest_d {
					closest = p.Name
				}
			}
			log.Errorc(ctxS, "processor not found", "name", flow.Name+"/"+step.Type, "closest", closest)
			return nil, fmt.Errorf("Processor " + flow.Name + "/" + step.Type + " not found, maybe " + closest)
		} else {
			ctx2 := ctx
			if reader {
				ctx2 = readerContext
				log.Debugc(ctxS, "reader", "type", step.Type)
			}

			p2 := *p // FIXME: really? Clone Processor
			p = &p2
			p.Flow = flow
			p.FlowStep = step

			// For subprocessor
			p.Context = ctx2
			p.Ctl = ctl
			p.Cin = in
			p.Cout = out

			if reader || writer {
				position := "reader"
				if writer {
					position = "writer"
				}
				upstream := flow.Upstream
				if upstream == "" {
					upstream = "none"
				}
				p.OutCounter = promauto.NewCounter(prometheus.CounterOpts{
					Name: "er_messages_total",
					Help: "The total number messages processed",
					ConstLabels: prometheus.Labels{
						"position":       position,
						"type":           step.Type,
						"stream":         flow.Name,
						"upstream":       upstream,
						"er_instance_id": p.Instance_id,
					},
				})
				p.OutDataCounter = promauto.NewCounter(prometheus.CounterOpts{
					Name: "er_messages_bytes_total",
					Help: "The total volume of messages processed in bytes",
					ConstLabels: prometheus.Labels{
						"position":       position,
						"type":           step.Type,
						"stream":         flow.Name,
						"upstream":       upstream,
						"er_instance_id": p.Instance_id,
					},
				})
				p.OutAckCounter = promauto.NewCounter(prometheus.CounterOpts{
					Name: "er_ack_messages_total",
					Help: "The total number messages acked",
					ConstLabels: prometheus.Labels{
						"position":       position,
						"type":           step.Type,
						"stream":         flow.Name,
						"upstream":       upstream,
						"er_instance_id": p.Instance_id,
					},
				})
			}

			runtimeProcessor = append(runtimeProcessor, p)

			// Avoid pring paswords
			log.Infoc(ctxS, "info", "name", flow.Name, "processorName", p.Name) //, "conf", fmt.Sprintf("%+v", p.Conf))
			if step.Scale > 0 {
				ParallelOrdered(ctx, channelName+"-scale", step.Scale, ctl, in.C, out.C, channels, p)
				/*} else if step.ScaleUnordered > 0 {
				ParallelUnordered(ctx, channelName+"-scale", step.ScaleUnordered, ctl, in.C, out.C, channels, p)
				*/
			} else {
				r, err := p.Conf.Start(ctx2, p, ctl, in.GetC(), out.GetC())
				if err != nil {
					log.Fatalc(ctxS, "failed to start", "name", flow.Name+"/"+step.Type, "err", err)
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
	log.Infoc(ctxS, "started", "name", flow.Name, "desc", flowtxt)
	return runtimeProcessor, nil
}
