// QA work Phase 2 — WorkSettingsPanel (settings-komplett) + verdrahtete prefs.
// Öffnet Modul-Einstellungen von /work, prüft alle 6 Sektionen, Label-Add,
// Custom-Field-Form, Zeit-Toggle; dann persönliche Default-Ansicht-Wirkung.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')

const STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUP = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(work|common|moduleSettings|settings)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
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

try {
  // 1. Open module settings from /work → context preselects "work"
  await page.goto(`${BASE}/#/work/projects`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2800)
  await page.getByText(/Modul-Einstellungen/i).first().click({ timeout: 6000 }).catch((e) => { out.openErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1000)
  // ensure the work entry is selected (label "Projekte & Aufgaben")
  await page.getByText(/Projekte & Aufgaben/).first().click({ timeout: 4000 }).catch((e) => { out.entryErr = String(e).split('\n')[0] })
  await page.waitForTimeout(900)

  out.hasPersonal = await page.getByText('Persönliche Ansicht').first().isVisible().catch(() => false)
  out.hasLabels = await page.getByText('Label-Taxonomie').first().isVisible().catch(() => false)
  out.hasTemplates = await page.getByText('Projekt-Vorlagen').first().isVisible().catch(() => false)
  out.hasStatusSet = await page.getByText('Standard-Workflow').first().isVisible().catch(() => false)
  out.hasFields = await page.getByText('Eigene Felder').first().isVisible().catch(() => false)
  out.hasTime = await page.getByText('Zeit-Regeln').first().isVisible().catch(() => false)
  out.seedLabels = await page.getByText(/^Bug$|^Feature$/).count().catch(() => -1)
  out.seedTemplates = await page.getByText(/Sprint-Projekt|Kunden-Onboarding/).count().catch(() => -1)
  await shot(page, 'work-settings-panel.png')
  out.panelRawKeys = await scanRawKeys(page)

  // 2. Add a label
  await page.getByText('Neues Label').first().click({ timeout: 4000 }).catch((e) => { out.addLabelErr = String(e).split('\n')[0] })
  await page.waitForTimeout(400)
  await page.getByPlaceholder('Label-Name…').first().fill('QA-Test').catch(() => {})
  await page.getByRole('button', { name: 'Hinzufügen' }).first().click({ timeout: 3000 }).catch(() => {})
  await page.waitForTimeout(500)
  out.labelAdded = await page.getByText('QA-Test').first().isVisible().catch(() => false)

  // 3. Open custom-field add form
  await page.getByText('Neues Feld').first().click({ timeout: 4000 }).catch((e) => { out.fieldFormErr = String(e).split('\n')[0] })
  await page.waitForTimeout(500)
  out.fieldFormVisible = await page.getByPlaceholder('Feldname…').first().isVisible().catch(() => false)
  await shot(page, 'work-settings-field-form.png')

  // 4. Narrow widths
  await page.setViewportSize({ width: 760, height: 1000 })
  await page.waitForTimeout(600)
  await shot(page, 'work-settings-half.png')
  await page.setViewportSize({ width: 1440, height: 1100 })
  await page.waitForTimeout(400)
  out.settingsRawKeys = await scanRawKeys(page)

  // 5. Verify personal default view wiring: set defaultView via store, reload, open project
  await page.evaluate(() => {
    const raw = localStorage.getItem('cosmi-work-prefs')
    const p = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    p.state = { ...(p.state || {}), defaultView: 'gantt' }
    localStorage.setItem('cosmi-work-prefs', JSON.stringify(p))
  })
  await page.goto(`${BASE}/#/work/projects/prj-001`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  // Gantt view shows the "Kritischer Pfad" legend
  out.defaultViewGanttApplied = await page.getByText(/Kritischer Pfad/i).first().isVisible().catch(() => false)
  await shot(page, 'work-settings-defaultview-gantt.png')

  // 6. Verify my-tasks grouping wiring: set groupBy=priority, reload my-tasks
  await page.evaluate(() => {
    const raw = localStorage.getItem('cosmi-work-prefs')
    const p = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    p.state = { ...(p.state || {}), myTasksGroupBy: 'priority' }
    localStorage.setItem('cosmi-work-prefs', JSON.stringify(p))
  })
  await page.goto(`${BASE}/#/work/my-tasks`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2200)
  out.groupByPriorityApplied = await page.getByText(/^Dringend$|^Hoch$|^Niedrig$/).count().catch(() => -1)
  await shot(page, 'work-settings-mytasks-priority.png')
  out.myTasksRawKeys = await scanRawKeys(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
}

out.failedApiRequests = [...new Set(failed)].slice(0, 20)
out.pageErrors = [...new Set(errors)].slice(0, 12)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
