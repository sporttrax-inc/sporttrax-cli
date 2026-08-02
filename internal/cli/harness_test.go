package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/zalando/go-keyring"
)

// setupCLITest isolates a test from the real machine: mock keychain, temp
// config dir, and a token provided via the env var path.
func setupCLITest(t *testing.T) {
	t.Helper()
	keyring.MockInit()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("AppData", dir)
	t.Setenv("SPORTTRAX_API_TOKEN", "test-token")
}

// runCommand executes the real root command with args, capturing stdout
// and stderr. Buffers are not TTYs, so output takes the piped paths.
func runCommand(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	resetFlags(rootCmd)
	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs(args)
	err = rootCmd.ExecuteContext(context.Background())
	return outBuf.String(), errBuf.String(), err
}

// resetFlags restores every flag in the tree to its default so state
// doesn't leak between tests (cobra flag values are sticky).
func resetFlags(c *cobra.Command) {
	c.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	for _, sub := range c.Commands() {
		resetFlags(sub)
	}
}
