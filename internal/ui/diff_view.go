package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/petera/gt/internal/git"
)

// diffView renders the full inline diff screen.
func (m Model) diffView() string {
	d := m.diff
	if d == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString(m.diffTitleBar())
	b.WriteString("\n")
	b.WriteString(railFaint.Render("┊"))
	b.WriteString("\n")

	h := m.height
	if h < 10 {
		h = 24
	}
	// title + blank + separator + hints = 4 fixed lines
	contentLines := h - 4

	start, end := m.diffVisibleRange(contentLines)
	for i := start; i < end; i++ {
		b.WriteString(m.renderDiffViewLine(i))
		b.WriteString("\n")
	}

	// Pad to keep separator pinned at bottom
	rendered := end - start
	for range contentLines - rendered {
		b.WriteString("\n")
	}

	// Separator
	sepWidth := m.width
	if sepWidth <= 0 {
		sepWidth = 64
	}
	b.WriteString(fgFaint.Render(strings.Repeat("─", sepWidth)))
	b.WriteString("\n")

	if m.diffSearching {
		b.WriteString(styleStatusBar.Render("/") + " " + m.commitInput.View())
	} else {
		hints := "j/k=line  space/ctrl+d/u=page  ]/[=hunk  /=search  e=editor  L=less  q=back"
		if m.diffSearch != "" {
			hints = "n/N=match  " + hints
		}
		b.WriteString(styleStatusBar.Render(hints))
	}

	return b.String()
}

// diffTitleBar renders the top line of the diff view.
// File diffs: ┊ branch · diff file (section)    +N/-N · hunk M/T
// Commit diffs: ┊ sha7 · diff file               +N/-N · hunk M/T
func (m Model) diffTitleBar() string {
	d := m.diff

	var left string
	if d.SHA != "" {
		// Commit diff — lead with the SHA instead of branch
		left = railFaint.Render("┊") + " " +
			shaIris.Render(d.SHA) + " " +
			fgFaint.Render("·") + " " +
			fgSoft.Render("diff") + " " +
			d.Path
	} else {
		branch := ""
		if m.status != nil {
			branch = m.status.Branch
			if branch == "(detached)" || branch == "" {
				branch = "detached"
			}
		}
		var sectLabel string
		switch d.Section {
		case git.SectionUnstaged:
			sectLabel = "unstaged"
		case git.SectionStaged:
			sectLabel = "staged"
		case git.SectionUntracked:
			sectLabel = "untracked"
		}
		left = railFaint.Render("┊") + " " +
			branchIris.Render(branch) + " " +
			fgFaint.Render("·") + " " +
			fgSoft.Render("diff") + " " +
			d.Path +
			" " + fgFaint.Render("("+sectLabel+")")
	}

	// Hunk position: which hunk contains the cursor
	hunkNum := 1
	if len(m.diffFlat) > 0 && m.diffCursor < len(m.diffFlat) {
		hunkNum = m.diffFlat[m.diffCursor].hunkIdx + 1
	}
	totalHunks := len(d.Hunks)

	stats := formatStats(d.Added, d.Deleted)
	hunkInfo := fgFaint.Render(fmt.Sprintf("hunk %d/%d", hunkNum, totalHunks))
	var right string
	if stats != "" {
		right = stats + " " + fgFaint.Render("·") + " " + hunkInfo
	} else {
		right = hunkInfo
	}

	used := visibleLen(left) + visibleLen(right)
	if m.width > 0 && used < m.width {
		return left + strings.Repeat(" ", m.width-used) + right
	}
	return left + "  " + right
}

// diffVisibleRange returns the [start, end) slice of m.diffFlat that should
// be rendered, keeping the cursor centred in the available content area.
func (m Model) diffVisibleRange(available int) (int, int) {
	n := len(m.diffFlat)
	if n == 0 || available <= 0 {
		return 0, 0
	}
	start := m.diffCursor - available/2
	if start < 0 {
		start = 0
	}
	end := start + available
	if end > n {
		end = n
		start = end - available
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

// renderDiffViewLine renders the line at flat index i, with cursor highlight if active.
func (m Model) renderDiffViewLine(i int) string {
	vl := m.diffFlat[i]
	hunk := m.diff.Hunks[vl.hunkIdx]

	var content string
	if vl.lineIdx == -2 {
		// File separator (multi-file commit diffs)
		fp := hunk.FilePath
		line := railFaint.Render("┊") + " " + fgSoft.Render(fp)
		if i == m.diffCursor {
			return applyCursorBg(line, m.width)
		}
		return line
	}
	if vl.lineIdx == -1 {
		// Hunk header line: split "@@ ... @@" from trailing context
		raw := hunk.Header
		rest := raw[2:] // skip leading "@@"
		end := strings.Index(rest, "@@")
		if end >= 0 {
			atAt := raw[:end+4] // "@@ ... @@"
			ctx := raw[end+4:]  // " func name() {" or ""
			content = " " + shaIris.Render(atAt) + fgSoft.Render(ctx)
		} else {
			content = " " + shaIris.Render(raw)
		}
	} else {
		dl := hunk.Lines[vl.lineIdx]
		switch dl.Kind {
		case git.LineAdded:
			content = addStyle.Render(" +") + highlightDiffContent(dl.Content, m.diffSearch, addStyle)
		case git.LineRemoved:
			content = delStyle.Render(" -") + highlightDiffContent(dl.Content, m.diffSearch, delStyle)
		default: // LineContext
			content = "  " + highlightDiffContent(dl.Content, m.diffSearch, lipgloss.NewStyle())
		}
	}

	line := railFaint.Render("┊") + content
	if i == m.diffCursor {
		return applyCursorBg(line, m.width)
	}
	return line
}

// highlightDiffContent renders content with the base style, injecting
// searchHlStyle around every occurrence of pattern (case-insensitive).
// If pattern is empty or not found, the content is rendered with base style only.
func highlightDiffContent(content, pattern string, base lipgloss.Style) string {
	if pattern == "" {
		return base.Render(content)
	}
	lower := strings.ToLower(content)
	lowerPat := strings.ToLower(pattern)
	if !strings.Contains(lower, lowerPat) {
		return base.Render(content)
	}
	var b strings.Builder
	i := 0
	for i < len(content) {
		idx := strings.Index(lower[i:], lowerPat)
		if idx < 0 {
			b.WriteString(base.Render(content[i:]))
			break
		}
		if idx > 0 {
			b.WriteString(base.Render(content[i : i+idx]))
		}
		b.WriteString(searchHlStyle.Render(content[i+idx : i+idx+len(pattern)]))
		i += idx + len(pattern)
	}
	return b.String()
}
