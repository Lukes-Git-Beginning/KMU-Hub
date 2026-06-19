#!/bin/bash
# Hook: erzwingt Conventional-Commit-Messages.
# Laeuft als PreToolUse auf dem Bash-Tool, wenn `git commit` erkannt wird.
# Exit 2 = blockieren, Exit 0 = erlauben.
#
# KANONISCH (Source of Truth: claude-command-center). Nicht in Ziel-Projekten
# direkt editieren — Aenderungen hier vornehmen und via sync.ps1 propagieren.

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | grep -o 'git commit.*' || true)

if [ -z "$COMMAND" ]; then
  exit 0
fi

# Commit-Message aus -m-Flag extrahieren (Double- oder Single-Quotes).
MSG=$(echo "$COMMAND" | grep -oP '(?<=-m ")[^"]*' || echo "$COMMAND" | grep -oP "(?<=-m ')[^']*" || true)

# Heredoc-Commits (cat <<'EOF') nicht greifbar -> durchlassen.
if [ -z "$MSG" ]; then
  exit 0
fi

if ! echo "$MSG" | grep -qP '^(feat|fix|docs|refactor|test|chore)(\(.+\))?(!)?:\s'; then
  echo "Commit-Message folgt nicht dem Conventional-Commits-Format." >&2
  echo "Erforderlich: feat:|fix:|docs:|refactor:|test:|chore: gefolgt von der Beschreibung" >&2
  echo "Beispiel: feat(scope): add user authentication endpoint" >&2
  exit 2
fi

exit 0
