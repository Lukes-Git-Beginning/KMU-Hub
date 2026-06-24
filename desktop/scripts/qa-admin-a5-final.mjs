// A-5 final sweep — all 4 new admin tabs @1440 + @1024, DE + EN.
// Screenshots + raw-key / {{var}} / pageerror scan per page.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = process.env.QA_FE || 'http://localhost:5174'
const outDir = resolve('.qa-admin-a5')
const TABS = ['users', 'roles', 'license', 'branding']
const WIDTHS = [{ name: 'w1440', width: 1440, height: 950 }, { name: 'w1024', width: 1024, height: 820 }]

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const prep = (locale) => `
  try { const K='cosmi-ui'; const r=localStorage.getItem(K); const p=r?JSON.parse(r):{state:{},version:0}; p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(K,JSON.stringify(p)); } catch(e){}
  try { sessionStorage.setItem('cosmi:launch-played','1') } catch(e){}
  try { localStorage.setItem('cosmi-locale', JSON.stringify({state:{locale:'${locale}'},version:0})) } catch(e){}
`

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const report = []

for (const locale of ['de', 'en']) {
  for (const size of WIDTHS) {
    const ctx = await browser.newContext({ viewport: { width: size.width, height: size.height }, reducedMotion: 'reduce' })
    await ctx.addInitScript(ELECTRON_STUB)
    await ctx.addInitScript(prep(locale))
    const page = await ctx.newPage()
    const errors = []
    page.on('pageerror', (e) => errors.push(String(e)))
    for (const tab of TABS) {
      await page.goto(`${FE}/#/admin/${tab}`, { waitUntil: 'domcontentloaded', timeout: 30000 })
      await page.waitForTimeout(2600)
      await page.screenshot({ path: resolve(outDir, `${locale}-${tab}-${size.name}.png`) })
      // Raw-key scan over visible text.
      const findings = await page.evaluate(() => {
        const txt = document.body.innerText || ''
        const out = []
        if (/\{\{/.test(txt)) out.push('double-brace')
        const rawKey = txt.match(/\b(admin|layout|moduleSettings|config|common|shared)\.[a-zA-Z]+\.[a-zA-Z.]+/g)
        if (rawKey) out.push('rawkey:' + [...new Set(rawKey)].slice(0, 5).join(','))
        return out
      })
      report.push({ locale, tab, size: size.name, errors: errors.length, findings })
    }
    await ctx.close()
  }
}
await browser.close()
const bad = report.filter((r) => r.errors > 0 || r.findings.length > 0)
console.log(JSON.stringify({ total: report.length, clean: report.length - bad.length, issues: bad }, null, 2))
