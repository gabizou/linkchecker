package linkchecker_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"com.gabizou/actors/pkg/linkchecker"
	"github.com/google/go-cmp/cmp"
)

func TestLinkChecker(t *testing.T) {
	reader := strings.NewReader(`<a href="https://google.com/"/a>`)
	lines := linkchecker.GetListOfLinks(reader)
	expected := []string{"https://google.com/"}
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
	got := linkchecker.GetBrokenLinks(server.Client(), want)
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
	got := linkchecker.GetBrokenLinks(server.Client(), links)
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}