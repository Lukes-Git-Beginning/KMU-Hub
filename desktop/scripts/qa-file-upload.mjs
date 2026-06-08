// QA Chat file upload: existing message attachment renders, file picker adds
// a pending chip, sending fires the upload request, no raw keys / page errors.
import { chromium } from 'playwright'
import { mkdir, writeFile } from 'node:fs/promises'
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
const tmpFile = resolve(outDir, 'upload-sample.txt')
await writeFile(tmpFile, 'QA upload sample content')

const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
const errors = []
const uploadRequests = []
page.on('pageerror', (e) => errors.push(String(e)))
page.on('request', (r) => { if (r.url().includes('/api/v1/files/upload')) uploadRequests.push(r.method()) })

try {
  await page.goto(`${BASE}/#/kommunikation?bereich=team`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  await page.getByRole('button', { name: /^Team$/ }).first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(500)

  // Open allgemein → existing message has a PDF attachment
  await page.getByText('allgemein', { exact: false }).first().click({ timeout: 5000 }).catch((e) => { out.channelErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  out.existingAttachmentVisible = await page.getByText(/staging-zugang\.pdf/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'file-upload-existing.png'), fullPage: false })

  // Pick a file via the hidden input → pending chip appears
  await page.setInputFiles('input[type=file]', tmpFile).catch((e) => { out.setFilesErr = String(e).split('\n')[0] })
  await page.waitForTimeout(700)
  out.pendingChipVisible = await page.getByText(/upload-sample\.txt/).first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, 'file-upload-pending.png'), fullPage: false })

  // Send → fires upload request
  const composer = page.locator('textarea').last()
  await composer.fill('Hier die Datei').catch(() => {})
  await composer.press('Enter').catch(() => {})
  await page.waitForTimeout(1500)
  out.uploadRequestFired = uploadRequests.length > 0
  out.uploadMethod = uploadRequests[0] ?? null
  out.chipClearedAfterSend = !(await page.getByText(/upload-sample\.txt/).first().isVisible().catch(() => false))

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
