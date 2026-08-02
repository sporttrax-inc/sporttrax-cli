package mcp

import (
	"context"
	"net/url"
	"strconv"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
)

type listBaseEventsArgs struct {
	Sport    string `json:"sport,omitempty" jsonschema:"sport filter: track (track & field), xc (cross country), or road (road racing)"`
	MarkType string `json:"mark_type,omitempty" jsonschema:"mark type filter: time, distance, or score"`
	Limit    int    `json:"limit,omitempty" jsonschema:"maximum base events to return; default 25, max 100 — keep small"`
}

type getBaseEventArgs struct {
	ID int64 `json:"id" jsonschema:"numeric base event ID"`
}

// listBaseEventsSchema tightens the inferred schema with the server's
// enums so AI callers cannot send silently-no-match filter values.
func listBaseEventsSchema() (*jsonschema.Schema, error) {
	schema, err := jsonschema.For[listBaseEventsArgs](nil)
	if err != nil {
		return nil, err
	}
	schema.Properties["sport"].Enum = enumOf(api.Sports...)
	schema.Properties["mark_type"].Enum = enumOf(api.MarkTypes...)
	return schema, nil
}

func registerBaseEventTools(s *sdk.Server, client *api.Client) {
	baseEventsSchema, err := listBaseEventsSchema()
	if err != nil {
		panic(err) // static schema construction; fails only on a programming error
	}
	sdk.AddTool(s, &sdk.Tool{
		Name:        "list_base_events",
		InputSchema: baseEventsSchema,
		Description: "List the ungendered event catalog — the events themselves, independent of gender. " +
			"Returns {data, count, has_more}: data records are (id, constant, sport, sport_group, " +
			"mark_type, distance_type, total_distance, total_distance_unit) plus a boolean " +
			"classification (is_track, is_field, is_relay, is_sprint, is_distance, is_hurdles, " +
			"is_jump, is_vertical_jump, is_horizontal_jump, is_throw, is_multi, has_wind); when " +
			"has_more is true the window was truncated at limit — raise limit before summarizing as " +
			"complete. Use this to group or classify results (all throws, all hurdles, everything " +
			"scored by time), and list_events for the gendered events results attach to.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, args listBaseEventsArgs) (*sdk.CallToolResult, any, error) {
		query := url.Values{}
		for param, v := range map[string]string{
			"filter[sport]":     args.Sport,
			"filter[mark_type]": args.MarkType,
		} {
			if v != "" {
				query.Set(param, v)
			}
		}
		res, err := client.List(ctx, "/base-events", query, boundedLimit(args.Limit))
		if err != nil {
			return nil, nil, err
		}
		return listResult(res)
	})

	sdk.AddTool(s, &sdk.Tool{
		Name: "get_base_event",
		Description: "Fetch one base event by its numeric ID. Returns the full catalog record as " +
			"JSON, including its sport group, mark type, distance, and boolean classification.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, args getBaseEventArgs) (*sdk.CallToolResult, any, error) {
		raw, err := client.GetRaw(ctx, "/base-events/"+strconv.FormatInt(args.ID, 10))
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(raw)
	})
}
