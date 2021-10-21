package linkchecker

import (
	"net/http"
)

type checker struct {
	mailbox          chan func()
	outbox           chan bool
	brokenCol        *collector
	workingCollector *collector
	client           *http.Client
}

func newChecker(client *http.Client, brokenCol *collector, workingCol *collector) *checker {
	c := &checker{
		mailbox:          make(chan func(), 1),
		outbox:           make(chan bool, 1),
		brokenCol:        brokenCol,
		workingCollector: workingCol,
		client:           client,
	}
	go c.start()
	return c
}

func (c *checker) check(link string) bool {
	c.mailbox <- func() {
		working := GetLinkStatus(c.client, link)
		if working == Up {
			c.workingCollector.addLink(link)
		} else {
			c.brokenCol.addLink(link)
		}
		c.outbox <- working == Up
	}
	return <-c.outbox
}

func (c *checker) start() {
	for {
		action := <-c.mailbox
		action()
	}
}
