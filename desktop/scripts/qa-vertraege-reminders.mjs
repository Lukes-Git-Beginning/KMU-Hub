/**
 * V-3 QA — Fristen-Reminder als Notifications (idempotent)
 *
 * (a) /vertraege öffnen (Hook feuert) → genau 3 contract_expiry-Reminder im Store
 *     (v-3/v-5/v-11), im Notification-Center sichtbar mit gerendertem ICU-Plural
 *     ("… Tagen"), keine Raw-Keys/Doppelklammern.
 * (b) Reload + erneut /vertraege: weiterhin GENAU dieselben 3 IDs (keine Duplikate).
 * (c) "Auslaufend"-Tab ist nicht mehr leer (>= 3 Verträge).
 *
 * Idempotenz wird über den persistierten Store-State (localStorage) geprüft —
 * robuster als DOM-Zählung. Das DOM/Screenshot dient als visueller Beleg.
 *
 * Sub-Terminal: :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/vertraege-reminders')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

// Volle, eindeutige Titel (Substring darf nicht zugleich im Partner stecken)
const EXPIRING_TITLES = ['Microsoft 365 Business', 'Müller Metallbau Rahmenvertrag', 'Lagerraum Augsburg']

/** Liest die persistierten contract-expiry-Notification-IDs aus dem Store. */
function readExpiryIds(page) {
  return page.evaluate(() => {
    const raw = localStorage.getItem('cosmi-notifications')
    if (!raw) return []
    const j = JSON.parse(raw)
    const notifs = j.state?.notifications || j.notifications || []
    return notifs.filter((n) => String(n.id).startsWith('contract-expiry')).map((n) => n.id).sort()
  })
}

async function syncThenInspect(page) {
  await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1800) // Hook feuert + persist
  const ids = await readExpiryIds(page)
  await page.goto(`${BASE}/#/notifications`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  return ids
}

const browser = await chromium.launch()
const out = []

const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error') errs.push('console: ' + m.text()) })

try {
  // ── (a0) Notification-Center DIREKT (ohne Verträge-Besuch) ───────
  // Beweist, dass die Fristen-Reminder persistent aus der MSW-Quelle kommen
  // (Bell/Center/Dashboard lesen via useNotifications) — nicht nur als Toast.
  await page.goto(`${BASE}/#/notifications`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1800)
  const centerText = await page.evaluate(() => document.body.textContent || '')
  const titlesInCenter = EXPIRING_TITLES.filter((t) => centerText.includes(t))
  const centerPlural = /\bTagen?\b/.test(centerText)
  const centerRawKey = /vertraege\.notifications\.|notifications\.center\./.test(centerText)
  await page.screenshot({ path: resolve(outDir, 'a0-center-direct.png'), fullPage: false })
  out.push({
    step: 'a0-center-msw-direct',
    titlesInCenter,
    centerPlural,
    centerRawKey,
    pass: titlesInCenter.length === 3 && centerPlural && !centerRawKey,
  })

  // ── (a) erste Synchronisation (zustand-Toast-Pfad) ───────────────
  const firstIds = await syncThenInspect(page)
  const bodyText = await page.evaluate(() => document.body.textContent || '')
  const rawKey = /vertraege\.notifications\./.test(bodyText)
  const doubleBrace = /\{\{|\}\}/.test(bodyText)
  const pluralRendered = /\bTagen?\b/.test(bodyText)
  const titlesVisible = EXPIRING_TITLES.filter((t) => bodyText.includes(t))
  await page.screenshot({ path: resolve(outDir, 'a-notification-center.png'), fullPage: false })

  out.push({
    step: 'a-first-sync',
    expiryIds: firstIds,
    exactlyThree: firstIds.length === 3,
    uniqueIds: new Set(firstIds).size === firstIds.length,
    titlesVisible,
    rawKey,
    doubleBrace,
    pluralRendered,
    pass: firstIds.length === 3 && new Set(firstIds).size === 3 &&
      titlesVisible.length === 3 && !rawKey && !doubleBrace && pluralRendered && errs.length === 0,
    pageErrors: errs.slice(0, 5),
  })

  // ── (b) Reload → erneut synchronisieren → keine Duplikate ────────
  await page.reload({ waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(800)
  const secondIds = await syncThenInspect(page)
  await page.screenshot({ path: resolve(outDir, 'b-after-reload.png'), fullPage: false })
  const identical = secondIds.length === 3 &&
    JSON.stringify(secondIds) === JSON.stringify(firstIds)

  out.push({
    step: 'b-idempotent-reload',
    expiryIdsAfterReload: secondIds,
    identicalToFirst: identical,
    pass: identical,
  })

  // ── (c) Auslaufend-Tab nicht leer ────────────────────────────────
  await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const tabClicked = await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'))
    const tab = btns.find((b) => /Auslaufend|Expiring|Expirant|scadenza/i.test(b.textContent || ''))
    if (tab) { tab.click(); return true }
    return false
  })
  await page.waitForTimeout(1000)
  const expiringRows = await page.evaluate(() => document.querySelectorAll('table tbody tr').length)
  await page.screenshot({ path: resolve(outDir, 'c-auslaufend-tab.png'), fullPage: false })

  out.push({
    step: 'c-auslaufend-tab',
    tabClicked,
    expiringRows,
    pass: tabClicked && expiringRows >= 3,
  })
} catch (e) {
  out.push({ step: 'error', pass: false, error: String(e).split('\n')[0] })
} finally {
  await ctx.close()
}

await browser.close()

const allPass = out.every((s) => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== V-3 QA RESULT: ${allPass ? 'ALL PASS' : 'FAIL'} ===`)
out.forEach((s) => console.log(`  ${s.pass ? '✓' : '✗'} ${s.step}`))
process.exit(allPass ? 0 : 1)
