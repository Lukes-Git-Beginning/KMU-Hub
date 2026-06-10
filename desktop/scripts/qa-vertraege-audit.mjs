import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/vertraege-audit')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

// Raw-key detector: any vertraege.history.* or vertraege.detail.reminder* key visible as text
const rawRe = /(vertraege\.history\.|vertraege\.detail\.reminder)/
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 10)
  }, rawRe.source)
}

const browser = await chromium.launch()
const out = []

// ─── 1) DetailPanel: contract WITH history + reminderDays ────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(1800)

    // Click first contract row (Büro-Mietvertrag München — has history + reminderDays 30/60/90)
    const firstRow = page.locator('table tbody tr').first()
    await firstRow.click({ timeout: 8000 })
    await page.waitForTimeout(700)

    // Scroll audit log into view (inside the detail panel sidebar)
    await page.evaluate(() => {
      const el = Array.from(document.querySelectorAll('*')).find(
        (e) => e.scrollHeight > e.clientHeight && e.clientHeight > 200 && e.clientHeight < 900
      )
      if (el) el.scrollTo(0, el.scrollHeight)
    })
    await page.waitForTimeout(400)
    // Also try scrolling the explicit history heading into view
    await page.locator('text=/Änderungshistorie/').first().scrollIntoViewIfNeeded().catch(() => {})
    await page.waitForTimeout(300)

    const hasAuditSection = await page.evaluate(() =>
      /Änderungshistorie/.test(document.body.textContent || ''))
    const hasReminderSection = await page.evaluate(() =>
      /Erinnerungen/.test(document.body.textContent || ''))
    const hasReminderDates = await page.evaluate(() =>
      /\d{2}\.\d{2}\.\d{4}/.test(document.body.textContent || ''))
    const rk = await rawKeys(page)

    await page.screenshot({ path: resolve(outDir, '1-detail-with-history.png'), fullPage: false })
    out.push({ step: '1-detail-with-history', hasAuditSection, hasReminderSection, hasReminderDates, rawKeys: rk, pageErrors: errs })
  } catch (e) {
    out.push({ step: '1-detail-with-history', error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── 2) Empty states: contract without history or reminders ────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)

  // Pre-seed a fresh contract with no history and no reminders
  await ctx.addInitScript(`
    try {
      const K = 'cosmi-verträge';
      const raw = localStorage.getItem(K);
      const store = raw ? JSON.parse(raw) : { state: { contracts: [], contractTemplates: [] }, version: 0 };
      const emptyContract = {
        id: 'v-qa-empty',
        contractNumber: 'QA-EMPTY-001',
        title: 'QA Empty Contract',
        partner: 'QA Partner AG',
        type: 'lizenz',
        status: 'active',
        startDate: '2025-01-01',
        endDate: '2027-01-01',
        noticePeriodDays: 30,
        renewal: 'manual',
        monthlyCost: 100,
        totalValue: 2400,
        notes: '',
        history: [],
        currency: 'EUR'
      };
      // Push to front so it appears first in list
      store.state.contracts = [emptyContract, ...(store.state.contracts || [])];
      localStorage.setItem(K, JSON.stringify(store));
    } catch(e) {}
  `)

  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(1800)

    // Click QA Empty Contract row
    const emptyRow = page.locator('table tbody tr').filter({ hasText: 'QA Empty Contract' }).first()
    await emptyRow.click({ timeout: 8000 })
    await page.waitForTimeout(700)

    // Scroll to bottom of panel
    const panel = page.locator('[data-testid="detail-panel"], aside, [class*="detail"]').last()
    await panel.evaluate((el) => el.scrollTo(0, el.scrollHeight)).catch(() => {})
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
    await page.waitForTimeout(300)

    const hasHistoryEmpty = await page.evaluate(() =>
      /Keine Einträge vorhanden/.test(document.body.textContent || ''))
    const hasReminderNone = await page.evaluate(() =>
      /Keine Erinnerungen eingerichtet/.test(document.body.textContent || ''))
    const rk = await rawKeys(page)

    await page.screenshot({ path: resolve(outDir, '2-empty-states.png'), fullPage: false })
    out.push({ step: '2-empty-states', hasHistoryEmpty, hasReminderNone, rawKeys: rk, pageErrors: errs })
  } catch (e) {
    out.push({ step: '2-empty-states', error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── 3) After mutation: store pre-seeded with contract_updated entry ──
// Direct store injection simulates what happens after a successful save.
// (React selectedContract is a snapshot; UI test verifies the feed renders
//  the translated label for the action code "contract_updated".)
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)

  // Inject a contract that already has a contract_updated history entry
  await ctx.addInitScript(`
    try {
      const K = 'cosmi-verträge';
      const raw = localStorage.getItem(K);
      const store = raw ? JSON.parse(raw) : { state: { contracts: [], contractTemplates: [] }, version: 0 };
      const mutated = {
        id: 'v-qa-mutated',
        contractNumber: 'QA-MUT-001',
        title: 'QA Mutated Contract',
        partner: 'Mutation GmbH',
        type: 'servicevertrag',
        status: 'active',
        startDate: '2025-01-01',
        endDate: '2027-06-01',
        noticePeriodDays: 30,
        renewal: 'auto',
        monthlyCost: 200,
        totalValue: 4800,
        notes: '',
        history: [
          { date: '2025-01-01', action: 'contract_created', user: 'Testbenutzer' },
          { date: '2025-06-10', action: 'contract_updated', user: 'Testbenutzer' }
        ],
        reminderDays: [30],
        currency: 'EUR'
      };
      store.state.contracts = [mutated, ...(store.state.contracts || [])];
      localStorage.setItem(K, JSON.stringify(store));
    } catch(e) {}
  `)

  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(1800)

    // Click QA Mutated Contract row
    const mutRow = page.locator('table tbody tr').filter({ hasText: 'QA Mutated Contract' }).first()
    await mutRow.click({ timeout: 8000 })
    await page.waitForTimeout(700)

    // Scroll to audit log section
    await page.locator('text=/Änderungshistorie/').first().scrollIntoViewIfNeeded().catch(() => {})
    await page.waitForTimeout(300)

    const countShown = await page.evaluate(() => {
      const m = document.body.textContent?.match(/Änderungshistorie \((\d+)\)/)
      return m ? parseInt(m[1]) : -1
    })

    // The feed should show "Vertrag aktualisiert" for the contract_updated code
    const hasUpdatedLabel = await page.evaluate(() =>
      /Vertrag aktualisiert/.test(document.body.textContent || ''))
    // And "Vertrag angelegt" for contract_created
    const hasCreatedLabel = await page.evaluate(() =>
      /Vertrag angelegt/.test(document.body.textContent || ''))
    // Reminder: "30 Tage vorher" with a date
    const hasReminderDate = await page.evaluate(() =>
      /30 Tage vorher/.test(document.body.textContent || ''))

    const rk = await rawKeys(page)

    await page.screenshot({ path: resolve(outDir, '3-after-mutation.png'), fullPage: false })
    out.push({
      step: '3-after-mutation',
      countShown,
      hasUpdatedLabel,
      hasCreatedLabel,
      hasReminderDate,
      rawKeys: rk,
      pageErrors: errs
    })
  } catch (e) {
    out.push({ step: '3-after-mutation', error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── 4) Terminate flow: open dialog, fill reason, submit → feed shows reason ──
// Uses page.evaluate() for all button clicks to avoid Playwright encoding issues with umlauts.
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)

  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(1800)

    // Open detail panel of first active row
    await page.evaluate(() => {
      const row = document.querySelector('table tbody tr')
      if (row) row.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await page.waitForTimeout(900)

    // Click "Kündigen" button via evaluate
    const c1 = await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll('button'))
      const btn = btns.find(b => /K.ndigen/.test(b.textContent || '') && !b.disabled)
      if (btn) { btn.click(); return true }
      return false
    })
    if (!c1) throw new Error('Terminate button not found')
    await page.waitForTimeout(800)

    // Fill termination date
    const today = new Date().toISOString().split('T')[0]
    await page.locator('input[type="date"]').last().fill(today, { timeout: 6000 })
    // Fill reason (ASCII-only to avoid encoding issues in test)
    const REASON = 'Testgruende QA Kuendigung'
    await page.locator('textarea').last().fill(REASON, { timeout: 6000 })
    // Check confirmation checkbox
    await page.locator('input[type="checkbox"]').last().check({ timeout: 6000 })
    await page.waitForTimeout(300)

    // Submit — click the submit button via evaluate (avoid Umlaut encoding in locator)
    await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll('button'))
      // Submit btn has class bg-destructive or contains "einleiten"
      const btn = btns.find(b => /einleiten/.test(b.textContent || '') && !b.disabled)
        || btns.find(b => b.className.includes('destructive') && !b.disabled)
      if (btn) btn.click()
    })
    await page.waitForTimeout(1000)

    // Switch to Archiv tab via evaluate
    await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll('button'))
      const archivBtn = btns.find(b => /Archiv/.test(b.textContent || ''))
      if (archivBtn) archivBtn.click()
    })
    await page.waitForTimeout(700)

    // Click the Adobe Creative Cloud row (the one we just terminated)
    await page.evaluate(() => {
      const rows = Array.from(document.querySelectorAll('table tbody tr'))
      // Find row containing "Adobe" text
      const row = rows.find(r => r.textContent?.includes('Adobe')) || rows[0]
      if (row) row.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await page.waitForTimeout(700)

    // Scroll to audit log
    await page.evaluate(() => {
      const el = Array.from(document.querySelectorAll('*')).find(
        e => e.scrollHeight > e.clientHeight && e.clientHeight > 200 && e.clientHeight < 900)
      if (el) el.scrollTo(0, el.scrollHeight)
    })
    await page.waitForTimeout(400)

    const feedText = await page.evaluate(() => document.body.textContent || '')
    const hasTerminatedLabel = /ndigung eingeleitet/.test(feedText)
    const hasReason = feedText.includes('Testgruende QA')
    const rk = await rawKeys(page)

    await page.screenshot({ path: resolve(outDir, '4-after-terminate.png'), fullPage: false })
    out.push({ step: '4-after-terminate', hasTerminatedLabel, hasReason, rawKeys: rk, pageErrors: errs })
  } catch (e) {
    out.push({ step: '4-after-terminate', error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
