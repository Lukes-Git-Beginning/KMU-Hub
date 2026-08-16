// Screenshot-QA fuer Etappe 1: Helpdesk-Assignee (G0-7), Kalender-Buchungslink
// (G0-10) und das ausgeblendete Infrastruktur-Panel (Sidebar + Dashboard).
// Demo-Mode, Auth-Bypass=admin. Setzt cosmi:launch-played, um die
// Launch-Animation zu ueberspringen.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = process.env.QA_FE || 'http://localhost:5173'
const outDir = resolve('.qa-etappe1')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const PREP = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
  try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}
`

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(PREP)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

const shots = [
  { name: 'dashboard', hash: '#/', wait: 3200 },
  { name: 'helpdesk-list', hash: '#/helpdesk', wait: 3000 },
  { name: 'kalender-buchung', hash: '#/kalender', wait: 3000 },
  { name: 'infrastruktur-route', hash: '#/infrastruktur', wait: 2600 },
]

const results = []
for (const s of shots) {
  const before = errors.length
  try {
    await page.goto(`${FE}/${s.hash}`, { waitUntil: 'domcontentloaded', timeout: 30000 })
    await page.waitForTimeout(s.wait)
    const file = resolve(outDir, `${s.name}.png`)
    await page.screenshot({ path: file, fullPage: false })
    results.push({ shot: s.name, file, newErrors: errors.length - before })
  } catch (err) {
    results.push({ shot: s.name, error: String(err) })
  }
}

// Sidebar must no longer offer Infrastruktur anywhere.
await page.goto(`${FE}/#/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
await page.waitForTimeout(2600)
const infraLinks = await page.locator('a[href*="infrastruktur"]').count()
results.push({ check: 'sidebar+dashboard links to /infrastruktur', count: infraLinks, expected: 0 })

// Helpdesk: open the new-ticket dialog and read the assignee options.
try {
  await page.goto(`${FE}/#/helpdesk`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(3000)
  const newBtn = page.getByRole('button', { name: /neues ticket|new ticket/i }).first()
  if (await newBtn.count()) {
    await newBtn.click()
    await page.waitForTimeout(1600)
    await page.screenshot({ path: resolve(outDir, 'helpdesk-new-ticket.png'), fullPage: false })
    const opts = await page.locator('select').last().locator('option').allTextContents()
    results.push({ check: 'new-ticket assignee options', options: opts })
  } else {
    results.push({ check: 'new-ticket dialog', error: 'button not found' })
  }
} catch (err) {
  results.push({ check: 'new-ticket dialog', error: String(err) })
}

await browser.close()
console.log(JSON.stringify({ results, pageErrors: errors.slice(0, 5) }, null, 2))
