/**
 * Playwright QA — UserProfileCard (profil P-5), :5174.
 *
 * The profile card popover is reachable wherever UserProfileTrigger wraps an
 * avatar (dashboard TeamStatus, chat, work). Verifies: popover opens, presence +
 * "Nachricht senden" present, and the action navigates to the DM (kommunikation).
 * Run: node scripts/qa-profil-card.mjs
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/profil-card')
await mkdir(outDir, { recursive: true })

const STUB = `(function(){var noop=function(){return Promise.resolve()};var anyh={get:function(_t,p){return p==='then'?undefined:new Proxy(noop,anyh)},apply:function(){return Promise.resolve()}};var auth={getStoredTokens:function(){return Promise.resolve({accessToken:'d',refreshToken:'d'})},storeTokens:function(){return Promise.resolve()},clearTokens:function(){return Promise.resolve()}};var root={auth:auth};window.electronAPI=new Proxy(root,{get:function(t,p){return p in t?t[p]:(p==='then'?undefined:new Proxy(noop,anyh))}});})()`
const WS = `(function(){function F(){return{readyState:3,close:function(){},send:function(){},addEventListener:function(){},removeEventListener:function(){}}}window.WebSocket=F})()`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const LOC = `try{localStorage.setItem('cosmi-locale',JSON.stringify({state:{locale:'de'},version:0}))}catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(WS); await ctx.addInitScript(ONB); await ctx.addInitScript(LOC)
const page = await ctx.newPage()
const pageErrors = []
page.on('pageerror', (e) => pageErrors.push(String(e).split('\n')[0]))
const out = {}
try {
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  // Reveal the TeamStatus widget (team view + scroll)
  await page.locator('button:has-text("Team")').first().evaluate((el) => el.click()).catch(() => {})
  await page.waitForTimeout(1200)
  await page.mouse.wheel(0, 1600)
  await page.waitForTimeout(800)

  const triggers = page.locator('button.rounded-full.bg-primary\\/10')
  out.triggerCount = await triggers.count()
  const card = page.locator('[data-testid="user-profile-card"]')

  // Click triggers until one yields an "other user" card (has "Nachricht senden").
  out.cardVisible = false
  out.hasSendMessage = false
  out.hasPresence = false
  const n = Math.min(out.triggerCount, 8)
  for (let i = 0; i < n; i++) {
    await triggers.nth(i).evaluate((el) => el.click()).catch(() => {})
    await page.waitForTimeout(700)
    const vis = await card.isVisible().catch(() => false)
    if (vis) {
      out.cardVisible = true
      out.hasPresence = out.hasPresence || await card.evaluate((el) => /Online|Abwesend|Nicht stören|Im Anruf|Offline/i.test(el.textContent || '')).catch(() => false)
      const send = await card.locator('button:has-text("Nachricht senden")').first().isVisible().catch(() => false)
      if (send) { out.hasSendMessage = true; break }
    }
    await page.keyboard.press('Escape').catch(() => {})
    await page.waitForTimeout(200)
  }
  await page.screenshot({ path: resolve(outDir, '1-card-open.png') })

  // Click "Nachricht senden" → should navigate to kommunikation (DM intent)
  const sendBtn = card.locator('button:has-text("Nachricht senden")').first()
  if (await sendBtn.isVisible().catch(() => false)) {
    await sendBtn.evaluate((el) => el.click())
    await page.waitForTimeout(1800)
  }
  out.urlAfterSend = page.url()
  out.navigatedToKommunikation = /kommunikation/.test(page.url())
  await page.screenshot({ path: resolve(outDir, '2-after-send.png') })
} catch (e) {
  out.error = String(e).split('\n')[0]
} finally {
  out.pageErrors = pageErrors.slice(0, 10)
  await ctx.close(); await browser.close()
}
console.log(JSON.stringify(out, null, 2))
