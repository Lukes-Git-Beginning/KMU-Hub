// QA dokumente Feedback-Runde (Darien 2026-06-10):
// 1) Klick öffnet Dokument-Viewer (nicht Seitenpanel)
// 2) Viewer-Toolbar-Aktionen vorhanden
// 3) Info-Panel default zu, togglebar, zeigt Details + Aktivität
// 4) Hover vergrößert Karte
// 5) Seiten-Vorschau in Kacheln + Settings-Schalter deaktiviert sie live
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

const docsSidebar = page.locator('aside').filter({ hasText: 'Favoriten' })

try {
  await page.goto(`${BASE}/#/dokumente`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2800)

  // 5a. Preview tiles on by default (page-look) + 4. hover zoom
  await shot(page, 'dokumente-fb-grid-previews.png')
  const card = page.getByText('Projektplan_KMU_Hub_v2.pdf').first()
  await card.hover()
  await page.waitForTimeout(450)
  await shot(page, 'dokumente-fb-card-hover.png')

  // 1. Plain click opens the VIEWER (not the right side panel)
  await card.click()
  await page.waitForTimeout(1500)
  out.viewerOpened = await page.locator('[role="dialog"]').getByText('Projektplan_KMU_Hub_v2.pdf').first().isVisible().catch(() => false)
  // side panel would show "Eigenschaften"-style DetailPanel — make sure the
  // viewer toolbar (Info button) is what opened instead
  out.viewerInfoButton = await page.getByRole('button', { name: 'Info' }).first().isVisible().catch(() => false)
  // 3. info panel starts collapsed
  out.infoCollapsedByDefault = !(await page.getByText('Aktivität').first().isVisible().catch(() => false))
  // 2. toolbar actions present
  out.toolbarRename = await page.locator('[role="dialog"]').getByText('Umbenennen').first().isVisible().catch(() => false)
  out.toolbarShare = await page.locator('[role="dialog"]').getByText('Teilen', { exact: true }).first().isVisible().catch(() => false)
  out.toolbarVersions = await page.locator('[role="dialog"]').getByText('Versionsverlauf').first().isVisible().catch(() => false)
  await shot(page, 'dokumente-fb-viewer.png')

  // 3. toggle info panel → details + activity trail
  await page.getByRole('button', { name: 'Info' }).first().click()
  await page.waitForTimeout(1200)
  out.infoDetailsVisible = await page.getByText('Details', { exact: true }).first().isVisible().catch(() => false)
  out.activityUploadEntry = await page.getByText('hat die Datei hochgeladen').first().isVisible().catch(() => false)
  out.tagsSection = await page.getByText('Tag hinzufügen').first().isVisible().catch(() => false)
  await shot(page, 'dokumente-fb-viewer-info.png')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(600)

  // 5b. settings switch disables previews live
  await page.getByText(/Modul-Einstellungen/i).first().click({ timeout: 6000 })
  await page.waitForTimeout(1200)
  const previewSwitch = page
    .locator('div')
    .filter({ hasText: /^Dateivorschau in Kacheln/ })
    .getByRole('switch')
    .first()
  out.settingsSwitchVisible = await previewSwitch.isVisible().catch(() => false)
  await previewSwitch.click({ timeout: 6000 }).catch((e) => { out.switchErr = String(e).split('\n')[0] })
  await page.waitForTimeout(600)
  await page.keyboard.press('Escape')
  await page.waitForTimeout(800)
  await shot(page, 'dokumente-fb-grid-no-previews.png')

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}

out.failedApiRequests = [...new Set(failed)].slice(0, 20)
out.pageErrors = [...new Set(errors)].slice(0, 12)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
