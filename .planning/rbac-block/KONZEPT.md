# RBAC-Block — Berechtigungs-Baukasten (Konzept + Recherche)

> **Auftrag (Darien, 2026-07-16):** Nächster großer Block nach dem Branchen-Block. Firmen sollen Rollen + Berechtigungen **selbst erstellen** können (Custom Roles). Zweiteilung: **HR** verwaltet das Fachlich-Organisatorische (Mitarbeiter anlegen, Profile, Zuordnungen), **IT** bekommt den technischen **Baukasten bis ins kleinste Detail** (Rollen erstellen, z.B. „Rolle ohne Projekt-Modul", „nur lesen, nichts bearbeiten"). **Zentria richtet beim 1-Woche-Onsite-Onboarding alles initial ein**, die Kunden-IT übernimmt danach.
> **Status:** Recherche abgeschlossen (3 Markt-Reports 2026-07-16), Produktfragen mit Darien in Klärung. Baustart erst nach Besprechen-Gate.
> ⚠ Diese Entscheidung **ersetzt** die alte Festlegung „5 feste Rollen, keine Custom Roles in 1.0" (A-2, `mocks/data/admin-permissions.ts`).

---

## 1. Ist-Stand Cosmi (Code-verifiziert 2026-07-16)

### Backend (Luke) — deutlich weiter als gedacht
- **Voll normalisiertes RBAC-Schema** (Migration 000002): `roles` / `permissions` (resource × action) / `role_permissions` / `user_roles`. Nichts ist als Enum verdrahtet → Custom Roles sind DB-seitig möglich.
- **Echtes Enforcement:** praktisch jede Modul-Route prüft `middleware.RequirePermission("<resource>", "<action>")` (z.B. `produktion:bom`+`write`); 30+ Seed-Migrationen decken alle Module ab. `RequireRole` für Auth/Admin-Routen.
- **User-Verwaltung existiert:** `HandleListUsers/UpdateUser/AssignRole/RemoveRole` + **Invitations-Endpoints** (`POST/GET/DELETE /invitations`) in `route_auth.go`.
- **Lücken:**
  1. `roles` hat **kein `tenant_id`** → Rollen sind global, nicht pro Firma (Blocker für Custom Roles).
  2. Rollennamen **hart verdrahtet** im Validator (`oneof=admin manager member`); FE kennt 5 Rollen (admin/manager/member/hr/it_support), BE seedet 3–4 → Drift.
  3. **Kein Rollen-CRUD / keine Matrix-API** (Rollen anlegen, Permissions einer Rolle lesen/schreiben).
  4. **Kein Daten-Scope** im Permission-Modell (eigene/Team/alle) — nur resource×action.
  5. Permission-Änderungs-Propagation (Caching?) ungeklärt.

### Frontend
- `config/roles.ts`: 5 statische Rollen, nur grobes Nav-/Registry-Gating (`roles:`-Filter an Settings-Einträgen).
- **A-1 Benutzerverwaltung** (Admin-Hub): Liste/Invite/Rolle/Deaktivieren — mock-first, Contract `AdminUser`.
- **A-2 Rechte-Matrix** (Admin-Hub): 7 Gruppen × 17 Capabilities × 5 Rollen, editierbar, mock-first (`mocks/data/admin-permissions.ts`). **Wird von nichts enforced.**
- **Kein `useCapability`-Hook, null Aktions-Gates in den Modulen** — jeder `member` kann heute Projekte erstellen, Aufgaben löschen, Wiki-Artikel anlegen. Das ist DIE Lücke.
- **Modul-Leiter-System existiert und wirkt:** `tenant_module_leads` (Migration 138) + `useIsModuleLead`, gatet tenant-Settings-Schreibrechte; Zuweisung am Mitarbeiterprofil (Team → MemberDetailPanel).

---

## 2. Markt-Erkenntnisse (3 Reports, 2026-07-16)

