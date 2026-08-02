package cli

import (
	"encoding/csv"
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

// stubAPI serves two pages of meets plus a single-meet endpoint,
// mimicking the server's cursor pagination and envelope. Payloads include
// a future_field the CLI doesn't know about, to prove --json passthrough.
func TestMeetListCSV(t *testing.T) {
	setupCLITest(t)
	srv, _, _ := stubAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "meet", "list", "--csv")
	if err != nil {
		t.Fatalf("meet list --csv: %v", err)
	}
	rows, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil {
		t.Fatalf("not valid CSV: %v\n%s", err, out)
	}
	if len(rows) != 4 { // header + 3 meets
		t.Fatalf("want header + 3 rows, got %d: %v", len(rows), rows)
	}
	if rows[0][0] != "ID" || rows[0][1] != "NAME" {
		t.Fatalf("CSV must keep its header row, got %v", rows[0])
	}
	if rows[1][1] != "State Championship" {
		t.Fatalf("row data wrong: %v", rows[1])
	}
	// A venue carries a comma; the reader proves it survived quoting.
	if rows[1][5] != "Memorial Stadium — Boise, ID" {
		t.Fatalf("comma-bearing value lost: %q", rows[1][5])
	}
}

func TestMeetViewCSV(t *testing.T) {
	setupCLITest(t)
	srv, _, _ := stubAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "meet", "view", "7", "--csv")
	if err != nil {
		t.Fatalf("meet view --csv: %v", err)
	}
	rows, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil {
		t.Fatalf("not valid CSV: %v\n%s", err, out)
	}
	if rows[0][0] != "field" || rows[0][1] != "value" {
		t.Fatalf("detail CSV needs a field/value header, got %v", rows[0])
	}
	if rows[1][0] != "ID" || rows[1][1] != "7" {
		t.Fatalf("first pair wrong: %v", rows[1])
	}
}

// A stray argument used to be accepted and ignored, which silently gave
// the wrong answer to `meet list --json name` — what a gh user types,
// since gh's --json takes fields.
func TestListCommandsRejectStrayArguments(t *testing.T) {
	setupCLITest(t)
	srv, requests, _ := stubAPI(t)

	for _, args := range [][]string{
		{"meet", "list", "stray"},
		{"result", "list", "--meet", "1", "stray"},
		{"event", "list", "stray"},
		{"base-event", "list", "stray"},
		{"env", "list", "stray"},
		{"version", "stray"},
	} {
		_, _, err := runCommand(t, append([]string{"--api-url", srv.URL}, args...)...)
		if err == nil || !strings.Contains(err.Error(), "unknown command") && !strings.Contains(err.Error(), "accepts") {
			t.Errorf("%v should reject the stray arg, got %v", args, err)
		}
	}
	if requests.Load() != 0 {
		t.Fatalf("argument validation must precede any request, got %d", requests.Load())
	}
}

