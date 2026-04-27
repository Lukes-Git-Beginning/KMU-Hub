# DEPRECATED — buchhaltung Modul

**Status:** Dead code seit 2026-04-27. Wird in Folge-Sprint entfernt.

## Warum deprecated

- Kein Router-Eintrag in `App.tsx` — Pfad nicht erreichbar
- Kein Sidebar-Eintrag in `nav-items.ts`
- Importiert Komponenten aus `@/modules/finanzen/` (`BelegketteTab`, `BankingWidget`)
- Nutzt den alten Mock-Store `useFinanceStore` (Zustand) — `finanzen` ist auf TanStack Query + Backend migriert
- Doppelt vorhandene Tabs (Rechnungen, Angebote, Mahnungen, Belegkette, Banking) sind in `finanzen` schon backend-connected (ZUGFeRD, QR-Rechnung, GoBD, Bexio-Export)

## Was migriert wird (Folge-Sprint)

Drei unique Tabs aus `BuchhaltungPage.tsx` wandern als neue Tabs in den umbenannten Buchhaltungs-Hub (`finanzen`-Modul, Display-Name "Buchhaltung"):

1. **Ausgaben** (Zeile 342-378) — Approve/Reject-Workflow
2. **Transaktionen** (Zeile 380-575) — Einnahmen/Ausgaben-Journal
3. **Berichte** (Zeile 577+) — Bar-/Pie-Charts (entscheiden ob noetig oder via FinanceDashboard ersetzt)

## Plan-Referenz

`~/.claude/plans/als-n-chstes-brauch-ich-peppy-lighthouse.md` — Sprint 1A, Tasks 14-17.

## Bei Refactor beachten

- BuchhaltungPage nutzt `useFinanceStore` (alter Zustand-Mock). Die migrierten Tabs koennen den Mock vorerst weiter nutzen — Luke wired Backend in Sprint Backend nach.
- ExpenseFormDialog, InvoiceFormDialog etc. sind in beiden Modulen vorhanden (Duplikate). Bei Migration die `finanzen`-Variante nutzen, `buchhaltung`-Variante wird mit dem Folder geloescht.
