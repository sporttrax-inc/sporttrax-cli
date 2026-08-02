package version

import "testing"

func TestShortCommitMatchesLdflagsShape(t *testing.T) {
	// goreleaser injects .ShortCommit; the build-info path carries a full
	// SHA, so it is trimmed to the same width rather than printing two
	// different shapes depending on how the binary was made.
	if got := shortCommit("a3b65b1a8e43ff0011223344556677889900aabb"); got != "a3b65b1" {
		t.Fatalf("shortCommit = %q, want a3b65b1", got)
	}
	if got := shortCommit("abc"); got != "abc" {
		t.Fatalf("short input must pass through, got %q", got)
	}
	if got := shortCommit(""); got != "" {
		t.Fatalf("empty input must pass through, got %q", got)
	}
}

// The package always reports something usable: ldflags win when present,
// build info fills in otherwise, and the defaults remain as a last resort.
func TestVersionIsNeverEmpty(t *testing.T) {
	if Version == "" {
		t.Error("Version must never be empty")
	}
	if Commit == "" {
		t.Error("Commit must never be empty")
	}
}

// An injected stamp is authoritative — build info must not overwrite it.
// This is why the vars start empty: `make build` injects the literal
// "dev", and inferring over the top of it would make local builds claim a
// version nobody released.
func TestBuildInfoDoesNotOverrideLdflags(t *testing.T) {
	origVersion, origCommit := Version, Commit
	t.Cleanup(func() { Version, Commit = origVersion, origCommit })

	for _, injected := range []string{"1.2.3", "dev"} {
		Version, Commit = injected, "deadbee"
		stampFromBuildInfo()
		if Version != injected || Commit != "deadbee" {
			t.Fatalf("injected %q was overwritten: %s (%s)", injected, Version, Commit)
		}
	}
}
