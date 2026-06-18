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

## Phase 9 — UserProfileCard-Overlay + Ping→Chat  ·  marathon/luke-fe  ·  Status: ⬜ ungereviewt

**Hinklicken (Pfad in der App):**
- Route(n): Kommunikation/Team-Chat, Dashboard, Profil/Zeiterfassung, Aufgaben
- Schritte:
  1. `/kommunikation?bereich=team` → Channel „allgemein" öffnen → Members-Panel (Personen-Icon rechts oben) → Avatar eines Members anklicken → Karte mit Name, Rolle, Abteilung, Presence-Badge, E-Mail erscheint
  2. In der offenen Karte: „Nachricht senden" → navigiert zu `/kommunikation?bereich=team` und selektiert den DM-Channel
  3. Dashboard (`/`) → Widgets-Abschnitt (nach unten scrollen) → TeamStatus-Widget → Member-Avatar anklicken → gleiche Karte
  4. Eigenes Profil-Avatar (Topbar) → Karte zeigt „Profil öffnen"-Link statt „Nachricht senden"

**Worauf achten (Feinschliff):**
- [ ] Karte öffnet via Popover (Radix, kein HoverCard) — nur bei Klick, nicht bei Hover
- [ ] `stopPropagation()` am Trigger: Klick löst keine Row/Button-Parent-Aktion aus
- [ ] Presence-Badge (grüne/gelbe/rote/graue Dot) spiegelt globalen `usePresenceStore` korrekt
- [ ] Own-Profile: zeigt „Profil öffnen" (navigiert `/profil`), KEIN „Nachricht senden"
- [ ] Other-User: zeigt „Nachricht senden" (Button, disabled while loading DM)
- [ ] Keine Raw-i18n-Keys (`profil.card.*`), keine Emojis, alle 4 Sprachen (de/en/fr/it)
- [ ] Kein „undefined" bei Usern ohne Position/Abteilung (S4-QA grün)
- [ ] Call-Sites alle 5: MessageBubble, ChannelMemberList, TeamView, CommentThread, TeamStatus

**i18n:** 5 neue Keys `profil.card.*` (loadError, openProfile, sendError, sendMessage, sending) — alle 4 Sprachen synchron, korrekte Umlaute, `{var}`-Interpolation.

**tsc-Gate:** Granular grün (chat.ts-Handler, UserProfileCard.tsx, navigation.ts je 0 Fehler isoliert). ⚠ Scoped-tsc über `ChatLayout.tsx` bzw. `MessageList/MessageBubble/ChannelMemberList` **crasht den TS-Compiler** („Debug Failure. No error for last overload signature") — Tooling-Bug, kein Code-Fehler (Vite-Build + App laufen, QA grün). Crash betrifft auch den committeten Stand; Bisektion: Bestands-Chat-Dateien (ChannelList/Header/SearchPanel/Mentions/Thread/MessageInput) crashen NICHT (nur vorbestehende typed-i18n-Baseline-Fehler in ChannelHeader.tsx:197). Künftige Gates im Chat-Scope granular schneiden.

**QA:** `desktop/scripts/qa-phase9.mjs` — 4 Szenarien, alle 4/4 grün (Hauptsession nachgefahren, Screenshots angesehen).
- S1: Chat Members-Panel → Avatar → Karte sichtbar (Presence-Badge, kein undefined)
- S2: „Nachricht senden" → **harter Assert:** Chat-Header (h2) zeigt exakt den Namen aus der Karte (`headerAfterSend === cardName`) → DM nachweislich selektiert
- S3: Dashboard TeamStatus-Widget → Avatar → Karte sichtbar
- S4: Card ohne Position/Abteilung → kein undefined, kein Raw-Key

**Screenshots:** `desktop/.qa-screenshots/phase9-profile-card/` (s1-01 bis s4-01)

**Architektur-Entscheidungen:**
- **NavigationIntent** `send-message`: Daten-Typ von `Record<string,string>` auf `{name, userId?, channelId?, contactId?}` spezifiziert (additiv, rückwärtskompatibel)
- **ChatLayout** konsumiert Intent mit `channelId` **reaktiv** (Effect auf Store-Wert, nicht mount-only) — nötig, weil die Karte auch INNERHALB des Chats geöffnet wird (MessageBubble/ChannelMemberList) und ChatLayout dabei gemountet bleibt. Konsumiert NUR send-message-Intents mit channelId; andere Intent-Typen bleiben für ihre Konsumenten liegen.
- **MSW-Mock** `toHrEmployee` (team.ts): `userId`, `userName`, `userEmail`, `positionTitle` als camelCase ergänzt (war snake-only, bedingte `member.userId`-Prüfung in TeamStatus)
- **MSW-Mock** `POST /channels/dm` (chat.ts): echtes Get-or-create — existierender DM pro `other_user_id` wird wiederverwendet, neue DMs werden mit aufgelöstem Namen (EMPLOYEES-Lookup) in `mockDMs` registriert, damit Channel-Detail/Liste danach auflösen (vorher: 404 → leerer Header nach Ping→Chat).

**Hauptsession-Spot-Check (Nachtrag, vor Push per Amend gefixt):**
1. **Ping→Chat aus dem Chat heraus war defekt** — Intent wurde nur im Mount-Effect konsumiert; bei offenem Chat blieb der alte Channel aktiv (Screenshot-Beweis s2-02 alt). Fix: reaktiver Effect (s.o.).
2. **S2-Assert war weich** — `pass` hing an `urlHasKomm` (trivial wahr, Test startet auf /kommunikation); `dmChannelVisible` war eine Tautologie. Fix: harter Header-Name-Assert. Wiederholung des Phase-7-Musters → Lesson bleibt scharf.
3. QA-`BASE` war auf Port 5174 hartkodiert (Zufall der Subagent-Umgebung) → `process.env.QA_BASE ?? 5173`.

**Bekannte offene Punkte / Backend-Bedarf:**
- `useEmployee(id)`: Lazy-Fetch bei Popover-Öffnung — echtes Backend liefert Employee-Details; Mock gibt bereits konsistente Daten
- `useGetOrCreateDM()`: Mutation erstellt/findet DM-Channel; ohne Backend fällt die Navigation auf `channelId=undefined` zurück (navigiert trotzdem zu `/kommunikation`, kein Crash)
- Presence: lokal gemockt via `usePresenceStore`; echte Server-Presence = Backend
- Members-Panel zeigt für frisch erstellte DM-Channels generische „Unbekannt"-Mock-Member (statisches `mockChannelMembers`, channel-unabhängig) — vorbestehende Mock-Limitierung, kosmetisch
- Mock-Daten mit ASCII-Umlauten sichtbar in der Karte („Geschaeftsfuehrer" aus `mock-db.ts` EMPLOYEES) — Teil des bekannten repo-weiten Umlaut-Sweeps für main (Session-4-Lesson 3), NICHT in dieser FE-Lane fixen
- **Offene Frage für Darien:** TS-Compiler-Crash bei scoped tsc über den Chat-Graphen (s. tsc-Gate) — TS-Versions-Bump als Fix evaluieren? Restliche 9 Avatar-Call-Sites (ProfileMenu, SidebarUser, ClassicSidebar, MentionAutocomplete, NotificationsFeed, ActivitySection, IncomingCallOverlay, CampaignCard, ChannelList) bewusst nicht migriert — Follow-up-Phase?

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _<hier trägt der Reviewer Feedback ein → wird zu TaskCreate-Items>_

---

<!-- Phasen-Einträge hier anhängen — Struktur aus _TEMPLATE.md kopieren.
     Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->
