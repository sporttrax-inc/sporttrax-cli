package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sporttrax-inc/sporttrax-cli/internal/auth"
)

// stubDirectoryAPI serves the show-only athlete and team endpoints.
func stubDirectoryAPI(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/public-api/v1/athletes/992", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": 992, "first_name": "Maya", "last_name": "Rivera",
			"gender": "female", "hs_graduation_year": 2027, "future_field": "surprise"}`)
	})
	mux.HandleFunc("/public-api/v1/teams/55", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": 55, "name": "Salem Hills", "display_name": "Salem Hills HS",
			"abbr": "SHHS", "level": "high_school", "sport": "track", "team_type": "school",
			"city": "Salem", "state_code": "UT", "future_field": "surprise"}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestAthleteView(t *testing.T) {
	setupCLITest(t)
	srv := stubDirectoryAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "athlete", "view", "992")
	if err != nil {
		t.Fatalf("athlete view: %v", err)
	}
	for _, want := range []string{"ID\t992", "First name\tMaya", "Last name\tRivera", "Gender\tfemale", "HS graduation year\t2027"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestTeamView(t *testing.T) {
	setupCLITest(t)
	srv := stubDirectoryAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "team", "view", "55")
	if err != nil {
		t.Fatalf("team view: %v", err)
	}
	for _, want := range []string{"Name\tSalem Hills", "Display name\tSalem Hills HS", "Abbr\tSHHS", "State code\tUT"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestDirectoryJSONPassthrough(t *testing.T) {
	setupCLITest(t)
	srv := stubDirectoryAPI(t)

	for _, tc := range []struct{ args []string }{
		{[]string{"athlete", "view", "992"}},
		{[]string{"team", "view", "55"}},
	} {
		out, _, err := runCommand(t, append([]string{"--api-url", srv.URL}, append(tc.args, "--json")...)...)
		if err != nil {
			t.Fatalf("%v --json: %v", tc.args, err)
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(out), &rec); err != nil {
			t.Fatalf("not JSON: %v", err)
		}
		if rec["future_field"] != "surprise" {
			t.Errorf("%v: server fields must pass through verbatim: %v", tc.args, rec)
		}
	}
}

func TestDirectoryCommandsRequireLogin(t *testing.T) {
	setupCLITest(t)
	t.Setenv("SPORTTRAX_API_TOKEN", "")
	srv := stubDirectoryAPI(t)

	for _, args := range [][]string{{"athlete", "view", "992"}, {"team", "view", "55"}} {
		_, _, err := runCommand(t, append([]string{"--api-url", srv.URL}, args...)...)
		if !errors.Is(err, auth.ErrNotFound) {
			t.Errorf("%v without a token should report ErrNotFound, got %v", args, err)
		}
	}
}
