// QA KO-10: holistic depth pass across the whole kommunikation module.
// Walks Team chat (channels, DMs, panels) + Posteingang and scans for raw i18n
// keys and page errors across every view.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')

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
  return [...new Set([...text.matchAll(/\b(chat|common|kommunikation)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = { rawKeysByView: {} }
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  // Team chat
  await page.goto(`${BASE}/#/kommunikation?bereich=team`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(2800)
  await page.getByText('allgemein', { exact: false }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(900)
  out.rawKeysByView.teamChannel = await scanRawKeys(page)
  await page.screenshot({ path: resolve(outDir, 'ko10-1-team.png'), fullPage: false })

  // Cycle the inbox panels
  for (const [label, name] of [['unread', 'Alle ungelesenen'], ['mentions', 'Erwähnungen'], ['bookmarks', 'Lesezeichen']]) {
    await page.getByRole('button', { name: new RegExp(name) }).first().click({ timeout: 4000 }).catch(() => {})
    await page.waitForTimeout(500)
    out.rawKeysByView[label] = await scanRawKeys(page)
  }
  await page.screenshot({ path: resolve(outDir, 'ko10-2-panels.png'), fullPage: false })

  // DM (1:1)
  await page.getByText('Julia Hoffmann', { exact: false }).first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(700)
  out.rawKeysByView.dm = await scanRawKeys(page)

  // Posteingang
  await page.goto(`${BASE}/#/kommunikation?bereich=posteingang`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(2000)
  await page.locator('button').filter({ has: page.locator('.bg-primary-light') }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(900)
  out.rawKeysByView.posteingang = await scanRawKeys(page)
  await page.screenshot({ path: resolve(outDir, 'ko10-3-posteingang.png'), fullPage: false })

  // Aggregate
  out.allRawKeys = [...new Set(Object.values(out.rawKeysByView).flat())]
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
