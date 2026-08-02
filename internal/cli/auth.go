package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
	"github.com/sporttrax-inc/sporttrax-cli/internal/auth"
	"github.com/sporttrax-inc/sporttrax-cli/internal/ui"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with the SportTrax API",
	Long: `Manage authentication with the SportTrax API.

Tokens are stored per environment (keyed by API host), so you can be logged
in to production, staging, and testing at the same time. Target an
environment with --env (or a raw URL with --api-url):

  sporttrax auth login                     # production
  sporttrax --env testing auth login       # stock environment
  sporttrax --env ultra auth login         # custom environment from config.yaml
  sporttrax env list                       # see all environments`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in with a personal access token",
	Long: `Log in to a SportTrax environment with a personal access token.

Create a token in the web UI under API Tokens, making sure the "public-api"
permission is checked, then paste it at the prompt. The token is validated
against the API and stored in the OS keychain (or a config file if no
keychain is available).`,
	RunE: runAuthLogin,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status for the selected environment",
	RunE:  runAuthStatus,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove the stored token for the selected environment",
	RunE:  runAuthLogout,
}

func init() {
	authLoginCmd.Flags().Bool("with-token", false, "read the token from standard input")
	authLoginCmd.Flags().Bool("no-browser", false, "do not open the token creation page in a browser")
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	env, err := environment(cmd)
	if err != nil {
		return err
	}
	baseURL := env.APIURL
	host, err := api.Host(baseURL)
	if err != nil {
		return err
	}

	token, err := readLoginToken(cmd, baseURL)
	if err != nil {
		return err
	}
	if token == "" {
		return errors.New("no token provided")
	}

	client := newClient(cmd, env, token)
	var me api.Me
	var validateErr error
	if ui.IsTTY(os.Stdout) {
		_ = spinner.New().
			Title("Validating token against " + baseURL + "...").
			Action(func() { me, _, validateErr = client.MeInfo(cmd.Context()) }).
			Run()
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Validating token against %s...\n", baseURL)
		me, _, validateErr = client.MeInfo(cmd.Context())
	}
	if validateErr != nil {
		return fmt.Errorf("token validation failed: %w", validateErr)
	}

	source, err := auth.Save(host, token)
	if err != nil {
		return fmt.Errorf("storing token: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Logged in to %s as %s (token stored in %s)\n",
		host, identity(me), source)
	if os.Getenv(auth.EnvToken) != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "note: %s is set and takes precedence over the stored token\n", auth.EnvToken)
	}
	return nil
}

func readLoginToken(cmd *cobra.Command, baseURL string) (string, error) {
	withToken, _ := cmd.Flags().GetBool("with-token")
	stdinIsTTY := term.IsTerminal(int(os.Stdin.Fd()))

	if withToken || !stdinIsTTY {
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}

	tokenURL := strings.TrimRight(baseURL, "/") + "/user/api-tokens"
	if noBrowser, _ := cmd.Flags().GetBool("no-browser"); !noBrowser {
		openBrowser(tokenURL)
	}

	var token string
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("Paste your personal access token").
			Description("Create one at " + tokenURL + "\n" +
				`with the "public-api" permission checked. It is shown only once.`).
			EchoMode(huh.EchoModePassword).
			Value(&token),
	)).Run()
	if err != nil {
		return "", fmt.Errorf("reading token: %w", err)
	}
	return strings.TrimSpace(token), nil
}

type authStatus struct {
	Host        string `json:"host"`
	APIURL      string `json:"api_url"`
	LoggedIn    bool   `json:"logged_in"`
	Token       string `json:"token,omitempty"` // masked
	TokenSource string `json:"token_source,omitempty"`
	Valid       bool   `json:"valid"`
	User        string `json:"user,omitempty"`
	Role        string `json:"role,omitempty"`
	Error       string `json:"error,omitempty"`
}

