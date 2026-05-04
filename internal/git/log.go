package git

import (
	"os/exec"
	"strings"
)

type LogEntry struct {
	SHA   string
	Title string
}

func GetLog(repoRoot string) ([]LogEntry, error) {
	cmd := exec.Command("git", "log", "-n", "50", "--oneline", "--decorate=short")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		// Not an error if repo has no commits yet
		if strings.Contains(err.Error(), "exit status") {
			return nil, nil
		}
		return nil, err
	}
	return parseLog(string(out)), nil
}

func GetCommitFiles(repoRoot, sha string) ([]string, error) {
	cmd := exec.Command("git", "show", "--name-only", "--format=", sha)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func parseLog(out string) []LogEntry {
	var entries []LogEntry
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		entries = append(entries, LogEntry{
			SHA:   parts[0],
			Title: parts[1],
		})
	}
	return entries
}
