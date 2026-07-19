# Hetzner-Review-Checkliste (Cosmi-exe, app.zentria.tech)

> **Zweck:** Darien prüft Änderungen hands-on auf der über Hetzner laufenden Cosmi-exe, parallel während Main + Sub weiterbauen.
> **Ablauf:** Alles geht auf `main`. Hier sammeln sich konkrete „klick hier → erwarte das"-Items. Abgehakt = von Darien geprüft + ok.
> **⚡ Darien-Entscheid 2026-07-19: SAMMEL-REVIEW AM ENDE** — es gibt keine Zwischen-Reviews mehr; diese Liste sammelt ALLE Review-Items (RBAC-Batches, Modul-Reviews, Rest) und wird nach Abschluss der Bau-Strecke in einem Rutsch abgearbeitet.
> **Stand:** 2026-07-19.

## ✅ Voraussetzungen geklärt (Darien, 2026-06-24)
1. **Deploy:** macht **Luke** (irgendwann). Push auf `main` ≠ sofort live → Änderungen erscheinen auf Hetzner erst nach Lukes Deploy.
2. **Mode: LIVE** (echtes Prod-Backend, kein MSW).
   - **Folge A:** Backend-Echt-Schaltungen (dialer/dashboard/HR) sind nach Lukes Deploy prüfbar — zeigen **echte Prod-Daten** (ohne Prod-Seed ggf. leer, nicht kaputt).
   - **Folge B (wichtig):** **FE-mock-Module zeigen im Live-Mode NICHT ihre MSW-Demodaten** — sie treffen das echte Backend. Module mit fehlendem Backend (🔒, z. B. **security/DSGVO**) sind auf Hetzner-Live **erst prüfbar, wenn Luke das Backend gebaut hat**. Bis dahin: reine FE/UX-Reviews dieser Module besser lokal.

---

## A · Jetzt prüfbar (FE/UX, Demo-Mode) — review-reife Module
> Die 15 FE-fertigen Module sind „review-reif". Pro Modul achten auf: tote Buttons, leere Zustände, Raw-i18n-Keys, `{{var}}`, Umlaut-Fehler, Detail-Modals (ganze Zeile klickbar, sticky Close), Sortierung.
- [ ] kontakte · [ ] calendar · [ ] dokumente · [ ] finanzen/Buchhaltung · [ ] work · [ ] team
- [ ] dashboard · [ ] vertraege · [ ] helpdesk · [ ] automatisierung · [ ] profil · [ ] mails · [ ] kommunikation · [ ] berichte · [ ] wiki

