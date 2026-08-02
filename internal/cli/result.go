package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
	"github.com/sporttrax-inc/sporttrax-cli/internal/ui"
)

var resultCmd = &cobra.Command{
	Use:   "result",
	Short: "Work with results",
}

var resultListCmd = &cobra.Command{
	Use:   "list",
	Short: "List results for a meet, athlete, or team",
	Long: `List results. At least one anchor filter is required: --meet,
--athlete, or --team.`,
	Example: `  sporttrax result list --meet 4821
  sporttrax result list --athlete 992 --sport track --round finals
  sporttrax result list --team 55 --from 2026-06-01 --sort -at
  sporttrax result list --meet 4821 --official true --json | jq '.[].place'`,
	Args: cobra.NoArgs,
	RunE: runResultList,
}

var resultViewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View a single result",
	Example: `  sporttrax result view 120345
  sporttrax result view 120345 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runResultView,
}

func init() {
	f := resultListCmd.Flags()
	f.Int64("meet", 0, "anchor: filter by meet ID")
	f.Int64("athlete", 0, "anchor: filter by athlete ID")
	f.Int64("team", 0, "anchor: filter by team ID")
	f.Int64("event", 0, "filter by event ID")
	f.Int64("base-event", 0, "filter by base event ID")
	f.String("gender", "", "filter by gender: "+strings.Join(api.Genders, ", "))
	f.String("sport", "", "filter by sport: "+strings.Join(api.Sports, ", "))
	f.String("level", "", "filter by level: "+strings.Join(api.Levels, ", "))
	f.String("round", "", "filter by round: "+strings.Join(api.Rounds, ", "))
	f.String("official", "", "filter by official status: true, false")
	f.String("relay", "", "filter by relay results: true, false")
	f.String("from", "", "results on or after this date (YYYY-MM-DD)")
	f.String("to", "", "results on or before this date (YYYY-MM-DD)")
	f.String("sort", "", "sort order: "+strings.Join(api.ResultSorts, ", ")+" (default id)")
	f.StringP("limit", "L", "30", `maximum results to fetch, or "all"`)
	// Scoped to this command: it is the only place a mark is rendered as
	// a single value. The detail view lists the english and metric forms
	// as their own fields, so there is nothing for it to choose there.
	f.String("units", "",
		`unit system for displayed marks: english, metric (default: "units" in config.yaml, else the mark's native form)`)

	for flag, values := range map[string][]string{
		"gender":   api.Genders,
		"sport":    api.Sports,
		"level":    api.Levels,
		"round":    api.Rounds,
		"official": boolValues,
		"relay":    boolValues,
		"sort":     api.ResultSorts,
	} {
		_ = resultListCmd.RegisterFlagCompletionFunc(flag,
			cobra.FixedCompletions(values, cobra.ShellCompDirectiveNoFileComp))
	}

	resultCmd.AddCommand(resultListCmd, resultViewCmd)
	rootCmd.AddCommand(resultCmd)
}

