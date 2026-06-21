// QA — dialer D-2 supervisor dashboard: agent roster + recent calls log (dev :5174)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/dialer-d2')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1500, height: 940 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
page.setDefaultTimeout(9000)
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const bodyText = () => page.evaluate(() => document.body.textContent || '')

await page.goto(`${BASE}/#/dialer/supervisor`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500)

const body = await bodyText()
out.navHasSupervisor = /Supervisor/.test(body)
out.agentsShown = /Thomas Meier/.test(body) && /Sabine Fischer/.test(body) && /Kevin Baumann/.test(body)
out.statusImGespraech = /Im Gespräch/.test(body)
out.statusPause = /Pause/.test(body)
out.recentCallsHeading = /Letzte Anrufe/.test(body)
out.recentHasContacts = /Friedrich Gruber|Hans Müller|Claudia Weber/.test(body)
out.recentHasOutcomes = /Termin vereinbart|Erreicht|Nicht erreicht|Wiedervorlage/.test(body)
out.kpiActiveAgents = /Aktive Agenten/.test(body)
out.kpiAppointments = /Termine heute/.test(body)
out.relativeTimeShown = /vor \d+ Min\.|vor \d+ Std\./.test(body)
out.noRawKeys = !/dialer\.supervisor\.|dialer\.nav\.supervisor/.test(body)
await page.screenshot({ path: resolve(outDir, 'd2-1-supervisor.png'), fullPage: true })

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