## B · Bereit zum Review (gemergt auf main)
- [ ] **RBAC R-3 Batch 1 — Aktions-Gating work/dokumente/kontakte/finanzen/wiki** (Session #17). **Demo-Mode prüfen** (Rechte kommen aus MSW; Live hat den permissions-Endpoint noch nicht → FE-Fallback löst Presets auf). Schnellster Weg: Verwaltung → Rollen → Rolle öffnen → „Als Rolle anzeigen"; echte Sessions über den ProfileSwitcher unten rechts. Achten auf:
  - Als **Elena (Nur Lesen)**: „Nur Ansicht"-Chip im Modul-Header · KEINE Erstellen/Bearbeiten/Löschen-Controls · finanzen-Tabs reduziert (kein Export/Mahnwesen/Ausgaben/Banking) · Zeilen-Menüs nur Details/PDF, **Versenden ausgegraut mit Hover-Hinweis** · dokumente-Kontextmenü: **Herunterladen ausgegraut mit Hinweis**, Edit-Einträge weg · kontakte: **Import-Button ausgegraut mit Hinweis**, kein vCard-Export im Menü.
  - Als **Max (Aushilfe/Extern)**: „Eingeschränkt"-Chip in Aufgaben · eigene zugewiesene Tasks abhakbar + kommentierbar, sonst nichts · Deep-Link `/#/finanzen` oder `/#/kontakte` → **„Kein Zugriff"-Seite** (nicht leer/Redirect).
  - Beträge-Maskierung `•••`: Rolle „Lager & Logistik" im Editor öffnen → Buchhaltung sichtbar + Rechnungen-Ansehen AN (amounts AUS lassen) → „Als Rolle anzeigen" → Buchhaltung: alle €-Werte als Punkte, Anzahl/Prozent normal.
  - Als **admin**: ALLES unverändert da (Regression).
  - ⚠ Bekannter Vorbestand (NICHT dieser Batch): Rechnungen-Liste zeigt 0,00 € je Zeile (Listen-API ohne Positionen) — separater Fix folgt.
- [ ] **RBAC R-3 Batch 2 — team-Aktionen · dashboard Ebene-2 · Verwaltungs-Tabs** (Session #18). **Demo-Mode prüfen**, Werkzeuge wie Batch 1 (Editor-Preview + ProfileSwitcher). Achten auf:
  - Als **Max (Extern)**: Dashboard zeigt NUR Projekt/Aufgaben/Dokumente-Karten, QuickActions nur „Neues Projekt/Dokument", keine Alert-Banner, kein Umsatz-Chart-Widget; Schnellaktionen-Widget ohne CRM-Buttons.
  - Als **Sarah (Teamleiter)**: Team ohne „Mitarbeiter erstellen" + ohne Lohnvorbereitung-Tab, Anfragen-Tab MIT Genehmigen; Karten-Menü ohne „Deaktivieren"; Dashboard ohne Buchhaltungs-Karte.
  - Als **Elena (Nur Lesen)**: fremdes Mitglieder-Profil = nur Übersicht-Tab mit Hinweis „Weitere Details sind für deine Rolle nicht sichtbar" (kein leeres weißes Modal, kein Dokumente-Tab).
  - Als **Nina (HR-Admin)**: Verwaltung zeigt NUR Benutzer+Rollen (kein Lizenz/Branding/IT/Sicherheit/Abrechnung/Integrationen); Direkt-URL `/#/admin/license` springt auf Benutzer.
  - Als **Thomas (IT-Admin)**: Verwaltung ohne Lizenz/Branding/Abrechnung; Sicherheit-Tab da, dort KEIN DSGVO/Auskunft-Sub-Tab (kein gdpr:execute); landet auf Audit-Log.
  - Als **admin**: alle 8 Verwaltungs-Tabs + alle 10 Security-Sub-Tabs (Start = Audit-Log), Team-Payroll „Prüfen & freigeben" da (Regression).
  - ⚠ Bekannte Nebenbefunde (NICHT dieser Batch): Benachrichtigungs-Widget zeigt extern CRM-Demo-Events (Seeds empfänger-agnostisch, → Luke-Paket) · Dashboard-Moduleinstellungen: 4 Widgets ohne Namens-Label (Vorbestand DashboardSettingsPanel).
- [ ] **RBAC R-3 Batch 3 — Aktions-Gating inventar/einkauf/produktion/vertraege/helpdesk** (Session #19). **Demo-Mode prüfen**, Werkzeuge wie Batch 1 (Editor-Preview + ProfileSwitcher unten rechts). Achten auf:
  - Als **Elena (Nur Lesen)**: „Nur Ansicht"-Chip in allen 5 Modulen · inventar ohne „Artikel hinzufügen"/CSV-Export/„Neue Inventur starten" · einkauf: Draft-Bestellung öffnen → **„An Lieferant senden" sichtbar aber ausgegraut mit Hover-Hinweis** (einzige disabled-Ausnahme), Bearbeiten/PDF/Stornieren weg · produktion ohne „Neuer Auftrag", Maschinen-Detail: Status als Text statt Dropdown · verträge ohne „Vertrag anlegen"/Zeilen-Menü-Mutationen.
  - Als **Markus (Mitarbeiter)**: helpdesk = **Requester-Modell**: nur EIGENE Tickets (5 Stück: 3 als Melder, 2 als Bearbeiter), „Neues Ticket" DA, Vorlagen-Button + Statistik-Tab WEG, kein Agent-Aktions-Block im Ticket-Detail · produktion: kein „Neuer Auftrag", aber **CSV/Laufkarte-Export bleibt** (Werker-Fall) + Schritt-Abhaken im Auftrags-Detail geht · inventar: Bestandsbewegung erfassen geht (Zeilen-Menü), aber kein Artikel-Anlegen/Export, im Bewegungs-Dialog KEINE „Korrektur"-Option · einkauf: Bestellung als Entwurf anlegen geht, Senden ausgegraut.
  - Als **Max (Extern)**: Deep-Links `/#/helpdesk`, `/#/produktion`, `/#/inventar` → „Kein Zugriff"-Seite.
  - Als **admin**: alle Create-Buttons/Tabs/Exporte unverändert (Regression); helpdesk zeigt ALLE Tickets, „Zugewiesen an" jetzt mit echten Namen statt `usr-…`-Ids (Adapter-Fix).
  - ⚠ Bekannte Nebenbefunde: helpdesk-Zuweisen-Dropdown führt weiterhin die zwei Freitext-Agenten „Marco Hartmann/Sandra Bürki" (Vorbestand, → Luke-Paket assignee-Referenzen) · KB-Artikel-Ansicht: `savedBody`-Crash-Vorbestand beim gespeicherten Artikel wurde als Drive-by gefixt.
- [ ] **security / DSGVO** ✅ gemergt (`43fecf37`, S-1…S-5) — **Demo-Mode prüfbar** (reine FE/MSW-Arbeit). Hub `/admin/security` (10 Sub-Tabs). Achten auf: alle Seiten crashfrei, keine Raw-Keys (DE+EN), DSGVO-Flows durchklickbar — Audit (Filter/Export), DSAR Art.15 (Cross-Modul-Suche + Export), Export Art.15/20 (Genehmigen/Download + Frist), Erasure Art.17 (Preview/Execute + Legal-Hold-Hinweis), Retention (DACH-Fristen + Auto-Löschung-Toggle), Sessions (beenden), Vault, PW-Policy, IP-Access, 2FA. Sub-Bericht: `.planning/parallel-batch/qa-security.md`.
- [ ] **zeiterfassung** (Main, echt-geschaltet) — siehe C.

## C · Backend-Echt-Schaltung (lokal verifiziert — Hetzner erst nach Deploy + Prod-Seed)
> Diese sind gegen das **lokale** Backend + lokale Demo-Seeds live verifiziert (Screenshots lokal). Auf Hetzner brauchen sie (a) Deploy der Backend-Fixes, (b) Prod-Demo-Daten.
- [x] **dialer-Supervisor** — lokal verifiziert (2 BE-Bugs gefixt: recent-calls-SQL + protojson-Null). Hetzner: nach Deploy + Dialer-Seed.
- [x] **dashboard-Layout** — Persistenz-Roundtrip lokal verifiziert (war schon verkabelt).
- [x] **zeiterfassung/HR** — BE-Bug gefixt (NULL `correction_reason` brach die Einträge-Liste). Hetzner: nach Deploy + HR-Seed.
- **Backend-Fixes, die deployt werden sollten (Luke):** `9dfcf89e` dialer recent-calls · `b7242926` hr correction_reason. Beide brechen sonst echte Daten still.

## D · Erledigt-Bestätigung (von Darien)
_(hier hakt Darien ab, was er auf Hetzner geprüft + ok befunden hat)_
