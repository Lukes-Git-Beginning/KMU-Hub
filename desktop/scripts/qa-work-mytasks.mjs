// QA — work "Meine Aufgaben" quick-actions (W-1). Dev :5173.
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
  await page.goto(`${BASE}/#/work/my-tasks`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4000)
  // Rows no longer carry role=button (sibling-button layout) — count the per-row action menus.
  out.taskRows = await page.getByRole('button', { name: /Aktionen/ }).count()
  out.completeCheckboxes = await page.getByRole('button', { name: /^Abschließen$/ }).count()

  // open action menu
  await page.getByRole('button', { name: /Aktionen/ }).first().click()
  await page.waitForTimeout(700)
  out.menuComplete = await vis(/Abschließen|Wieder öffnen/)
  out.menuAssign = await vis(/Mir zuweisen/)
  out.menuDue = await vis(/Nächste Woche/)
  out.menuMove = await vis(/verschieben/)
  out.menuDelete = await vis(/^Löschen$/)
  out.detailLeaked = page.url().includes('/tasks/')
  await page.screenshot({ path: resolve(outDir, '2-action-menu.png') })

  // delete → confirm dialog
  await page.getByRole('button', { name: /^Löschen$/ }).first().click().catch(() => {})
  await page.waitForTimeout(600)
  out.deleteConfirmShown = await vis(/Aufgabe löschen\?/)
  await page.screenshot({ path: resolve(outDir, '3-delete-confirm.png') })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // quick-complete first task
  const before = await page.getByRole('button', { name: /Aktionen/ }).count()
  await page.getByRole('button', { name: /^Abschließen$/ }).first().click().catch(() => {})
  await page.waitForTimeout(1400)
  out.rowsBefore = before
  out.rowsAfterComplete = await page.getByRole('button', { name: /Aktionen/ }).count()
  out.strikethrough = await page.locator('.line-through').count()
  // toggle "show completed" to confirm it's there + struck through
  await page.getByRole('button', { name: /Erledigte anzeigen/ }).click().catch(() => {})
  await page.waitForTimeout(1200)
  out.strikethroughWithCompleted = await page.locator('.line-through').count()
  await page.screenshot({ path: resolve(outDir, '4-after-complete.png'), fullPage: true })

  const body = await page.evaluate(() => document.body.innerText)
  out.rawKeys = [...new Set([...body.matchAll(/\bwork\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))].slice(0, 10)
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errs.slice(0, 6)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
