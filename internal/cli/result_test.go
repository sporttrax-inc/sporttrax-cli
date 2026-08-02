package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

// stubResultsAPI serves two pages of results plus a single-result
// endpoint, mimicking the server envelope. Payloads carry a future_field
// to prove --json passthrough.
func stubResultsAPI(t *testing.T) (*httptest.Server, *atomic.Int32, *atomic.Value) {
	t.Helper()
	var requests atomic.Int32
	var lastQuery atomic.Value
	mux := http.NewServeMux()
	mux.HandleFunc("/public-api/v1/results", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		lastQuery.Store(r.URL.Query())
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("cursor") == "" {
			fmt.Fprint(w, `{
				"data": [
					{"id": 11, "meet_id": 4821, "at": "2026-06-01", "round": "finals", "is_relay_team": false,
					 "athlete": {"first_name": "Pat", "last_name": "Runner", "grade": 11},
					 "team": {"name": "Salem Hills", "abbr": "SH"},
					 "meet": {"name": "State Championship"},
					 "event": {"name": "100 Meter", "name_with_gender": "Boys 100 Meter"},
					 "division": {"name": "Varsity", "abbr": "V"},
					 "mark": {"display": "12.34", "english": "12.34e", "metric": "12.34m"}, "is_legal": true, "place": 1,
					 "future_field": "surprise"},
					{"id": 12, "meet_id": 4821, "at": "2026-06-01", "round": "finals", "is_relay_team": true,
					 "relay_team_name": "Salem Hills A",
					 "athlete": {"first_name": "", "last_name": "", "grade": null},
					 "team": {"name": "Salem Hills", "abbr": "SH"},
					 "mark": {"display": "42.10"}, "is_legal": true, "place": null,
					 "future_field": "surprise"}
				],
				"links": {"next": "/public-api/v1/results?cursor=page2"}
			}`)
			return
		}
		fmt.Fprint(w, `{
			"data": [
				{"id": 13, "meet_id": 4821, "at": "2026-06-02", "round": "prelims", "is_relay_team": false,
				 "athlete": {"first_name": "Sam", "last_name": "Jumper", "grade": null},
				 "team": {"name": "Salem Hills", "abbr": "SH"},
				 "mark": {"display": null}, "invalid_status": "dnf", "place": null,
				 "future_field": "surprise"}
			],
			"links": {"next": null}
		}`)
	})
	mux.HandleFunc("/public-api/v1/results/9", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": 9, "meet_id": 4821, "event_id": 3, "base_event_id": 2,
			"is_relay_team": false, "sport": "track", "gender": "male", "level": "high_school",
			"round": "finals", "at": "2026-06-01", "is_indoor": false, "timing_type": "fat",
			"athlete": {"first_name": "Pat", "last_name": "Runner", "grade": 11},
			"team": {"name": "Salem Hills", "abbr": "SH"},
			"meet": {"name": "State Championship"},
			"event": {"name": "100 Meter", "name_with_gender": "Boys 100 Meter"},
			"division": {"name": "Division 1", "abbr": "D1"},
			"mark": {"value": "12.34", "type": "time", "display": "12.34", "english": "12.34", "metric": "12.34"},
			"wind": 1.2, "is_legal": true, "invalid_status": null,
			"place": 1, "flight_place": 2, "flight_group_place": 3,
			"points": 10.5, "is_official": true, "is_valid": true,
			"future_field": "surprise"}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &requests, &lastQuery
}

