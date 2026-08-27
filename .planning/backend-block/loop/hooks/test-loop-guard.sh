#!/bin/bash
# Regressionstest fuer loop-guard.sh. Aus dem Repo-Root laufen lassen:
#   bash .planning/backend-block/loop/hooks/test-loop-guard.sh
# Muss vor JEDEM Nachtlauf gruen sein — der Guard ist die einzige harte Grenze
# zwischen dem Loop und einem unbeaufsichtigten Production-Deploy.

GUARD=".planning/backend-block/loop/hooks/loop-guard.sh"
FAILED=0

test_cmd() {
  desc="$1"; cmd="$2"; want="$3"
  json=$(python -c "import json,sys; print(json.dumps({'tool_name':'Bash','tool_input':{'command':sys.argv[1]}}))" "$cmd")
  echo "$json" | bash "$GUARD" >/dev/null 2>&1
  rc=$?
  got="ALLOW"; [ $rc -eq 2 ] && got="BLOCK"
  if [ "$got" = "$want" ]; then
    echo "  ok   [$got] $desc"
  else
    echo "  FAIL [got=$got want=$want] $desc :: $cmd"
    FAILED=$((FAILED + 1))
  fi
}

echo "=== MUSS BLOCKEN ==="
test_cmd "push main"                "git push origin main" BLOCK
test_cmd "push bare"                "git push" BLOCK
test_cmd "push HEAD:main"           "git push origin HEAD:main" BLOCK
test_cmd "push loop-branch"         "git push origin backend-loop" BLOCK
test_cmd "push -u loop-branch"      "git push -u origin backend-loop" BLOCK
test_cmd "push force"               "git push --force origin backend-loop" BLOCK
test_cmd "push --tags"              "git push origin backend-loop --tags" BLOCK
test_cmd "chained push main"        "cd /repo && git push origin main" BLOCK
test_cmd "gh pr create"             "gh pr create --draft --base main --head backend-loop" BLOCK
test_cmd "checkout main"            "git checkout main" BLOCK
test_cmd "switch main"              "git switch main" BLOCK
test_cmd "merge fremder Branch"     "git merge backend-loop" BLOCK
test_cmd "merge feature-branch"     "git merge feat/partition-ephemeral-logs" BLOCK
test_cmd "rebase (History-Rewrite)" "git rebase origin/main" BLOCK
test_cmd "rebase -i"                "git rebase -i HEAD~3" BLOCK
test_cmd "reset --hard"             "git reset --hard HEAD~1" BLOCK
test_cmd "branch -f main"           "git branch -f main HEAD" BLOCK
test_cmd "gh pr merge"              "gh pr merge 42 --squash" BLOCK
test_cmd "gh pr ready"              "gh pr ready 42" BLOCK
test_cmd "gh workflow run CD"       "gh workflow run CD" BLOCK
test_cmd "gh api write"             "gh api -X POST /repos/x/y/merges" BLOCK
test_cmd "prod ssh"                 "ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195" BLOCK
test_cmd "prod ip"                  "curl https://178.104.38.195/health" BLOCK
test_cmd "env.production"           "cat /opt/kmuhub/.env.production" BLOCK
test_cmd "prod compose"             "docker compose -f deploy/docker/docker-compose.prod.yml ps" BLOCK
test_cmd "deploy.sh"                "sudo bash deploy/scripts/deploy.sh --force" BLOCK

echo "=== MUSS DURCHLASSEN ==="
test_cmd "merge origin/main"        "git merge origin/main --no-edit" ALLOW
test_cmd "merge origin/main ff"     "git merge --ff-only origin/main" ALLOW
test_cmd "merge --abort"            "git merge --abort" ALLOW
test_cmd "rebase --abort"           "git rebase --abort" ALLOW
test_cmd "fetch"                    "git fetch --all --prune" ALLOW
test_cmd "commit mit Wort 'main'"   "git commit -m 'fix(crm): align wire shape with main branch contract'" ALLOW
test_cmd "checkout -- . (verwerfen)" "git checkout -- ." ALLOW
test_cmd "checkout loop branch"     "git checkout backend-loop" ALLOW
test_cmd "log main..loop"           "git log --oneline main..backend-loop" ALLOW
test_cmd "go build"                 "go build -p 2 ./internal/document/..." ALLOW
test_cmd "golangci-lint"            "golangci-lint run --config backend/.golangci.yml" ALLOW
test_cmd "go test"                  "go test ./internal/document/..." ALLOW
test_cmd "lokales compose"          "docker compose -f deploy/docker/docker-compose.yml up -d postgres" ALLOW
test_cmd "lokale Migration"         "migrate -path backend/migrations -database \$DB up" ALLOW

echo "=== BACKLOG-YAML-GATE ==="
# Der Guard blockt einen Commit, der BACKLOG.yml unparsebar hinterlaesst. Der
# Fixture-Pfad kommt ueber LOOP_GUARD_BACKLOG, damit der Test die echte Datei
# nicht anfassen muss.
BROKEN_BACKLOG=$(mktemp)
printf 'units:
  - id: kaputt
    status: todo
    done_when:
      - `beginnt mit backtick`
' > "$BROKEN_BACKLOG"
commit_json=$(python -c "import json; print(json.dumps({'tool_name':'Bash','tool_input':{'command':'git commit -m test'}}))")
echo "$commit_json" | LOOP_GUARD_BACKLOG="$BROKEN_BACKLOG" bash "$GUARD" >/dev/null 2>&1
rc=$?
if [ $rc -eq 2 ]; then
  echo "  ok   [BLOCK] commit bei unparsebarem BACKLOG.yml"
else
  echo "  FAIL [rc=$rc want=2] commit bei unparsebarem BACKLOG.yml"
  FAILED=$((FAILED + 1))
fi
rm -f "$BROKEN_BACKLOG"
test_cmd "commit bei heilem BACKLOG.yml" "git commit -m 'chore(loop): mark unit in progress'" ALLOW

echo ""
if [ $FAILED -eq 0 ]; then
  echo "Guard gruen — alle Faelle korrekt."
  exit 0
fi
echo "Guard ROT — $FAILED Fall/Faelle falsch. NICHT starten."
exit 1
