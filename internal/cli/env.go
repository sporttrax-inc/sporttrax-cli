package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
	"github.com/sporttrax-inc/sporttrax-cli/internal/auth"
	"github.com/sporttrax-inc/sporttrax-cli/internal/config"
	"github.com/sporttrax-inc/sporttrax-cli/internal/ui"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Inspect SportTrax environments",
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List known environments and login state",
	Long: `List the environments the CLI can target: the built-in ones
plus any defined or overridden in the config file. Select one with --env
or SPORTTRAX_ENV.`,
	Example: `  sporttrax env list
  sporttrax env list --json`,
	Args: cobra.NoArgs,
	RunE: runEnvList,
}

func init() {
	envCmd.AddCommand(envListCmd)
	rootCmd.AddCommand(envCmd)
}

type envInfo struct {
	Name          string `json:"name"`
	APIURL        string `json:"api_url"`
	Source        string `json:"source"`
	PusherAppKey  string `json:"pusher_app_key,omitempty"`
	PusherCluster string `json:"pusher_cluster,omitempty"`
	LoggedIn      bool   `json:"logged_in"`
	TokenSource   string `json:"token_source,omitempty"`
}

func runEnvList(cmd *cobra.Command, args []string) error {
	envs, err := config.Load()
	if err != nil {
		return err
	}
	builtin := config.Builtin()

	names := make([]string, 0, len(envs))
	for name := range envs {
		names = append(names, name)
	}
	sort.Strings(names)

	infos := make([]envInfo, 0, len(names))
	for _, name := range names {
		env := envs[name]
		info := envInfo{
			Name:          name,
			APIURL:        env.APIURL,
			Source:        "config",
			PusherAppKey:  env.Pusher.AppKey,
			PusherCluster: env.Pusher.Cluster,
		}
		if b, ok := builtin[name]; ok {
			if b == env {
				info.Source = "built-in"
			} else {
				info.Source = "built-in+config"
			}
		}
		if host, err := api.Host(env.APIURL); err == nil {
			if _, src, err := auth.Token(host); err == nil {
				info.LoggedIn = true
				info.TokenSource = string(src)
			}
		}
		infos = append(infos, info)
	}

	out := cmd.OutOrStdout()
	if jsonOutput(cmd) {
		return ui.JSON(out, infos)
	}

	// TTY rows show masked keys and placeholders; piped TSV and CSV carry
	// raw values so scripts and spreadsheets get lossless data.
	isTTY := ui.IsTTY(out) && !csvOutput(cmd)
	tbl := ui.Table{Headers: []string{"NAME", "API URL", "SOURCE", "PUSHER KEY", "LOGGED IN"}}
	for _, info := range infos {
		pusherKey := info.PusherAppKey
		if isTTY {
			pusherKey = "—"
			if info.PusherAppKey != "" {
				pusherKey = auth.Mask(info.PusherAppKey)
			}
		}
		loggedIn := "no"
		if info.LoggedIn {
			loggedIn = "yes"
			if isTTY {
				loggedIn = "yes (" + info.TokenSource + ")"
			}
		}
		tbl.Rows = append(tbl.Rows, []string{info.Name, info.APIURL, info.Source, pusherKey, loggedIn})
	}
	if csvOutput(cmd) {
		return tbl.RenderCSV(out)
	}
	if err := tbl.Render(out); err != nil {
		return err
	}

	if isTTY {
		if path, err := config.File(); err == nil {
			fmt.Fprintf(out, "\nCustom environments: %s\n", path)
		}
	}
	return nil
}
