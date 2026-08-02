package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
	"github.com/sporttrax-inc/sporttrax-cli/internal/auth"
	"github.com/sporttrax-inc/sporttrax-cli/internal/config"
)

const (
	// EnvAPIURL overrides the API base URL when --api-url is not passed.
	EnvAPIURL = "SPORTTRAX_API_URL"
	// EnvEnvironment selects a named environment when --env is not passed.
	EnvEnvironment = "SPORTTRAX_ENV"

	defaultEnvironment = "production"
)

var rootCmd = &cobra.Command{
	Use:           "sporttrax",
	Short:         "SportTrax command-line interface",
	Long:          "sporttrax is a fast, cross-platform CLI for the SportTrax APIs.",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if noColor, _ := cmd.Flags().GetBool("no-color"); noColor {
			os.Setenv("NO_COLOR", "1") // honored by lipgloss/termenv
		}
	},
}

// Execute runs the root command under ctx (cancelled on Ctrl-C/SIGTERM).
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

// Root exposes the command tree for documentation generation (cmd/gen-docs).
func Root() *cobra.Command {
	return rootCmd
}

func init() {
	rootCmd.PersistentFlags().String("env", envOr(EnvEnvironment, defaultEnvironment),
		"named environment: production, staging, testing, or one from config.yaml (env: "+EnvEnvironment+")")
	rootCmd.PersistentFlags().String("api-url", os.Getenv(EnvAPIURL),
		"base URL override for the SportTrax API, https only (http is allowed for localhost); takes precedence over --env (env: "+EnvAPIURL+")")
	rootCmd.PersistentFlags().Bool("json", false, "output as JSON instead of formatted text")
	rootCmd.PersistentFlags().Bool("csv", false, "output as CSV with a header row (for spreadsheets)")
	rootCmd.MarkFlagsMutuallyExclusive("json", "csv")
	rootCmd.PersistentFlags().Bool("insecure", false,
		"skip TLS certificate verification (self-signed dev instances only; refused for the stock environments)")
	rootCmd.PersistentFlags().Bool("verbose", os.Getenv("SPORTTRAX_DEBUG") != "",
		"log API requests to stderr (env: SPORTTRAX_DEBUG)")
	rootCmd.PersistentFlags().Bool("no-color", false,
		"disable color and styling (NO_COLOR is also honored)")
}

// displayUnits resolves the mark display unit system: --units flag, then
// config.yaml, then "" (native). Affects rendering only, never JSON.
func displayUnits(cmd *cobra.Command) (string, error) {
	units, _ := cmd.Flags().GetString("units")
	if units == "" {
		units = config.Units()
	}
	if units != "" && units != "english" && units != "metric" {
		return "", fmt.Errorf("invalid units %q: must be english or metric", units)
	}
	return units, nil
}

// requireToken resolves the stored token for env's host, with a
// login-hinting error (exit code 4) when absent.
func requireToken(env config.Environment) (string, error) {
	host, err := api.Host(env.APIURL)
	if err != nil {
		return "", err
	}
	token, _, err := auth.Token(host)
	if errors.Is(err, auth.ErrNotFound) {
		return "", fmt.Errorf("not logged in to %s — run `sporttrax auth login`: %w", host, err)
	}
	return token, err
}

// newClient builds the API client for cmd's environment, wiring verbose
// logging and user-facing warnings to stderr.
func newClient(cmd *cobra.Command, env config.Environment, token string) *api.Client {
	c := api.New(env.APIURL, token, env.Insecure)
	c.Warn = cmd.ErrOrStderr()
	if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
		c.Verbose = cmd.ErrOrStderr()
	}
	return c
}

// jsonOutput reports whether --json was requested.
func jsonOutput(cmd *cobra.Command) bool {
	j, _ := cmd.Flags().GetBool("json")
	return j
}

// csvOutput reports whether --csv was requested. Mutually exclusive with
// --json: --json is the data contract, CSV is a spreadsheet convenience.
func csvOutput(cmd *cobra.Command) bool {
	c, _ := cmd.Flags().GetBool("csv")
	return c
}

// environment resolves the target environment for cmd. Precedence:
// --api-url flag / SPORTTRAX_API_URL (URL override wins), then --env flag /
// SPORTTRAX_ENV, then production.
func environment(cmd *cobra.Command) (config.Environment, error) {
	name, _ := cmd.Flags().GetString("env")
	env, err := config.Resolve(name)
	if err != nil {
		if url, _ := cmd.Flags().GetString("api-url"); url != "" {
			// A URL override doesn't need a valid named environment.
			env = config.Environment{APIURL: url}
		} else {
			return config.Environment{}, err
		}
	}
	if url, _ := cmd.Flags().GetString("api-url"); url != "" {
		env.APIURL = url
	}
	// Every request carries the user's token, so the transport is checked
	// before any client is built: no plaintext HTTP off-machine.
	if err := api.ValidateBaseURL(env.APIURL); err != nil {
		return config.Environment{}, err
	}
	flagInsecure, _ := cmd.Flags().GetBool("insecure")
	if flagInsecure {
		env.Insecure = true
	}
	if env.Insecure {
		if err := checkInsecure(cmd, env.APIURL, flagInsecure); err != nil {
			return config.Environment{}, err
		}
	}
	return env, nil
}

// checkInsecure applies the policy for disabling TLS verification.
// Self-signed certificates are a local-development concern, so silence is
// earned only there: config.yaml consent stays quiet for the local
// machine, every other host warns on every command, and the stock
// environments — which have real certificates — are refused outright so a
// stray config edit cannot silently expose a production token.
func checkInsecure(cmd *cobra.Command, apiURL string, fromFlag bool) error {
	host, err := api.Host(apiURL)
	if err != nil {
		return err
	}
	switch {
	case config.IsBuiltinHost(host):
		return fmt.Errorf(
			"refusing to disable TLS certificate verification for %s: it has a valid certificate — remove `insecure` for this environment", host)
	case !api.IsLoopbackHost(host):
		fmt.Fprintf(cmd.ErrOrStderr(),
			"warning: TLS certificate verification disabled for %s — your token is exposed to interception\n", host)
	case fromFlag:
		fmt.Fprintln(cmd.ErrOrStderr(), "warning: TLS certificate verification disabled")
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
