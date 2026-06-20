import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/b5-5-coupling')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2400)

// Open the module-settings overlay for berichte via the sidebar footer button.
await page.getByText(/Modul-Einstellun/).first().click().catch(() => {})
await page.waitForTimeout(900)
await page.screenshot({ path: resolve(outDir, 'b5-5-settings-open.png') })

// Find the tenant "delivery" section and reveal the release-gate hint.
const tenant = page.getByText(/Zustellung|Versand|Geplante Berichte/).first()
if (await tenant.count()) await tenant.click().catch(() => {})
await page.waitForTimeout(500)
out.releaseGateHint = await page
  .getByText(/erst ab dem Status .* ausgeführt|nur ab Status|Freigegeben.*ausgeführt/)
  .count()
await page.screenshot({ path: resolve(outDir, 'b5-5-settings-tenant.png'), fullPage: true })

out.errs = errs.length
out.errDetail = errs.slice(0, 3)
console.log(JSON.stringify(out, null, 2))
await browser.close()
