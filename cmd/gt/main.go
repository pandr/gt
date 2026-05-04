package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/petera/gt/internal/ui"
)

func main() {
	repoRoot, err := repoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gt: not inside a git repository")
		os.Exit(1)
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = repoRoot
	}

	m := ui.NewModel(repoRoot, cwd, buildVersion())
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gt: %v\n", err)
		os.Exit(1)
	}
}

func buildVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	var rev, date string
	var modified bool
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) > 7 {
				rev = s.Value[:7]
			} else {
				rev = s.Value
			}
		case "vcs.time":
			if len(s.Value) >= 10 {
				date = s.Value[:10]
			}
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if rev == "" {
		return ""
	}
	s := rev
	if modified {
		s += "+"
	}
	if date != "" {
		s += " " + date
	}
	return s
}

func repoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
