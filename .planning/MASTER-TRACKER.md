# Master-Tracker — Modul-Bau bis „persönliche Modul-Reviews"

> **Zweck:** Ein Ort, an dem wir den Gesamtfortschritt sehen und Phase für Phase abarbeiten.
> Detailpläne pro Modul: `.planning/reviews/<modul>.md` + `.planning/module-phase-plans.md`.
> **Stand:** 2026-06-16. Pflege: nach jedem Modul/Phasen-Abschluss Status hier aktualisieren.

## Legende
✅ fertig · 🔨 in Arbeit/teilfertig · ⬜ nicht begonnen · 🔒 wartet auf Luke-Backend · 🔁 Re-Check nötig

**Demo-Tiefe** = der projektweite Standard ab 2026-06-16 ([[feedback_module_depth_standard]]): jede Liste öffnet eine echte Detail-Ansicht, jeder Download/Export wirkt sichtbar, Placeholder→MSW. **Jedes Modul braucht diesen Pass, bevor es „review-reif" ist** — auch die schon „fertigen" (vor dem Standard gebaut). finanzen P2.5 ist die Referenz.

## Gesamt-Überblick
- **~25 Kern-Module + 7 Branchen-Module.**
- **Fertig (Kern-Phasen):** ~4 — kontakte, calendar, dokumente, zeiterfassung.
- **In Arbeit:** ~6 — finanzen, berichte, notifications, wiki, formulare, settings.
- **Nicht begonnen:** ~13 Kern + 7 Branchen.
- **Grobe Reife: ~20 % des Modul-Baus.** Dazu kommt der Demo-Tiefe-Pass pro Modul (neuer Standard) — der vergrößert den Rest spürbar.
- **Persönliche Modul-Reviews** starten, wenn ein Modul „Phasen ✅ + Demo-Tiefe ✅" ist (rollierend pro Modul möglich, nicht erst ganz am Ende).

---

