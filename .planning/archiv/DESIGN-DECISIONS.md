# Design- & Produktentscheidungs-Leitfaden — für die autonomen Bau-Ströme

> **Pflichtlektüre für jeden Bau-Strom (Nico · Dein-PC · Luke-FE).** Verlinkt aus jedem Starttext.
> **Zweck:** Ihr baut allein an euren PCs und müsst dabei Design-/Produktentscheidungen treffen, die das Haupt-Team (Darien) sonst live mit Claude bespricht — *wie sieht es aus, wo gehört es hin, was fehlt noch*. Dieses Dokument macht euch dafür entscheidungsfähig: ihr sollt entscheiden **wie Darien entscheiden würde**, nicht auf Rückfrage warten.
> **Verhältnis zu anderen Docs:** Die volle Design-Tiefe (Philosophie, Motion-Token-Tabelle, Joy-Matrix, Personality-Copy) steht in **`.knowledge/design.md`** — dort nachschlagen. **`CLAUDE.md`** (Repo-Root, Abschnitt UI/UX) ist die Kurzform. Dieses Doc ergänzt beides um die **Entscheidungs-Heuristiken** und die **wiederkehrenden UX-Muster mit echten Komponentennamen**.

---

## 0. Das Entscheidungs-Protokoll (das Wichtigste)

Ihr lauft unbeaufsichtigt. Also gilt: **nie blockieren, nie raten ohne Spur.**

1. **Entscheidbar aus diesem Doc / design.md / CLAUDE.md / vorhandenem Muster im Repo?** → entscheiden, bauen, im Review-Faden (`reviews/<modul>.md`) **eine Zeile** notieren: „Entscheidung: X, weil Y."
2. **Echte Domänen-/Produktunklarheit** (was bedeutet das fachlich, welcher Flow ist gewollt)? → **sinnvollen Default bauen**, im Review-Faden als „**offene Frage:** … (Default gebaut: …)" notieren. Darien klärt beim Merge-Review.
3. **Reine Geschmacksfrage ohne richtige/falsche Antwort?** → die **konservativere, ruhigere** Variante wählen (Apple-Linse: Reduktion). Nicht die auffälligere.

> Faustregel: Eine getroffene Entscheidung + notierte Begründung ist **immer** besser als Stillstand. Ein dokumentierter Default ist **immer** besser als ein stiller Default.

---

## 1. Cosmi-Identität in einem Satz

