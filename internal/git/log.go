package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type LogEntry struct {
	SHA   string
	Refs  []string
	Title string
	Time  time.Time
}

func GetLog(repoRoot string) ([]LogEntry, error) {
	cmd := exec.Command("git", "log", "-n", "50", "--format=%h%x09%ct%x09%D%x09%s")
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

func GetFileLog(repoRoot, path string) ([]LogEntry, error) {
	cmd := exec.Command("git", "log", "--follow", "-n", "100", "--format=%h%x09%ct%x09%D%x09%s", "--", path)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
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

func GetCommitBody(repoRoot, sha string) ([]string, error) {
	cmd := exec.Command("git", "show", "-s", "--format=%B", sha)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	// skip title line and any immediately following blank line
	if len(lines) > 0 {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	// strip trailing blank lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, nil
}

func parseLog(out string) []LogEntry {
	var entries []LogEntry
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		var t time.Time
		if secs, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
			t = time.Unix(secs, 0)
		}
		var refs []string
		if parts[2] != "" {
			for _, r := range strings.Split(parts[2], ", ") {
				if r = strings.TrimSpace(r); r != "" {
					refs = append(refs, r)
				}
			}
		}
		entries = append(entries, LogEntry{
			SHA:   parts[0],
			Refs:  refs,
			Title: parts[3],
			Time:  t,
		})
	}
	return entries
}
