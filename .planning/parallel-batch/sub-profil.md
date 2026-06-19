# Sub-Terminal — profil review-reif machen (P-1 … P-5)

> **Du bist das Sub-Terminal im Klon `…/KMU-Hub-review`, Dev-Port 5174.** Lies ZUERST `.planning/parallel-batch/README.md` (Lane-Regeln, Build-+-Verify-Standard, Gates). Du baust **nur profil**. automatisierung gehört dem Main-Terminal — fass es nicht an.
>
> **Selbst-enthaltend.** Scope = „Dokumente echt (MSW) + current-user + Cleanup" (mock-first, siehe README „Scope-Entscheidungen"). **KEIN** echtes Avatar-/Dokument-Storage-Backend (Luke). Bau die 5 Phasen ohne Rückfragen ab. Melde Darien nach jeder Phase „P-x fertig, n/5".

## Branch-Setup (einmalig, ZUERST)
```bash
cd "C:/Users/darie/Documents/KMU-Hub-review"
git checkout main && git pull --ff-only origin main   # holt dieses Paket + Skeleton-Stand (0596e5bc+)
git checkout -b parallel/profil
cd desktop && npm run dev -- --port 5174               # Dev-Server in EIGENEM Port
```
Alle P-Punkte committest + pushst du auf **`parallel/profil`** (`git push -u origin parallel/profil`). **Kein** Rebase von main nötig — dein Branch ist isoliert. Das Main-Terminal merged am Ende kontrolliert.

## Ausgangslage (Ist-Abgleich 2026-06-19)
profil ist eine **Tab-Shell** (`modules/profil/ProfilPage.tsx`, 70 Z., 4 Tabs lazy: Profil / Abwesenheiten / Dokumente / Zeiterfassung). Vieles ist **schon echt verkabelt** (TanStack Query gegen MSW): Presence-Picker (`stores/presence`), Urlaub/Krankmeldung (`hr-hooks`), Clock-In/Out + Live-Timer + ArbZG. Der Arbeitsvorrat liegt in vier Löchern: **(1) Dokumente-Tab ist leer + Stubs, (2) Profil-Defaults hardcoden „Darien Morales" statt current-user, (3) Avatar/DND nur halb, (4) ein verwaister Toter-Code-Ordner.**

i18n: `profil.*` in `i18n/messages/de.json` (Cluster bei Z. 5449–5749, `profil.zeiterfassung.*` verstreut bis ~9291) + en/fr/it. Projekt nutzt `{var}` single-brace.

## Workflow pro Phase
bauen → i18n ×4 (`{var}`, ICU-Plural) → MSW-/Demo-Daten falls nötig → Compile-Gate (`npm run build > /tmp/build.log 2>&1; echo "EXIT $?"` + `grep -i error`, **nie `| tail`**) → Playwright-Screenshot-QA gegen **:5174** + **Bilder ansehen** → iterieren → commit + push auf `parallel/profil` → Eintrag in `qa-profil.md`.

---

### P-1 — current-user-Single-Source + Account-Info  `[FUNDAMENT, zuerst]`
**Ist:** `stores/settings.ts:124–133` hardcodet die Profil-Defaults als **„Darien Morales"** (Vorname/Nachname/E-Mail/Telefon/Position/Bio). Das widerspricht dem projektweiten current-user = **Stefan Vogel / usr-e1** ([[reference_current_user_source]], `CURRENT_USER` in `mocks/shared-ids.ts`). `ProfilTab.tsx:129` liest die E-Mail bereits korrekt aus `useAuthStore(s=>s.user)`, aber Name/Telefon/Position/Bio kommen aus dem hardcodierten Settings-Store. Zudem: `profil.info.memberSince` (de.json:5522) existiert, aber das UI-Element ist entfernt (`ProfilTab.tsx:354–356`, Kommentar „User hat kein created_at").
**Soll:**
- Profil-Default-Seed in `stores/settings.ts` auf **Stefan Vogel** ausrichten (Name/E-Mail/Position konsistent mit `CURRENT_USER`/Sidebar/Topbar). Wenn möglich Name/Position aus dem current-user-Objekt ableiten statt doppelt zu hardcoden; mindestens aber den Seed = Stefan Vogel.
- **memberSince demo-tief lösen:** eine Demo-`joinedAt`/`memberSince`-Quelle einführen (z.B. Feld am current-user/Profil-Seed, fester Demo-Wert) und die Account-Info-Sektion (`ProfilTab.tsx:342–357`) wieder mit „Mitglied seit …" füllen. (Kein Backend-Call — Demo-Wert genügt.) Den toten Key dadurch wieder nutzen.
**Verify:** Profil-Tab zeigt durchgängig Stefan Vogel (Name/E-Mail/Position = Sidebar/Topbar), „Mitglied seit" sichtbar; EN-Switch sauber; 0 Raw-Keys.

