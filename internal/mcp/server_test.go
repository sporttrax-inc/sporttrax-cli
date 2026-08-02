package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
)

// connect wires the MCP server (backed by a stub SportTrax API) to an
// in-memory MCP client, returning the client session.
func connect(t *testing.T, apiURL string) *sdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	server := NewServer(api.New(apiURL, "test-token", false), "test")
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func stubAPI(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var queries []string
	mux := http.NewServeMux()
	mux.HandleFunc("/public-api/v1/meets", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data": [
			{"id": 1, "name": "State Championship", "sport": "track", "future_field": "surprise"}
		], "links": {"next": null}}`)
	})
	mux.HandleFunc("/public-api/v1/meets/7", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": 7, "name": "Regional Final", "future_field": "surprise"}`)
	})
	mux.HandleFunc("/public-api/v1/results", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data": [
			{"id": 11, "meet_id": 4821, "mark": {"display": "12.34"}, "future_field": "surprise"}
		], "links": {"next": null}}`)
	})
	mux.HandleFunc("/public-api/v1/results/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": 9, "mark": {"display": "12.34"}, "future_field": "surprise"}`)
	})
	mux.HandleFunc("/public-api/v1/events", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data": [
			{"id": 1, "constant": "female-100-meter-hurdles", "name": "100m Hurdles",
			 "gender": "female", "base_event_id": 1, "future_field": "surprise"}
		], "links": {"next": null}}`)
	})
	mux.HandleFunc("/public-api/v1/events/1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": 1, "name": "100m Hurdles", "future_field": "surprise"}`)
	})
	mux.HandleFunc("/public-api/v1/base-events", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data": [
			{"id": 1, "constant": "100-meter-hurdles", "sport": "track", "mark_type": "time",
			 "tfrrs_code": "100h", "is_hurdles": true, "future_field": "surprise"}
		], "links": {"next": null}}`)
	})
	mux.HandleFunc("/public-api/v1/base-events/1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": 1, "constant": "100-meter-hurdles", "tfrrs_code": "100h",
			"future_field": "surprise"}`)
	})
	mux.HandleFunc("/public-api/v1/athletes/992", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": 992, "first_name": "Maya", "last_name": "Rivera", "future_field": "surprise"}`)
	})
	mux.HandleFunc("/public-api/v1/teams/55", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": 55, "name": "Salem Hills", "abbr": "SHHS", "future_field": "surprise"}`)
	})
	mux.HandleFunc("/public-api/v1/me", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"user": {"id": 42, "name": "Jeff Hansen", "role": "admin"},
			"token": {"name": "cli", "abilities": ["public-api"], "last_used_at": null},
			"access": {"public_api": true, "reason": null},
			"rate_limit": {"per_minute": null, "per_day": null},
			"future_field": "surprise"}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &queries
}

func textOf(t *testing.T, res *sdk.CallToolResult) string {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %v", res.Content)
	}
	if len(res.Content) != 1 {
		t.Fatalf("want 1 content block, got %d", len(res.Content))
	}
	text, ok := res.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatalf("want text content, got %T", res.Content[0])
	}
	return text.Text
}

func TestToolsAreRegisteredWithSchemas(t *testing.T) {
	srv, _ := stubAPI(t)
	session := connect(t, srv.URL)

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	byName := map[string]*sdk.Tool{}
	for _, tool := range tools.Tools {
		byName[tool.Name] = tool
	}
	// Parity: every data capability the CLI has must be reachable here.
	want := []string{
		"whoami",
		"list_meets", "get_meet",
		"list_results", "get_result",
		"list_events", "get_event",
		"list_base_events", "get_base_event",
		"get_athlete", "get_team",
	}
	for _, name := range want {
		tool, ok := byName[name]
		if !ok {
			t.Fatalf("tool %q not registered (have %v)", name, tools.Tools)
		}
		if tool.Description == "" || tool.InputSchema == nil {
			t.Fatalf("tool %q must have a description and input schema", name)
		}
	}
	if len(byName) != len(want) {
		t.Fatalf("registered %d tools, expected exactly %d — update this list when adding one", len(byName), len(want))
	}
}

