package processor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"axway.com/qlt-router/src/tools"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
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
	/*type S FlowStep
	type T struct {
		*S   `yaml:",inline"`
		Conf yaml.Node `yaml:"Conf"`
	}
	obj := &{S: (*S)(f)}*/

	type Obj struct {
		Name           string    `yaml:"name"`
		ScaleUnordered int       `yaml:"scaleUnordered"`
		ScaleOrdered   int       `yaml:"scaleOrdered"`
		Conf           yaml.Node `yaml:"conf"`
	}
	obj := &Obj{}

	if err := n.Decode(obj); err != nil {
		return err
	}

	// log.Debugf("FlowStep: Unmarshall %+v %+v %+v", *obj, *f)

	p := RegisteredProcessors.Get(obj.Name)
	if p == nil {
		processors := make([]string, 0)
		for _, p := range RegisteredProcessors {
			processors = append(processors, p.Name)
		}
		s := strings.Join(processors, ",")
		return errors.New("yaml: unknown processor '" + obj.Name + "' " + s)
	}
	log.Debugf("Unmarshall yaml: FlowStep1: %p %+v", p.Conf, p.Conf)
	f.Conf = p.Conf.Clone()
	f.Name = obj.Name
	f.ScaleOrdered = obj.ScaleOrdered
	f.ScaleUnordered = obj.ScaleUnordered

	err := obj.Conf.Decode(f.Conf)

	log.Debugf("Unmarshall yaml: FlowStep2: %p %+v", f.Conf, f.Conf)
	return err
}

func (flow *Flow) Start(ctx context.Context, all bool, ctl chan ControlEvent, channels *Channels, processors *Processors) ([]*Processor, error) {
	if flow.Disable && !all {
		log.Warnln("flow", flow.Name, "disabled")
		return nil, nil
	}
	log.Infoln("flow", flow.Name, "starting...")
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

		if p == nil { // FIXME: cannot be nil : already check when loading configuration
			log.Errorln("Processor", flow.Name+"/"+step.Name, "not found")
			closest := (*processors)[0].Name
			closest_d := tools.Levenshtein(step.Name, closest)
			for _, p := range *processors {
				d := tools.Levenshtein(step.Name, p.Name)
				if d < closest_d {
					closest = p.Name
				}
			}
			log.Errorln("Processor", flow.Name+"/"+step.Name, "not found, maybe", closest)
			return nil, fmt.Errorf("Processor " + flow.Name + "/" + step.Name + " not found, maybe " + closest)
		} else {
			p2 := *p // Clone Processor
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
			log.Infoln("flow", flow.Name, p.Name, fmt.Sprintf("%+v", p.Conf))
			if step.ScaleOrdered > 0 {
				ParallelOrdered(ctx, channelName+"-scale", step.ScaleOrdered, ctl, in.C, out.C, channels, p)
			} else if step.ScaleUnordered > 0 {
				ParallelUnordered(ctx, channelName+"-scale", step.ScaleUnordered, ctl, in.C, out.C, channels, p)
			} else {
				r, err := p.Conf.Start(ctx, p, ctl, in.GetC(), out.GetC())
				if err != nil {
					log.Fatalln(flow.Name+"/"+step.Name, "failed to start", err)
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
	log.Infoln("flow", flow.Name, "::=", flowtxt)
	return runtimeProcessor, nil
}
