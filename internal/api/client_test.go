package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetSendsIdentificationHeaders(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "1|tok", false)
	if err := c.Get(context.Background(), "/events", nil, nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Get("Authorization") != "Bearer 1|tok" {
		t.Errorf("Authorization = %q", got.Get("Authorization"))
	}
	if got.Get("X-SPORTTRAX-CLIENT-DEVICE-TYPE") != "cli" {
		t.Errorf("device type = %q, want cli", got.Get("X-SPORTTRAX-CLIENT-DEVICE-TYPE"))
	}
	if got.Get("X-SPORTTRAX-CLI-VERSION") == "" || got.Get("X-SPORTTRAX-CLIENT-NOW") == "" {
		t.Errorf("missing version/now headers: %v", got)
	}
	if !strings.HasPrefix(got.Get("User-Agent"), "sporttrax-cli/") {
		t.Errorf("User-Agent = %q", got.Get("User-Agent"))
	}
}

func TestGetRetriesRateLimitThenSucceeds(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	var out map[string]bool
	if err := New(srv.URL, "t", false).Get(context.Background(), "/meets", nil, &out); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if attempts != 2 || !out["ok"] {
		t.Fatalf("attempts=%d out=%v, want retry then success", attempts, out)
	}
}

func TestGetMapsAuthErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"This token is not permitted to use the public API."}`))
	}))
	defer srv.Close()

	err := New(srv.URL, "t", false).Get(context.Background(), "/meets", nil, nil)
	if !IsAuthError(err) {
		t.Fatalf("IsAuthError = false for %v", err)
	}
	if !strings.Contains(err.Error(), "not permitted to use the public API") {
		t.Fatalf("server message not surfaced: %v", err)
	}
}

func TestGetMaintenanceHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-SPORTTRAX-DOWN", "1")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	err := New(srv.URL, "t", false).Get(context.Background(), "/meets", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "maintenance") {
		t.Fatalf("maintenance header not mapped: %v", err)
	}
}

func TestGetHonorsCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := New(srv.URL, "t", false).Get(ctx, "/meets", nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestValidateBaseURLRefusesPlaintext(t *testing.T) {
	ok := []string{
		"https://sporttrax.com",
		"https://localhost:8555",
		"http://localhost:8555",
		"http://127.0.0.1:8555",
		"http://[::1]:8555",
	}
	for _, u := range ok {
		if err := ValidateBaseURL(u); err != nil {
			t.Errorf("ValidateBaseURL(%q) = %v, want allowed", u, err)
		}
	}

	// Anything that would put a bearer token on the wire in the clear, and
	// anything whose scheme we can't reason about.
	bad := []string{
		"http://sporttrax.com",
		"http://staging.sporttrax.com",
		"http://192.168.1.10:8555",
		"ftp://sporttrax.com",
		"sporttrax.com",
	}
	for _, u := range bad {
		if err := ValidateBaseURL(u); err == nil {
			t.Errorf("ValidateBaseURL(%q) = nil, want refusal", u)
		}
	}
}

func TestIsLoopbackHost(t *testing.T) {
	for _, h := range []string{"localhost", "localhost:8555", "127.0.0.1:9000", "[::1]:80", "::1"} {
		if !IsLoopbackHost(h) {
			t.Errorf("IsLoopbackHost(%q) = false", h)
		}
	}
	for _, h := range []string{"sporttrax.com", "192.168.1.10:8555", "notlocalhost.com"} {
		if IsLoopbackHost(h) {
			t.Errorf("IsLoopbackHost(%q) = true", h)
		}
	}
}

func TestGetRejectsOversizedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// A well-formed but unending array: a decoder without a ceiling
		// would keep allocating until the process died.
		fmt.Fprint(w, `{"data":[`)
		chunk := strings.Repeat(`{"id":1},`, 1024)
		for written := 0; written < maxResponseBytes+len(chunk); written += len(chunk) {
			if _, err := fmt.Fprint(w, chunk); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	var out map[string]any
	err := New(srv.URL, "t", false).Get(context.Background(), "/meets", nil, &out)
	if err == nil || !strings.Contains(err.Error(), "exceeded") {
		t.Fatalf("want size-limit error, got %v", err)
	}
}

func TestListReportsHasMore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("cursor") == "" {
			fmt.Fprint(w, `{"data": [{"id": 1}, {"id": 2}], "links": {"next": "/x?cursor=p2"}}`)
			return
		}
		fmt.Fprint(w, `{"data": [{"id": 3}], "links": {"next": null}}`)
	}))
	defer srv.Close()
	c := New(srv.URL, "t", false)

	// Limit hit while a next cursor exists: more available.
	res, err := c.List(context.Background(), "/things", nil, 2)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Items) != 2 || !res.HasMore {
		t.Fatalf("want 2 items with HasMore=true, got %d/%v", len(res.Items), res.HasMore)
	}

	// Depleting everything: no more.
	res, err = c.List(context.Background(), "/things", nil, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Items) != 3 || res.HasMore {
		t.Fatalf("want 3 items with HasMore=false, got %d/%v", len(res.Items), res.HasMore)
	}
}
