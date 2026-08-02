// Package version reports the build's identity.
//
// Release binaries get it from ldflags (see .goreleaser.yaml). Builds made
// with `go install module@version` do not run those, so the values fall
// back to what the toolchain embeds — otherwise every go-install user
// would report "dev", and "what version are you on?" would have no useful
// answer for half the people who have it.
package version

import "runtime/debug"

// Set via ldflags at build time. They start empty rather than at "dev" so
// an injected value is distinguishable from an absent one: `make build`
// injects the literal "dev", and that has to win over anything inferred.
var (
	Version string
	Commit  string
)

func init() {
	stampFromBuildInfo()
	if Version == "" {
		Version = "dev"
	}
	if Commit == "" {
		Commit = "none"
	}
}

// stampFromBuildInfo fills in only what ldflags left empty.
func stampFromBuildInfo() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	// "(devel)" is what an unstamped in-module build reports; it says
	// less than "dev", so it is not worth adopting.
	if Version == "" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}
	if Commit != "" {
		return
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && s.Value != "" {
			Commit = shortCommit(s.Value)
			return
		}
	}
}

// shortCommit trims a full SHA to the 7 characters goreleaser injects, so
// both build paths print the same shape.
func shortCommit(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
