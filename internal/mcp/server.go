// Package mcp exposes the SportTrax public API as Model Context Protocol
// tools over stdio. Tool handlers share the typed internal/api client with
// the CLI commands — one code path for auth, retries, and headers.
//
// Parity rule (see CLAUDE.md): every data capability the CLI gains must be
// registered here as a tool in the same change, with a schema and a
// description good enough for an AI to use the tool without trial and
// error.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
)

// NewServer builds the MCP server with all SportTrax tools registered.
func NewServer(client *api.Client, version string) *sdk.Server {
	s := sdk.NewServer(&sdk.Implementation{Name: "sporttrax", Version: version}, nil)
	registerWhoamiTool(s, client)
	registerMeetTools(s, client)
	registerResultTools(s, client)
	registerEventTools(s, client)
	registerBaseEventTools(s, client)
	registerAthleteTools(s, client)
	registerTeamTools(s, client)
	return s
}

const (
	defaultToolLimit = 25
	maxToolLimit     = 100
)

// boundedLimit keeps an AI caller's page size sane: unset means a small
// default, and no caller can ask for more than the server's own cap.
func boundedLimit(limit int) int {
	if limit <= 0 {
		return defaultToolLimit
	}
	if limit > maxToolLimit {
		return maxToolLimit
	}
	return limit
}

type whoamiArgs struct{}

func registerWhoamiTool(s *sdk.Server, client *api.Client) {
	sdk.AddTool(s, &sdk.Tool{
		Name: "whoami",
		Description: "Return the authenticated SportTrax user (id, name, role), the token's " +
			"name and abilities, an access block, and the effective rate limits " +
			"(null = unlimited). Call this first to learn who you are and what " +
			"access applies before using other tools.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, args whoamiArgs) (*sdk.CallToolResult, any, error) {
		_, raw, err := client.MeInfo(ctx)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(raw)
	})
}

// Run serves MCP over stdio until ctx is cancelled.
func Run(ctx context.Context, client *api.Client, version string) error {
	return NewServer(client, version).Run(ctx, &sdk.StdioTransport{})
}

type listMeetsArgs struct {
	Sport string `json:"sport,omitempty" jsonschema:"sport filter: track (track & field), xc (cross country), or road (road racing)"`
	Name  string `json:"name,omitempty" jsonschema:"partial meet name filter"`
	From  string `json:"from,omitempty" jsonschema:"only meets starting on or after this date (YYYY-MM-DD)"`
	To    string `json:"to,omitempty" jsonschema:"only meets starting on or before this date (YYYY-MM-DD)"`
	State string `json:"state,omitempty" jsonschema:"two-letter venue state code, e.g. ID"`
	City  string `json:"city,omitempty" jsonschema:"venue city filter; state is required when city is set"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum meets to return; default 25, max 100 — keep small"`
}

// listMeetsSchema is the inferred schema tightened with value constraints
// so AI callers can't send invalid filter values.
func listMeetsSchema() (*jsonschema.Schema, error) {
	schema, err := jsonschema.For[listMeetsArgs](nil)
	if err != nil {
		return nil, err
	}
	schema.Properties["sport"].Enum = enumOf(api.Sports...)
	schema.Properties["state"].Pattern = "^[A-Za-z]{2}$"
	return schema, nil
}

type getMeetArgs struct {
	ID int64 `json:"id" jsonschema:"numeric meet ID"`
}

func registerMeetTools(s *sdk.Server, client *api.Client) {
	meetsSchema, err := listMeetsSchema()
	if err != nil {
		panic(err) // static schema construction; fails only on a programming error
	}
	sdk.AddTool(s, &sdk.Tool{
		Name:        "list_meets",
		InputSchema: meetsSchema,
		Description: "List published SportTrax meets, newest first. Returns {data, count, has_more}: " +
			"data is an array of meet records (id, name, sport, status, is_sanctioned, timezone, " +
			"first/last_session_starting_at, venue (name, city, state_code)); when has_more is true " +
			"the window was truncated at limit — raise limit before summarizing as complete. " +
			"Use get_meet for a single known ID.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, args listMeetsArgs) (*sdk.CallToolResult, any, error) {
		if args.City != "" && args.State == "" {
			return nil, nil, fmt.Errorf("state is required when city is set (city names collide across states)")
		}
		query := url.Values{}
		for param, v := range map[string]string{
			"filter[sport]": args.Sport,
			"filter[name]":  args.Name,
			"filter[from]":  args.From,
			"filter[to]":    args.To,
			"filter[state]": args.State,
			"filter[city]":  args.City,
		} {
			if v != "" {
				query.Set(param, v)
			}
		}
		res, err := client.List(ctx, "/meets", query, boundedLimit(args.Limit))
		if err != nil {
			return nil, nil, err
		}
		return listResult(res)
	})

	sdk.AddTool(s, &sdk.Tool{
		Name: "get_meet",
		Description: "Fetch one SportTrax meet by its numeric ID. Returns the full meet " +
			"record as JSON, including venue and session times.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, args getMeetArgs) (*sdk.CallToolResult, any, error) {
		raw, err := client.GetRaw(ctx, "/meets/"+strconv.FormatInt(args.ID, 10))
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(raw)
	})
}

// jsonResult returns v marshalled as compact JSON text content — the
// server's records pass through verbatim, same contract as --json.
func jsonResult(v any) (*sdk.CallToolResult, any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, nil, err
	}
	return &sdk.CallToolResult{
		Content: []sdk.Content{&sdk.TextContent{Text: string(data)}},
	}, nil, nil
}

// listResult wraps a collection window as {data, count, has_more} so AI
// callers know when they received a truncated window rather than
// everything — records inside data still pass through verbatim.
func listResult(res api.ListResult) (*sdk.CallToolResult, any, error) {
	return jsonResult(map[string]any{
		"data":     res.Items,
		"count":    len(res.Items),
		"has_more": res.HasMore,
	})
}
