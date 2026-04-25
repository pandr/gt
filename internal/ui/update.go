package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/petera/gt/internal/git"
)

// refreshMsg carries fresh status+log data after a background reload.
type refreshMsg struct {
	status *git.Status
	log    []git.LogEntry
	err    error
}

type dirContentsMsg struct {
	dirPath string
	files   []git.FileEntry
	err     error
}

type wtFilesMsg struct {
	files []git.FileEntry
	err   error
}

func (m Model) Init() tea.Cmd {
	return refresh(m.repoRoot)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case refreshMsg:
		if msg.err != nil {
			m.toast = msg.err.Error()
		} else {
			m.status = msg.status
			m.log = msg.log
			m.buildRows()
			m.pruneTags()
			m.clampCursor()
		}
		return m, nil

	case dirContentsMsg:
		if msg.err == nil {
			m.dirContents[msg.dirPath] = msg.files
			m.buildRows()
		}
		return m, nil

	case wtFilesMsg:
		if msg.err == nil {
			m.wtFiles = msg.files
			m.buildRows()
		}
		return m, nil

	case execDoneMsg:
		// Ignore non-zero exit from diff commands: git diff --no-index exits 1 when
		// differences are found, and the pager may exit non-zero on user interrupt.
		return m, nil

	case editorDoneMsg:
		if msg.err != nil {
			m.toast = msg.err.Error()
			return m, nil
		}
		err := git.CommitFile(m.repoRoot, msg.filePath)
		os.Remove(msg.filePath)
		m.mode = modeNormal
		m.commitInput.Blur()
		if err != nil {
			m.toast = err.Error()
			return m, nil
		}
		return m, refresh(m.repoRoot)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear toast on any keypress
	if m.toast != "" {
		m.toast = ""
	}

	// Help overlay
	if m.mode == modeHelp {
		m.mode = modeNormal
		return m, nil
	}

	// Commit input mode
	if m.mode == modeCommit {
		return m.handleCommitKey(msg)
	}

	// Tag prefix mode (waiting for second key after ;)
	if m.mode == modeTagPrefix {
		return m.handleTagPrefixKey(msg)
	}

	// Confirm mode (y/n for destructive actions)
	if m.mode == modeConfirm {
		return m.handleConfirmKey(msg)
	}

	// Normal mode
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "?":
		m.mode = modeHelp
		return m, nil

	case "R":
		return m, refresh(m.repoRoot)

	case "j", "down":
		m.cursor++
		m.clampCursor()

	case "k", "up":
		m.cursor--
		m.clampCursor()

	case "g":
		m.cursor = 0

	case "G":
		m.cursor = len(m.rows) - 1
		m.clampCursor()

	case "ctrl+d":
		m.cursor += m.height / 2
		m.clampCursor()

	case "ctrl+u":
		m.cursor -= m.height / 2
		m.clampCursor()

	case "right":
		return m.doExpand()

	case "left":
		return m.doCollapse()

	case "d":
		return m, m.doDiff(m.cursorRow(), nil)

	case "x":
		return m.doRmCached()

	case "X":
		return m.doRmFileConfirm()

	case "s":
		return m.doStage()

	case "u":
		return m.doUnstage()

	case "t":
		r := m.cursorRow()
		k := tagKey(r)
		if k != "" {
			m.tags[k] = !m.tags[k]
			if !m.tags[k] {
				delete(m.tags, k)
			}
		}

	case ";":
		if len(m.tags) > 0 {
			m.mode = modeTagPrefix
		}

	case "T":
		m.tags = make(map[string]bool)

	case "c":
		m.mode = modeCommit
		m.commitInput.SetValue("")
		m.commitInput.Focus()
		return m, textinput.Blink
	}

	return m, nil
}

func (m Model) handleCommitKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.commitInput.Blur()
		return m, nil

	case "enter":
		msg := m.commitInput.Value()
		if msg == "" {
			m.toast = "commit message is empty"
			m.mode = modeNormal
			m.commitInput.Blur()
			return m, nil
		}
		m.mode = modeNormal
		m.commitInput.Blur()
		err := git.Commit(m.repoRoot, msg)
		if err != nil {
			m.toast = err.Error()
			return m, nil
		}
		return m, refresh(m.repoRoot)

	case "ctrl+g":
		f, err := os.CreateTemp("", "gt-commit-*.txt")
		if err != nil {
			m.toast = err.Error()
			return m, nil
		}
		f.WriteString(m.commitInput.Value())
		f.Close()
		return m, execEditor(f.Name())
	}

	var cmd tea.Cmd
	m.commitInput, cmd = m.commitInput.Update(msg)
	return m, cmd
}

