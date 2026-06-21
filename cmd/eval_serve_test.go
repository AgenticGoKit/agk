package cmd

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseServeCmd(t *testing.T) {
	cases := []struct {
		in       string
		wantName string
		wantArgs []string
	}{
		{"", "go", []string{"run", "."}},
		{"   ", "go", []string{"run", "."}},
		{"./server", "./server", []string{}},
		{"go run ./cmd/server", "go", []string{"run", "./cmd/server"}},
		{"mybin --eval --port 8787", "mybin", []string{"--eval", "--port", "8787"}},
	}
	for _, c := range cases {
		name, args := parseServeCmd(c.in)
		if name != c.wantName || !reflect.DeepEqual(args, c.wantArgs) {
			t.Errorf("parseServeCmd(%q) = (%q, %v), want (%q, %v)", c.in, name, args, c.wantName, c.wantArgs)
		}
	}
}

func TestWaitForHealthyBecomesHealthy(t *testing.T) {
	var ready atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" && ready.Load() {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "starting", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	// Flip to healthy shortly after we start polling.
	go func() {
		time.Sleep(300 * time.Millisecond)
		ready.Store(true)
	}()

	if err := waitForHealthy(server.URL, 5*time.Second); err != nil {
		t.Fatalf("waitForHealthy returned error: %v", err)
	}
}

func TestWaitForHealthyTimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "never ready", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	if err := waitForHealthy(server.URL, 1*time.Second); err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}
