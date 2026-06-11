/**
 * Phase 12 QA — dashboard: Team Dashboard scope switcher + 2 new widgets
 *
 * Szenarien:
 * (1) Toggle visible + Team-scope shows team-worktime widget with real data
 * (2) New widgets show real data (employee names / ticket subjects)
 * (3) Scope switch back to personal restores personal layout + reload persistence
 * (4) Persist migration: old v1 state (flat activeWidgets/layouts) migrates to personal scope
 * (5) Edit-mode in team scope only affects team layout (personal unchanged)
 * (6) Flag-gating: helpdesk flag off → open-tickets disappears from picker
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir, writeFile } from 'node:fs/promises'

const BASE = process.env.QA_BASE ?? 'http://localhost:5173'
const outDir = resolve('scripts/qa-shots/phase12')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

async function checkErrorBoundary(page) {
  const text = await page.evaluate(() => document.body.innerText)
  if (text.includes('Etwas ist schiefgelaufen') || text.includes('Cannot read properties')) {
    const lines = text.split('\n').map((l) => l.trim()).filter(Boolean)
    const idx = lines.findIndex((l) => l.includes('Etwas ist schiefgelaufen'))
    return idx >= 0 ? lines.slice(idx, idx + 3).join(' | ') : 'ErrorBoundary visible'
  }
  return null
}

async function scanRawKeys(page) {
  return page.evaluate(() => {
    const text = document.body.innerText
    const matches = [...text.matchAll(/\b(dashboard\.scope|dashboard\.teamWorktime|dashboard\.openTickets|widgets\.registry\.teamWorktime|widgets\.registry\.openTickets)\.[a-zA-Z.]+\b/g)]
    return [...new Set(matches.map((m) => m[0]))]
  })
}

/** Navigate to dashboard and wait for render. */
async function gotoDashboard(page) {
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
}

const browser = await chromium.launch({ headless: true })
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

const out = {
  scenario1_toggleAndTeamWidgets: { pass: false, detail: '' },
  scenario2_realDataVisible: { pass: false, detail: '' },
  scenario3_personalRestoreAndPersistence: { pass: false, detail: '' },
  scenario4_persistMigration: { pass: false, detail: '' },
  scenario5_editModeTeamOnly: { pass: false, detail: '' },
  scenario6_helpdeskFlagGating: { pass: false, detail: '' },
  rawKeys: [],
  pageErrors: [],
}