## Cluster 1 — Vertrieb & Kommunikation
| Modul | Phasen-Status | Demo-Tiefe | Plan |
|---|---|---|---|
| **kontakte** | ✅ P0–7 (P8 Finanzberatungs-Tiefe offen) | 🔁 weitgehend (Re-Check) | reviews/kontakte.md, build-progress.md |
| **calendar** | ✅ (Marathon, „komplett" — verifizieren) | 🔁 Re-Check | reviews/calendar.md |
| **notifications** | 🔨 Quiet-Hours (Nico) — P1/P2 tw. | ⬜ | reviews/notifications.md |
| **kommunikation** | ⬜ (3-Panel-Chat da) | ⬜ | module-phase-plans §kommunikation |
| **mails** | ⬜ Neubau 🔒 | ⬜ | module-phase-plans §mails |
| **video/meetings** | ⬜ 🔒 LiveKit | ⬜ | module-phase-plans §video |
| **dialer** | 🔨 FE-P1 ✅, P2–5 🔒 LiveKit | ⬜ | project_dialer |

## Cluster 2 — Arbeit
| Modul | Phasen-Status | Demo-Tiefe | Plan |
|---|---|---|---|
| **dokumente** | ✅ (Marathon, „komplett" — verifizieren) | 🔁 Re-Check | reviews/dokumente.md |
| **zeiterfassung** | ✅ P1–P5 (`0303a37`); Dead-Code-Cleanup offen | 🔁 Re-Check | reviews/zeiterfassung.md |
| **work** | ⬜ (sehr vollständig FE) | ⬜ | module-phase-plans §work |
| **wiki** | 🔨 Nico-Gate grün — tw. | ⬜ | reviews/wiki.md |
| **formulare** | 🔨 Strom N — tw. | ⬜ | reviews/formulare.md, nico-block |
| **berichte** | 🔨 Sparklines (Nico) — P1 tw. | ⬜ | reviews/berichte.md, nico-block |
| **team** | ⬜ (12 Tabs, tw. Query) | ⬜ | module-phase-plans §team |
| **helpdesk** | ⬜ (auf Zustand) | ⬜ | module-phase-plans §helpdesk |

## Cluster 3 — Finanzen
| Modul | Phasen-Status | Demo-Tiefe | Plan |
|---|---|---|---|
| **finanzen** (= „Buchhaltung") | 🔨 P1 ✅ · P2 ✅ · **P2.5a ✅** · P2.5b–e + P3–P5 offen | 🔨 **P2.5 = Referenz, läuft** | reviews/finanzen.md |
| **buchhaltung** (dead) | ⬜ Aufräum-Task (lebt in finanzen); Rename geparkt (mit Luke) | – | reviews/finanzen.md §Geparkt |
| **vertraege** | ⬜ (FE-Only auf Zustand) | ⬜ | reviews/vertraege.md |

## Cluster 4 — System & Konto
| Modul | Phasen-Status | Demo-Tiefe | Plan |
|---|---|---|---|
| **settings** | 🔨 Fundament ✅ · P2–P5 offen | ⬜ | module-phase-plans §settings |
| **dashboard** | ⬜ (vollständig, Persistenz Mock) | ⬜ | reviews/dashboard.md |
| **profil** | ⬜ | ⬜ | reviews/profil.md |
| **security** | ⬜ 🔒 **DSGVO = P0-Launch-Blocker** | ⬜ | module-phase-plans §security |
| **admin** | ⬜ 🔒 | ⬜ | module-phase-plans §admin |
| **automatisierung** | ⬜ 🔒 Engine (großer Block) | ⬜ | module-phase-plans §automatisierung |

## Cluster 5 — Branchen (pilot-getrieben, später; ⚠ = PWA-Bedarf)
| Modul | Phasen-Status | Demo-Tiefe | Plan |
|---|---|---|---|
| **rapporte** ⚠ | ⬜ (Solar-Pilot-Kern) | ⬜ | module-phase-plans §rapporte |
| **schichten** ⚠ | ⬜ | ⬜ | §schichten |
| **fuhrpark** ⚠ | ⬜ | ⬜ | §fuhrpark |
| **vermietung** ⚠ | ⬜ | ⬜ | §vermietung |
| **inventar** ⚠ | ⬜ | ⬜ | §inventar |
| **einkauf** | ⬜ | ⬜ | §einkauf |
| **produktion** ⚠ | ⬜ 🔒 MRP | ⬜ | §produktion |

---

## Was als Nächstes (aktive Spur)
1. **finanzen P2.5** fertig bauen (Referenz für Demo-Tiefe):
   - [x] **P2.5a** — Angebots-/Gutschrift-Detailansicht + Quote-Mutationen (`395c02f`)
   - [ ] P2.5b — Ausgaben-/Transaktions-/Wiederkehrend-Detail + OP-Liste→Rechnung + Dashboard-Listen klickbar
   - [ ] P2.5c — PDF-Vorschau aus echten Daten + Downloads (PDF/CSV) sichtbar wirksam
   - [ ] P2.5d — Mahnwesen verkabeln + Mahn-Detail + Zahlung/Settings speichern
   - [ ] P2.5e — Hardcoded-Mocks auf MSW (Banking, Belegkette, Stunden→Rechnung, Audit-Log)
2. **finanzen P3–P5** (DATEV/Bexio · E-Rechnung/GoBD · Banking/EÜR).
3. Dann nächstes Modul nach bestätigter Reihenfolge — pro Modul: Phasen + **Demo-Tiefe-Audit** + Review.

## Offene Klärungen (für die Gesamt-Reihenfolge)
- Reihenfolge nach finanzen bestätigen (Vorschlag aus module-phase-plans: team → mails → kommunikation → work/zeiterfassung-Rest → dashboard → security(P0!) → …).
- **security/DSGVO ist P0-Launch-Blocker** — sollte nicht zu spät kommen (hängt an Luke).
- Marathon-Module (berichte/notifications/wiki/formulare/calendar/dokumente) realen Stand gegen ihre review-Docs verifizieren (hier teils „verifizieren" markiert).
