/**
 * QA — formulare F-3: DnD field builder.
 * Opens a form → Bearbeiten → editor, reads the field order, drags field #1's
 * grip handle below field #3 and asserts the order changed + persisted. Also
 * adds a field via the palette. 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-f3')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

async function fieldOrder(page) {
  return page.$$eval('[aria-label="Feld verschieben"]', (els) =>
    els.map((h) => {
      const row = h.parentElement
      const label = row && row.querySelector('.truncate')
      return label ? label.textContent.trim() : ''
    }),
  )
}

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error') errs.push('console: ' + m.text()) })

const out = []
try {
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4000)

  // Open Kundenfeedback → Bearbeiten
  await page.locator('.grid [role="button"]:has-text("Kundenfeedback")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(900)
  await page.locator('[role="dialog"] button:has-text("Bearbeiten")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1200)

  const before = await fieldOrder(page)
  const handles = page.locator('[aria-label="Feld verschieben"]')
  const handleCount = await handles.count()
  await page.screenshot({ path: resolve(outDir, '0-editor.png'), fullPage: true })

  // Drag field #1 handle down to below field #3
  let moved = false
  if (handleCount >= 3) {
    const sb = await handles.nth(0).boundingBox()
    const db = await handles.nth(2).boundingBox()
    if (sb && db) {
      await page.mouse.move(sb.x + sb.width / 2, sb.y + sb.height / 2)
      await page.mouse.down()
      await page.mouse.move(sb.x + sb.width / 2, sb.y + sb.height / 2 + 8, { steps: 4 })
      await page.mouse.move(db.x + db.width / 2, db.y + db.height / 2 + 12, { steps: 12 })
      await page.mouse.move(db.x + db.width / 2, db.y + db.height / 2 + 20, { steps: 4 })
      await page.waitForTimeout(200)
      await page.mouse.up()
      await page.waitForTimeout(800)
      moved = true
    }
  }
  const after = await fieldOrder(page)
  await page.screenshot({ path: resolve(outDir, '1-editor-after-drag.png'), fullPage: true })
  out.push({ check: 'dnd', handleCount, moved, before, after, changed: JSON.stringify(before) !== JSON.stringify(after) })

  // Persistence: SAVE the reorder, then re-open to confirm it stuck (MSW PATCH)
  await page.locator('button:has-text("Speichern")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(1000)
  await page.locator('.grid [role="button"]:has-text("Kundenfeedback")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(700)
  await page.locator('[role="dialog"] button:has-text("Bearbeiten")').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(1000)
  const persisted = await fieldOrder(page)
  out.push({ check: 'persistence', persisted, matchesAfter: JSON.stringify(persisted) === JSON.stringify(after) })

  // Add a field via the palette
  await page.locator('button:has-text("Feld hinzufügen")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(400)
  await page.locator('button:has-text("Datum")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(600)
  const afterAdd = await fieldOrder(page)
  await page.screenshot({ path: resolve(outDir, '2-editor-after-add.png'), fullPage: true })
  out.push({ check: 'add-field', count: afterAdd.length, last: afterAdd[afterAdd.length - 1] })
} catch (e) {
  out.push({ fatal: String(e) })
} finally {
  console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
  await browser.close()
}
