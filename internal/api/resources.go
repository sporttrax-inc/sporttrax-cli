package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Typed views of the server's PublicApi V1 resources, used for table
// rendering only. --json output passes the server's raw JSON through
// untouched (see List/GetRaw) so new server fields reach consumers without
// a CLI release.

// Venue is the nested venue block on a meet.
type Venue struct {
	Name      string `json:"name"`
	City      string `json:"city"`
	StateCode string `json:"state_code"`
}

// Meet mirrors the server's MeetResource.
type Meet struct {
	ID                     int64  `json:"id"`
	Name                   string `json:"name"`
	Sport                  string `json:"sport"`
	Status                 string `json:"status"`
	IsSanctioned           bool   `json:"is_sanctioned"`
	Timezone               string `json:"timezone"`
	FirstSessionStartingAt string `json:"first_session_starting_at"`
	LastSessionStartingAt  string `json:"last_session_starting_at"`
	Venue                  *Venue `json:"venue"`
}

// Athlete mirrors the server's AthleteResource.
type Athlete struct {
	ID               int64  `json:"id"`
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	Gender           string `json:"gender"`
	HSGraduationYear *int   `json:"hs_graduation_year"`
}

// Team mirrors the server's TeamResource.
type Team struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Abbr        string `json:"abbr"`
	Level       string `json:"level"`
	Sport       string `json:"sport"`
	TeamType    string `json:"team_type"`
	City        string `json:"city"`
	StateCode   string `json:"state_code"`
}

// Event mirrors the server's EventResource: one gendered event, pointing
// at the base event it instantiates.
type Event struct {
	ID             int64  `json:"id"`
	Constant       string `json:"constant"`
	Name           string `json:"name"`
	NameWithGender string `json:"name_with_gender"`
	Gender         string `json:"gender"`
	BaseEventID    int64  `json:"base_event_id"`
	MultiEventID   *int64 `json:"multi_event_id"`
}

// BaseEvent mirrors the server's BaseEventResource: the ungendered event
// catalog entry.
//
// The TFRRS and Hy-Tek codes the server carries are integration
// identifiers for other systems and are deliberately absent here, so no
// rendered view shows them. They still reach consumers untouched through
// --json and the MCP tools, which are raw passthrough by contract.
type BaseEvent struct {
	ID                int64   `json:"id"`
	Constant          string  `json:"constant"`
	Sport             string  `json:"sport"`
	SportGroup        string  `json:"sport_group"`
	MarkType          string  `json:"mark_type"`
	DistanceType      *string `json:"distance_type"`
	IsTrack           bool    `json:"is_track"`
	IsField           bool    `json:"is_field"`
	IsRelay           bool    `json:"is_relay"`
	IsSprint          bool    `json:"is_sprint"`
	IsDistance        bool    `json:"is_distance"`
	IsHurdles         bool    `json:"is_hurdles"`
	IsJump            bool    `json:"is_jump"`
	IsVerticalJump    bool    `json:"is_vertical_jump"`
	IsHorizontalJump  bool    `json:"is_horizontal_jump"`
	IsThrow           bool    `json:"is_throw"`
	IsMulti           bool    `json:"is_multi"`
	HasWind           bool    `json:"has_wind"`
	TotalDistance     *string `json:"total_distance"`
	TotalDistanceUnit *string `json:"total_distance_unit"`
}

// ResultAthlete is the denormalized athlete block on a result.
type ResultAthlete struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Grade     *int   `json:"grade"`
}

// ResultTeam is the denormalized team block on a result.
type ResultTeam struct {
	Name string `json:"name"`
	Abbr string `json:"abbr"`
}

// ResultDivision is the denormalized division block on a result.
type ResultDivision struct {
	Name string `json:"name"`
	Abbr string `json:"abbr"`
}

// ResultEvent is the event-catalog block on a result.
type ResultEvent struct {
	Name           *string `json:"name"`
	NameWithGender *string `json:"name_with_gender"`
}

