package linkchecker

import (
	"io"
	"net/http"

	"github.com/antchfx/htmlquery"
)

func GetListOfLinks(reader io.Reader) []string {
	doc, err := htmlquery.Parse(reader)
	if err != nil {
		return nil
	}
	list := htmlquery.Find(doc, "//a/@href")
	var links []string
	for _, n := range list {
		attr := htmlquery.SelectAttr(n, "href")
		links = append(links, attr)
	}
	return links
}


func IsLinkUp(client *http.Client, url string) (up bool) {
	resp, err := client.Head(url)
	if err != nil {
		return false
	}
	var statusCode int
	if resp != nil {
		statusCode = resp.StatusCode
	} else {
		statusCode = 0
	}
	if resp.Body != nil {
		resp.Body.Close()
	}
	// todo let's check the status code against a list of known good status codes
	return statusCode == http.StatusOK
}

func GetBrokenLinks(client *http.Client, links []string) []string {
	brokenLinks := make([]string, 0)
	for _, link := range links {
		if !IsLinkUp(client, link) {
			brokenLinks = append(brokenLinks, link)
		}
	}

	return brokenLinks
}
