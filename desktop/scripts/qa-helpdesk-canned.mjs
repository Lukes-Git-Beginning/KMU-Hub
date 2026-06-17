/**
 * QA — helpdesk H-5 Canned Responses CRUD against :5174.
 * Opens the "Vorlagen" slide-over, creates a template, checks it appears and
 * survives reload, then deletes one and checks the count drops.
 * Screenshots: .qa-screenshots/helpdesk-canned/
 *
 * Usage: node scripts/qa-helpdesk-canned.mjs
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots', 'helpdesk-canned')
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

// The canned panel is a fixed slide-over (z-50), not role=dialog → scope by its title.
const panel = () => page.locator('div').filter({ hasText: /Vorlagen verfügbar/ }).last()
const availableCount = async () => {
  const txt = await page.getByText(/\d+ Vorlagen verfügbar/).first().textContent().catch(() => '')
  const m = /(\d+) Vorlagen/.exec(txt || '')
  return m ? Number(m[1]) : -1
}
const openCanned = async () => {
  await page.locator('header button:has-text("Vorlagen"), button:has-text("Vorlagen")').first().click()
  await page.waitForTimeout(700)
}

const out = []
try {
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)

  await openCanned()
  const before = await availableCount()
  await page.screenshot({ path: resolve(outDir, '0-list.png') })

  // create — exact "Neu" (avoid the panel-covered header "Neues Ticket")
  await page.getByRole('button', { name: 'Neu', exact: true }).click()
  await page.waitForTimeout(500)
  await page.getByPlaceholder('Vorlagenname...').fill('QA Vorlage Persistenz')
  await page.locator('input[placeholder="/gruss"]').fill('/qa')
  await page.getByRole('button', { name: 'Erstellen', exact: true }).click()
  await page.waitForTimeout(700)
  const afterCreate = await availableCount()
  const newVisible = await page.getByText('QA Vorlage Persistenz').count()
  await page.screenshot({ path: resolve(outDir, '1-after-create.png') })
  out.push({ step: 'create', before, afterCreate, newVisible })

  // reload → persistence, then reopen
  await page.reload({ waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  await openCanned()
  const afterReload = await availableCount()
  const persisted = await page.getByText('QA Vorlage Persistenz').count()
  out.push({ step: 'persistence', afterReload, persisted })

  // delete the new one — scope to its .group card, hover, click trash (title="Löschen")
  const card = page.locator('.group').filter({ hasText: 'QA Vorlage Persistenz' }).first()
  await card.hover()
  await page.waitForTimeout(200)
  await card.locator('button[title="Löschen"]').first().click({ force: true }).catch((e) => errs.push('del:' + e))
  await page.waitForTimeout(600)
  const afterDelete = await availableCount()
  const goneNow = await page.getByText('QA Vorlage Persistenz').count()
  await page.screenshot({ path: resolve(outDir, '2-after-delete.png') })
  out.push({ step: 'delete', afterDelete, goneNow })

  out.push({ pageErrors: errs.slice(0, 6) })
} catch (e) {
  out.push({ error: String(e).split('\n')[0], pageErrors: errs.slice(0, 6) })
} finally {
  await ctx.close(); await browser.close()
}

console.log(JSON.stringify(out, null, 2))
