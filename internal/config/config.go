// Package config resolves named SportTrax environments. The stock
// environments (production, staging, testing) are built in, and
// config.yaml in the sporttrax-cli user config dir can override them or
// define custom ones
// such as a local dev instance
// (only non-empty fields override, so setting just a pusher app_key on a
// stock environment works). Example:
//
//	environments:
//	  ultra:
//	    api_url: https://ultra.sporttrax.dev
//	    pusher:
//	      app_key: 1aeac22772e92e87b632
//	      cluster: us3
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Pusher holds the Pusher-protocol connection details for an environment
// (used for WebSocket streaming). Setting host targets any server that
// speaks the same protocol instead of Pusher's own cluster.
type Pusher struct {
	AppKey  string `yaml:"app_key,omitempty"`
	Cluster string `yaml:"cluster,omitempty"`
	Scheme  string `yaml:"scheme,omitempty"`
	Host    string `yaml:"host,omitempty"`
	Port    int    `yaml:"port,omitempty"`
}

// Environment is one SportTrax deployment the CLI can target.
type Environment struct {
	APIURL string `yaml:"api_url,omitempty"`
	// Insecure skips TLS certificate verification — for local/dev
	// instances with self-signed certs. Never needed for the stock
	// environments.
	Insecure bool   `yaml:"insecure,omitempty"`
	Pusher   Pusher `yaml:"pusher,omitempty"`
}

type file struct {
	// Units is the preferred unit system for displayed marks: english or
	// metric. Empty means the mark's native display form. Affects table
	// and detail rendering only — JSON always carries both forms.
	Units        string                 `yaml:"units,omitempty"`
	Environments map[string]Environment `yaml:"environments"`
}

// Units returns the configured display unit system ("english", "metric",
// or "" for native), best-effort.
func Units() string {
	path, err := File()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path) //nolint:gosec // fixed path from File(), not user input
	if err != nil {
		return ""
	}
	var f file
	if err := yaml.Unmarshal(data, &f); err != nil {
		return ""
	}
	return f.Units
}

// Builtin returns the stock environments. Pusher app keys are public
// client identifiers (already shipped in the web/app frontends); the
// server-side app secret never belongs here. URLs and keys match
// sporttrax-app's per-environment config.
func Builtin() map[string]Environment {
	pusher := func(appKey string) Pusher {
		return Pusher{AppKey: appKey, Cluster: "us3", Scheme: "https"}
	}
	return map[string]Environment{
		"production": {APIURL: "https://sporttrax.com", Pusher: pusher("094d85a65948eb11ae83")},
		"staging":    {APIURL: "https://staging.sporttrax.com", Pusher: pusher("10a2e4d76e39794f9a77")},
		"testing":    {APIURL: "https://testing.sporttrax.com", Pusher: pusher("5788e4c6957412025155")},
	}
}

// hostOf extracts the host of an environment's API URL, "" when unparsable.
func hostOf(apiURL string) string {
	u, err := url.Parse(apiURL)
	if err != nil {
		return ""
	}
	return u.Host
}

// IsBuiltinHost reports whether host belongs to a stock environment. Those
// deployments have real certificates, so a request to relax TLS
// verification for one of them is always a mistake or an attack.
func IsBuiltinHost(host string) bool {
	for _, env := range Builtin() {
		if h := hostOf(env.APIURL); h != "" && strings.EqualFold(h, host) {
			return true
		}
	}
	return false
}

// KnownHosts returns the hosts of every environment the user has declared:
// the stock ones plus anything in config.yaml. Declaring an environment is
// how a user vouches for a host — see auth.Token, which will only send the
// SPORTTRAX_API_TOKEN env var to a host on this list.
func KnownHosts() ([]string, error) {
	envs, err := Load()
	if err != nil {
		return nil, err
	}
	hosts := make([]string, 0, len(envs))
	for _, env := range envs {
		if h := hostOf(env.APIURL); h != "" {
			hosts = append(hosts, h)
		}
	}
	sort.Strings(hosts)
	return hosts, nil
}

// Dir returns the CLI's config directory: $XDG_CONFIG_HOME/sporttrax-cli
// or ~/.config/sporttrax-cli on macOS and Linux alike (the common CLI
// convention, rather than ~/Library/Application Support on macOS), and the
// platform config dir on Windows.
func Dir() (string, error) {
	if runtime.GOOS == "windows" {
		base, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(base, "sporttrax-cli"), nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "sporttrax-cli"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "sporttrax-cli"), nil
}

// File returns the path of the environments config file.
func File() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load returns all environments: builtins overlaid with the config file.
func Load() (map[string]Environment, error) {
	envs := Builtin()

	path, err := File()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path) //nolint:gosec // fixed path from File(), not user input
	if errors.Is(err, os.ErrNotExist) {
		return envs, nil
	}
	if err != nil {
		return nil, err
	}
	var f file
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	for name, override := range f.Environments {
		envs[name] = merge(envs[name], override)
	}
	return envs, nil
}

// Resolve returns the environment with the given name.
func Resolve(name string) (Environment, error) {
	envs, err := Load()
	if err != nil {
		return Environment{}, err
	}
	env, ok := envs[name]
	if !ok {
		names := make([]string, 0, len(envs))
		for n := range envs {
			names = append(names, n)
		}
		sort.Strings(names)
		path, _ := File()
		return Environment{}, fmt.Errorf(
			"unknown environment %q (known: %v; define custom environments in %s)", name, names, path)
	}
	return env, nil
}

// merge overlays non-empty fields of override onto base.
func merge(base, override Environment) Environment {
	if override.APIURL != "" {
		base.APIURL = override.APIURL
	}
	if override.Insecure {
		base.Insecure = true
	}
	if override.Pusher.AppKey != "" {
		base.Pusher.AppKey = override.Pusher.AppKey
	}
	if override.Pusher.Cluster != "" {
		base.Pusher.Cluster = override.Pusher.Cluster
	}
	if override.Pusher.Scheme != "" {
		base.Pusher.Scheme = override.Pusher.Scheme
	}
	if override.Pusher.Host != "" {
		base.Pusher.Host = override.Pusher.Host
	}
	if override.Pusher.Port != 0 {
		base.Pusher.Port = override.Pusher.Port
	}
	return base
}