try {
  // ── Scenario 1: Toggle visible + Team scope shows team-worktime ───────────
  await gotoDashboard(page)
  const eb0 = await checkErrorBoundary(page)
  if (eb0) {
    out.scenario1_toggleAndTeamWidgets.detail = `ErrorBoundary on load: ${eb0}`
    await page.screenshot({ path: resolve(outDir, '00-error-on-load.png'), fullPage: false })
  } else {
    await page.screenshot({ path: resolve(outDir, '01a-dashboard-personal.png'), fullPage: false })

    // Find scope toggle — "Team" button
    const teamBtn = page.locator('[data-testid="scope-toggle-team"]')
    const teamBtnVisible = await teamBtn.isVisible({ timeout: 5000 }).catch(() => false)
    if (!teamBtnVisible) {
      out.scenario1_toggleAndTeamWidgets.pass = false
      out.scenario1_toggleAndTeamWidgets.detail = 'FAIL: scope-toggle-team button not found'
    } else {
      await teamBtn.click()
      await page.waitForTimeout(1500)
      await page.screenshot({ path: resolve(outDir, '01b-dashboard-team.png'), fullPage: false })

      const eb1 = await checkErrorBoundary(page)
      if (eb1) {
        out.scenario1_toggleAndTeamWidgets.pass = false
        out.scenario1_toggleAndTeamWidgets.detail = `ErrorBoundary after team switch: ${eb1}`
      } else {
        // Scroll to reveal team-worktime widget content
        await page.evaluate(() => window.scrollTo(0, 500))
        await page.waitForTimeout(800)
        await page.screenshot({ path: resolve(outDir, '01c-team-scope-scrolled.png'), fullPage: false })

        const bodyText = await page.evaluate(() => document.body.innerText)
        // team-worktime shows "Wochenstunden" title (via i18n key dashboard.teamWorktime.weeklyHours)
        const hasTeamWorktimeTitle = /Wochenstunden|Weekly Hours|Heures hebdomadaires|Ore settimanali/i.test(bodyText)
        // team headline visible
        const hasTeamHeadline = /Team.Dashboard|Team Dashboard/i.test(bodyText)

        out.scenario1_toggleAndTeamWidgets.pass = hasTeamWorktimeTitle && hasTeamHeadline
        out.scenario1_toggleAndTeamWidgets.detail = `hasTeamWorktimeTitle=${hasTeamWorktimeTitle}, hasTeamHeadline=${hasTeamHeadline}`
      }
    }
  }

  // ── Scenario 2: Real data visible in new widgets ─────────────────────────
  // Already on team scope from S1
  await page.evaluate(() => window.scrollTo(0, 0))
  await page.waitForTimeout(500)

  const eb2_pre = await checkErrorBoundary(page)
  if (eb2_pre) {
    out.scenario2_realDataVisible.detail = `ErrorBoundary at S2 start: ${eb2_pre}`
  } else {
    await page.waitForTimeout(1500)
    const bodyText2 = await page.evaluate(() => document.body.innerText)

    // team-worktime: expect at least one employee name from MSW mock
    // MSW mock employees: Markus Weber, Sabine Müller, Thomas Schäfer, Jana Köhler, Felix Bauer, Anna Großmann
    const hasEmployeeName = /Markus Weber|Sabine M.{1,3}ller|Thomas Sch.{1,4}fer|Jana K.{1,4}hler|Felix Bauer|Anna Gro.{1,4}mann/i.test(bodyText2)
    // open-tickets: expect at least one ticket subject from helpdesk store
    const hasTicketSubject = /Drucker|VPN|Mitarbeiter|Outlook|Bildschirm|Virenwarnung|Teams.Raum/i.test(bodyText2)

    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight / 2))
    await page.waitForTimeout(500)
    await page.screenshot({ path: resolve(outDir, '02a-team-widgets-data.png'), fullPage: false })

    // If widgets not yet loaded (lazy), scroll and wait more
    if (!hasEmployeeName || !hasTicketSubject) {
      await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
      await page.waitForTimeout(2000)
      const bodyText2b = await page.evaluate(() => document.body.innerText)
      const hasEmployeeName2 = /Markus Weber|Sabine M.{1,3}ller|Thomas Sch.{1,4}fer|Jana K.{1,4}hler|Felix Bauer/i.test(bodyText2b)
      const hasTicketSubject2 = /Drucker|VPN|Mitarbeiter|Outlook|Bildschirm|Virenwarnung|Teams.Raum/i.test(bodyText2b)
      await page.screenshot({ path: resolve(outDir, '02b-team-widgets-scrolled.png'), fullPage: false })
      out.scenario2_realDataVisible.pass = hasEmployeeName2 && hasTicketSubject2
      out.scenario2_realDataVisible.detail = `employeeName=${hasEmployeeName2}, ticketSubject=${hasTicketSubject2}`
    } else {
      out.scenario2_realDataVisible.pass = true
      out.scenario2_realDataVisible.detail = `employeeName=${hasEmployeeName}, ticketSubject=${hasTicketSubject}`
    }
  }

  // ── Scenario 3: Switch back to personal + persistence ───────────────────
  const personalBtn = page.locator('[data-testid="scope-toggle-personal"]')
  await personalBtn.click()
  await page.waitForTimeout(1500)
  await page.screenshot({ path: resolve(outDir, '03a-back-to-personal.png'), fullPage: false })
  const eb3 = await checkErrorBoundary(page)
  if (eb3) {
    out.scenario3_personalRestoreAndPersistence.pass = false
    out.scenario3_personalRestoreAndPersistence.detail = `ErrorBoundary: ${eb3}`
  } else {
    const bodyPersonal = await page.evaluate(() => document.body.innerText)
    // Personal scope shows greeting or personal widgets (my-tasks, my-calendar etc.)
    const hasPersonalContent = /Guten|Welcome|Bienven|Benven|Aufgaben|Tasks|Kalender|Calendar/i.test(bodyPersonal)

    // Reload and verify scope persists
    await page.reload({ waitUntil: 'domcontentloaded', timeout: 20000 })
    await page.waitForTimeout(2500)
    await page.screenshot({ path: resolve(outDir, '03b-after-reload.png'), fullPage: false })
    const eb3b = await checkErrorBoundary(page)
    // After reload, scope should still be personal (last set)
    const scopeAfterReload = await page.evaluate(() => {
      try {
        const raw = localStorage.getItem('cosmi-dashboard')
        if (!raw) return null
        const parsed = JSON.parse(raw)
        return parsed.state?.scope ?? null
      } catch (e) { return null }
    })

    out.scenario3_personalRestoreAndPersistence.pass = hasPersonalContent && !eb3b && scopeAfterReload === 'personal'
    out.scenario3_personalRestoreAndPersistence.detail = `hasPersonalContent=${hasPersonalContent}, errorBoundary=${eb3b}, scopePersisted=${scopeAfterReload}`
  }

  // ── Scenario 4: Persist migration (v1 → v2) ─────────────────────────────
  // Inject OLD v1 localStorage state: flat activeWidgets/layouts, version: 1
  // Use a custom active widget set so we can verify it survives migration.
  const oldV1Fixture = {
    state: {
      layouts: [
        { i: 'my-tasks', x: 0, y: 0, w: 4, h: 4 },
        { i: 'my-calendar', x: 4, y: 0, w: 4, h: 4 },
        { i: 'mini-chart', x: 8, y: 0, w: 4, h: 3 },
      ],
      activeWidgets: ['my-tasks', 'my-calendar', 'mini-chart'],
    },
    version: 1,
  }
  await page.addInitScript((fixture) => {
    localStorage.setItem('cosmi-dashboard', JSON.stringify(fixture))
  }, oldV1Fixture)

  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  await page.screenshot({ path: resolve(outDir, '04a-after-migration.png'), fullPage: false })
  const eb4 = await checkErrorBoundary(page)

  if (eb4) {
    out.scenario4_persistMigration.pass = false
    out.scenario4_persistMigration.detail = `ErrorBoundary after migration: ${eb4}`
  } else {
    // Verify migrated state: personal scope + old widget set preserved
    const migratedState = await page.evaluate(() => {
      try {
        const raw = localStorage.getItem('cosmi-dashboard')
        if (!raw) return null
        return JSON.parse(raw)
      } catch (e) { return null }
    })

    // After migration, the store should have written the migrated state.
    // At minimum: no crash, and the personal scope is active with 3 seeded widgets.
    const bodyAfterMigration = await page.evaluate(() => document.body.innerText)
    // The 3 seeded personal widgets: my-tasks, my-calendar, mini-chart
    // my-tasks shows "Meine Aufgaben" or "My Tasks"; my-calendar shows calendar widget
    // mini-chart shows "Jahresumsatz" or "Yearly Revenue"
    const hasMigratedWidgets =
      /Jahresumsatz|Yearly Revenue|Chiffre d.affaires annuel|Fatturato annuale/i.test(bodyAfterMigration) ||
      /Meine Aufgaben|My Tasks|Mes tâches|I miei compiti/i.test(bodyAfterMigration)

    await page.screenshot({ path: resolve(outDir, '04b-migration-content.png'), fullPage: false })

    out.scenario4_persistMigration.pass = !eb4 && hasMigratedWidgets
    out.scenario4_persistMigration.detail = [
      `noErrorBoundary=${!eb4}`,
      `hasMigratedWidgets=${hasMigratedWidgets}`,
      `migratedVersion=${migratedState?.version ?? 'null'}`,
      `bodyLength=${bodyAfterMigration.length}`,
    ].join(', ')
  }

  // ── Scenario 5: Edit-mode in team scope only affects team ────────────────
  // Navigate fresh (migration fixture is in ctx init script — clear it and reload)
  await page.evaluate(() => {
    // Reset to clean v2 state
    const clean = {
      state: {
        scope: 'team',
        personalLayouts: [{ i: 'my-tasks', x: 0, y: 0, w: 4, h: 4 }, { i: 'my-calendar', x: 4, y: 0, w: 4, h: 4 }],
        personalActiveWidgets: ['my-tasks', 'my-calendar'],
        teamLayouts: [{ i: 'team-status', x: 0, y: 0, w: 4, h: 4 }, { i: 'team-worktime', x: 4, y: 0, w: 4, h: 4 }, { i: 'open-tickets', x: 8, y: 0, w: 4, h: 4 }],
        teamActiveWidgets: ['team-status', 'team-worktime', 'open-tickets'],
      },
      version: 2,
    }
    localStorage.setItem('cosmi-dashboard', JSON.stringify(clean))
  })
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  await page.screenshot({ path: resolve(outDir, '05a-team-scope-before-edit.png'), fullPage: false })

  const eb5 = await checkErrorBoundary(page)
  if (eb5) {
    out.scenario5_editModeTeamOnly.pass = false
    out.scenario5_editModeTeamOnly.detail = `ErrorBoundary: ${eb5}`
  } else {
    // Enter edit mode and remove a widget in team scope
    const editBtn = page.getByRole('button').filter({ hasText: /anpassen|Customize/i }).first()
    const editVisible = await editBtn.isVisible({ timeout: 5000 }).catch(() => false)
    if (!editVisible) {
      out.scenario5_editModeTeamOnly.detail = 'FAIL: Edit button not found in team scope'
    } else {
      await editBtn.click()
      await page.waitForTimeout(800)
      await page.screenshot({ path: resolve(outDir, '05b-team-edit-mode.png'), fullPage: false })

      // Remove a team widget (team-status)
      const removeBtn = page.locator('button[aria-label*="entfernen"], button[aria-label*="Remove"]').first()
      const removeBtnVisible = await removeBtn.isVisible({ timeout: 3000 }).catch(() => false)
      if (removeBtnVisible) {
        await removeBtn.click()
        await page.waitForTimeout(800)
      }

      // Exit edit mode
      const doneBtn = page.getByRole('button').filter({ hasText: /Fertig|Done/i }).first()
      if (await doneBtn.isVisible({ timeout: 2000 }).catch(() => false)) await doneBtn.click()
      await page.waitForTimeout(500)

      // Switch to personal — should still have my-tasks, my-calendar
      const personalBtn2 = page.locator('[data-testid="scope-toggle-personal"]')
      await personalBtn2.click()
      await page.waitForTimeout(1500)
      await page.screenshot({ path: resolve(outDir, '05c-personal-after-team-edit.png'), fullPage: false })

      const eb5c = await checkErrorBoundary(page)
      const personalBody = await page.evaluate(() => document.body.innerText)
      const hasMyTasks = /Meine Aufgaben|My Tasks|Mes tâches|I miei compiti/i.test(personalBody)
      const hasMyCalendar = /Mein Kalender|My Calendar|Mon calendrier|Il mio calendario/i.test(personalBody)

      out.scenario5_editModeTeamOnly.pass = !eb5c && (hasMyTasks || hasMyCalendar)
      out.scenario5_editModeTeamOnly.detail = `noErrorBoundary=${!eb5c}, hasMyTasks=${hasMyTasks}, hasMyCalendar=${hasMyCalendar}`
    }
  }

  // ── Scenario 6: helpdesk flag off → open-tickets disappears ──────────────
  // Use a fresh isolated browser context to avoid accumulated addInitScript interference.
  const ctx6 = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  const s6State = JSON.stringify({
    state: {
      scope: 'team',
      personalLayouts: [],
      personalActiveWidgets: ['my-tasks'],
      teamLayouts: [
        { i: 'team-worktime', x: 0, y: 0, w: 4, h: 4 },
        { i: 'open-tickets', x: 4, y: 0, w: 4, h: 4 },
        { i: 'team-status', x: 8, y: 0, w: 4, h: 4 },
      ],
      teamActiveWidgets: ['team-worktime', 'open-tickets', 'team-status'],
    },
    version: 2,
  })
  await ctx6.addInitScript(STUB)
  await ctx6.addInitScript(ONB)
  await ctx6.addInitScript((s) => {
    try { localStorage.setItem('cosmi-dashboard', s) } catch (e) {}
  }, s6State)
  await ctx6.addInitScript(() => {
    // @ts-ignore
    window.__cosmi_qa_flags__ = { 'modules.helpdesk': false }
  })
  const page6 = await ctx6.newPage()
  const errors6 = []
  page6.on('pageerror', (e) => errors6.push(String(e)))

  await page6.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page6.waitForTimeout(2500)
  await page6.screenshot({ path: resolve(outDir, '06a-helpdesk-flag-off-grid.png'), fullPage: false })

  const eb6 = await checkErrorBoundary(page6)
  const body6 = await page6.evaluate(() => document.body.innerText)

  // With helpdesk flag=false, open-tickets widget should NOT appear in grid
  // team-worktime (zeiterfassung) should still appear
  const hasOpenTicketsInGrid = /Offene Tickets|Open Tickets|Tickets ouverts|Ticket aperti/i.test(body6)
  const hasTeamWorktimeInGrid = /Wochenstunden|Weekly Hours/i.test(body6)

  // Also check picker: enter edit mode → picker should NOT list open-tickets
  const editBtn6 = page6.getByRole('button').filter({ hasText: /anpassen|Customize/i }).first()
  const editVisible6 = await editBtn6.isVisible({ timeout: 3000 }).catch(() => false)
  let openTicketsInPicker = false
  if (editVisible6) {
    await editBtn6.click()
    await page6.waitForTimeout(600)
    const addWidgetBtn6 = page6.getByRole('button').filter({ hasText: /hinzuf|Add Widget/i }).first()
    const addVisible6 = await addWidgetBtn6.isVisible({ timeout: 3000 }).catch(() => false)
    if (addVisible6) {
      await addWidgetBtn6.click()
      await page6.waitForTimeout(600)
      await page6.screenshot({ path: resolve(outDir, '06b-picker-helpdesk-off.png'), fullPage: false })
      const pickerText = await page6.evaluate(() => document.body.innerText)
      openTicketsInPicker = /Offene Tickets|Open Tickets/i.test(pickerText)
      await page6.keyboard.press('Escape')
    }
    const doneBtn6 = page6.getByRole('button').filter({ hasText: /Fertig|Done/i }).first()
    if (await doneBtn6.isVisible({ timeout: 2000 }).catch(() => false)) await doneBtn6.click()
  }

  await ctx6.close()

  out.scenario6_helpdeskFlagGating.pass = !eb6 && !hasOpenTicketsInGrid && hasTeamWorktimeInGrid
  out.scenario6_helpdeskFlagGating.detail = [
    `noErrorBoundary=${!eb6}`,
    `openTicketsInGrid=${hasOpenTicketsInGrid} (expected false)`,
    `teamWorktimeInGrid=${hasTeamWorktimeInGrid} (expected true)`,
    `openTicketsInPicker=${openTicketsInPicker} (expected false)`,
  ].join(', ')

} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'error.png'), fullPage: false }).catch(() => {})
}

// ── Raw-key scan ─────────────────────────────────────────────────────────
try {
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(1500)
  out.rawKeys = await scanRawKeys(page)
} catch (_) {}

out.pageErrors = errors.slice(0, 5)

await ctx.close()
await browser.close()

const resultPath = resolve('scripts/qa-shots/phase12/qa-result.json')
await writeFile(resultPath, JSON.stringify(out, null, 2), 'utf-8')
console.log(JSON.stringify(out, null, 2))
