# Review-Fäden — zeiterfassung

> Eine Datei pro Modul. Der Bau-Strom trägt nach JEDER fertigen Phase einen Eintrag ein (Struktur siehe `_TEMPLATE.md`). Vorlage für Dariens Feinschliff-Review (Pfad nachklicken, worauf achten, Screenshots, offene Punkte).

**Modul:** `zeiterfassung` · **Strom:** D · **Reviewer (zugeteilt):** offen

---

<!-- Phasen-Einträge hier anhängen — Struktur aus _TEMPLATE.md kopieren.
     Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->

## 🟡 Befund — Spec-Abgleich vor P1: Modul pausiert, Architektur-Frage an Luke (2026-06-10)

**Kein Code geändert.** Vor dem Bauen Code-Stand geprüft (Tageslehre) — Ergebnis kippt die Spec komplett:

**Es gibt ZWEI parallele Zeiterfassungs-UIs:**
1. **Live** (`modules/profil/tabs/ZeiterfassungTab.tsx`, auch auf `/zeiterfassung` gemountet): **API-backed** via `hr-hooks.ts` gegen das existierende HR-Backend (`/api/v1/hr/time/*` — Clock-In/Out, Pausen, ArbZG-Severity, Daily/Weekly-Summaries, Korrektur-Workflow mit Genehmigung). 3 Inline-Views: Heute/Woche/Korrekturen. Funktional solide, aber UI karger als der tote Satz.
2. **Tot** (`modules/profil/tabs/zeiterfassung/` — 10 Dateien: TodayView, WeekView, MonthView, OverviewView, ReportsView, TeamView, CategoriesView, ManualEntryForm, ExportDialog, ApprovalBanner): **nirgends gemountet** (kein externer Import), hängen am lokalen Mock-Store `stores/timetracking.ts`. UI-reich (Dashboard, 4-Wochen-Trend, Export-Dialog, Kategorien/Templates, Wochenfreigabe), aber Daten rein lokal.

**Dazu zwei konkurrierende Topbar-Widgets:** `ClockInButton` (echte API) und `TimeTrackerWidget` (Mock-Store, navigiert zudem nach `/profil` statt `/zeiterfassung`) — nicht synchronisiert, einer zeigt im Echtbetrieb falsche Daten.

**Warum pausiert:** Jede Weiterarbeit (Shell, Projekt-Picker, Export) erfordert zuerst die Datenquellen-Entscheidung — toten Satz an die HR-API anschließen (UI-Substanz retten) vs. Live-Tab ausbauen + toten Satz löschen (wie calendar-Cleanup). Das berührt Lukes HR-Backend-Lane → **Entscheidung mit Luke**, dann ist zeiterfassung wieder Strom-D-baubar.

**Backend-Gaps unabhängig von der Entscheidung** (in `backend-gaps.md` ergänzt): manueller Zeiteintrag (es gibt nur Clock-In/Out + Korrekturen), `project_id`/`customer_id` am WorkTimeEntry, Export-Endpoint (CSV/DATEV), Wochen-Freigabe-Workflow.

**Reviewer-Notizen:**
- _Entscheidung Luke/Darien hier eintragen → daraus werden die echten P1-Phasen._
