/**
 * QA — helpdesk H-4 agent actions (assign / escalate / merge) against :5174.
 * Verifies each action takes visible effect (header badge, meta, thread note)
 * and merge closes the source ticket. Screenshots: .qa-screenshots/helpdesk-actions/
 *
 * Usage: node scripts/qa-helpdesk-actions.mjs
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots', 'helpdesk-actions')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const WIPE = `try{if(!sessionStorage.getItem('qa-hd-wiped')){localStorage.removeItem('cosmi-helpdesk');sessionStorage.setItem('qa-hd-wiped','1')}}catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1480, height: 1000 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(WIPE)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error') errs.push('[console] ' + m.text()) })

const dialog = () => page.locator('[role="dialog"]')
const headerBadges = () => dialog().locator('h3').locator('xpath=ancestor::div[1]').textContent().catch(() => '')
const threadCount = async () => (await dialog().getByText(/Nachrichten \(/).first().textContent().catch(() => '')) || ''
const openRow = async (text) => {
  const row = page.locator('tbody tr').filter({ hasText: text }).first()
  await row.evaluate((el) => el.click()).catch(async () => { await row.click() })
  await page.waitForTimeout(900)
}

const out = []
try {
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)

  // ── Escalate a low-priority ticket (tk-5 "Bildschirm flackert" = low) ──
  await openRow('Bildschirm flackert')
  const dlgBefore = await dialog().textContent()
  const threadBefore = await threadCount()
  await dialog().locator('button:has-text("Eskalieren")').first().click()
  await page.waitForTimeout(700)
  const dlgAfter = await dialog().textContent()
  const threadAfterEsc = await threadCount()
  await page.screenshot({ path: resolve(outDir, '0-escalated.png') })
  out.push({
    step: 'escalate',
    priorityBeforeLow: /Niedrig/.test(dlgBefore || ''),
    priorityAfterMedium: /Mittel/.test(dlgAfter || ''),
    threadBefore: threadBefore.trim(), threadAfterEsc: threadAfterEsc.trim(),
    escalationNote: await dialog().getByText(/[Ee]skaliert/).count(),
  })

  // ── Assign to the OTHER agent (tk-5 is Sandra Bürki by default → assign to Marco) ──
  await page.locator('[role="dialog"] select[aria-label="Zuweisen"]').selectOption({ label: 'Marco Hartmann' }).catch((e) => errs.push('assign:' + e))
  await page.waitForTimeout(700)
  const dlgAssign = await dialog().textContent()
  await page.screenshot({ path: resolve(outDir, '1-assigned.png') })
  out.push({
    step: 'assign',
    assignedToMarco: /Marco Hartmann/.test(dlgAssign || ''),
    assignNote: await dialog().getByText(/zugewiesen an Marco Hartmann/).count(),
  })

  // ── Merge into another open ticket ──
  const mergeSel = page.locator('[role="dialog"] select[aria-label="In Ticket zusammenführen…"]')
  const options = await mergeSel.locator('option').count()
  const targetValue = await mergeSel.locator('option').nth(1).getAttribute('value')
  await mergeSel.selectOption(targetValue)
  await page.waitForTimeout(900)
  const dialogStillOpen = await dialog().isVisible().catch(() => false)
  await page.screenshot({ path: resolve(outDir, '2-after-merge.png'), fullPage: true })
  // reopen source ticket (tk-5) — should now be closed with a merge note
  await openRow('Bildschirm flackert')
  const dlgSource = await dialog().textContent()
  out.push({
    step: 'merge',
    mergeOptions: options,
    dialogClosedAfterMerge: !dialogStillOpen,
    sourceClosed: /Geschlossen/.test(dlgSource || ''),
    sourceMergeNote: await dialog().getByText(/[Zz]usammengeführt mit/).count(),
  })

  out.push({ pageErrors: errs.slice(0, 6) })
} catch (e) {
  out.push({ error: String(e).split('\n')[0], pageErrors: errs.slice(0, 6) })
} finally {
  await ctx.close(); await browser.close()
}

console.log(JSON.stringify(out, null, 2))
