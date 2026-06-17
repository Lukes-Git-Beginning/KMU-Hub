# QA-Protokoll — team (Main-Terminal, Branch `main`)

> Build-+-Verify pro Punkt: bauen → i18n ×4 → Playwright-Screenshot **angesehen** → commit. Dev gegen :5173.

| Punkt | Inhalt | Status | Verify |
|---|---|---|---|
| **TM-1** | Abwesenheiten-Bug: Handler `{entries}` + camelCase `AbsenceEntry` + Date-/Dept-Filter; Demo-Daten angereichert (heute aktiv); hr.ts-Duplikat raus | ✅ | `qa-team-absences.mjs`: team-Kalender zeigt Lena (Krankheit Di–Do) / Markus (Homeoffice heute) / Sophie (Sonderurlaub Mo22), korrekte Abteilungen; Felix/Julia korrekt rausgefiltert (außerhalb Fenster); Dashboard-Widget hat Markus + 2× „abwesend"; 0 raw keys, 0 pageErrors. Screenshots angesehen. |
| **TM-2** | SelfServiceView verkabeln (useSelfProfile/useLeaveBalance/useLeaveRequests/useCreateLeaveRequest, Anträge wirken, Download Blob); adaptEmployee +phone/location/workload/initials; team.ts Stefan-Seeds + POST-Handler | ✅ | `qa-team-selfservice.mjs`: Profil = **Stefan Vogel** (echt, kein Jonas Diaz), Salden 17/30 + 4/5; 3 echte Anträge; **Create-Roundtrip wirkt** (neuer Antrag 01.09. erscheint → „4 Anträge" + Toast + fließt in HR-Anfragen-Tab); 0 raw keys, 0 `{{}}`, 0 pageErrors. Build grün (1m51s). Screenshots angesehen. |
| **TM-3** | PersonnelDocuments echt + Download/Preview | ⬜ | |
| **TM-4** | OrgChart-Actions + i18n `{{}}`→`{}` (8 Keys) + `team.page.title` | ⬜ | |
| **TM-5** | Schulungen Zustand→MSW-Hook (P1) + handleDeactivate-Fix | ⬜ | |

## Nebenbefunde (Cleanup, nicht TM-Scope)
- **Umlaut-Bug:** `DEPARTMENTS` in `mocks/mock-db` hat „Geschaeftsführung" (ae statt ä) — sichtbar im Abteilungs-Chip + Abwesenheitskalender. User-facing Umlaut-Verstoß. Kandidat für TM-4-Sammel-Commit oder separaten Cleanup.
