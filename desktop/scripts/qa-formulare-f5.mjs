/**
 * QA — formulare F-5: module settings + prefs apply + multi-state QA.
 * 1) Settings overlay renders the Formulare panel (personal + tenant), no raw keys.
 * 2) personal defaultTab applies (page opens on the persisted tab).
 * 3) personal defaultExportFormat applies (ExportMenu lists it first + "Standard").
 * 4) Screenshots at two widths. 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-f5')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const prefs = (state) => `try{localStorage.setItem('cosmi-formulare-prefs', JSON.stringify({state:${JSON.stringify(state)},version:0}))}catch(e){}`
const RAW_RE = /(formulare\.[a-z][a-zA-Z.]+|moduleSettings\.[a-z]|\{\{)/

async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim()).slice(0, 10)
  }, RAW_RE.source)
}

const browser = await chromium.launch()
const out = []
const errs = []

// ── 1) Settings overlay render ──────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('render: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('render console: ' + m.text()) })
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  await page.locator('a:has-text("Einstellungen"), button:has-text("Einstellungen")').first().click({ timeout: 8000 }).catch(() => {})
  await page.waitForTimeout(1200)
  const dlg = page.locator('[role="dialog"]')
  // Ensure the Formulare entry is selected in the overlay's left nav.
  await dlg.locator('button:has-text("Formulare"), [role="button"]:has-text("Formulare")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(700)
  const overlayText = await dlg.first().textContent().catch(() => '')
  const hasPanel =
    /Standard-Tab/.test(overlayText || '') &&
    /Standard-Exportformat/.test(overlayText || '') &&
    /Aufbewahrung/.test(overlayText || '')
  await page.screenshot({ path: resolve(outDir, '0-settings-overlay.png'), fullPage: true })
  out.push({ check: 'settings-render', hasPanel, rawKeys: await rawKeys(page) })
  await ctx.close()
}

// ── 2) defaultTab applies ───────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  await ctx.addInitScript(prefs({ defaultTab: 'vorlagen', defaultExportFormat: 'csv', defaultConsentText: 'X', defaultPrivacyUrl: 'https://x', notifyOnSubmission: true, retentionDays: 365 }))
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('tab: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  const activeTab = await page.locator('.tab-accent-active, button.border-primary').first().textContent().catch(() => '')
  await page.screenshot({ path: resolve(outDir, '1-default-tab-vorlagen.png') })
  out.push({ check: 'default-tab', activeTab, appliesVorlagen: /Vorlagen/.test(activeTab || '') })
  await ctx.close()
}

// ── 3) defaultExportFormat applies ──────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  await ctx.addInitScript(prefs({ defaultTab: 'eingänge', defaultExportFormat: 'xlsx', defaultConsentText: 'X', defaultPrivacyUrl: 'https://x', notifyOnSubmission: true, retentionDays: 365 }))
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('export: ' + String(e)))
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  await page.locator('.space-y-4 > .rounded-lg').first().locator('button:has-text("Exportieren")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(500)
  const menuItems = await page.locator('.absolute.z-20 button').allTextContents().catch(() => [])
  await page.screenshot({ path: resolve(outDir, '2-export-default-xlsx.png') })
  out.push({ check: 'default-export', firstItem: menuItems[0] || null, xlsxFirst: /Excel/.test(menuItems[0] || ''), hasStandardMark: menuItems.some((m) => /Standard/.test(m)) })
  await ctx.close()
}

// ── 4) Width responsiveness ─────────────────────────────────────
for (const w of [1280, 1024]) {
  const ctx = await browser.newContext({ viewport: { width: w, height: 900 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  await page.screenshot({ path: resolve(outDir, `3-width-${w}.png`), fullPage: true })
  out.push({ check: `width-${w}`, rawKeys: await rawKeys(page) })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
