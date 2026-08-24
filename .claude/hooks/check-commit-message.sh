#!/bin/bash
# Hook: erzwingt Conventional-Commit-Messages.
# Laeuft als PreToolUse auf dem Bash-Tool, wenn `git commit` erkannt wird.
# Exit 2 = blockieren, Exit 0 = erlauben.
#
# KANONISCH (Source of Truth: claude-command-center). Nicht in Ziel-Projekten
# direkt editieren — Aenderungen hier vornehmen und via sync.ps1 propagieren.

INPUT=$(cat)

# JSON-Escaping der Doppelquotes VOR der Extraktion aufloesen.
# Ohne diesen Schritt lief der Hook ins Leere: der PreToolUse-stdin ist JSON, dort steht
# `git commit -m \"feat: x\"` — der Lookbehind unten sucht aber `-m "` und fand nie etwas.
# MSG blieb leer, der Hook liess jede doppelt gequotete Message durch. Am 2026-08-24 durch
# scharfes Ausloesen aufgedeckt: `git commit -m "kaputte message ohne praefix"` ging durch.
# Single-Quotes waren nie betroffen (JSON escaped die nicht) — deshalb fiel es nicht auf.
UNESCAPED=$(printf '%s' "$INPUT" | sed 's/\\"/"/g')
COMMAND=$(echo "$UNESCAPED" | grep -o 'git commit.*' || true)

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
