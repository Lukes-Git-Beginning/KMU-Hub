/**
 * QA script — Dashboard Phase 5: Widget-Gating by Module Feature Flags
 *
 * Scenarios:
 *  (a) Normal load — widget grid renders, all seeded widgets visible (fail-open/demo)
 *  (b) Picker open — widget list populated (fail-open/demo ensures all are shown)
 *  (c) Inject CRM=false via window.__cosmi_qa_flags__ + queryClient re-render trigger →
 *      CRM-mapped widgets disappear from Grid AND Picker; others remain
 *
 * Expectations: rawKeys: [], pageErrors: [], gating pass=true in (c)
 *
 * QA infrastructure:
 *  - window.__cosmi_queryClient__: exposed by App.tsx in DEV mode — triggers re-render
 *  - window.__cosmi_qa_flags__:    checked by useFeatureFlags before IS_DEMO (DEV only,
 *    tree-shaken in production) — allows per-key flag override in QA scripts
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/dashboard-gating')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const SEED_WIDGETS = `
  try {
    localStorage.setItem('cosmi-dashboard', JSON.stringify({
      state: {
        activeWidgets: ['recent-contacts', 'kpi-deals', 'my-tasks', 'time-clock', 'team-status'],
        layouts: [
          { i: 'recent-contacts', x: 0, y: 0, w: 4, h: 3, minW: 2, minH: 2 },
          { i: 'kpi-deals',       x: 4, y: 0, w: 4, h: 3, minW: 3, minH: 2 },
          { i: 'my-tasks',        x: 8, y: 0, w: 4, h: 4, minW: 3, minH: 3 },
          { i: 'time-clock',      x: 0, y: 3, w: 4, h: 3, minW: 3, minH: 2 },
          { i: 'team-status',     x: 4, y: 3, w: 4, h: 4, minW: 3, minH: 3 },
        ],
      },
      version: 0,
    }))
  } catch(e) {}
`

const RAW_RE = /(widgets\.|dashboard\.|moduleSettings\.)/
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 10)
  }, RAW_RE.source)
}

async function widgetGridText(page) {
  return page.evaluate(() => {
    const layout = document.querySelector('.layout')
    return layout ? layout.textContent || '' : ''
  })
}

// Inject QA flag overrides.
// Uses window.__cosmi_qa_flags__ (checked by useFeatureFlags before IS_DEMO in DEV builds).
// Then triggers a React Query re-render via queryClient.setQueryData.
async function injectFlagsForQA(page, flags) {
  return page.evaluate((flagsPayload) => {
    // 1. Set QA override — checked by useFeatureFlags before IS_DEMO
    window.__cosmi_qa_flags__ = flagsPayload
    // 2. Trigger re-render via queryClient.setQueryData(['feature-flags'])
    const qc = window.__cosmi_queryClient__
    if (!qc) return false
    qc.setQueryData(['feature-flags'], { version: 'qa-override', flags: flagsPayload })
    return true
  }, flags)
}

const browser = await chromium.launch()
const out = []

// ──────────────────────────────────────────────────────────────────────────────
// Scenario (a): Normal load → all seeded widgets visible (demo=fail-open)
// ──────────────────────────────────────────────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  await ctx.addInitScript(SEED_WIDGETS)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))

  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2000)

    await page.evaluate(() => {
      const el = document.querySelector('.layout')
      if (el) el.scrollIntoView({ behavior: 'instant' })
    })
    await page.waitForTimeout(600)

    const layoutExists = await page.evaluate(() => !!document.querySelector('.layout'))
    const gridText = await widgetGridText(page)
    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'a-normal-load.png') })

    const seededNames = ['Letzte Kontakte', 'Deal-Überblick', 'Meine Aufgaben', 'Stempeluhr', 'Team-Status']
    const visibleSeeded = seededNames.filter((n) => gridText.includes(n))

    out.push({
      scenario: 'a-normal-load',
      layoutExists,
      visibleWidgets: visibleSeeded,
      rawKeys: rk,
      pageErrors: errs.length,
      pass: layoutExists && visibleSeeded.length === 5,
      note: 'Demo mode = fail-open → all seeded widgets visible',
    })
  } catch (e) {
    out.push({ scenario: 'a-normal-load', error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ──────────────────────────────────────────────────────────────────────────────
// Scenario (b): Picker opens and shows widgets
// ──────────────────────────────────────────────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  await ctx.addInitScript(`
    try {
      localStorage.setItem('cosmi-dashboard', JSON.stringify({
        state: { activeWidgets: [], layouts: [] }, version: 0
      }))
    } catch(e) {}
  `)

  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))

  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2000)

    const editBtn = page.locator('button:has-text("Dashboard anpassen"), button:has-text("Bearbeiten")').first()
    const editVisible = await editBtn.isVisible().catch(() => false)
    if (editVisible) {
      await editBtn.click({ timeout: 5000 })
      await page.waitForTimeout(800)
    }

    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
    await page.waitForTimeout(400)

    const addBtn = page.locator('button:has-text("Widget hinzufügen")').first()
    const addVisible = await addBtn.isVisible().catch(() => false)

    let pickerNames = []
    let pickerRk = []

    if (addVisible) {
      await addBtn.click({ timeout: 5000 })
      await page.waitForTimeout(800)
      pickerNames = await page.evaluate(() => {
        const dialog = document.querySelector('[role="dialog"]')
        if (!dialog) return []
        return Array.from(dialog.querySelectorAll('button p.text-sm.font-medium'))
          .map((el) => el.textContent?.trim())
          .filter(Boolean)
      })
      pickerRk = await rawKeys(page)
      await page.screenshot({ path: resolve(outDir, 'b-picker-open.png') })
    } else {
      await page.screenshot({ path: resolve(outDir, 'b-no-add-btn.png') })
    }

    out.push({
      scenario: 'b-picker-populated',
      editBtnVisible: editVisible,
      addBtnVisible: addVisible,
      pickerWidgetCount: pickerNames.length,
      pickerWidgetNames: pickerNames,
      rawKeys: pickerRk,
      pageErrors: errs.length,
      pass: addVisible ? pickerNames.length >= 10 : null,
      note: 'All 19 widgets available in picker (demo = all flags on)',
    })
  } catch (e) {
    out.push({ scenario: 'b-picker-populated', error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ──────────────────────────────────────────────────────────────────────────────
// Scenario (c): QA flag override — disable CRM → CRM widgets hidden
// Uses window.__cosmi_qa_flags__ (DEV-only, tree-shaken in prod) which is
// checked BEFORE the IS_DEMO bypass in useFeatureFlags.isEnabled().
// ──────────────────────────────────────────────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  await ctx.addInitScript(SEED_WIDGETS)

  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))

  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2000)

    // Inject: CRM=false, everything else=true
    const allModuleFlags = {
      'modules.crm': false,
      'modules.finance': true,
      'modules.tasks': true,
      'modules.calendar': true,
      'modules.chat': true,
      'modules.team': true,
      'modules.zeiterfassung': true,
      'modules.documents': true,
      'modules.meetings': true,
      'modules.mail': true,
      'modules.dialer': true,
      'modules.helpdesk': true,
      'modules.projects': true,
      'modules.inventar': true,
      'modules.einkauf': true,
      'modules.produktion': true,
      'modules.berichte': true,
      'modules.formulare': true,
      'modules.wiki': true,
      'modules.rapporte': true,
      'modules.vermietung': true,
      'modules.vertraege': true,
      'modules.schichten': true,
      'modules.fuhrpark': true,
    }
    const injected = await injectFlagsForQA(page, allModuleFlags)
    await page.waitForTimeout(1000)

    // Scroll to widget grid
    await page.evaluate(() => {
      const el = document.querySelector('.layout')
      if (el) el.scrollIntoView({ behavior: 'instant' })
    })
    await page.waitForTimeout(600)

    const gridText = await widgetGridText(page)

    const crmNames = ['Letzte Kontakte', 'Deal Pipeline', 'Deal-Überblick']
    const crmVisibleInGrid = crmNames.filter((n) => gridText.includes(n))
    const nonCrmNames = ['Meine Aufgaben', 'Stempeluhr', 'Team-Status']
    const nonCrmVisibleInGrid = nonCrmNames.filter((n) => gridText.includes(n))

    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'c-crm-disabled-grid.png') })

    // Also open picker to verify CRM absent there too
    const editBtn = page.locator('button:has-text("Dashboard anpassen"), button:has-text("Bearbeiten")').first()
    const editVisible = await editBtn.isVisible().catch(() => false)
    if (editVisible) {
      await editBtn.click({ timeout: 5000 })
      await page.waitForTimeout(600)
    }
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
    await page.waitForTimeout(400)

    const addBtn = page.locator('button:has-text("Widget hinzufügen")').first()
    const addVisible = await addBtn.isVisible().catch(() => false)
    let pickerNames = []
    let pickerHasCrmWidgets = null

    if (addVisible) {
      await addBtn.click({ timeout: 5000 })
      await page.waitForTimeout(800)
      pickerNames = await page.evaluate(() => {
        const dialog = document.querySelector('[role="dialog"]')
        if (!dialog) return []
        return Array.from(dialog.querySelectorAll('button p.text-sm.font-medium'))
          .map((el) => el.textContent?.trim())
          .filter(Boolean)
      })
      pickerHasCrmWidgets = crmNames.some((n) => pickerNames.includes(n))
      await page.screenshot({ path: resolve(outDir, 'c-crm-disabled-picker.png') })
    } else {
      await page.screenshot({ path: resolve(outDir, 'c-no-add-btn.png') })
    }

    const pass =
      injected &&
      crmVisibleInGrid.length === 0 &&
      nonCrmVisibleInGrid.length >= 2 &&
      (pickerHasCrmWidgets === false || pickerHasCrmWidgets === null)

    out.push({
      scenario: 'c-crm-module-disabled',
      qaFlagsInjected: injected,
      crmWidgetsInGrid: crmVisibleInGrid,
      nonCrmWidgetsInGrid: nonCrmVisibleInGrid,
      pickerWidgetNames: pickerNames,
      pickerHasCrmWidgets,
      rawKeys: rk,
      pageErrors: errs.length,
      pass,
      note: 'CRM=false via __cosmi_qa_flags__ → crm widgets hidden; non-crm visible',
    })
  } catch (e) {
    out.push({ scenario: 'c-crm-module-disabled', error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
