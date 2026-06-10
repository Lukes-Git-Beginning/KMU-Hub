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

---

## Phase 6 — Benachrichtigungs-Quick-Card + echte Account-Info + Settings-Shortcuts  ·  marathon/luke-fe  ·  Status: ⬜ ungereviewt

**Hinklicken (Pfad in der App):**
- Route(n): `/profil` → Tab „Mein Profil"
- Schritte:
  1. Quick-Card „Benachrichtigungen" direkt unterhalb des Header-Blocks: Sound-Toggle klicken (sofortiger Toggle, rein lokal). DND-Toggle zeigt „Deaktiviert" — im Dev ohne Backend disabled (grau, kein Klick möglich).
  2. Link „Alle Benachrichtigungs-Einstellungen" (oben rechts in der Karte) anklicken → öffnet `/settings` mit aktivem Tab „Benachrichtigungen" (Deep-Link via `location.state`).
  3. Karte „Kontoinformationen": zeigt echte Rolle aus Auth-Store (Administrator/Projektleiter/Mitarbeiter), E-Mail aus Auth-Store. Kein „Management", kein „Januar 2024".
  4. Karte „Einstellungen" (Shortcuts): Button „Erscheinungsbild" → öffnet `/settings` Tab „Darstellung"; Button „Sicherheit" → öffnet `/settings` Tab „Sicherheit".

**Was gebaut / Architektur-Entscheidungen:**
- **Quick-Card**: Bewusst kein Matrix-Duplikat (Backlog P2+P3 gemerged). Nur 2 Controls: DND (Backend) + Sound (lokal). Link zur vollen Settings-Matrix.
- **DND-Fallback**: Wenn `useDNDStatus` noch lädt (`isLoading=true`) oder Fehler (`isError=true`) → `dndBackendAvailable=false` → Switch disabled + Hinweistext. Konsistent mit NotificationSettingsTab (kein DND-Crash ohne Backend).
- **Deep-Link-Mechanik**: `navigate('/settings', { state: { tab: 'notifications' } })` — `location.state` wird in `SettingsPage.useState`-Initialwert ausgewertet (`stateTab ?? 'profile'`). URL bleibt sauber (`/#/settings`), kein Hash-Param. Minimal-invasiver Eingriff in SettingsPage (3 Zeilen: `useLocation` import + `stateTab` extract + `useState` initialwert).
- **Account-Info**: `memberSince`-Zeile entfernt (User-Objekt hat kein `created_at`). Rolle aus `user.roles[0]` mit bekannten Keys gemappt, unbekannte Rollen raw angezeigt.
- **E-Mail**: Aus `useAuthStore().user.email`, Fallback auf `profile.email` → kein Crash wenn user null.

**i18n:** 12 neue Keys `profil.notifications.*` + `profil.shortcuts.*` — alle 4 Sprachen (de/en/fr/it) synchron. Alle Umlaute korrekt (ü, ö). Keine `{{var}}`-Interpolation, keine `_one/_other`-Plurale.

**tsc-Gate:** 0 Fehler in geänderten Dateien (`tsconfig.phase6check.json`, scoped auf ProfilTab + SettingsPage).

**QA:** `desktop/scripts/qa-profil-account.mjs` — 4 Szenarien, alle Screenshots ansehen.
- rawKeys: [] (alle 3 Steps)
- pageErrors: [] (alle 3 Steps)
- hardcodedManagement: false, hardcodedJanuar2024: false
- soundToggled: true, notifTabActive: true, themeHeadingVisible: true

**Screenshots:** `desktop/.qa-screenshots/profil-account/` (01–08)

**Bekannte offene Punkte / Darien-Fragen:**
- **memberSince/created_at**: `/api/v1/auth/me` liefert kein `created_at`-Feld. Soll `UserInfo` um `created_at` erweitert werden? → Darien-Frage
- **DND im Profil ausreichend?**: Aktuell nur On/Off-Toggle ohne Ablaufzeit. Reicht das als Quick-Control, oder Matrix-Subset (Email/Push/InApp je Kanal) gewünscht? → Product-Entscheidung
- **Abteilung**: `profil.info.department` Key existiert noch in i18n, aber Feld entfernt (kein Mapping in User-Objekt). Keys könnten bei späterer HR-Integration genutzt werden.

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _<hier trägt der Reviewer Feedback ein → wird zu TaskCreate-Items>_

---

<!-- Phasen-Einträge hier anhängen — Struktur aus _TEMPLATE.md kopieren.
     Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->
