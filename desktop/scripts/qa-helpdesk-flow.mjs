/**
 * QA — helpdesk interaction + persistence flow (parallel-batch sub-terminal).
 *
 * Drives real mutations against the :5174 QA dev server and checks they take
 * effect AND survive a reload (zustand persist key `cosmi-helpdesk`). The store
 * is wiped once per context (sessionStorage flag), so the in-test reload keeps
 * mutated state. Screenshots land in `.qa-screenshots/helpdesk-flow/`.
 *
 * Usage: node scripts/qa-helpdesk-flow.mjs
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots', 'helpdesk-flow')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const WIPE = `try{if(!sessionStorage.getItem('qa-hd-wiped')){localStorage.removeItem('cosmi-helpdesk');sessionStorage.setItem('qa-hd-wiped','1')}}catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1480, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
await ctx.addInitScript(WIPE)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error') errs.push('[console] ' + m.text()) })

const out = []
const rowCount = () => page.locator('tbody tr').count()
const firstDialogText = () => page.evaluate(() => document.querySelector('[role="dialog"]')?.textContent || '')

try {
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  const rowsBefore = await rowCount()

  // ── H-3a: create a new ticket ──────────────────────────────────────────
  await page.locator('button:has-text("Neues Ticket")').first().click()
  await page.waitForTimeout(600)
  await page.locator('[role="dialog"] input').first().fill('QA Testticket Persistenz')
  await page.locator('[role="dialog"] textarea').first().fill('Automatischer QA-Lauf: dieses Ticket muss nach Reload erhalten bleiben.')
  await page.locator('[role="dialog"] button:has-text("Ticket erstellen")').first().click()
  await page.waitForTimeout(800)
  const rowsAfter = await rowCount()
  const firstRowText = await page.locator('tbody tr').first().textContent()
  await page.screenshot({ path: resolve(outDir, '0-after-create.png'), fullPage: true })
  out.push({ step: 'create-ticket', rowsBefore, rowsAfter, firstRowHasNew: /QA Testticket Persistenz/.test(firstRowText || ''), firstRowText: (firstRowText || '').replace(/\s+/g, ' ').trim().slice(0, 80) })

  // ── H-3b: open the new ticket, send a reply ────────────────────────────
  const row = page.locator('tbody tr').first()
  await row.evaluate((el) => el.click()).catch(async () => { await row.click() })
  await page.waitForTimeout(900)
  const threadBefore = await page.locator('[role="dialog"]').getByText(/Nachrichten \(/).first().textContent().catch(() => '')
  await page.locator('[role="dialog"] textarea').first().fill('Antwort vom QA-Agent — bitte ignorieren.')
  // send button is the icon button next to the textarea (last button in footer)
  await page.locator('[role="dialog"] button:has(svg)').last().click()
  await page.waitForTimeout(700)
  const threadAfter = await page.locator('[role="dialog"]').getByText(/Nachrichten \(/).first().textContent().catch(() => '')
  const replyVisible = await page.locator('[role="dialog"]').getByText('Antwort vom QA-Agent — bitte ignorieren.').count()
  await page.screenshot({ path: resolve(outDir, '1-after-reply.png') })
  out.push({ step: 'reply', threadBefore: (threadBefore || '').trim(), threadAfter: (threadAfter || '').trim(), replyVisible })

  // ── H-3c: change status ────────────────────────────────────────────────
  await page.locator('[role="dialog"] button:has-text("Status ändern"), [role="dialog"] button:has-text("Offen"), [role="dialog"] button:has-text("In Bearbeitung")').first().click().catch(() => {})
  await page.waitForTimeout(400)
  // open the status dropdown (the control under "Status ändern")
  const statusText = await firstDialogText()
  out.push({ step: 'status-control-present', hasChangeStatus: /Status ändern/.test(statusText) })
  await page.keyboard.press('Escape').catch(() => {})
  await page.waitForTimeout(400)

  // ── H-3d: reload → persistence ─────────────────────────────────────────
  await page.reload({ waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  const rowsReload = await rowCount()
  const newStillThere = await page.locator('tbody tr').filter({ hasText: 'QA Testticket Persistenz' }).count()
  // open it again, confirm the reply survived
  const row2 = page.locator('tbody tr').filter({ hasText: 'QA Testticket Persistenz' }).first()
  await row2.evaluate((el) => el.click()).catch(async () => { await row2.click() })
  await page.waitForTimeout(900)
  const replyPersisted = await page.locator('[role="dialog"]').getByText('Antwort vom QA-Agent — bitte ignorieren.').count()
  await page.screenshot({ path: resolve(outDir, '2-after-reload.png') })
  out.push({ step: 'persistence', rowsReload, newStillThere, replyPersisted })

  out.push({ pageErrors: errs.slice(0, 6) })
} catch (e) {
  out.push({ error: String(e).split('\n')[0], pageErrors: errs.slice(0, 6) })
} finally {
  await ctx.close()
  await browser.close()
}

console.log(JSON.stringify(out, null, 2))
