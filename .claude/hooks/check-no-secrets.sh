#!/bin/bash
# Hook: Prevent committing files that likely contain secrets
# Runs as PreToolUse on Bash tool when git add is detected
# Exit 2 = block, Exit 0 = allow

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | grep -oP '(git add .*)' || true)

if [ -z "$COMMAND" ]; then
  exit 0
fi

BLOCKED_PATTERNS=(".env" "credentials.json" ".pem" ".key" "secrets.json" "service-account.json")
ALLOWED_SUFFIXES=(".example" ".template" ".sample")

# Strip leading "git add" and iterate actual path arguments.
PATHS=$(echo "$COMMAND" | sed -E 's/^git add //')

for PATH_ARG in $PATHS; do
  # Skip flags and empty entries.
  case "$PATH_ARG" in
    -*|"") continue ;;
  esac

  # Skip paths that are clearly example/template/sample files.
  IS_ALLOWED=false
  for SUFFIX in "${ALLOWED_SUFFIXES[@]}"; do
    case "$PATH_ARG" in
      *"$SUFFIX") IS_ALLOWED=true; break ;;
    esac
  done
  [ "$IS_ALLOWED" = true ] && continue

  for PATTERN in "${BLOCKED_PATTERNS[@]}"; do
    case "$PATH_ARG" in
      *"$PATTERN"*)
        echo "BLOCKED: Attempting to stage file matching sensitive pattern: $PATTERN" >&2
        echo "Path: $PATH_ARG" >&2
        echo "Secrets and credentials must never be committed." >&2
        echo "Use environment variables instead (see CLAUDE.md section 9)." >&2
        echo "Hint: filenames ending in .example, .template, or .sample are exempt." >&2
        exit 2
        ;;
    esac
  done
done

exit 0