// meet list validated neither filter, while result/event/base-event and
// the list_meets MCP tool all did.
func TestMeetListValidatesFiltersBeforeRequest(t *testing.T) {
	setupCLITest(t)
	srv, requests, _ := stubAPI(t)

	_, _, err := runCommand(t, "--api-url", srv.URL, "meet", "list", "--sport", "swimming")
	if err == nil || !strings.Contains(err.Error(), "must be one of") {
		t.Fatalf("invalid --sport must be rejected locally, got %v", err)
	}
	_, _, err = runCommand(t, "--api-url", srv.URL, "meet", "list", "--state", "IDAHO")
	if err == nil || !strings.Contains(err.Error(), "two-letter") {
		t.Fatalf("invalid --state must be rejected locally, got %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("validation must precede any request, got %d", requests.Load())
	}

	// Valid values still reach the API.
	if _, _, err := runCommand(t, "--api-url", srv.URL, "meet", "list", "--sport", "road", "--state", "id"); err != nil {
		t.Fatalf("valid filters must be accepted: %v", err)
	}
}

// --units is scoped to the one command that renders a mark as a single
// value; anywhere else it would be silently ignored.
func TestUnitsIsScopedToResultList(t *testing.T) {
	setupCLITest(t)
	srv, _, _ := stubAPI(t)

	if _, _, err := runCommand(t, "--api-url", srv.URL, "meet", "list", "--units", "metric"); err == nil {
		t.Fatal("--units on meet list should be an unknown flag, not silently ignored")
	}
}

func TestVersionSupportsStructuredOutput(t *testing.T) {
	setupCLITest(t)

	out, _, err := runCommand(t, "version", "--json")
	if err != nil {
		t.Fatalf("version --json: %v", err)
	}
	var info map[string]string
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	for _, k := range []string{"version", "commit", "os", "arch"} {
		if _, ok := info[k]; !ok {
			t.Errorf("missing %q in %v", k, info)
		}
	}

	out, _, err = runCommand(t, "version", "--csv")
	if err != nil {
		t.Fatalf("version --csv: %v", err)
	}
	if !strings.HasPrefix(out, "field,value\n") {
		t.Fatalf("version --csv = %q", out)
	}
}

func TestCSVAndJSONAreMutuallyExclusive(t *testing.T) {
	setupCLITest(t)
	srv, _, _ := stubAPI(t)

	_, _, err := runCommand(t, "--api-url", srv.URL, "meet", "list", "--csv", "--json")
	if err == nil || !strings.Contains(err.Error(), "none of the others can be") {
		t.Fatalf("--csv with --json must be rejected, got %v", err)
	}
}

// The API emits session times as "2006-01-02 15:04:05" while other
// timestamps are RFC3339; list columns show the date for both, and
// anything unrecognized passes through as-is.
func TestDateOfAcceptsServerTimestampFormats(t *testing.T) {
	for in, want := range map[string]string{
		"2026-06-01T09:00:00-06:00": "2026-06-01",
		"2026-08-08 16:31:00":       "2026-08-08",
		"2026-08-08":                "2026-08-08",
		"":                          "",
		"not a timestamp":           "not a timestamp",
	} {
		if got := dateOf(in); got != want {
			t.Errorf("dateOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func stubAPI(t *testing.T) (*httptest.Server, *atomic.Int32, *atomic.Value) {
	t.Helper()
	var requests atomic.Int32
	var lastQuery atomic.Value
	mux := http.NewServeMux()
	mux.HandleFunc("/public-api/v1/meets", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		lastQuery.Store(r.URL.Query())
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("cursor") == "" {
			fmt.Fprint(w, `{
				"data": [
					{"id": 1, "name": "State Championship", "sport": "track", "status": "completed",
					 "first_session_starting_at": "2026-06-01T09:00:00-06:00",
					 "venue": {"name": "Memorial Stadium", "city": "Boise", "state_code": "ID"},
					 "future_field": "surprise"},
					{"id": 2, "name": "City Invite", "sport": "track", "status": "completed",
					 "first_session_starting_at": "2026-05-20T09:00:00-06:00", "venue": null,
					 "future_field": "surprise"}
				],
				"links": {"next": "/public-api/v1/meets?cursor=page2"}
			}`)
			return
		}
		fmt.Fprint(w, `{
			"data": [
				{"id": 3, "name": "Twilight Meet", "sport": "track", "status": "published",
				 "first_session_starting_at": "2026-07-04T18:00:00-06:00", "venue": null,
				 "future_field": "surprise"}
			],
			"links": {"next": null}
		}`)
	})
	mux.HandleFunc("/public-api/v1/meets/7", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": 7, "name": "Regional Final", "sport": "track",
			"status": "published", "is_sanctioned": true,
			"timezone": "America/Boise",
			"first_session_starting_at": "2026-07-10T10:00:00-06:00",
			"venue": {"name": "Dona Larsen Park", "city": "Boise", "state_code": "ID"},
			"future_field": "surprise"}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &requests, &lastQuery
}

func TestMeetListPipedTSV(t *testing.T) {
	setupCLITest(t)
	srv, _, _ := stubAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "meet", "list")
	if err != nil {
		t.Fatalf("meet list: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 rows, got %d: %q", len(lines), out)
	}
	if lines[0] != "1\tState Championship\ttrack\tcompleted\t2026-06-01\tMemorial Stadium — Boise, ID" {
		t.Fatalf("row = %q", lines[0])
	}
}

func TestMeetListJSONPassesServerFieldsThrough(t *testing.T) {
	setupCLITest(t)
	srv, _, _ := stubAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "meet", "list", "--json")
	if err != nil {
		t.Fatalf("meet list --json: %v", err)
	}
	var meets []map[string]any
	if err := json.Unmarshal([]byte(out), &meets); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(meets) != 3 || meets[0]["future_field"] != "surprise" {
		t.Fatalf("unknown server fields must pass through: %v", meets)
	}
}

func TestMeetListLimitStopsPaging(t *testing.T) {
	setupCLITest(t)
	srv, requests, lastQuery := stubAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "meet", "list", "--limit", "2")
	if err != nil {
		t.Fatalf("meet list --limit 2: %v", err)
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

func TestMeetListFilterMapping(t *testing.T) {
	setupCLITest(t)
	srv, _, lastQuery := stubAPI(t)

	_, _, err := runCommand(t, "--api-url", srv.URL, "meet", "list",
		"--sport", "track", "--name", "state", "--from", "2026-01-01",
		"--state", "ID", "--city", "Boise")
	if err != nil {
		t.Fatalf("meet list with filters: %v", err)
	}
	q := lastQuery.Load().(url.Values)
	for param, want := range map[string]string{
		"filter[sport]": "track",
		"filter[name]":  "state",
		"filter[from]":  "2026-01-01",
		"filter[state]": "ID",
		"filter[city]":  "Boise",
	} {
		if got := q.Get(param); got != want {
			t.Fatalf("%s = %q, want %q", param, got, want)
		}
	}
}

func TestMeetListCityRequiresState(t *testing.T) {
	setupCLITest(t)
	srv, requests, _ := stubAPI(t)

	_, _, err := runCommand(t, "--api-url", srv.URL, "meet", "list", "--city", "Boise")
	if err == nil || !strings.Contains(err.Error(), "--state is required") {
		t.Fatalf("want state-required error, got %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("validation must happen before any API request, got %d", requests.Load())
	}
}

func TestMeetViewTSVAndJSON(t *testing.T) {
	setupCLITest(t)
	srv, _, _ := stubAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "meet", "view", "7")
	if err != nil {
		t.Fatalf("meet view: %v", err)
	}
	if !strings.Contains(out, "Name\tRegional Final") || !strings.Contains(out, "Venue\tDona Larsen Park — Boise, ID") {
		t.Fatalf("unexpected view output:\n%s", out)
	}
	if !strings.Contains(out, "Sanctioned\ttrue") {
		t.Fatalf("booleans must render true/false:\n%s", out)
	}

	out, _, err = runCommand(t, "--api-url", srv.URL, "meet", "view", "7", "--json")
	if err != nil {
		t.Fatalf("meet view --json: %v", err)
	}
	var meet map[string]any
	if err := json.Unmarshal([]byte(out), &meet); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if meet["future_field"] != "surprise" {
		t.Fatalf("unknown server fields must pass through: %v", meet)
	}
}

func TestMeetViewRejectsNonNumericID(t *testing.T) {
	setupCLITest(t)
	_, _, err := runCommand(t, "meet", "view", "abc")
	if err == nil || !strings.Contains(err.Error(), "must be a number") {
		t.Fatalf("want numeric-id error, got %v", err)
	}
}

func TestMeetListInvalidLimit(t *testing.T) {
	setupCLITest(t)
	_, _, err := runCommand(t, "meet", "list", "--limit", "zero")
	if err == nil || !strings.Contains(err.Error(), "--limit") {
		t.Fatalf("want limit error, got %v", err)
	}
}

func TestMeetListRequiresLogin(t *testing.T) {
	setupCLITest(t)
	t.Setenv("SPORTTRAX_API_TOKEN", "")
	srv, _, _ := stubAPI(t)

	_, _, err := runCommand(t, "--api-url", srv.URL, "meet", "list")
	if !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("want auth.ErrNotFound (exit code 4 path), got %v", err)
	}
}
