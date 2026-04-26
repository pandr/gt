package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/petera/gt/internal/git"
)

type confirmKind int

const (
	confirmNone confirmKind = iota
	confirmRmFile // git rm (delete from disk)
)

type mode int

const (
	modeNormal mode = iota
	modeCommit
	modeTagPrefix // waiting for ;<cmd>
	modeHelp
	modeConfirm // waiting for y/n on a destructive action
)

// rowKind identifies what a cursor row represents.
type rowKind int

const (
	rowSectionHeader rowKind = iota
	rowFile
	rowCommit
	rowDir       // expandable directory node
	rowSeparator // blank line between groups, not navigable
)

// row is a single renderable + navigable item in the list.
type row struct {
	kind    rowKind
	section git.Section
	file    *git.FileEntry // nil for header/commit/dir rows
	commit  *git.LogEntry  // nil for header/file/dir rows
	dirPath string         // rowDir only
	depth   int            // indentation level (0 = top level)
}

// Model is the bubbletea model.
type Model struct {
	repoRoot string
	cwd      string
	status   *git.Status
	log      []git.LogEntry

	rows   []row
	cursor int

	tags map[string]bool // keyed by path (or sha for commits)

	// expandable dir state
	openDirs    map[string]bool           // dir path → open
	dirContents map[string][]git.FileEntry // dir path → listed files (populated on expand)

	// working tree section
	wtOpen  bool
	wtFiles []git.FileEntry // all tracked files under cwd

	mode        mode
	commitInput textinput.Model

	// confirm mode
	confirmPrompt string
	confirmKind   confirmKind
	confirmPath   string

	toast string // transient error message

	width  int
	height int
}

func NewModel(repoRoot, cwd string) Model {
	ti := textinput.New()
	ti.Placeholder = "Commit message (Enter=commit  Ctrl-g=editor  Esc=cancel)"
	ti.CharLimit = 500

	return Model{
		repoRoot:    repoRoot,
		cwd:         cwd,
		tags:        make(map[string]bool),
		openDirs:    make(map[string]bool),
		dirContents: make(map[string][]git.FileEntry),
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
			fe := &m.status.Untracked[i]
			if fe.IsDir {
				m.rows = append(m.rows, row{kind: rowDir, section: git.SectionUntracked, dirPath: fe.Path})
				if m.openDirs[fe.Path] {
					for j := range m.dirContents[fe.Path] {
						child := &m.dirContents[fe.Path][j]
						m.rows = append(m.rows, row{kind: rowFile, section: git.SectionUntracked, file: child, depth: 1})
					}
				}
			} else {
				m.rows = append(m.rows, row{kind: rowFile, section: git.SectionUntracked, file: fe})
			}
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

	// Working tree section (between git status and recent commits)
	m.rows = append(m.rows, row{kind: rowSeparator})
	m.rows = append(m.rows, row{kind: rowSectionHeader, section: git.SectionWorkingTree})
	if m.wtOpen && len(m.wtFiles) > 0 {
		for _, group := range groupByTopDir(m.wtFiles) {
			if group.isDir {
				m.rows = append(m.rows, row{kind: rowDir, section: git.SectionWorkingTree, dirPath: group.name})
				if m.openDirs[group.name] {
					for j := range group.files {
						m.rows = append(m.rows, row{kind: rowFile, section: git.SectionWorkingTree, file: &group.files[j], depth: 1})
					}
				}
			} else {
				m.rows = append(m.rows, row{kind: rowFile, section: git.SectionWorkingTree, file: &group.files[0]})
			}
		}
	}

	// Recent commits section
	m.rows = append(m.rows, row{kind: rowSeparator})
	m.rows = append(m.rows, row{kind: rowSectionHeader, section: git.SectionLog})
	for i := range m.log {
		m.rows = append(m.rows, row{kind: rowCommit, section: git.SectionLog, commit: &m.log[i]})
	}
}

// wtGroup is a top-level entry in the working tree: either a plain file or a dir.
type wtGroup struct {
	name  string
	isDir bool
	files []git.FileEntry
}

// groupByTopDir groups flat file list by first path component.
func groupByTopDir(files []git.FileEntry) []wtGroup {
	var order []string
	byDir := make(map[string]*wtGroup)
	for _, f := range files {
		idx := strings.Index(f.Path, "/")
		var key string
		if idx < 0 {
			key = f.Path // top-level file
		} else {
			key = f.Path[:idx+1] // dir with trailing slash: "cmd/"
		}
		if _, ok := byDir[key]; !ok {
			order = append(order, key)
			isDir := idx >= 0
			byDir[key] = &wtGroup{name: key, isDir: isDir}
		}
		byDir[key].files = append(byDir[key].files, f)
	}
	result := make([]wtGroup, 0, len(order))
	for _, k := range order {
		result = append(result, *byDir[k])
	}
	return result
}

// skipSeparators nudges the cursor past any rowSeparator rows in direction d (+1 or -1).
func (m *Model) skipSeparators(d int) {
	for m.cursor >= 0 && m.cursor < len(m.rows) && m.rows[m.cursor].kind == rowSeparator {
		m.cursor += d
	}
	m.clampCursor()
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
