/**
 * DEMO (nicht QA) — Router-Tabs (Kontakte) vs. State-Tabs (Helpdesk) IM EDITOR.
 *
 * Zeigt Darien den Unterschied visuell:
 *   1. Normales Kontakte-Modul  → obere Reiter-Leiste ist da (sieht aus wie Tabs)
 *   2. Normales Helpdesk-Modul  → obere Reiter-Leiste ist da (sieht auch aus wie Tabs)
 *   3. Editor auf Kontakte      → in der Vorschau FEHLT die Reiter-Leiste (Router-Fall)
 *   4. Editor auf Helpdesk      → Reiter-Leiste DA + anklickbar (State-Fall)
 *   5. Editor Helpdesk: zweiten Reiter klicken → Ansicht wechselt (Beweis: funktioniert)
 *
 * Screenshots → .qa-screenshots/router-vs-state/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5175'
const outDir = resolve('.qa-screenshots/router-vs-state')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch({ headless: true })
const ctx = await b.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const shot = (n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
const wait = (ms) => page.waitForTimeout(ms)
const goHash = async (hash, settle = 2200) => {
  await page.evaluate((h) => { window.location.hash = h }, hash)
  await wait(settle)
}

try {
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await wait(2500)

  // 1 — normal Kontakte module (router tab bar)
  await goHash('#/kontakte')
  await shot('01-kontakte-normal.png')

  // 2 — normal Helpdesk module (state tab bar)
  await goHash('#/helpdesk')
  await shot('02-helpdesk-normal.png')

  // 3 — editor on Kontakte (router → no tab bar in preview)
  await goHash('#/editor-window?module=kontakte', 3000)
  await shot('03-editor-kontakte.png')

  // 4 — editor on Helpdesk (state → tab bar present + clickable)
  await goHash('#/editor-window?module=helpdesk', 3000)
  await shot('04-editor-helpdesk.png')

  // 5 — click the 2nd helpdesk tab inside the editor to prove it switches
  const tab = page.locator('button', { hasText: /Wissensdatenbank|Statistik/ }).first()
  if ((await tab.count()) > 0) {
    await tab.click()
    await wait(1200)
  }
  await shot('05-editor-helpdesk-tab-switched.png')

  console.log('Demo-Screenshots geschrieben nach', outDir)
} catch (e) {
  console.log('FEHLER:', String(e).slice(0, 400))
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}
