#!/usr/bin/env bash
# Creates a throwaway demo repo in /tmp/gt-demo. Safe to re-run.
set -e

REPO=/tmp/gt-demo
rm -rf "$REPO"
mkdir -p "$REPO"
cd "$REPO"

git init -q
git config user.name "Demo"
git config user.email "demo@example.com"

mkdir -p src

cat > src/main.go << 'EOF'
package main

import "fmt"

func main() {
	fmt.Println("hello, world")
}
EOF

cat > src/util.go << 'EOF'
package main

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
EOF

cat > README.md << 'EOF'
# myproject

A sample Go project.
EOF

git add .
git commit -q -m "initial commit"

cat >> README.md << 'EOF'

## Install

    go build ./...
EOF
git add README.md
git commit -q -m "add install instructions"

echo "1.0.0" > VERSION
git add VERSION
git commit -q -m "tag v1.0.0"

# ---- interesting working tree state ----

# Unstaged modifications
cat >> src/main.go << 'EOF'

func greet(name string) string {
	return "Hello, " + name + "!"
}
EOF

cat >> README.md << 'EOF'

## Usage

Run `./myproject` to start.
EOF

# Staged change
cat >> src/util.go << 'EOF'

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
EOF
git add src/util.go

# Untracked files
cat > src/handler.go << 'EOF'
package main

import "fmt"

func handle(msg string) {
	fmt.Println("handling:", msg)
}
EOF

mkdir -p docs
cat > docs/api.md << 'EOF'
# API Reference

## greet(name string) string

Returns a greeting for the given name.
EOF

cat > docs/guide.md << 'EOF'
# Getting Started

Clone the repo and run `go build ./...`.
EOF
