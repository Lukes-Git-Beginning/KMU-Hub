/**
 * Smoke — X-4 store splits (automatisierung/mails/formulare/berichte).
 * Pure plumbing (personal + tenant scope split, backend-persisted). Nothing
 * changes visually; this only confirms the split consumers render without
 * crashing (personal pages + settings-overlay tenant sections).
 */
import { chromium } from 'playwright'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))

const out = []
// 1) Personal-store pages render (FormularePage consumes 5 tenant fields for consent)
for (const route of ['formulare', 'automatisierung', 'berichte', 'mails']) {
  const before = errs.length
  await page.goto(`${BASE}/#/${route}`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1800)
  const bodyLen = (await page.evaluate(() => document.body.textContent || '')).length
  out.push({ step: `page:${route}`, rendered: bodyLen > 500, newErrors: errs.length - before, pass: bodyLen > 500 && errs.length === before })
}

// 2) Settings overlay per module → tenant section renders (personal + tenant split)
const panelChecks = [
  { route: 'formulare', tenant: /Einwilligung|Datenschutz|Aufbewahrung|Benachrichtigung/ },
  { route: 'automatisierung', tenant: /Aufbewahrung|Fehlschlag|Protokoll|Benachrichtig/ },
  { route: 'berichte', tenant: /Format|Domain|Empfänger|erlaubt/i },
  { route: 'mails', tenant: /Aufbewahrung|externe Bilder|Badge/ },
]
for (const c of panelChecks) {
  const before = errs.length
  await page.goto(`${BASE}/#/${c.route}`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  await page.evaluate(() => {
    const el = Array.from(document.querySelectorAll('button, a, [role="button"]')).find((e) => /Modul-Einstellung/.test(e.textContent || ''))
    el?.click()
  })
  await page.waitForTimeout(2000)
  const txt = await page.evaluate(() => document.body.textContent || '')
  out.push({ step: `settings:${c.route}`, tenantSectionRendered: c.tenant.test(txt), newErrors: errs.length - before, pass: errs.length === before })
}

out.push({ step: 'pageerrors', errors: errs.slice(0, 10), pass: errs.length === 0 })
await ctx.close(); await b.close()

const allPass = out.every((s) => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== X-4 SPLITS SMOKE: ${allPass ? 'ALL PASS' : 'REVIEW'} ===`)
out.forEach((s) => console.log(`  ${s.pass ? '✓' : '✗'} ${s.step}`))
