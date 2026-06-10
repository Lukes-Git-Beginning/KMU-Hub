// QA dokumente Settings-Pilot — DokumenteSettingsPanel (settings-komplett)
// + verdrahtete Prefs (Standard-Ansicht, SortMenu, Dichte, OnlyOffice-Gate).
// Öffnet Modul-Einstellungen von /dokumente (Context-Preselect), prüft alle
// 4 Sektionen, dann Pref-Wirkung in der Dateiliste nach Reload.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')

const STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUP = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(dokumente|common|moduleSettings|settings)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}
async function shot(page, name) {
  await page.screenshot({ path: resolve(outDir, name), fullPage: true })
}
const setPrefs = (state) => `
  try {
    const K='cosmi-dokumente-prefs'
    const r=localStorage.getItem(K)
    const p=r?JSON.parse(r):{state:{},version:0}
    p.state={...(p.state||{}),...${JSON.stringify(state)}}
    localStorage.setItem(K,JSON.stringify(p))
    localStorage.removeItem('cosmi-view-pref-default')
  } catch(e) {}
`

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = {}
const errors = []
const failed = []

const ctx = await browser.newContext({ viewport: { width: 1440, height: 1100 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(SUP)
const page = await ctx.newPage()
page.on('pageerror', (e) => errors.push(String(e).split('\n')[0]))
page.on('response', (r) => { const u = r.url(); if (u.includes('/api/v1/') && r.status() >= 400) failed.push(`${r.status()} ${u.replace(BASE, '')}`) })

try {
  // 1. Dokumente page: SortMenu sits in the toolbar
  await page.goto(`${BASE}/#/dokumente`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2800)
  out.sortMenuVisible = await page.getByTitle('Sortieren nach').first().isVisible().catch(() => false)
  await shot(page, 'dokumente-page-toolbar.png')
  await page.getByTitle('Sortieren nach').first().click({ timeout: 4000 }).catch((e) => { out.sortOpenErr = String(e).split('\n')[0] })
  await page.waitForTimeout(500)
  out.sortDropdownDirection = await page.getByText('Richtung').first().isVisible().catch(() => false)
  await shot(page, 'dokumente-sortmenu-open.png')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)

  // 2. Module settings from /dokumente → context preselects the new entry
  await page.getByText(/Modul-Einstellungen/i).first().click({ timeout: 6000 }).catch((e) => { out.openErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  out.hasPersonal = await page.getByText('Ansicht & Verhalten').first().isVisible().catch(() => false)
  out.hasStorage = await page.getByText('Speicher & Aufbewahrung').first().isVisible().catch(() => false)
  out.hasFileTypes = await page.getByText('Erlaubte Dateitypen').first().isVisible().catch(() => false)
  out.hasSharing = await page.getByText('Freigabe & Bearbeitung').first().isVisible().catch(() => false)
  out.quotaTiers = await page.getByText(/^Starter$|^Business$|^Enterprise$/).count().catch(() => -1)
  out.fileTypeGroups = await page.getByText(/^Tabellen$|^Präsentationen$|^Audio & Video$/).count().catch(() => -1)
  await shot(page, 'dokumente-settings-panel.png')
  out.panelRawKeys = await scanRawKeys(page)

  // lower tenant sections (below the fold of the overlay)
  await page.getByText('Erlaubte Dateitypen').first().scrollIntoViewIfNeeded().catch(() => {})
  await page.waitForTimeout(400)
  await shot(page, 'dokumente-settings-tenant-lower.png')
  await page.getByText('OnlyOffice-Editor').first().scrollIntoViewIfNeeded().catch(() => {})
  await page.waitForTimeout(400)
  await shot(page, 'dokumente-settings-sharing.png')

  // 3. Narrow width
  await page.setViewportSize({ width: 760, height: 1000 })
  await page.waitForTimeout(600)
  await shot(page, 'dokumente-settings-half.png')
  await page.setViewportSize({ width: 1440, height: 1100 })
  await page.waitForTimeout(400)

  // 4. Pref wiring: default view list + compact density + sort name/asc.
  // Hash-goto does NOT reload the page → use reload() so the persisted store
  // rehydrates and the settings overlay is gone.
  await page.evaluate(setPrefs({ defaultView: 'list', density: 'compact', sortField: 'name', sortDir: 'asc' }))
  await page.reload({ waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  out.overlayClosed = !(await page.getByText('Modul-Einstellungen').first().isVisible().catch(() => false))
  // list view shows the column header row ("Größe" only exists in the list header)
  out.defaultViewListApplied = await page.getByText('Größe').first().isVisible().catch(() => false)
  out.sortMenuShowsName = await page.getByTitle('Sortieren nach').first().innerText().then((s) => s.includes('Name')).catch(() => false)
  await shot(page, 'dokumente-list-compact-sortname.png')

  // 5. Grid + compact
  await page.evaluate(setPrefs({ defaultView: 'grid', density: 'compact' }))
  await page.reload({ waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  out.gridApplied = !(await page.getByText('Größe').first().isVisible().catch(() => false))
  await shot(page, 'dokumente-grid-compact.png')
  out.gridRawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}

out.failedApiRequests = [...new Set(failed)].slice(0, 20)
out.pageErrors = [...new Set(errors)].slice(0, 12)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
