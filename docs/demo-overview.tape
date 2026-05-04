Output docs/demo-overview.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "SeaShells"
Set Padding 24
Set TypingSpeed 70ms

Hide
Type "cd /Users/petera/git/gt && go build -o /tmp/gt-demo-bin ./cmd/gt/ 2>/dev/null && bash docs/demo-setup.sh && mkdir -p /tmp/gt-demo-path && cp /tmp/gt-demo-bin /tmp/gt-demo-path/gt && cd /tmp/gt-demo && export PATH=/tmp/gt-demo-path:$PATH && export PS1='$ ' && export GT_DEMO_KEYS=1"
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

# Navigate into the Untracked dir to show expansion
Type "j"
Sleep 400ms
Type "l"
Sleep 1100ms
Type "h"
Sleep 800ms

# Cruise down through all sections
Type "j"
Sleep 350ms
Type "j"
Sleep 350ms
Type "j"
Sleep 350ms
Type "j"
Sleep 350ms
Type "j"
Sleep 350ms
Type "j"
Sleep 350ms
Type "j"
Sleep 600ms

# Navigate back up
Type "k"
Sleep 280ms
Type "k"
Sleep 280ms
Type "k"
Sleep 280ms
Type "k"
Sleep 280ms
Type "k"
Sleep 280ms
Type "k"
Sleep 280ms

Sleep 1s

Type "q"
Sleep 500ms
