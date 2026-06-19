/**
 * Playwright interaction-QA — profil Dokumente-Tab (P-2), :5174.
 *
 * Exercises the MSW-backed documents flow: list renders, row click opens the
 * DetailModal preview, upload dialog accepts a (synthetic) file and the new
 * document appears in the list. Run: node scripts/qa-profil-docs.mjs
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/profil-docs')
await mkdir(outDir, { recursive: true })

const STUB = `(function(){var noop=function(){return Promise.resolve()};var anyh={get:function(_t,p){return p==='then'?undefined:new Proxy(noop,anyh)},apply:function(){return Promise.resolve()}};var auth={getStoredTokens:function(){return Promise.resolve({accessToken:'d',refreshToken:'d'})},storeTokens:function(){return Promise.resolve()},clearTokens:function(){return Promise.resolve()}};var root={auth:auth};window.electronAPI=new Proxy(root,{get:function(t,p){return p in t?t[p]:(p==='then'?undefined:new Proxy(noop,anyh))}});})()`
const WS = `(function(){function F(){return{readyState:3,close:function(){},send:function(){},addEventListener:function(){},removeEventListener:function(){}}}window.WebSocket=F})()`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const LOC = `try{localStorage.setItem('cosmi-locale',JSON.stringify({state:{locale:'de'},version:0}))}catch(e){}`

const RAW_RE = /(profil\.[a-z]|api\.hr\.|\{\{)/i
async function scanLeaks(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src, 'i')
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => (n.textContent || '').trim()).slice(0, 15)
  }, RAW_RE.source)
}

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(WS); await ctx.addInitScript(ONB); await ctx.addInitScript(LOC)
const page = await ctx.newPage()
const pageErrors = []
page.on('pageerror', (e) => pageErrors.push(String(e).split('\n')[0]))
const out = {}
try {
  await page.goto(`${BASE}/#/profil`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2500)
  // Dokumente tab (index 3)
  await page.locator('[role="tab"]').nth(3).evaluate((el) => el.click())
  await page.waitForTimeout(1500)

  const rows = page.locator('div[role="button"][aria-label^="Dokument öffnen"]')
  out.docCountInitial = await rows.count()
  await page.screenshot({ path: resolve(outDir, '1-list.png') })

  // Open preview on the first document row
  await rows.first().evaluate((el) => el.click())
  await page.waitForTimeout(900)
  out.previewBadgeVisible = await page.locator('text=Demo-Vorschau').first().isVisible().catch(() => false)
  out.previewHintVisible = await page.locator('text=Produktivbetrieb').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, '2-preview-modal.png') })
  // Close modal (Escape)
  await page.keyboard.press('Escape')
  await page.waitForTimeout(500)

  // Upload flow
  await page.locator('button:has-text("Hochladen")').first().evaluate((el) => el.click())
  await page.waitForTimeout(700)
  out.uploadDialogVisible = await page.locator('text=Dokument hochladen').first().isVisible().catch(() => false)
  await page.locator('input[type="file"]').setInputFiles({
    name: 'Test-Zeugnis_2026.pdf',
    mimeType: 'application/pdf',
    buffer: Buffer.from('%PDF-1.4 demo content for QA'),
  })
  await page.waitForTimeout(400)
  await page.screenshot({ path: resolve(outDir, '3-upload-dialog.png') })
  // Submit (the dialog footer "Hochladen" button — last match)
  await page.locator('button:has-text("Hochladen")').last().evaluate((el) => el.click())
  await page.waitForTimeout(1500)
  out.docCountAfter = await page.locator('div[role="button"][aria-label^="Dokument öffnen"]').count()
  out.newDocVisible = await page.locator('text=Test-Zeugnis_2026.pdf').first().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, '4-after-upload.png') })

  out.leaks = await scanLeaks(page)
} catch (e) {
  out.error = String(e).split('\n')[0]
} finally {
  out.pageErrors = pageErrors.slice(0, 10)
  await ctx.close(); await browser.close()
}
console.log(JSON.stringify(out, null, 2))
