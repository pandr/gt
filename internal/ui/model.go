package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/petera/gt/internal/git"
)

type mode int

const (
	modeNormal mode = iota
	modeCommit
	modeTagPrefix // waiting for ;<cmd>
	modeHelp
)

// rowKind identifies what a cursor row represents.
type rowKind int

const (
	rowSectionHeader rowKind = iota
	rowFile
	rowCommit
)

// row is a single renderable + navigable item in the list.
type row struct {
	kind    rowKind
	section git.Section
	file    *git.FileEntry // nil for header/commit rows
	commit  *git.LogEntry  // nil for header/file rows
}

// Model is the bubbletea model.
type Model struct {
	repoRoot string
	status   *git.Status
	log      []git.LogEntry

	rows   []row
	cursor int

	tags map[string]bool // keyed by path (or sha for commits)

	mode        mode
	commitInput textinput.Model

	toast string // transient error message

	width  int
	height int
}

func NewModel(repoRoot string) Model {
	ti := textinput.New()
	ti.Placeholder = "Commit message (Enter=commit  Ctrl-g=editor  Esc=cancel)"
	ti.CharLimit = 500

	return Model{
		repoRoot:    repoRoot,
		tags:        make(map[string]bool),
		commitInput: ti,
	}
}

// buildRows rebuilds the flat list of navigable rows from current status/log.
func (m *Model) buildRows() {
	m.rows = m.rows[:0]

	if m.status != nil {
		// Untracked section
		m.rows = append(m.rows, row{kind: rowSectionHeader, section: git.SectionUntracked})
		for i := range m.status.Untracked {
			m.rows = append(m.rows, row{kind: rowFile, section: git.SectionUntracked, file: &m.status.Untracked[i]})
		}

		// Unstaged section
		m.rows = append(m.rows, row{kind: rowSectionHeader, section: git.SectionUnstaged})
		for i := range m.status.Unstaged {
			m.rows = append(m.rows, row{kind: rowFile, section: git.SectionUnstaged, file: &m.status.Unstaged[i]})
		}

		// Staged section
		m.rows = append(m.rows, row{kind: rowSectionHeader, section: git.SectionStaged})
		for i := range m.status.Staged {
			m.rows = append(m.rows, row{kind: rowFile, section: git.SectionStaged, file: &m.status.Staged[i]})
		}
	}

	// Recent commits section
	m.rows = append(m.rows, row{kind: rowSectionHeader, section: git.SectionLog})
	for i := range m.log {
		m.rows = append(m.rows, row{kind: rowCommit, section: git.SectionLog, commit: &m.log[i]})
	}
}

// clampCursor ensures cursor is within [0, len(rows)-1].
func (m *Model) clampCursor() {
	if len(m.rows) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
}

// pruneTags removes tags whose paths no longer appear in status.
func (m *Model) pruneTags() {
	if m.status == nil {
		m.tags = make(map[string]bool)
		return
	}
	valid := make(map[string]bool)
	for _, f := range m.status.Untracked {
		valid[f.Path] = true
	}
	for _, f := range m.status.Unstaged {
		valid[f.Path] = true
	}
	for _, f := range m.status.Staged {
		valid[f.Path] = true
	}
	for _, e := range m.log {
		valid[e.SHA] = true
	}
	for k := range m.tags {
		if !valid[k] {
			delete(m.tags, k)
		}
	}
}

// tagKey returns the string key for a row in the tags map.
func tagKey(r row) string {
	if r.file != nil {
		return r.file.Path
	}
	if r.commit != nil {
		return r.commit.SHA
	}
	return ""
}
