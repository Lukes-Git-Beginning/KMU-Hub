// QA — dialer final audit: all nav surfaces render, no raw keys / errors (dev :5174)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/dialer-final')
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
const rawKeysOf = () => page.evaluate(() => {
  const all = Array.from(document.querySelectorAll('body *'))
    .filter((n) => n.children.length === 0)
    .map((n) => (n.textContent || '').trim())
  return [...new Set(all.filter((t) => /^(dialer|common|shared|moduleSettings)\.[a-zA-Z]/.test(t)))].slice(0, 10)
})

const surfaces = [
  ['campaigns', '/#/dialer/campaigns'],
  ['detail', '/#/dialer/campaigns/dlr-camp-001'],
  ['dashboard', '/#/dialer/dashboard'],
  ['supervisor', '/#/dialer/supervisor'],
  ['settingsInModule', '/#/dialer/settings'],
]
for (const [name, url] of surfaces) {
  await page.goto(`${BASE}${url}`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2800)
  out[`${name}_rawKeys`] = await rawKeysOf()
  out[`${name}_nan`] = /NaN/.test(await page.evaluate(() => document.body.textContent || ''))
  await page.screenshot({ path: resolve(outDir, `final-${name}.png`), fullPage: false })
}

out.pageErrors = errs.slice(0, 10)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
