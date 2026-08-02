package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sporttrax-inc/sporttrax-cli/internal/api"
	"github.com/sporttrax-inc/sporttrax-cli/internal/auth"
)

func stubMeAPI(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/public-api/v1/me", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"user": {"id": 42, "name": "Jeff Hansen", "role": "admin"},
			"token": {"name": "cli", "abilities": ["public-api"], "last_used_at": null},
			"access": {"public_api": true, "reason": null},
			"rate_limit": {"per_minute": null, "per_day": null}}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestAuthStatusShowsIdentity(t *testing.T) {
	setupCLITest(t)
	srv := stubMeAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "auth", "status")
	if err != nil {
		t.Fatalf("auth status: %v", err)
	}
	if !strings.Contains(out, "User:   Jeff Hansen (admin)") || !strings.Contains(out, "Status: ✓ valid") {
		t.Fatalf("identity missing from status:\n%s", out)
	}
}

func TestAuthStatusJSONIncludesIdentity(t *testing.T) {
	setupCLITest(t)
	srv := stubMeAPI(t)

	out, _, err := runCommand(t, "--api-url", srv.URL, "auth", "status", "--json")
	if err != nil {
		t.Fatalf("auth status --json: %v", err)
	}
	var status map[string]any
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if status["user"] != "Jeff Hansen" || status["role"] != "admin" || status["valid"] != true {
		t.Fatalf("identity missing from JSON status: %v", status)
	}
}

func TestAuthStatusOmitsRoleWhenNull(t *testing.T) {
	setupCLITest(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/public-api/v1/me", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"user": {"id": 7, "name": "Pat Runner", "role": null},
			"token": {"name": "cli", "abilities": ["public-api"], "last_used_at": null},
			"access": {"public_api": true, "reason": null},
			"rate_limit": {"per_minute": 15, "per_day": 1000}}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	out, _, err := runCommand(t, "--api-url", srv.URL, "auth", "status")
	if err != nil {
		t.Fatalf("auth status: %v", err)
	}
	userLine := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "User:") {
			userLine = line
		}
	}
	if userLine != "User:   Pat Runner" {
		t.Fatalf("role must be omitted entirely when null, got %q in:\n%s", userLine, out)
	}

	out, _, err = runCommand(t, "--api-url", srv.URL, "auth", "status", "--json")
	if err != nil {
		t.Fatalf("auth status --json: %v", err)
	}
	if strings.Contains(out, `"role"`) {
		t.Fatalf("role key must be absent from JSON when null:\n%s", out)
	}
}

func TestInsecureWarningOnlyForFlag(t *testing.T) {
	setupCLITest(t)
	srv := stubMeAPI(t)

	// config.yaml-declared insecure is recorded consent: no warning.
	cfgDir := os.Getenv("XDG_CONFIG_HOME") + "/sporttrax-cli"
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := fmt.Sprintf("environments:\n  local:\n    api_url: %s\n    insecure: true\n", srv.URL)
	if err := os.WriteFile(cfgDir+"/config.yaml", []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := runCommand(t, "--env", "local", "auth", "status")
	if err != nil {
		t.Fatalf("auth status: %v", err)
	}
	if strings.Contains(stderr, "TLS certificate verification disabled") {
		t.Fatalf("config-declared insecure must not warn:\n%s", stderr)
	}

	// The ad-hoc flag warns every time.
	_, stderr, err = runCommand(t, "--api-url", srv.URL, "--insecure", "auth", "status")
	if err != nil {
		t.Fatalf("auth status --insecure: %v", err)
	}
	if !strings.Contains(stderr, "TLS certificate verification disabled") {
		t.Fatalf("--insecure flag must warn:\n%s", stderr)
	}
}

func TestInsecureRefusedForStockEnvironments(t *testing.T) {
	setupCLITest(t)

	// A config.yaml that quietly turns off certificate verification for
	// production would expose the token to interception with no output.
	cfgDir := os.Getenv("XDG_CONFIG_HOME") + "/sporttrax-cli"
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := "environments:\n  production:\n    insecure: true\n"
	if err := os.WriteFile(cfgDir+"/config.yaml", []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, err := runCommand(t, "auth", "status")
	if err == nil || !strings.Contains(err.Error(), "refusing to disable TLS") {
		t.Fatalf("insecure on a stock environment must be refused, got %v", err)
	}
}

func TestInsecureWarnsForRemoteHostEvenFromConfig(t *testing.T) {
	setupCLITest(t)

	// Consent recorded in config.yaml buys silence only for the local
	// machine; a remote host warns on every command.
	cfgDir := os.Getenv("XDG_CONFIG_HOME") + "/sporttrax-cli"
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := "environments:\n  ultra:\n    api_url: https://ultra.sporttrax.dev\n    insecure: true\n"
	if err := os.WriteFile(cfgDir+"/config.yaml", []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	// logout resolves the environment without making a request, so the
	// policy is exercised without touching the network.
	_, stderr, err := runCommand(t, "--env", "ultra", "auth", "logout")
	if err != nil {
		t.Fatalf("auth logout: %v", err)
	}
	if !strings.Contains(stderr, "verification disabled for ultra.sporttrax.dev") {
		t.Fatalf("remote insecure host must warn:\n%s", stderr)
	}
}

func TestPlaintextAPIURLRefused(t *testing.T) {
	setupCLITest(t)

	_, _, err := runCommand(t, "--api-url", "http://sporttrax.com", "meet", "list")
	if err == nil || !strings.Contains(err.Error(), "plaintext HTTP") {
		t.Fatalf("http:// to a remote host must be refused, got %v", err)
	}

	// Local development over http stays workable.
	if err := api.ValidateBaseURL("http://localhost:8555"); err != nil {
		t.Fatalf("localhost http must remain allowed: %v", err)
	}
}

func TestEnvTokenNotSentToUndeclaredHost(t *testing.T) {
	setupCLITest(t) // sets SPORTTRAX_API_TOKEN

	_, _, err := runCommand(t, "--api-url", "https://attacker.example", "meet", "list")
	if !errors.Is(err, auth.ErrHostNotAllowed) {
		t.Fatalf("env token must not follow --api-url to an undeclared host, got %v", err)
	}
}

func TestAuthLoginWithTokenValidatesViaMe(t *testing.T) {
	setupCLITest(t)
	srv := stubMeAPI(t)
	t.Setenv("SPORTTRAX_API_TOKEN", "") // login must not shortcut via env

	rootCmd.SetIn(strings.NewReader("test-token\n"))
	defer rootCmd.SetIn(nil)
	out, _, err := runCommand(t, "--api-url", srv.URL, "auth", "login", "--with-token")
	if err != nil {
		t.Fatalf("auth login: %v", err)
	}
	if !strings.Contains(out, "Logged in to") || !strings.Contains(out, "as Jeff Hansen (admin)") {
		t.Fatalf("login should report identity:\n%s", out)
	}
}

func TestAuthLoginRejectsBadToken(t *testing.T) {
	setupCLITest(t)
	srv := stubMeAPI(t)
	t.Setenv("SPORTTRAX_API_TOKEN", "")

	rootCmd.SetIn(strings.NewReader("wrong-token\n"))
	defer rootCmd.SetIn(nil)
	_, _, err := runCommand(t, "--api-url", srv.URL, "auth", "login", "--with-token")
	if err == nil || !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("bad token must fail login, got %v", err)
	}
}
