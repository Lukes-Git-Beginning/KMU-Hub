# R-3 Batch 1 — Recherche-Gate-Ergebnis + entschiedene Gating-Konvention

> Erstellt Session #16 (2026-07-19). Recherche-Gate für Batch 1 (work/documents/crm/finance/wiki) ist DURCHLAUFEN:
> 3 parallele Web-Agents, 11 Produkte. Die 4 gebündelten Fragen wurden Darien gestellt — **alle 4 Empfehlungen bestätigt (2026-07-19)**.
> **Das Bau-Terminal kann Batch 1 OHNE weitere Fragen bauen** (Ablauf-Schritte 1–3 aus R3-BRIEFING erledigt).

## 1. Die 4 Darien-Entscheide (verbindlich für R-3, alle Batches)

1. **Gating-Konvention = Markt-Hybrid, Default VERSTECKEN.** Alles, was die Rolle permanent nicht darf, erscheint nicht (hidden) — auch Erstellen/Bearbeiten (ersetzt die alte Briefing-Konvention „workflow-relevant = disabled+Tooltip"). **disabled+Tooltip nur als bewusste Ausnahme** für seltene, delegierbare, erklärungsbedürftige Aktionen. Ausnahmen-Liste Batch 1:
   - `crm:import:run` (HubSpot-Muster: Button faded + „Recht fehlt — an Verwaltung wenden")
   - `finance:invoice:send` / `finance:quote:send` (Erstellen ohne Versand-Recht: Versenden-Button disabled + Tooltip, Zoho-Books-Freigabe-Logik light)
   - `documents:file:download` (Dropbox-Muster: Button grayed + Tooltip „Download deaktiviert")
   - Drittes Pattern wo passend: **Wert sichtbar, Edit gesperrt** (Felder wie Beträge — nicht Button, sondern read-only-Feld mit Hover-Hinweis).
2. **Read-only-Kommunikation = dezenter Header-Chip.** Shared Badge („Nur Ansicht" / „Eingeschränkt") im Modul-Header, nur sichtbar wenn die Rolle im Modul unter Vollzugriff liegt (Asana-„Comment only"-/Google-„Viewing"-Muster). Einmal shared bauen, in allen Modulen nutzen. Verhindert das Confluence-Problem (Buttons fehlen kommentarlos).
3. **Scope „eigene" = Edit-Controls still verstecken.** Fremde Objekte bleiben lesbar (read-Recht), Edit/Delete-Controls erscheinen dort schlicht nicht; Detail-Fenster read-only (HubSpot-Muster). KEINE disabled-Buttons mit Besitzer-Tooltip, KEIN Listen-Filtern.
4. **Deep-Link ohne Recht = leichte „Kein Zugriff"-Seite.** Shared NoAccess-View (Modul-Name, kurzer Satz, „Wende dich an deine Verwaltung") statt leerer Seite/stillem Redirect — an den Modul-Routen verdrahtet (Google-„You need access"-Muster; bei fast allen Wettbewerbern eine Lücke).

## 2. Markt-Synthese (11 Produkte, 3 Reports)

### Konsens über alle Produkte

| Befund | Marktlage |
|---|---|
| Löschen/Export/Import/Admin ohne Recht | **Alle 11 Produkte: versteckt.** Kein Produkt zeigt einen ausgegrauten Delete-Button |
| Erstellen/Bearbeiten ohne Recht | Ebenfalls fast überall **versteckt** (Asana, monday, ClickUp, Confluence, Notion, Zoho, Pipedrive, Drive) |
| disabled+Tooltip | Nur als Ausnahme: seltene, delegierbare Aktionen (HubSpot-Import „Not authorized", Dropbox-Download „Downloads disabled") |
| Drittes Pattern | **Wert sichtbar, aber read-only** (monday Column-Permissions: Wert da, nicht klickbar; HubSpot Property-Hover „no permission to edit"; „Can't view" = durchgestrichenes Auge-Icon statt Wert) |
| Read-only-Anzeige | Dezentes **Badge/Icon im Header** (Asana „Comment only"-Badge, Google Docs Auge-Icon „Viewing"), NIE roter Banner. Confluence-Schwäche: gar keine Erklärung |
| Deep-Link ohne Recht | Google „You need access"-Seite + Request-Access = Gold-Standard; SharePoint seit 2026 nachgezogen; Rest schwach |
| Scope „eigene" | HubSpot: fremder Datensatz per Direktlink lesbar, Edit blockiert mit Meldung; Pipedrive: Listen gefiltert bzw. „Item hidden"-Platzhalter bei Verknüpfungen |
| UX-Faustregel (Smashing/SIDP) | **Permanent fehlendes Recht → hidden; temporär/erklärungsbedürftig → disabled+Tooltip** |

### Domänen-Besonderheiten

- **work (Asana/monday/ClickUp):** Kommentar-only-Rollen etabliert (Asana Commenter darf auf SELBST zugewiesenen Aufgaben Due-Date/Felder ändern = scope-own-Vorbild). Drag&Drop für Viewer deaktiviert. Viewer-Board = keine Action-Bars, kein „+".
- **documents (Drive/SharePoint/Dropbox):** Download-Sperre: Google/SharePoint verstecken, Dropbox disabled+Tooltip (→ unser Entscheid: Dropbox-Muster). Upload/Neu für Viewer konsequent weg — SharePoints berüchtigter Bug (New sichtbar, Fehler erst beim Speichern) = Anti-Pattern. Kontextmenü-Einträge (Delete/Rename/Move) überall hidden.
- **crm (Zoho/Pipedrive/HubSpot):** Export/Import ohne Recht = Link komplett weg (DSGVO-Begründung). HubSpot Property-Permissions = Best-in-Class Feld-Gating (Auge-Icon). Reports-Recht weg ⇒ Nav-Punkt weg.
- **finance (lexoffice/sevdesk/Zoho Books):** DACH-Muster = Navigations-Ebene verstecken („Haken weg ⇒ Menüpunkt weg"), Beträge in EIGENEN Belegen bleiben sichtbar — versteckt wird die Auswertungs-Ebene (GuV/EÜR/Reports), nicht die Transaktions-Ebene. Zoho Books: Versenden ≠ Erstellen via Freigabe-Workflow (einziges Produkt). Steuerberater = eigener Zugangstyp mit vollem Daten-Lesezugriff ohne Admin (= unser `readonly`-Preset/Elena ✓).
- **wiki (Confluence/Notion):** Edit-Button/Toolbar für Viewer komplett absent (kein grayed-out). Notion „Request access" via Share-Menü (seit 2024). Confluence rote Lock-Icons für Restrictions. Kein Produkt hat sauberes publish≠edit auf Rollen-Ebene — unser `wiki:article:publish`-Fein-Schalter ist Differenzierung.

## 3. Umsetzungs-Regeln fürs Bau-Terminal (abgeleitet, verbindlich)

1. **Shared zuerst bauen:** (a) `RestrictedModeBadge` (Header-Chip, aus useCapabilitySet abgeleitet: Modul sichtbar, aber keine create/edit/delete-Grants ⇒ „Nur Ansicht"; teilweise ⇒ optional „Eingeschränkt"), (b) `NoAccessView` (Kein-Zugriff-Seite) + Routen-Guard an den Modul-Routen (`<modul>:module:view`), (c) i18n ×4 für beide.
2. **Hidden per Konvention:** Kein `disabled` ohne expliziten Ausnahme-Grund. Leere Action-Bars vermeiden — wenn eine Toolbar durch Gating leer würde, ganz weglassen (Reduced-State, kein Gerippe).
3. **Kein Flackern:** `useCapability` ist loading=deny — bei Listen `useCapabilitySet().ready` abwarten (Stolperstein aus R3-BRIEFING §4 gilt weiter).
4. **Scope own:** Vergleich gegen `CURRENT_USER.id` bzw. assignee/owner-Feld; Objekte ohne Owner-Feld → Owner nachseeden oder als Gap notieren, NICHT stillschweigend `all` (Briefing §1.4).
5. **QA:** pro Modul mind. 2 Rollen (admin + extern/readonly) via Editor-Preview „Als Rolle anzeigen" + ProfileSwitcher; Bilder ansehen. Ausnahme-Buttons (disabled+Tooltip) explizit screenshotten.
6. **BE-Seed-Abgleich** pro Modul gegen `backend/migrations/*seed*permissions*`, Lücken in backend-gaps §RBAC.

## 4. Quellen (Auswahl)

- Asana Comment/View-Only: asana.com/inside-asana/comment-view-only-permissions-asana · forum.asana.com/t/684050
- monday Column/Board-Permissions: support.monday.com/hc/en-us/articles/360011926640 · 115005315809
- ClickUp Permissions: help.clickup.com/hc/en-us/articles/6309221065495
- Confluence Page Restrictions/Read-only: confluence.atlassian.com/doc/page-restrictions-139414.html
- Notion Sharing & Permissions: notion.com/help/sharing-and-permissions · Release 2.39 (Request Access)
- Google Drive shared-drive Rollen/Download-Sperre: support.google.com/a/users/answer/12380484 · support.google.com/drive/answer/14254362
- SharePoint Access-Denied/IRM: learn.microsoft.com/en-us/troubleshoot/sharepoint/... · MC1188599 (moderne Request-Access-Seite)
- Dropbox Link-Permissions: help.dropbox.com/share/set-link-permissions
- Zoho CRM Profile/Sharing/FLS: help.zoho.com/portal/en/kb/crm/security-control/...
- Pipedrive Visibility Groups/Permission Sets: support.pipedrive.com/en/article/visibility-groups · permission-sets
- HubSpot Permissions/Property-Restrictions: knowledge.hubspot.com/user-management/hubspot-user-permissions-guide · properties/restrict-view-edit-access-for-properties
- lexoffice Benutzer/Berechtigungen: help.lexware.de/de-form/articles/8315057 · sevdesk: hilfe.sevdesk.de/de/articles/9382074 · Zoho Books Users/Roles + Transaction Approval: zoho.com/us/books/help/settings/users.html
- Hidden vs. Disabled (UX-Grundlage): smashingmagazine.com/2024/05/hidden-vs-disabled-ux/
