import { chromium } from 'playwright'
import { resolve } from 'node:path'
const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/dialer-tiefe')
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page = await ctx.newPage()
await page.goto(`${BASE}/#/dialer/supervisor`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2500)
const clicked = await page.evaluate(() => {
  const el = Array.from(document.querySelectorAll('button, a, [role="button"]')).find((e) => /Modul-Einstellung/.test(e.textContent || ''))
  if (el) { el.click(); return true }
  return false
})
// Auf das echte Panel-Element warten (überspringt die CosmiLaunch-Reload-Animation)
try {
  await page.getByText('Anruf-Workflow', { exact: false }).first().waitFor({ state: 'visible', timeout: 12000 })
} catch { /* fällt auf Screenshot zurück */ }
await page.waitForTimeout(600)
await page.screenshot({ path: resolve(outDir, 'cd-dialer-settings-panel.png'), fullPage: false })
const txt = await page.evaluate(() => document.body.textContent || '')
console.log(JSON.stringify({
  clicked,
  hasPersonal: /Anruf-Workflow|Nachbearbeitungszeit/.test(txt),
  hasTenant: /Team-Vorgaben|gleichzeitige Anrufe|Aufzeichnungs/.test(txt),
}, null, 2))
await ctx.close(); await b.close()
