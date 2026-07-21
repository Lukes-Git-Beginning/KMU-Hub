# R-6 Recherche-Gate — Ergebnis (2026-07-21, Session #24)

> Markt-Recherche (2 Web-Agents: Odoo/Entra/Google · Salesforce/Zoho/Personio) + Explore-Ist-Analyse. Entscheidungsvorlage für die gebündelten Fragen in §3. Bau-Schnitt in §4.

## §1 Markt-Kern

### Der wichtigste Befund: NIEMAND kann echten per-User-Entzug — Cosmi baut ein Alleinstellungsmerkmal
- **Odoo** (der berüchtigte Fall): rein additiv über Gruppen. Kritik-Katalog = unsere Anti-Muster-Liste: implizite Gruppen (`implied_ids`) nur im Developer-Mode sichtbar · **Orphaned Permissions** (Elterngruppe entfernt → Kind-Gruppen bleiben, GitHub #24720) · keine Provenance pro Recht · kein Audit nativ · Einzel-Ausnahme = Einzel-User-Gruppe anlegen (Explosion).
- **Salesforce** (Industrie-Standard): Profile + Permission Sets (additiv) + **Muting Permission Sets** = die einzige echte Deny-Mechanik am Markt — aber NUR innerhalb einer Permission Set Group (max. 1 Muting-Set/Gruppe, wirkt nicht gegen andere Quellen!). Dependency-Kaskaden beim Muten (Read muten → Create/Edit/Delete mit-gemutet) mit Bestätigungs-Dialog. Herkunfts-Anzeige nur über Zusatz-App (UAPA) — kein Konflikt-Highlighting, kein Badge in der User-Liste. Profil-Wechsel lässt Permission Sets stumm bestehen → Zombie-Rechte.
- **Entra ID:** additiv; einziges Plus: „Assignment Path"-Spalte (direct vs. via Gruppe) — hinter P1-Lizenz. **Google Workspace:** additiv; bestes Provenance-Panel (Privileg → aus welcher Rolle), aber read-only. **Zoho:** 1 User = 1 Profil, Ausnahme = Profil klonen (Explosion). **Personio:** nur Rollen-Stacking; Einzel-Ausnahme = Mini-Rolle mit Employee-Filter.

### Übertragbare Muster (bauen)
1. **Provenance pro Recht** (Google-Muster, bei uns via `sources` schon da) — jede Zeile zeigt „aus Rolle X" vs. „persönlich".
2. **Deny sichtbar machen statt weglassen** (Salesforce-Anti-Lücke): „Von Rolle erlaubt, persönlich entzogen" = durchgestrichen + Herkunft — der Admin darf nie rätseln.
3. **„Angepasst"-Badge in der User-Liste** — hat KEIN Marktprodukt nativ; echter Differenziator gegen stille Sonderrechte.
4. **Kein stilles Behalten bei Rollen-Wechsel** (Salesforce-Zombie-Lektion) und **kein stilles Verwerfen** (Datenverlust) → Bestätigungs-Dialog mit Abweichungs-Liste.
5. **Klarer Reset-Fluss** pro Schalter UND global („Auf Rollen-Stand zurücksetzen") — fehlt überall am Markt.
6. **Audit pro Override-Änderung Pflicht** (Odoo-Totalausfall als Warnung) — bei uns via R-5-Infrastruktur trivial (`permission.override_set/removed` reserviert).

## §2 Ist-Analyse (Kern, mit Pfaden)

- **RoleEditorPage.tsx (888 Z.) ist bereits sauber zerlegt:** `ModulePane` (Z. 541–694) + `CapabilityRow` (Z. 701–755, `readonly`-Prop = exakt die gewünschte Ausgegraut-Optik) + `StagedChangesList` — dateilokal, nur Export nötig. Abweichungs-Dot via `deviationDiff` gegen `baseGrants` (Z. 128), „Auf Vorlage zurücksetzen" `resetModuleToBase()` (Z. 201), Eskalations-Block `escalationKeys` (Z. 139, blockt Commit), Selbst-Aussperr-Warnung (Z. 146). Draft-State + `commit()` → PUT permissions (Z. 216–232).
- **Resolution hat EINE Andock-Stelle:** `resolveCapabilities(roleIds)` → `Record<string, CapabilityGrant{scope, sources[]}>` (mocks/data/rbac.ts Z. 539, SCOPE_ORDER widest-wins). Konsumenten: `effectivePermissionsBody()` (handlers/rbac.ts Z. 69) + `fallbackPermissions()` (stores/permissions.ts Z. 64). Neue pure Funktion `applyUserOverrides(resolved, overrides)` NACH resolveCapabilities — beide Aufrufer, fertig.
- **Provenance-Infra trägt 'override' schon fast:** `sources: string[]` — Sentinel `'override'` + Sonderfall-Chip in `EffectivePermissionsView` (Z. 125–145). Für Deny-Darstellung: `CapabilityGrant` um `effect?: 'deny'` erweitern ODER `deniedByOverride: string[]` — Entscheidung beim Bauen (Variante a = einfacher fürs FE).
- **Einstiegs-Flächen:** `UserRolesSection` (Chips + „Effektive Rechte"-Button Z. 144) in UserDetailModal (Z. 121) + MemberProfileContent (Z. 419) · Benutzerliste `RoleChipCell` (UsersAdminHubTab Z. 253) für das Badge.
- **Preview trägt es ohne Umbau:** `startPreview({label, capabilities})` — User-Preview = `applyUserOverrides(resolveCapabilities(rolesForUser(id)), USER_OVERRIDES[id])`.
- **MSW-Muster fertig:** `USER_ROLE_ASSIGNMENTS`-Analogie → `USER_OVERRIDES`-Map + GET/PUT/DELETE `/admin/users/:userId/overrides`; Audit-Interceptoren nach R-5-Muster (writeAuditEvent, Keys reserviert).

## §3 Entscheidungsvorlage (gebündelte Fragen — Antworten in §0 des R6-Briefings festschreiben)

**Ohne Frage gesetzt (technisch zwingend fürs entschiedene UI-Bild, Einspruch möglich):**
- **Override gewinnt IMMER pro Key** über die gesamte Rollen-Union (egal wie viele Rollen) — sonst ergäbe die Toggle-Ansicht keinen eindeutigen Zustand. Nicht überschriebene Keys folgen LIVE den Rollen.
- **Deny-Darstellung:** durchgestrichen + „persönlich entzogen"-Herkunft in Effektive-Rechte (nie stilles Weglassen) — Markt-Lücke, §1.2.
- **Guardrails aus §3 des Briefings** (Eskalation, Selbst-Aussperrung, Last-Admin, deny auf admin:* für letzten Vollzugriff, Audit pro Änderung) = Pflicht, keine Frage.

**Frage ① Rollen-Wechsel-Verhalten:** Was passiert mit Overrides, wenn sich die Rollen eines Users ändern? (Empfehlung: BEHALTEN + Bestätigungs-Dialog mit Abweichungs-Liste beim Rollen-Wechsel — nie stumm wie Salesforce, nie Datenverlust.)
**Frage ② Ebene-1-Sichtbarkeit:** Dürfen Overrides auch Module ein-/ausblenden oder nur Ebene 2/3? (Empfehlung: JA, volle Ebenen — Referenzfall „Aushilfe + Projekt-Schreiben" braucht ggf. Modul-Sicht; Editor kann es schon; weniger Sonderfälle.)
**Frage ③ Wer darf Overrides setzen:** eigener Katalog-Key `admin:user_override:manage` (admin-only Default) vs. am bestehenden `admin:role:assign` mitfahren. (Empfehlung: eigener Key — hr_admin hat role:assign, soll aber nicht zwingend Rechte feinjustieren.)
**Frage ④ Übersicht/Transparenz:** Reicht „Angepasst"-Badge + Filter „Nur angepasste" in der Benutzerliste, oder zusätzlich eigener Report? (Empfehlung: Badge + Filter, kein eigener Report in 1.0.)

## §4 Bau-Schnitt (nach Darien-OK, ~2 Runs)

**Run 1 — Fundament + Editor:** Typen (`OverrideEffect`, `CapabilityOverride`, `UserOverrides`, CRUD-Inputs) in rbac-types · `USER_OVERRIDES`-Map + `applyUserOverrides()` (pure) in mocks/data/rbac.ts · CRUD-Handler + Audit-Interceptoren (`permission.override_set/removed`, old/new = Override-Delta) · `effectivePermissionsBody()` + `fallbackPermissions()` anwenden · `ModulePane`/`CapabilityRow` exportieren (oder `components/shared/rbac/CapabilityEditor.tsx`) · Override-Editor im User-Kontext (Route/Modal an UserDetailModal + Team-Profil): Rollen-Union als ausgegrauter Basis-Stand, „Benutzerdefiniert"-Schalter aktiviert, jeder Toggle-Flip = allow/deny-Abweichung mit Dot, Reset pro Zeile + global.
**Run 2 — Darstellung + Guardrails + QA:** Effektive-Rechte: 'override'-Chip + Durchgestrichen-Deny · „Angepasst"-Badge (UserRolesSection + RoleChipCell) + Listen-Filter · Rollen-Wechsel-Dialog (laut Entscheid ①) · Eskalations-/Selbst-Aussperr-/Last-Admin-/admin:*-deny-Guardrails · Preview „Als dieser User" · i18n ×4 (zentral, Du-Form, echte fr/it — R-5-Lektion!) · gescopter tsc + eslint · QA-Script (Pflicht-Szenarien aus Briefing §5: Aushilfen-Referenzfall · deny-Fall · Reset · Eskalations-Block · Badge) + Bilder ansehen.

**Agent-Lektionen aus R-5 (für die Briefs):** Agents schreiben KEINE i18n-json direkt (zentrales Script) · Du-Form · echte Umlaute · „Anbieter-Zugang"-Klasse von Übersetzungsfehlern einkalkulieren · Verify-Block Pflicht.
