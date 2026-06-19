/**
 * Playwright interaction-QA — profil P-3 (avatar upload + DND toggle), :5174.
 *
 * Avatar: pick a synthetic PNG → routed through the demo MSW endpoint → image
 * appears + survives a reload (local persistence). DND: toggle flips state.
 * Run: node scripts/qa-profil-avatar-dnd.mjs
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/profil-avatar-dnd')
await mkdir(outDir, { recursive: true })

const STUB = `(function(){var noop=function(){return Promise.resolve()};var anyh={get:function(_t,p){return p==='then'?undefined:new Proxy(noop,anyh)},apply:function(){return Promise.resolve()}};var auth={getStoredTokens:function(){return Promise.resolve({accessToken:'d',refreshToken:'d'})},storeTokens:function(){return Promise.resolve()},clearTokens:function(){return Promise.resolve()}};var root={auth:auth};window.electronAPI=new Proxy(root,{get:function(t,p){return p in t?t[p]:(p==='then'?undefined:new Proxy(noop,anyh))}});})()`
const WS = `(function(){function F(){return{readyState:3,close:function(){},send:function(){},addEventListener:function(){},removeEventListener:function(){}}}window.WebSocket=F})()`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const LOC = `try{localStorage.setItem('cosmi-locale',JSON.stringify({state:{locale:'de'},version:0}))}catch(e){}`

// 1x1 transparent PNG
const PNG = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==', 'base64')

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

  // ── Avatar upload ──
  out.avatarImgBefore = await page.locator('img[src^="data:image"]').count()
  await page.locator('input[type="file"]').first().setInputFiles({ name: 'avatar.png', mimeType: 'image/png', buffer: PNG })
  await page.waitForTimeout(1500)
  out.avatarImgAfter = await page.locator('img[src^="data:image"]').count()
  await page.screenshot({ path: resolve(outDir, '1-avatar-set.png') })

  // ── Avatar survives reload (settings store persistence) ──
  await page.reload({ waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2500)
  out.avatarImgAfterReload = await page.locator('img[src^="data:image"]').count()
  await page.screenshot({ path: resolve(outDir, '2-avatar-after-reload.png') })

  // ── DND toggle ──
  const dnd = page.locator('button[role="switch"][aria-label="Bitte nicht stören"]').first()
  out.dndVisible = await dnd.isVisible().catch(() => false)
  out.dndDisabled = await dnd.isDisabled().catch(() => null)
  const before = await dnd.getAttribute('aria-checked').catch(() => null)
  await dnd.evaluate((el) => el.click()).catch(() => {})
  await page.waitForTimeout(800)
  const after = await dnd.getAttribute('aria-checked').catch(() => null)
  out.dndBefore = before
  out.dndAfter = after
  out.dndToggled = before !== after
  await page.screenshot({ path: resolve(outDir, '3-dnd-toggled.png') })
} catch (e) {
  out.error = String(e).split('\n')[0]
} finally {
  out.pageErrors = pageErrors.slice(0, 10)
  await ctx.close(); await browser.close()
}
console.log(JSON.stringify(out, null, 2))
