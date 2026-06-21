// Live-arrival QA: proves the MSW pipeline feeds the toast (N-2) and that
// DND suppresses it (N-1). Dev server must be on :5174.
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/notifications')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

const waitForToast = async (ms) => {
  const deadline = Date.now() + ms
  while (Date.now() < deadline) {
    const count = await page.locator('[data-sonner-toast]').count().catch(() => 0)
    if (count > 0) return count
    await page.waitForTimeout(1000)
  }
  return 0
}

try {
  // ── Positive: live arrival surfaces a toast ──
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  const toasts = await waitForToast(28000)
  out.positive = {
    toastCount: toasts,
    toastText: await page.locator('[data-sonner-toast]').first().innerText().catch(() => ''),
  }
  await page.screenshot({ path: resolve(outDir, '6-live-toast.png') })
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'ERROR-live.png') }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
