#!/bin/bash
# Backend-Nachtloop — harte Sicherheitsgrenze.
#
# PreToolUse-Hook auf Bash + PowerShell. Wird NUR waehrend Loop-Laeufen aktiv
# (kommt ueber .planning/backend-block/loop/loop-settings.json via --settings,
# nicht ueber .claude/settings.json — die kanonischen Hooks bleiben unberuehrt).
#
# Exit 2 = blockieren (stderr geht an Claude zurueck), Exit 0 = erlauben.
#
# Warum ein Hook und kein Prompt-Text: dokumentierte Faelle, in denen Agenten
# "nicht pushen" ignoriert haben (Welle 2: 2/3 pushten trotzdem auf main ->
# CI rot + Stub-CD nach Production). Text ist keine Grenze, exit 2 ist eine.

set -uo pipefail

INPUT=$(cat)

# Kommando aus dem Hook-JSON ziehen. Kein jq auf dieser Maschine -> Python.
PY=""
if command -v python >/dev/null 2>&1; then
  PY="python"
elif command -v python3 >/dev/null 2>&1; then
  PY="python3"
fi

CMD=""
if [ -n "$PY" ]; then
  CMD=$(printf '%s' "$INPUT" | "$PY" -c \
    "import sys,json;d=json.load(sys.stdin);print(d.get('tool_input',{}).get('command',''))" 2>/dev/null)
fi

# Fallback: laesst sich das Kommando nicht sauber extrahieren, wird das rohe
# JSON gescannt. Konservativ statt fail-open — ein Fehlalarm kostet eine
# Iteration, ein durchgerutschter Push kostet einen Production-Deploy.
if [ -z "$CMD" ]; then
  CMD="$INPUT"
fi

block() {
  echo "BLOCKIERT vom Backend-Nachtloop-Guard: $1" >&2
  echo "" >&2
  echo "Erlaubt ist ausschliesslich Arbeit auf dem Branch 'backend-loop'." >&2
  echo "Du committest lokal und pushst hoechstens 'git push origin backend-loop'." >&2
  echo "main, Merges, Production und Deploy gehoeren Luke — trag den Punkt ins" >&2
  echo "JOURNAL.md ein und mach mit der naechsten Unit weiter." >&2
  exit 2
}

# --- Git: push ---------------------------------------------------------------
# Erlaubt ist genau eine Form: der Loop-Branch auf origin. Alles andere blockt.
if echo "$CMD" | grep -Eq '(^|[^a-zA-Z-])git[[:space:]]+push'; then
  if echo "$CMD" | grep -Eq '(--force|--force-with-lease|[[:space:]]-f([[:space:]]|$))'; then
    block "Force-Push. Nie auf diesem Repo (Regel: kein Force-Push, kein reset --hard)."
  fi
  if echo "$CMD" | grep -Eq '(--tags|--all|--mirror)'; then
    block "git push --tags/--all/--mirror."
  fi
  if ! echo "$CMD" | grep -Eq 'git[[:space:]]+push([[:space:]]+(-u|--set-upstream))?[[:space:]]+origin[[:space:]]+(HEAD:)?backend-loop[[:space:]]*($|&|\||;)'; then
    block "Push-Ziel ist nicht 'origin backend-loop'. Push auf main loest CI->CD und damit einen unbeaufsichtigten Production-Deploy aus."
  fi
fi

# --- Git: main anfassen ------------------------------------------------------
if echo "$CMD" | grep -Eq 'git[[:space:]]+(checkout|switch)([[:space:]]+-[a-zA-Z-]+)*[[:space:]]+main([[:space:]]|$|&|\||;)'; then
  block "Wechsel auf main. Der Loop arbeitet ausschliesslich auf backend-loop."
fi
if echo "$CMD" | grep -Eq 'git[[:space:]]+merge([[:space:]]|$)'; then
  block "git merge. Das Zusammenfuehren nach main entscheidet Luke nach dem Review."
fi
if echo "$CMD" | grep -Eq 'git[[:space:]]+branch[[:space:]]+(-f|-D|-M)[[:space:]]+main'; then
  block "Manipulation des main-Branches."
fi
if echo "$CMD" | grep -Eq 'git[[:space:]]+reset[[:space:]]+--hard'; then
  block "git reset --hard. Zum Verwerfen eines gescheiterten Bauversuchs: 'git checkout -- .' plus 'git clean -fd'."
fi

# --- GitHub ------------------------------------------------------------------
if echo "$CMD" | grep -Eq 'gh[[:space:]]+pr[[:space:]]+merge'; then
  block "gh pr merge. Der Draft-PR wird von Luke gemergt, nicht vom Loop."
fi
if echo "$CMD" | grep -Eq 'gh[[:space:]]+pr[[:space:]]+(ready|edit[^|]*--ready)'; then
  block "PR aus dem Draft-Status nehmen. Der PR bleibt Draft bis zum Review."
fi
if echo "$CMD" | grep -Eq 'gh[[:space:]]+workflow[[:space:]]+(run|enable|disable)'; then
  block "Workflow-Dispatch. Insbesondere 'gh workflow run CD' deployt nach Production."
fi
if echo "$CMD" | grep -Eq 'gh[[:space:]]+api[^|]*-X[[:space:]]*(POST|PUT|PATCH|DELETE)'; then
  block "Schreibender GitHub-API-Aufruf."
fi

# --- Production --------------------------------------------------------------
if echo "$CMD" | grep -Eq '178\.104\.38\.195|deploy@|hetzner_kmuhub|app\.zentria\.tech'; then
  block "Zugriff auf den Production-Server."
fi
if echo "$CMD" | grep -Eq '\.env\.production|PRODUCTION_TEMPLATE'; then
  block "Zugriff auf Production-Secrets."
fi
if echo "$CMD" | grep -Eq 'docker-compose\.prod\.yml|deploy/scripts/deploy\.sh|smoke-prod'; then
  block "Production-Compose oder Deploy-Skript."
fi

exit 0
