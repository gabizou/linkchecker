package actor

import (
	"fmt"
)

const mailboxSize = 100

type action func()
type Actor struct {
	mailbox              chan action
	messageToRespondWith string
}

func (a *Actor) SetGreeting(greeting string) {
	a.mailbox <- func() {
		a.messageToRespondWith = greeting
	}
}

func (a *Actor) GreetMe(person string) string {
	receiver := make(chan string)
	a.mailbox <- func() {
		receiver <- fmt.Sprintf("%s, %s", a.messageToRespondWith, person)
	}
	for {
		msg := <-receiver
		return msg
	}
}

func New() *Actor {
	ch := make(chan action, mailboxSize)
	actor := &Actor{mailbox: ch}
	go actor.loop()
	return actor
}

func (a *Actor) loop() {
	for {
		action := <-a.mailbox
		action()
	}
}
