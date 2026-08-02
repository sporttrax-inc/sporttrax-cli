// Package ui renders command output following the conventions of gh/aws/
// stripe: human-formatted (aligned, lightly styled) tables when stdout is a
// terminal, plain tab-separated values without headers when piped, and JSON
// on demand. All data commands should render through this package so format
// switching stays one code path.
package ui

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

// IsTTY reports whether w is an interactive terminal.
func IsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// Sanitize makes server-supplied text safe to write to a terminal. Meet,
// athlete, team, and venue names are user-submitted content on the
// platform, so they can carry escape sequences that clear the screen,
// forge output, or retitle the window; tabs and newlines would also break
// the framing of piped rows. Only the rendered forms are sanitized —
// --json and the MCP tools stay verbatim, since they are the data
// contract and are not interpreted by a terminal.
func Sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			return ' '
		case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f:
			return -1
		}
		return r
	}, ansi.Strip(s))
}

// sanitizeRows returns rows with every cell sanitized, leaving the
// caller's slice untouched.
func sanitizeRows(rows [][]string) [][]string {
	out := make([][]string, len(rows))
	for i, row := range rows {
		cells := make([]string, len(row))
		for j, cell := range row {
			cells[j] = Sanitize(cell)
		}
		out[i] = cells
	}
	return out
}

// JSON writes v as indented JSON.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Note prints a dimmed advisory line on a TTY; silent when piped so rows
// stay machine-clean.
func Note(w io.Writer, msg string) {
	if !IsTTY(w) {
		return
	}
	faint := lipgloss.NewStyle().Faint(true)
	fmt.Fprintln(w, faint.Render(msg))
}

// KeyValues renders a two-column detail view: bold aligned keys on a TTY,
// key/value TSV lines when piped.
func KeyValues(w io.Writer, pairs [][2]string) error {
	if !IsTTY(w) {
		for _, p := range pairs {
			if _, err := fmt.Fprintf(w, "%s\t%s\n", p[0], Sanitize(p[1])); err != nil {
				return err
			}
		}
		return nil
	}
	width := 0
	for _, p := range pairs {
		if len(p[0]) > width {
			width = len(p[0])
		}
	}
	bold := lipgloss.NewStyle().Bold(true)
	for _, p := range pairs {
		key := fmt.Sprintf("%-*s", width, p[0])
		if _, err := fmt.Fprintf(w, "%s  %s\n", bold.Render(key), Sanitize(p[1])); err != nil {
			return err
		}
	}
	return nil
}

// csvSafe neutralizes spreadsheet formula injection. A cell that a
// spreadsheet reads as a formula runs when the file is opened, and these
// values carry user-submitted names, so `=HYPERLINK(...)` in a meet name
// would execute on the user's machine. Prefixing with an apostrophe makes
// the cell literal text — the same defense in spirit as sanitizing
// terminal escapes, for the program that opens the file instead of the
// terminal that prints it.
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '@':
		return "'" + s
	case '-':
		// A leading minus is usually data here — wind, points, a
		// descending sort value — so only guard what isn't a number.
		if _, err := strconv.ParseFloat(s, 64); err == nil {
			return s
		}
		return "'" + s
	}
	return s
}

// csvRow prepares one record: sanitized like every other rendered form,
// then made inert for spreadsheets.
func csvRow(cells []string) []string {
	out := make([]string, len(cells))
	for i, c := range cells {
		out[i] = csvSafe(Sanitize(c))
	}
	return out
}

// CSV writes a header row followed by records, RFC 4180 quoted. Unlike
// the piped TSV form, CSV keeps its header: it is written to be opened in
// a spreadsheet, where labelled columns are the point.
func CSV(w io.Writer, headers []string, rows [][]string) error {
	cw := csv.NewWriter(w)
	if len(headers) > 0 {
		if err := cw.Write(csvRow(headers)); err != nil {
			return err
		}
	}
	for _, row := range rows {
		if err := cw.Write(csvRow(row)); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// KeyValuesCSV writes a detail view as a two-column field/value sheet.
func KeyValuesCSV(w io.Writer, pairs [][2]string) error {
	rows := make([][]string, 0, len(pairs))
	for _, p := range pairs {
		rows = append(rows, []string{p[0], p[1]})
	}
	return CSV(w, []string{"field", "value"}, rows)
}

// Table is a set of rows for a list command.
type Table struct {
	Headers []string
	Rows    [][]string
}

// RenderCSV writes the table as CSV, headers included.
func (t Table) RenderCSV(w io.Writer) error {
	return CSV(w, t.Headers, t.Rows)
}

// Render writes the table: styled and aligned with headers on a TTY,
// header-less tab-separated lines when piped (grep/cut friendly).
func (t Table) Render(w io.Writer) error {
	rows := sanitizeRows(t.Rows)
	if !IsTTY(w) {
		for _, row := range rows {
			if _, err := fmt.Fprintln(w, strings.Join(row, "\t")); err != nil {
				return err
			}
		}
		return nil
	}

	// Rounded borders match the huh prompt aesthetic; some older Windows
	// terminal fonts render the rounded corner glyphs as boxes, so fall
	// back to square corners there. Borders are dimmed so the data
	// carries the contrast.
	border := lipgloss.RoundedBorder()
	if runtime.GOOS == "windows" {
		border = lipgloss.NormalBorder()
	}
	headerStyle := lipgloss.NewStyle().Bold(true)
	cellStyle := lipgloss.NewStyle().Padding(0, 1)
	tbl := table.New().
		Border(border).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		BorderHeader(true).
		BorderColumn(true).
		BorderRow(false).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return cellStyle.Inherit(headerStyle)
			}
			return cellStyle
		}).
		Headers(t.Headers...).
		Rows(rows...)
	_, err := fmt.Fprintln(w, tbl.Render())
	return err
}
