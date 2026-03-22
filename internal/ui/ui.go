// Package ui provides styled terminal output for thaw CLI commands.
package ui

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/colorprofile"
	"github.com/sargunv/thaw/internal/state"
)

const verbWidth = 12 // len("Materialized"), the longest verb

// Printer handles styled CLI output.
type Printer struct {
	w     *colorprofile.Writer
	verb  lipgloss.Style
	muted lipgloss.Style
}

// NewPrinter creates a Printer with styles adapted to the terminal background.
func NewPrinter(w *colorprofile.Writer, isDark bool) *Printer {
	ld := lipgloss.LightDark(isDark)
	return &Printer{
		w: w,
		verb: lipgloss.NewStyle().
			Bold(true).
			Foreground(ld(lipgloss.Green, lipgloss.Color("#006600"))).
			Width(verbWidth).
			Align(lipgloss.Right),
		muted: lipgloss.NewStyle().
			Foreground(ld(lipgloss.BrightWhite, lipgloss.BrightBlack)),
	}
}

// PrintMaterialized prints the success message for the materialize command.
func (p *Printer) PrintMaterialized(path string) {
	_, _ = fmt.Fprintf(p.w, "%s %s\n", p.verb.Render("Materialized"), path)
}

// PrintRestored prints the success message for the restore command.
func (p *Printer) PrintRestored(path, target string) {
	_, _ = fmt.Fprintf(p.w, "%s %s %s %s\n",
		p.verb.Render("Restored"),
		path,
		p.muted.Render("->"),
		target,
	)
}

// PrintUntracked prints the success message for the untrack command.
func (p *Printer) PrintUntracked(path string) {
	_, _ = fmt.Fprintf(p.w, "%s %s\n", p.verb.Render("Untracked"), path)
}

// PrintStatus prints the status table.
func (p *Printer) PrintStatus(entries map[string]state.Entry) {
	paths := slices.Sorted(maps.Keys(entries))

	rows := make([][]string, len(paths))
	for i, path := range paths {
		e := entries[path]
		rows[i] = []string{
			path,
			e.Target,
			e.MaterializedAt.Local().Format(time.DateTime),
		}
	}

	t := table.New().
		Headers("PATH", "ORIGINAL TARGET", "MATERIALIZED AT").
		Rows(rows...).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderRow(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return p.muted
			}
			if col == 2 {
				return p.muted
			}
			return lipgloss.NewStyle()
		})

	_, _ = fmt.Fprintln(p.w, t)
}

// PrintNoMaterialized prints the empty-state message.
func (p *Printer) PrintNoMaterialized() {
	_, _ = fmt.Fprintln(p.w, p.muted.Render("No materialized files"))
}
