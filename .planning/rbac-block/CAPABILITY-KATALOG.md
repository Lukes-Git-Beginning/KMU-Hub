# Capability-Katalog — Fein-Berechtigungen pro Modul (Entwurf R-0)

> Arbeitsdokument zum RBAC-Block (`KONZEPT.md` §3). **Status: Entwurf** — Kern-Module hier vor-kuratiert (aus Code-Kenntnis), finale Kuratierung je R-3-Batch **gegen die echte Modul-UI** (jeden sichtbaren Button/Export/Dialog einer Capability zuordnen). Ziel bis 1.0: **alle 32 Module fein** (Darien 2026-07-16).
>
> **Referenz-Fall „Aushilfe":** work lesen ✓ + zugewiesen-bekommen ✓ + kommentieren ✓, bearbeiten/erstellen ✗ · dokumente lesen ✓, herunterladen ✗ · übrige Module unsichtbar.

## Konventionen

- **Schlüssel-Format:** `<modul>:<gegenstand>` × Aktion — kompatibel zu Lukes BE (`permissions(resource, action)`, Seeds existieren; z.B. `produktion:bom`×`write`, `rapporte:approve`).
- **Ebene 1 — Sichtbarkeit:** `<modul>:module`×`view` (Nav + Route). Sensible Bereiche einzeln: z.B. `team:salary`×`view`, `settings:tenant`×`view`.
- **Ebene 2 — Basis-Aktionen (jedes Modul):** `read / create / edit / delete / export` + **Scope** `own / team / all` (Scope = eigene Dimension am Grant, nicht eigene Capability).
- **Ebene 3 — Fein-Schalter:** nur Aktionen, die KMUs real getrennt vergeben (3–8 pro Modul). Kein technischer Vollausbau (Entra-Falle).
- **Default-Deny:** neue Module/Capabilities sind für alle Nicht-Admin-Rollen AUS.
- Jede Capability bekommt im FE einen i18n-Klartext („kann Rechnungen sehen, aber nicht erstellen").

## Kern-Module (vor-kuratiert, R-3-Batch 1)

### work (Projekte & Aufgaben)
| Capability | Bedeutung |
|---|---|
| `work:module`×view | Modul sichtbar |
| `work:task`×read/create/edit/delete + Scope | Aufgaben-Basis (edit mit Scope `own` = nur eigene) |
| `work:task`×be_assigned | kann Aufgaben **zugewiesen bekommen** + Status der eigenen Aufgabe weiterschalten — ohne allgemeines Edit (Aushilfen-Fall) |
| `work:task`×comment | Kommentar/Info eintragen |
| `work:project`×read/create/edit/delete | Projekte (create = **nur Projektleiter/Manager** — Dariens Kernbeispiel) |
| `work:project`×manage_members | Mitglieder/Status-Sets/Vorlagen verwalten |
| `work:time`×log | Zeit auf Aufgaben buchen |
| `work:board`×export | Board/Listen-Exporte |

### dokumente
| Capability | Bedeutung |
|---|---|
| `documents:module`×view · `documents:file`×read | sehen/öffnen (Viewer) |
| `documents:file`×download | **Herunterladen getrennt von Ansehen** (Dariens Beispiel) |
| `documents:file`×upload/create · edit (rename/move) · delete | Ablage-Basis |
| `documents:share`×manage | teilen/freigeben (intern) |
| `documents:share_link`×create | externe Links (sensibler!) |
| `documents:version`×restore | Versionen wiederherstellen |
| `documents:template`×manage | Vorlagen verwalten |

### kontakte/crm
`crm:module`×view · `crm:contact`×read/create/edit/delete + Scope (own/team/all — Außendienst-Fall) · `crm:contact`×export (DSGVO-sensibel, eigener Schalter) · `crm:deal`×read/create/edit + Scope · `crm:pipeline`×manage · `crm:import`×run · `crm:advisory`×read/write (Beratungsprotokolle = geschützte Kategorie) · `crm:segment`×override

### finanzen/buchhaltung
`finance:module`×view · `finance:invoice`×read/create/edit/delete · `finance:invoice`×send (versenden ≠ erstellen) · `finance:dunning`×run (Mahnung auslösen) · `finance:quote`×read/create/send · `finance:amounts`×view (**Beträge/Umsätze sehen** — getrennt von Belegzugriff, z.B. für Assistenz) · `finance:export`×run (DATEV/CSV) · `finance:incoming`×review/book · `finance:settings`×manage

### team (HR) — inkl. Datenkategorien (Personio-Muster)
`team:module`×view · `team:employee`×read/create/edit/deactivate + Scope (own/reporting_line/all) · **Datenkategorien je ×view/×edit:** `team:data_personal` · `team:data_job` · `team:salary` (**auch vor IT-Admin schützbar** — Marktlücke) · `team:documents` · `team:absence`×read/approve · `team:role`×assign (**Rollen zuweisen — HR**, ≠ erstellen) · `team:training`×manage · `team:payroll`×view/run

### wiki
`wiki:module`×view · `wiki:article`×read/create · `wiki:article`×edit + Scope (own/all — „fremde Artikel bearbeiten" getrennt) · ×delete · `wiki:article`×publish (freigeben ≠ entwerfen) · `wiki:share_token`×create · `wiki:template`×manage

### settings + security + admin (IT-Baukasten)
`settings:personal`×manage (jeder) · `settings:tenant`×manage (Modul-Leiter/IT — existiert als Mechanik) · `admin:module`×view · `admin:user`×read/invite/deactivate · `admin:role`×read/**create/edit/delete** (`manage_roles` = **nur IT**) · `admin:role`×assign (`assign_roles` = HR+IT) · `admin:license`×manage · `admin:branding`×manage · `admin:integrations`×manage · `security:module`×view · `security:audit`×read · `security:policy`×manage · `security:gdpr`×execute (Export/Löschung — höchste Stufe) · `admin:impersonate`×run („View as", auditiert)

## Übrige Module (Schema steht, Fein-Kuratierung im jeweiligen R-3-Batch)

| Modul | Ebene-3-Kandidaten (beim Batch gegen UI verifizieren) |
|---|---|
| kalender | Termine für andere anlegen · Buchungsseiten verwalten · Kalender freigeben |
| mail | Senden als Team-Postfach · Vorlagen verwalten · Regeln verwalten |
| kommunikation | Kanäle erstellen · externe Kanäle verbinden · Nachrichten löschen (Moderation) · Team-Postfach zuweisen/claim |
| video/meetings | Meeting planen · Aufzeichnung starten/ansehen · Recording-Policy |
| helpdesk | Tickets zuweisen · schließen · Queues verwalten · KB-Artikel pflegen |
| zeiterfassung | fremde Zeiten sehen/bearbeiten (Scope) · Woche freigeben/genehmigen · Export |
| berichte | Bericht erstellen · freigeben/release · planen (Scheduling) · teilen/extern |
| automatisierung | Workflows sehen · erstellen/aktivieren (mächtig — läuft unter Creator-Rechten!) |
| formulare | Formular erstellen · veröffentlichen · Submissions sehen/exportieren |
| dialer | Kampagne verwalten · Agent-Workspace · Outcomes exportieren · Supervisor-Sicht |
| notifications | (fast nur personal — tenant-Policy manage) |
| dashboard | Layout tenant-weit setzen |
| vertraege | Vertrag anlegen · kündigen/Status · Dokument hochladen · Beträge sehen |
| inventar | Bewegung buchen · Inventur starten/zählen/abschließen · Artikel verwalten · Export |
| einkauf | Bestellung anlegen · **freigeben/senden** (Threshold existiert) · Wareneingang buchen · Lieferant verwalten · Katalog/Warenkorb · Abruf anlegen |
| produktion | Auftrag anlegen · **Status wechseln (starten/abschließen/stornieren)** · Schritt abhaken · QS erfassen · Stückliste verwalten · Maschine verwalten · Laufkarte/CSV |
| vermietung | Reservierung anlegen · Ausgabe/Rücknahme · Objekt verwalten · Protokoll |
| rapporte | Rapport erstellen · **einreichen ≠ freigeben** (`rapporte:approve` existiert im BE!) · PDF-Export |
| schichten | Plan bearbeiten/veröffentlichen · Tausch beantragen ≠ genehmigen · Vorlagen |
| fuhrpark | Fahrt eintragen · Fahrzeug verwalten · Fahrtenbuch-Export |
| profil/mein-bereich | (personal only — kein Gating nötig außer Sichtbarkeit einzelner Widgets) |

## Arbeitsanweisung je R-3-Batch (pro Modul)
1. Modul-UI sichten: jeden Button/Dialog/Export listen → Capability zuordnen (bestehende BE-Seeds als Basis: `backend/migrations/*seed*permissions*`).
2. Katalog-Tabelle hier finalisieren (3–8 Fein-Schalter; Rest fällt unter Basis-Aktionen).
3. Gating einbauen (`useCapability`), leere/gesperrte Zustände sauber (Button versteckt vs. disabled+Tooltip — Konvention: **sicherheitsrelevant = versteckt**, workflow-relevant = disabled mit Begründung).
4. Screenshot-QA mit mind. 2 Rollen (Admin + Aushilfe/Nur-Lesen) — Bilder ansehen.
5. BE-Seed-Abgleich: fehlende resource×action-Paare als Migration-Wunsch in backend-gaps §RBAC nachtragen.
