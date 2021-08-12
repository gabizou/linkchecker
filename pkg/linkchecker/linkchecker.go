package linkchecker

import (
	"net/http"
	"sync"

	"github.com/antchfx/htmlquery"
)

func GetListOfLinks(client *http.Client, link string) []string {
	get, err := client.Get(link)
	if err != nil {
		return nil
	}
	if get.StatusCode != http.StatusOK  {
		return nil
	}
	doc, err := htmlquery.Parse(get.Body)
	if err != nil {
		return nil
	}
	defer get.Body.Close()
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

func CrawlPageRecusively(client *http.Client, link string) []string {
	var brokenLinks []string
	linksToCrawl := make([]string, 1)
	linksToCrawl[0] = link
	for len(linksToCrawl) > 0 {
		// get next link to check
		linkToCrawl := linksToCrawl[len(linksToCrawl)-1]
		// remove link from queue
		linksToCrawl = linksToCrawl[:len(linksToCrawl)-1]

		// if it is not valid, add to broken links list & skip
		if !IsLinkUp(client, linkToCrawl) {
			brokenLinks = append(brokenLinks, linkToCrawl)
			continue
		}
		// if it is not in our domain we skip
		if !isInOurDomain(link) {
			continue
		}
		// otherwise we get it's body & add links to linksToCrawl
		 subLinks := GetListOfLinks(client, link)
		 linksToCrawl = append(linksToCrawl, subLinks...)
	}

	return brokenLinks
}

func isInOurDomain(link string) bool {
	return true
}

func ParseLinks(client *http.Client, links []string) ([]string, []string) {
	var wg sync.WaitGroup
	brokenLinks := NewSyncSlice()
	workingLinks := NewSyncSlice()
	for _, link := range links {
		link := link
		go func() {
			wg.Add(1)
			defer wg.Done()
			if !IsLinkUp(client, link) {
				brokenLinks.Append(link)
			} else {
				workingLinks.Append(link)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	return brokenLinks.Items, workingLinks.Items
}
