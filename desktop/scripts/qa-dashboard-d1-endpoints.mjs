/** D-1 endpoint probe: confirm dashboard persistence handlers return 200 (were 404). */
import { chromium } from 'playwright'

const BASE = 'http://localhost:5173'
const API = 'http://localhost:8080'
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext()
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page = await ctx.newPage()
await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2800) // let MSW service worker activate

const res = await page.evaluate(async (API) => {
  const probe = async (method, path, body) => {
    try {
      const r = await fetch(`${API}${path}`, {
        method,
        headers: body ? { 'Content-Type': 'application/json' } : undefined,
        body: body ? JSON.stringify(body) : undefined,
      })
      let echoed = null
      try { echoed = await r.json() } catch {}
      return { method, path, status: r.status, echoedWidgets: echoed?.active_widgets ?? null }
    } catch (e) { return { method, path, error: String(e) } }
  }
  return [
    await probe('GET', '/api/v1/dashboard/layout'),
    await probe('PUT', '/api/v1/dashboard/layout', { layout: [{ i: 'my-tasks', x: 0, y: 0, w: 4, h: 4 }], active_widgets: ['my-tasks', 'kpi-deals'] }),
    await probe('GET', '/api/v1/dashboard/layout'),
    await probe('DELETE', '/api/v1/dashboard/layout'),
    await probe('GET', '/api/v1/dashboard/defaults/admin'),
    await probe('PUT', '/api/v1/dashboard/defaults/manager', { layout: [], active_widgets: ['kpi-tasks'] }),
  ]
}, API)
console.log(JSON.stringify(res, null, 2))
await browser.close()
