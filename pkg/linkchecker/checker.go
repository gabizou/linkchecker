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

func VerifyStatus(client *http.Client, url string) (httpStatus int, ok bool) {
	resp, err := client.Head(url)
	var statusCode int
	if resp != nil {
		statusCode = resp.StatusCode
	} else {
		statusCode = 0
	}
	if resp.Body != nil {
		resp.Body.Close()
	}
	return statusCode, err == nil
}
