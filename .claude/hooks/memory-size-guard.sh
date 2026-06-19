#!/bin/bash
# SessionStart-Hook: warnt, wenn die Projekt-MEMORY.md ihr Budget sprengt.
# Die Harness laedt nur die ersten ~200 Zeilen / 25 KB von MEMORY.md.
# Budget mit Puffer: 150 Zeilen / 18 KB. Nur Warnung, nie blockieren (exit 0).
#
# KANONISCH + projekt-agnostisch (Source of Truth: claude-command-center):
# leitet den Memory-Pfad dynamisch aus dem cwd ab. Claude Code flacht den
# absoluten Pfad ab, indem jedes Nicht-Alphanumerische -> '-' wird.

LINE_LIMIT=150
BYTE_LIMIT=18432

# Windows-Pfad des cwd (Git Bash: `pwd -W` -> C:/Users/...), sonst POSIX-Pfad.
WINPATH=$(pwd -W 2>/dev/null || pwd)
# Projekt-Key: jedes Zeichen ausser [A-Za-z0-9] -> '-'
PROJECT_KEY=$(printf '%s' "$WINPATH" | sed -E 's/[^A-Za-z0-9]/-/g')

MEMORY_FILE="$HOME/.claude/projects/${PROJECT_KEY}/memory/MEMORY.md"

[ -f "$MEMORY_FILE" ] || exit 0

LINE_COUNT=$(wc -l < "$MEMORY_FILE" | tr -d ' ')
BYTE_COUNT=$(wc -c < "$MEMORY_FILE" | tr -d ' ')

if [ "$LINE_COUNT" -gt "$LINE_LIMIT" ] || [ "$BYTE_COUNT" -gt "$BYTE_LIMIT" ]; then
  MSG="MEMORY.md BUDGET UEBERSCHRITTEN: ${LINE_COUNT}/${LINE_LIMIT} Zeilen, ${BYTE_COUNT}/${BYTE_LIMIT} Bytes. Harness-Hardlimit: 200 Zeilen / 25 KB. AKTION VOR neuen Memory-Writes: aelteste Eintraege nach memory/archive/ verschieben und ihre Index-Zeilen entfernen."
  printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}' "$MSG"
fi

exit 0
