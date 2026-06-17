# Kombinierte QA-Liste — 10 Phasen (dashboard + vertraege)

> **Beide Module review-reif, alle Commits auf `main`.** Reihenfolge: **Pattern-Entscheidung zuerst** (falls die nicht passt, betrifft sie mehrere Punkte), dann pro Modul. Details + alle Screenshots: `qa-dashboard.md` / `qa-vertraege.md`.
> Stück für Stück durchgehen; Dev-Server läuft auf **:5173** (frisch gestartet).

---

## 🔴 ZUERST — Pattern-Entscheidung

**1. [PATTERN] vertraege: Detail öffnet als zentriertes Modal (V-1)**
- `/#/vertraege`, Tab „Aktiv" → Zeile klicken (z. B. „Büro-Mietvertrag München").
- Erwartung: **zentriertes** DetailModal (nicht mehr Slide-over rechts). Beim Scrollen bleiben Header (Close) + Footer (Bearbeiten/Unterschrift/Kündigen) stehen. Ganze Zeile klickbar; Drei-Punkte/Bearbeiten öffnet NICHT das Detail. Tab fokussiert Zeile → Enter öffnet.
- Worauf achten: Subtitle „Mietvertrag · MV-2024-001", Status-Badge, alle 9+ Sektionen da, kein abgeschnittener Header.
- **Wenn dir das Modal-Pattern hier zusagt, passt es für alle vertraege-Punkte.**

---

## 🟠 vertraege (V-2…V-5)

**2. PDF-Vorschau echt (V-2)** — Vertrag mit Dokument öffnen → Sektion „Dokumente" → Dateiname klicken → **PDF rendert im iframe** (4 Verträge haben echte PDFs: Büro-Mietvertrag, Thomas Berger Arbeitsvertrag, Helvetia SLA, Allianz). *Headed/echte App nötig — headless hat keinen PDF-Viewer.*

**3. Fristen-Reminder (V-3)** — Bell/`/notifications` → „Vertrag läuft bald ab — Microsoft 365 … in 18 Tagen" (+ Müller 47 T, Lagerraum 82 T), Subtitle „N ungelesene Benachrichtigungen" (kein Raw-Key mehr). `/vertraege` öffnen → kurzer Toast. Tab **„Auslaufend" → 3 Verträge**. Mehrmals neu laden → keine Duplikate.

**4. E-Signatur Demo-Rücklauf (V-4)** — Vertrag mit Unterzeichnern (z. B. „Müller Metallbau Rahmenvertrag") → Footer „Unterschrift". „Zur Unterschrift senden" → Signer „Gesendet" + **Demo-Hinweis** (kein echter Mailversand), Dialog bleibt offen. „Rücklauf simulieren" → Angesehen → Unterschrieben, Audit-Log füllt sich. Skribble bleibt „Bald verfügbar".

**5. Nummernkreis + Audit-User + Template (V-5)** — „Vertrag anlegen" → Nummer **vorbefüllt** (V-2026-001), anlegen → erneut öffnen → **002**. Audit-Log zeigt **„Markus Weber"** (kein „Aktueller Benutzer"). Tab „Vorlagen" → „Vertrag aus Vorlage" → ausfüllen → **legt wirklich an** (war toter Button). Einstellungen → Nummernkreis-Format ändern → Live-Vorschau.

---

## 🟢 dashboard (D-1…D-5)

**6. Admin-Crash-Fix (D-1)** — `/#/settings/dashboard` → Tab Administrator → **„Aktuelles Layout als Standard"** klicken → grüner Erfolgs-Toast, **kein Weiß-Screen** (war harter Crash). Dashboard anpassen → Widget hinzufügen/entfernen → überlebt Reload.

**7. Tote Buttons jetzt funktional (D-2)** — MyTasks-Zeile klicken → „Meine Aufgaben". „Empfohlene Widgets"-Karte „+" → Toast „… hinzugefügt" + Karte weg. „Heute im Überblick" → **„8 ungelesene Nachrichten"** (war 0). Geburtstage-Widget lädt 5 Einträge.

**8. KPI-Gating (D-3)** — nichts Sichtbares: Demo zeigt absichtlich **alle** Widgets; Lizenz-Gating wirkt technisch (per Flag getestet). Nur zur Info.

**9. Team-Dashboard (D-4)** — Umschalter oben rechts auf **„Team"** → Team-Status (Presence), Geburtstage, Stempeluhr, Offene Tickets, **Team-Arbeitszeit** mit 6 Mitarbeitern + unterschiedlichen Wochenstunden (echte MSW-Daten, kein Fake mehr).

**10. DnD + Cross-Module-Link (D-5)** — „Dashboard anpassen" → jedes Widget bekommt Grip (verschieben) + Eck-Resize + X. „Heute im Überblick" → „offene Aufgaben" klicken → landet in „Meine Aufgaben" (war toter Link).

---

## ⚠️ Offene Nebenbefunde (zur Kenntnis / Entscheidung)

- **dashboard: Abwesenheiten-Widget leer** („0 Personen heute abwesend"). Vorbestehender **HR-Pipeline-Bug außerhalb der dashboard-Lane**: `useAbsenceCalendar` erwartet `data.entries`, Handler liefert `{absences}` + Feld-Mismatch + Duplikat-Handler (`hr.ts` liefert `[]`, überschattet `team.ts`). → eigener HR-/team-Lane-Fix.
- **vertraege: Branch-Iso nicht genutzt** — das Sub-Terminal hat direct-to-main gepusht statt auf `parallel/vertraege`. Kein Schaden (Lane-Trennung hielt, 0 i18n-Konflikte, 0 Doppel-Keys), aber die geplante Branch-Sicherung griff nicht. Für nächste Parallel-Runde: Umstell-Anweisung früher/expliziter.
- **vertraege Detail (Dariens alte Frage):** standalone Share-Link etc. nicht Teil dieses Tiefe-Passes.
