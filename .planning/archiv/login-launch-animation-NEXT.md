# START-PROMPT — Cosmi Öffnungs-/Flug-Animation (neuer Terminal)

> Diese Datei in der neuen Session zuerst lesen. Self-contained. Baut auf dem
> bereits gemergten Login-/Launch-Screen auf (Memory [[login-launch-screen]]).

## Was schon da ist (auf `main`)
- **Login-/Launch-Screen** fertig: `modules/auth/AuthLayout.tsx` rendert auf
  durchgehendem Weltraum-Hintergrund (`SpaceBackground.tsx`) die CosmiLaunch-
  Animation (`CosmiLaunch.tsx`, vendored: Sterne→C→COSMI). Intro zentriert →
  Logo gleitet rechts (GPU-transform), Login-Fenster faded links ein.
- Choreografie-Tokens in `styles/animations.css` (`--dur-launch-*`,
  `.cosmi-launch-logo`, `.cosmi-auth-form`, `.cosmi-star`).
- Dev-/QA-Preview: Route `/#/launch-preview` (nur DEV, ohne GuestRoute) +
  `scripts/qa-launch.mjs` (Playwright gegen vite.qa :5174).
- CosmiLaunch-Props: `size`, `shiftLogo` ("none"|"up"|"left"), `onWordmark`,
  `onComplete`. `/* eslint-disable */` MUSS vor `// @ts-nocheck` stehen.

## DAS NEUE FEATURE (Darien-Spec, wörtlich umgesetzt)
**Problem heute:** Nach dem Laden/Login wird das Fenster kurz **weiß** und man
sieht eine **Vorschau/Skelett** vom ladenden Dashboard (= Suspense-Fallback
`ModuleLoadingFallback`). Das ist ein harter, hässlicher Übergang.