func TestResultListRequiresAnchor(t *testing.T) {
	setupCLITest(t)
	srv, requests, _ := stubResultsAPI(t)

	_, _, err := runCommand(t, "--api-url", srv.URL, "result", "list")
	if err == nil || !strings.Contains(err.Error(), "--meet, --athlete, or --team") {
		t.Fatalf("want anchor error naming the flags, got %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("anchor validation must happen before any API request, got %d", requests.Load())
	}
}

func TestResultListPipedTSV(t *testing.T) {
	setupCLITest(t)
	srv, _, _ := stubResultsAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "result", "list", "--meet", "4821")
	if err != nil {
		t.Fatalf("result list: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 rows, got %d: %q", len(lines), out)
	}
	if lines[0] != "11\t2026-06-01\tState Championship\tBoys 100 Meter\tPat Runner (grade 11)\tSH\tV\tfinals\t12.34\t1" {
		t.Fatalf("row = %q", lines[0])
	}
	// Relay rows show the relay team name in the athlete column.
	if !strings.Contains(lines[1], "Salem Hills A") {
		t.Fatalf("relay row must use relay_team_name: %q", lines[1])
	}
	// Markless results show the status code in the outcome column.
	if cols := strings.Split(lines[2], "\t"); len(cols) < 9 || cols[8] != "DNF" {
		t.Fatalf("DNF row must show the outcome status code: %q", lines[2])
	}
}

func TestResultListJSONPassesServerFieldsThrough(t *testing.T) {
	setupCLITest(t)
	srv, _, _ := stubResultsAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "result", "list", "--meet", "4821", "--json")
	if err != nil {
		t.Fatalf("result list --json: %v", err)
	}
	var results []map[string]any
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) != 3 || results[0]["future_field"] != "surprise" {
		t.Fatalf("unknown server fields must pass through: %v", results)
	}
}

func TestResultListLimitStopsPaging(t *testing.T) {
	setupCLITest(t)
	srv, requests, lastQuery := stubResultsAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "result", "list", "--meet", "4821", "--limit", "2")
	if err != nil {
		t.Fatalf("result list --limit 2: %v", err)
	}
	if n := len(strings.Split(strings.TrimSpace(out), "\n")); n != 2 {
		t.Fatalf("want 2 rows, got %d", n)
	}
	if requests.Load() != 1 {
		t.Fatalf("limit satisfied by page 1; want 1 request, got %d", requests.Load())
	}
	q := lastQuery.Load().(url.Values)
	if got := q.Get("per_page"); got != "2" {
		t.Fatalf("per_page should match remaining limit, got %q", got)
	}
}

func TestResultListFilterMapping(t *testing.T) {
	setupCLITest(t)
	srv, _, lastQuery := stubResultsAPI(t)

	_, _, err := runCommand(t, "--api-url", srv.URL, "result", "list",
		"--meet", "4821", "--athlete", "992", "--event", "3",
		"--gender", "male", "--sport", "track", "--level", "high_school", "--round", "finals",
		"--official", "true", "--relay", "false",
		"--from", "2026-01-01", "--to", "2026-12-31", "--sort", "-at")
	if err != nil {
		t.Fatalf("result list with filters: %v", err)
	}
	q := lastQuery.Load().(url.Values)
	for param, want := range map[string]string{
		"filter[meet_id]":       "4821",
		"filter[athlete_id]":    "992",
		"filter[event_id]":      "3",
		"filter[gender]":        "male",
		"filter[sport]":         "track",
		"filter[level]":         "high_school",
		"filter[round]":         "finals",
		"filter[is_official]":   "1", // booleans map to 1/0, not true/false
		"filter[is_relay_team]": "0",
		"filter[from]":          "2026-01-01",
		"filter[to]":            "2026-12-31",
		"sort":                  "-at",
	} {
		if got := q.Get(param); got != want {
			t.Fatalf("%s = %q, want %q", param, got, want)
		}
	}
}

// The server's Sport enum is track, xc, road — client-side validation
// must not be narrower than the server's, or it locks users out of real
// data (road meets exist and the API filters on them).
func TestResultListAcceptsEveryServerSport(t *testing.T) {
	setupCLITest(t)
	srv, _, queries := stubResultsAPI(t)

	for _, sport := range []string{"track", "xc", "road"} {
		if _, _, err := runCommand(t, "--api-url", srv.URL, "result", "list",
			"--meet", "4821", "--sport", sport); err != nil {
			t.Fatalf("--sport %s must be accepted: %v", sport, err)
		}
		q, _ := queries.Load().(url.Values)
		if got := q.Get("filter[sport]"); got != sport {
			t.Fatalf("filter[sport] = %q, want %q", got, sport)
		}
	}
}

