// QA dokumente P1 — Move/Copy-Dialog, echter Download, Demo-Handler
// (Versions, Shares, Favoriten, Breadcrumbs). Rechtsklick-Kontextmenü-Flows
// mit Verifikation, dass Move/Copy wirklich in der Liste ankommen.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')

const STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUP = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(dokumente|common|moduleSettings)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}
async function shot(page, name) {
  await page.screenshot({ path: resolve(outDir, name), fullPage: true })
}

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

// The dokumente folder sidebar (NOT the app-shell nav, which also says
// "Projekte") — identified by its "Favoriten" entry.
const docsSidebar = page.locator('aside').filter({ hasText: 'Favoriten' })
const sidebarFolder = (name) =>
  docsSidebar.getByRole('button', { name, exact: true }).first()

try {
  await page.goto(`${BASE}/#/dokumente`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2800)

  // 1. Breadcrumbs (path handler fix): open "Projekte" → breadcrumb row appears
  await sidebarFolder('Projekte').click()
  await page.waitForTimeout(1200)
  out.breadcrumbVisible = await page.getByText('Alle Dateien').first().isVisible().catch(() => false)
  await shot(page, 'dokumente-p1-breadcrumbs.png')

  // 2. Move: right-click a file in Projekte → Verschieben → pick Marketing
  await page.getByText('Testbericht_v2.pdf').first().click({ button: 'right' })
  await page.waitForTimeout(500)
  await shot(page, 'dokumente-p1-contextmenu.png')
  await page.getByText(/^Verschieben/).first().click()
  await page.waitForTimeout(800)
  out.moveDialogVisible = await page.getByText('Datei verschieben').first().isVisible().catch(() => false)
  await shot(page, 'dokumente-p1-move-dialog.png')
  // current folder must be disabled-marked
  out.currentFolderHint = await page.getByText('Aktueller Ordner').first().isVisible().catch(() => false)
  await page.locator('[role="dialog"]').getByText('Marketing', { exact: true }).first().click()
  await page.locator('[role="dialog"]').getByRole('button', { name: 'Verschieben' }).click()
  // wait out the success toast (contains the filename → would false-positive)
  await page.waitForTimeout(5000)
  out.movedGoneFromSource = !(await page.getByText('Testbericht_v2.pdf').first().isVisible().catch(() => false))
  await sidebarFolder('Marketing').click()
  await page.waitForTimeout(1200)
  out.movedArrivedInTarget = await page.getByText('Testbericht_v2.pdf').first().isVisible().catch(() => false)
  await shot(page, 'dokumente-p1-after-move.png')

  // 3. Copy: copy Brandbook_2026.pdf into Vorlagen
  await page.getByText('Brandbook_2026.pdf').first().click({ button: 'right' })
  await page.waitForTimeout(500)
  await page.getByText(/^Kopieren/).first().click()
  await page.waitForTimeout(800)
  await page.locator('[role="dialog"]').getByText('Vorlagen', { exact: true }).first().click()
  await page.locator('[role="dialog"]').getByRole('button', { name: 'Kopieren' }).click()
  await page.waitForTimeout(1200)
  out.copyStillInSource = await page.getByText('Brandbook_2026.pdf').first().isVisible().catch(() => false)
  await sidebarFolder('Vorlagen').click()
  await page.waitForTimeout(1200)
  out.copyArrivedInTarget = await page.getByText('Brandbook_2026.pdf').first().isVisible().catch(() => false)
  await shot(page, 'dokumente-p1-after-copy.png')

  // 4. Download: real browser download event (data: URL from demo handler)
  const downloadPromise = page.waitForEvent('download', { timeout: 8000 }).catch(() => null)
  await page.getByText('Angebot_Vorlage.docx').first().click({ button: 'right' })
  await page.waitForTimeout(500)
  await page.getByText('Herunterladen', { exact: true }).first().click().catch(async () => {
    await page.getByText('Download', { exact: true }).first().click()
  })
  const download = await downloadPromise
  out.downloadFired = !!download
  out.downloadFilename = download ? download.suggestedFilename() : null

  // 5. Versions: context menu → Versionsverlauf → "Initiale Version"
  await page.keyboard.press('Escape')
  await page.waitForTimeout(600)
  await page.getByText('Angebot_Vorlage.docx').first().click({ button: 'right', timeout: 8000 })
    .catch((e) => { out.versionsRightClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(600)
  await shot(page, 'dokumente-p1-versions-menu.png')
  await page.getByText('Versionsverlauf', { exact: true }).first().click({ timeout: 8000 })
    .catch((e) => { out.versionsClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  out.versionsVisible = await page.getByText('Initiale Version').first().isVisible().catch(() => false)
  await shot(page, 'dokumente-p1-versions.png')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 6. Shares: Vertrag_Gruber (file-005) has a seeded share with Julia Hofmann
  await sidebarFolder('Verträge').click()
  await page.waitForTimeout(1200)
  await page.getByText('Vertrag_Gruber_Maschinenbau.pdf').first().click({ button: 'right', timeout: 8000 })
    .catch((e) => { out.shareRightClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(600)
  await page.getByText('Teilen', { exact: true }).first().click({ timeout: 8000 })
    .catch((e) => { out.shareClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  out.seededShareVisible = await page.getByText('Julia Hofmann').first().isVisible().catch(() => false)
  await shot(page, 'dokumente-p1-share-dialog.png')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 7. Favorites view: 3 seeded favorites
  await page.getByText('Favoriten').first().click()
  await page.waitForTimeout(1200)
  out.favoritesCount = await page.locator('div.grid.grid-cols-2, div.space-y-1').first()
    .locator(':scope > *').count().catch(() => -1)
  out.favoriteSeedVisible = await page.getByText('Budget_2026.xlsx').first().isVisible().catch(() => false)
  await shot(page, 'dokumente-p1-favorites.png')

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}

out.failedApiRequests = [...new Set(failed)].slice(0, 20)
out.pageErrors = [...new Set(errors)].slice(0, 12)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
