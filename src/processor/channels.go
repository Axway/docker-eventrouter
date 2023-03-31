package processor

import (
	"axway.com/qlt-router/src/config"
	log "github.com/sirupsen/logrus"
)

var flowChannelSize = config.DeclareInt("processor.flowChannelSize", 100, "Default Processor channel size")

type Channel struct {
	Name string
	Size int
	Pos  int               // Only debug/ui purposes
	C    chan AckableEvent `json:"-"`
}

func (c *Channel) GetC() chan AckableEvent {
	if c != nil {
		return c.C
	}
	return nil
}

type Channels struct {
	Channels []Channel
}

func NewChannels() *Channels {
	return &Channels{}
}

func (c *Channels) Display() {
	for _, channel := range c.Channels {
		log.Println("channel", channel.Name, len(channel.C))
	}
}

func (c *Channels) Update() {
	for _, channel := range c.Channels {
		channel.Pos = len(channel.C)
	}
}

func (c *Channels) Create(name string, size int) *Channel {
	if size == -1 {
		size = flowChannelSize
	}
	ch := make(chan AckableEvent, size)
	channel := c._add(name, size, ch)
	return channel
}

func (c *Channels) _add(name string, size int, ch chan AckableEvent) *Channel {
	channel := Channel{name, size, 0, ch}
	c.Channels = append(c.Channels, channel)
	return &channel
}
