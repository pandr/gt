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
	FilePath string // which file this hunk belongs to (populated for commit diffs)
	Header   string // full "@@ ... @@" line
	OldStart int
	NewStart int
	Lines    []DiffLine
}

type ParsedDiff struct {
	Path    string  // display name: file path for file diffs, sha7 for whole-commit diffs
	SHA     string  // non-empty for commit diffs
	Section Section
	Hunks   []Hunk
	Added   int
	Deleted int
}

// TotalLines returns the total number of content lines across all hunks,
// used for the large-diff fallback threshold.
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

	hunks, added, deleted := parseDiffOutput(stdout.String())
	return &ParsedDiff{Path: path, Section: section, Hunks: hunks, Added: added, Deleted: deleted}, nil
}

// ParseCommitDiff runs git show for the given commit SHA and returns a ParsedDiff.
// Pass a non-empty filePath to restrict to a single file within the commit.
func ParseCommitDiff(repoRoot, sha, filePath string) (*ParsedDiff, error) {
	args := []string{"show", "--format=", sha}
	if filePath != "" {
		args = append(args, "--", filePath)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	sha7 := sha
	if len(sha7) > 7 {
		sha7 = sha7[:7]
	}
	displayPath := sha7
	if filePath != "" {
		displayPath = filePath
	}

	hunks, added, deleted := parseDiffOutput(stdout.String())
	return &ParsedDiff{
		Path:    displayPath,
		SHA:     sha7,
		Section: SectionCommit,
		Hunks:   hunks,
		Added:   added,
		Deleted: deleted,
	}, nil
}

// parseDiffOutput parses raw unified diff text into hunks and line totals.
// It tracks FilePath per hunk by watching +++ b/<path> lines, so it handles
// both single-file and multi-file (git show) output.
func parseDiffOutput(raw string) ([]Hunk, int, int) {
	var hunks []Hunk
	var cur *Hunk
	var currentFile string
	oldLine, newLine := 0, 0
	added, deleted := 0, 0

	for _, line := range strings.Split(raw, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ b/"):
			currentFile = line[6:]
		case strings.HasPrefix(line, "@@ "):
			if cur != nil {
				hunks = append(hunks, *cur)
			}
			oldLine, newLine = parseHunkStarts(line)
			cur = &Hunk{FilePath: currentFile, Header: line, OldStart: oldLine, NewStart: newLine}
		case cur == nil:
			// still in file header
		case strings.HasPrefix(line, "+"):
			cur.Lines = append(cur.Lines, DiffLine{Kind: LineAdded, Content: line[1:], NewNum: newLine})
			added++
			newLine++
		case strings.HasPrefix(line, "-"):
			cur.Lines = append(cur.Lines, DiffLine{Kind: LineRemoved, Content: line[1:], OldNum: oldLine})
			deleted++
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
		hunks = append(hunks, *cur)
	}
	return hunks, added, deleted
}

// parseHunkStarts extracts the old and new starting line numbers from a @@ header.
// Format: "@@ -oldStart[,oldCount] +newStart[,newCount] @@[ context]"
func parseHunkStarts(line string) (oldStart, newStart int) {
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
