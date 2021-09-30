package linkchecker

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/antchfx/htmlquery"
)

func Run() {
	Debug = os.Stdout
	if len (os.Args) < 2 {
		programName := os.Args[0]
		fmt.Fprintf(os.Stderr, "Usage: %s Link", programName)
		os.Exit(1)
	}
	link := os.Args[1]
	brokenLinks := CrawlPageRecursively(http.DefaultClient, link)
	for _, link := range brokenLinks {
		fmt.Printf("BROKEN: %s\n", link)
	}
}

func GetListOfLinks(client *http.Client, link string) []string {
	fmt.Fprintf(Debug, "Get request on: %s\n", link)
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

func GetLinkStatus(client *http.Client, url string) PageStatus {
	// if there is no colon in the url then prepend the domain to the url
	fmt.Fprintf(Debug, "GetLinkStatus: %s\n", url)
	resp, err := client.Head(url)
	fmt.Fprintln(Debug, "GOT HEAD")
	if err != nil {
		fmt.Fprintf(Debug, "Err when getting head: %v\n", err)
		return Down
	}
	var statusCode int
	if resp != nil {
		statusCode = resp.StatusCode
	}
	if resp.Body != nil {
		resp.Body.Close()
	}
	// todo let's check the status code against a list of known good status codes
	fmt.Fprintf(Debug, "Status Code for: %s \n is: %d\n", url, statusCode)
	switch statusCode {
	case http.StatusOK,http.StatusAccepted,http.StatusCreated:
		return Up
	case http.StatusTooManyRequests:
		return RateLimited
	default:
		return Down
	}
}

func PrependDomainIfNecessary(link string, domain string) string {
	if strings.HasPrefix(link, "/") { // todo domain = localhost may break this
		return domain + link
	}
	return link
}

// todo - fix for other protocols
func PrependHttpsIfNecessary(link string) string {
	if strings.HasPrefix(link, "http") {
		return link
	}
	return "https://" + link
}

type PageStatus int

const (
	Up PageStatus = iota
	Down
	RateLimited
)

func ExtractDomain(link string) string {
	parse, err := url.Parse(PrependHttpsIfNecessary(link))
	if err != nil {
		return link // todo ?
	}
	return parse.Host
}

func CrawlPageRecursively(client *http.Client, link string) []string {
	var brokenLinks []string
	linksToCrawl := make([]string, 1)
	linksToCrawl[0] = link
	domain := ExtractDomain(link)
	checkedLinks := make(map[string]bool)
	for len(linksToCrawl) > 0 {
		time.Sleep(500 * time.Millisecond)

		// get next link to check
		linkToCrawl := linksToCrawl[len(linksToCrawl)-1]
		fmt.Fprintf(Debug, "Checking link: %s", linkToCrawl)
		linkToCrawl = PrependDomainIfNecessary(linkToCrawl, domain)
		linkToCrawl = PrependHttpsIfNecessary(linkToCrawl)
		// remove link from queue
		linksToCrawl = linksToCrawl[:len(linksToCrawl)-1]
		checkedLinks[linkToCrawl] = true

		// check wait timer and, if there is a wait timer for this domain
		// wait that amount of time

		// if it is not valid, add to broken links list & skip
		// tell us up,down or rate limited
		// if rate limited, then set a timer for that domain
		// and add that link to the back of the queue
		if GetLinkStatus(client, linkToCrawl) == Down {
			brokenLinks = append(brokenLinks, linkToCrawl)
			continue
		}
		// if it is not in our domain we skip
		if !IsInOurDomain(linkToCrawl, domain) {
			continue
		}
		// otherwise we get it's body & add links to linksToCrawl
		subLinks := GetListOfLinks(client, linkToCrawl)
		for _, subLink := range subLinks {
			_, ok := checkedLinks[subLink]
			if !ok {
				linksToCrawl = append(linksToCrawl, subLink)
			}
		}
	}

	return brokenLinks
}

func IsInOurDomain(link, domain string) bool {
	parse, err := url.Parse(link)
	if err != nil {
		return false
	}
	host := parse.Host
	fmt.Fprintf(Debug, "Host: %s\n", host)
	fmt.Fprintf(Debug, "Domain: %s\n", domain)
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
			isLinkBroken := GetLinkStatus(client, link) == Down
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
