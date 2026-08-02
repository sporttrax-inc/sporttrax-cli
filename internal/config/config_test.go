package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("AppData", dir)
	path, err := File()
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestBuiltinsWithoutConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("AppData", dir)

	env, err := Resolve("testing")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if env.APIURL != "https://testing.sporttrax.com" || env.Pusher.Cluster != "us3" {
		t.Fatalf("unexpected builtin: %+v", env)
	}
}

func TestCustomEnvironmentAndPartialOverride(t *testing.T) {
	writeConfig(t, `
environments:
  ultra:
    api_url: https://ultra.sporttrax.dev
    insecure: true
    pusher:
      app_key: 1aeac22772e92e87b632
  production:
    pusher:
      app_key: prodkey123456
`)

	ultra, err := Resolve("ultra")
	if err != nil {
		t.Fatalf("Resolve ultra: %v", err)
	}
	if ultra.APIURL != "https://ultra.sporttrax.dev" || ultra.Pusher.AppKey != "1aeac22772e92e87b632" || !ultra.Insecure {
		t.Fatalf("unexpected custom env: %+v", ultra)
	}

	// Insecure must never leak into environments that don't set it.
	if tst, err := Resolve("testing"); err != nil || tst.Insecure {
		t.Fatalf("testing should not be insecure: %+v err=%v", tst, err)
	}

	// A partial override keeps the builtin URL and cluster defaults.
	prod, err := Resolve("production")
	if err != nil {
		t.Fatalf("Resolve production: %v", err)
	}
	if prod.APIURL != "https://sporttrax.com" || prod.Pusher.AppKey != "prodkey123456" || prod.Pusher.Cluster != "us3" {
		t.Fatalf("partial override lost fields: %+v", prod)
	}
}

func TestUnknownEnvironmentListsKnown(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("AppData", dir)

	_, err := Resolve("nope")
	if err == nil {
		t.Fatal("want error for unknown environment")
	}
	for _, name := range []string{"production", "staging", "testing"} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("error should list %q: %v", name, err)
		}
	}
}