### Report A — HR-Systeme (Personio/BambooHR/HiBob/Factorial)
- **Personio-Modell (DACH-Referenz):** Rechte = **Datenkategorie × Zugriffsebene (keine/ansehen/bearbeiten) × Scope (eigene Daten / Reporting-Line / Custom-Filter / alle)**. Auto-Rollenzuweisung per Bedingung (Abteilung=X → Rolle Y) seit 2026.
- **BambooHR:** Custom Access Levels feldgenau, additiv, Default = „kein Zugriff" (sicherer Default). HiBob: Two-Axis (People-Scope × Data-Scope), aber nur Kategorie-, nicht Feld-Ebene.
- **Drei Marktlücken = Cosmi-Chancen:**
  1. **HR-Admin ≠ IT-Admin kann KEIN System sauber** (Personio: explizit „not on the roadmap") — „Technischer Admin ohne Gehalts-/HR-Datenzugriff" wäre ein echtes Verkaufsargument.
  2. **Befristete Rechte/Vertretung nirgends nativ** (Steuerberater-/Prüfer-Zugang, Urlaubsvertretung).
  3. **Gehalts-Schutz nur auf Kategorie-Ebene**, nicht Feld-Ebene.
- HR-Must-haves: Vorgesetzten-Scope (Reporting-Line), Self-Service eigene Daten (+optionaler Genehmigungsschritt), Gehaltsdaten separat schützbar, Audit-Log, Rollen-Zuweisung bei Eintritt regelbasiert, Offboarding=Deaktivierung.

### Report B — ERP/SaaS Custom-Roles (Odoo/weclapp/Xentral/monday/Asana/Notion/Zoho)
- **Bestes Grundmodell: Zoho-Split** — **Profil = was darf ich TUN** (Modul + Aktionen) getrennt von **Rolle/Scope = was darf ich SEHEN** (eigene/Team/alle, hierarchisch). Löst das KMU-Kernproblem („Außendienst: nur eigene Kontakte, aber voll editieren · Buchhaltung: alles sehen, nur lesen").
- **weclapp-Basisrollen** gegen Wildwuchs: Custom-Profil hat `based_on`-Pointer auf System-Preset, UI zeigt „X Abweichungen vom Standard".
- **Granularität 1.0:** Modul an/aus · 5 Aktions-Bits (lesen/erstellen/bearbeiten/löschen/exportieren) · Daten-Scope (eigene/Team/alle) · Objekt-Freigaben (privates Projekt/Dokument für Person/Team freigeben, nur additiv). **Später:** Feld-Level, Geschäftsaktions-Level (Rechnung freigeben), befristete Rechte.
- **Editor-UI:** Zwei-Pane (Modul-Baum links, Aktions-Matrix + Scope-Dropdown rechts), `based_on`-Badge, **Rollen-Vergleich** nebeneinander, **„Als Rolle anzeigen"-Preview** (kaum ein Wettbewerber hat das — Killer-Feature).
- **Fallen:** Rollen-Explosion (Limit ~20 Custom-Profile + Duplikat-Warnung) · Default-Deny für neue Module · **Single-Profil pro User** statt Multi-Rollen-Union · Presets nie löschen/editieren (nur klonen) · `manage_roles` ≠ `assign_roles` trennen.

### Report C — IT-Admin/Enterprise-Patterns (Entra/Google/NIST/GDAP)
- **Admin-Dreiteilung** (Entra-Muster): Rollen-/Sicherheits-Verwaltung (IT) ≠ User-Verwaltung (HR) ≠ Helpdesk. Scoped Admins können nie höhere Rollen anfassen.
- **Rollen-Anzahl KMU 5–200 MA:** 7–12 gut entworfene Rollen; max. 2 Vererbungsebenen; 80/20-Test.
- **Guardrails P0 (ohne die nicht launchen):** Mindestens-1-Admin-Schutz · Selbst-Aussperr-Schutz · Privilege-Escalation-Guard (niemand gibt sich selbst höhere Rollen) · Least-Privilege-Default für neue User · unveränderliches Audit-Log für Rechteänderungen · scoped Admin kann höhere Rollen nicht editieren. **P1:** Break-Glass-Account/Tenant + Alert · Vier-Augen für Admin-Zuweisung · „View as user". **P2:** befristete Rechte, Access-Reviews.
- **Zentria-Onsite-Zugang nach GDAP-Muster** (Microsoft-Partner-Modell): „Zentria Setup Admin" mit **Ablaufdatum** (z.B. 14 Tage nach Onboarding), Kunde sieht jederzeit ob/wann Zentria zugriff, kann Zugang per Klick entziehen, jede Aktion im Audit-Log. **DSGVO-Verkaufsargument, hat kein DACH-KMU-ERP.**
- **Baukasten =** Branchen-Rollen-Templates (unveränderliche Originale, klonbar) + Setup-Wizard (Branche → Template → Anpassen → Zuweisen) + Config-Export/Import (JSON, Zentria-intern wiederverwendbar von Kunde zu Kunde).

---

## 3. Ziel-Architektur Cosmi (Vorschlag)

### Capability-Modell: 3 Ebenen (Dariens Detaillierungs-Vorgabe, 2026-07-16)

> **Referenz-Fall (Darien):** Profil „Aushilfe" — kann Aufgaben **zugewiesen bekommen** und Projekte **lesen**, aber nichts eintragen/ändern; Dokumente ansehen, aber Download steuerbar; „Info im Projekt eintragen" (kommentieren) einzeln schaltbar. Diese Tiefe gilt für **alle Module inkl. Einstellungen- und Sicherheits-Bereich** — die IT stellt ganz Cosmi ein: was sieht man, welche Funktionen hat man.

1. **Ebene 1 — Sichtbarkeit:** Modul in Nav sichtbar ja/nein + sensible **Bereiche innerhalb** (Einstellungen-Overlay, Admin-Hub, Sicherheit, Gehaltsdaten in Team) einzeln sichtbar/unsichtbar.
2. **Ebene 2 — Basis-Aktionen (einheitlich in jedem Modul):** `lesen / erstellen / bearbeiten / löschen / exportieren` + **Daten-Scope** `eigene / Team / alle`. Deckt ~80 % konsistent ab.
3. **Ebene 3 — Fein-Capabilities (kuratiert pro Modul, je ~3–8):** modulspezifische Einzelschalter für genau die Aktionen, die im Alltag getrennt vergeben werden. Beispiele:
   - **work:** `aufgaben zugewiesen bekommen + eigene abhaken` (ohne Bearbeiten-Recht!) · `kommentieren/Info eintragen` · `Projekt erstellen` · `fremde Aufgaben bearbeiten` (getrennt von eigenen)
   - **dokumente:** `herunterladen` (getrennt von ansehen) · `hochladen` · `teilen/freigeben`
   - **finanzen:** `Rechnung versenden` · `Mahnung auslösen` · `Beträge sehen`
   - **wiki:** `Artikel erstellen` · `fremde Artikel bearbeiten`
   - **team:** `Gehaltsdaten sehen` (Kategorie-Schutz) · `Mitarbeiter anlegen/deaktivieren` · `Rollen zuweisen`
   - **settings/security:** `Modul-Einstellungen (tenant) ändern` · `Sicherheitsbereich sehen` · `Integrationen verwalten` · `Rollen erstellen`

   Damit ist das Aushilfen-Profil exakt abbildbar: work → lesen ✓, zugewiesen-bekommen ✓, kommentieren ✓/✗, bearbeiten ✗ · dokumente → lesen ✓, herunterladen ✗ · alles andere → Modul unsichtbar.

**Warum kuratiert statt „alles frei":** Lukes BE-Modell (`resource × action`) trägt beliebig feine Capabilities schon heute (es gibt bereits `produktion:bom`≠`produktion:workstep`, `rapporte:approve`). Die Kunst ist der **kuratierte Katalog** — pro Modul die 3–8 Schalter, die KMUs wirklich brauchen (Personio/weclapp-Lehre), statt 500 technischer Checkboxen (Entra-Falle). Der Katalog wird pro Modul aus der realen UI abgeleitet (welche Buttons/Aktionen existieren) und ist erweiterbar, ohne das Modell zu ändern.

### Berechtigungs-Modell: zwei Achsen + ein Zusatz
1. **Permission-Profil** (was darf ich tun) — die 3 Ebenen oben, gespeichert als `permissions(resource, action)`. Passt exakt auf Lukes Schema — die Seeds existieren schon.
2. **Daten-Scope** (was darf ich sehen) — pro Modul(-Bereich): `eigene / Team (Reporting-Line) / alle`. **Neu im BE-Modell.**
3. **Objekt-Grants** (Zusatz, Phase 2): privates Projekt/Dokument für Person/Team freigeben — nur additiv fürs Sehen, nie über die Profil-Grenze hinaus.

### Rollen-Modell (Darien-Entscheidung 2026-07-16)
- `roles` + `tenant_id NULL` = **System-Presets** (unveränderlich, Zentria-gepflegt) · `tenant_id` gesetzt = **Custom-Rollen der Firma**.
- Custom-Rolle = Klon eines Presets (`based_on`), Abweichungen sichtbar. Limit ~20/Tenant + Duplikat-Warnung.
- **1 Person = 1 Account/Userprofil, aber ein Account kann MEHRERE Rollen tragen** (z.B. Vertriebsleiter + Lagerleiter). Rechte addieren sich (Union). Lukes `user_roles` (n:m) trägt das bereits.
  - **Pflicht-Gegengewicht zur Union-Falle:** „**Effektive Rechte**"-Ansicht pro User (aufgelöste Summe aller Rollen, mit Herkunfts-Badge „aus Rolle X") + Overlap-Hinweis beim Zuweisen + Audit. Ohne das wird Multi-Rolle unauditierbar (Markt-Lehre).
- Modul-Leiter bleibt als orthogonale Zusatz-Eigenschaft (existiert, bewährt).
- Presets 1.0: Vollzugriff (Admin) · IT-Admin (technisch, **ohne HR-/Gehaltsdaten** — Marktlücke!) · HR/People-Admin · Teamleiter/Manager · Mitarbeiter · Nur-Lesen · **Aushilfe/Extern** (Referenz-Fall §3). **Dazu bis 1.0: 3 Branchen-Template-Sets** (Handwerk / Dienstleister / Handel) fürs Onsite-Playbook.

### Admin-Trennung (Dariens HR/IT-Bild, marktbestätigt)
| Wer | Darf | Darf nicht |
|---|---|---|
| **IT-Admin** | Rollen erstellen/bearbeiten (`manage_roles`), System-Settings, Integrationen, Audit einsehen | HR-Datenkategorien (Gehalt etc.) sehen |
| **HR/People-Admin** | Mitarbeiter anlegen/einladen/deaktivieren, **Rollen zuweisen** (`assign_roles`), Profile/Zuordnungen, HR-Daten nach Kategorie-Rechten | Rollen erstellen/ändern, System-Settings |
| **Zentria (Onsite)** | Alles während Setup-Fenster (GDAP-artig, Ablaufdatum, kundensichtbar, entziehbar) | dauerhaften Zugriff behalten |

### HR-Seite (Team-Modul, Personio-inspiriert)
- Mitarbeiter anlegen → Einladung → **Rolle bei Eintritt** (regelbasiert nach Abteilung optional später); Offboarding = Deaktivieren (Seat frei, Login gesperrt).
- HR-**Datenkategorien** (Persönlich / Job / Gehalt / Dokumente / Abwesenheiten) × (keine/ansehen/bearbeiten) × Scope (eigene/Reporting-Line/alle) — 1.0 mindestens: Gehalt/Verträge als geschützte Kategorie, Vorgesetzten-Scope, Self-Service eigene Daten.

### Guardrails 1.0 (P0-Set aus Report C)
Mindestens-1-Admin · Selbst-Aussperr-Schutz · Privilege-Escalation-Guard · Least-Privilege-Default · Audit-Log für Rechteänderungen (immutable) · Default-Deny neue Module · `manage_roles` ≠ `assign_roles`.

### Editor-UI (IT-Baukasten im Admin-Hub)
Zwei-Pane-Editor (Modul-Baum ↔ Aktions-Matrix + Scope) · based_on-Badge „X Abweichungen" · Rollen-Vergleich · **„Als Rolle anzeigen"-Preview** (Shell simuliert die Rolle) · Bulk-Toggles („alle Branchen-Module aus") · Plain-Language-Zusammenfassung („kann Rechnungen sehen, aber nicht erstellen").

---

## 4. Phasenplan (final, 2026-07-16 — Bau in NEUEN Terminals nach dem 10-Phasen-Batch-Muster; dieses Terminal hat nur geplant)

**FE (mock-first, bewährtes Muster; Contract so schneiden, dass Lukes BE ihn 1:1 übernehmen kann):**
- **R-0 Katalog-Finalisierung (Planung, teils erledigt):** `CAPABILITY-KATALOG.md` — Fein-Capabilities pro Modul kuratieren. Kern-Module hier vor-kuratiert; restliche Module werden je R-3-Batch vor dem Gating kuratiert (Modul-UI sichten → Schalter ableiten → in Katalog eintragen → dann gaten).
- **R-1 Fundament:** Contract `Role`/`RolePermissions`/`EffectivePermissions` (tenant-scoped, based_on, Multi-Rollen-Union) · `useCapability`-Hook (liest effektive Rechte, MSW `GET /me/permissions`) · `config/roles.ts` → dynamische Rollen aus API/MSW · Default-Deny-Mechanik für neue Module.
- **R-2 Rollen-Baukasten (Admin-Hub):** Rollen-Liste (Presets + Custom + Branchen-Sets) · Zwei-Pane-Editor (Modul-Baum × Ebene-1/2/3-Schalter + Scope) · Klonen/based_on-Abweichungs-Badge · Rollen-Vergleich · **„Als Rolle anzeigen"-Preview** · **„Effektive Rechte"-Ansicht pro User** (Multi-Rollen-Auflösung mit Herkunft) · Guardrails-UI (letzte-Admin-Sperre, Selbst-Aussperr-Warnung, Eskalations-Guard) · löst die A-2-Matrix ab.
- **R-3 Enforcement-Sweep (größter Teil, batch-weise à ~5 Module):** pro Modul: Katalog kuratieren (falls Rest-Modul) → alle Aktions-Buttons/Dialoge/Exporte/Sichtbarkeiten an `useCapability` gaten → Screenshot-QA mit 2+ Rollen (z.B. Admin vs. Aushilfe). Start mit den wichtigsten: work, dokumente, crm/kontakte, finanzen, team, wiki, settings/security — dann alle übrigen. **Bis 1.0: alle 32 Module.**
- **R-4 HR-Seite (Team-Modul):** Anlegen→Einladen→Rollen-Zuweisung-Flow konsolidiert (HR darf zuweisen, nicht erstellen) · Datenkategorien-Rechte (Gehalt/Verträge geschützt — auch vor IT-Admin!) · Vorgesetzten-Scope (Reporting-Line) · Offboarding/Deaktivierung.
- **R-5 Audit + Zentria-Zugang + Branchen-Sets:** Rechteänderungs-Log (UI) · „View as" ausgebaut · Zentria-Setup-Zugang (Ablauf + Sichtbarkeit, GDAP-light) · 3 Branchen-Template-Sets finalisieren (mit Darien je Set durchgehen).

**BE-Paket für Luke (🔴, parallel ab R-1 — in backend-gaps.md §RBAC eingetragen):**
`roles.tenant_id` + `based_on` · Rollen-CRUD + Rollen-Permissions-API (`GET/POST/PATCH/DELETE /admin/roles`, `PUT /admin/roles/:id/permissions`) · Validator-Entkopplung (Rollennamen dynamisch statt `oneof=admin manager member`) · Daten-Scope-Dimension im Permission-Modell (eigene/Team/alle) · Guardrails serverseitig (last-admin, self-lockout, escalation-guard, Default-Deny) · Audit-Events für Rechteänderungen · **`GET /me/permissions`** (aufgelöste effektive Rechte — die eine Quelle fürs FE-Gating) · Permission-Cache-Invalidierung bei Rollen-Änderung · Invite-Flow-Rest (Mail).

---

## 5. Produktentscheidungen (Darien, 2026-07-16 — Besprechen-Gate durchlaufen)

1. **Granularität:** volle Fein-Capability-Tiefe (Ebene 3) über ALLE Module inkl. Settings/Security — „so genau wie's geht". **Staffelung:** Plan/Katalog für alle 32 Module JETZT (dieses Terminal = nur Planung, kein Bau); der Bau startet mit den wichtigsten Modulen, aber **bis Release 1.0 werden alle Module fein kuratiert** — nichts bleibt dauerhaft auf Basis-Bits.
2. **Profil-Modell:** 1 Person = 1 Account; ein Account kann **mehrere Rollen** tragen (Vertriebsleiter + Lagerleiter). Union additiv + Pflicht-Feature „Effektive Rechte"-Ansicht (§3 Rollen-Modell).
3. **Templates:** Standard-Presets + **alle 3 Branchen-Sets bis 1.0** (Handwerk/Dienstleister/Handel).
4. **Zentria-Setup-Zugang:** GDAP-artig, Ausgestaltung 1.0-light (Ablaufdatum + Audit-Sichtbarkeit), Dashboard-Ausbau später — keine Einwände.
5. **Reihenfolge:** RBAC-Block **komplett zuerst**; Onboarding/Info-Center O-0 erst **nach dem Review** (damit es so genau wie möglich auf dem finalen Stand aufsetzt).

---

*Recherche-Reports (Roh-Synthesen) archiviert im Chat-Verlauf Session #13; Quellen in den Reports. Master-Tracker-Eintrag folgt nach Besprechen-Gate.*
