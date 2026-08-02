package mcp

import (
	"context"
	"net/url"
	"strconv"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
)

type listEventsArgs struct {
	Gender       string `json:"gender,omitempty" jsonschema:"gender filter: male, female, or mixed"`
	BaseEventID  int64  `json:"base_event_id,omitempty" jsonschema:"only events instantiating this base event"`
	MultiEventID int64  `json:"multi_event_id,omitempty" jsonschema:"only events belonging to this multi event"`
	Limit        int    `json:"limit,omitempty" jsonschema:"maximum events to return; default 25, max 100 — keep small"`
}

type getEventArgs struct {
	ID int64 `json:"id" jsonschema:"numeric event ID"`
}

// listEventsSchema tightens the inferred schema with the server's enum so
// AI callers cannot send a silently-no-match value.
func listEventsSchema() (*jsonschema.Schema, error) {
	schema, err := jsonschema.For[listEventsArgs](nil)
	if err != nil {
		return nil, err
	}
	schema.Properties["gender"].Enum = enumOf(api.Genders...)
	return schema, nil
}

func registerEventTools(s *sdk.Server, client *api.Client) {
	eventsSchema, err := listEventsSchema()
	if err != nil {
		panic(err) // static schema construction; fails only on a programming error
	}
	sdk.AddTool(s, &sdk.Tool{
		Name:        "list_events",
		InputSchema: eventsSchema,
		Description: "List the gendered events results are recorded against. Returns {data, count, " +
			"has_more}: data records are (id, constant, name, name_with_gender, gender, " +
			"base_event_id, multi_event_id); when has_more is true the window was truncated at limit " +
			"— raise limit before summarizing as complete. An event is one gender's instance of a " +
			"base event, so \"Female 100m Hurdles\" and \"Male 100m Hurdles\" are two events sharing " +
			"one base_event_id. Use list_base_events for the ungendered catalog, and get_event for a " +
			"single known ID.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, args listEventsArgs) (*sdk.CallToolResult, any, error) {
		query := url.Values{}
		if args.Gender != "" {
			query.Set("filter[gender]", args.Gender)
		}
		for param, v := range map[string]int64{
			"filter[base_event_id]":  args.BaseEventID,
			"filter[multi_event_id]": args.MultiEventID,
		} {
			if v > 0 {
				query.Set(param, strconv.FormatInt(v, 10))
			}
		}
		res, err := client.List(ctx, "/events", query, boundedLimit(args.Limit))
		if err != nil {
			return nil, nil, err
		}
		return listResult(res)
	})

	sdk.AddTool(s, &sdk.Tool{
		Name: "get_event",
		Description: "Fetch one gendered event by its numeric ID. Returns the full event record as " +
			"JSON, including the base_event_id it instantiates.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, args getEventArgs) (*sdk.CallToolResult, any, error) {
		raw, err := client.GetRaw(ctx, "/events/"+strconv.FormatInt(args.ID, 10))
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(raw)
	})
}
