#!/bin/bash
# Testet .claude/hooks/check-no-attribution.sh gegen 13 Faelle.
# Liegt bewusst als DATEI vor: staenden die Muster im Bash-Kommando, wuerde der Hook
# den Testaufruf selbst blocken (was er beim ersten Versuch auch getan hat).
cd "$(dirname "$0")/../../.." || exit 1
H=.claude/hooks/check-no-attribution.sh
FAIL=0

t() {
  local name="$1" cmd="$2" want="$3"
  local json out code
  json=$(printf '{"tool_name":"Bash","tool_input":{"command":%s}}' "$(printf '%s' "$cmd" | PYTHONIOENCODING=utf-8 python -c 'import json,sys; print(json.dumps(sys.stdin.read(), ensure_ascii=False))')")
  out=$(printf '%s' "$json" | bash "$H" 2>&1); code=$?
  if [ "$code" -eq "$want" ]; then
    printf "  OK    exit=%s  %s\n" "$code" "$name"
  else
    printf "  FEHL  exit=%s (erwartet %s)  %s\n" "$code" "$want" "$name"
    printf "        -> %s\n" "$(printf '%s' "$out" | head -1)"
    FAIL=$((FAIL+1))
  fi
}

AI_LINE='Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>'
SESSION='Claude-Session: https://claude.ai/code/session_01NvUiAbDG'
GEN='Generated with [Claude Code]'
ROBOT=$(printf '\xf0\x9f\xa4\x96')

echo "=== Muss BLOCKEN (exit 2) ==="
t "Co-Authored-By mit Claude"   "git commit -m 'chore: x' -m '$AI_LINE'"                 2
t "Generated with"              "git commit -m 'feat: y

$GEN'"                                                                                   2
t "Session-Link"                "git commit -m 'fix: z

$SESSION'"                                                                               2
t "Robot-Emoji"                 "git commit -m 'docs: a

$ROBOT Generated'"                                                                       2
t "amend mit Attribution"       "git commit --amend -m 'chore: b' -m '$AI_LINE'"         2
t "Heredoc-Commit"              "git commit -F - <<EOF
chore: c

$AI_LINE
EOF"                                                                                     2
t "Copilot statt Claude"        "git commit -m 'chore: d' -m 'Co-Authored-By: GitHub Copilot'" 2

echo ""
echo "=== Muss DURCHGEHEN (exit 0) ==="
t "menschlicher Co-Autor Luke"  "git commit -m 'feat: e

Co-Authored-By: Luke <luke@zentria.tech>'"                                               0
t "menschlicher Co-Autor Nico"  "git commit -m 'fix: f

Co-Authored-By: Nico Mueller <nico@zentria.tech>'"                                       0
t "normaler Commit"             "git commit -m 'feat(kontakte): add anonymisation'"      0
t "kein git commit"             "git log --oneline -5"                                   0
t "grep ohne commit"            "grep -rn Co-Authored-By docs/"                          0
t "npm run build"               "npm run build"                                          0

echo ""
if [ "$FAIL" -eq 0 ]; then echo "ALLE 13 FAELLE GRUEN"; else echo "$FAIL FAELLE ROT"; fi
exit "$FAIL"
