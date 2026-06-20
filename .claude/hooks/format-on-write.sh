#!/bin/bash
# PostToolUse-Hook: formatiert die gerade geschriebene/editierte Datei deterministisch.
# Dispatcher nach Dateiendung:
#   .ts/.tsx -> naechstes lokal installiertes prettier --write  (reines Formatieren). Kein prettier im
#               Baum -> still no-op. eslint --fix ist bewusst KEIN Fallback, um Autofix-Nebenwirkungen
#               (z.B. unused-imports-Entfernung) mitten im Edit zu vermeiden -> eslint laeuft via
#               lint-Script/Editor/CI, nicht hier.
#   .py      -> ruff format                                       (format-only, bewusst KEIN `ruff check --fix`)
#   sonst    -> still no-op
#
# Immer exit 0 (PostToolUse blockiert nie). Formatter-Output wird verschluckt, damit der Hook
# ausschliesslich sauberes hookSpecificOutput-JSON auf stdout schreibt. Tool nicht gefunden = no-op.
#
# KANONISCH + projekt-agnostisch (Source of Truth: claude-command-center). Kein jq/python-Dependency:
# file_path wird defensiv per grep aus dem PostToolUse-stdin-JSON geparst.

INPUT=$(cat)

# Ersten file_path aus dem JSON ziehen (Windows-Pfade kommen JSON-escaped: C:\\Users\\...).
RAW=$(printf '%s' "$INPUT" | grep -oP '"file_path"\s*:\s*"\K[^"]+' | head -n 1)
[ -z "$RAW" ] && exit 0

# JSON-escapte Doppel-Backslashes -> normale Slashes (Git-Bash- + Tool-tauglich).
FILE=$(printf '%s' "$RAW" | sed 's#\\\\#/#g')

# Nur formatieren, wenn die Datei wirklich existiert (Guard gegen geloeschte/fehlgeschlagene Writes).
[ -f "$FILE" ] || exit 0

EXT=$(printf '%s' "${FILE##*.}" | tr '[:upper:]' '[:lower:]')
BASE=$(basename "$FILE")
RAN=""

case "$EXT" in
  ts|tsx)
    # Vom Datei-Verzeichnis nach oben den naechsten Package-Root mit installiertem prettier suchen.
    # ENTRY = node-Eintrittspunkt (prettier v3: bin/prettier.cjs, v2: bin-prettier.js); leer => .bin-Shim.
    D=$(dirname "$FILE")
    PKG=""
    ENTRY=""
    I=0
    while [ -n "$D" ] && [ "$I" -lt 40 ]; do
      if [ -f "$D/node_modules/prettier/bin/prettier.cjs" ]; then
        PKG="$D"; ENTRY="node_modules/prettier/bin/prettier.cjs"; break
      elif [ -f "$D/node_modules/prettier/bin-prettier.js" ]; then
        PKG="$D"; ENTRY="node_modules/prettier/bin-prettier.js"; break
      elif [ -f "$D/node_modules/.bin/prettier" ]; then
        PKG="$D"; ENTRY=""; break
      fi
      PARENT=$(dirname "$D")
      [ "$PARENT" = "$D" ] && break
      D="$PARENT"
      I=$((I + 1))
    done
    [ -z "$PKG" ] && exit 0   # kein prettier im Baum -> still no-op (eslint --fix ist bewusst KEIN Fallback)

    # node + aufgeloester Eintrittspunkt bevorzugt (umgeht Exec-Bit-Probleme der .bin-Shim unter
    # Windows), sonst Fallback auf die Shim. prettier sucht .prettierrc selbst ab dem File aufwaerts.
    if [ -n "$ENTRY" ] && command -v node >/dev/null 2>&1; then
      ( cd "$PKG" && node "$ENTRY" --write "$FILE" ) >/dev/null 2>&1 || true
    else
      ( cd "$PKG" && ./node_modules/.bin/prettier --write "$FILE" ) >/dev/null 2>&1 || true
    fi
    RAN="prettier --write"
    ;;
  py)
    command -v ruff >/dev/null 2>&1 || exit 0   # ruff nicht auf PATH -> no-op
    ruff format "$FILE" >/dev/null 2>&1 || true
    RAN="ruff format"
    ;;
  *)
    exit 0
    ;;
esac

# Claude informieren, dass die Datei evtl. auf Disk veraendert wurde (verhindert stale Edits).
if [ -n "$RAN" ]; then
  MSG="format-on-write: ran ${RAN} on ${BASE} (file may have changed on disk; re-read before further edits)"
  printf '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"%s"}}' "$MSG"
fi

exit 0
