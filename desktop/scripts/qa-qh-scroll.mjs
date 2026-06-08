// Reproduce the real scroll behaviour: scroll so the quiet-hours toggle is visible,
// then capture viewport (not fullPage) screenshots in both toggle states + measure
// whether an anchor ABOVE quiet hours stays put in the viewport.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/qh-scroll')
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, p) => (p === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`
const SUPPRESS = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 760 } }) // shorter viewport → page scrolls
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  await page.goto(`${BASE}/#/settings`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2000)
  await page.getByText('Benachrichtigungen', { exact: false }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(800)

  const qhSwitch = page.locator('button[role="switch"]').last()
  // ensure active so the options are shown first
  if ((await qhSwitch.getAttribute('aria-checked')) !== 'true') { await qhSwitch.click(); await page.waitForTimeout(500) }

  // Scroll the quiet-hours toggle into the middle of the viewport (mimics the user being there)
  await qhSwitch.scrollIntoViewIfNeeded()
  await page.waitForTimeout(400)

  // Anchor above quiet hours: the "Stummgeschaltet" heading
  const anchor = page.getByText('Stummgeschaltet', { exact: false }).first()
  const yActive = (await anchor.boundingBox().catch(() => null))?.y
  await page.screenshot({ path: resolve(outDir, 'A-active.png') }) // viewport only

  // Deactivate — options collapse
  await qhSwitch.click()
  await page.waitForTimeout(700)
  const yInactive = (await anchor.boundingBox().catch(() => null))?.y
  await page.screenshot({ path: resolve(outDir, 'B-inactive.png') })

  out.anchorYActive = yActive
  out.anchorYInactive = yInactive
  out.anchorMovedPx = (yActive != null && yInactive != null) ? Math.round(Math.abs(yActive - yInactive)) : null
  out.scrollTop = await page.evaluate(() => {
    const sc = [...document.querySelectorAll('*')].find((el) => el.scrollHeight > el.clientHeight + 40 && getComputedStyle(el).overflowY !== 'visible')
    return sc ? sc.scrollTop : window.scrollY
  })
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 4)
await ctx.close(); await browser.close()
console.log(JSON.stringify(out, null, 2))
