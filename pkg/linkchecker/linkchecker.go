package linkchecker

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/antchfx/htmlquery"
)

func Run() {
	Debug = os.Stdout
	if len (os.Args) < 2 {
		programName := os.Args[0]
		fmt.Fprintf(os.Stderr, "Usage: %s Link\n", programName)
		os.Exit(1)
	}
	link := os.Args[1]
	brokenLinks := CrawlWebsite(http.DefaultClient, link)
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
	fmt.Fprintf(Debug, "GOT HEAD\n")
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

func CrawlWebsite(client *http.Client, link string) []string {
	var brokenLinks []string
	linksToCrawl := make([]string, 1)
	linksToCrawl[0] = link
	//checkedLinks := make(map[string]bool)
	broken, _ := ParseLinks(client, link, linksToCrawl)
	brokenLinks = append(brokenLinks, broken...)
	//for len(linksToCrawl) > 0 {
	//	time.Sleep(500 * time.Millisecond)
	//
	//	linkToCrawl := linksToCrawl[len(linksToCrawl)-1]
	//	linkToCrawl = canonicalizeLink(linkToCrawl, domain)
	//	fmt.Fprintf(Debug, "Checking link: %s\n", linkToCrawl)
	//	linksToCrawl = linksToCrawl[:len(linksToCrawl)-1]
	//	checkedLinks[linkToCrawl] = true
	//
	//
	//
	//	if !IsInOurDomain(linkToCrawl, domain) {
	//		continue
	//	}
	//	//
	//	//subLinks := GetListOfLinks(client, linkToCrawl)
	//	//
	//	//fmt.Fprintf(Debug, "Checked Link: %v\n", checkedLinks)
	//	//
	//	//for _, sublink := range subLinks {
	//	//	sublink = canonicalizeLink(sublink, domain)
	//	//	checked := checkedLinks[sublink]
	//	//	if !checked {
	//	//		linksToCrawl = append(linksToCrawl, sublink)
	//	//	}
	//	//}
	//}

	return brokenLinks
}

func canonicalizeLink(link, domain string) string {
	link = PrependDomainIfNecessary(link, domain)
	link = PrependHttpsIfNecessary(link)
	return link
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

var workerPool = runtime.NumCPU() * 8

func ParseLinks(client *http.Client, website string, links []string) (broken []string, working []string) {
	var wg sync.WaitGroup
	brokenLinks := newCollector()
	workingLinks := newCollector()
	mailbox := make(chan string, workerPool)
	stop := make(chan interface{})
	wg.Add(workerPool)
	domain := ExtractDomain(website)
	appendLinks := make(chan string, 512)
	visited := make(map[string]bool)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			case link :=<-appendLinks:
				if !visited[link] {
					wg.Add(1)
					go func() {
						defer wg.Done()
						mailbox<-link
					}()
					visited[link] = true
				}
			}
		}
	}()
	for i := 0; i < workerPool; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				case link := <-mailbox:
					status := GetLinkStatus(client, link)
					switch status {
					case Down:
						fmt.Fprintf(Debug, "link is broken: %v\n", link)
						brokenLinks.addLink(link)
					case RateLimited:
						fmt.Printf("Getting rate limited on %s\n", link)
					case Up:
						fmt.Printf("Got OK for link, will get sublinks %s\n", link)
						subLinks := GetListOfLinks(client, link)
						for _, sublink := range subLinks {
							sublink = canonicalizeLink(sublink, domain)
							appendLinks <- sublink
						}
						workingLinks.addLink(link)
					}
				}
			}
		}()
	}
	for _, link := range links {
		link := link
		mailbox <- link
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range time.NewTicker(time.Second).C {
			fmt.Printf("length of mailbox: %d\n", len(mailbox) + len(appendLinks))
			if len(mailbox) == 0 && len(appendLinks) == 0 {
				close(stop)
				return
			}
		}
	}()
	wg.Wait()

	return brokenLinks.getLinks(), workingLinks.getLinks()
}
