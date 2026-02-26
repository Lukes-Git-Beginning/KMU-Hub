#!/bin/bash
# Hook: Enforce conventional commit messages
# Runs as PreToolUse on Bash tool when git commit is detected
# Exit 2 = block, Exit 0 = allow

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | grep -o 'git commit.*' || true)

if [ -z "$COMMAND" ]; then
  exit 0
fi

# Extract commit message from -m flag
MSG=$(echo "$COMMAND" | grep -oP '(?<=-m ")[^"]*' || echo "$COMMAND" | grep -oP "(?<=-m ')[^']*" || true)

# Also handle heredoc-style commits (cat <<'EOF')
if [ -z "$MSG" ]; then
  exit 0
fi

# Check conventional commit format
if ! echo "$MSG" | grep -qP '^(feat|fix|docs|refactor|test|chore)(\(.+\))?(!)?:\s'; then
  echo "Commit message does not follow conventional commits format." >&2
  echo "Required: feat:|fix:|docs:|refactor:|test:|chore: followed by description" >&2
  echo "Example: feat(auth): add JWT token refresh endpoint" >&2
  exit 2
fi

exit 0