// identity formats a /me response for display: "Jeff Hansen (admin)".
func identity(me api.Me) string {
	name := me.User.Name
	if name == "" {
		name = "unknown user"
	}
	if me.User.Role != "" {
		name += " (" + me.User.Role + ")"
	}
	return name
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	env, err := environment(cmd)
	if err != nil {
		return err
	}
	baseURL := env.APIURL
	host, err := api.Host(baseURL)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	status := authStatus{Host: host, APIURL: baseURL}

	token, source, err := auth.Token(host)
	if errors.Is(err, auth.ErrNotFound) {
		if jsonOutput(cmd) {
			_ = ui.JSON(out, status)
			return fmt.Errorf("not logged in to %s: %w", host, auth.ErrNotFound)
		}
		fmt.Fprintf(out, "✗ Not logged in to %s\n", host)
		printOtherHosts(out, host)
		return fmt.Errorf("run `sporttrax auth login` to authenticate: %w", auth.ErrNotFound)
	}
	if err != nil {
		return err
	}
	status.LoggedIn = true
	status.Token = auth.Mask(token)
	status.TokenSource = string(source)

	me, _, validateErr := newClient(cmd, env, token).MeInfo(cmd.Context())
	status.Valid = validateErr == nil
	if validateErr != nil {
		status.Error = validateErr.Error()
	} else {
		status.User = me.User.Name
		status.Role = me.User.Role
	}

	if jsonOutput(cmd) {
		if err := ui.JSON(out, status); err != nil {
			return err
		}
		return validateErr
	}
	if csvOutput(cmd) {
		// Machine-readable forms carry the unmasked token, as piped
		// output already does — masking is a TTY affordance.
		if err := ui.KeyValuesCSV(out, [][2]string{
			{"Host", status.Host},
			{"API URL", status.APIURL},
			{"Logged in", strconv.FormatBool(status.LoggedIn)},
			{"Token", token},
			{"Token source", status.TokenSource},
			{"Valid", strconv.FormatBool(status.Valid)},
			{"User", status.User},
			{"Role", status.Role},
			{"Error", status.Error},
		}); err != nil {
			return err
		}
		return validateErr
	}

	fmt.Fprintf(out, "Host:   %s\n", host)
	fmt.Fprintf(out, "Token:  %s (from %s)\n", status.Token, status.TokenSource)
	if validateErr != nil {
		fmt.Fprintln(out, "Status: ✗ invalid")
		return validateErr
	}
	fmt.Fprintf(out, "User:   %s\n", identity(me))
	fmt.Fprintln(out, "Status: ✓ valid")
	printOtherHosts(out, host)
	return nil
}

// printOtherHosts lists other environments with stored tokens so users
// juggling production/staging/testing can see what they're logged in to.
func printOtherHosts(out io.Writer, current string) {
	hosts := auth.StoredHosts()
	var others []string
	for _, h := range hosts {
		if h != current {
			others = append(others, h)
		}
	}
	if len(others) == 0 {
		return
	}
	sort.Strings(others)
	fmt.Fprintf(out, "\nAlso logged in to: %s (select with --env or --api-url)\n", strings.Join(others, ", "))
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	env, err := environment(cmd)
	if err != nil {
		return err
	}
	host, err := api.Host(env.APIURL)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()

	removed, err := auth.Delete(host)
	if err != nil {
		return err
	}
	if len(removed) == 0 {
		fmt.Fprintf(out, "No stored token for %s\n", host)
	} else {
		for _, source := range removed {
			fmt.Fprintf(out, "✓ Removed token for %s from %s\n", host, source)
		}
	}
	if os.Getenv(auth.EnvToken) != "" {
		fmt.Fprintf(out, "note: %s is still set in your environment and will be used for requests\n", auth.EnvToken)
	}
	return nil
}

// openBrowser opens url in the default browser, best-effort.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url) //nolint:gosec // url comes from the user's own env config, not remote input
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url) //nolint:gosec // url comes from the user's own env config, not remote input
	default:
		cmd = exec.Command("xdg-open", url) //nolint:gosec // url comes from the user's own env config, not remote input
	}
	_ = cmd.Start()
}
