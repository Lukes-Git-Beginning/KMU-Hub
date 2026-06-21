// QA — dialer D-1 CTI/CRM bridge: call activity -> contact timeline + "Im CRM öffnen" (dev :5174)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const API = 'http://localhost:8080'
const outDir = resolve('.qa-screenshots/dialer-d1')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
// reset persisted dialer store so we always start at idle/no-campaign
const RESET = `try{localStorage.removeItem('cosmi-dialer')}catch(e){}`

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1500, height: 940 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
await ctx.addInitScript(RESET)
const page = await ctx.newPage()
page.setDefaultTimeout(9000)
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const bodyText = () => page.evaluate(() => document.body.textContent || '')
const step = async (name, fn) => {
  try { await fn() } catch (err) { out[`ERR_${name}`] = String(err).split('\n')[0] }
}

await page.goto(`${BASE}/#/dialer/workspace`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500)

// ── Idle: pick the active campaign ──
await step('idle', async () => {
  await page.screenshot({ path: resolve(outDir, 'd1-0-idle.png'), fullPage: false })
  const camp = page.getByRole('button').filter({ hasText: /Kalt-Akquise/ }).first()
  if (await camp.count()) { await camp.click(); await page.waitForTimeout(900) }
})

// ── Load next -> dialing phase (shows "Im CRM öffnen") ──
await step('dialing', async () => {
  await page.getByRole('button', { name: /Nächsten Kontakt laden/ }).click()
  await page.waitForTimeout(1200)
  const body = await bodyText()
  out.dialingShowsContact = /Schneider|Huber|Bauer|Wagner|Berger/.test(body)
  out.openInCrmShown = /Im CRM öffnen/.test(body)
  out.callBtnTranslated = /Anrufen/.test(body) && !/dialer\.workspace/.test(body)
  await page.screenshot({ path: resolve(outDir, 'd1-1-dialing.png'), fullPage: false })
})

// ── Dial -> on_call ──
await step('onCall', async () => {
  await page.getByRole('button', { name: /^Anrufen$/ }).click()
  await page.waitForTimeout(1200)
  out.onCallOpenInCrm = /Im CRM öffnen/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'd1-2-oncall.png'), fullPage: false })
})

// ── Hang up -> wrap up, choose outcome, complete ──
await step('wrapUp', async () => {
  await page.getByRole('button', { name: /Auflegen/ }).click()
  await page.waitForTimeout(1000)
  await page.screenshot({ path: resolve(outDir, 'd1-3-wrapup.png'), fullPage: false })
  // pick an outcome card
  const outcome = page.getByText('Erreicht', { exact: true }).first()
  if (await outcome.count()) { await outcome.click(); await page.waitForTimeout(500) }
  await page.getByRole('button', { name: /^Abschließen$/ }).click()
  await page.waitForTimeout(1500)
  out.backToIdle = /Nächsten Kontakt laden|Aktive Kampagne/.test(await bodyText())
})

// ── Verify the activity landed on the contact timeline (MSW) ──
await step('timelineApi', async () => {
  // cc004 (first pending) = Anna Schneider = ct-004
  const data = await page.evaluate(async (api) => {
    const r = await fetch(`${api}/api/v1/crm/contacts/ct-004/timeline`)
    return r.json()
  }, API)
  out.timelineTotal = data.total
  out.timelineHasCall = (data.events || []).some((e) => /Telefonanruf/.test(e.title || ''))
  out.timelineFirstTitle = data.events?.[0]?.title ?? null
})

// ── Visit the contact in CRM and screenshot the timeline ──
await step('crmTimeline', async () => {
  await page.goto(`${BASE}/#/kontakte?contact=ct-004`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2500)
  out.crmShowsCall = /Telefonanruf/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'd1-4-crm-timeline.png'), fullPage: true })
})

out.pageErrors = errs.slice(0, 8)
out.rawKeys = await page.evaluate(() => {
  const all = Array.from(document.querySelectorAll('body *'))
    .filter((n) => n.children.length === 0)
    .map((n) => (n.textContent || '').trim())
  return [...new Set(all.filter((t) => /^(dialer|common|shared)\.[a-zA-Z]/.test(t)))].slice(0, 12)
})
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
