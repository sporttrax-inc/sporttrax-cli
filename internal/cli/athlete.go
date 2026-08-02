package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
	"github.com/sporttrax-inc/sporttrax-cli/internal/ui"
)

var athleteCmd = &cobra.Command{
	Use:   "athlete",
	Short: "Work with athletes",
	Long: `Look up an athlete by ID.

The API exposes athletes individually, not as a list, so there is no
athlete search. Athlete IDs come from results: ` + "`sporttrax result list" + `
--meet <id>` + "`" + ` carries an athlete_id on every row.`,
}

var athleteViewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View a single athlete",
	Example: `  sporttrax athlete view 992
  sporttrax athlete view 992 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAthleteView,
}

func init() {
	athleteCmd.AddCommand(athleteViewCmd)
	rootCmd.AddCommand(athleteCmd)
}

func runAthleteView(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid athlete id %q: must be a number", args[0])
	}
	env, err := environment(cmd)
	if err != nil {
		return err
	}
	token, err := requireToken(env)
	if err != nil {
		return err
	}

	raw, err := newClient(cmd, env, token).GetRaw(cmd.Context(), "/athletes/"+strconv.FormatInt(id, 10))
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if jsonOutput(cmd) {
		return ui.JSON(out, raw)
	}

	var a api.Athlete
	if err := json.Unmarshal(raw, &a); err != nil {
		return fmt.Errorf("unexpected athlete payload: %w", err)
	}
	render := ui.KeyValues
	if csvOutput(cmd) {
		render = ui.KeyValuesCSV
	}
	return render(out, [][2]string{
		{"ID", strconv.FormatInt(a.ID, 10)},
		{"First name", a.FirstName},
		{"Last name", a.LastName},
		{"Gender", a.Gender},
		{"HS graduation year", intPtr(a.HSGraduationYear)},
	})
}
