---
tags: [strategie, stack, entscheidungen]
updated: 2026-04-18
---
# Tech-Stack & Strategy Decisions

## Strategische Entscheidungen
- **Deutschland-First**: EUR default, 19%/7% MWSt, de-DE locale for beta
- **Payroll anti-feature**: NEVER built, integration only (Bexio/Lexware)
- **OnlyOffice → Collabora geplant** (MPL 2.0 sicherer als AGPL) — **noch NICHT umgesetzt**, OnlyOffice weiterhin aktiv (Prod mit `JWT_ENABLED=true`)
- **Modul-x-User pricing**: Kein fixes Tier-System, 23 Module (2-7 EUR/User/Mo), Branchenpakete mit 15% Rabatt — siehe [[pricing]]
- **"Kommunikation" module** = frontend name for Unified Inbox
- **Industry modules (14)** = zum Launch echt, Feature-Flag-gated via `modules.<name>` (Safety-Net) — siehe [[architektur]]
- **Mobile = PWA auf Desktop-Basis** (Entscheidung 2026-04-18) — kein React Native, `mobile/`-Ordner entfernt, ehrlicher Pitch
- **WASM-Plugin-System** — Feature-Flag OFF bis Phase D (`plugins.wasm=false` + Build-Tag `no_wasm`), siehe [[integrationen]]
- **Scope rename**: buchhaltung -> finanzen (invoices, quotes, dunning, DATEV only)

## Frontend-Bibliotheken

### i18n (seit 2026-04-06)
| Paket | Version | Zweck |
|-------|---------|-------|
| i18next | v26.0.3 | Core — flat keys, ICU, statisches Bundle |
| react-i18next | v17.0.2 | React Bindings — `useTranslation()` Hook |
| i18next-icu | v2.4.3 | ICU Plural/Select-Syntax |

Vollstaendige Dokumentation: [[i18n]]

### UI / Interaktion
| Paket | Version | Zweck |
|-------|---------|-------|
| ~~motion~~ | ~~v12~~ | Entfernt 2026-04-08 — 0 Imports, alle Animationen CSS/rAF |
| @dnd-kit/core + sortable + utilities | v6/v10/v3 | Drag & Drop |
| @xyflow/react | v12 | Flow-Diagramme (Workflow-Editor, Plugin-Graph) |
| @tanstack/react-virtual | v3.13 | Virtuelle Listen (MessageList, grosse Datasets) |
| @tiptap/* | v3.20 | Rich-Text-Editor (Wiki, Chat, Dokumente) |
| frimousse | v0.3 | Emoji-Picker (Chat) |
| sonner | v2 | Toast-Notifications |
| rrule | v2.8 | Kalender-Recurrence |
| dompurify + @types/dompurify | v3 | HTML-Sanitization (XSS-Hardening) — Wrapper `lib/sanitize.ts` (seit 2026-04-18) |

### Build & Performance (seit 2026-04-08)
| Paket | Version | Zweck |
|-------|---------|-------|
| babel-plugin-react-compiler | latest | Automatisches Memoization (annotation mode) |
| v8-compile-cache | latest | V8 Bytecode Cache fuer schnelleren Kaltstart |
| idb-keyval | v6 | IndexedDB Wrapper fuer React Query Persister |

### Fonts (self-hosted seit 2026-04-08)
- Plus Jakarta Sans (Display): latin + latin-ext, WOFF2 in `public/fonts/`
- JetBrains Mono (Mono): latin + latin-ext, WOFF2
- `@font-face` mit `font-display: swap`, kein Google CDN

### Basis-Stack (unveraendert)
| Paket | Version | Zweck |
|-------|---------|-------|
| React | v19 | UI Framework |
| Electron | v33 | Desktop Shell |
| TypeScript | v5.7 | Typsicherheit |
| Tailwind CSS | v4 | Styling (Vite-Plugin, pre-compiled, CSP-safe) |
| TanStack Query | v5 | Server State + Caching |
| React Router | v7 | Navigation |
| Zustand | v5 | Client State |
| Radix UI | 12 Primitives | Accessible Dialogs/Modals |
| openapi-fetch | — | Typisierter API-Client |

## Dev Tooling & Environment
- VS Code: Go, vscode-proto3, OpenAPI Editor, ESLint+Prettier, Tailwind IntelliSense
- GitHub CLI: `"C:/Program Files/GitHub CLI/gh.exe"`
- Go 1.25.6: `export PATH="$PATH:/c/Program Files/Go/bin:$HOME/go/bin"`
- protoc: `C:/Users/Luke/AppData/Local/Microsoft/WinGet/Packages/Google.Protobuf_Microsoft.Winget.Source_8wekyb3d8bbwe/bin/protoc.exe`
- golangci-lint v2.8, config: `backend/.golangci.yml` (version: "2")
- Windows 11, bash via Git Bash

## Verwandte Notes
- [[architektur]] — Technische Architektur
- [[i18n]] — Internationalisierung (i18next)
- [[design]] — Frontend Design System
- [[milestones]] — Feature-Phasen
- [[integrationen]] — Externe Services (Bexio, Lexware, LiveKit)
- [[deployment]] — Docker & CI/CD
- [[troubleshooting]] — Dev-Tooling & Pfade