**Gewünscht:** eine **Übergangs-/Öffnungs-Animation** („Flug nach vorn / rein
ins Bild"), die diesen Übergang ersetzt. Zwei Einstiegs-Pfade, **beide enden in
DERSELBEN Flug-/Öffnungs-Animation**:

- **Fall A — Start MIT Token (bleibt eingeloggt):** die Cosmi-Logo-Animation
  läuft wie gewohnt (CosmiLaunch), **dann ein Flug nach vorn / Zoom „rein ins
  Bild"**; beim Reinfliegen **blendet das Cosmi-Logo aus** → das Dashboard wird
  sichtbar. Also: Logo-Animation → Fly-in/Zoom → Logo-Fade-out → App offen.
- **Fall B — Login eingeben:** bleibt wie jetzt (Login-Screen). **Erst nach
  korrektem Login** kommt die **gleiche** Flug-/Öffnungs-Animation rein ins
  Dashboard (wie Fall A, nur ohne das vorherige Logo-Intro).

Kernstück = die **Fly-in/Zoom-Öffnungs-Animation mit Logo-Fade-out**, geteilt von
beiden Pfaden, die den Skelett-Weiß-Flash überdeckt.

## Technische Integrationspunkte
1. **App-weiter Startup/Transition-Overlay nötig** (über dem Router):
   - **Fall A (authed Start):** Overlay spielt CosmiLaunch beim App-Start,
     während `auth.initialize()` läuft → danach Fly-in → Dashboard. Das ist der
     noch offene „app-weite Startup-Splash" (war Task #8). **Achtung
     `DEV_BYPASS_AUTH`** (`App.tsx`: `import.meta.env.DEV` → auto-auth in
     dev/QA) — Branch-Logik + QA-Preview drum herum bauen.
   - **Fall B (post-login):** nach `login()`-Erfolg in `LoginPage.tsx`
     (`navigate('/')`) das Fly-in-Overlay über das Dashboard legen, bis das
     Dashboard geladen ist (oder feste Dauer), dann ausfaden.
2. **Den Weiß-/Skelett-Flash maskieren:** Das Overlay muss ÜBER dem
   Route-Wechsel + Suspense-Fallback liegen (`lazyRoute`/`ModuleLoadingFallback`
   in `App.tsx`). Fly-in läuft, Dashboard lädt dahinter, dann Overlay weg.
3. **Fly-in technisch:** GPU-only (transform: scale/translateZ + perspective +
   opacity). Tokens/Keyframes in `styles/animations.css` (kein magic-number in
   Komponenten). `prefers-reduced-motion` → sofort rein, kein Zoom.
4. **CosmiLaunch wiederverwenden:** das Logo-Fade-out beim Reinfliegen kann ein
   neuer Zustand/Prop am Logo-Container sein (scale-up + opacity→0). CosmiLaunch
   selbst nicht umbauen müssen — der Container in AuthLayout/Overlay animiert.

## Relevante Dateien
- `modules/auth/AuthLayout.tsx`, `CosmiLaunch.tsx`, `SpaceBackground.tsx`
- `styles/animations.css` (Choreografie-Tokens)
- `App.tsx` (Router, `ProtectedRoute`/`GuestRoute`, `lazyRoute`+Suspense,
  `DEV_BYPASS_AUTH`), `main.tsx` (Boot, `startDemoMode`)
- `stores/auth.ts` (`login`, `initialize`, isLoading/isAuthenticated)

## Verify
- `scripts/qa-launch.mjs` erweitern (Fly-in-Phase + Fall A/B). Route
  `/#/launch-preview`. Für Fall A evtl. eigene Preview-Route, da DEV_BYPASS
  auto-einloggt. Screenshots der Phasen ANSEHEN (Build-+-Verify-Standard).
- Build-+-Verify-Standard: i18n falls Texte, gescopter tsc (nur geänderte
  Dateien, App.tsx zieht den ganzen Graph → isolieren), ESLint, ein Commit+Push.

## Offener Nebenpunkt — Icon (zwei getrennte Sachen)

### (1) Lokaler Shortcut-Cache (Darien sieht noch das Atom)
Die installierte `Cosmi.exe` trägt das korrekte dunkle Cosmi-C (per `rcedit`
gesetzt, verifiziert). Die **Desktop-Verknüpfung zeigt aber noch das Electron-
Atom** = **Windows-Icon-Cache**, NICHT das Icon-File (Darien-Files `icon-128/144/
180.png` aus Temp sind byte-identisch zu `assets/branding/cosmi-icons/`). Fix
(später, nicht dringend): Cache hart leeren —
`ie4uinit.exe -ClearIconCache`, dann `iconcache*.db` /
`%LOCALAPPDATA%\Microsoft\Windows\Explorer\iconcache_*.db` löschen +
`Stop-Process -Name explorer -Force; Start-Process explorer` (schließt
Explorer-Fenster). Ggf. Verknüpfung neu erstellen. Das `.exe`-Icon selbst muss
NICHT neu gemacht werden.

### (2) Distribution-Installer
Das App-Icon ist im **gebauten/installierten** `.exe` korrekt (per `rcedit`
nachträglich gesetzt; `electron-builder.yml` hat jetzt `signAndEditExecutable:
true` + `win.icon: build/icon.ico`). ABER: auf dieser Dev-Maschine scheitert
electron-builders Icon-Edit beim Build an `winCodeSign`-Symlinks (Developer-Mode/
Admin fehlt) → der **verteilbare Installer** bekommt das Icon nur, wenn der Build
auf einer Maschine mit Symlink-Rechten läuft ODER per `afterPack`-Hook rcedit
direkt aufgerufen wird. Gehört zur [[auto-update-initiative]] (Signing +
zuverlässiges Icon-Embed). Für „die anderen" notfalls Installer + `rcedit
Cosmi.exe --set-icon build/icon.ico` (Binary: `node_modules/electron-winstaller/
vendor/rcedit.exe`).
