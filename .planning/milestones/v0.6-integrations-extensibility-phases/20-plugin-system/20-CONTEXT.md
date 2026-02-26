# Phase 20: Plugin System + Industry Templates

## Context

Final phase of KMU Hub v1. Implements the extensibility layer that enables per-company customization without core code changes. Two-tier plugin architecture: config-based (validation rules, workflow automation) for simple customizations, WASM-based (wazero runtime) for complex logic. Industry templates bundle pre-configured field definitions, validation rules, and workflow automations for common DACH verticals.

## Scope

- Config-based customization engine (validation rules, workflow rules)
- WASM plugin runtime with sandbox (memory limits, execution timeout, rate limiting)
- Plugin lifecycle management (install, enable, disable, uninstall)
- Hook system (before/after CRUD for all modules)
- gRPC plugin service (32 RPCs) + gateway HTTP routes
- Industry templates: Handwerk, Beratung, Handel (seeds)
- Fuhrpark reference WASM plugin with SDK
- Frontend admin UI (plugin management, rule editors, template gallery)

## Plans

| Plan | Focus | Files | LOC |
|------|-------|-------|-----|
| 20-01 | Data foundation + config validation | 4 | ~580 |
| 20-02 | WASM runtime + sandbox | 7 | ~770 |
| 20-03 | Extension points + gRPC plugin API | 14 | ~4,700 |
| 20-04 | Industry templates (Fuhrpark) | 15 | ~3,300 |

## Key Decisions

- wazero (pure Go) over wasmtime for zero-CGO deployment
- 64MB memory limit + 5s execution timeout per WASM invocation
- Token bucket rate limiting (100 calls/min per plugin)
- Before-hooks: sequential pipe (data flows through priority order)
- After-hooks: parallel fire-and-forget (side effects)
- Plugin service standalone binary on :50060 (gRPC) / :9100 (health)

## Completed

2026-02-26 -- All 4 plans implemented in commit 5827ec4
