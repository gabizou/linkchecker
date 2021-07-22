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
	status, ok := linkchecker.VerifyStatus(server.Client(), server.URL)
	if !ok {
		t.Fatal("got not ok statement")
	}
	if status != http.StatusOK {
		t.Fatalf("Got a non ok status %d", status)
	}
}

func TestNotOk(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "Service Unavailable", http.StatusInternalServerError)
	}))
	status, ok := linkchecker.VerifyStatus(server.Client(), server.URL)
	if ok {
		t.Fatal("got ok statement")
	}
	if status != http.StatusInternalServerError {
		t.Fatalf("Got a non 500 status %d", status)
	}
}
