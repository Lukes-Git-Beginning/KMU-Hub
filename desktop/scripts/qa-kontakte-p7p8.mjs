import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, p) => (p === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
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
  return [...new Set([...text.matchAll(/\b(crm|advisory|kontakte|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const out = {}

try {
  await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(3000)

  // Open first contact -> detail modal
  await page.locator('div[role="button"].cursor-pointer').first().click({ timeout: 8000 })
  await page.waitForTimeout(1200)
  out.detailOpen = await page.getByText(/Segment [ABC]/).first().isVisible().catch(() => false)

  // Segment override popover
  await page.getByRole('button', { name: /Segment [ABC]/ }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(600)
  out.segPopover = await page.getByText('Segment manuell setzen').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'kontakte-segment-override.png'), fullPage: false })
  // pick A, then close popover
  await page.getByText('Segment A').first().click({ timeout: 3000 }).catch(() => {})
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(400)

  // Advisory: open tab -> new protocol -> editor -> PDF preview
  await page.getByRole('tab', { name: 'Beratungsprotokolle' }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(700)
  await page.getByRole('button', { name: 'Neues Protokoll' }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1500)
  out.editorOpen = await page.getByRole('button', { name: 'Als PDF / Drucken' }).first().isVisible().catch(() => false)
  await page.getByRole('button', { name: 'Als PDF / Drucken' }).first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(900)
  out.pdfPreview = await page.getByText('Geeignetheitserklärung').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'kontakte-advisory-pdf.png'), fullPage: false })
  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 5)
await ctx.close(); await browser.close()
console.log(JSON.stringify(out, null, 2))
