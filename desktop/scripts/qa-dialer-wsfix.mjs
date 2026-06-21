// QA — dialer workspace blank-screen fix: phase-without-contact recovers to idle (:5173)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/dialer-wsfix')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1400, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
page.setDefaultTimeout(9000)
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const bodyText = () => page.evaluate(() => document.body.textContent || '')
const wsContentLen = () =>
  page.evaluate(() => {
    // text length of the phase content area (below the dialer tab bar)
    const el = document.querySelector('.animate-scale-in') || document.querySelector('main')
    return (el?.textContent || '').trim().length
  })
const step = async (name, fn) => {
  try { await fn() } catch (err) { out[`ERR_${name}`] = String(err).split('\n')[0] }
}

await page.goto(`${BASE}/#/dialer/workspace`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3200)
out.idleRendersFresh = (await wsContentLen()) > 50

// Drive into a non-idle phase: available → pick campaign → load next contact.
await step('enterCall', async () => {
  // Set agent available
  const pill = page.locator('button').filter({ hasText: /Verfügbar|Offline|Pause|Abwesend|Status/ }).first()
  if (await pill.count()) {
    await pill.click()
    await page.waitForTimeout(400)
    const avail = page.getByText('Verfügbar', { exact: false }).last()
    if (await avail.count()) { await avail.click(); await page.waitForTimeout(500) }
  }
  // Pick the demo campaign
  const camp = page.getByText('Kalt-Akquise Kampagne Q2/2026', { exact: false }).first()
  if (await camp.count()) { await camp.click(); await page.waitForTimeout(600) }
  // Load next contact
  const load = page.getByText('Nächsten Kontakt laden', { exact: false }).first()
  if (await load.count()) { await load.click(); await page.waitForTimeout(1200) }
  out.inCallPhase = /verbindet|Anrufen|Auflegen|Gespräch|Notizen/i.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'wsfix-1-incall.png'), fullPage: false })
})

// Leave the workspace and return — this remounts the page, dropping activeContact.
await step('roundTrip', async () => {
  await page.getByText('Dashboard', { exact: true }).first().click()
  await page.waitForTimeout(1200)
  await page.getByText('Workspace', { exact: true }).first().click()
  await page.waitForTimeout(1500)
  out.afterReturnContentLen = await wsContentLen()
  out.notBlank = (await wsContentLen()) > 50
  out.recoveredToIdle = /Verfügbar|Kampagne|Nächsten Kontakt laden|HEUTE|Anrufe/i.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'wsfix-2-returned.png'), fullPage: false })
})

out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
