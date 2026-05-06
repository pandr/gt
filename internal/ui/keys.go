package ui

type keyBinding struct {
	Key  string
	Desc string
}

var keyBindings = []keyBinding{
	{"j / ↓", "down"},
	{"k / ↑", "up"},
	{"g", "top"},
	{"G", "bottom"},
	{"Ctrl-d", "half page down"},
	{"Ctrl-u", "half page up"},
	{"l / →", "expand directory / section"},
	{"h / ←", "collapse directory / section"},
	{"enter", "open — expand dir/commit, diff file or section"},
	{"d / space", "diff at cursor"},
	{"v", "view file in pager"},
	{"e", "open file in editor"},
	{"s", "stage at cursor"},
	{"u", "unstage at cursor"},
	{"r", "restore (discard unstaged changes)"},
	{"x", "untrack file (keep on disk)"},
	{"X", "untrack + delete file (asks y/n)"},
	{"t", "toggle tag"},
	{";s / ;u / ;d", "act on all tagged"},
	{"T", "clear all tags"},
	{"!", "run shell command (refreshes after)"},
	{"c", "commit prompt"},
	{"A", "amend last commit (unpushed only)"},
	{"R", "refresh"},
	{"?", "help"},
	{"q / Ctrl-c", "quit"},
}
