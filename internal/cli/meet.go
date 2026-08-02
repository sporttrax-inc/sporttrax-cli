package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
	"github.com/sporttrax-inc/sporttrax-cli/internal/ui"
)

var meetCmd = &cobra.Command{
	Use:   "meet",
	Short: "Work with meets",
	Long: `Browse published meets.

Meets are the top of the tree: a meet's ID anchors ` + "`sporttrax result list`" + `,
which is where athlete and team IDs come from in turn.`,
}

var meetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List published meets",
	Example: `  sporttrax meet list
  sporttrax meet list --sport track --from 2026-06-01
  sporttrax meet list --state ID --city Boise
  sporttrax meet list --name "state" --limit 100
  sporttrax meet list --json | jq '.[].name'`,
	Args: cobra.NoArgs,
	RunE: runMeetList,
}

var meetViewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View a single meet",
	Example: `  sporttrax meet view 4821
  sporttrax meet view 4821 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runMeetView,
}

func init() {
	f := meetListCmd.Flags()
	f.String("sport", "", "filter by sport: "+strings.Join(api.Sports, ", "))
	f.String("name", "", "filter by name (partial match)")
	f.String("from", "", "meets starting on or after this date (YYYY-MM-DD)")
	f.String("to", "", "meets starting on or before this date (YYYY-MM-DD)")
	f.String("state", "", "filter by two-letter venue state code, e.g. ID")
	f.String("city", "", "filter by venue city (requires --state)")
	f.StringP("limit", "L", "30", `maximum meets to fetch, or "all"`)
	_ = meetListCmd.RegisterFlagCompletionFunc("sport",
		cobra.FixedCompletions(api.Sports, cobra.ShellCompDirectiveNoFileComp))
	meetCmd.AddCommand(meetListCmd, meetViewCmd)
	rootCmd.AddCommand(meetCmd)
}

func runMeetList(cmd *cobra.Command, args []string) error {
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

	city, _ := cmd.Flags().GetString("city")
	state, _ := cmd.Flags().GetString("state")
	if city != "" && state == "" {
		return fmt.Errorf("--state is required when --city is used (city names collide across states)")
	}
	if state != "" && !isStateCode(state) {
		return fmt.Errorf("invalid --state %q: must be a two-letter state code, e.g. ID", state)
	}
	if v, _ := cmd.Flags().GetString("sport"); v != "" {
		if err := validateEnum("sport", v, api.Sports); err != nil {
			return err
		}
	}

	query := url.Values{}
	for flag, param := range map[string]string{
		"sport": "filter[sport]",
		"name":  "filter[name]",
		"from":  "filter[from]",
		"to":    "filter[to]",
		"state": "filter[state]",
		"city":  "filter[city]",
	} {
		if v, _ := cmd.Flags().GetString(flag); v != "" {
			query.Set(param, v)
		}
	}

	res, err := newClient(cmd, env, token).List(cmd.Context(), "/meets", query, limit)
	if err != nil {
		return err
	}
	items := res.Items

	out := cmd.OutOrStdout()
	if jsonOutput(cmd) {
		return ui.JSON(out, items)
	}

	tbl := ui.Table{Headers: []string{"ID", "NAME", "SPORT", "STATUS", "STARTS", "VENUE"}}
	for _, raw := range items {
		var m api.Meet
		if err := json.Unmarshal(raw, &m); err != nil {
			return fmt.Errorf("unexpected meet payload: %w", err)
		}
		tbl.Rows = append(tbl.Rows, []string{
			strconv.FormatInt(m.ID, 10), m.Name, m.Sport, m.Status,
			dateOf(m.FirstSessionStartingAt), venueOf(m.Venue),
		})
	}
	if csvOutput(cmd) {
		return tbl.RenderCSV(out)
	}
	if len(tbl.Rows) == 0 && ui.IsTTY(out) {
		fmt.Fprintln(out, "No meets found")
		return nil
	}
	if err := tbl.Render(out); err != nil {
		return err
	}
	if res.HasMore {
		ui.Note(out, fmt.Sprintf("Showing %d meets — more available; raise --limit or use --limit all", len(items)))
	}
	return nil
}

func runMeetView(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid meet id %q: must be a number", args[0])
	}
	env, err := environment(cmd)
	if err != nil {
		return err
	}
	token, err := requireToken(env)
	if err != nil {
		return err
	}

	raw, err := newClient(cmd, env, token).GetRaw(cmd.Context(), "/meets/"+strconv.FormatInt(id, 10))
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if jsonOutput(cmd) {
		return ui.JSON(out, raw)
	}

	var m api.Meet
	if err := json.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("unexpected meet payload: %w", err)
	}
	render := ui.KeyValues
	if csvOutput(cmd) {
		render = ui.KeyValuesCSV
	}
	return render(out, [][2]string{
		{"ID", strconv.FormatInt(m.ID, 10)},
		{"Name", m.Name},
		{"Sport", m.Sport},
		{"Status", m.Status},
		{"Sanctioned", strconv.FormatBool(m.IsSanctioned)},
		{"Timezone", m.Timezone},
		{"First session", m.FirstSessionStartingAt},
		{"Last session", m.LastSessionStartingAt},
		{"Venue", venueOf(m.Venue)},
	})
}

// isStateCode reports whether s is a two-letter state code. The server
// enforces the same shape, so checking here turns a round trip into an
// immediate, actionable error.
func isStateCode(s string) bool {
	if len(s) != 2 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}

// parseLimit reads --limit: a positive count, or "all" (0 = deplete) with
// a rate-limit warning.
func parseLimit(cmd *cobra.Command) (int, error) {
	s, _ := cmd.Flags().GetString("limit")
	if s == "all" {
		fmt.Fprintln(cmd.ErrOrStderr(),
			"fetching all pages — note the API rate limit (15 req/min for non-admin accounts)")
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid --limit %q: must be a positive number or \"all\"", s)
	}
	return n, nil
}

// dateOf renders a timestamp as its date for list columns. The API emits
// session times as "2006-01-02 15:04:05" and other timestamps as RFC3339,
// so both are accepted; anything else passes through untouched rather
// than being guessed at.
func dateOf(timestamp string) string {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, timestamp); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return timestamp
}

func venueOf(v *api.Venue) string {
	if v == nil {
		return ""
	}
	s := v.Name
	if v.City != "" {
		if s != "" {
			s += " — "
		}
		s += v.City
		if v.StateCode != "" {
			s += ", " + v.StateCode
		}
	}
	return s
}
