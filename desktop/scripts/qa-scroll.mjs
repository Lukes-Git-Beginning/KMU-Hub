// QA: öffnet Kontakt-Detail (schmal/einspaltig), scrollt die Detail-Spalte
// schrittweise durch und screenshottet, um untere Sektionen zu verifizieren.
import { chromium } from 'playwright'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const CONTACT = process.argv[2] || 'Hans Müller'

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`
const SUPPRESS_ONBOARDING = `
  try { const KEY='cosmi-ui'; const raw=localStorage.getItem(KEY); const p=raw?JSON.parse(raw):{state:{},version:0}; p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(KEY,JSON.stringify(p)) } catch(e){}
`

const browser = await chromium.launch()
// schmal = einspaltig, alle Sektionen vertikal gestapelt
const ctx = await browser.newContext({ viewport: { width: 600, height: 900 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const out = {}
try {
  await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  await page.getByText(CONTACT, { exact: false }).first().click({ timeout: 5000 })
  await page.waitForTimeout(1500)
  // Finde den scrollbaren Detail-Container und scrolle in Schritten
  for (let i = 0; i < 3; i++) {
    await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('div'))
      const scrollable = els.filter((el) => el.scrollHeight > el.clientHeight + 40 && getComputedStyle(el).overflowY !== 'visible')
      // den innersten/größten scrollbaren nehmen
      const target = scrollable.sort((a, b) => b.scrollHeight - a.scrollHeight)[0]
      if (target) target.scrollTop += target.clientHeight * 0.85
    })
    await page.waitForTimeout(700)
    await page.screenshot({ path: resolve(outDir, `kontakt-360-scroll${i}__narrow.png`) })
  }
  out.errors = errors.length
  out.errorMsgs = errors.slice(0, 2)
} catch (err) {
  out.error = String(err).split('\n')[0]
  out.errors = errors.length
}
await browser.close()
console.log(JSON.stringify(out, null, 2))