func TestCatalogToolsFilterAndPassThrough(t *testing.T) {
	srv, queries := stubAPI(t)
	session := connect(t, srv.URL)

	cases := []struct {
		tool      string
		args      map[string]any
		wantQuery []string
	}{
		{
			"list_events",
			map[string]any{"gender": "female", "base_event_id": 1, "limit": 5},
			[]string{"filter%5Bgender%5D=female", "filter%5Bbase_event_id%5D=1", "per_page=5"},
		},
		{
			"list_base_events",
			map[string]any{"sport": "track", "mark_type": "time"},
			[]string{"filter%5Bsport%5D=track", "filter%5Bmark_type%5D=time"},
		},
	}
	for _, tc := range cases {
		*queries = nil
		res, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: tc.tool, Arguments: tc.args})
		if err != nil {
			t.Fatalf("%s: %v", tc.tool, err)
		}
		var envelope struct {
			Data    []map[string]any `json:"data"`
			Count   int              `json:"count"`
			HasMore bool             `json:"has_more"`
		}
		if err := json.Unmarshal([]byte(textOf(t, res)), &envelope); err != nil {
			t.Fatalf("%s result is not JSON: %v", tc.tool, err)
		}
		if envelope.Count != 1 || envelope.Data[0]["future_field"] != "surprise" {
			t.Fatalf("%s must pass server fields through verbatim: %v", tc.tool, envelope)
		}
		if len(*queries) != 1 {
			t.Fatalf("%s: want 1 request, got %d", tc.tool, len(*queries))
		}
		for _, want := range tc.wantQuery {
			if !strings.Contains((*queries)[0], want) {
				t.Errorf("%s: missing %q in %q", tc.tool, want, (*queries)[0])
			}
		}
	}
}

func TestCatalogToolsRejectInvalidEnums(t *testing.T) {
	srv, queries := stubAPI(t)
	session := connect(t, srv.URL)

	for _, tc := range []struct {
		tool string
		args map[string]any
	}{
		{"list_events", map[string]any{"gender": "women"}},
		{"list_base_events", map[string]any{"sport": "swimming"}},
		{"list_base_events", map[string]any{"mark_type": "seconds"}},
	} {
		*queries = nil
		res, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: tc.tool, Arguments: tc.args})
		if err == nil && !res.IsError {
			t.Errorf("%s %v must be rejected by the schema enum", tc.tool, tc.args)
		}
		if len(*queries) != 0 {
			t.Errorf("%s: schema validation must precede any request, got %d", tc.tool, len(*queries))
		}
	}
}

func TestDirectoryTools(t *testing.T) {
	srv, _ := stubAPI(t)
	session := connect(t, srv.URL)

	for _, tc := range []struct {
		tool string
		id   int
		want string
	}{
		{"get_athlete", 992, "Maya"},
		{"get_team", 55, "Salem Hills"},
	} {
		res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
			Name: tc.tool, Arguments: map[string]any{"id": tc.id},
		})
		if err != nil {
			t.Fatalf("%s: %v", tc.tool, err)
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(textOf(t, res)), &rec); err != nil {
			t.Fatalf("%s result is not JSON: %v", tc.tool, err)
		}
		if rec["future_field"] != "surprise" {
			t.Errorf("%s must pass server fields through verbatim: %v", tc.tool, rec)
		}
		if !strings.Contains(fmt.Sprint(rec), tc.want) {
			t.Errorf("%s missing %q: %v", tc.tool, tc.want, rec)
		}
	}
}

// The CLI hides the TFRRS/Hy-Tek integration codes, but MCP results are
// raw passthrough by contract, so they must still arrive intact.
func TestBaseEventToolKeepsIntegrationCodes(t *testing.T) {
	srv, _ := stubAPI(t)
	session := connect(t, srv.URL)

	res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name: "get_base_event", Arguments: map[string]any{"id": 1},
	})
	if err != nil {
		t.Fatalf("get_base_event: %v", err)
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(textOf(t, res)), &rec); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if rec["tfrrs_code"] != "100h" {
		t.Fatalf("raw passthrough must keep tfrrs_code: %v", rec)
	}
}

func TestListMeetsToolFiltersAndPassthrough(t *testing.T) {
	srv, queries := stubAPI(t)
	session := connect(t, srv.URL)

	res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      "list_meets",
		Arguments: map[string]any{"sport": "track", "state": "ID", "city": "Boise", "limit": 5},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	var envelope struct {
		Data    []map[string]any `json:"data"`
		Count   int              `json:"count"`
		HasMore bool             `json:"has_more"`
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &envelope); err != nil {
		t.Fatalf("tool result is not JSON: %v", err)
	}
	if envelope.Count != 1 || len(envelope.Data) != 1 || envelope.Data[0]["future_field"] != "surprise" {
		t.Fatalf("server fields must pass through verbatim inside data: %v", envelope)
	}
	if envelope.HasMore {
		t.Fatal("single exhausted page must report has_more=false")
	}
	if len(*queries) != 1 {
		t.Fatalf("want 1 API request, got %d", len(*queries))
	}
	q := (*queries)[0]
	for _, want := range []string{
		"filter%5Bsport%5D=track",
		"filter%5Bstate%5D=ID",
		"filter%5Bcity%5D=Boise",
		"per_page=5",
	} {
		if !strings.Contains(q, want) {
			t.Fatalf("filters/limit not mapped, missing %q in %q", want, q)
		}
	}
}

