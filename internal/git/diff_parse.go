package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type LineKind int

const (
	LineContext LineKind = iota
	LineAdded
	LineRemoved
)

type DiffLine struct {
	Kind    LineKind
	Content string // raw content without the leading +/-/space indicator
	OldNum  int    // line number in original file (0 if n/a)
	NewNum  int    // line number in new file (0 if n/a)
}

type Hunk struct {
	Header   string // full "@@ ... @@" line
	OldStart int
	NewStart int
	Lines    []DiffLine
}

type ParsedDiff struct {
	Path    string
	Section Section
	Hunks   []Hunk
	Added   int
	Deleted int
}

// TotalLines returns the total number of content lines across all hunks
// (not counting hunk headers), used for the large-diff fallback threshold.
func (d *ParsedDiff) TotalLines() int {
	n := 0
	for i := range d.Hunks {
		n += len(d.Hunks[i].Lines)
	}
	return n
}

// ParseDiff runs git diff for the given file and section and returns a
// structured ParsedDiff. The command is run without --color so the output
// can be parsed; colours are applied by the caller.
func ParseDiff(repoRoot string, section Section, path string) (*ParsedDiff, error) {
	var args []string
	switch section {
	case SectionUntracked:
		args = []string{"diff", "--no-index", "/dev/null", path}
	case SectionUnstaged:
		args = []string{"diff", "--", path}
	case SectionStaged:
		args = []string{"diff", "--staged", "--", path}
	default:
		return nil, fmt.Errorf("unsupported section for inline diff: %v", section)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		// git diff --no-index exits 1 when files differ; that is not an error.
		if section == SectionUntracked {
			if _, ok := err.(*exec.ExitError); ok {
				err = nil
			}
		}
		if err != nil {
			return nil, err
		}
	}

	return parseDiff(stdout.String(), section, path), nil
}

func parseDiff(raw string, section Section, path string) *ParsedDiff {
	d := &ParsedDiff{Path: path, Section: section}

	var cur *Hunk
	oldLine, newLine := 0, 0

	for _, line := range strings.Split(raw, "\n") {
		switch {
		case strings.HasPrefix(line, "@@ "):
			if cur != nil {
				d.Hunks = append(d.Hunks, *cur)
			}
			oldLine, newLine = parseHunkStarts(line)
			cur = &Hunk{Header: line, OldStart: oldLine, NewStart: newLine}

		case cur == nil:
			// still in the file header, skip

		case strings.HasPrefix(line, "+"):
			cur.Lines = append(cur.Lines, DiffLine{Kind: LineAdded, Content: line[1:], NewNum: newLine})
			d.Added++
			newLine++

		case strings.HasPrefix(line, "-"):
			cur.Lines = append(cur.Lines, DiffLine{Kind: LineRemoved, Content: line[1:], OldNum: oldLine})
			d.Deleted++
			oldLine++

		case strings.HasPrefix(line, " "):
			cur.Lines = append(cur.Lines, DiffLine{Kind: LineContext, Content: line[1:], OldNum: oldLine, NewNum: newLine})
			oldLine++
			newLine++

		case strings.HasPrefix(line, `\ `):
			// "\ No newline at end of file" — skip

		}
	}
	if cur != nil {
		d.Hunks = append(d.Hunks, *cur)
	}
	return d
}

// parseHunkStarts extracts the old and new starting line numbers from a @@ header.
// Format: "@@ -oldStart[,oldCount] +newStart[,newCount] @@[ context]"
func parseHunkStarts(line string) (oldStart, newStart int) {
	// Fields: ["@@", "-a,b", "+c,d", "@@", ...]
	for _, f := range strings.Fields(line) {
		if strings.HasPrefix(f, "-") {
			oldStart = parseFirstInt(f[1:])
		} else if strings.HasPrefix(f, "+") {
			newStart = parseFirstInt(f[1:])
		}
	}
	return
}

func parseFirstInt(s string) int {
	if i := strings.IndexByte(s, ','); i >= 0 {
		s = s[:i]
	}
	n, _ := strconv.Atoi(s)
	return n
}
