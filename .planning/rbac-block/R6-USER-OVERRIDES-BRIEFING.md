# R-6 Per-User-Overrides — Terminal-Briefing (erstellt Session #20-Nachgang, 2026-07-19)

> **Status: IN UMSETZUNG (Session #24, 2026-07-21).** Recherche-Gate durchlaufen → `R6-RECHERCHE.md` (Markt-Kern, Ist-Analyse mit Pfaden, Bau-Schnitt). Gebündelte Fragen gestellt, ALLE 4 Empfehlungen bestätigt.

## §0 Darien-Entscheide (2026-07-21, verbindlich)

1. **Rollen-Wechsel:** Overrides werden BEHALTEN + Bestätigungs-Dialog beim Rollen-Wechsel (Liste der bestehenden Abweichungen, aktive Bestätigung) — nie stumm behalten (Salesforce-Zombie), nie stumm verwerfen.
2. **Ebene 1 JA:** Overrides dürfen auch Modul-Sichtbarkeit schalten (voller Editor-Umfang, keine Ebenen-Ausnahme).
3. **Berechtigung:** eigener Katalog-Key `admin:user_override:manage` (fine, admin-only per Default; hr_admin behält role:assign OHNE Feinjustieren).
4. **Transparenz:** „Angepasst"-Badge in der Benutzerliste + Filter „Nur angepasste Benutzer". KEIN eigener Report in 1.0.

**Ohne Frage gesetzt (technisch zwingend, mit Darien geteilt):** Override gewinnt IMMER pro Key über die gesamte Rollen-Union (auch Scope-Absenkung) · nicht überschriebene Keys folgen LIVE den Rollen · Deny-Darstellung = durchgestrichen + „persönlich entzogen" (nie stilles Weglassen) · Guardrails aus §3 = Pflicht.

> **Ablauf zwingend: 1) Recherche-Gate → 2) gebündelte Fragen an Darien → 3) Darien-OK → 4) bauen → 5) Gates.** [1–3 erledigt, §0 oben.]

## 1 · Was Darien entschieden hat (verbindlich)

**Feature:** Rechte pro einzelnem User feinjustieren, ohne für Einzelfälle neue Rollen anzulegen. Referenzfall: „Alle Aushilfen dürfen nichts in Projekte schreiben — außer dieser einen."

**Variante: B = VOLL** — Overrides können Rechte **hinzufügen UND entziehen** (nicht die additive Sparvariante). Bewusst gegen meine Additiv-Empfehlung entschieden; die Komplexitätsfolgen (Deny-Semantik in Resolution/Preview/Effektive-Rechte) sind Teil des Pakets.

**UI-Bild (Dariens Beschreibung, 1:1 umsetzen):**
- Am User gibt es die **gleiche Toggle-Ansicht wie beim Rollen-Erstellen** (Rollen-Editor-Muster: Modul-Baum links, Subject-Cards mit Schaltern rechts).
- Solange die Rolle(n) führen, sind die Toggles **ausgegraut und zeigen den geerbten Stand** — „wie bei den voreingestellten Profilen" (= Preset-Rollen im Editor sind read-only mit Klon-CTA; exakt diese Optik).
- Ein Button **„Benutzerdefiniert"** schaltet die Ansicht editierbar → jeder umgelegte Schalter wird eine persönliche Abweichung, die NUR diesen User betrifft.

## 2 · Recherche-Gate (VOR den Fragen)

