#!/usr/bin/env bash
# SessionStart-Hook: holt den neuesten Stand von origin und meldet Rueckstand.
# Luke pusht in dichten Wellen direct-to-main. Nicht-blockierend, faellt nie die Session.
# GIT_TERMINAL_PROMPT=0 = nie auf eine Credential-Abfrage warten (schnell scheitern statt haengen).
# timeout 8 = den Sessionstart nie laenger als 8s an einem langsamen Netz aufhalten. Vorher 20s:
# der Doctor-Lauf am 2026-08-24 mass ueber 38 Sessionstarts einen Median von 2,4s, aber ein
# Maximum von 20,99s — also exakt den alten Timeout, an dem der Start voll ausgesessen wurde.
# 8s deckt den Median mit Reserve; laeuft der Fetch laenger, ist der Stand eben eine Session alt.
#
# WARUM ZWEI AUSGABEKANAELE:
# Die Vorgaengerfassung schrieb nur `systemMessage`. Das zeigt die Warnung dem NUTZER, legt sie
# aber nicht in den Modell-Kontext — Claude sah den Rueckstand nicht. Am 2026-08-24 lagen so
# 431 Commits Rueckstand unbemerkt vor, obwohl der Hook bei jedem Start lief.
# `hookSpecificOutput.additionalContext` ist der Kanal, der Claude erreicht; `systemMessage`
# bleibt zusaetzlich drin, damit die Meldung im Terminal weiter sichtbar ist.
#
# ANGEPASST von der Setup-Kur 2026-08-24 (vorher `.claude/session-fetch.sh`, nur lokal und
# ungetrackt). Gehoert ins claude-command-center zurueckgespielt.

git rev-parse --is-inside-work-tree >/dev/null 2>&1 || exit 0

GIT_TERMINAL_PROMPT=0 timeout 8 git fetch -q 2>/dev/null

BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)

# Upstream bevorzugt; ohne gesetzten Upstream auf origin/<branch> ausweichen.
UPSTREAM=$(git rev-parse --abbrev-ref '@{u}' 2>/dev/null)
[ -z "$UPSTREAM" ] && git rev-parse --verify "origin/$BRANCH" >/dev/null 2>&1 && UPSTREAM="origin/$BRANCH"
[ -z "$UPSTREAM" ] && exit 0

BEHIND=$(git rev-list --count "HEAD..$UPSTREAM" 2>/dev/null || echo 0)
case "$BEHIND" in ''|*[!0-9]*) exit 0 ;; esac
[ "$BEHIND" -eq 0 ] && exit 0

MSG="git: ${BRANCH} liegt ${BEHIND} Commit(s) hinter ${UPSTREAM} — vor dem Arbeiten 'git pull', sonst wird auf einem veralteten Stand gebaut."
printf '{"systemMessage":"%s","hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}' "$MSG" "$MSG"

exit 0
