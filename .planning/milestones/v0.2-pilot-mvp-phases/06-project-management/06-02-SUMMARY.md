---
phase: "06"
plan: "02"
subsystem: "work-service-backend"
tags: ["go", "project-management", "service-layer", "repository-pattern", "unit-tests"]
dependency-graph:
  requires: ["06-01"]
  provides: ["project-service", "status-service", "project-repository", "status-repository"]
  affects: ["06-03", "06-04", "06-05"]
tech-stack:
  added: []
  patterns: ["repository-interface", "mock-repository-testing", "membership-based-authorization"]
key-files:
  created:
    - "backend/internal/work/project/errors.go"
    - "backend/internal/work/project/repository.go"
    - "backend/internal/work/project/postgres_repository.go"
    - "backend/internal/work/project/service.go"
    - "backend/internal/work/project/service_test.go"
    - "backend/internal/work/status/errors.go"
    - "backend/internal/work/status/repository.go"
    - "backend/internal/work/status/postgres_repository.go"
    - "backend/internal/work/status/service.go"
    - "backend/internal/work/status/service_test.go"
  modified: []
decisions:
  - key: "project-key-normalization"
    value: "Keys auto-normalize to uppercase (lowercase 'abc' becomes 'ABC'), validation only rejects non-alphanumeric"
  - key: "status-authorization-delegation"
    value: "Status service trusts caller for authorization; project membership checks done at gRPC server layer"
  - key: "preference-nil-return"
    value: "GetUserPreference returns nil (not error) when no preference set, caller uses defaults"
metrics:
  duration: "~7min"
  completed: "2026-02-08"
---

# Phase 6 Plan 2: Project & Status Service Summary

Project and status service packages with repository interfaces, PostgreSQL implementations, business logic, and 71 unit tests.

## One-liner

Project CRUD with membership-based auth (owner/member/viewer) + per-project status management with reorder and template copy.

## Task Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Project package (repo, postgres, service, tests) | d9bc14b | 5 files in work/project/ |
| 2 | Status package (repo, postgres, service, tests) | 783b43e | 5 files in work/status/ |

## What Was Built

### Project Package (`internal/work/project/`)

**Repository Interface** (17 methods):
- CRUD: Create, GetByID, List, Update, Archive
- Members: AddMember, RemoveMember, UpdateMemberRole, GetMember, ListMembers, IsMember, CountOwners
- Templates: SaveAsTemplate, GetForTemplate
- Preferences: GetUserPreference, SetUserPreference
- Checks: KeyExists, GetProjectKey

**PostgreSQL Implementation**:
- List query JOINs project_members to filter by user access (admin bypasses)
- Subqueries for member_count, task_count, owner_name in list views
- AddMember uses ON CONFLICT DO NOTHING for idempotency
- SaveAsTemplate copies project via INSERT...SELECT with is_template=true
- KeyExists checks among active (non-archived) projects only

**Service Business Logic**:
- Create: name validation (3-255 chars), key validation (2-10 uppercase alphanumeric), unique key check, auto-add creator as owner
- Get: membership check (admin bypass)
- Update: owner-or-admin authorization, archived project protection
- Archive: owner-or-admin, sets archived_at timestamp
- AddMember: owner-or-admin, validates role, archived protection
- RemoveMember: owner-or-admin, cannot remove last owner
- UpdateMemberRole: owner-or-admin, cannot demote last owner
- SaveAsTemplate: validates source exists, name/key validation
- CreateFromTemplate: validates source is template, copies description, adds creator as owner
- Preferences: validates view_type is "list" or "kanban"

**Unit Tests**: 43 tests covering all service methods, authorization, edge cases

### Status Package (`internal/work/status/`)

**Repository Interface** (11 methods):
- CRUD: Create, GetByID, ListByProject, Update, Delete
- Reorder: bulk sort_order update within transaction
- Checks: CountTasksWithStatus, HasDefault, GetNextSortOrder, NameExistsInProject
- Template: CopyStatusesForProject (INSERT...SELECT with new UUIDs)

**Service Business Logic**:
- Create: name validation (1-100 chars), case-insensitive duplicate name check, auto sort_order
- Update: name validation, duplicate name check (excluding self)
- Delete: cannot delete default status (is_default=true)
- Reorder: validates all IDs belong to specified project, no duplicates
- CopyForProject: delegates to repo for template instantiation

**Unit Tests**: 28 tests covering all service methods, reorder edge cases, template copy

## Decisions Made

1. **Project key normalization**: Keys are auto-uppercased before validation. Input "mp" becomes "MP". Only non-alphanumeric characters are rejected.
2. **Status authorization delegation**: Status service does NOT check project membership -- the gRPC server layer handles that before calling status service methods.
3. **Nil preference pattern**: GetUserPreference returns nil (not error) when no preference exists, allowing callers to apply defaults.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed lowercase key test case**
- **Found during:** Task 1 verification
- **Issue:** Test expected "abc" key to fail with ErrKeyInvalidChars, but service normalizes to uppercase "ABC" first (which is valid)
- **Fix:** Replaced "lowercase" test case with "dot" (A.B) which is genuinely invalid
- **Files modified:** service_test.go
- **Commit:** d9bc14b (included in task commit)

## Verification Results

- `go test ./internal/work/... -v -count=1`: 71/71 PASS (43 project + 28 status)
- `go vet ./internal/work/...`: clean
- `go build ./internal/work/...`: clean
- Project service enforces: name validation, key uniqueness, membership checks, owner protection
- Status service enforces: name uniqueness per project, default status protection, reorder validation

## Next Phase Readiness

Both packages are ready for:
- **06-03** (Task Service): Tasks reference projects and statuses; both services provide the needed interfaces
- **06-04** (gRPC Server): Project and status services ready for handler wiring
- **06-05** (Gateway HTTP routes): Services ready for HTTP handler layer

## Self-Check: PASSED
