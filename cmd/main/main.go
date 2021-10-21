package main

import (
	"net/http"
	_ "net/http/pprof"

	"com.gabizou/actors/pkg/linkchecker"
)

func main() {
	go http.ListenAndServe("localhost:8080", nil)
	linkchecker.Run()
}
