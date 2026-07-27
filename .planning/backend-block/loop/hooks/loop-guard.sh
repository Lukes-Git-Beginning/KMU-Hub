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
  echo "Erlaubt ist ausschliesslich lokale Arbeit auf dem Branch 'backend-loop'." >&2
  echo "Du committest lokal und pushst NIE - Pushes kosten Actions-Minuten und" >&2
  echo "triggern API-Key-abgerechnete Review-Workflows." >&2
  echo "main, Merges, Push, PR, Production und Deploy gehoeren Luke - trag den" >&2
  echo "Punkt ins JOURNAL.md ein und mach mit der naechsten Unit weiter." >&2
  exit 2
}

# --- Git: push ---------------------------------------------------------------
# Der Agent pusht GAR NICHT. Zwei Gruende:
#   1. Push auf main loest CI->CD und damit einen unbeaufsichtigten Production-
#      Deploy aus.
#   2. Jeder Push gegen einen offenen PR startet einen CI-Lauf - Runner-Minuten
#      auf einem privaten Repo. Zwanzig Pushes pro Nacht waeren zwanzig Laeufe
#      fuer ein Signal, das einer am Ende genauso gibt.
#
# KORREKTUR 2026-07-27: eine fruehere Fassung dieses Kommentars behauptete, zwei
# PR-Workflows liefen mit einem separat abgerechneten ANTHROPIC_API_KEY. Das ist
# falsch - ein solches Secret existiert im Repo nicht (`gh secret list`). Claude
# PR Review und Security Review nutzen CLAUDE_CODE_OAUTH_TOKEN, also dasselbe
# Abo, ueber das auch der Loop laeuft: sie kosten Cap, kein Geld. Die Blockade
# bleibt trotzdem, jetzt aus dem richtigen Grund (Actions-Minuten + Deploy).
#
# Den einen Push pro Nacht macht run-loop.ps1 nach der letzten Iteration. Der
# Treiber laeuft nicht unter diesem Hook - die Grenze fuer den Agenten bleibt
# damit unveraendert hart.
if echo "$CMD" | grep -Eq '(^|[^a-zA-Z-])git[[:space:]]+push'; then
  block "git push. Der Loop committet ausschliesslich lokal - jeder Push kostet Actions-Minuten und triggert API-Key-abgerechnete Review-Workflows. Luke pusht beim Review."
fi

# --- Git: main anfassen ------------------------------------------------------
if echo "$CMD" | grep -Eq 'git[[:space:]]+(checkout|switch)([[:space:]]+-[a-zA-Z-]+)*[[:space:]]+main([[:space:]]|$|&|\||;)'; then
  block "Wechsel auf main. Der Loop arbeitet ausschliesslich auf backend-loop."
fi
# Erlaubt ist ausschliesslich, origin/main IN den Loop-Branch zu ziehen. Das ist
# ungefaehrlich (main wird nie beschrieben) und die Alternative zu `rebase`:
# ein Rebase wuerde die Branch-Historie umschreiben und danach einen Force-Push
# erzwingen - den es hier nicht geben darf.
if echo "$CMD" | grep -Eq 'git[[:space:]]+merge([[:space:]]|$)'; then
  # Flags duerfen vor oder hinter der Ref stehen, deshalb wird nur geprueft, ob
  # im selben Kommando 'origin/main' bzw. '--abort' vorkommt.
  if ! echo "$CMD" | grep -Eq 'git[[:space:]]+merge[^;&|]*(origin/main|--abort)'; then
    block "git merge mit anderem Ziel als origin/main. Das Zusammenfuehren nach main entscheidet Luke nach dem Review."
  fi
fi

# Rebase schreibt die Branch-Historie um und macht den naechsten Push
# force-pflichtig. Auf dem Loop-Branch wird stattdessen gemergt.
if echo "$CMD" | grep -Eq 'git[[:space:]]+rebase([[:space:]]+(-i|--interactive))'; then
  block "interaktiver Rebase."
fi
if echo "$CMD" | grep -Eq 'git[[:space:]]+rebase([[:space:]]|$)' \
   && ! echo "$CMD" | grep -Eq 'git[[:space:]]+rebase[[:space:]]+--abort'; then
  block "git rebase. Er schreibt die Historie um und erzwingt danach einen Force-Push. Nimm stattdessen 'git merge origin/main'."
fi
if echo "$CMD" | grep -Eq 'git[[:space:]]+branch[[:space:]]+(-f|-D|-M)[[:space:]]+main'; then
  block "Manipulation des main-Branches."
fi
if echo "$CMD" | grep -Eq 'git[[:space:]]+reset[[:space:]]+--hard'; then
  block "git reset --hard. Zum Verwerfen eines gescheiterten Bauversuchs: 'git checkout -- .' plus 'git clean -fd'."
fi

# --- GitHub ------------------------------------------------------------------
if echo "$CMD" | grep -Eq 'gh[[:space:]]+pr[[:space:]]+(create|merge|reopen)'; then
  block "gh pr create/merge. Ein PR startet Actions und die API-Key-abgerechneten Review-Workflows. PRs legt Luke an."
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
