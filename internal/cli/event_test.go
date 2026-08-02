package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sporttrax-inc/sporttrax-cli/internal/auth"
)

// stubCatalogAPI serves events and base-events, mimicking the server
// envelope. Payloads carry a future_field to prove --json passthrough,
// and base events carry the TFRRS/Hy-Tek codes the CLI does not render.
func stubCatalogAPI(t *testing.T) (*httptest.Server, *atomic.Int32, *atomic.Value) {
	t.Helper()
	var requests atomic.Int32
	var lastQuery atomic.Value
	mux := http.NewServeMux()

	mux.HandleFunc("/public-api/v1/events", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		lastQuery.Store(r.URL.Query())
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("cursor") == "" {
			fmt.Fprint(w, `{
				"data": [
					{"id": 1, "constant": "female-100-meter-hurdles", "name": "100m Hurdles",
					 "name_with_gender": "Female 100m Hurdles", "gender": "female",
					 "base_event_id": 1, "multi_event_id": null, "future_field": "surprise"},
					{"id": 2, "constant": "male-110-meter-hurdles", "name": "110m Hurdles",
					 "name_with_gender": "Male 110m Hurdles", "gender": "male",
					 "base_event_id": 2, "multi_event_id": null, "future_field": "surprise"}
				],
				"links": {"next": "/public-api/v1/events?cursor=page2"}
			}`)
			return
		}
		fmt.Fprint(w, `{"data": [{"id": 3, "constant": "female-100-meter", "name": "100m Dash",
			"name_with_gender": "Female 100m Dash", "gender": "female", "base_event_id": 3,
			"multi_event_id": null}], "links": {"next": null}}`)
	})

	mux.HandleFunc("/public-api/v1/events/1", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": 1, "constant": "female-100-meter-hurdles", "name": "100m Hurdles",
			"name_with_gender": "Female 100m Hurdles", "gender": "female", "base_event_id": 1,
			"multi_event_id": null, "future_field": "surprise"}`)
	})

	mux.HandleFunc("/public-api/v1/base-events", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		lastQuery.Store(r.URL.Query())
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data": [{"id": 1, "constant": "100-meter-hurdles", "sport": "track",
			"sport_group": "track_sprint", "mark_type": "time", "distance_type": null,
			"tfrrs_code": "100h", "hytek_code": "100h", "is_track": true, "is_field": false,
			"is_relay": false, "is_sprint": true, "is_distance": false, "is_hurdles": true,
			"is_jump": false, "is_vertical_jump": false, "is_horizontal_jump": false,
			"is_throw": false, "is_multi": false, "has_wind": true, "total_distance": "100.00",
			"total_distance_unit": "meter", "future_field": "surprise"}],
			"links": {"next": null}}`)
	})

	mux.HandleFunc("/public-api/v1/base-events/1", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": 1, "constant": "100-meter-hurdles", "sport": "track",
			"sport_group": "track_sprint", "mark_type": "time", "distance_type": null,
			"tfrrs_code": "100h", "hytek_code": "100h", "is_track": true, "is_field": false,
			"is_relay": false, "is_sprint": true, "is_distance": false, "is_hurdles": true,
			"is_jump": false, "is_vertical_jump": false, "is_horizontal_jump": false,
			"is_throw": false, "is_multi": false, "has_wind": true, "total_distance": "100.00",
			"total_distance_unit": "meter"}`)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &requests, &lastQuery
}

func TestEventListTSVAndFilters(t *testing.T) {
	setupCLITest(t)
	srv, _, queries := stubCatalogAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "event", "list",
		"--gender", "female", "--base-event", "1", "--multi-event", "4")
	if err != nil {
		t.Fatalf("event list: %v", err)
	}
	if !strings.Contains(out, "1\tfemale-100-meter-hurdles\t100m Hurdles\tfemale\t1") {
		t.Fatalf("unexpected TSV rows:\n%s", out)
	}
	q, _ := queries.Load().(url.Values)
	for param, want := range map[string]string{
		"filter[gender]":         "female",
		"filter[base_event_id]":  "1",
		"filter[multi_event_id]": "4",
	} {
		if got := q.Get(param); got != want {
			t.Errorf("%s = %q, want %q", param, got, want)
		}
	}
}

func TestEventListJSONPassthrough(t *testing.T) {
	setupCLITest(t)
	srv, _, _ := stubCatalogAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "event", "list", "--json", "--limit", "2")
	if err != nil {
		t.Fatalf("event list --json: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if len(items) != 2 || items[0]["future_field"] != "surprise" {
		t.Fatalf("server fields must pass through verbatim: %v", items)
	}
}

func TestEventListRejectsInvalidGenderBeforeRequest(t *testing.T) {
	setupCLITest(t)
	srv, requests, _ := stubCatalogAPI(t)

	_, _, err := runCommand(t, "--api-url", srv.URL, "event", "list", "--gender", "women")
	if err == nil || !strings.Contains(err.Error(), "must be one of") {
		t.Fatalf("invalid gender must be rejected, got %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("validation must precede any request, got %d", requests.Load())
	}
}

func TestEventListStopsAtLimit(t *testing.T) {
	setupCLITest(t)
	srv, requests, _ := stubCatalogAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "event", "list", "--limit", "2")
	if err != nil {
		t.Fatalf("event list --limit 2: %v", err)
	}
	if lines := strings.Count(strings.TrimSpace(out), "\n") + 1; lines != 2 {
		t.Fatalf("want 2 rows, got %d:\n%s", lines, out)
	}
	if requests.Load() != 1 {
		t.Fatalf("limit must stop paging after 1 request, got %d", requests.Load())
	}
}

func TestEventView(t *testing.T) {
	setupCLITest(t)
	srv, _, _ := stubCatalogAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "event", "view", "1")
	if err != nil {
		t.Fatalf("event view: %v", err)
	}
	for _, want := range []string{"Constant\tfemale-100-meter-hurdles", "Name with gender\tFemale 100m Hurdles", "Base event ID\t1"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestBaseEventListAndFilters(t *testing.T) {
	setupCLITest(t)
	srv, _, queries := stubCatalogAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "base-event", "list",
		"--sport", "track", "--mark-type", "time")
	if err != nil {
		t.Fatalf("base-event list: %v", err)
	}
	// Composed distance is labeled by its parent field.
	if !strings.Contains(out, "1\t100-meter-hurdles\ttrack\ttrack_sprint\ttime\t100.00 meter") {
		t.Fatalf("unexpected TSV row:\n%s", out)
	}
	q, _ := queries.Load().(url.Values)
	if q.Get("filter[sport]") != "track" || q.Get("filter[mark_type]") != "time" {
		t.Fatalf("filters not mapped: %v", q)
	}
}

func TestBaseEventListRejectsInvalidMarkTypeBeforeRequest(t *testing.T) {
	setupCLITest(t)
	srv, requests, _ := stubCatalogAPI(t)

	_, _, err := runCommand(t, "--api-url", srv.URL, "base-event", "list", "--mark-type", "seconds")
	if err == nil || !strings.Contains(err.Error(), "must be one of") {
		t.Fatalf("invalid mark type must be rejected, got %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("validation must precede any request, got %d", requests.Load())
	}
}

// The TFRRS and Hy-Tek codes are integration identifiers for other
// systems: never rendered, but never withheld from --json either.
func TestBaseEventHidesIntegrationCodesButJSONKeepsThem(t *testing.T) {
	setupCLITest(t)
	srv, _, _ := stubCatalogAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "base-event", "view", "1")
	if err != nil {
		t.Fatalf("base-event view: %v", err)
	}
	for _, hidden := range []string{"tfrrs", "TFRRS", "hytek", "Hy-Tek", "100h"} {
		if strings.Contains(out, hidden) {
			t.Fatalf("rendered view must not show %q:\n%s", hidden, out)
		}
	}
	if !strings.Contains(out, "Hurdles\ttrue") || !strings.Contains(out, "Total distance\t100.00 meter") {
		t.Fatalf("expected fields missing:\n%s", out)
	}

	jsonOut, _, err := runCommand(t, "--api-url", srv.URL, "base-event", "view", "1", "--json")
	if err != nil {
		t.Fatalf("base-event view --json: %v", err)
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &rec); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if rec["tfrrs_code"] != "100h" || rec["hytek_code"] != "100h" {
		t.Fatalf("--json is never curated; codes must survive: %v", rec)
	}
}

func TestCatalogCommandsRequireLogin(t *testing.T) {
	setupCLITest(t)
	t.Setenv("SPORTTRAX_API_TOKEN", "")
	srv, _, _ := stubCatalogAPI(t)

	for _, args := range [][]string{
		{"event", "list"},
		{"event", "view", "1"},
		{"base-event", "list"},
		{"base-event", "view", "1"},
	} {
		_, _, err := runCommand(t, append([]string{"--api-url", srv.URL}, args...)...)
		if !errors.Is(err, auth.ErrNotFound) {
			t.Errorf("%v without a token should report ErrNotFound, got %v", args, err)
		}
	}
}

func TestEventAndBaseEventRejectNonNumericIDs(t *testing.T) {
	setupCLITest(t)
	for _, args := range [][]string{
		{"event", "view", "abc"},
		{"base-event", "view", "abc"},
		{"athlete", "view", "abc"},
		{"team", "view", "abc"},
	} {
		_, _, err := runCommand(t, args...)
		if err == nil || !strings.Contains(err.Error(), "must be a number") {
			t.Errorf("%v should reject a non-numeric id, got %v", args, err)
		}
	}
}