// ResultMeet is the meet block on a result.
type ResultMeet struct {
	Name *string `json:"name"`
}

// ResultMark is the mark block on a result: the mark encoding only.
type ResultMark struct {
	Value   string  `json:"value"`
	Type    string  `json:"type"`
	Display string  `json:"display"`
	English *string `json:"english"`
	Metric  *string `json:"metric"`
}

// Result mirrors the server's ResultResource.
type Result struct {
	ID               int64          `json:"id"`
	MeetID           int64          `json:"meet_id"`
	AthleteID        *int64         `json:"athlete_id"`
	TeamID           *int64         `json:"team_id"`
	EventID          int64          `json:"event_id"`
	BaseEventID      int64          `json:"base_event_id"`
	RelayTeamID      *int64         `json:"relay_team_id"`
	IsRelayTeam      bool           `json:"is_relay_team"`
	Sport            string         `json:"sport"`
	Gender           string         `json:"gender"`
	Level            string         `json:"level"`
	Round            string         `json:"round"`
	At               string         `json:"at"`
	IsIndoor         bool           `json:"is_indoor"`
	TimingType       string         `json:"timing_type"`
	Athlete          ResultAthlete  `json:"athlete"`
	Team             ResultTeam     `json:"team"`
	RelayTeamName    *string        `json:"relay_team_name"`
	MeetStateCode    *string        `json:"meet_state_code"`
	Meet             ResultMeet     `json:"meet"`
	Event            ResultEvent    `json:"event"`
	Division         ResultDivision `json:"division"`
	Mark             ResultMark     `json:"mark"`
	Wind             *float64       `json:"wind"`
	IsLegal          bool           `json:"is_legal"`
	InvalidStatus    *string        `json:"invalid_status"`
	Place            *int           `json:"place"` // the main place: within the meet event round
	FlightPlace      *int           `json:"flight_place"`
	FlightGroupPlace *int           `json:"flight_group_place"`
	Points           *float64       `json:"points"`
	IsOfficial       bool           `json:"is_official"`
	IsValid          bool           `json:"is_valid"`
}

const maxPerPage = 100 // server cap

type page struct {
	Data  []json.RawMessage `json:"data"`
	Links struct {
		Next *string `json:"next"`
	} `json:"links"`
}

// ListResult is a fetched collection window. HasMore reports whether the
// server had more records beyond the window (the API's cursor pagination
// carries no total count, only a next cursor).
type ListResult struct {
	Items   []json.RawMessage
	HasMore bool
}

// List fetches a collection endpoint, following pagination cursors until
// limit items are collected (limit <= 0 fetches every page). Items are
// returned as raw JSON so server fields pass through untouched.
func (c *Client) List(ctx context.Context, path string, query url.Values, limit int) (ListResult, error) {
	q := url.Values{}
	for k, v := range query {
		q[k] = v
	}

	res := ListResult{Items: []json.RawMessage{}}
	for {
		perPage := maxPerPage
		if limit > 0 {
			if remaining := limit - len(res.Items); remaining < perPage {
				perPage = remaining
			}
		}
		q.Set("per_page", strconv.Itoa(perPage))

		var p page
		if err := c.Get(ctx, path, q, &p); err != nil {
			return ListResult{}, err
		}
		res.Items = append(res.Items, p.Data...)

		if limit > 0 && len(res.Items) >= limit {
			res.HasMore = p.Links.Next != nil || len(res.Items) > limit
			res.Items = res.Items[:limit]
			return res, nil
		}
		if p.Links.Next == nil || len(p.Data) == 0 {
			return res, nil
		}
		next, err := url.Parse(*p.Links.Next)
		if err != nil {
			return ListResult{}, fmt.Errorf("bad pagination link %q: %w", *p.Links.Next, err)
		}
		cursor := next.Query().Get("cursor")
		if cursor == "" {
			return res, nil
		}
		q.Set("cursor", cursor)
	}
}

// GetRaw fetches a single resource as raw JSON.
func (c *Client) GetRaw(ctx context.Context, path string) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.Get(ctx, path, nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}
