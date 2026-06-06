// QA-Screenshot-Werkzeug — rendert die laufende Demo-App (localhost:5173) per
// Playwright-Chromium und schießt Screenshots pro Route × Fenstergröße.
// Nutzung: node scripts/qa-shot.mjs "kontakte,crm" [outDir]
// HashRouter → Routen werden als #/<route> aufgerufen.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const routes = (process.argv[2] || 'kontakte').split(',').map((r) => r.trim())
const outDir = resolve(process.argv[3] || '.qa-screenshots')

const SIZES = [
  { name: 'full', width: 1440, height: 900 },
  { name: 'half', width: 720, height: 900 },
  { name: 'small', width: 500, height: 800 },
]

// Stub fuer die Electron-Preload-Bridge (im reinen Chromium nicht vorhanden).
// Generischer Proxy: jeder Zugriff liefert eine no-op-Funktion, die ein
// aufgeloestes Promise zurueckgibt — verhindert Boot-Crashes bei IPC-Aufrufen.
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`

// Onboarding-Wizard vorab als abgeschlossen markieren (useUIStore, persist-Key
// 'cosmi-ui'), damit das "Willkommen bei Cosmi"-Modal die Screenshots nicht
// verdeckt. version weggelassen → zustand merged ueber den initialState.
const SUPPRESS_ONBOARDING = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
`

// Fallback: falls das Modal trotzdem rendert, "Ueberspringen"/"Skip" klicken.
async function dismissOnboarding(page) {
  for (const name of [/überspringen/i, /skip/i]) {
    const btn = page.getByRole('button', { name }).first()
    try {
      if (await btn.isVisible({ timeout: 500 })) {
        await btn.click()
        await page.waitForTimeout(300)
        return
      }
    } catch (e) {}
  }
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const results = []

for (const route of routes) {
  const slug = route.replace(/[^a-z0-9]+/gi, '-')
  for (const size of SIZES) {
    const ctx = await browser.newContext({ viewport: { width: size.width, height: size.height } })
    await ctx.addInitScript(ELECTRON_STUB)
    await ctx.addInitScript(SUPPRESS_ONBOARDING)
    const page = await ctx.newPage()
    const errors = []
    page.on('pageerror', (e) => errors.push(String(e)))
    try {
      await page.goto(`${BASE}/#/${route}`, { waitUntil: 'domcontentloaded', timeout: 20000 })
      await page.waitForTimeout(1500)
      await dismissOnboarding(page)
      await page.waitForTimeout(1000) // Daten/Render absetzen lassen
      const file = resolve(outDir, `${slug}__${size.name}.png`)
      await page.screenshot({ path: file, fullPage: false })
      results.push({ route, size: size.name, file, errors: errors.length })
    } catch (err) {
      results.push({ route, size: size.name, error: String(err), errors: errors.length })
    } finally {
      await ctx.close()
    }
  }
}

await browser.close()
console.log(JSON.stringify(results, null, 2))
