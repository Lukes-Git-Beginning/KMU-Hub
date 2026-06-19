#!/bin/bash
# SessionStart-Hook: kompakte Git-Orientierung (letzte Commits, geaenderte
# Dateien, juengste Aktivitaet). Robust gegen Nicht-Git-Verzeichnisse und
# frische Repos ohne Commits. Nur Kontext, nie blockieren (exit 0).
#
# KANONISCH (Source of Truth: claude-command-center). Ersetzt den frueher
# inline in settings.json eingebetteten Befehl durch ein versioniertes Skript.

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  exit 0
fi

echo '## Git State'
git log --oneline -5 2>/dev/null || echo '(noch keine Commits)'
echo ''
echo '## Modified Files'
git status --short 2>/dev/null | head -15
echo ''
echo '## Recent Activity (3 days)'
git log --since='3 days ago' --oneline --no-merges 2>/dev/null | head -10

exit 0
