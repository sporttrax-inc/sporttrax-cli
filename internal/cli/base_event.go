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

var baseEventCmd = &cobra.Command{
	Use:   "base-event",
	Short: "Work with base events",
	Long: `Browse the ungendered event catalog.

A base event is the event itself — 100-meter-hurdles, shot put, 5k — with
its sport, mark type, and classification. ` + "`sporttrax event`" + ` lists the
gendered events that instantiate these.`,
}

var baseEventListCmd = &cobra.Command{
	Use:   "list",
	Short: "List base events",
	Example: `  sporttrax base-event list
  sporttrax base-event list --sport track
  sporttrax base-event list --mark-type time --limit all
  sporttrax base-event list --csv > event-catalog.csv`,
	Args: cobra.NoArgs,
	RunE: runBaseEventList,
}

var baseEventViewCmd = &cobra.Command{
	Use:     "view <id>",
	Short:   "View a single base event",
	Example: `  sporttrax base-event view 1`,
	Args:    cobra.ExactArgs(1),
	RunE:    runBaseEventView,
}

func init() {
	f := baseEventListCmd.Flags()
	f.String("sport", "", "filter by sport: "+strings.Join(api.Sports, ", "))
	f.String("mark-type", "", "filter by mark type: "+strings.Join(api.MarkTypes, ", "))
	f.StringP("limit", "L", "30", `maximum base events to fetch, or "all"`)
	for flag, values := range map[string][]string{
		"sport":     api.Sports,
		"mark-type": api.MarkTypes,
	} {
		_ = baseEventListCmd.RegisterFlagCompletionFunc(flag,
			cobra.FixedCompletions(values, cobra.ShellCompDirectiveNoFileComp))
	}

	baseEventCmd.AddCommand(baseEventListCmd, baseEventViewCmd)
	rootCmd.AddCommand(baseEventCmd)
}

func runBaseEventList(cmd *cobra.Command, args []string) error {
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
	for flag, spec := range map[string]struct {
		param   string
		allowed []string
	}{
		"sport":     {"filter[sport]", api.Sports},
		"mark-type": {"filter[mark_type]", api.MarkTypes},
	} {
		v, _ := cmd.Flags().GetString(flag)
		if v == "" {
			continue
		}
		if err := validateEnum(flag, v, spec.allowed); err != nil {
			return err
		}
		query.Set(spec.param, v)
	}

	res, err := newClient(cmd, env, token).List(cmd.Context(), "/base-events", query, limit)
	if err != nil {
		return err
	}
	items := res.Items

	out := cmd.OutOrStdout()
	if jsonOutput(cmd) {
		return ui.JSON(out, items)
	}

	tbl := ui.Table{Headers: []string{"ID", "CONSTANT", "SPORT", "SPORT GROUP", "MARK TYPE", "TOTAL DISTANCE"}}
	for _, raw := range items {
		var b api.BaseEvent
		if err := json.Unmarshal(raw, &b); err != nil {
			return fmt.Errorf("unexpected base event payload: %w", err)
		}
		tbl.Rows = append(tbl.Rows, []string{
			strconv.FormatInt(b.ID, 10), b.Constant, b.Sport, b.SportGroup,
			b.MarkType, totalDistanceOf(b),
		})
	}
	if csvOutput(cmd) {
		return tbl.RenderCSV(out)
	}
	if len(tbl.Rows) == 0 && ui.IsTTY(out) {
		fmt.Fprintln(out, "No base events found")
		return nil
	}
	if err := tbl.Render(out); err != nil {
		return err
	}
	if res.HasMore {
		ui.Note(out, fmt.Sprintf("Showing %d base events — more available; raise --limit or use --limit all", len(items)))
	}
	return nil
}

func runBaseEventView(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid base event id %q: must be a number", args[0])
	}
	env, err := environment(cmd)
	if err != nil {
		return err
	}
	token, err := requireToken(env)
	if err != nil {
		return err
	}

	raw, err := newClient(cmd, env, token).GetRaw(cmd.Context(), "/base-events/"+strconv.FormatInt(id, 10))
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if jsonOutput(cmd) {
		return ui.JSON(out, raw)
	}

	var b api.BaseEvent
	if err := json.Unmarshal(raw, &b); err != nil {
		return fmt.Errorf("unexpected base event payload: %w", err)
	}
	render := ui.KeyValues
	if csvOutput(cmd) {
		render = ui.KeyValuesCSV
	}
	return render(out, [][2]string{
		{"ID", strconv.FormatInt(b.ID, 10)},
		{"Constant", b.Constant},
		{"Sport", b.Sport},
		{"Sport group", b.SportGroup},
		{"Mark type", b.MarkType},
		{"Distance type", strPtr(b.DistanceType)},
		{"Total distance", totalDistanceOf(b)},
		{"Track", strconv.FormatBool(b.IsTrack)},
		{"Field", strconv.FormatBool(b.IsField)},
		{"Relay", strconv.FormatBool(b.IsRelay)},
		{"Sprint", strconv.FormatBool(b.IsSprint)},
		{"Distance", strconv.FormatBool(b.IsDistance)},
		{"Hurdles", strconv.FormatBool(b.IsHurdles)},
		{"Jump", strconv.FormatBool(b.IsJump)},
		{"Vertical jump", strconv.FormatBool(b.IsVerticalJump)},
		{"Horizontal jump", strconv.FormatBool(b.IsHorizontalJump)},
		{"Throw", strconv.FormatBool(b.IsThrow)},
		{"Multi", strconv.FormatBool(b.IsMulti)},
		{"Wind", strconv.FormatBool(b.HasWind)},
	})
}

// totalDistanceOf composes the distance display ("100.00 meter"), labeled
// by its parent field. Empty when the base event carries no distance.
func totalDistanceOf(b api.BaseEvent) string {
	d := strPtr(b.TotalDistance)
	if d == "" {
		return ""
	}
	if u := strPtr(b.TotalDistanceUnit); u != "" {
		return d + " " + u
	}
	return d
}