func (m Model) handleTagPrefixKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = modeNormal
	switch msg.String() {
	case "d":
		return m, m.doDiff(row{}, m.tags)
	case "s":
		return m.doTaggedStage()
	case "u":
		return m.doTaggedUnstage()
	case "esc":
		return m, nil
	}
	m.toast = fmt.Sprintf("unknown tag command: %s", msg.String())
	return m, nil
}

func (m Model) cursorRow() row {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return row{}
	}
	return m.rows[m.cursor]
}

// doDiff builds and executes the diff for the cursor row or tagged set.
func (m Model) doDiff(r row, tags map[string]bool) tea.Cmd {
	// Tagged diff: if we have tags, diff all of them
	if len(tags) > 0 {
		return m.taggedDiff(tags)
	}

	switch r.kind {
	case rowFile:
		cmd := git.DiffCmd(m.repoRoot, r.section, r.file.Path)
		return execDiff(cmd)
	case rowSectionHeader:
		cmd := git.DiffCmd(m.repoRoot, r.section, "")
		return execDiff(cmd)
	case rowCommit:
		cmd := git.ShowCmd(m.repoRoot, r.commit.SHA)
		return execDiff(cmd)
	}
	return nil
}

func (m Model) taggedDiff(tags map[string]bool) tea.Cmd {
	// Collect tagged file rows grouped by section
	var cmds []tea.Cmd
	for k := range tags {
		// Find the row for this key
		for _, r := range m.rows {
			if tagKey(r) == k && r.kind == rowFile {
				cmd := git.DiffCmd(m.repoRoot, r.section, r.file.Path)
				if cmd != nil {
					cmds = append(cmds, execDiff(cmd))
				}
			} else if tagKey(r) == k && r.kind == rowCommit {
				cmd := git.ShowCmd(m.repoRoot, r.commit.SHA)
				cmds = append(cmds, execDiff(cmd))
			}
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Sequence(cmds...)
}

func (m Model) doStage() (Model, tea.Cmd) {
	r := m.cursorRow()
	switch r.kind {
	case rowFile:
		if r.section == git.SectionUntracked || r.section == git.SectionUnstaged {
			if err := git.Stage(m.repoRoot, r.file.Path); err != nil {
				m.toast = err.Error()
				return m, nil
			}
			return m, refresh(m.repoRoot)
		}
	case rowSectionHeader:
		if r.section == git.SectionUntracked {
			paths := make([]string, len(m.status.Untracked))
			for i, f := range m.status.Untracked {
				paths[i] = f.Path
			}
			if len(paths) > 0 {
				if err := git.Stage(m.repoRoot, paths...); err != nil {
					m.toast = err.Error()
					return m, nil
				}
				return m, refresh(m.repoRoot)
			}
		} else if r.section == git.SectionUnstaged {
			paths := make([]string, len(m.status.Unstaged))
			for i, f := range m.status.Unstaged {
				paths[i] = f.Path
			}
			if len(paths) > 0 {
				if err := git.Stage(m.repoRoot, paths...); err != nil {
					m.toast = err.Error()
					return m, nil
				}
				return m, refresh(m.repoRoot)
			}
		}
	}
	return m, nil
}

func (m Model) doUnstage() (Model, tea.Cmd) {
	r := m.cursorRow()
	switch r.kind {
	case rowFile:
		if r.section == git.SectionStaged {
			if err := git.Unstage(m.repoRoot, r.file.Path); err != nil {
				m.toast = err.Error()
				return m, nil
			}
			return m, refresh(m.repoRoot)
		}
	case rowSectionHeader:
		if r.section == git.SectionStaged {
			paths := make([]string, len(m.status.Staged))
			for i, f := range m.status.Staged {
				paths[i] = f.Path
			}
			if len(paths) > 0 {
				if err := git.Unstage(m.repoRoot, paths...); err != nil {
					m.toast = err.Error()
					return m, nil
				}
				return m, refresh(m.repoRoot)
			}
		}
	}
	return m, nil
}

func (m Model) doTaggedStage() (Model, tea.Cmd) {
	var paths []string
	for k := range m.tags {
		for _, r := range m.rows {
			if tagKey(r) == k && r.kind == rowFile &&
				(r.section == git.SectionUntracked || r.section == git.SectionUnstaged) {
				paths = append(paths, r.file.Path)
			}
		}
	}
	if len(paths) == 0 {
		m.toast = "no stageable tagged files"
		return m, nil
	}
	if err := git.Stage(m.repoRoot, paths...); err != nil {
		m.toast = err.Error()
		return m, nil
	}
	return m, refresh(m.repoRoot)
}

func (m Model) doTaggedUnstage() (Model, tea.Cmd) {
	var paths []string
	for k := range m.tags {
		for _, r := range m.rows {
			if tagKey(r) == k && r.kind == rowFile && r.section == git.SectionStaged {
				paths = append(paths, r.file.Path)
			}
		}
	}
	if len(paths) == 0 {
		m.toast = "no unstageable tagged files"
		return m, nil
	}
	if err := git.Unstage(m.repoRoot, paths...); err != nil {
		m.toast = err.Error()
		return m, nil
	}
	return m, refresh(m.repoRoot)
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	m.mode = modeNormal
	m.confirmPrompt = ""
	switch msg.String() {
	case "y", "Y":
		switch m.confirmKind {
		case confirmRmFile:
			if err := git.RmFile(m.repoRoot, m.confirmPath); err != nil {
				m.toast = err.Error()
				return m, nil
			}
			return m, refresh(m.repoRoot)
		}
	}
	m.confirmKind = confirmNone
	m.confirmPath = ""
	return m, nil
}

func (m Model) doExpand() (Model, tea.Cmd) {
	r := m.cursorRow()
	switch r.kind {
	case rowDir:
		if m.openDirs[r.dirPath] {
			return m, nil // already open
		}
		m.openDirs[r.dirPath] = true
		if r.section == git.SectionUntracked {
			return m, expandUntrackedDir(m.repoRoot, r.dirPath)
		}
		// Working tree dir: files already in wtFiles, just rebuild
		m.buildRows()
		return m, nil
	case rowSectionHeader:
		if r.section == git.SectionWorkingTree && !m.wtOpen {
			m.wtOpen = true
			if len(m.wtFiles) == 0 {
				return m, fetchWTFiles(m.cwd)
			}
			m.buildRows()
		}
	}
	return m, nil
}

func (m Model) doCollapse() (Model, tea.Cmd) {
	r := m.cursorRow()
	switch r.kind {
	case rowDir:
		m.openDirs[r.dirPath] = false
		m.buildRows()
	case rowFile:
		// If this file is a child (depth>0), collapse its parent dir
		if r.depth > 0 {
			// find parent rowDir above cursor
			for i := m.cursor - 1; i >= 0; i-- {
				if m.rows[i].kind == rowDir {
					m.openDirs[m.rows[i].dirPath] = false
					m.cursor = i
					m.buildRows()
					break
				}
			}
		}
	case rowSectionHeader:
		if r.section == git.SectionWorkingTree {
			m.wtOpen = false
			m.buildRows()
		}
	}
	return m, nil
}

func (m Model) doRmCached() (Model, tea.Cmd) {
	r := m.cursorRow()
	if r.kind != rowFile || r.section != git.SectionWorkingTree {
		return m, nil
	}
	if err := git.RmCached(m.repoRoot, r.file.Path); err != nil {
		m.toast = err.Error()
		return m, nil
	}
	// remove from wtFiles so the row disappears without a full refresh
	newFiles := m.wtFiles[:0]
	for _, f := range m.wtFiles {
		if f.Path != r.file.Path {
			newFiles = append(newFiles, f)
		}
	}
	m.wtFiles = newFiles
	m.buildRows()
	return m, refresh(m.repoRoot)
}

func (m Model) doRmFileConfirm() (Model, tea.Cmd) {
	r := m.cursorRow()
	if r.kind != rowFile || r.section != git.SectionWorkingTree {
		return m, nil
	}
	m.confirmKind = confirmRmFile
	m.confirmPath = r.file.Path
	m.confirmPrompt = fmt.Sprintf("Delete %s from disk? [y/N]", r.file.Path)
	m.mode = modeConfirm
	return m, nil
}

func expandUntrackedDir(repoRoot, dirPath string) tea.Cmd {
	return func() tea.Msg {
		files, err := git.ListUntrackedInDir(repoRoot, dirPath)
		return dirContentsMsg{dirPath: dirPath, files: files, err: err}
	}
}

func fetchWTFiles(cwd string) tea.Cmd {
	return func() tea.Msg {
		files, err := git.ListTrackedUnder(cwd)
		return wtFilesMsg{files: files, err: err}
	}
}

func refresh(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		status, err := git.GetStatus(repoRoot)
		if err != nil {
			return refreshMsg{err: err}
		}
		log, err := git.GetLog(repoRoot)
		if err != nil {
			return refreshMsg{err: err}
		}
		return refreshMsg{status: status, log: log}
	}
}
