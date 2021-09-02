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
	up := linkchecker.IsLinkUp(server.Client(), server.URL)
	if !up {
		t.Fatal("got not ok statement")
	}
}

func TestCannonicalURL_Relative(t *testing.T) {
	domain := "bitfield.com"
	input := "/about"
	want := "https://bitfield.com/about"
	got := linkchecker.CanonnicalizeURL("https", domain, input)
	if want != got {
		t.Fatalf("Want: %s, Got: %s", want, got)
	}
}

func TestCannonicalURL_Absolute(t *testing.T) {
	domain := "bitfield.com"
	input := "https://bitfield.com/about"
	want := "https://bitfield.com/about"
	got := linkchecker.CanonnicalizeURL("https", domain, input)
	if want != got {
		t.Fatalf("Want: %s, Got: %s", want, got)
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
	linkchecker.Debug = os.Stdout
	got := linkchecker.CrawlPageRecusively(server.Client(), "https","127.0.0.1", server.URL)
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestIsInOurDomain(t *testing.T) {
	testCases := []struct {
		link string
		want  bool
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