func TestResultListRejectsInvalidEnumBeforeRequest(t *testing.T) {
	setupCLITest(t)
	srv, requests, _ := stubResultsAPI(t)

	for flag, bad := range map[string]string{
		"gender": "men", "sport": "swimming", "level": "hs", "round": "semis",
		"official": "yes", "sort": "name",
	} {
		_, _, err := runCommand(t, "--api-url", srv.URL, "result", "list", "--meet", "4821", "--"+flag, bad)
		if err == nil || !strings.Contains(err.Error(), "must be one of") {
			t.Fatalf("--%s %q must be rejected with valid values listed, got %v", flag, bad, err)
		}
	}
	if requests.Load() != 0 {
		t.Fatalf("enum validation must happen before any API request, got %d", requests.Load())
	}
}

func TestResultViewTSVAndJSON(t *testing.T) {
	setupCLITest(t)
	srv, _, _ := stubResultsAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "result", "view", "9")
	if err != nil {
		t.Fatalf("result view: %v", err)
	}
	for _, want := range []string{
		"Athlete\tPat Runner (grade 11)",
		"Team\tSalem Hills (SH)",
		"Meet\tState Championship",
		"Event\tBoys 100 Meter",
		"Mark display\t12.34",
		"Mark english\t12.34",
		"Mark metric\t12.34",
		"Wind\t1.2",
		"Legal\ttrue",
		"Place\t1",
		"Flight place\t2",
		"Flight group place\t3",
		"Points\t10.5",
		"Official\ttrue",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("view missing %q:\n%s", want, out)
		}
	}

	out, _, err = runCommand(t, "--api-url", srv.URL, "result", "view", "9", "--json")
	if err != nil {
		t.Fatalf("result view --json: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["future_field"] != "surprise" {
		t.Fatalf("unknown server fields must pass through: %v", result)
	}
}

func TestResultListRequiresLogin(t *testing.T) {
	setupCLITest(t)
	t.Setenv("SPORTTRAX_API_TOKEN", "")
	srv, _, _ := stubResultsAPI(t)

	_, _, err := runCommand(t, "--api-url", srv.URL, "result", "list", "--meet", "4821")
	if err == nil || !strings.Contains(err.Error(), "auth login") {
		t.Fatalf("want login-hinting error, got %v", err)
	}
}

func TestResultListMarkUnits(t *testing.T) {
	setupCLITest(t)
	srv, _, _ := stubResultsAPI(t)

	// --units selects the conversion; rows without one fall back to the
	// native display form.
	out, _, err := runCommand(t, "--api-url", srv.URL, "result", "list", "--meet", "4821", "--units", "english")
	if err != nil {
		t.Fatalf("result list --units english: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if cols := strings.Split(lines[0], "\t"); cols[8] != "12.34e" {
		t.Fatalf("english units not selected: %q", lines[0])
	}
	if cols := strings.Split(lines[1], "\t"); cols[8] != "42.10" {
		t.Fatalf("missing conversion must fall back to display: %q", lines[1])
	}

	_, _, err = runCommand(t, "--api-url", srv.URL, "result", "list", "--meet", "4821", "--units", "imperial")
	if err == nil || !strings.Contains(err.Error(), "english or metric") {
		t.Fatalf("invalid units must be rejected, got %v", err)
	}

	// units from config.yaml apply without the flag.
	cfgDir := os.Getenv("XDG_CONFIG_HOME") + "/sporttrax-cli"
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgDir+"/config.yaml", []byte("units: metric\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, err = runCommand(t, "--api-url", srv.URL, "result", "list", "--meet", "4821")
	if err != nil {
		t.Fatalf("result list with config units: %v", err)
	}
	lines = strings.Split(strings.TrimSpace(out), "\n")
	if cols := strings.Split(lines[0], "\t"); cols[8] != "12.34m" {
		t.Fatalf("config units not applied: %q", lines[0])
	}
}
