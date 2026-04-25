package ui

import (
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func getPager() string {
	for _, env := range []string{"GIT_PAGER", "PAGER"} {
		if p := os.Getenv(env); p != "" {
			return p
		}
	}
	return "less -R"
}

// execDiff pipes a git command through the user's pager so short diffs don't flash.
func execDiff(gitCmd *exec.Cmd) tea.Cmd {
	if gitCmd == nil {
		return nil
	}
	pagerParts := strings.Fields(getPager())
	pagerCmd := exec.Command(pagerParts[0], pagerParts[1:]...)

	pr, pw, err := os.Pipe()
	if err != nil {
		return tea.ExecProcess(gitCmd, func(err error) tea.Msg {
			return execDoneMsg{err: err}
		})
	}
	gitCmd.Stdout = pw
	gitCmd.Stderr = pw
	pagerCmd.Stdin = pr

	if err := gitCmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return func() tea.Msg { return execDoneMsg{err: err} }
	}
	go func() {
		gitCmd.Wait()
		pw.Close()
	}()

	return tea.ExecProcess(pagerCmd, func(err error) tea.Msg {
		pr.Close()
		return execDoneMsg{err: err}
	})
}

// execEditor opens $EDITOR on a temp file and returns its path via msg.
func execEditor(filePath string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, filePath)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorDoneMsg{filePath: filePath, err: err}
	})
}

type execDoneMsg struct{ err error }
type editorDoneMsg struct {
	filePath string
	err      error
}
