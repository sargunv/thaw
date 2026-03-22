package ui

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"charm.land/lipgloss/v2"
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
			Foreground(lipgloss.Green).
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

// PrintCleared prints the success message for the clear command.
func (p *Printer) PrintCleared(path string) {
	_, _ = fmt.Fprintf(p.w, "%s %s\n", p.verb.Render("Cleared"), path)
}

// PrintStatus prints the status table with column-aligned entries.
func (p *Printer) PrintStatus(entries map[string]state.Entry) {
	paths := slices.Sorted(maps.Keys(entries))

	maxPathWidth := 0
	maxTargetWidth := 0
	for _, path := range paths {
		if w := lipgloss.Width(path); w > maxPathWidth {
			maxPathWidth = w
		}
		if w := lipgloss.Width(entries[path].Target); w > maxTargetWidth {
			maxTargetWidth = w
		}
	}

	for _, path := range paths {
		e := entries[path]
		pathPad := maxPathWidth - lipgloss.Width(path)
		targetPad := maxTargetWidth - lipgloss.Width(e.Target)
		_, _ = fmt.Fprintf(p.w, "%s%*s %s %s%*s  %s\n",
			path, pathPad, "",
			p.muted.Render("->"),
			e.Target, targetPad, "",
			p.muted.Render(e.MaterializedAt.Local().Format(time.DateTime)),
		)
	}
}

// PrintNoMaterialized prints the empty-state message.
func (p *Printer) PrintNoMaterialized() {
	_, _ = fmt.Fprintln(p.w, p.muted.Render("No materialized files"))
}
