import { chromium } from 'playwright'
import { resolve } from 'node:path'
const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/phase3-qa')
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const browser = await chromium.launch()
const out = []
for (const [name, w] of [['Martin Berger', 500], ['Martin Berger', 360]]) {
  const ctx = await browser.newContext({ viewport: { width: w, height: 800 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []; page.on('pageerror', e => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded' }); await page.waitForTimeout(1800)
    await page.getByText(name, { exact: false }).first().click({ timeout: 5000 }); await page.waitForTimeout(1000)
    const prof = page.getByText(/Profiling/i).first()
    if (await prof.count()) { await prof.scrollIntoViewIfNeeded().catch(()=>{}); await page.waitForTimeout(400) }
    // Verlauf aufklappen falls vorhanden (mehr Metadaten = mehr Overflow-Risiko)
    const file = resolve(outDir, `consent-revoked-${w}.png`)
    await page.screenshot({ path: file })
    out.push({ name, w, errs: errs.length, file })
  } catch (e) { out.push({ name, w, error: String(e).split('\n')[0] }) } finally { await ctx.close() }
}
await browser.close(); console.log(JSON.stringify(out, null, 2))
