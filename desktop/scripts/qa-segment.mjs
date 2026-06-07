// QA P8-3: Segment-Badge im Kontakt-Header + Mandanten-Segmente in CRM-Settings.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const CONTACT = process.argv[2] || 'Karl Bauer'

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`
const SUPPRESS_ONBOARDING = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
`

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(advisory|kontakte|common|crm|settings|moduleSettings)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  await page.getByText(CONTACT, { exact: false }).first().click({ timeout: 5000 })
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'segment-detail.png') })
  out.headerText = await page.evaluate(() => {
    const h = document.querySelector('.bg-gradient-to-br')
    return h ? h.innerText.replace(/\n+/g, ' | ').slice(0, 220) : 'header-not-found'
  })
  out.detailRawKeys = await scanRawKeys(page)
  // close modal
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(500)

  // Open module settings overlay (bottom-left button) — CRM preselected on /kontakte
  await page.getByText(/Modul-Einstellungen/i).first().click({ timeout: 5000 })
  await page.waitForTimeout(1200)
  await page.getByText(/Mandanten-Segmente/i).first().scrollIntoViewIfNeeded().catch(() => {})
  await page.waitForTimeout(400)
  await page.screenshot({ path: resolve(outDir, 'segment-settings.png'), fullPage: true })
  out.segmentsText = await page.evaluate(() => {
    const els = [...document.querySelectorAll('section, div')].filter((d) => d.textContent?.includes('Mandanten-Segmente'))
    return els.length ? els[els.length - 1].innerText.slice(0, 260) : 'segments-not-found'
  })
  out.settingsRawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 3)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
