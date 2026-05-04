package git

import (
	"fmt"
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

func GetCommitFiles(repoRoot, sha string) ([]FileEntry, error) {
	cmd := exec.Command("git", "show", "--numstat", "--format=", sha)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var files []FileEntry
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		var added, deleted int
		fmt.Sscan(parts[0], &added)
		fmt.Sscan(parts[1], &deleted)
		files = append(files, FileEntry{Path: parts[2], Added: added, Deleted: deleted})
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
