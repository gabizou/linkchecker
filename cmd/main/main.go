package main

import (
	"fmt"

	"com.gabizou/actors/pkg/actor"
)

func main() {
	f := actor.New()
	f.SetGreeting("hello")
	fmt.Println(f.GreetMe("world"))

}
