// QA work (Projekte/Aufgaben) Phase 1 — Demo-Mode reparieren.
// Klickt durch: Projekt-Liste → Projekt-Detail (Kanban/Liste/Gantt) →
// Task-Detail-Panel (alle Tabs) → MyTasks. Sammelt pageErrors (= fehlende/
// falsch geshapte MSW-Handler), Raw-Keys, Screenshots @ 3 Breiten.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  window.electronAPI = new Proxy(noop, handler)
`
const SUPPRESS_ONBOARDING = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
`

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
const failedRequests = []

const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
const page = await ctx.newPage()
page.on('pageerror', (e) => errors.push(String(e).split('\n')[0]))
page.on('response', (r) => {
  const u = r.url()
  if (u.includes('/api/v1/') && r.status() >= 400) failedRequests.push(`${r.status()} ${u.replace(BASE, '')}`)
})

try {
  // 1. Projects list
  await page.goto(`${BASE}/#/work/projects`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2800)
  out.projectCards = await page.locator('text=/Cosmi v2.0|Website Relaunch|Mobile App/').count().catch(() => -1)
  await shot(page, 'work-projects-list.png')
  out.listRawKeys = await scanRawKeys(page)

  // 2. Open first project → Kanban board
  await page.getByText('Cosmi v2.0').first().click({ timeout: 6000 }).catch((e) => { out.openProjectErr = String(e).split('\n')[0] })
  await page.waitForTimeout(2200)
  out.kanbanColumns = await page.locator('text=/Offen|In Bearbeitung|Review|Erledigt/').count().catch(() => -1)
  out.kanbanCards = await page.locator('[class*="cursor"]:has-text("Plugin-API")').count().catch(() => -1)
  await shot(page, 'work-project-kanban.png')
  out.kanbanRawKeys = await scanRawKeys(page)

  // 3. Switch to list view (toggle buttons — Liste)
  await page.getByRole('button', { name: /Liste/i }).first().click({ timeout: 4000 }).catch((e) => { out.listViewErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  await shot(page, 'work-project-list.png')

  // 4. Switch to Gantt view
  await page.getByRole('button', { name: /Gantt/i }).first().click({ timeout: 4000 }).catch((e) => { out.ganttErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1200)
  await shot(page, 'work-project-gantt.png')

  // 5. Back to kanban + open a task → detail panel/page
  await page.getByRole('button', { name: /Kanban|Board/i }).first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(1000)
  await page.getByText('Plugin-API v2 Architektur').first().click({ timeout: 6000 }).catch((e) => { out.openTaskErr = String(e).split('\n')[0] })
  await page.waitForTimeout(2000)
  out.taskTitleVisible = await page.getByText('Plugin-API v2 Architektur').first().isVisible().catch(() => false)
  await shot(page, 'work-task-detail.png')
  out.taskDetailRawKeys = await scanRawKeys(page)

  // 6. Click through task-detail tabs/sections if present (Kommentare, Abhängigkeiten, Zeit, Dateien, Aktivität, Felder)
  for (const label of ['Kommentare', 'Abhängigkeiten', 'Zeit', 'Dateien', 'Aktivität', 'Verlauf', 'Unteraufgaben', 'Felder', 'Verknüpfungen']) {
    const el = page.getByRole('tab', { name: new RegExp(label, 'i') }).first()
    const btn = page.getByRole('button', { name: new RegExp(`^${label}`, 'i') }).first()
    if (await el.isVisible().catch(() => false)) { await el.click().catch(() => {}); await page.waitForTimeout(700) }
    else if (await btn.isVisible().catch(() => false)) { await btn.click().catch(() => {}); await page.waitForTimeout(700) }
  }
  await shot(page, 'work-task-detail-tabs.png')

  // 7. My Tasks
  await page.goto(`${BASE}/#/work/my-tasks`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2200)
  out.myTasksRows = await page.locator('text=/Plugin-API|Design-System|Notification/').count().catch(() => -1)
  await shot(page, 'work-my-tasks.png')
  out.myTasksRawKeys = await scanRawKeys(page)

  // 8. Narrow widths on project board
  await page.goto(`${BASE}/#/work/projects`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(1500)
  await page.setViewportSize({ width: 720, height: 900 })
  await page.waitForTimeout(700)
  await shot(page, 'work-projects-half.png')
  await page.setViewportSize({ width: 500, height: 800 })
  await page.waitForTimeout(700)
  await shot(page, 'work-projects-small.png')
} catch (err) {
  out.error = String(err).split('\n')[0]
}

out.failedApiRequests = [...new Set(failedRequests)].slice(0, 20)
out.pageErrors = [...new Set(errors)].slice(0, 12)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
