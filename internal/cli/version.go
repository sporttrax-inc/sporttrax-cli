package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/sporttrax-inc/sporttrax-cli/internal/ui"
	"github.com/sporttrax-inc/sporttrax-cli/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the sporttrax version",
	Example: `  sporttrax version
  sporttrax version --json`,
	Args: cobra.NoArgs,
	RunE: runVersion,
}

// versionInfo is the structured form. Field names are the labels
// lowercased, so --json stays guessable from the printed line.
type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

func runVersion(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	info := versionInfo{
		Version: version.Version,
		Commit:  version.Commit,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
	switch {
	case jsonOutput(cmd):
		return ui.JSON(out, info)
	case csvOutput(cmd):
		return ui.KeyValuesCSV(out, [][2]string{
			{"Version", info.Version},
			{"Commit", info.Commit},
			{"OS", info.OS},
			{"Arch", info.Arch},
		})
	}
	fmt.Fprint(out, versionString())
	return nil
}

func versionString() string {
	return fmt.Sprintf("sporttrax %s (%s) %s/%s\n", version.Version, version.Commit, runtime.GOOS, runtime.GOARCH)
}

func init() {
	// Make --version print the same line as the version subcommand.
	rootCmd.Version = version.Version
	rootCmd.SetVersionTemplate(versionString())
	rootCmd.AddCommand(versionCmd)
}
