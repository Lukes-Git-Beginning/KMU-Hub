// QA P4 zeiterfassung — settings panel (personal+tenant) + export dialog.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/ze-p4')
const ELECTRON_STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUPPRESS = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = ['zeiterfassung.', 'moduleSettings.', 'profil.zeiterfassung.']

async function dismiss(page) {
  for (const name of [/überspringen/i, /skip/i]) {
    const b = page.getByRole('button', { name }).first()
    try { if (await b.isVisible({ timeout: 500 })) { await b.click(); await page.waitForTimeout(300) } } catch (e) {}
  }
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const results = []

const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

try {
  // ── Settings panel via overlay (context-preselected from /zeiterfassung) ──
  await page.goto(`${BASE}/#/zeiterfassung`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2000)
  await dismiss(page)
  await page.getByText(/Modul-Einstellungen/i).first().click()
  await page.waitForTimeout(900)
  await page.screenshot({ path: resolve(outDir, '01-settings-panel.png') })
  let body = (await page.locator('body').innerText()).toLowerCase()
  results.push({ shot: 'settings panel', errors: errors.length, rawKeys: RAW.filter((p) => body.includes(p)) })
  // close overlay
  await page.keyboard.press('Escape')
  await page.waitForTimeout(500)

  // ── Export dialog from Auswertungen ──
  await page.getByRole('tab', { name: /auswertungen/i }).first().click()
  await page.waitForTimeout(1200)
  await page.getByRole('button', { name: /^export$/i }).first().click()
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, '02-export-dialog.png') })
  body = (await page.locator('body').innerText()).toLowerCase()
  results.push({ shot: 'export dialog', errors: errors.length, rawKeys: RAW.filter((p) => body.includes(p)) })
} catch (err) {
  results.push({ error: String(err).slice(0, 200) })
} finally {
  await ctx.close()
}

await browser.close()
console.log(JSON.stringify(results, null, 2))
