// gen-notices regenerates THIRD_PARTY_NOTICES.md from the modules the
// released binary actually links. The MIT and BSD licenses covering most
// of the dependency tree require their copyright notice and license text
// to travel with binary distributions, so the generated file ships inside
// the release archives. Run via `make notices`; never edit the output by
// hand.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// module is one dependency of the built binary.
type module struct {
	Path    string
	Version string
	Dir     string
}

// licenseNames are the filenames upstream projects use, in the order we
// prefer them. NOTICE is collected in addition to a license, not instead
// of one: Apache-2.0 section 4(d) requires it to be carried along.
var licenseNames = []string{
	"LICENSE", "LICENSE.txt", "LICENSE.md", "LICENCE", "LICENCE.txt",
	"COPYING", "COPYING.txt", "LICENSE-MIT", "LICENSE-APACHE",
}

func main() {
	out := flag.String("out", "THIRD_PARTY_NOTICES.md", "output file")
	pkg := flag.String("pkg", "./cmd/sporttrax", "package whose link-time dependencies are listed")
	flag.Parse()

	mods, err := linkedModules(*pkg)
	if err != nil {
		fail(err)
	}
	doc, missing, err := render(mods)
	if err != nil {
		fail(err)
	}
	// A dependency whose license we cannot find must not pass silently —
	// shipping it unattributed is the exact failure this file prevents.
	if len(missing) > 0 {
		fail(fmt.Errorf("no license file found for: %s", strings.Join(missing, ", ")))
	}
	if err := os.WriteFile(*out, []byte(doc), 0o600); err != nil {
		fail(err)
	}
	fmt.Printf("third-party notices for %d modules written to %s\n", len(mods), *out)
}

// linkedModules returns the modules contributing code to pkg's binary,
// deduplicated and sorted. Test-only dependencies are excluded: they are
// not distributed, so they carry no notice obligation.
func linkedModules(pkg string) ([]module, error) {
	//nolint:gosec // build-time generator: pkg is a --pkg flag set by a developer running `make notices`, never remote input, and it is passed as an argv element rather than through a shell
	cmd := exec.Command("go", "list", "-deps",
		"-f", "{{with .Module}}{{.Path}}\t{{.Version}}\t{{.Dir}}{{end}}", pkg)
	cmd.Stderr = os.Stderr
	outBytes, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}

	self, err := selfModule()
	if err != nil {
		return nil, err
	}

	seen := map[string]module{}
	for _, line := range strings.Split(string(outBytes), "\n") {
		fields := strings.Split(strings.TrimSpace(line), "\t")
		if len(fields) != 3 || fields[0] == "" || fields[0] == self {
			continue // stdlib has no module; skip our own code too
		}
		seen[fields[0]] = module{Path: fields[0], Version: fields[1], Dir: fields[2]}
	}

	mods := make([]module, 0, len(seen))
	for _, m := range seen {
		mods = append(mods, m)
	}
	sort.Slice(mods, func(i, j int) bool { return mods[i].Path < mods[j].Path })
	return mods, nil
}

func selfModule() (string, error) {
	cmd := exec.Command("go", "list", "-m")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go list -m: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// render builds the notices document, returning any modules whose license
// text could not be located.
func render(mods []module) (doc string, missing []string, err error) {
	var b strings.Builder
	b.WriteString(`# Third-party notices

The ` + "`sporttrax`" + ` binary is statically linked and includes code from the
Go modules listed below. Each is distributed under its own license, and
those licenses are reproduced here in full, as their terms require.

This file is generated from the module graph by ` + "`make notices`" + ` — do not
edit it by hand. The license of the sporttrax CLI itself is in
[LICENSE](LICENSE).

`)
	for _, m := range mods {
		texts, err := licenseTexts(m.Dir)
		if err != nil {
			return "", nil, err
		}
		if len(texts) == 0 {
			missing = append(missing, m.Path+"@"+m.Version)
			continue
		}
		fmt.Fprintf(&b, "---\n\n## %s\n\nVersion: %s\n\n", m.Path, m.Version)
		for _, t := range texts {
			fmt.Fprintf(&b, "### %s\n\n```\n%s\n```\n\n", t.name, strings.TrimRight(t.body, "\n"))
		}
	}
	return b.String(), missing, nil
}

type licenseText struct {
	name string
	body string
}

// licenseTexts collects the license (and any NOTICE) shipped in a module
// directory.
func licenseTexts(dir string) ([]licenseText, error) {
	if dir == "" {
		return nil, nil
	}
	var texts []licenseText
	for _, name := range licenseNames {
		body, err := readIfPresent(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		if body != "" {
			texts = append(texts, licenseText{name: name, body: body})
			break // one license file per module is the norm
		}
	}
	for _, name := range []string{"NOTICE", "NOTICE.txt"} {
		body, err := readIfPresent(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		if body != "" {
			texts = append(texts, licenseText{name: name, body: body})
			break
		}
	}
	return texts, nil
}

func readIfPresent(path string) (string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is a module cache dir from go list, not user input
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
