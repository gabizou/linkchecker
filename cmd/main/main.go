package main

import (
	"fmt"
	"net/http"
	"os"

	"com.gabizou/actors/pkg/linkchecker"
)

func main() {
	linkchecker.Debug = os.Stdout
	brokenLinks := linkchecker.CrawlPageRecusively(http.DefaultClient, "https://bitfieldconsulting.com/")
	for _, link := range brokenLinks {
		fmt.Printf("BROKEN: %s\n", link)
	}
}
