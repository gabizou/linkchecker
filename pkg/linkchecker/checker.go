package linkchecker

import (
	"io"
	"net/http"
	"sync"

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


type syncSlice struct {
	Items []string
	mutex sync.Mutex
}

func NewSyncSlice() syncSlice {
	return syncSlice{Items: []string{}}
}

func (ss *syncSlice) Append(s string) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	ss.Items = append(ss.Items, s)
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
	var wg sync.WaitGroup
	discoveredBrokenLinks := NewSyncSlice()
	for _, link := range links {
		link := link
		go func() {
			if !IsLinkUp(client, link) {
				wg.Add(1)
				discoveredBrokenLinks.Append(link)
				defer wg.Done()
				wg.Done()
			}
		}()
	}
	wg.Wait()

	return discoveredBrokenLinks.Items
}
