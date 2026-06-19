/**
 * QA — notifications N-5: sound toggle effective + demo-depth sweep.
 * Verifies: the "Testen" button actually drives the Web Audio chime (oscillator
 * spy), the toggle disables it; sweeps center/modal/empty states across DE+EN
 * and 1440/1024 collecting raw keys + console errors. Sub-terminal → :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/notif-n5')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const AUDIOSPY = `window.__chimes=0;try{const P=(window.AudioContext||window.webkitAudioContext)&&(window.AudioContext||window.webkitAudioContext).prototype;if(P&&P.createOscillator){const o=P.createOscillator;P.createOscillator=function(){const osc=o.call(this);const s=osc.start.bind(osc);osc.start=function(){window.__chimes++;try{return s.apply(osc,arguments)}catch(e){return undefined}};return osc};}}catch(e){}`
const localeScript = (l) => `try{localStorage.setItem('cosmi-locale',JSON.stringify({state:{locale:'${l}'},version:0}))}catch(e){}`
const RAW_RE = /([a-z]+\.[a-z]+\.[a-z]+|\{\{)/

async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('main *, .max-w-3xl *, [role="dialog"] *'))
      .filter((n) => n.children.length === 0 && rx.test((n.textContent || '').trim()))
      .map((n) => n.textContent.trim()).slice(0, 6)
  }, RAW_RE.source)
}
const clickCard = async (loc) => loc.evaluate((el) => el.click()).catch(() => loc.click({ timeout: 4000 }))

const browser = await chromium.launch()
const out = []

async function makeCtx({ width, height, locale }) {
  const ctx = await browser.newContext({ viewport: { width, height } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  await ctx.addInitScript(AUDIOSPY)
  if (locale) await ctx.addInitScript(localeScript(locale))
  return ctx
}

try {
  // ── Context A: DE @1440 — sound test + empty-unread ─────────────────────
  {
    const ctx = await makeCtx({ width: 1440, height: 1000, locale: 'de' })
    const page = await ctx.newPage(); const errs = []
    page.on('pageerror', (e) => errs.push(String(e)))
    page.on('console', (m) => { if (m.type() === 'error') errs.push('c:' + m.text()) })

    await page.goto(`${BASE}/#/notifications`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(3000)
    await page.screenshot({ path: resolve(outDir, 'de-1440-center.png') })
    const rkCenter = await rawKeys(page)

    // open settings overlay → sound section → Testen
    await page.locator('nav a:has-text("Modul-Einstellungen")').first().click({ timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(1200)
    await page.locator('button:has-text("Benachrichtigungen")').first().click({ timeout: 4000 }).catch(() => {})
    await page.waitForTimeout(1000)
    const soundSection = await page.locator('text=Ton bei neuen Benachrichtigungen').count()
    await page.locator('text=Ton bei neuen Benachrichtigungen').scrollIntoViewIfNeeded().catch(() => {})
    await page.screenshot({ path: resolve(outDir, 'de-1440-sound.png') })
    const chimesBefore = await page.evaluate(() => window.__chimes)
    await page.locator('button:has-text("Testen")').first().click({ timeout: 4000 }).catch(() => {})
    await page.waitForTimeout(500)
    const chimesAfter = await page.evaluate(() => window.__chimes)
    // toggle sound off → Testen disabled
    await page.locator('button[role="switch"]').last().click({ timeout: 4000 }).catch(() => {})
    await page.waitForTimeout(400)
    const testDisabled = await page.locator('button:has-text("Testen")').first().isDisabled().catch(() => null)
    out.push({ ctx: 'de-1440', soundSection, chimesBefore, chimesAfter, chimed: chimesAfter > chimesBefore, testDisabledWhenOff: testDisabled, rawKeysCenter: rkCenter })

    // close overlay, mark all read → unread tab empty
    await page.keyboard.press('Escape'); await page.waitForTimeout(400)
    await page.locator('button:has-text("Alle als gelesen markieren")').first().click({ timeout: 4000 }).catch(() => {})
    await page.waitForTimeout(1000)
    await page.locator('button:has-text("Ungelesen")').first().click({ timeout: 4000 }).catch(() => {})
    await page.waitForTimeout(1000)
    const emptyState = await page.locator('text=Alles gelesen, text=Keine ungelesenen').count().catch(() => 0)
    await page.screenshot({ path: resolve(outDir, 'de-1440-empty-unread.png') })
    out.push({ ctx: 'de-1440-empty', emptyUnreadShown: emptyState, pageErrors: errs.filter((e) => !/ERR_CONNECTION_REFUSED|websocket/i.test(e)).slice(0, 4) })
    await ctx.close()
  }

  // ── Context B: EN @1440 — english labels + modal ────────────────────────
  {
    const ctx = await makeCtx({ width: 1440, height: 1000, locale: 'en' })
    const page = await ctx.newPage(); const errs = []
    page.on('pageerror', (e) => errs.push(String(e)))
    page.on('console', (m) => { if (m.type() === 'error') errs.push('c:' + m.text()) })

    await page.goto(`${BASE}/#/notifications`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(3000)
    const enAllModules = await page.locator('text=All modules').count()
    const enSettings = await page.locator('button:has-text("Settings")').count()
    await page.screenshot({ path: resolve(outDir, 'en-1440-center.png') })
    const rkEn = await rawKeys(page)

    await clickCard(page.locator('.space-y-2 .cursor-pointer').first())
    await page.waitForTimeout(900)
    const enModalOpen = await page.locator('[role="dialog"]:has-text("Mark as read"), [role="dialog"] button:has-text("Open")').count()
    await page.screenshot({ path: resolve(outDir, 'en-1440-modal.png') })
    out.push({ ctx: 'en-1440', enAllModules, enSettings, enModalOpen, rawKeysEn: rkEn, pageErrors: errs.filter((e) => !/ERR_CONNECTION_REFUSED|websocket/i.test(e)).slice(0, 4) })
    await ctx.close()
  }

  // ── Context C: DE @1024 — responsive ────────────────────────────────────
  {
    const ctx = await makeCtx({ width: 1024, height: 800, locale: 'de' })
    const page = await ctx.newPage(); const errs = []
    page.on('pageerror', (e) => errs.push(String(e)))
    page.on('console', (m) => { if (m.type() === 'error') errs.push('c:' + m.text()) })

    await page.goto(`${BASE}/#/notifications`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(3000)
    await page.screenshot({ path: resolve(outDir, 'de-1024-center.png') })
    const rk1024 = await rawKeys(page)
    out.push({ ctx: 'de-1024', rawKeys1024: rk1024, pageErrors: errs.filter((e) => !/ERR_CONNECTION_REFUSED|websocket/i.test(e)).slice(0, 4) })
    await ctx.close()
  }
} catch (e) {
  out.push({ error: String(e).split('\n')[0] })
} finally {
  await browser.close()
}

console.log(JSON.stringify(out, null, 2))
