package mcp

import (
	"context"
	"strconv"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
)

// Athletes and teams are fetched one at a time: the API exposes no index
// for either, so there is no search tool to register. IDs come off result
// records, which carry athlete_id and team_id.

type getAthleteArgs struct {
	ID int64 `json:"id" jsonschema:"numeric athlete ID, as found in a result's athlete_id"`
}

type getTeamArgs struct {
	ID int64 `json:"id" jsonschema:"numeric team ID, as found in a result's team_id"`
}

func registerAthleteTools(s *sdk.Server, client *api.Client) {
	sdk.AddTool(s, &sdk.Tool{
		Name: "get_athlete",
		Description: "Fetch one athlete by numeric ID. Returns (id, first_name, last_name, gender, " +
			"hs_graduation_year). There is no athlete search — get IDs from the athlete_id on " +
			"result records returned by list_results.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, args getAthleteArgs) (*sdk.CallToolResult, any, error) {
		raw, err := client.GetRaw(ctx, "/athletes/"+strconv.FormatInt(args.ID, 10))
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(raw)
	})
}

func registerTeamTools(s *sdk.Server, client *api.Client) {
	sdk.AddTool(s, &sdk.Tool{
		Name: "get_team",
		Description: "Fetch one team by numeric ID. Returns (id, name, display_name, abbr, level, " +
			"sport, team_type, city, state_code). There is no team search — get IDs from the " +
			"team_id on result records returned by list_results.",
	}, func(ctx context.Context, req *sdk.CallToolRequest, args getTeamArgs) (*sdk.CallToolResult, any, error) {
		raw, err := client.GetRaw(ctx, "/teams/"+strconv.FormatInt(args.ID, 10))
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(raw)
	})
}
