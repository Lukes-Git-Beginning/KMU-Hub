---
tags: [strategie, stack, entscheidungen]
updated: 2026-03-05
---
# Tech-Stack & Strategy Decisions

## Strategische Entscheidungen
- **Deutschland-First**: EUR default, 19%/7% MWSt, de-DE locale for beta
- **Payroll anti-feature**: NEVER built, integration only (Bexio/Lexware)
- **Collabora replaces OnlyOffice** (MPL 2.0 safer than AGPL)
- **Modul-x-User pricing**: Kein fixes Tier-System, 23 Module (2-7 EUR/User/Mo), Branchenpakete mit 15% Rabatt — siehe [[pricing]]
- **"Kommunikation" module** = frontend name for Unified Inbox
- **Industry modules** = Phase 20 plugin candidates, NOT core
- **Scope rename**: buchhaltung -> finanzen (invoices, quotes, dunning, DATEV only)

## Dev Tooling & Environment
- VS Code: Go, vscode-proto3, OpenAPI Editor, ESLint+Prettier, Tailwind IntelliSense
- GitHub CLI: `"C:/Program Files/GitHub CLI/gh.exe"`
- Go 1.25.6: `export PATH="$PATH:/c/Program Files/Go/bin:$HOME/go/bin"`
- protoc: `C:/Users/Luke/AppData/Local/Microsoft/WinGet/Packages/Google.Protobuf_Microsoft.Winget.Source_8wekyb3d8bbwe/bin/protoc.exe`
- Windows 11, bash via Git Bash

## Verwandte Notes
- [[architektur]] — Technische Architektur
- [[milestones]] — Feature-Phasen
- [[integrationen]] — Externe Services (Bexio, Lexware, LiveKit)
- [[deployment]] — Docker & CI/CD
- [[troubleshooting]] — Dev-Tooling & Pfade
