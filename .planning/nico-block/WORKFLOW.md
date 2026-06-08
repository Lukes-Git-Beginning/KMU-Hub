# WORKFLOW — so bauen & verifizieren wir (1:1 wie das Haupt-Team)

> Dieser Prozess ist der Grund, warum unsere Phasen sauber werden. Nicos Claude soll **exakt** so arbeiten. Es geht nicht um „einen Skill anklicken", sondern um eine feste Schritt-für-Schritt-Schleife mit echter Verifikation.

## Welche Werkzeuge / Skills wofür

| Zweck | Womit | Ist das ein Skill? |
|---|---|---|
| Code bauen im Cosmi-Stil | Skill **`frontend-design`** (automatisch aktiv) | ja, auto |
| App im echten UI ansehen | Skill **`/run`** | ja |
| „Funktioniert die Änderung wirklich?" | Skill **`/verify`** | ja |
| Diff-Selbstreview vor „fertig" | Skill **`/code-review medium`** | ja |
| **Typecheck** (Fehler finden) | `tsc` über gescopte tsconfig | **kein Skill** — Befehl (s.u.) |
| **Screenshot-Überprüfung** | **Playwright-Script** (`qa-*.mjs`) macht Screenshots → **Claude liest die PNG-Dateien selbst** und bewertet sie | **kein Skill** — Prozess (s.u.) |

**Wichtig:** Die Screenshot-Prüfung braucht keinen speziellen Skill. Sie besteht aus zwei Teilen: (1) ein Playwright-Script klickt die Funktion durch und legt Screenshots ab, (2) **Claude öffnet diese PNG-Dateien (Read-Werkzeug) und schaut sie wirklich an** — auf Raw-Keys, abgeschnittene Texte, Emojis, kaputtes Layout, leere Zustände. Genau dieses „Claude schaut sich den Screenshot an" ist der Qualitäts-Hebel. Nicos Claude muss das aktiv tun, nicht nur das Script laufen lassen.

## Einmal-Setup für die Verifikation
Playwright installieren (einmalig, im Ordner `desktop/`):
```bash
npm install -D playwright
npx playwright install chromium
```

---

## Die feste Schleife pro Phase

### Schritt 1 — Bauen
Code entlang der Definition-of-Done der Spec. i18n in 4 Sprachen. Demo-Handler falls die Spec es verlangt.

### Schritt 2 — Typecheck (gescopt, schnell)
Der volle Typecheck dauert ~20–30 Min — **nie** als Gate nutzen. Stattdessen eine gescopte Config nur über die geänderten Dateien. Claude legt sie an, z.B. `desktop/tsconfig.nicocheck.json`:
```json
{
  "extends": "./tsconfig.web.json",
  "include": [
    "src/renderer/src/global.d.ts",
    "src/renderer/src/vite-env.d.ts",
    "src/renderer/src/i18n/i18next.d.ts",
    "<deine geänderte Datei 1>",
    "<deine geänderte Datei 2>"
  ]
}
```
(Die drei `.d.ts` immer mit drin, sonst kommt Baseline-Rauschen.) Laufen:
```bash
cd desktop && node_modules/.bin/tsc --noEmit -p tsconfig.nicocheck.json
```
- **`exit code 0`** = alles im Scope sauber → super.
- Fehler? Nur die in **deinen** Dateien zählen — danach filtern (z.B. `... | grep DeinDateiname`). Baseline-Fehler in fremden Dateien (asset-Importe, `electronAPI`) ignorieren.
- ⚠ **Nie** parallel einen Warte-Loop (`until … do :`) laufen lassen — der frisst die CPU und bremst tsc massiv aus. Foreground laufen lassen und warten.
- Danach die `tsconfig.nicocheck.json` wieder löschen (nicht committen).

### Schritt 3 — Screenshot-QA (Playwright)
Claude schreibt pro Phase ein `desktop/scripts/qa-<phase>.mjs`. **Vorlage** (kopiere ein bestehendes `qa-*.mjs`, dieselbe Boilerplate):
```js
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')

// Stub für Electron-API (App läuft sonst nicht im reinen Browser)
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, p) => (p === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`
// Onboarding-Overlay unterdrücken
const SUPPRESS_ONBOARDING = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
`
// Findet versehentlich sichtbare i18n-Roh-Keys (z.B. "notifications.foo.bar")
async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(notifications|berichte|wiki|formulare|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } }) // volle Breite!
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const out = {}

try {
  await page.goto(`${BASE}/#/<deine-route>`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  // ... die Kern-Funktion klicken + prüfen, z.B.:
  out.featureVisible = await page.getByText(/Erwarteter Text/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, '<phase>.png'), fullPage: true })
  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 5)
await ctx.close(); await browser.close()
console.log(JSON.stringify(out, null, 2))
```
Laufen (Dev-Server muss auf :5173 laufen):
```bash
cd desktop && node scripts/qa-<phase>.mjs
```
Erwartung im JSON-Output: `featureVisible: true`, `rawKeys: []`, `pageErrors: []`.

### Schritt 4 — Screenshots ANSEHEN (der eigentliche Qualitäts-Check)
Claude öffnet die erzeugten PNGs in `desktop/.qa-screenshots/` **mit dem Read-Werkzeug** und prüft visuell:
- Keine Roh-Keys (`modul.xyz.key`) sichtbar
- Keine Emojis im UI
- Texte nicht abgeschnitten, Umlaute korrekt (für/löschen)
- Kein kaputtes/leeres Layout, keine Card-in-Card
- **Mehrere Zustände** geprüft: leer / mit Daten / langer Text
> Das ist nicht optional. „Script lief grün" reicht NICHT — Claude muss die Bilder wirklich anschauen.

### Schritt 5 — Iterieren
QA rot oder Screenshot zeigt ein Problem → fixen → Schritt 2–4 wiederholen, bis alles grün.

### Schritt 6 — Commit & Übergabe
- Ein Commit pro Phase (Conventional Commits, Englisch, **keine** AI-Attribution), push.
- An das Haupt-Team: Phasen-Nr, Commit-SHA, ausgefüllte Verify-Checkliste (RUNBOOK Abschnitt 3), Screenshots.

---

## Warum das die gleiche Gründlichkeit bringt
Das Haupt-Team macht **genau diese 6 Schritte** für jede Phase — gescopter tsc + Playwright-QA + **Screenshots wirklich ansehen** + iterieren. In dieser Schleife haben wir zuletzt mehrere echte Bugs gefangen (falsche i18n-Plurale, kaputte Mock-Handler, Typfehler), die ein reiner „kompiliert ja"-Check übersehen hätte. Wenn Nicos Claude dieselbe Schleife fährt, kommen vergleichbare Ergebnisse heraus.
