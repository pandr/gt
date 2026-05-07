Output docs/demo-diff.gif

Set Shell "bash"
Set FontSize 20
Set Width 1200
Set Height 700
Set Theme "SeaShells"
Set Padding 24
Set TypingSpeed 70ms

Hide
Type "cd /Users/petera/git/gt && go build -o /tmp/gt-demo-bin ./cmd/gt/ 2>/dev/null && bash docs/demo-setup.sh && mkdir -p /tmp/gt-demo-path && cp /tmp/gt-demo-bin /tmp/gt-demo-path/gt && cd /tmp/gt-demo && export PATH=/tmp/gt-demo-path:$PATH && export PS1='$ '"
Enter
Sleep 12s
Type "clear"
Enter
Sleep 500ms
Show

# Launch gt
Type "gt"
Sleep 500ms
Enter
Sleep 2s

# Navigate to README.md in Unstaged (j j j j from Untracked header)
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms

# Diff the file — pager opens
Sleep 500ms
Type "d"
Sleep 2800ms
Type "q"
Sleep 900ms

# Navigate to "add install instructions" commit (6 more j's from README.md)
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 500ms

# Expand the commit to show its files
Type "l"
Sleep 1000ms

# Navigate into the file entry
Type "j"
Sleep 400ms

# Diff that file within the commit
Type "d"
Sleep 2800ms
Type "q"
Sleep 900ms

Type "q"
Sleep 300ms
