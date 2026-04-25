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
	{"d", "diff at cursor"},
	{"s", "stage at cursor"},
	{"u", "unstage at cursor"},
	{"t", "toggle tag"},
	{";s / ;u / ;d", "act on all tagged"},
	{"T", "clear all tags"},
	{"c", "commit prompt"},
	{"R", "refresh"},
	{"?", "help"},
	{"q / Ctrl-c", "quit"},
}
