package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	mcpsrv "github.com/sporttrax-inc/sporttrax-cli/internal/mcp"
	"github.com/sporttrax-inc/sporttrax-cli/internal/version"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run an MCP server exposing SportTrax data as AI tools",
	Long: `Run a Model Context Protocol server over stdio, exposing the
SportTrax public API as structured tools for AI assistants. Authentication
and environment selection work exactly like every other command: the stored
token for --env (or SPORTTRAX_API_TOKEN) is used.

Register with Claude Code:

  claude mcp add sporttrax -- sporttrax mcp

Or in any MCP host's config:

  {"command": "sporttrax", "args": ["mcp"]}

Point it at another environment with --env, e.g.:

  claude mcp add sporttrax-testing -- sporttrax --env testing mcp

Tools: whoami, list_meets, get_meet, list_results, get_result,
list_events, get_event, list_base_events, get_base_event, get_athlete,
get_team`,
	RunE: runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func runMCP(cmd *cobra.Command, args []string) error {
	env, err := environment(cmd)
	if err != nil {
		return err
	}
	token, err := requireToken(env)
	if err != nil {
		return err
	}
	// stdout carries the MCP protocol; status goes to stderr.
	fmt.Fprintf(cmd.ErrOrStderr(), "sporttrax MCP server: %s over stdio\n", env.APIURL)
	return mcpsrv.Run(cmd.Context(), newClient(cmd, env, token), version.Version)
}
