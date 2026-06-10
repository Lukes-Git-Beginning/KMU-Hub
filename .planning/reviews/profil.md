# Review-Fäden — profil

> Eine Datei pro Modul. Der Bau-Strom trägt nach JEDER fertigen Phase einen Eintrag ein (Struktur siehe `_TEMPLATE.md`). Vorlage für Dariens Feinschliff-Review (Pfad nachklicken, worauf achten, Screenshots, offene Punkte).

**Modul:** `profil` · **Strom:** L · **Reviewer (zugeteilt):** offen

---

## Phase 1 — Presence-Status + Avatar-Upload-UI  ·  marathon/luke-fe  ·  Status: ⬜ ungereviewt

**Hinklicken (Pfad in der App):**
- Route(n): `/profil` → Tab „Mein Profil" → Header-Karte
- Schritte: Status-Badge neben der Rolle anklicken → Dropdown (Online/Abwesend/Beschäftigt/Unsichtbar) → „Beschäftigt" wählen → Avatar-Punkt wird rot UND Topbar-Status wechselt auf „Nicht stören" → Reload → bleibt. Kamera-Button am Avatar → Bild wählen → Vorschau erscheint sofort, persistiert lokal (DataURL im settings-Store).

**Worauf achten (Feinschliff):**
- [ ] Status-Picker andockt an BESTEHENDEN `stores/presence.myStatus` (kein Parallel-Store) — spiegelt sich app-weit (Topbar, kommunikation)
- [ ] Dot-Farben konsistent mit kommunikation (emerald/amber/red/slate); „Unsichtbar" (offline) ist hier zusätzlich wählbar — kommunikation bietet nur 3 Stati an (Abweichung gewollt? → Feinschliff)
- [ ] Avatar-Vorschau: kein Bruch ohne Bild (Fallback-Initialen), Bilddatei-/Größen-Validierung (max 1,5 MB wegen localStorage)
- [ ] Keine Raw-i18n-Keys (QA: 0, 4 Sprachen), 760 geprüft

**Screenshots:** `desktop/.qa-screenshots/profil-presence/` (profil-header, picker-open, status-dnd, profil-760) — QA `desktop/scripts/qa-profil-presence.mjs`

**Bekannte offene Punkte / Backend-Bedarf:**
- Avatar: „Speichern" = mock-first (DataURL in `settings.profile.avatarUrl`, localStorage). Echter Upload-Endpoint (MinIO, `POST /users/avatar` o.ä.) = Luke — Toast weist auf Vorschau-Charakter hin.
- Presence: lokal persistiert (`stores/presence`), echte Server-Presence (Broadcast an andere Clients) = Backend.
- Statischer „Online"-Badge wurde durch den echten Picker ersetzt; alter Key `profil.status.online` bleibt ungenutzt zurück (kein Schaden).

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _<hier trägt der Reviewer Feedback ein → wird zu TaskCreate-Items>_

---

<!-- Phasen-Einträge hier anhängen — Struktur aus _TEMPLATE.md kopieren.
     Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->
