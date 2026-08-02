package ui

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
)

func TestTableRendersTSVWhenPiped(t *testing.T) {
	var buf bytes.Buffer
	tbl := Table{
		Headers: []string{"NAME", "URL"},
		Rows: [][]string{
			{"production", "https://sporttrax.com"},
			{"testing", "https://testing.sporttrax.com"},
		},
	}
	if err := tbl.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := "production\thttps://sporttrax.com\ntesting\thttps://testing.sporttrax.com\n"
	if buf.String() != want {
		t.Fatalf("piped output = %q, want header-less TSV %q", buf.String(), want)
	}
}

func TestSanitizeStripsTerminalControlSequences(t *testing.T) {
	cases := map[string]string{
		"Evil\x1b[2J Meet":              "Evil Meet",   // clear screen
		"Evil\x1b]0;pwned\x07 Meet":     "Evil Meet",   // window title (OSC)
		"\x1b[31mRED\x1b[0m":            "RED",         // color
		"Boise\tID":                     "Boise ID",    // would break TSV framing
		"line1\nline2":                  "line1 line2", // would forge a row
		"State Championship":            "State Championship",
		"Ana Ruiz — Sprint 100m (élan)": "Ana Ruiz — Sprint 100m (élan)", // non-ASCII survives
	}
	for in, want := range cases {
		if got := Sanitize(in); got != want {
			t.Errorf("Sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTableAndKeyValuesSanitizeServerText(t *testing.T) {
	var buf bytes.Buffer
	tbl := Table{
		Headers: []string{"ID", "NAME"},
		Rows:    [][]string{{"1", "Evil\x1b[2J\x1b]0;pwned\x07 Meet"}},
	}
	if err := tbl.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.ContainsRune(buf.String(), 0x1b) {
		t.Fatalf("table passed an escape through: %q", buf.String())
	}
	if buf.String() != "1\tEvil Meet\n" {
		t.Fatalf("table output = %q", buf.String())
	}

	buf.Reset()
	if err := KeyValues(&buf, [][2]string{{"Name", "Evil\x1b]0;pwned\x07 Meet"}}); err != nil {
		t.Fatalf("KeyValues: %v", err)
	}
	if strings.ContainsRune(buf.String(), 0x1b) {
		t.Fatalf("key-values passed an escape through: %q", buf.String())
	}
}

// Sanitizing must not mutate the caller's rows — commands reuse them.
func TestTableRenderLeavesRowsUntouched(t *testing.T) {
	rows := [][]string{{"1", "Evil\x1b[2J Meet"}}
	tbl := Table{Headers: []string{"ID", "NAME"}, Rows: rows}
	if err := tbl.Render(&bytes.Buffer{}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if rows[0][1] != "Evil\x1b[2J Meet" {
		t.Fatalf("Render mutated the caller's row: %q", rows[0][1])
	}
}

func TestCSVKeepsHeadersAndQuotesSeparators(t *testing.T) {
	var buf bytes.Buffer
	tbl := Table{
		Headers: []string{"ID", "NAME", "VENUE"},
		Rows: [][]string{
			{"1", "State Championship", "Memorial Stadium — Boise, ID"},
			{"2", `He said "go"`, ""},
		},
	}
	if err := tbl.RenderCSV(&buf); err != nil {
		t.Fatalf("RenderCSV: %v", err)
	}
	// Unlike piped TSV, CSV keeps its header: it is opened in a
	// spreadsheet, where labelled columns are the point.
	want := "ID,NAME,VENUE\n" +
		"1,State Championship,\"Memorial Stadium — Boise, ID\"\n" +
		"2,\"He said \"\"go\"\"\",\n"
	if buf.String() != want {
		t.Fatalf("CSV = %q, want %q", buf.String(), want)
	}

	// It must survive a round trip through a real CSV reader.
	rows, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	if err != nil {
		t.Fatalf("output is not valid CSV: %v", err)
	}
	if len(rows) != 3 || rows[1][2] != "Memorial Stadium — Boise, ID" {
		t.Fatalf("round trip lost data: %v", rows)
	}
}

// A cell a spreadsheet reads as a formula executes on open, and these
// values carry user-submitted names.
func TestCSVNeutralizesFormulaInjection(t *testing.T) {
	dangerous := map[string]string{
		`=HYPERLINK("http://evil","click")`: `'=HYPERLINK("http://evil","click")`,
		`+1+1`:                              `'+1+1`,
		`@SUM(A1:A9)`:                       `'@SUM(A1:A9)`,
		`-cmd|' /c calc'!A0`:                `'-cmd|' /c calc'!A0`,
	}
	// Negative numbers are data here (wind, points), not formulas —
	// quoting them as text would break every spreadsheet that reads them.
	safe := []string{"-1.2", "-0.5", "-3", "1.2", "State Championship", ""}

	for in, want := range dangerous {
		if got := csvSafe(in); got != want {
			t.Errorf("csvSafe(%q) = %q, want %q", in, got, want)
		}
	}
	for _, in := range safe {
		if got := csvSafe(in); got != in {
			t.Errorf("csvSafe(%q) = %q, want it unchanged", in, got)
		}
	}
}

func TestKeyValuesCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := KeyValuesCSV(&buf, [][2]string{{"ID", "7"}, {"Name", "Regional, Final"}}); err != nil {
		t.Fatalf("KeyValuesCSV: %v", err)
	}
	want := "field,value\nID,7\nName,\"Regional, Final\"\n"
	if buf.String() != want {
		t.Fatalf("KeyValuesCSV = %q, want %q", buf.String(), want)
	}
}

// CSV is a rendered form, so it carries the same sanitization as the
// table and detail views.
func TestCSVSanitizesServerText(t *testing.T) {
	var buf bytes.Buffer
	tbl := Table{Headers: []string{"ID", "NAME"}, Rows: [][]string{{"1", "Evil\x1b[2J Meet"}}}
	if err := tbl.RenderCSV(&buf); err != nil {
		t.Fatalf("RenderCSV: %v", err)
	}
	if strings.ContainsRune(buf.String(), 0x1b) {
		t.Fatalf("CSV passed an escape through: %q", buf.String())
	}
	if buf.String() != "ID,NAME\n1,Evil Meet\n" {
		t.Fatalf("CSV = %q", buf.String())
	}
}

func TestJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, map[string]int{"a": 1}); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var round map[string]int
	if err := json.Unmarshal(buf.Bytes(), &round); err != nil || round["a"] != 1 {
		t.Fatalf("invalid JSON output %q: %v", buf.String(), err)
	}
}