### P-2 — Dokumente-Tab echt (MSW)  `[KERN, größter Punkt]`
**Ist:** `DokumenteTab.tsx` nutzt `useEmployeeDocuments(employeeId)` + `useDocumentCategories` (Z. 27–48), aber es gibt **keinen MSW-Handler** für `GET/POST /api/v1/hr/employees/:id/documents` → Liste bleibt **leer**. Upload (`handleUpload`, Z. 79–82) ist toast-only (`_uploadMutation` importiert, nie aufgerufen). Preview (Z. 233) + Download (Z. 240) sind toast-only.
**Soll (Muster: team `PersonnelDocuments`→MSW aus letztem Batch):**
- In `mocks/handlers/hr.ts` (bereits in `index.ts` registriert → **kein** index.ts-Touch) Handler ergänzen: `GET …/employees/:id/documents` (5–8 realistische Demo-Docs: Arbeitsvertrag, Gehaltsabrechnungen, Zertifikate, Bescheinigungen — mit Kategorie, Dateiname, Größe, Datum), passender Kategorien-Endpoint für `useDocumentCategories`, `POST …/documents` (Upload → neues Doc, **stateful** in der Session), `GET …/documents/:docId/download` (liefert ein Blob).
- `handleUpload` auf `useUploadEmployeeDocument` umstellen (Toast-Stub raus) → neues Dokument erscheint sofort in der Liste.
- Preview → zentriertes `shared/DetailModal` (Doc-Metadaten + Platzhalter-Vorschau), Download → echter Blob-Download (wie team).
- Doc-Liste/Karten ganze Zeile klickbar (`role=button`), wo sinnvoll.
**Verify:** Dokumente-Tab zeigt Demo-Docs; Upload fügt sichtbar hinzu (überlebt im Session-State); Preview öffnet Modal; Download löst Datei aus; 0 Raw-Keys.

### P-3 — Avatar-Upload demo-real (MSW) + DND-Fallback
**Ist:** Avatar-Upload (`ProfilTab.tsx:89–113`) schreibt nur eine Data-URL lokal in `settings.profile.avatarUrl`; Toast/Key `profil.avatar.saved` sagt „(Vorschau — Upload folgt …)". DND-Toggle (`ProfilTab.tsx:254–259`) ist `disabled` wenn `!dndBackendAvailable`.
**Soll:**
- Avatar über einen Demo-MSW-Handler in `hr.ts` führen (`POST …/employees/:id/avatar` → gibt die hochgeladene URL zurück; Mutation aktualisiert Profil/User), so dass es sich demo-real verhält ([[feedback_module_depth_standard]] „Placeholder→MSW"). Den „Upload folgt"-Disclaimer entfernen (Demo zeigt es als erledigt). Lokale Persistenz darf bleiben.
- DND-Toggle: **lokalen Demo-Fallback** ergänzen, damit der Schalter im Demo (ohne echtes Backend) umschaltbar ist + den Zustand sichtbar hält.
**Verify:** Avatar wechseln → erscheint sofort + überlebt Reload; DND lässt sich im Demo an/aus schalten; EN sauber.

### P-4 — Dead-Code-Cleanup (verwaister zeiterfassung-Ordner)
**Ist:** `modules/profil/tabs/zeiterfassung/` enthält **9+ verwaiste Dateien** (`TodayView`, `WeekView`, `MonthView`, `ReportsView`, `TeamView`, `OverviewView`, `CategoriesView`, `ApprovalBanner`, `ExportDialog`, `ManualEntryForm`, `time-utils.ts`). `ZeiterfassungTab.tsx` importiert davon **nichts** — es nutzt `modules/zeiterfassung/components/*` + eigene Inline-Funktionen. Die Ordner-Dateien referenzieren nur sich selbst.
**Soll:**
- Verwaisten Ordner `modules/profil/tabs/zeiterfassung/` löschen. **VORHER absichern:** `grep` über `modules/` ob wirklich kein aktiver Import dorthin zeigt (außer ordner-intern). Wenn ein einzelner Import doch existiert → diesen Punkt stoppen + Darien melden.
- Verwaiste `profil.zeiterfassung.*`-i18n-Keys: **konservativ** purgen — pro Key per `grep` prüfen, ob ein aktiver `t('profil.zeiterfassung.…')`-Aufruf existiert (ZeiterfassungTab + dessen echte Imports). Nur Keys mit **null** aktiver Nutzung in allen 4 Sprachdateien entfernen; im Zweifel **behalten**.
**Verify:** `npm run build` grün, keine Missing-Import-Fehler; Zeiterfassung-Tab funktioniert unverändert (nutzt `modules/zeiterfassung/components`); 0 Raw-Keys.

### P-5 — Demo-Tiefe-Schlusscheck + Profil-Karte
**Ist:** `components/user/UserProfileCard.tsx` (Popover-Overlay, Ping→Chat via `useGetOrCreateDM` + NavigationIntent, Z. 275–348) existiert.
**Soll:**
- **Profil-Karte verifizieren/polieren:** Overlay erreichbar (z.B. von Avatar/Name), „Nachricht senden" öffnet/navigiert zum DM; Status/Presence korrekt; keine toten Buttons.
- **Schlusscheck über alle 4 Tabs:** restliche toast-only/console.log/tote Buttons in profil durchgehen → verkabeln oder ehrlich als „Demo" kennzeichnen. 0 Raw-Keys, 0 `{{var}}`, EN-Switch sauber.
- Detail-Öffnungen (Doc-Preview etc.) = `shared/DetailModal`, sticky Close.
**Verify:** Screenshots aller 4 Tabs @ 1440 + 1024, Profil-Karten-Overlay, 0 Raw-Keys/Doppelklammern/Console-Errors.

---

## Definition of Done (profil review-reif)
Alle 5 Phasen verifiziert (Screenshots **angesehen**), 0 Raw-Keys / 0 Doppelklammern / 0 Console-Errors, jede Phase ein Commit+Push auf `parallel/profil`, `qa-profil.md` gepflegt. Dann Darien: „profil 5/5 fertig". **Out of scope (NICHT bauen):** echtes Avatar-/Dokument-Storage-Backend, echte DND-Backend-Anbindung, Settings-Registry-Eintrag (profil-Settings leben unter `/settings`).
