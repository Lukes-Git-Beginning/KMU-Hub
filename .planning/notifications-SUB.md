# notifications — Sub-Terminal-Paket (review-reif bringen)

> Copy-paste-Block unten ins KMU-Hub-review-Terminal (Port 5174). Disjunkt zum Haupt
> (Haupt baut wiki Phase B unter `modules/wiki` + `shared/document`; Sub fasst NUR
> `modules/notifications`, `mocks/handlers/notifications.ts`, `mocks/data`, settings-notifications-Panel + nav-items an). Merge-Konflikte praktisch ausgeschlossen.

## Ist-Stand (gescoutet)
P1 Notification-Center solide (eigene Route `/notifications`, MSW stateful, DetailModal, EmptyStates). P2 Quiet-Hours/DND **UI-vollständig aber funktional wirkungslos** (Toast prüft sie nicht). Sidebar-Badges pro Modul laufen schon (`useModuleUnreadCounts`). Zwei parallele Stores (Zustand `stores/notifications.ts` für Toast/Dashboard-Widget vs. MSW-React-Query für Center/Bell) — inkohärent. i18n 55 Keys, sauber, keine `{{}}`.

```
Du bist das Sub-Terminal im Sub-Terminal-5-Phasen-Modus. Arbeitsverzeichnis = dieser KMU-Hub-review-Klon (Dev-Server Port 5174). Das Hauptterminal baut parallel wiki Phase B (modules/wiki + components/shared/document) — du fasst NUR notifications an: modules/notifications/, mocks/handlers/notifications.ts, mocks/data/, das settings-Notifications-Panel und nav-items.ts. Sprache: Deutsch (Umlaute, Eszett, Akzente — NIE ASCII-Ersatz).

SCHRITT 0 — aktuellen Stand holen:  git pull --rebase origin main

KONTEXT (Ist-Stand): notifications hat ein funktionsfähiges Notification-Center (Route /notifications, NotificationCenter.tsx, MSW stateful: read/read-all/preferences/quiet-hours/dnd/mutes), eine Header-Glocke (NotificationBell.tsx) und einen Toast-Listener (NotificationToast.tsx). Sidebar-Badges pro Modul laufen via useModuleUnreadCounts. ABER: Quiet-Hours + DND sind UI-vollständig aber wirkungslos (der Toast unterdrückt nichts), Toast hängt am alten Zustand-Store statt an der MSW-Pipeline, Pin/Dismiss sind nur Session-State (kein MSW), kein Sidebar-Nav-Eintrag für /notifications, mutesState leer, kein group_count-Seed, OS-Notification nur bei !document.hasFocus.

DEIN BATCH — notifications zu „review-reif", 5 Phasen je ein Commit:
- N-1 Quiet-Hours + DND durchsetzen: NotificationToast (und Bell-Live-Toasts) prüfen vor dem Anzeigen dnd-Status + isWithinQuietHours(quietHoursState) und unterdrücken dann. Kleiner Helper + Guards. So wird das P2-Feature echt wirksam (Demo: DND an → keine Toasts).
- N-2 Store-Kohärenz: Toast/Dashboard-Widget auf die MSW-React-Query-Pipeline umlenken (eine Quelle der Wahrheit). NotificationFeedWidget aus der API statt Zustand-Store. Bridge ODER MSW-Polling, das neue Notifications „eintreffen" lässt (analog Chat).
- N-3 Sidebar-Nav-Eintrag für /notifications („Benachrichtigungen") + Demo-Seeds: 2-3 mutesState-Einträge (damit Stummgeschaltet-Liste nicht leer) + min. 1 Notification mit group_count>1 (damit „+N weitere" sichtbar).
- N-4 Demo-Tiefe: Pin/Dismiss an MSW persistieren (POST /:id/pin, /:id/dismiss — neue Handler), Deep-Link springt zum konkreten Item (nicht nur Modul-Start). Tote Buttons/leere Zustände prüfen.
- N-5 Settings-Vollständigkeit + Schlusscheck: notifications-Settings-Panel personal + tenant sauber (ModuleSettingsShell), Module-Channel-Matrix mit den event-type-Präferenzen kohärent. Demo-Tiefe-Audit + i18n ×4 + QA.

BUILD-+-VERIFY-STANDARD PRO PHASE (verbindlich):
bauen → i18n ×4 (i18n/messages/{de,en,fr,it}.json; {var} NICHT {{var}}; Plural als ICU {count, plural, one {…} other {…}}) → gescopter Typecheck (eigenes tsconfig nach Muster tsconfig.r6check.json über die geänderten notifications-Dateien; desktop/node_modules/.bin/tsc -p … --noEmit, foreground, echter Exit, NIE | tail) → Playwright-Screenshot-QA gegen http://localhost:5174 (#/notifications + Bell-Popover + Settings) → die PNGs WIRKLICH ansehen (Raw-Keys/Emojis/Theme/Layout/leere Zustände, mehrere Zustände) → iterieren bis grün → ein Commit.

PROJEKTWEITE STANDARDS (sonst Review-Rückläufer):
Detail = shared/DetailModal. Sticky Back/Close. Leere Zustände = EmptyState mit Aktion. Keine Toast-Stubs/toten Endpoints — echte MSW-Funktion. KEINE Emojis (Personality via Custom-SVG/Motion/Wording). Theme-Tokens. Motion nur transform/opacity. Skeleton statt Spinner. CURRENT_USER aus mocks/data/shared-ids. Neue Dateien unter mocks/data/ brauchen git add -f.

GIT:
Conventional Commits, English imperative, KEINE AI-Attribution. Commit pro Phase mit EXPLIZITEN Datei-Pfaden — NIE git add -A/. Nach jedem Commit: git push origin main; bei Ablehnung git pull --rebase origin main, dann erneut push. Dev-Server (5174) NICHT killen.

ABSCHLUSS: kurze Bilanz — welche Phasen committed (Hashes), QA-Ergebnis je Phase, was offen blieb.
```
