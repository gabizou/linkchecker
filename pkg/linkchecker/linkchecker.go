package linkchecker

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/antchfx/htmlquery"
)

func GetListOfLinks(client *http.Client, link string) []string {
	get, err := client.Get(link)
	if err != nil {
		return nil
	}
	if get.StatusCode != http.StatusOK {
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

var Debug = io.Discard

func IsLinkUp(client *http.Client, url string) (up bool) {
	// if there is no colon in the url then prepend the domain to the url
	fmt.Fprintf(Debug, "IsLinkUp: %s\n", url)
	resp, err := client.Head(url)
	fmt.Fprintln(Debug, "GOT HEAD")
	if err != nil {
		fmt.Fprintf(Debug, "Err when getting head: %v", err)
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
	fmt.Fprintf(Debug, "Status Code for: %s \n is: %d\n", url, statusCode)
	return statusCode == http.StatusOK
}

func CanonnicalizeURL(protocol, domain, url string) string {
	if strings.Contains(url, ":") {
		return url
	}
	return protocol + "://" + domain + url
}

func CrawlPageRecusively(client *http.Client, protocol, domain, link string) []string {
	var brokenLinks []string
	linksToCrawl := make([]string, 1)
	linksToCrawl[0] = link
	for len(linksToCrawl) > 0 {
		// get next link to check
		linkToCrawl := linksToCrawl[len(linksToCrawl)-1]
		linkToCrawl = CanonnicalizeURL(protocol, domain, linkToCrawl)
		// remove link from queue
		linksToCrawl = linksToCrawl[:len(linksToCrawl)-1]

		// if it is not valid, add to broken links list & skip
		if !IsLinkUp(client, linkToCrawl) {
			brokenLinks = append(brokenLinks, linkToCrawl)
			continue
		}
		// if it is not in our domain we skip
		if !IsInOurDomain(link, domain) {
			continue
		}
		// otherwise we get it's body & add links to linksToCrawl
		subLinks := GetListOfLinks(client, link)
		linksToCrawl = append(linksToCrawl, subLinks...)
	}

	return brokenLinks
}

func IsInOurDomain(link, domain string) bool {
	parse, err := url.Parse(link)
	if err != nil {
		return false
	}
	fmt.Fprintf(Debug, "Full Host: %s\n", parse.Host)
	host := parse.Host
	if strings.Contains(parse.Host, ":") {
		splitHost, _, err := net.SplitHostPort(parse.Host)
		if err != nil {
			fmt.Fprintf(Debug, "SplitHostPort err on: %s\n", parse.Host)
			return false
		}
		host = splitHost
	}
	fmt.Fprintf(Debug, "Host: %s\n", host)
	return host == domain
}

func ParseLinks(client *http.Client, links []string) (broken []string, working []string) {
	var wg sync.WaitGroup
	brokenLinks := NewSyncSlice()
	workingLinks := NewSyncSlice()
	for _, link := range links {
		link := link
		wg.Add(1)
		go func() {
			fmt.Fprintln(Debug, "inside go func")
			defer wg.Done()
			isLinkBroken := !IsLinkUp(client, link)
			fmt.Fprintf(Debug, "isLinkBroken: %v\n", isLinkBroken)
			if isLinkBroken {
				brokenLinks.Append(link)
			} else {
				workingLinks.Append(link)
			}
		}()
	}
	wg.Wait()

	return brokenLinks.Items, workingLinks.Items
}
