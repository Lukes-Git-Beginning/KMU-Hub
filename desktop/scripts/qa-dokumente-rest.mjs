// QA dokumente-Reste — Demo-Sidebar mit echten Team-/Projekt-Spaces
// + TemplateGallery erstellt echte Dateien (Upload-Roundtrip).
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

  // 1. Sidebar spaces differ now
  out.teamRootVisible = await docsSidebar.getByText('Team-Dateien').first().isVisible().catch(() => false)
  out.teamVertriebVisible = await docsSidebar.getByText('Vertrieb', { exact: true }).first().isVisible().catch(() => false)
  out.projectVisible = await docsSidebar.getByText('Projekt TechVision').first().isVisible().catch(() => false)
  // the old bug: "Meine Dateien" appeared 3x — now only once
  out.meineDateienCount = await docsSidebar.getByText('Meine Dateien', { exact: true }).count().catch(() => -1)
  await shot(page, 'dokumente-rest-sidebar.png')

  // 2. Team folder shows its file
  await docsSidebar.getByRole('button', { name: 'Vertrieb', exact: true }).first().click({ timeout: 8000 })
    .catch((e) => { out.vertriebClickErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  out.teamFileVisible = await page.getByText('Pitch_Deck_Vertrieb.pdf').first().isVisible().catch(() => false)
  await shot(page, 'dokumente-rest-team-folder.png')

  // 3. Template creates a real file in the active folder (Vorlagen)
  await docsSidebar.getByRole('button', { name: 'Vorlagen', exact: true }).first().click({ timeout: 8000 })
  await page.waitForTimeout(1200)
  await page.getByText('Aus Vorlage').first().click({ timeout: 8000 })
  await page.waitForTimeout(1000)
  await shot(page, 'dokumente-rest-template-gallery.png')
  await page.getByText('Arbeitsvertrag', { exact: true }).first().click({ timeout: 8000 })
  await page.waitForTimeout(5500) // upload + toast ausklingen lassen
  // Erfolg = Dialog zu UND Datei als eigener Listeneintrag (exakter Knoten,
  // nicht der Karten-Wrapper "Arbeitsvertrag"+".docx" — war ein False Positive)
  out.galleryClosedAfterCreate = !(await page.getByText('Neu aus Vorlage').first().isVisible().catch(() => false))
  out.templateFileCreated = await page
    .locator('p, span, h3')
    .filter({ hasText: /^Arbeitsvertrag\.docx$/ })
    .first()
    .isVisible()
    .catch(() => false)
  await shot(page, 'dokumente-rest-template-created.png')

  out.rawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}

out.failedApiRequests = [...new Set(failed)].slice(0, 20)
out.pageErrors = [...new Set(errors)].slice(0, 12)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
