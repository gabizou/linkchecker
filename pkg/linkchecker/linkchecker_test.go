package linkchecker_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"com.gabizou/actors/pkg/linkchecker"
	"github.com/google/go-cmp/cmp"
)

func TestLinkChecker(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		fmt.Fprintf(writer, `<a href="%s">Link Here</a>`, "https://www.example.com/")
	}))
	lines := linkchecker.GetListOfLinks(server.Client(), server.URL)
	expected := []string{"https://www.example.com/"}
	if !cmp.Equal(expected, lines) {
		t.Fatal(cmp.Diff(expected, lines))
	}
}

func TestLinkStatus(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	got := linkchecker.GetLinkStatus(server.Client(), server.URL)
	if got != linkchecker.Up {
		t.Fatal("got not ok statement")
	}
}

func TestLinkStatusTooManyRequests(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusTooManyRequests)
	}))
	got := linkchecker.GetLinkStatus(server.Client(), server.URL)
	if got != linkchecker.RateLimited {
		t.Fatal("got not RateLimited statement")
	}
}

// what is a broken link?
// TestLinkStatus_Down checks if a link is down
func TestLinkStatus_Down(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "Service Unavailable", http.StatusInternalServerError)
	}))
	got := linkchecker.GetLinkStatus(server.Client(), server.URL)
	if got != linkchecker.Down {
		t.Fatal("Report up for a bad link")
	}
}

func TestPrependDomainIfNecessary(t *testing.T) {
	tcs := []struct {
		link string
		want string
	}{
		{"google.com/more", "google.com/more"},
		{"/more-from-google", "google.com/more-from-google"},
		{"https://google.com/more", "https://google.com/more"},
		{"google.com:8080/more", "google.com:8080/more"},
		{"/more", "google.com/more"},
	}
	for _, tc := range tcs {
		got := linkchecker.PrependDomainIfNecessary(tc.link, "google.com")
		if tc.want != got {
			t.Fatalf("Want: %s, Got: %s", tc.want, got)
		}
	}
}

func TestPrependHttpsIfNecessary(t *testing.T) {
	tcs := []struct {
		link string
		want string
	}{
		{"google.com/more", "https://google.com/more"},
		{"https://google.com/more", "https://google.com/more"},
		{"google.com:8080/more", "https://google.com:8080/more"},
		{"https://127.0.0.1:59476", "https://127.0.0.1:59476"},
	}
	for _, tc := range tcs {
		got := linkchecker.PrependHttpsIfNecessary(tc.link)
		if tc.want != got {
			t.Fatalf("Want: %s, Got: %s", tc.want, got)
		}
	}
}

func TestVerifyBrokenLinks(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "Service Unavailable", http.StatusInternalServerError)
	}))
	want := []string{server.URL}
	got, _ := linkchecker.ParseLinks(server.Client(), want)
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestVerifyValidLinks(t *testing.T) {
	want := []string{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	links := []string{server.URL}
	got, _ := linkchecker.ParseLinks(server.Client(), links)
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestNewSyncSlice(t *testing.T) {
	slice := linkchecker.NewSyncSlice()
	if slice.Items == nil {
		t.Fatal("links is nil")
	}
	slice.Append("www.example.com")
	if len(slice.Items) != 1 {
		t.Fatalf("items array elements: want %d, got %d", 1, len(slice.Items))
	}
}

func TestVerifySubPages(t *testing.T) {
	badServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "Service Unavailable", http.StatusInternalServerError)
	}))
	want := []string{badServer.URL}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		fmt.Fprintf(writer, `<a href="%s">Link Here</a>`, badServer.URL)
	}))
	linkchecker.Debug = os.Stdout
	got := linkchecker.CrawlWebsite(server.Client(), server.URL)
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestCyclicLinkLoops(t *testing.T) {
	var otherURL string
	var handlerHeadCallCount int
	var handlerGetCallCount int
	server1 := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		switch request.Method {
		case http.MethodGet:
			handlerGetCallCount++
		case http.MethodHead:
			handlerHeadCallCount++
		}
		fmt.Fprintf(writer, `<a href="%s">Link Here</a>`, otherURL)
	}))
	var want []string
	wantGetCount := 1
	wantHeadCount := 1
	otherURL = server1.URL

	linkchecker.Debug = os.Stdout
	got := linkchecker.CrawlWebsite(server1.Client(), server1.URL)
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
	if wantGetCount != handlerGetCallCount {
		t.Fatalf("handler get called %d times, expected %d times", handlerGetCallCount, wantGetCount)
	}
	if wantHeadCount != handlerHeadCallCount {
		t.Fatalf("handler head called %d times, expected %d times", handlerHeadCallCount, wantHeadCount)
	}
}

func TestExtractDomain(t *testing.T) {
	url := "https://google.com/search"
	got := linkchecker.ExtractDomain(url)
	want := "google.com"
	if got != want {
		t.Fatal(cmp.Diff(want, got))
	}
	url = "google.com/search/more"
	got = linkchecker.ExtractDomain(url)
	want = "google.com"
	if got != want {
		t.Fatal(cmp.Diff(want, got))
	}
	url = "google.com:8080/search/more"
	got = linkchecker.ExtractDomain(url)
	want = "google.com:8080"
	if got != want {
		t.Fatal(cmp.Diff(want, got))
	}
	url = "https://127.0.0.1:59454"
	got = linkchecker.ExtractDomain(url)
	want = "127.0.0.1:59454"
	if got != want {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestIsInOurDomain(t *testing.T) {
	testCases := []struct {
		link string
		want bool
	}{
		{"http://www.google.com", false},
		{"https://www.google.com", false},
		{"https://www.google.com/bitfieldconsulting.com/you'vebeengotten", false},
		{"https://bitfieldconsulting.com/", true},
		{"https://bitfieldconsulting.com/moreStuff", true},
	}
	for _, tc := range testCases {
		got := linkchecker.IsInOurDomain(tc.link, "bitfieldconsulting.com")
		if tc.want != got {
			t.Errorf("Link: %s, Want: %t, got: %t", tc.link, tc.want, got)
		}
	}
}

func TestOnlyHeadForThirdPartySites(t *testing.T) {
	thirdPartyHeadCallCount := 0
	thirdParty := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		switch request.Method {
		case http.MethodGet:
			t.Fatal("Called get on third party site")
		case http.MethodHead:
			thirdPartyHeadCallCount++
		}
	}))
	ourWebsite := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(writer, `<a href="%s">Link Here</a>`, thirdParty.URL)
	}))

	linkchecker.Debug = os.Stdout
	_ = linkchecker.CrawlWebsite(ourWebsite.Client(), ourWebsite.URL)
	if thirdPartyHeadCallCount != 1 {
		t.Fatalf("handler get called %d times, expected %d times", thirdPartyHeadCallCount, 1)
	}
}
