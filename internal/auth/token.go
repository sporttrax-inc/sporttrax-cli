// Package auth stores and resolves the SportTrax personal access token.
//
// Resolution order: SPORTTRAX_API_TOKEN env var, then the OS keychain, then
// a hosts.json config file (the fallback for headless machines without a
// keychain). Keychain and file entries are keyed by API host so tokens for
// production and staging can coexist.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
	"github.com/sporttrax-inc/sporttrax-cli/internal/config"
)

const (
	// EnvToken overrides all stored tokens when set.
	EnvToken       = "SPORTTRAX_API_TOKEN" //nolint:gosec // env var name, not a credential
	keyringService = "sporttrax-cli"
)

// Source identifies where a token was found or stored.
type Source string

const (
	SourceEnv      Source = "environment variable"
	SourceKeychain Source = "system keychain"
	SourceFile     Source = "config file"
	SourceNone     Source = ""
)

// ErrNotFound is returned when no token is stored for a host.
var ErrNotFound = errors.New("no token found")

// ErrHostNotAllowed is returned when SPORTTRAX_API_TOKEN is set but the
// target host is not one the user has vouched for.
var ErrHostNotAllowed = errors.New("token not allowed for this host")

// envTokenAllowed reports whether the SPORTTRAX_API_TOKEN env var may be
// sent to host. Stored tokens are keyed by host and so can only ever reach
// the host they were saved for; the env var has no such binding, which
// would otherwise let any `--api-url` (or SPORTTRAX_API_URL) aim a
// production token at an arbitrary server. It is honored only for hosts
// the user has declared — a stock environment, one from config.yaml, or
// the local machine.
func envTokenAllowed(host string) (bool, error) {
	if api.IsLoopbackHost(host) {
		return true, nil
	}
	hosts, err := config.KnownHosts()
	if err != nil {
		return false, err
	}
	for _, h := range hosts {
		if strings.EqualFold(h, host) {
			return true, nil
		}
	}
	return false, nil
}

// Token returns the token for host and where it came from.
func Token(host string) (string, Source, error) {
	envToken := os.Getenv(EnvToken)
	if envToken != "" {
		allowed, err := envTokenAllowed(host)
		if err != nil {
			return "", SourceNone, err
		}
		if allowed {
			return envToken, SourceEnv, nil
		}
		// Not allowed here, but a stored token is bound to this host by
		// construction, so it is still safe to fall through to one.
	}
	if t, source, ok := storedToken(host); ok {
		return t, source, nil
	}
	if envToken != "" {
		return "", SourceNone, fmt.Errorf(
			"%s is set but %s is not a known environment — declare it under `environments:` in config.yaml, or log in to it with `sporttrax auth login`: %w",
			EnvToken, host, ErrHostNotAllowed)
	}
	return "", SourceNone, ErrNotFound
}

// storedToken looks up the host-keyed token: keychain first, config file
// fallback.
func storedToken(host string) (string, Source, bool) {
	if t, err := keyring.Get(keyringService, host); err == nil && t != "" {
		return t, SourceKeychain, true
	}
	if t, err := fileToken(host); err == nil && t != "" {
		return t, SourceFile, true
	}
	return "", SourceNone, false
}

// Save stores the token for host, preferring the OS keychain and falling
// back to the config file. It returns where the token was stored. The
// config file always records the host (with an empty token when the real
// one is in the keychain) so StoredHosts can enumerate environments.
func Save(host, token string) (Source, error) {
	if err := keyring.Set(keyringService, host, token); err == nil {
		if err := saveFileToken(host, ""); err != nil {
			return SourceNone, err
		}
		return SourceKeychain, nil
	}
	if err := saveFileToken(host, token); err != nil {
		return SourceNone, err
	}
	return SourceFile, nil
}

// StoredHosts returns the hosts that have a stored login, best-effort.
func StoredHosts() []string {
	hosts, _, err := readHosts()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(hosts))
	for h := range hosts {
		names = append(names, h)
	}
	return names
}

// Delete removes the stored token for host from the keychain and config
// file. It returns the sources a token was actually removed from.
func Delete(host string) ([]Source, error) {
	var removed []Source
	if err := keyring.Delete(keyringService, host); err == nil {
		removed = append(removed, SourceKeychain)
	}
	hosts, path, err := readHosts()
	if err != nil {
		return removed, err
	}
	if entry, ok := hosts[host]; ok {
		delete(hosts, host)
		if err := writeHosts(path, hosts); err != nil {
			return removed, err
		}
		// An empty token was just the host marker for a keychain entry,
		// not a token removal worth reporting.
		if entry.Token != "" {
			removed = append(removed, SourceFile)
		}
	}
	return removed, nil
}

// Mask returns a display-safe form of a token.
func Mask(token string) string {
	if len(token) < 12 {
		return "****"
	}
	return token[:4] + "…" + token[len(token)-4:]
}

// ConfigFile returns the path of the fallback token file.
func ConfigFile() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "hosts.json"), nil
}

type hostEntry struct {
	Token string `json:"token"`
}

func readHosts() (map[string]hostEntry, string, error) {
	path, err := ConfigFile()
	if err != nil {
		return nil, "", err
	}
	hosts := map[string]hostEntry{}
	data, err := os.ReadFile(path) //nolint:gosec // fixed path from ConfigFile, not user input
	if errors.Is(err, os.ErrNotExist) {
		return hosts, path, nil
	}
	if err != nil {
		return nil, path, err
	}
	restrict(path)
	if err := json.Unmarshal(data, &hosts); err != nil {
		return nil, path, fmt.Errorf("parsing %s: %w", path, err)
	}
	return hosts, path, nil
}

func writeHosts(path string, hosts map[string]hostEntry) error {
	if len(hosts) == 0 {
		err := os.Remove(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(hosts, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	// WriteFile's mode applies only when it creates the file, so a
	// hosts.json that already existed with looser permissions (restored
	// from a backup, copied between machines, written under a permissive
	// umask) would keep them and take a plaintext token anyway.
	restrict(path)
	return nil
}

// restrict narrows path to owner-only when it is readable by anyone else,
// best-effort: on Windows the mode bits are not meaningful and the
// keychain is the primary store there anyway.
func restrict(path string) {
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm()&0o077 == 0 {
		return
	}
	_ = os.Chmod(path, 0o600)
}

func fileToken(host string) (string, error) {
	hosts, _, err := readHosts()
	if err != nil {
		return "", err
	}
	return hosts[host].Token, nil
}

func saveFileToken(host, token string) error {
	hosts, path, err := readHosts()
	if err != nil {
		return err
	}
	hosts[host] = hostEntry{Token: token}
	return writeHosts(path, hosts)
}
