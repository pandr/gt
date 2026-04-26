package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/petera/gt/internal/git"
)

var (
	styleBranch    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	styleHeader    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	styleCursor    = lipgloss.NewStyle().Background(lipgloss.Color("237")).Bold(true)
	styleTagged    = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	styleStatusBar = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleToast     = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	styleHelp      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	styleDim       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleAdd       = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleMod       = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleDel       = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

func (m Model) View() string {
	if m.mode == modeHelp {
		return m.helpView()
	}

	var b strings.Builder

	// Branch header
	b.WriteString(m.branchHeader())
	b.WriteString("\n\n")

	// Rows
	visibleStart, visibleEnd := m.visibleRange()
	for i := visibleStart; i < visibleEnd; i++ {
		r := m.rows[i]
		line := m.renderRow(r, i == m.cursor)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Fill remaining lines to keep status bar at bottom
	rendered := visibleEnd - visibleStart
	headerLines := 2 // branch + blank
	statusLines := 1
	if m.mode == modeCommit {
		statusLines = 2
	}
	h := m.height
	if h < 10 {
		h = 24
	}
	remaining := h - headerLines - rendered - statusLines
	for ; remaining > 0; remaining-- {
		b.WriteString("\n")
	}

	// Status bar / commit input
	b.WriteString(m.statusBar())

	return b.String()
}

func (m Model) branchHeader() string {
	if m.status == nil {
		return styleBranch.Render("Loading…")
	}
	branch := m.status.Branch
	if branch == "(detached)" || branch == "" {
		branch = "(detached HEAD)"
	}
	s := styleBranch.Render("On branch " + branch)
	if m.status.Upstream != "" {
		parts := []string{}
		if m.status.Ahead > 0 {
			parts = append(parts, fmt.Sprintf("↑%d", m.status.Ahead))
		}
		if m.status.Behind > 0 {
			parts = append(parts, fmt.Sprintf("↓%d", m.status.Behind))
		}
		info := m.status.Upstream
		if len(parts) > 0 {
			info += "  " + strings.Join(parts, " ")
		}
		s += styleDim.Render("   "+info)
	}
	return s
}

func (m Model) visibleRange() (int, int) {
	if len(m.rows) == 0 {
		return 0, 0
	}
	headerLines := 2
	statusLines := 1
	if m.mode == modeCommit {
		statusLines = 2
	}
	h := m.height
	if h < 10 {
		h = 24 // sensible default before first WindowSizeMsg
	}
	available := h - headerLines - statusLines
	if available < 1 {
		available = 1
	}

	// Simple scroll: keep cursor visible
	start := m.cursor - available/2
	if start < 0 {
		start = 0
	}
	end := start + available
	if end > len(m.rows) {
		end = len(m.rows)
		start = end - available
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

func (m Model) renderRow(r row, isCursor bool) string {
	line := m.rowContent(r)

	tagged := tagKey(r) != "" && m.tags[tagKey(r)]
	if tagged {
		line = styleTagged.Render("*") + line[1:]
	}

	if isCursor {
		// pad to width so the highlight spans the full line
		padded := line
		if m.width > 0 {
			visible := visibleLen(line)
			if visible < m.width {
				padded = line + strings.Repeat(" ", m.width-visible)
			}
		}
		return styleCursor.Render(padded)
	}
	return line
}

func (m Model) rowContent(r row) string {
	indent := strings.Repeat("  ", r.depth)
	switch r.kind {
	case rowSectionHeader:
		return m.sectionHeader(r.section)
	case rowFile:
		if r.file == nil {
			return ""
		}
		return indent + m.fileRow(r.file, r.section)
	case rowCommit:
		if r.commit == nil {
			return ""
		}
		sha := r.commit.SHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		return "  " + styleDim.Render(sha) + " " + r.commit.Title
	case rowDir:
		return m.dirRow(r)
	case rowSeparator:
		return ""
	}
	return ""
}

func (m Model) dirRow(r row) string {
	indent := strings.Repeat("  ", r.depth)
	arrow := "▶"
	if m.openDirs[r.dirPath] {
		arrow = "▼"
	}
	name := r.dirPath
	extra := ""
	if !m.openDirs[r.dirPath] {
		if children, ok := m.dirContents[r.dirPath]; ok {
			extra = styleDim.Render(fmt.Sprintf("  (%d files)", len(children)))
		}
	}
	return indent + "  " + styleDim.Render(arrow) + " " + styleAdd.Render(name) + extra
}

func (m Model) sectionHeader(s git.Section) string {
	var name string
	var count int
	switch s {
	case git.SectionUntracked:
		name = "Untracked"
		if m.status != nil {
			count = len(m.status.Untracked)
		}
	case git.SectionUnstaged:
		name = "Unstaged"
		if m.status != nil {
			count = len(m.status.Unstaged)
		}
	case git.SectionStaged:
		name = "Staged"
		if m.status != nil {
			count = len(m.status.Staged)
		}
	case git.SectionLog:
		name = "Recent commits"
		return styleHeader.Render(name)
	case git.SectionWorkingTree:
		if m.wtOpen {
			return styleHeader.Render("Working tree  ./")
		}
		label := "Working tree  ./"
		if len(m.wtFiles) > 0 {
			label += fmt.Sprintf("  (%d files)", len(m.wtFiles))
		}
		return styleHeader.Render(label) + styleDim.Render("  → to expand")
	}
	return styleHeader.Render(fmt.Sprintf("%s (%d)", name, count))
}

func (m Model) fileRow(f *git.FileEntry, section git.Section) string {
	xy := f.XY
	var indicator string
	var indicatorStyle lipgloss.Style

	switch section {
	case git.SectionUntracked:
		indicator = "?"
		indicatorStyle = styleAdd
	case git.SectionUnstaged:
		y := rune(xy[1])
		indicator, indicatorStyle = xyIndicator(y)
	case git.SectionStaged:
		x := rune(xy[0])
		indicator, indicatorStyle = xyIndicator(x)
	case git.SectionWorkingTree:
		return "   " + styleDim.Render(f.Path)
	}

	return "  " + indicatorStyle.Render(indicator) + " " + f.Path
}

func xyIndicator(ch rune) (string, lipgloss.Style) {
	switch ch {
	case 'A':
		return "A", styleAdd
	case 'M', 'T', 'U':
		return "M", styleMod
	case 'D':
		return "D", styleDel
	case 'R':
		return "R", styleMod
	case 'C':
		return "C", styleAdd
	}
	return string(ch), styleDim
}

func (m Model) statusBar() string {
	if m.mode == modeHelp {
		return ""
	}
	if m.mode == modeCommit {
		return "\n" + m.commitInput.View()
	}
	if m.mode == modeTagPrefix {
		return styleStatusBar.Render(";_ — waiting for command (s=stage  u=unstage  d=diff)")
	}
	if m.mode == modeConfirm {
		return styleToast.Render(m.confirmPrompt)
	}

	var parts []string
	if m.toast != "" {
		parts = append(parts, styleToast.Render(m.toast))
	} else {
		n := len(m.tags)
		if n > 0 {
			parts = append(parts, styleTagged.Render(fmt.Sprintf("[%d tagged]", n)))
		}
		parts = append(parts, styleStatusBar.Render("c=commit  d=diff  s=stage  u=unstage  t=tag  ?=help  q=quit"))
	}
	return strings.Join(parts, "   ")
}

func (m Model) helpView() string {
	var b strings.Builder
	b.WriteString("Key bindings\n\n")
	for _, kb := range keyBindings {
		b.WriteString(fmt.Sprintf("  %-22s %s\n", kb.Key, kb.Desc))
	}
	b.WriteString("\nPress ? or q to close")
	return styleHelp.Render(b.String())
}

// visibleLen returns the visible (non-ANSI) length of a string.
// Rough approximation: strip ANSI and count runes.
func visibleLen(s string) int {
	inEscape := false
	n := 0
	for _, r := range s {
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		if r == '\x1b' {
			inEscape = true
			continue
		}
		n++
	}
	return n
}
