/**
 * Playwright Screenshot-QA — profil module (parallel-batch sub-terminal, :5174).
 *
 * Drives the QA-only Vite web server (vite.qa.config.mjs) with demo-mode MSW on.
 * Seeds demo auth tokens so the auth store loads the real demo user
 * (Stefan Vogel / usr-e1 via /auth/me) app-wide — same identity the sidebar/
 * topbar show — and stubs WebSocket to keep the console clean.
 *
 * Captures all 4 profil tabs at 1440 and 1024, scans for raw i18n keys and
 * double-brace interpolation leaks, and collects console/page errors.
 *
 * Usage:  node scripts/qa-profil.mjs [de|en]
 *   lang arg (default de) forces the locale via the cosmi-locale store.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const LANG = (process.argv[2] === 'en' || process.argv[2] === 'fr' || process.argv[2] === 'it') ? process.argv[2] : 'de'
const outDir = resolve('.qa-screenshots/profil')
await mkdir(outDir, { recursive: true })

// electronAPI stub: getStoredTokens returns demo tokens so auth/me is fetched
// (→ Stefan Vogel). Any other electronAPI.* call resolves to a no-op promise.
const STUB = `(function(){
  var noop=function(){return Promise.resolve()};
  var anyh={get:function(_t,p){return p==='then'?undefined:new Proxy(noop,anyh)},apply:function(){return Promise.resolve()}};
  var auth={
    getStoredTokens:function(){return Promise.resolve({accessToken:'demo-access-token-000',refreshToken:'demo-refresh-token-000'})},
    storeTokens:function(){return Promise.resolve()},
    clearTokens:function(){return Promise.resolve()}
  };
  var root={auth:auth};
  window.electronAPI=new Proxy(root,{get:function(t,p){return p in t ? t[p] : (p==='then'?undefined:new Proxy(noop,anyh))}});
})()`

// Stub WebSocket — the demo backend has no WS endpoint; keep the console clean.
const WS_STUB = `(function(){
  function FakeWS(){return {readyState:3,close:function(){},send:function(){},addEventListener:function(){},removeEventListener:function(){}}}
  window.WebSocket=FakeWS;
})()`

// Skip onboarding overlay.
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

// Force locale via the cosmi-locale persisted store.
const LOCALE = `try{localStorage.setItem('cosmi-locale', JSON.stringify({state:{locale:'${LANG}'},version:0}))}catch(e){}`

// Raw-key / double-brace leak scan over visible text nodes.
const RAW_RE = /(profil\.[a-z]|api\.hr\.|\{\{)/i
async function scanLeaks(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src, 'i')
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => (n.textContent || '').trim())
      .slice(0, 15)
  }, RAW_RE.source)
}

const TABS = ['profil', 'zeiterfassung', 'abwesenheiten', 'dokumente']

async function shoot(page, viewport, tag) {
  await page.setViewportSize(viewport)
  const results = {}
  for (let i = 0; i < TABS.length; i++) {
    const tabBtn = page.locator('[role="tab"]').nth(i)
    await tabBtn.evaluate((el) => el.click()).catch(() => {})
    await page.waitForTimeout(1400)
    await page.screenshot({ path: resolve(outDir, `${tag}-${i + 1}-${TABS[i]}.png`) })
    results[TABS[i]] = await scanLeaks(page)
  }
  return results
}

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(WS_STUB)
await ctx.addInitScript(ONB)
await ctx.addInitScript(LOCALE)

const page = await ctx.newPage()
const consoleErrors = []
const pageErrors = []
page.on('pageerror', (e) => pageErrors.push(String(e).split('\n')[0]))
page.on('console', (m) => { if (m.type() === 'error') consoleErrors.push(m.text().slice(0, 200)) })

const out = { lang: LANG, leaks: {}, bodySample: {} }
try {
  await page.goto(`${BASE}/#/profil`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2800)

  out.leaks['1440'] = await shoot(page, { width: 1440, height: 950 }, '1440')
  out.leaks['1024'] = await shoot(page, { width: 1024, height: 800 }, '1024')

  // Identity sanity (profil tab): capture key body text + full-page shot so the
  // account-info ("Mitglied seit") section below the fold is visible.
  await page.setViewportSize({ width: 1440, height: 950 })
  await page.locator('[role="tab"]').nth(0).evaluate((el) => el.click()).catch(() => {})
  await page.waitForTimeout(1000)
  await page.screenshot({ path: resolve(outDir, 'profil-fullpage.png'), fullPage: true })
  const body = await page.evaluate(() => document.body.innerText)
  out.bodySample = {
    hasStefanVogel: /Stefan Vogel/.test(body),
    hasGeschaeftsfuehrer: /Geschäftsführer|Managing Director|Geschaeftsfuehrer/.test(body),
    hasStefanEmail: /stefan\.vogel@techvision\.de/.test(body),
    hasMemberSince: /Mitglied seit|Member since|Membre depuis|Membro dal/.test(body),
    leftoverDarien: /Darien|Morales/.test(body),
  }
} catch (e) {
  out.error = String(e).split('\n')[0]
} finally {
  out.consoleErrors = consoleErrors.slice(0, 20)
  out.pageErrors = pageErrors.slice(0, 20)
  await ctx.close()
  await browser.close()
}

console.log(JSON.stringify(out, null, 2))