func TestListMeetsToolCityRequiresState(t *testing.T) {
	srv, queries := stubAPI(t)
	session := connect(t, srv.URL)

	res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      "list_meets",
		Arguments: map[string]any{"city": "Boise"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatal("city without state must be a tool error")
	}
	if len(*queries) != 0 {
		t.Fatalf("validation must happen before any API request, got %d", len(*queries))
	}
}

func TestGetMeetTool(t *testing.T) {
	srv, _ := stubAPI(t)
	session := connect(t, srv.URL)

	res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      "get_meet",
		Arguments: map[string]any{"id": 7},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	var meet map[string]any
	if err := json.Unmarshal([]byte(textOf(t, res)), &meet); err != nil {
		t.Fatalf("tool result is not JSON: %v", err)
	}
	if meet["name"] != "Regional Final" || meet["future_field"] != "surprise" {
		t.Fatalf("unexpected meet: %v", meet)
	}
}

func TestWhoamiTool(t *testing.T) {
	srv, _ := stubAPI(t)
	session := connect(t, srv.URL)

	res, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "whoami"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	var me map[string]any
	if err := json.Unmarshal([]byte(textOf(t, res)), &me); err != nil {
		t.Fatalf("tool result is not JSON: %v", err)
	}
	user, _ := me["user"].(map[string]any)
	if user["name"] != "Jeff Hansen" || user["role"] != "admin" || me["future_field"] != "surprise" {
		t.Fatalf("whoami must pass /me through verbatim: %v", me)
	}
}

func TestListResultsToolAnchorsFiltersAndPassthrough(t *testing.T) {
	srv, queries := stubAPI(t)
	session := connect(t, srv.URL)

	// Missing anchor is rejected before any request.
	res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      "list_results",
		Arguments: map[string]any{"gender": "male"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError || len(*queries) != 0 {
		t.Fatalf("missing anchor must error with zero requests (requests=%d)", len(*queries))
	}

	// Invalid enum is rejected by the schema before any request.
	res, err = session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      "list_results",
		Arguments: map[string]any{"meet_id": 4821, "round": "semis"},
	})
	if (err == nil && !res.IsError) || len(*queries) != 0 {
		t.Fatalf("invalid round must be rejected by schema enum (requests=%d)", len(*queries))
	}

	// Valid call maps filters (booleans as 1/0) and passes JSON through.
	res, err = session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      "list_results",
		Arguments: map[string]any{"meet_id": 4821, "round": "finals", "official": true, "limit": 5},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	var envelope struct {
		Data    []map[string]any `json:"data"`
		Count   int              `json:"count"`
		HasMore bool             `json:"has_more"`
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &envelope); err != nil {
		t.Fatalf("tool result is not JSON: %v", err)
	}
	if envelope.Count != 1 || len(envelope.Data) != 1 || envelope.Data[0]["future_field"] != "surprise" {
		t.Fatalf("server fields must pass through verbatim inside data: %v", envelope)
	}
	q := (*queries)[0]
	for _, want := range []string{
		"filter%5Bmeet_id%5D=4821",
		"filter%5Bround%5D=finals",
		"filter%5Bis_official%5D=1",
		"per_page=5",
	} {
		if !strings.Contains(q, want) {
			t.Fatalf("filters not mapped, missing %q in %q", want, q)
		}
	}
}

func TestGetResultTool(t *testing.T) {
	srv, _ := stubAPI(t)
	session := connect(t, srv.URL)

	res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      "get_result",
		Arguments: map[string]any{"id": 9},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(textOf(t, res)), &result); err != nil {
		t.Fatalf("tool result is not JSON: %v", err)
	}
	if result["future_field"] != "surprise" {
		t.Fatalf("get_result must pass /results/{id} through verbatim: %v", result)
	}
}

// Road racing is one of the server's three sports (Sport::TRACK, XC,
// ROAD); the schema enum must not lock AI callers out of it.
func TestListMeetsToolAcceptsRoadSport(t *testing.T) {
	srv, queries := stubAPI(t)
	session := connect(t, srv.URL)

	res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      "list_meets",
		Arguments: map[string]any{"sport": "road"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("road must be an accepted sport: %v", res.Content)
	}
	if len(*queries) != 1 || !strings.Contains((*queries)[0], "filter%5Bsport%5D=road") {
		t.Fatalf("road filter not forwarded: %v", *queries)
	}
}

func TestListMeetsToolRejectsInvalidSport(t *testing.T) {
	srv, queries := stubAPI(t)
	session := connect(t, srv.URL)

	res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      "list_meets",
		Arguments: map[string]any{"sport": "swimming"},
	})
	// The schema enum must reject the value — either as a protocol error
	// or a tool error — without any API request being made.
	if err == nil && !res.IsError {
		t.Fatal("invalid sport value must be rejected by the schema enum")
	}
	if len(*queries) != 0 {
		t.Fatalf("schema validation must happen before any API request, got %d", len(*queries))
	}
}
