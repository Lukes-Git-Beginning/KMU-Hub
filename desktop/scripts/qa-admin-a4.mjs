// A-4 QA — admin/Branding on :5174, DE + EN. Captures form + live preview,
// then changes name + accent + uploads a logo to verify the preview updates.
import { chromium } from 'playwright'
import { mkdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = process.env.QA_FE || 'http://localhost:5174'
const outDir = resolve('.qa-admin-a4')
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const prep = (locale) => `
  try { const K='cosmi-ui'; const r=localStorage.getItem(K); const p=r?JSON.parse(r):{state:{},version:0}; p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(K,JSON.stringify(p)); } catch(e){}
  try { sessionStorage.setItem('cosmi:launch-played','1') } catch(e){}
  try { localStorage.setItem('cosmi-locale', JSON.stringify({state:{locale:'${locale}'},version:0})) } catch(e){}
`

// A tiny SVG logo as a temp upload file.
const logoSvg = `<svg xmlns="http://www.w3.org/2000/svg" width="120" height="32"><rect width="120" height="32" rx="6" fill="#6366F1"/><text x="60" y="21" font-family="sans-serif" font-size="14" fill="#fff" text-anchor="middle" font-weight="700">ACME</text></svg>`
const logoPath = resolve(outDir, '_logo.svg')

await mkdir(outDir, { recursive: true })
await writeFile(logoPath, logoSvg, 'utf8')
const browser = await chromium.launch()

for (const locale of ['de', 'en']) {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(prep(locale))
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  const shot = async (n) => { await page.waitForTimeout(450); await page.screenshot({ path: resolve(outDir, `${locale}-${n}.png`) }) }

  await page.goto(`${FE}/#/admin/branding`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(2400)
  await shot('01-default')

  // Change workspace name
  await page.locator('#brand-name').fill('ACME GmbH')
  // Pick the indigo accent swatch (#6366F1) and upload a logo
  await page.evaluate(() => {
    const b = [...document.querySelectorAll('button[aria-label="#6366F1"]')][0]
    if (b) b.click()
  })
  await page.waitForTimeout(300)
  const fileInput = page.locator('input[type="file"]').first()
  await fileInput.setInputFiles(logoPath).catch(() => {})
  await page.waitForTimeout(600)
  await shot('02-customized')

  console.log(`[${locale}] pageerrors=${errors.length}`, errors.slice(0, 2))
  await ctx.close()
}
await browser.close()
console.log('done')