**Auftrag:** Wie lösen Produkte per-User-Overrides ÜBER einem Rollenmodell — Verhalten + Optik:
- **Odoo** (Access Rights pro User + Gruppen — der bekannteste Voll-Override-Fall; auch die bekannten Kritikpunkte einsammeln: Warum gilt Odoo-Rechteverwaltung als unübersichtlich, was müssen wir besser machen?)
- **Microsoft Entra/M365** (Rollen + direkte Zuweisungen; wie zeigen sie „woher kommt dieses Recht?")
- **Google Workspace** (Admin-Rollen + individuelle Privilegien)
- **Zoho One/CRM** (Profile + per-User-Ausnahmen?) · **Personio** (individuelle Zugriffsrechte pro Mitarbeiter?)
- Fokusfragen: ① Wie wird ein Override-Konflikt angezeigt (Rolle sagt JA, Override sagt NEIN)? ② Reset-Fluss („zurück auf Rollen-Stand" — pro Schalter und global)? ③ Wie sieht die User-Liste aus (Badge „angepasst"?)? ④ Was passiert beim Rollen-Wechsel eines Users mit bestehenden Overrides? ⑤ Audit-Darstellung.
- Ergebnis → `R6-RECHERCHE.md` + Entscheidungsvorlage für die gebündelten Fragen.

**Erwartbare Darien-Fragen (im Gate vorbereiten):** Override-Vorrang vs. Multi-Rollen-Union (Vorschlag: Override gewinnt IMMER pro Key, egal wie viele Rollen) · Rollen-Wechsel-Verhalten (Overrides behalten? verwerfen? fragen?) · dürfen Overrides auch Modul-Sichtbarkeit (Ebene 1) schalten oder nur Ebene 2/3? · wer darf Overrides setzen (`admin:role:assign` oder eigener Key `admin:user_override:manage`?) · Limit/Übersicht (Report „User mit Abweichungen").

## 3 · Architektur-Vorgaben (Basis steht — NICHT neu erfinden)

**Semantik (Vorschlag, im Gate bestätigen):**
```
effective(user) = resolveCapabilities(rollen)        // bestehende Union, unverändert
                  → apply userOverrides:              // NEUE Schicht obendrauf
                      key → { mode: 'allow', scope } | { mode: 'deny' }
```
- `allow` setzt/erweitert (auch Scope-Anhebung own→all), `deny` entfernt den Key komplett. Override gewinnt pro Key über ALLE Rollen.
- Nicht überschriebene Keys folgen weiterhin LIVE den Rollen (der Kernvorteil gegenüber dem Klon-Weg — kein Drift).
- `sources` bekommt die neue Herkunft `'override'` → Effektive-Rechte-Ansicht zeigt „persönlich angepasst" (Provenance-Infra existiert seit R-1, Multi-Rollen-Badges nutzen sie schon).

**Wiederverwendung statt Neubau:**
| Baustein | Wiederverwendung |
|---|---|
| `RoleEditorPage` (Two-Pane, Baum, Subject-Cards, Scope-Selects, Draft-Leiste, Diff-Popover) | Editor-UI im User-Kontext („based_on" = live aufgelöste Rollen-Union statt Preset); read-only-Modus = exakt die Preset-Optik |
| Abweichungs-Dot + „Auf Vorlage zurücksetzen" (RoleEditorPage) | wird „Abweichung von der Rolle" + „Auf Rollen-Stand zurücksetzen" pro Zeile |
| `resolveCapabilities` + `SCOPE_ORDER` (mocks/data/rbac.ts) | bleibt; Override-Anwendung als zweiter Schritt DANACH |
| Eskalations-BLOCK + Selbst-Aussperr-Warnung (Editor-Confirm) | gilt 1:1 auch für Overrides (niemand vergibt per Override Rechte über die eigenen hinaus) |
| `startPreview` („Als Rolle anzeigen") | „Als dieser User anzeigen" — Preview mit Overrides |
| `UserRolesSection` im Team-Profil (§Rollen & Zugriff) | bekommt den „Benutzerdefiniert"-Einstieg + „Angepasst"-Badge |
| MSW `handlers/rbac.ts` + `USER_ROLE_ASSIGNMENTS`-Muster | neue mutierbare `USER_OVERRIDES`-Map + CRUD-Endpoints, stateful |

**Guardrails (Pflicht):** Eskalations-Guard · Selbst-Aussperrung (eigener Account) · Last-Admin bleibt unberührbar · deny auf `admin:*` für den letzten Vollzugriff blocken · Audit-Event pro Override-Änderung (wer→wem→Key→alt/neu) · Benutzerliste zeigt „Angepasst"-Badge (Transparenz gegen stille Sonderrechte).

**Deny-Komplexität ehrlich einplanen (der Preis von Variante B):** Effektive-Rechte muss „Rolle sagt ja, persönlich entzogen" verständlich darstellen (durchgestrichen + Herkunft, nicht einfach weglassen — sonst rätselt jeder Admin) · Preview/QA-Matrix wächst (Rolle × Override-Kombis) · BE-Resolution zieht nach (Luke).

## 4 · Luke-Kontrakt (backend-gaps §RBAC, R-6-Absatz)

`user_permission_overrides` (tenant_id, user_id, permission_key, mode allow|deny, scope, created_by, created_at, updated_at) · Resolution serverseitig: Rollen-Union → Overrides anwenden (deny entfernt, allow setzt/erweitert) · `GET /admin/users/{id}/permissions` liefert Overrides in `sources` mit · CRUD `PUT/DELETE /admin/users/{id}/overrides` · Audit-Events · Guardrails serverseitig spiegeln. **Design-Vormerkung steht in backend-gaps, damit Luke die Erweiterungsstelle beim Grant-Modell von Anfang an mitdenkt.**

## 5 · Aufwand + Gates

**Schätzung: ~2 Runs** (Run 1: Resolution-Schicht + Editor-im-User-Kontext + MSW stateful · Run 2: Effektive-Rechte/deny-Darstellung, Badges, Guardrails, Preview, QA-Matrix). Standard-Gates wie alle R-Pakete: i18n ×4 · gescopter tsc (`tsconfig.rbaccheck.json` erweitern) · `eslint src/ --quiet` · Playwright-QA (Muster `qa-rbac-enforcement-b4.mjs`; Pflicht-Szenarien: Aushilfen-Referenzfall [extern + persönlich `work:project:edit`] · deny-Fall [member − `rapporte:report:create`] · Reset auf Rollen-Stand · Eskalations-Block · Badge in Benutzerliste) + Bilder ansehen · 1 feat- + 1 docs-Commit + Push.