func runResultList(cmd *cobra.Command, args []string) error {
	env, err := environment(cmd)
	if err != nil {
		return err
	}
	token, err := requireToken(env)
	if err != nil {
		return err
	}
	limit, err := parseLimit(cmd)
	if err != nil {
		return err
	}

	query := url.Values{}

	anchors := 0
	for flag, param := range map[string]string{
		"meet":       "filter[meet_id]",
		"athlete":    "filter[athlete_id]",
		"team":       "filter[team_id]",
		"event":      "filter[event_id]",
		"base-event": "filter[base_event_id]",
	} {
		if v, _ := cmd.Flags().GetInt64(flag); v > 0 {
			query.Set(param, strconv.FormatInt(v, 10))
			if flag == "meet" || flag == "athlete" || flag == "team" {
				anchors++
			}
		}
	}
	if anchors == 0 {
		return fmt.Errorf("at least one anchor filter is required: --meet, --athlete, or --team")
	}

	for flag, allowed := range map[string][]string{
		"gender": api.Genders,
		"sport":  api.Sports,
		"level":  api.Levels,
		"round":  api.Rounds,
	} {
		v, _ := cmd.Flags().GetString(flag)
		if v == "" {
			continue
		}
		if err := validateEnum(flag, v, allowed); err != nil {
			return err
		}
		query.Set("filter["+flag+"]", v)
	}

	for flag, param := range map[string]string{
		"official": "filter[is_official]",
		"relay":    "filter[is_relay_team]",
	} {
		v, _ := cmd.Flags().GetString(flag)
		if v == "" {
			continue
		}
		if err := validateEnum(flag, v, boolValues); err != nil {
			return err
		}
		query.Set(param, boolFilter(v))
	}

	for flag, param := range map[string]string{"from": "filter[from]", "to": "filter[to]"} {
		if v, _ := cmd.Flags().GetString(flag); v != "" {
			query.Set(param, v)
		}
	}

	if v, _ := cmd.Flags().GetString("sort"); v != "" {
		if err := validateEnum("sort", v, api.ResultSorts); err != nil {
			return err
		}
		query.Set("sort", v)
	}

	res, err := newClient(cmd, env, token).List(cmd.Context(), "/results", query, limit)
	if err != nil {
		return err
	}
	items := res.Items

	out := cmd.OutOrStdout()
	if jsonOutput(cmd) {
		return ui.JSON(out, items)
	}

	units, err := displayUnits(cmd)
	if err != nil {
		return err
	}
	tbl := ui.Table{Headers: []string{"ID", "AT", "MEET", "EVENT", "ATHLETE", "TEAM", "DIVISION", "ROUND", "MARK", "PLACE"}}
	for _, raw := range items {
		var r api.Result
		if err := json.Unmarshal(raw, &r); err != nil {
			return fmt.Errorf("unexpected result payload: %w", err)
		}
		division := r.Division.Abbr
		if division == "" {
			division = r.Division.Name
		}
		tbl.Rows = append(tbl.Rows, []string{
			strconv.FormatInt(r.ID, 10), r.At, strPtr(r.Meet.Name), eventOf(r), athleteOf(r),
			r.Team.Abbr, division, r.Round, markOf(r, units), intPtr(r.Place),
		})
	}
	if csvOutput(cmd) {
		return tbl.RenderCSV(out)
	}
	if len(tbl.Rows) == 0 && ui.IsTTY(out) {
		fmt.Fprintln(out, "No results found")
		return nil
	}
	if err := tbl.Render(out); err != nil {
		return err
	}
	if res.HasMore {
		ui.Note(out, fmt.Sprintf("Showing %d results — more available; raise --limit or use --limit all", len(items)))
	}
	return nil
}

