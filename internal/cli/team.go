package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
	"github.com/sporttrax-inc/sporttrax-cli/internal/ui"
)

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Work with teams",
	Long: `Look up a team by ID.

The API exposes teams individually, not as a list, so there is no team
search. Team IDs come from results: ` + "`sporttrax result list --meet <id>`" + `
carries a team_id on every row.`,
}

var teamViewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View a single team",
	Example: `  sporttrax team view 55
  sporttrax team view 55 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTeamView,
}

func init() {
	teamCmd.AddCommand(teamViewCmd)
	rootCmd.AddCommand(teamCmd)
}

func runTeamView(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid team id %q: must be a number", args[0])
	}
	env, err := environment(cmd)
	if err != nil {
		return err
	}
	token, err := requireToken(env)
	if err != nil {
		return err
	}

	raw, err := newClient(cmd, env, token).GetRaw(cmd.Context(), "/teams/"+strconv.FormatInt(id, 10))
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if jsonOutput(cmd) {
		return ui.JSON(out, raw)
	}

	var t api.Team
	if err := json.Unmarshal(raw, &t); err != nil {
		return fmt.Errorf("unexpected team payload: %w", err)
	}
	render := ui.KeyValues
	if csvOutput(cmd) {
		render = ui.KeyValuesCSV
	}
	return render(out, [][2]string{
		{"ID", strconv.FormatInt(t.ID, 10)},
		{"Name", t.Name},
		{"Display name", t.DisplayName},
		{"Abbr", t.Abbr},
		{"Level", t.Level},
		{"Sport", t.Sport},
		{"Team type", t.TeamType},
		{"City", t.City},
		{"State code", t.StateCode},
	})
}
