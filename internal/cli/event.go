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

var eventCmd = &cobra.Command{
	Use:   "event",
	Short: "Work with events",
	Long: `Browse the gendered events results are recorded against.

Each event instantiates a base event for one gender — "Female 100m
Hurdles" is the female event for the 100-meter-hurdles base event. See
` + "`sporttrax base-event`" + ` for the ungendered catalog.`,
}

var eventListCmd = &cobra.Command{
	Use:   "list",
	Short: "List events",
	Example: `  sporttrax event list
  sporttrax event list --gender female
  sporttrax event list --base-event 1
  sporttrax event list --limit all --csv > events.csv`,
	Args: cobra.NoArgs,
	RunE: runEventList,
}

var eventViewCmd = &cobra.Command{
	Use:     "view <id>",
	Short:   "View a single event",
	Example: `  sporttrax event view 1`,
	Args:    cobra.ExactArgs(1),
	RunE:    runEventView,
}

func init() {
	f := eventListCmd.Flags()
	f.String("gender", "", "filter by gender: "+strings.Join(api.Genders, ", "))
	f.Int64("base-event", 0, "filter by base event ID")
	f.Int64("multi-event", 0, "filter by multi event ID")
	f.StringP("limit", "L", "30", `maximum events to fetch, or "all"`)
	_ = eventListCmd.RegisterFlagCompletionFunc("gender",
		cobra.FixedCompletions(api.Genders, cobra.ShellCompDirectiveNoFileComp))

	eventCmd.AddCommand(eventListCmd, eventViewCmd)
	rootCmd.AddCommand(eventCmd)
}

func runEventList(cmd *cobra.Command, args []string) error {
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
	if v, _ := cmd.Flags().GetString("gender"); v != "" {
		if err := validateEnum("gender", v, api.Genders); err != nil {
			return err
		}
		query.Set("filter[gender]", v)
	}
	for flag, param := range map[string]string{
		"base-event":  "filter[base_event_id]",
		"multi-event": "filter[multi_event_id]",
	} {
		if v, _ := cmd.Flags().GetInt64(flag); v > 0 {
			query.Set(param, strconv.FormatInt(v, 10))
		}
	}

	res, err := newClient(cmd, env, token).List(cmd.Context(), "/events", query, limit)
	if err != nil {
		return err
	}
	items := res.Items

	out := cmd.OutOrStdout()
	if jsonOutput(cmd) {
		return ui.JSON(out, items)
	}

	tbl := ui.Table{Headers: []string{"ID", "CONSTANT", "NAME", "GENDER", "BASE EVENT ID"}}
	for _, raw := range items {
		var e api.Event
		if err := json.Unmarshal(raw, &e); err != nil {
			return fmt.Errorf("unexpected event payload: %w", err)
		}
		tbl.Rows = append(tbl.Rows, []string{
			strconv.FormatInt(e.ID, 10), e.Constant, e.Name, e.Gender,
			strconv.FormatInt(e.BaseEventID, 10),
		})
	}
	if csvOutput(cmd) {
		return tbl.RenderCSV(out)
	}
	if len(tbl.Rows) == 0 && ui.IsTTY(out) {
		fmt.Fprintln(out, "No events found")
		return nil
	}
	if err := tbl.Render(out); err != nil {
		return err
	}
	if res.HasMore {
		ui.Note(out, fmt.Sprintf("Showing %d events — more available; raise --limit or use --limit all", len(items)))
	}
	return nil
}

func runEventView(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid event id %q: must be a number", args[0])
	}
	env, err := environment(cmd)
	if err != nil {
		return err
	}
	token, err := requireToken(env)
	if err != nil {
		return err
	}

	raw, err := newClient(cmd, env, token).GetRaw(cmd.Context(), "/events/"+strconv.FormatInt(id, 10))
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if jsonOutput(cmd) {
		return ui.JSON(out, raw)
	}

	var e api.Event
	if err := json.Unmarshal(raw, &e); err != nil {
		return fmt.Errorf("unexpected event payload: %w", err)
	}
	render := ui.KeyValues
	if csvOutput(cmd) {
		render = ui.KeyValuesCSV
	}
	return render(out, [][2]string{
		{"ID", strconv.FormatInt(e.ID, 10)},
		{"Constant", e.Constant},
		{"Name", e.Name},
		{"Name with gender", e.NameWithGender},
		{"Gender", e.Gender},
		{"Base event ID", strconv.FormatInt(e.BaseEventID, 10)},
		{"Multi event ID", int64Ptr(e.MultiEventID)},
	})
}