func runResultView(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid result id %q: must be a number", args[0])
	}
	env, err := environment(cmd)
	if err != nil {
		return err
	}
	token, err := requireToken(env)
	if err != nil {
		return err
	}

	raw, err := newClient(cmd, env, token).GetRaw(cmd.Context(), "/results/"+strconv.FormatInt(id, 10))
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if jsonOutput(cmd) {
		return ui.JSON(out, raw)
	}

	var r api.Result
	if err := json.Unmarshal(raw, &r); err != nil {
		return fmt.Errorf("unexpected result payload: %w", err)
	}
	render := ui.KeyValues
	if csvOutput(cmd) {
		render = ui.KeyValuesCSV
	}
	return render(out, [][2]string{
		{"ID", strconv.FormatInt(r.ID, 10)},
		{"Meet ID", strconv.FormatInt(r.MeetID, 10)},
		{"Athlete ID", int64Ptr(r.AthleteID)},
		{"Team ID", int64Ptr(r.TeamID)},
		{"Event ID", strconv.FormatInt(r.EventID, 10)},
		{"Base event ID", strconv.FormatInt(r.BaseEventID, 10)},
		{"Relay team ID", int64Ptr(r.RelayTeamID)},
		{"Relay team", strconv.FormatBool(r.IsRelayTeam)},
		{"Sport", r.Sport},
		{"Gender", r.Gender},
		{"Level", r.Level},
		{"Round", r.Round},
		{"At", r.At},
		{"Indoor", strconv.FormatBool(r.IsIndoor)},
		{"Timing type", r.TimingType},
		{"Athlete", athleteOf(r)},
		{"Team", nameAbbr(r.Team.Name, r.Team.Abbr)},
		{"Relay team name", strPtr(r.RelayTeamName)},
		{"Meet state code", strPtr(r.MeetStateCode)},
		{"Meet", strPtr(r.Meet.Name)},
		{"Event", eventOf(r)},
		{"Division", nameAbbr(r.Division.Name, r.Division.Abbr)},
		{"Mark value", r.Mark.Value},
		{"Mark type", r.Mark.Type},
		{"Mark display", r.Mark.Display},
		{"Mark english", strPtr(r.Mark.English)},
		{"Mark metric", strPtr(r.Mark.Metric)},
		{"Wind", floatPtr(r.Wind)},
		{"Legal", strconv.FormatBool(r.IsLegal)},
		{"Invalid status", strPtr(r.InvalidStatus)},
		{"Place", intPtr(r.Place)},
		{"Flight place", intPtr(r.FlightPlace)},
		{"Flight group place", intPtr(r.FlightGroupPlace)},
		{"Points", floatPtr(r.Points)},
		{"Official", strconv.FormatBool(r.IsOfficial)},
		{"Valid", strconv.FormatBool(r.IsValid)},
	})
}

// markOf renders the results-column mark: the mark in the requested unit
// system (falling back to the native display form when no conversion
// exists), or the uppercase status code (DNF, DQ, ...) for markless
// results. Display-only — JSON always carries the full mark block.
func markOf(r api.Result, units string) string {
	display := r.Mark.Display
	switch units {
	case "english":
		if e := strPtr(r.Mark.English); e != "" {
			display = e
		}
	case "metric":
		if m := strPtr(r.Mark.Metric); m != "" {
			display = m
		}
	}
	if display == "" && r.InvalidStatus != nil {
		return strings.ToUpper(*r.InvalidStatus)
	}
	return display
}

// eventOf composes the event display, preferring the gendered name
// ("Girls 100 Meter") since result lists mix genders.
func eventOf(r api.Result) string {
	if n := strPtr(r.Event.NameWithGender); n != "" {
		return n
	}
	return strPtr(r.Event.Name)
}

// validateEnum rejects values outside the server's enum before any
// request — an invalid value would otherwise silently match nothing.
func validateEnum(flag, value string, allowed []string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("invalid --%s %q: must be one of %s", flag, value, strings.Join(allowed, ", "))
}

// boolFilter maps true/false to 1/0 — the server's boolean columns filter
// on numeric values, and a literal "true" string would coerce to 0.
func boolFilter(v string) string {
	if v == "true" {
		return "1"
	}
	return "0"
}

// athleteOf composes the athlete display; relay results carry the relay
// team name instead of an individual athlete.
func athleteOf(r api.Result) string {
	if r.IsRelayTeam {
		return strPtr(r.RelayTeamName)
	}
	name := strings.TrimSpace(r.Athlete.FirstName + " " + r.Athlete.LastName)
	if r.Athlete.Grade != nil {
		name += fmt.Sprintf(" (grade %d)", *r.Athlete.Grade)
	}
	return name
}

func nameAbbr(name, abbr string) string {
	if name == "" {
		return abbr
	}
	if abbr != "" && abbr != name {
		return name + " (" + abbr + ")"
	}
	return name
}

func strPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intPtr(i *int) string {
	if i == nil {
		return ""
	}
	return strconv.Itoa(*i)
}

func int64Ptr(i *int64) string {
	if i == nil {
		return ""
	}
	return strconv.FormatInt(*i, 10)
}

func floatPtr(f *float64) string {
	if f == nil {
		return ""
	}
	return strconv.FormatFloat(*f, 'f', -1, 64)
}
