# Entscheidungsvorlage: Schichten anschalten oder von der Preisliste nehmen

> Stand 2026-10-20, gegen Code, Website und Produktion geprüft. Entscheidet: Darien und Luke.
> Gebaut wird erst nach der Entscheidung.

## Die Lage in einem Satz

Schichten steht auf der Website als buchbares Modul (4 € pro Nutzer), ist in Produktion aber
abgeschaltet. Wer es heute bucht, bekommt nichts zu sehen.

## Was technisch da ist

| Bereich | Stand |
|---|---|
| Backend | 20 RPCs, 20 HTTP-Routen, 172 Tests, ~80 % Coverage, 5 Migrationen |
| Rechte | geseedet für alle Rollen (000256, 000161, 000224) |
| Tenant-Aktivierung | für bestehende Tenants aktiv (000250) |
| Desktop | Schichtplan-Seite mit Drag & Drop, Tausch-Anfragen, Einstellungen, i18n ×4 vollständig bis auf 1 Key |
| Deploy | Service läuft bereits in Produktion, nur die Gateway-Routen sind per Flag aus |

**Anschalten kostet technisch fast nichts:** eine Zeile im Gateway-Block von
`deploy/docker/docker-compose.yml` (`COSMI_MODULE_SCHICHTEN_ENABLED`), der Wert `true` in
`/opt/kmuhub/.env.production` und ein Neustart des Gateways.

## Was die Website verspricht und was fehlt

Quelle: `zentria-website/src/data/module-copy.ts:170-179` und `modules.ts:344-358`.

| Versprechen | Wirklichkeit |
|---|---|
| Dienstpläne per Drag & Drop | vorhanden |
| Tausch-Anfragen mit einem Klick | vorhanden, aber der Dialog bietet ungültige Partner an; das Backend lehnt die Genehmigung dann ab |
| Wochen- **und Monatsplan** | nur Wochenansicht |
| **Automatische Konfliktprüfung** | fehlt: Doppelbelegung einer Person wird nicht geprüft (offene Entscheidung `decide-schichten-assign-employee-overlap-guard`) |
| **Verfügbarkeits-Kalender pro Person** | Attrappe: feste Demo-Daten (`SchichtenPage.tsx:151`), Änderungen gehen beim Neuladen verloren |
| **Mobile-Push bei Plan-Änderungen** | fehlt ganz, der Service sendet keine Ereignisse |

Dazu kommt ein bekannter Fehler: Die Schicht-Statistik ignoriert den Datumsfilter.

## Die zwei Wege

**A — Anschalten.** Vorher müsste die Website auf das reduziert werden, was es gibt (Wochenplan,
Drag & Drop, Tausch), oder die Lücken werden geschlossen. Grobe Größenordnung für das Schließen:
Verfügbarkeit echt speichern und Konfliktprüfung je 1–2 PT, Monatsansicht 1–2 PT,
Benachrichtigungen 2–3 PT (hängt am Notification-Service). Das ist Feature-Bau, also genau das,
was der 90-Tage-Test zurückstellt.

**B — Von der Preisliste nehmen.** `status: 'planned'` in `modules.ts`. Dann laufen
`scripts/check-claims.mjs` und damit der Website-Build auf Blog-Preistabelle, Branchenpakete und
`preise.astro` auf. Diese Stellen sind mitzuziehen, außerdem `docs/PRICING.md`,
`.knowledge/pricing.md` und `desktop/src/renderer/src/lib/pricing.ts`. Etwa ein halber Tag.

## Gegenläufiger Befund

Berichte und Formulare sind in Produktion **an**, auf der Website aber als „geplant“ markiert.
Das ist der umgekehrte Fall und gehört in dieselbe Bereinigung.

## Empfehlung

Hängt an der Zielgruppe des 90-Tage-Tests:

- **Zielgruppe ohne Schichtbetrieb** (Beratung, Büro, Agentur): **B.** Ein Modul, das niemand im
  Test braucht, sollte nichts versprechen, was nicht da ist.
- **Zielgruppe mit Schichtbetrieb** (Pflege, Gastronomie, Handwerk mit Bereitschaft): **A**, aber
  zuerst die Website-Copy auf den echten Umfang kürzen und dann anschalten. Die Lücken werden nach
  dem tatsächlichen Bedarf des Piloten geschlossen, nicht vorab.

In beiden Fällen kommt die Berichte/Formulare-Korrektur auf der Website mit.