**Premium SaaS mit Editorial Touch — Apple-Disziplin fürs Tägliche, Discord-Wärme für die Momente.** Cosmi kopiert weder Apple noch Discord visuell; es nutzt sie nur als *Prinzipien*. (Details: `.knowledge/design.md` → „Designphilosophie".)

**Apple-Linse** (Reduktion, Hierarchie, Daily-Use-Disziplin) → für alles was 100×/Tag passiert: Tabellen, Forms, Filter, Navigation.
**Discord-Linse** (Wärme, Personality, Joy) → nur Empty-States, Onboarding, Success, Presence.

**Daily-Use-Test bei jeder Entscheidung:** *Nervt das nach Tag 30?* Wenn ja → ruhiger machen.

---

## 2. Harte Regeln — nicht verhandelbar (Quick-Reference)

| Regel | Konkret |
|---|---|
| **Fonts** | NIE Inter, Roboto, Arial, Space Grotesk, Helvetica, Open Sans. Erlaubt: Plus Jakarta Sans, Clash Display, Satoshi, JetBrains Mono, Playfair Display. |
| **Keine Emojis im UI** | Personality über `lucide-react`-Icons, Custom-SVG (`components/shared/illustrations/`), Motion, Wording. |
| **Echte Umlaute** | „für" nicht „fuer", „löschen" nicht „loeschen". Gilt für UI-Text UND Code-Kommentare/Docs wo Umlaute möglich sind. |
| **i18n ×4** | Jeder sichtbare Text als Key in `i18n/messages/{de,en,fr,it}.json`. Interpolation `{var}` (NICHT `{{var}}`). Plural als ICU: `{count, plural, one {…} other {…}}` (nie `_one`/`_other`). |
| **Motion** | Nur `transform`/`opacity` animieren (GPU), nie `width/height/margin/padding`. Tokens aus `lib/motion.ts` + `styles/animations.css`, keine Magic-Numbers. Nie `ease-in` für UI. |
| **Farben** | Nur über CSS-Variablen/Tailwind-Tokens (`text-foreground`, `bg-card`, `text-muted-foreground`, `border-border` …). Nie Hex hart im Markup. Muss in allen Themes funktionieren. |
| **Keine sichtbaren Scrollbars** | Bestehende Scroll-Container-Muster wiederverwenden. |
| **Keine nativen Browser-Controls** | Kein `<input type="time/date/color">` (sieht nach Browser aus). Cosmi-Komponenten nutzen: `shared/TimePicker`, `shared/ColorSwatchPicker`. Fehlt eine? In `shared/` bauen. |
| **Anti-Patterns** | Keine Card-in-Card, keine AI-Slop-Ästhetik (lila Gradienten), keine symmetrischen Bootstrap-3-Karten-Grids, keine Über-Animation. |

---

## 3. Wo gehört was hin? — IA-Heuristiken

Wenn ihr nicht sicher seid, wo ein Element/Feature platziert wird:

- **Detail-/Unteransicht** → braucht IMMER einen **Zurück-Weg** (Zurück-Button oben links). Keine Sackgassen. Für seitliche Details: `shared/DetailPanel`.
- **Listen/Tabellen** → brauchen: **Sortierung** (Feld + Richtung, via `shared/SortMenu` — nie nur „sortiert", immer wonach + auf/ab), **Filter**, und eine **Ansichts-/Dichte-Option** wo sinnvoll (`shared/LayoutSwitcher` für Grid/Liste/Tabelle).
- **Aktionen pro Eintrag** → `shared/ItemActions` (konsistentes Overflow-Menü), Bestätigung destruktiver Aktionen via `shared/ConfirmDialog`.
- **Seitentitel/Kopf** → `shared/PageHeader` (Titel, Untertitel, primäre Aktion rechts).
- **Kennzahlen** → `shared/StatCard` / `shared/InlineStat`.
- **Modul-Einstellungen** → in das Modul-Einstellungs-Fenster, NICHT verstreut. Siehe §5.
- **Globale vs. modul-lokale Aktion:** Suche/Quick-Actions sind global (`shared/GlobalSearch`). Modul-spezifische Aktionen bleiben im Modul-Kopf.

**Grundsatz:** Erst im Repo nach einem bestehenden Muster suchen (gibt es das Modul nebenan schon ähnlich?), dieses Muster übernehmen. **Wiederverwenden vor Neubauen.** Etwas Neues, das mehr als ein Modul braucht → in `components/shared/` bauen, nicht ins Modul kopieren.

---

## 4. UX-Vollständigkeit — „an alles denken"

Ein Feature ist erst fertig, wenn ALLE Zustände sauber aussehen — nicht nur der Happy Path mit drei Demo-Datensätzen:

- **Leerer Zustand** → nie ein leerer/kaputter Screen. `shared/EmptyState` + passende Illustration aus `shared/illustrations/` (es gibt u.a. `EmptyCalendar`, `EmptyContacts`, `EmptyDocuments`, `EmptyReports`, `EmptyWiki`, `EmptyGeneric`). Warme, kurze Copy (siehe Personality in `design.md`).
- **Lade-Zustand** → Skeleton-Shimmer (<1.5s), bei längerem Laden Illustration. `shared/LoadingSpinner` nur wo Skeleton nicht passt.
- **Fehler-Zustand** → empathische Copy + Lösungspfad, nie roher Error-Code.
- **Edge-Cases** → lange Texte (Truncation/Umbruch testen), viele Einträge, ein einziger Eintrag, sehr kurze Werte.
- **Volle Breite** → bei 1440px prüfen, nichts abgeschnitten, keine Card-in-Card.

> Darien-Standard: **„An alles denken + alles überprüfen."** QA über mehrere Datensätze UND mehrere Zustände — siehe §7.

---

## 5. Modul-Einstellungen — Standard pro Modul

Jedes Modul wird „settings-komplett" gebaut: ein Eintrag im Modul-Einstellungs-Fenster über `shared/ModuleSettingsShell`, registriert in `modules/settings/module-settings-registry.tsx`.

- **Zwei Bereiche:** `personal` (gilt nur für den Nutzer) + `tenant` (gilt org-weit, RBAC-geschützt). Scopes in `components/shared/module-settings-scope.ts`.
- Hook: `hooks/useModuleSettings.ts`.
- **Registry ist eine Hot File** (Merge) → nur **additiv** euren Eintrag anhängen, nichts umsortieren.
- Lieber pro Modul jetzt etwas mehr Zeit in saubere Settings stecken — spart später.

---

## 6. Joy-Moments — wann gespielt werden darf

Daily-Use-Module (Tabellen, Forms, Filter, Nav) bleiben **streng**: unsichtbar schnell, kein Bounce, keine Reveals. Joy NUR an:

- **Empty-States, Onboarding/First-Run, Success-Moments** (Deal won, Task done, Invoice paid), **Loading >1.5s**, **Avatar/Presence**.
- Werkzeuge dafür existieren: `shared/ConfettiBurst`, `shared/AnimatedCheckmark`, `shared/TextReveal`, NumberTicker.

Volle Matrix mit Timings: `.knowledge/design.md` → „Joy-Moments-Matrix". **Im Zweifel: streng.** Joy ist die Ausnahme, nicht der Default.

---

## 7. Verifikation — bevor etwas „fertig" ist

Der Build-+-Verify-Prozess steht in `nico-block/WORKFLOW.md` (gilt für alle Ströme). Die Design-relevante Essenz:

1. **Gescopter Typecheck** (nur eure Dateien) — „kompiliert" ist die Basis, nicht das Ziel.
2. **Playwright-Screenshot-QA** (`scripts/qa-<modul>-*.mjs`) — Script klickt durch, legt PNGs ab.
3. **Die Screenshots WIRKLICH mit dem Read-Tool ansehen** — das ist der eigentliche Qualitäts-Hebel. Prüfen auf: Roh-i18n-Keys, Emojis, abgeschnittene Texte, falsche Umlaute, kaputtes/leeres Layout, Card-in-Card, Theme-Bruch.
4. **Über mehrere Zustände** prüfen: leer / mit Daten / Edge-Case. Nicht nur ein Screenshot.

> „Script lief grün" reicht NICHT. Die Bilder müssen angeschaut werden.

---

## 8. Wenn ihr eine Pixel-Vorlage / einen Reference-Screenshot bekommt

Falls eine Spec einen Reference-Ordner oder ein Mockup nennt: das ist **Pixel-Wahrheit, keine Inspiration**. Farben mit dem Auge am Screenshot abgleichen (nicht schätzen), Layout 1:1 nachbauen. Abweichung nur mit Begründung im Review-Faden.

---

## 9. Kurz-Checkliste vor jedem Commit

- [ ] Sichtbarer Text = i18n-Key in 4 Sprachen, `{var}`-Klammern, ICU-Plural
- [ ] Keine Emojis, echte Umlaute, keine Hex-Farben, Tokens statt Magic-Numbers
- [ ] Empty-/Loading-/Error-/Edge-Zustand gebaut, nicht nur Happy Path
- [ ] Zurück-Weg in Detail-Views, Sortierung als Feld+Richtung wo Liste
- [ ] Wiederverwendet aus `shared/`/`ui/` statt kopiert; Neues Wiederverwendbares in `shared/`
- [ ] Nur `transform`/`opacity` animiert; Joy nur an erlaubten Momenten
- [ ] Screenshots angesehen (mehrere Zustände, 1440px), keine Roh-Keys
- [ ] Entscheidungen/offene Fragen im Review-Faden notiert
