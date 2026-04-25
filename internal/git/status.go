package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type FileEntry struct {
	Path        string
	OrigPath    string // non-empty on renames
	XY          string // two-char porcelain status
	InStaged    bool
	InUnstaged  bool
	InUntracked bool
	IsDir       bool // untracked directory (path ends with /)
}

type Status struct {
	Branch   string
	Upstream string
	Ahead    int
	Behind   int
	Staged   []FileEntry
	Unstaged []FileEntry
	Untracked []FileEntry
}

func GetStatus(repoRoot string) (*Status, error) {
	cmd := exec.Command("git", "status", "--porcelain=v2", "--branch", "-z")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseStatus(out), nil
}

func parseStatus(data []byte) *Status {
	s := &Status{}
	entries := bytes.Split(data, []byte{0})

	for i := 0; i < len(entries); i++ {
		line := string(entries[i])
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			s.Branch = strings.TrimPrefix(line, "# branch.head ")
		case strings.HasPrefix(line, "# branch.upstream "):
			s.Upstream = strings.TrimPrefix(line, "# branch.upstream ")
		case strings.HasPrefix(line, "# branch.ab "):
			var a, b int
			fmt.Sscanf(strings.TrimPrefix(line, "# branch.ab "), "+%d -%d", &a, &b)
			s.Ahead = a
			s.Behind = b
		case strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 "):
			// type 1: "1 XY sub mH mI mW hH hI path"            (9 space-fields)
			// type 2: "2 XY sub mH mI mW hH hI score path\0orig" (score field extra)
			fields := strings.Fields(line)
			if len(fields) < 9 {
				continue
			}
			xy := fields[1]
			if len(xy) < 2 {
				continue
			}
			x := xy[0]
			y := xy[1]
			var path, origPath string
			if line[0] == '2' {
				if len(fields) < 10 {
					continue
				}
				path = fields[9]
				if i+1 < len(entries) {
					origPath = string(entries[i+1])
					i++
				}
			} else {
				path = fields[8]
			}
			fe := FileEntry{
				Path:     path,
				OrigPath: origPath,
				XY:       xy,
			}
			if x != '.' && x != '?' {
				fe.InStaged = true
				s.Staged = append(s.Staged, fe)
			}
			if y != '.' && y != '?' {
				fe.InUnstaged = true
				s.Unstaged = append(s.Unstaged, fe)
			}
		case strings.HasPrefix(line, "? "):
			path := strings.TrimPrefix(line, "? ")
			s.Untracked = append(s.Untracked, FileEntry{
				Path:        path,
				XY:          "??",
				InUntracked: true,
				IsDir:       strings.HasSuffix(path, "/"),
			})
		}
	}
	return s
}

// ListUntrackedInDir returns untracked files inside an untracked directory.
func ListUntrackedInDir(repoRoot, dirPath string) ([]FileEntry, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard", "--", dirPath)
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
		files = append(files, FileEntry{Path: line, XY: "??", InUntracked: true})
	}
	return files, nil
}

// ListTrackedUnder lists all tracked files under the given directory (usually cwd).
func ListTrackedUnder(dir string) ([]FileEntry, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var files []FileEntry
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		files = append(files, FileEntry{Path: line, XY: ".."})
	}
	return files, nil
}
