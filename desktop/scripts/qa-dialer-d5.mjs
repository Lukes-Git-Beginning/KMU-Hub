// QA — dialer D-5 module-settings panel (dev :5174)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/dialer-d5')
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
const step = async (name, fn) => {
  try { await fn() } catch (err) { out[`ERR_${name}`] = String(err).split('\n')[0] }
}

await page.goto(`${BASE}/#/dialer/campaigns`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3000)

await step('openSettings', async () => {
  // Open the module-settings overlay at the dialer entry via the live UI store
  // (Vite serves the same singleton module the app imports).
  out.opened = await page.evaluate(async () => {
    const m = await import('/src/stores/ui.ts')
    m.useUIStore.getState().openSettingsOverlay('dialer')
    return true
  })
  await page.waitForTimeout(1200)
  // ensure the Dialer entry is selected (click it in the list if needed)
  const dialerEntry = page.getByRole('button', { name: /^Dialer$/ }).first()
  if (await dialerEntry.count()) { await dialerEntry.click(); await page.waitForTimeout(700) }
  const body = await bodyText()
  out.entryListed = /Dialer/.test(body)
  out.panelTitle = /Dialer-Einstellungen/.test(body)
  out.personalSection = /Anruf-Workflow/.test(body)
  out.wrapUpTime = /Standard-Nachbearbeitungszeit/.test(body)
  out.autoAdvance = /Automatisch zum nächsten Kontakt/.test(body)
  out.tenantSection = /Team-Vorgaben/.test(body)
  out.maxConcurrent = /Max\. gleichzeitige Anrufe/.test(body)
  out.defaultOutcome = /Standard-Gesprächsergebnis/.test(body)
  out.recordingConsent = /Aufzeichnungs-Einwilligung/.test(body)
  out.scopeGroups = /Persönlich/.test(body) && /Für alle/.test(body)
  out.noRawKeys = !/dialer\.settings\.(panel|personal|tenant)\./.test(body)
  out.rawKeys = await page.evaluate(() => {
    const all = Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0)
      .map((n) => (n.textContent || '').trim())
    return [...new Set(all.filter((t) => /^(dialer|common|shared|moduleSettings)\.[a-zA-Z]/.test(t)))].slice(0, 12)
  })
  await page.screenshot({ path: resolve(outDir, 'd5-1-settings-panel.png'), fullPage: false })
})

out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
