package mcp

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
)

type listResultsArgs struct {
	MeetID      int64  `json:"meet_id,omitempty" jsonschema:"anchor: meet ID — at least one of meet_id, athlete_id, team_id is required"`
	AthleteID   int64  `json:"athlete_id,omitempty" jsonschema:"anchor: athlete ID"`
	TeamID      int64  `json:"team_id,omitempty" jsonschema:"anchor: team ID"`
	EventID     int64  `json:"event_id,omitempty" jsonschema:"filter by event ID"`
	BaseEventID int64  `json:"base_event_id,omitempty" jsonschema:"filter by base event ID"`
	Gender      string `json:"gender,omitempty" jsonschema:"gender filter"`
	Sport       string `json:"sport,omitempty" jsonschema:"sport filter: track (track & field), xc (cross country), or road (road racing)"`
	Level       string `json:"level,omitempty" jsonschema:"competition level filter"`
	Round       string `json:"round,omitempty" jsonschema:"round filter"`
	Official    *bool  `json:"official,omitempty" jsonschema:"only official (true) or unofficial (false) results"`
	Relay       *bool  `json:"relay,omitempty" jsonschema:"only relay (true) or individual (false) results"`
	From        string `json:"from,omitempty" jsonschema:"only results on or after this date (YYYY-MM-DD)"`
	To          string `json:"to,omitempty" jsonschema:"only results on or before this date (YYYY-MM-DD)"`
	Sort        string `json:"sort,omitempty" jsonschema:"sort order; prefix - for descending (default id)"`
	Limit       int    `json:"limit,omitempty" jsonschema:"maximum results to return; default 25, max 100 — keep small"`
}

type getResultArgs struct {
	ID int64 `json:"id" jsonschema:"numeric result ID"`
}

func enumOf(values ...string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

// listResultsSchema tightens the inferred schema with the server's enum
// values so AI callers can't send silently-no-match filter values.
func listResultsSchema() (*jsonschema.Schema, error) {
	schema, err := jsonschema.For[listResultsArgs](nil)
	if err != nil {
		return nil, err
	}
	schema.Properties["gender"].Enum = enumOf(api.Genders...)
	schema.Properties["sport"].Enum = enumOf(api.Sports...)
	schema.Properties["level"].Enum = enumOf(api.Levels...)
	schema.Properties["round"].Enum = enumOf(api.Rounds...)
	schema.Properties["sort"].Enum = enumOf(api.ResultSorts...)
	return schema, nil
}

func registerResultTools(s *sdk.Server, client *api.Client) {
	resultsSchema, err := listResultsSchema()
	if err != nil {
		panic(err) // static schema construction; fails only on a programming error
	}
	sdk.AddTool(s, &sdk.Tool{
		Name:        "list_results",
		InputSchema: resultsSchema,
		Description: "List competition results, requiring at least one anchor (meet_id, athlete_id, or " +
			"team_id). Returns {data, count, has_more}; when has_more is true the window was truncated " +
			"at limit — raise limit before summarizing as complete. data records: ids (meet/athlete/" +
			"team/event/base_event/relay_team), sport, gender, level, round, at (date), timing_type, athlete " +
			"(first_name, last_name, grade), team (name, abbr), meet (name), event (name, " +
			"name_with_gender), " +
			"division, mark (value, type, display, " +
			"english, metric — all null for markless results), wind, is_legal, invalid_status " +
			"(DNS/FS/DNF/DQ/SCR/NM when the result has no mark; render its uppercase form in a " +
			"results column), place (THE place to use: the " +
			"athlete's position within the meet event round), flight_place and flight_group_place " +
			"(narrower within-flight positions), points, is_official, is_valid. Use get_result for a " +
			"single known ID.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, args listResultsArgs) (*sdk.CallToolResult, any, error) {
		if args.MeetID <= 0 && args.AthleteID <= 0 && args.TeamID <= 0 {
			return nil, nil, fmt.Errorf("at least one anchor is required: meet_id, athlete_id, or team_id")
		}
		query := url.Values{}
		for param, v := range map[string]int64{
			"filter[meet_id]":       args.MeetID,
			"filter[athlete_id]":    args.AthleteID,
			"filter[team_id]":       args.TeamID,
			"filter[event_id]":      args.EventID,
			"filter[base_event_id]": args.BaseEventID,
		} {
			if v > 0 {
				query.Set(param, strconv.FormatInt(v, 10))
			}
		}
		for param, v := range map[string]string{
			"filter[gender]": args.Gender,
			"filter[sport]":  args.Sport,
			"filter[level]":  args.Level,
			"filter[round]":  args.Round,
			"filter[from]":   args.From,
			"filter[to]":     args.To,
			"sort":           args.Sort,
		} {
			if v != "" {
				query.Set(param, v)
			}
		}
		// Boolean columns filter on 1/0, not true/false strings.
		for param, v := range map[string]*bool{
			"filter[is_official]":   args.Official,
			"filter[is_relay_team]": args.Relay,
		} {
			if v != nil {
				if *v {
					query.Set(param, "1")
				} else {
					query.Set(param, "0")
				}
			}
		}

		res, err := client.List(ctx, "/results", query, boundedLimit(args.Limit))
		if err != nil {
			return nil, nil, err
		}
		return listResult(res)
	})

	sdk.AddTool(s, &sdk.Tool{
		Name: "get_result",
		Description: "Fetch one competition result by its numeric ID. Returns the full result " +
			"record as JSON, including the mark, athlete, team, and placement details.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, args getResultArgs) (*sdk.CallToolResult, any, error) {
		raw, err := client.GetRaw(ctx, "/results/"+strconv.FormatInt(args.ID, 10))
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(raw)
	})
}
