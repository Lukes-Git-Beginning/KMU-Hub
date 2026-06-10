# Review-Fäden — vertraege

> Eine Datei pro Modul. Der Bau-Strom trägt nach JEDER fertigen Phase einen Eintrag ein (Struktur siehe `_TEMPLATE.md`). Vorlage für Dariens Feinschliff-Review (Pfad nachklicken, worauf achten, Screenshots, offene Punkte).

**Modul:** `vertraege` · **Strom:** L · **Reviewer (zugeteilt):** offen

---

## Phase 1 — Modul-Einstellungen (VertraegeSettingsPanel)  ·  marathon/luke-fe  ·  Status: ⬜ ungereviewt

**Hinklicken (Pfad in der App):**
- Route(n): `/vertraege` → Sidebar unten „Modul-Einstellungen" → Eintrag „Verträge" (kontext-vorausgewählt)
- Schritte: Persönlich: Standard-Tab umstellen (z.B. Vorlagen) → Overlay schließen → `/vertraege` neu laden → Modul öffnet auf dem gewählten Tab. Tabellendichte „Kompakt" → Vertragsliste wird enger. Für alle: Vertragsart deaktivieren → „Vertrag anlegen" bietet sie nicht mehr an; Kündigungsfrist/Verlängerung ändern → Dialog vorbelegt.

**Worauf achten (Feinschliff):**
- [ ] Layout/Hierarchie bei voller Breite + schmal (760 geprüft, Screenshot)
- [ ] Keine Raw-i18n-Keys, keine Emojis, keine ASCII-Umlaute (QA: 0 Raw-Keys, alle 4 Sprachen befüllt)
- [ ] Interaktionen echt: Standard-Tab + Dichte + Erinnerungs-Vorauswahl greifen real; Vertragsarten-Toggle + Standardwerte steuern den Neu-Dialog real
- [ ] Erinnerungs-Resolve: persönliche Vorauswahl überschreibt Unternehmens-Standard (Checkbox „Standard des Unternehmens verwenden")
- [ ] Lock-Verhalten „Für alle" für Nicht-Modulleiter

**Screenshots:** `desktop/.qa-screenshots/vertraege-settings/` (panel-top, panel-tenant, panel-pref-set, module-default-tab, panel-760) — QA `desktop/scripts/qa-vertraege-settings.mjs`

**Bekannte offene Punkte / Backend-Bedarf:**
- Tenant-Settings laufen mock-first (`stores/vertraegeSettings.ts`, localStorage) — Persistenz via `tenant_settings` (module_id='vertraege') = Luke-Backend; Settings-Fundament-Endpoints existieren seit Migration 000138.
- Nummernkreis: nur Format + Vorschau; automatische Vergabe = Backend.
- Standard-Laufzeit (Monate): Setting vorhanden, greift noch nicht im Dialog (Enddatum-Autofill bewusst nicht in dieser Phase).
- **Offene Frage für Darien:** vertraege-Domain fehlt komplett in der OpenAPI-Spec (vorbestehend, betrifft nicht diese FE-Phase).
- Beobachtung (vorbestehend, global): Topbar rendert ein „⚙️"-Emoji (innerText-Dump bei 760) — kollidiert mit der No-Emoji-Regel, nicht Teil dieser Phase.

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _<hier trägt der Reviewer Feedback ein → wird zu TaskCreate-Items>_

---

<!-- Phasen-Einträge hier anhängen — Struktur aus _TEMPLATE.md kopieren.
     Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->
