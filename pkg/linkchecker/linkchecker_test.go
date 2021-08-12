package linkchecker_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
	up := linkchecker.IsLinkUp(server.Client(), server.URL)
	if !up {
		t.Fatal("got not ok statement")
	}
}

// what is a broken link?
// TestNotOk checks if a link is down
func TestNotOk(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "Service Unavailable", http.StatusInternalServerError)
	}))
	up := linkchecker.IsLinkUp(server.Client(), server.URL)
	if up {
		t.Fatal("Report up for a bad link")
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
	got := linkchecker.CrawlPageRecusively(server.Client(), server.URL)
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}