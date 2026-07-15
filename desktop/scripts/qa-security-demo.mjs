/**
 * QA — Security admin hub, visual demo-mode pass (Session #8).
 * Luke cleaned security-client.ts (paths/envelopes/protojson) but could not run
 * Electron screenshot QA. This walks all 9 tabs of /admin/security in demo mode,
 * screenshots each, and scans for raw i18n keys / crashes / empty layouts.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/security-demo')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 960 } })
for (const s of [STUB, ONB, NOLAUNCH]) await ctx.addInitScript(s)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []

// Routed surface = SecurityAdminHubTab (admin console, horizontal sub-tabs).
const TABS = ['Audit-Log', 'Aufbewahrung', 'Sessions', 'Passwort-Richtlinie', 'IP-Whitelist', 'Datenschutz', 'DSAR', 'KI-Governance']

async function scanRawKeys() {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(security|admin|common|shared|settings)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}

try {
  await page.goto(`${BASE}/#/admin/security`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  const heading = await page.evaluate(() => document.body.innerText.includes('Audit') || document.body.innerText.includes('Sicherheit'))
  out.push({ step: 'admin/security hub renders', heading, pass: heading })

  for (let i = 0; i < TABS.length; i++) {
    const label = TABS[i]
    try {
      const btn = page.getByRole('button', { name: new RegExp(label.replace(/[-]/g, '.?')) }).first()
      const tabAlt = page.getByRole('tab', { name: new RegExp(label.replace(/[-]/g, '.?')) }).first()
      const target = (await btn.count()) ? btn : tabAlt
      if (await target.count()) { await target.click({ timeout: 5000 }).catch(() => {}); await page.waitForTimeout(1100) }
      await page.screenshot({ path: resolve(outDir, `${String(i + 1).padStart(2, '0')}-${label.replace(/[^a-zA-Z]/g, '')}.png`) })
      const raw = await scanRawKeys()
      const bodyLen = (await page.evaluate(() => document.body.innerText.length))
      out.push({ step: `tab ${label}`, clicked: await target.count() > 0, rawKeys: raw.slice(0, 6), hasContent: bodyLen > 200, pass: raw.length === 0 && bodyLen > 200 })
    } catch (e) {
      out.push({ step: `tab ${label}`, pass: false, error: String(e).split('\n')[0] })
    }
  }
} catch (e) {
  out.push({ step: 'fatal', pass: false, error: String(e).split('\n')[0] })
}

out.push({ step: 'pageerrors', errors: errs.slice(0, 10), pass: errs.length === 0 })
await ctx.close(); await b.close()
const allPass = out.every((s) => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== SECURITY DEMO QA: ${allPass ? 'ALL PASS' : 'REVIEW'} ===`)
out.forEach((s) => console.log(`  ${s.pass ? '✓' : '✗'} ${s.step}${s.rawKeys && s.rawKeys.length ? ' — RAW: ' + s.rawKeys.join(', ') : ''}`))
