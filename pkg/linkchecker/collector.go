package linkchecker

import (
	"context"
	"fmt"
)

type collector struct {
	mailbox chan func()
	outbox  chan []string
	stop    chan interface{}
	// this is our state stuff
	links   []string
}

func newCollector() *collector {
	col := &collector{
		make(chan func(), 1),
		make(chan []string, 1),
		make(chan interface{}, 1),
		make([]string, 0),
	}
	go col.start()
	return col
}

func (col *collector) addLink(link string) {
	col.mailbox <- func() {
		fmt.Printf("Adding link %s\n", link)
		col.links = append(col.links, link)
	}
}

func (col *collector) getLinks() []string {
	col.mailbox <- func() {
		links := make([]string, len(col.links))
		copy(links, col.links)
		col.outbox <- links
	}
	return <-col.outbox
}

func (col *collector) start() {
	for {
		select {
		case <-col.stop:
			return
		case action := <-col.mailbox:
			action()
		}
	}
}

func (col *collector) _stop() {
	col.stop <- context.Background()
}