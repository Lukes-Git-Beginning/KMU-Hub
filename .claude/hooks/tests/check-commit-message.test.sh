#!/bin/bash
# Testet .claude/hooks/check-commit-message.sh gegen beide Quote-Formen.
#
# ENTSTEHUNGSGRUND: Am 2026-08-24 stellte sich beim scharfen Ausloesen heraus, dass der Hook
# doppelt gequotete Messages nie geprueft hat — der PreToolUse-stdin ist JSON, dort steht
# `-m \"...\"`, der Lookbehind suchte aber `-m "`. Single-Quotes waren nie betroffen, deshalb
# fiel es monatelang nicht auf. Dieser Test haelt beide Formen fest.
cd "$(dirname "$0")/../../.." || exit 1
H=.claude/hooks/check-commit-message.sh
FAIL=0

t() {
  local name="$1" cmd="$2" want="$3"
  local json out code
  json=$(printf '{"tool_name":"Bash","tool_input":{"command":%s}}' \
    "$(printf '%s' "$cmd" | PYTHONIOENCODING=utf-8 python -c 'import json,sys; print(json.dumps(sys.stdin.read(), ensure_ascii=False))')")
  out=$(printf '%s' "$json" | bash "$H" 2>&1); code=$?
  if [ "$code" -eq "$want" ]; then
    printf "  OK    exit=%s  %s\n" "$code" "$name"
  else
    printf "  FEHL  exit=%s (erwartet %s)  %s\n" "$code" "$want" "$name"
    FAIL=$((FAIL+1))
  fi
}

echo "=== Muss BLOCKEN (exit 2) ==="
t 'kaputt, double quotes'  'git commit -m "kaputte message ohne praefix"'          2
t 'kaputt, single quotes'  "git commit -m 'kaputte message ohne praefix'"          2
t 'falsches Praefix'       'git commit -m "update: something"'                     2
t 'Praefix ohne Doppelp.'  'git commit -m "feat add contact endpoint"'             2

echo ""
echo "=== Muss DURCHGEHEN (exit 0) ==="
t 'feat, double quotes'    'git commit -m "feat: add contact endpoint"'            0
t 'fix mit Scope'          'git commit -m "fix(kontakte): guard against null"'     0
t 'chore, single quotes'   "git commit -m 'chore: bump deps'"                      0
t 'breaking change'        'git commit -m "feat(api)!: drop v1 endpoints"'         0
t 'docs'                   'git commit -m "docs: describe the backup procedure"'   0
t 'refactor'               'git commit -m "refactor: extract the parser"'          0
t 'test'                   'git commit -m "test: cover the empty case"'            0
t 'kein git commit'        'git log --oneline -5'                                  0

echo ""
if [ "$FAIL" -eq 0 ]; then echo "ALLE 12 FAELLE GRUEN"; else echo "$FAIL FAELLE ROT"; fi
exit "$FAIL"
