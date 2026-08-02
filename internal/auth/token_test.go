package auth

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/sporttrax-inc/sporttrax-cli/internal/config"
)

// setupIsolated points token storage at a temp config dir and a mock
// keychain so tests never touch the real system.
func setupIsolated(t *testing.T) {
	t.Helper()
	keyring.MockInit()
	dir := t.TempDir()
	switch filepath.Separator {
	case '/':
		// config.Dir honors XDG_CONFIG_HOME, then HOME; setting both
		// covers either resolution path.
		t.Setenv("XDG_CONFIG_HOME", dir)
		t.Setenv("HOME", dir)
	default:
		t.Setenv("AppData", dir)
	}
	t.Setenv(EnvToken, "")
}

func TestSaveAndResolve(t *testing.T) {
	setupIsolated(t)

	source, err := Save("testing.sporttrax.com", "1|secret")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if source != SourceKeychain {
		t.Fatalf("Save stored in %q, want keychain", source)
	}

	token, gotSource, err := Token("testing.sporttrax.com")
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if token != "1|secret" || gotSource != SourceKeychain {
		t.Fatalf("Token = %q from %q, want 1|secret from keychain", token, gotSource)
	}
}

func TestEnvOverridesStored(t *testing.T) {
	setupIsolated(t)

	if _, err := Save("sporttrax.com", "1|stored"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Setenv(EnvToken, "1|env")

	token, source, err := Token("sporttrax.com")
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if token != "1|env" || source != SourceEnv {
		t.Fatalf("Token = %q from %q, want env token to win", token, source)
	}
}

func TestEnvTokenRefusedForUnknownHost(t *testing.T) {
	setupIsolated(t)
	t.Setenv(EnvToken, "1|env")

	// A host nobody declared: the env token must not follow --api-url to
	// an arbitrary server.
	_, _, err := Token("attacker.example")
	if !errors.Is(err, ErrHostNotAllowed) {
		t.Fatalf("Token(attacker.example) err = %v, want ErrHostNotAllowed", err)
	}

	// Stock environments and the local machine are fine.
	for _, host := range []string{"sporttrax.com", "staging.sporttrax.com", "127.0.0.1:8555", "localhost:8555"} {
		token, source, err := Token(host)
		if err != nil || token != "1|env" || source != SourceEnv {
			t.Fatalf("Token(%s) = %q/%q err=%v, want the env token", host, token, source, err)
		}
	}
}

func TestStoredTokenStillUsedForUndeclaredHost(t *testing.T) {
	setupIsolated(t)

	// Logged in to a one-off host, with a CI token in the environment for
	// somewhere else. The stored token is bound to this host, so it is
	// safe — the env var simply doesn't apply here.
	if _, err := Save("ultra.sporttrax.dev", "1|stored"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Setenv(EnvToken, "1|env")

	token, source, err := Token("ultra.sporttrax.dev")
	if err != nil || token != "1|stored" || source != SourceKeychain {
		t.Fatalf("Token = %q/%q err=%v, want the stored token", token, source, err)
	}
}

func TestEnvTokenAllowedForConfiguredHost(t *testing.T) {
	setupIsolated(t)
	t.Setenv(EnvToken, "1|env")

	dir, err := config.Dir()
	if err != nil {
		t.Fatalf("config.Dir: %v", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(dir, "config.yaml")
	cfg := "environments:\n  ultra:\n    api_url: https://ultra.sporttrax.dev\n"
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Declaring the environment is how the user vouches for the host.
	token, source, err := Token("ultra.sporttrax.dev")
	if err != nil || token != "1|env" || source != SourceEnv {
		t.Fatalf("Token(ultra) = %q/%q err=%v, want the env token", token, source, err)
	}
}

func TestHostsFileIsNarrowedWhenTooOpen(t *testing.T) {
	if filepath.Separator != '/' {
		t.Skip("mode bits are not meaningful on Windows")
	}
	setupIsolated(t)

	path, err := ConfigFile()
	if err != nil {
		t.Fatalf("ConfigFile: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// A hosts.json left world-readable by a backup restore or a copy
	// between machines: WriteFile alone would preserve the mode.
	if err := os.WriteFile(path, []byte(`{"sporttrax.com":{"token":"1|old"}}`), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, _, err := Token("sporttrax.com"); err != nil {
		t.Fatalf("Token: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("hosts.json perm = %#o after read, want 0600", perm)
	}
}

func TestHostsAreIndependent(t *testing.T) {
	setupIsolated(t)

	if _, err := Save("sporttrax.com", "1|prod"); err != nil {
		t.Fatalf("Save prod: %v", err)
	}
	if _, err := Save("testing.sporttrax.com", "2|testing"); err != nil {
		t.Fatalf("Save testing: %v", err)
	}

	prod, _, _ := Token("sporttrax.com")
	testing_, _, _ := Token("testing.sporttrax.com")
	if prod != "1|prod" || testing_ != "2|testing" {
		t.Fatalf("cross-host mixup: prod=%q testing=%q", prod, testing_)
	}

	hosts := StoredHosts()
	slices.Sort(hosts)
	want := []string{"sporttrax.com", "testing.sporttrax.com"}
	if !slices.Equal(hosts, want) {
		t.Fatalf("StoredHosts = %v, want %v", hosts, want)
	}

	if _, err := Delete("testing.sporttrax.com"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, _, err := Token("testing.sporttrax.com"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("testing token should be gone, got err=%v", err)
	}
	if got, _, err := Token("sporttrax.com"); err != nil || got != "1|prod" {
		t.Fatalf("prod token should survive testing logout, got %q err=%v", got, err)
	}
}

func TestMask(t *testing.T) {
	if got := Mask("12|abcdefghijklmnop"); got != "12|a…mnop" {
		t.Fatalf("Mask = %q", got)
	}
	if got := Mask("short"); got != "****" {
		t.Fatalf("Mask short = %q, must not leak", got)
	}
}
