package git

import (
	"bytes"
	"os/exec"
	"strings"
)

type FileEntry struct {
	Path       string
	OrigPath   string // non-empty on renames
	XY         string // two-char porcelain status
	InStaged   bool
	InUnstaged bool
	InUntracked bool
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
			rest := strings.TrimPrefix(line, "# branch.ab ")
			var a, b int
			for _, part := range strings.Fields(rest) {
				if strings.HasPrefix(part, "+") {
					for _, r := range part[1:] {
						a = a*10 + int(r-'0')
					}
					s.Ahead = a
				} else if strings.HasPrefix(part, "-") {
					for _, r := range part[1:] {
						b = b*10 + int(r-'0')
					}
					s.Behind = b
				}
			}
		case strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 "):
			// "1 XY sub mH mI mW hH hI path" (ordinary changed)
			// "2 XY sub mH mI mW hH hI X/Y path\0origPath" (rename/copy)
			fields := strings.SplitN(line[2:], " ", 8)
			if len(fields) < 8 {
				continue
			}
			xy := fields[0]
			x := xy[0]
			y := xy[1]
			pathField := fields[7]
			var path, origPath string
			if line[0] == '2' {
				// next NUL-separated token is the origPath
				path = pathField
				if i+1 < len(entries) {
					origPath = string(entries[i+1])
					i++
				}
			} else {
				path = pathField
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
			})
		}
	}
	return s
}
