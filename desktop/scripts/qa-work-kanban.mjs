// QA — work Kanban quick-actions (W-2) + task-detail delete (W-3). Dev :5173.
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/work')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const vis = (re) => page.getByText(re).first().isVisible().catch(() => false)

try {
  // --- Reach a project Kanban board ---
  await page.goto(`${BASE}/#/work/projects`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  await page.locator('.cursor-pointer.transition-shadow').first().click().catch(() => {})
  await page.waitForTimeout(2500)
  out.onProject = page.url()
  await page.locator('button[title="Kanban"]').first().click().catch(() => {})
  await page.waitForTimeout(2500)

  // W-2: Kanban card action menu. Use exact name so we hit the menu button
  // (accessible name === "Aktionen") and NOT the dnd card (role=button whose
  // name merely *contains* "Aktionen").
  out.kanbanMenus = await page.getByRole('button', { name: 'Aktionen', exact: true }).count()
  await page.getByRole('button', { name: 'Aktionen', exact: true }).first().click().catch(() => {})
  await page.waitForTimeout(600)
  out.kbMenuComplete = await vis(/Abschließen|Wieder öffnen/)
  out.kbMenuAssign = await vis(/Mir zuweisen/)
  out.kbMenuDue = await vis(/Nächste Woche/)
  out.kbMenuDelete = await vis(/^Löschen$/)
  await page.screenshot({ path: resolve(outDir, 'kanban-1-menu.png') })
  // delete → confirm
  await page.getByRole('button', { name: /^Löschen$/ }).first().click().catch(() => {})
  await page.waitForTimeout(600)
  out.kbDeleteConfirm = await vis(/Aufgabe löschen\?/)
  await page.screenshot({ path: resolve(outDir, 'kanban-2-delete-confirm.png') })
  await page.getByRole('button', { name: /^Abbrechen$/ }).first().click().catch(() => {})
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // W-2: quick-complete checkbox on a card
  out.kbCheckboxes = await page.getByRole('button', { name: /^Abschließen$/ }).count()
  await page.getByRole('button', { name: /^Abschließen$/ }).first().click().catch(() => {})
  await page.waitForTimeout(1300)
  out.kbStrikethrough = await page.locator('.line-through').count()
  await page.screenshot({ path: resolve(outDir, 'kanban-3-after-complete.png') })

  // --- W-3 panel: click a card body → TaskDetailPanel → delete ---
  await page.locator('p.line-clamp-2').first().click().catch(() => {})
  await page.waitForTimeout(1500)
  const panelDelete = page.getByRole('button', { name: /^Löschen$/ })
  out.panelDeleteBtn = await panelDelete.count()
  await panelDelete.first().click().catch(() => {})
  await page.waitForTimeout(600)
  out.panelDeleteConfirm = await vis(/Aufgabe löschen\?/)
  await page.screenshot({ path: resolve(outDir, 'panel-delete-confirm.png') })
  await page.getByRole('button', { name: /^Abbrechen$/ }).first().click().catch(() => {})
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // --- W-3 page: MyTasks → project task → detail page → delete ---
  await page.goto(`${BASE}/#/work/my-tasks`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  await page.locator('button').filter({ hasText: /Plugin-API|Notification|Offline|SSL|AVV|Architektur/ }).first().click().catch(() => {})
  await page.waitForTimeout(2200)
  out.onDetailPage = page.url().includes('/tasks/')
  out.pageDeleteBtn = await page.getByRole('button', { name: /^Löschen$/ }).count()
  await page.getByRole('button', { name: /^Löschen$/ }).first().click().catch(() => {})
  await page.waitForTimeout(600)
  out.pageDeleteConfirm = await vis(/Aufgabe löschen\?/)
  await page.screenshot({ path: resolve(outDir, 'page-delete-confirm.png') })

  const body = await page.evaluate(() => document.body.innerText)
  out.rawKeys = [...new Set([...body.matchAll(/\bwork\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))].slice(0, 10)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errs.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
