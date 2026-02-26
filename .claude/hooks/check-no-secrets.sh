#!/bin/bash
# Hook: Prevent committing files that likely contain secrets
# Runs as PreToolUse on Bash tool when git add is detected
# Exit 2 = block, Exit 0 = allow

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | grep -oP '(git add .*)' || true)

if [ -z "$COMMAND" ]; then
  exit 0
fi

# Check for sensitive file patterns
BLOCKED_PATTERNS=(".env" "credentials.json" ".pem" ".key" "secrets.json" "service-account.json")

for PATTERN in "${BLOCKED_PATTERNS[@]}"; do
  if echo "$COMMAND" | grep -q "$PATTERN"; then
    echo "BLOCKED: Attempting to stage file matching sensitive pattern: $PATTERN" >&2
    echo "Secrets and credentials must never be committed." >&2
    echo "Use environment variables instead (see CLAUDE.md section 9)." >&2
    exit 2
  fi
done

exit 0
