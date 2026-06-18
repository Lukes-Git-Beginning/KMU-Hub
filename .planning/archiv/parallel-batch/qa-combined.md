# ▶ QA-REVIEW — Batch 2 (team + helpdesk) — MORGEN ZUERST

> **Darien, das ist deine Review-Liste für heute Früh.** Zwei Module sind diese Session review-reif geworden (Main = team, Sub = helpdesk). Beide auf `main` (helpdesk gemergt). Geh die Punkte durch, hak ab, melde Fund/OK pro Punkt. Dev-Server starten, `/team` und `/helpdesk` öffnen.
>
> **Merge-Status:** siehe unten („Merge-Status" — wird beim Merge aktualisiert).

---

## Modul 1 — team (HR)  ·  Main-Terminal, 8 Commits auf `main`

| # | Was prüfen | Klick-Pfad | Erwartet |
|---|---|---|---|
| TM-1 | **Abwesenheitskalender** nicht mehr leer | `/team` → Tab **Abwesenheiten** | Lena (Krankheit), Markus (Homeoffice heute), Sophie (Sonderurlaub) mit Abteilungen; Wochen-Navigation filtert korrekt |
| TM-1b | **Dashboard-Widget** Abwesenheiten | `/` (Übersicht) → Widget „Abwesenheiten" | zeigt heute Abwesende (nicht leer) |
| TM-2 | **Self-Service = echter User** | `/team` → Tab **Self-Service** → *Mein Profil* | **Stefan Vogel** (nicht „Jonas Diaz"), echte Daten (Telefon/Standort/Eintritt), Salden 17/30 + 4/5 |
| TM-2b | **Antrag wirkt** | Self-Service → *Meine Anträge* → „Neuer Antrag" (Typ+Zeitraum+Absenden) | neuer Antrag erscheint sofort, Zähler steigt, Toast; taucht auch in Tab *Anfragen* auf |
| TM-2c | **Gehaltszettel-Download** | Self-Service → *Gehaltsabrechnungen* → PDF-Button | lädt eine echte Datei (kein bloßer Toast) |
| TM-3 | **Personalakte echt** | `/team` → Tab **Personalakte** | echte Kollegen (Felix Krause/Kevin Baumann/Laura Neumann), KPIs 12/9/1/1, Status-Badges (abgelaufen rot) |
| TM-3b | **Vorschau + Download** | Personalakte → Auge-Icon (Vorschau), Download-Icon | Vorschau-Dialog mit Metadaten + „Demo-Vorschau"; Download lädt echte Datei; „Hochladen" nimmt Datei an |
| TM-4 | **OrgChart-Aktionen** | `/team` → Tab **Organigramm** → Person anklicken → E-Mail / Anrufen | öffnet Compose/Call (kein Toast „E-Mail: …") |
| TM-4b | **i18n** (überall) | sprachweit, v.a. Modul-Zuweisung | keine `{{count}}`/`{{user}}`-Rohtexte; Titel „Team" |
| TM-5 | **Deaktivieren wirkt** | `/team` → Tab **Mitglieder** → ⋮-Menü einer Person → „Deaktivieren" → bestätigen | Person ausgegraut + „Inaktiv"-Badge, Toast |
| TM-5b | **Schulungen** | `/team` → Tab **Schulungen** | rendert (Katalog + Teilnahmen); „Schulung anlegen" / „Teilnahme erfassen" wirken (Zustand-Demo) |
| — | **Umlaut** | überall wo „Geschäftsführung" steht | „Geschäftsführung"/„Geschäftsführer" (nicht „ae") |

**team — bewusst NICHT in diesem Batch (zur Info, kein Mangel):** Schulungen-MSW-Swap (Zustand-Store funktioniert + swap-ready, 🔒 Lukes Backend) · P2 Personalakte↔Dok-Verknüpfung tiefer + Organigramm editierbar. Details: `qa-team.md`.

---

## Modul 2 — helpdesk (Tickets)  ·  Sub-Terminal, H-1…H-8

> Demo-tief-Pass: Store hatte **keine** Actions (alle Mutationen waren Toast-Stubs) → jetzt verkabelt. Detail: `qa-helpdesk.md` (vom Sub).

| # | Was prüfen | Klick-Pfad | Erwartet |
|---|---|---|---|
| H-1/3 | **Mutationen wirken** (Kern!) | `/helpdesk` → Ticket anlegen / Status ändern / Antwort senden / CSAT | jede Aktion wirkt sichtbar + überlebt Reload (war vorher Toast ohne Effekt) |
| H-2 | **Ticket-Detail = DetailModal** | Ticket-Zeile anklicken | zentriertes Cosmi-Modal (nicht Slide-over); ganze Zeile klickbar; sticky Close |
| H-4 | **Zuweisen / Eskalieren / Mergen** | im Ticket-Modal die Aktionen | wirken im Store, mit Thread-/Status-Spur |
| H-5 | **Canned Responses CRUD** | Vorlagen-Panel → anlegen/bearbeiten/löschen | Liste aktualisiert sich, überlebt Reload |
| H-6 | **Settings-Panel** | Modul-Einstellungen → helpdesk | personal + tenant (Geschäftszeiten + Routing) editier-/speicherbar; Header-Buttons weg |
| H-7 | **SLA echt + Sortierung** | Ticket-Liste | SLA-Zeiten relativ zu heute (plausibel); Spalten sortierbar (Feld + Richtung) |
| H-8 | **i18n + Demo-Tiefe-Schliff** | sprachweit | keine Rohtexte/`{{}}`, keine toten Buttons (Stand verifizieren — letzter Sub-Punkt) |

**helpdesk — out of scope (Info):** TanStack-Migration, eigener MSW-Handler, CRM-Kontakt-Lookup (= Lukes Backend / späterer Batch).

---

## Merge-Status ✅
- **Gemergt:** `parallel/helpdesk` (`edf3c2e1`, H-1…**H-8 komplett**) → `main`, Merge-Commit **`a221278d`**.
- **i18n:** trotz Sub-Warnung **kein** Git-Konflikt — Auto-Merge kombinierte die Cluster sauber. Verifiziert: alle 4 Sprachen valide, **0 Duplikat-Keys** (de/en 9121, fr/it 9087 Keys).
- **Cross-Lane (Sub):** `dashboard/widgets/OpenTickets.tsx` (legitimer Verbraucher der neuen Ticket-SLA-Form), `module-settings-registry.tsx` (+helpdesk-Eintrag) — kein team-Overlap.
- **Build:** ✅ grün (electron-vite, exit 0) nach Merge — team + helpdesk kompilieren zusammen. Gepusht.
- **Untracked Sub-Helfer** (`vite.qa.config.mjs`, `add-helpdesk-i18n.mjs`) blieben beim Sub-Klon (reisen nicht mit dem Branch) — kein Handlungsbedarf.

## Nach dem Review
Pro Modul: Funde sammeln → Fix-Runde (wie F1-F7 beim letzten Batch) → dann beide → **Nico**. Nächstes Paar danach: Master-Tracker Review-Pipeline (#6 helpdesk ist dann durch → z.B. automatisierung-Tiefe-Pass / profil).
