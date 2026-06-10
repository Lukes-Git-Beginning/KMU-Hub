# Review-Fäden — dokumente

> Eine Datei pro Modul. Der Bau-Strom trägt nach JEDER fertigen Phase einen Eintrag ein (Struktur siehe `_TEMPLATE.md`). Vorlage für Dariens Feinschliff-Review (Pfad nachklicken, worauf achten, Screenshots, offene Punkte).

**Modul:** `dokumente` · **Strom:** D · **Reviewer (zugeteilt):** offen

---

<!-- Phasen-Einträge hier anhängen — Struktur aus _TEMPLATE.md kopieren.
     Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->

## ⬜ Pilot-Phase — DokumenteSettingsPanel + verdrahtete Prefs (2026-06-10)

**Spec-Korrektur zuerst (Lehre „Code vor Spec prüfen" bestätigt):** Der Phasenplan listet für P1 „Move/Copy-Endpoints durch Luke" als Backend-Bedarf — **die existieren längst** (`POST /files/{id}/copy|move` in `route_document.go`, FE-Hooks `useCopyFile`/`useMoveFile` fertig). Es fehlt nur der Dialog im FE. P1-Scope entsprechend kleiner als geplant.

**Hinklick-Pfad:**
- Route: `/dokumente` → unten links „Modul-Einstellungen" → Eintrag **„Dokumente"** (Context-Preselect wählt ihn automatisch, AKTIV-Badge)
- Toolbar: `/dokumente` → Sortier-Button (Pfeil-Icon) neben Grid/Liste-Toggle

**Gebaut:**
- `stores/dokumentePrefs.ts` (persönlich: Standard-Ansicht, Sortierung Feld+Richtung, Dichte) + `stores/dokumenteSettings.ts` (tenant, mock-first: Quota-Anzeige je Tarif, erlaubte Dateityp-Gruppen, Standard-Freigabe, OnlyOffice an/aus, Papierkorb-Tage)
- `modules/dokumente/settings/` — Panel via `ModuleSettingsShell` (1 personal + 3 tenant Sektionen), registriert in `module-settings-registry.tsx` (additiv, Hot File)
- **Prefs greifen real:** Standard-Ansicht = Fallback für Ordner ohne eigene Wahl; **SortMenu jetzt in der Toolbar** (Sort-State existierte, hatte aber kein UI — `shared/SortMenu`, schreibt in die Pref zurück); Dichte kompakt = engere Listenzeilen + 6-spaltiges Grid; OnlyOffice-Toggle blendet „In OnlyOffice bearbeiten" im Kontextmenü aus
- Demo-Handler: `GET /documents/files` sortiert jetzt nach `sort_field`/`sort_dir` (ignorierte die Parameter vorher — Sortierung war im Demo wirkungslos)
- i18n ×4 (41 Keys `dokumente.settings.*` + `moduleSettings.entries.dokumente`), `{var}`-Syntax

**Worauf achten (Feinschliff):**
- [ ] Settings-Panel voll + 760px schmal (Screenshots `dokumente-settings-half.png`)
- [ ] SortMenu-Trigger in der Toolbar: Platzierung/Gewicht okay neben View-Toggle?
- [ ] Dichte „Kompakt": Grid 6 Spalten zu eng bei 1280er-Screens?
- [ ] Quota-Tabelle: Tarifnamen Starter/Business/Enterprise = finale Pricing-Namen? (aus `pricing.md` übernommen, ggf. ORBIT-Naming)
- [ ] Keine Raw-Keys/Emojis/ASCII-Umlaute (QA: rawKeys [] in Panel+Page)

**Screenshots:** `desktop/.qa-screenshots/dokumente-*.png` (Panel, SortMenu offen, Liste kompakt+Name↑, Grid kompakt, 760px, Tenant-Sektionen unten)

**Verify:** gescopter tsc (`tsconfig.dokumente-settings.json`): **0 Fehler in geänderten Dateien** (Restfehler = bekannte Baseline in transitiven Imports wie `document-client`/`useProjects`) · `scripts/qa-dokumente-settings.mjs` grün (alle Sektionen, Pref→Reload→greift, rawKeys [], pageErrors [], failedApi []) · Screenshots angesehen. Nebenbefund gefixt: vorbestehender Typed-t()-Fehler in `ClassificationBadge` (`as const satisfies`).

**Bekannte offene Punkte / Backend-Bedarf:**
- 🟡 Tenant-Settings mock-first — Lukes Settings-Foundation (Migration 138, `route_settings.go`) ist inzwischen auf main; Wiring auf `tenant_settings` als Folgephase, wenn Branch gemerged ist
- 🟡 Lock-Ansicht (Nicht-Modulleiter) generisch via Shell, nicht eigens ge-QA-t (Demo-User ist Lead)
- 🟡 Dateityp-Gruppen/Standard-Freigabe werden noch nicht beim Upload/Share enforced (reine Settings-Verwaltung; Enforcement = P1-Folge)
- Nächste Phase (P1, korrigierter Scope): Move/Copy-Dialog (API fertig), echter Download (`useFileDownloadURL` ungenutzt), Demo-Handler-Lücken (Versions/Shares/Copy/Move/Upload)

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _…_
