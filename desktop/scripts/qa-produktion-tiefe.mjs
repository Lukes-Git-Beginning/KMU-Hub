/**
 * QA — Produktion Demo-Tiefe (Branchen-Block #7, letztes Modul).
 * Verifies: real per-order progress (was hard-coded 50 %), priority column,
 * order row → DetailModal, Laufkarte PDF, BOM back chain, REAL status
 * lifecycle (start/complete/cancel — was a dead toast), work-step check-off,
 * quality tab with order numbers (was UUID fragments) + QC detail chain,
 * stateful QC create (was lost on refetch), machine Gantt filled (was
 * permanently empty due to fixed Feb-2026 range) + machine modal, orders CSV,
 * stateful order create, settings panel, raw keys + pageerrors.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/produktion-tiefe')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`
const YEAR = new Date().getFullYear()

const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 950 }, acceptDownloads: true })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []

const dialogText = () => page.evaluate(() => Array.from(document.querySelectorAll('[role="dialog"]')).map((d) => d.textContent).join(' '))
const bodyText = () => page.evaluate(() => document.body.innerText)
const rawKeys = (txt) => (txt.match(/\b(produktion|shared|common|moduleSettings)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)

try {
  await page.goto(`${BASE}/#/produktion`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4500)
  await page.screenshot({ path: resolve(outDir, '01-auftraege.png') })

  // 1) Orders list: priority column + REAL progress (was 50 % for every
  //    running order — seeds give 40/25/75 %)
  const listTxt = await bodyText()
  out.push({
    step: 'orders list: priority column + real step-based progress',
    hasPriority: /P1 – Dringend|P1 –/.test(listTxt),
    hasRealProgress: /75%/.test(listTxt) && /40%/.test(listTxt) && !/50%/.test(listTxt),
    pass: /P1/.test(listTxt) && /75%/.test(listTxt) && /40%/.test(listTxt),
  })

  // 2) Order row → DetailModal (sections: progress, BOM, steps, scrap, QC, notes)
  //    — filter to a running order (default sort puts an old completed one first)
  await page.getByRole('button', { name: 'In Produktion', exact: true }).click()
  await page.waitForTimeout(600)
  await page.locator('tbody tr[role="button"]').first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(900)
  await page.screenshot({ path: resolve(outDir, '02-order-modal.png') })
  const d2 = await dialogText()
  out.push({
    step: 'order row → detail modal (sections)',
    hasSteps: /Schritten erledigt/.test(d2),
    hasBom: /SKU/.test(d2),
    hasScrapFromQc: /Ausschussrate/.test(d2),
    hasNotes: /Notizen/.test(d2),
    pass: /Schritten erledigt/.test(d2) && /SKU/.test(d2) && /Ausschussrate/.test(d2),
  })

  // 3) Laufkarte PDF export
  const [pdfDl] = await Promise.all([
    page.waitForEvent('download', { timeout: 6000 }),
    page.locator('[role="dialog"]').getByRole('button', { name: /Laufkarte/ }).click(),
  ])
  out.push({
    step: 'laufkarte pdf export (new)',
    filename: pdfDl.suggestedFilename(),
    pass: /^PA-.*\.pdf$/.test(pdfDl.suggestedFilename()),
  })

  // 4) BOM section click → BOM modal → back chain returns to order modal
  const orderTitle = await page.locator('[role="dialog"] h3').first().innerText()
  await page.locator('[role="dialog"] div[role="button"]').filter({ hasText: 'SKU' }).first().click()
  await page.waitForTimeout(900)
  const dBom = await dialogText()
  await page.screenshot({ path: resolve(outDir, '03-bom-via-backchain.png') })
  const hasBackBtn = await page.locator('[role="dialog"] button[aria-label="Zurück"]').count()
  if (hasBackBtn > 0) await page.locator('[role="dialog"] button[aria-label="Zurück"]').click()
  await page.waitForTimeout(700)
  const backTitle = await page.locator('[role="dialog"] h3').first().innerText().catch(() => '')
  out.push({
    step: 'order → bom modal → back chain',
    bomModal: /Positionen|Verwendet in/.test(dBom),
    backBtn: hasBackBtn > 0,
    backReturnsToOrder: backTitle === orderTitle,
    pass: hasBackBtn > 0 && /Verwendet in/.test(dBom) && backTitle === orderTitle,
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 5) REAL status change: planned order → Starten (was toast-only stub)
  await page.getByRole('button', { name: 'Geplant', exact: true }).click()
  await page.waitForTimeout(600)
  await page.locator('tbody tr[role="button"]').first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(700)
  await page.locator('[role="dialog"]').getByRole('button', { name: 'Starten', exact: true }).click()
  await page.waitForTimeout(1500)
  const d5 = await dialogText()
  const body5 = await bodyText()
  await page.screenshot({ path: resolve(outDir, '04-nach-starten.png') })
  out.push({
    step: 'status change planned → in_progress (real mutation, was dead toast)',
    modalBadgeUpdated: /In Produktion/.test(d5),
    toast: /gestartet/.test(body5),
    pass: /In Produktion/.test(d5) && /gestartet/.test(body5),
  })

  // 6) Work-step check-off (unused useUpdateWorkStep wired): start + complete step 1
  await page.locator('[role="dialog"] button[aria-label="Schritt starten"]').first().click()
  await page.waitForTimeout(1000)
  await page.locator('[role="dialog"] button[aria-label="Schritt abschließen"]').first().click()
  await page.waitForTimeout(1200)
  const d6 = await dialogText()
  await page.screenshot({ path: resolve(outDir, '05-schritt-abgehakt.png') })
  out.push({
    step: 'work-step check-off raises progress',
    stepDone: /1 von 3 Schritten erledigt/.test(d6),
    progressRaised: /33%/.test(d6),
    pass: /1 von 3 Schritten erledigt/.test(d6) && /33%/.test(d6),
  })

  // 7) Complete order (real mutation)
  await page.locator('[role="dialog"]').getByRole('button', { name: 'Abschließen', exact: true }).click()
  await page.waitForTimeout(1500)
  const d7 = await dialogText()
  await page.screenshot({ path: resolve(outDir, '06-nach-abschliessen.png') })
  out.push({
    step: 'complete order (real mutation)',
    badgeCompleted: /Abgeschlossen/.test(d7),
    pass: /Abgeschlossen/.test(d7),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 8) Cancel with inline confirm on the remaining planned order
  await page.locator('tbody tr[role="button"]').first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(600)
  await page.locator('[role="dialog"]').getByRole('button', { name: 'Stornieren', exact: true }).click()
  await page.waitForTimeout(400)
  const dConfirm = await dialogText()
  await page.screenshot({ path: resolve(outDir, '07-storno-confirm.png') })
  await page.locator('[role="dialog"]').getByRole('button', { name: 'Bestätigen', exact: true }).click()
  await page.waitForTimeout(1500)
  const d8 = await dialogText()
  out.push({
    step: 'cancel order with inline confirm',
    confirmShown: /wirklich stornieren/.test(dConfirm),
    badgeCancelled: /Storniert/.test(d8),
    pass: /wirklich stornieren/.test(dConfirm) && /Storniert/.test(d8),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)
  await page.getByRole('button', { name: 'Alle', exact: true }).click()
  await page.waitForTimeout(500)

  // 9) Orders CSV export
  const [csvDl] = await Promise.all([
    page.waitForEvent('download', { timeout: 6000 }),
    page.getByRole('button', { name: 'CSV', exact: true }).click(),
  ])
  out.push({
    step: 'orders csv export',
    filename: csvDl.suggestedFilename(),
    pass: /^produktionsauftraege-.*\.csv$/.test(csvDl.suggestedFilename()),
  })

  // 10) Quality tab: order numbers instead of UUID fragments + QC detail chain
  await page.getByRole('button', { name: /Qualität \(/ }).click()
  await page.waitForTimeout(800)
  const qTxt = await bodyText()
  await page.screenshot({ path: resolve(outDir, '08-qualitaet-tab.png') })
  await page.locator('tbody tr[role="button"]').first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(700)
  const dQc = await dialogText()
  await page.screenshot({ path: resolve(outDir, '09-qc-modal.png') })
  await page.locator('[role="dialog"]').getByRole('button', { name: /Auftrag öffnen/ }).click()
  await page.waitForTimeout(900)
  const dQcOrder = await dialogText()
  out.push({
    step: 'quality tab: order numbers + qc detail → order chain',
    orderNumbersInTab: new RegExp(`PA-${YEAR}-0\\d\\d`).test(qTxt),
    qcModal: /Prüfer/.test(dQc),
    chainToOrder: /Schritten erledigt|Fortschritt/.test(dQcOrder),
    pass: new RegExp(`PA-${YEAR}-0\\d\\d`).test(qTxt) && /Prüfer/.test(dQc) && /Fortschritt/.test(dQcOrder),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 11) Stateful QC create (was lost after refetch) — free-text inspector
  const qcCountBefore = (await bodyText()).match(/Qualität \((\d+)\)/)?.[1]
  await page.getByRole('button', { name: 'Qualitätsprüfung', exact: true }).click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(400)
  await page.locator('[role="dialog"] button[role="combobox"]').first().click()
  await page.waitForTimeout(300)
  await page.getByRole('option').first().click()
  await page.waitForTimeout(300)
  await page.getByPlaceholder(/prüfenden/).fill('QA Prüfer')
  await page.screenshot({ path: resolve(outDir, '10-qc-dialog.png') })
  await page.locator('[role="dialog"]').getByRole('button', { name: /Prüfung speichern/ }).click()
  await page.waitForTimeout(1500)
  const qcAfter = await bodyText()
  const qcCountAfter = qcAfter.match(/Qualität \((\d+)\)/)?.[1]
  out.push({
    step: 'stateful qc create (was vanishing after refetch)',
    before: qcCountBefore,
    after: qcCountAfter,
    inList: /QA Prüfer/.test(qcAfter),
    pass: Number(qcCountAfter) === Number(qcCountBefore) + 1 && /QA Prüfer/.test(qcAfter),
  })

  // 12) Machine Gantt filled (was permanently empty: fixed Feb-2026 range) +
  //     order numbers on blocks + machine modal with status select
  await page.getByRole('button', { name: /Maschinen \(/ }).click()
  await page.waitForTimeout(900)
  await page.screenshot({ path: resolve(outDir, '11-maschinen-gantt.png') })
  const ganttTxt = await bodyText()
  const blockCount = await page.locator('div[role="button"][title*="PA-"]').count()
  out.push({
    step: 'gantt filled with order-number blocks (was always empty)',
    blocks: blockCount,
    hasToday: /Heute/.test(ganttTxt),
    pass: blockCount >= 4 && /Heute/.test(ganttTxt),
  })
  // block click → order modal
  await page.locator('div[role="button"][title*="PA-"]').first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(700)
  const dBlock = await dialogText()
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)
  // machine row click → machine modal
  await page.locator('div[role="button"]').filter({ hasText: 'CNC-Fräse Alpha' }).first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(700)
  const dMachine = await dialogText()
  await page.screenshot({ path: resolve(outDir, '12-maschinen-modal.png') })
  out.push({
    step: 'gantt block → order modal · machine row → machine modal',
    blockOpensOrder: /Fortschritt/.test(dBlock),
    machineModal: /Maschinen-Details/.test(dMachine) && /Buchung/.test(dMachine),
    pass: /Fortschritt/.test(dBlock) && /Maschinen-Details/.test(dMachine),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 13) Stateful order create with tenant PA prefix (was vanishing after refetch)
  await page.getByRole('button', { name: 'Neuer Auftrag', exact: true }).click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(400)
  await page.locator('[role="dialog"] button[role="combobox"]').first().click()
  await page.waitForTimeout(300)
  await page.getByRole('option').first().click()
  await page.waitForTimeout(300)
  await page.locator('[role="dialog"] input[type="number"]').first().fill('7')
  await page.screenshot({ path: resolve(outDir, '13-neuer-auftrag-dialog.png') })
  await page.locator('[role="dialog"]').getByRole('button', { name: /Auftrag erstellen/ }).click()
  await page.waitForTimeout(1800)
  await page.getByRole('button', { name: /Aufträge \(/ }).click()
  await page.waitForTimeout(800)
  await page.getByRole('button', { name: 'Geplant', exact: true }).click()
  await page.waitForTimeout(600)
  const body13 = await bodyText()
  await page.screenshot({ path: resolve(outDir, '14-neuer-auftrag-in-liste.png') })
  out.push({
    step: 'stateful order create + tenant number prefix',
    newOrderInList: new RegExp(`PA-${YEAR}-\\d{4}-\\d{4}`).test(body13),
    pass: new RegExp(`PA-${YEAR}-\\d{4}-\\d{4}`).test(body13),
  })

  // 14) Settings panel registered (personal + tenant)
  await page.evaluate(() => {
    const el = Array.from(document.querySelectorAll('button, a, [role="button"]')).find((e) => /Modul-Einstellung/.test(e.textContent || ''))
    if (el) el.click()
  })
  try {
    await page.getByText('Nummernkreis-Präfix', { exact: false }).first().waitFor({ state: 'visible', timeout: 12000 })
  } catch { /* fällt auf Text-Assertion zurück */ }
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '15-settings-panel.png') })
  const txt14 = await bodyText()
  out.push({
    step: 'settings panel registered',
    hasPersonal: /Standard-Tab|Standard-Statusfilter/.test(txt14),
    hasTenant: /Nummernkreis-Präfix|Ausschuss-Warnschwelle|Qualitätsprüfung vor Abschluss/.test(txt14),
    pass: /Standard-Statusfilter/.test(txt14) && /Nummernkreis-Präfix/.test(txt14),
  })

  // 15) raw keys + pageerrors
  const fullTxt = await bodyText()
  out.push({ step: 'raw i18n keys', found: rawKeys(fullTxt), pass: rawKeys(fullTxt).length === 0 })
  out.push({ step: 'pageerrors', errs: errs.slice(0, 5), pass: errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).split('\n')[0], pass: false })
  await page.screenshot({ path: resolve(outDir, 'fatal.png') }).catch(() => {})
}

const allPass = out.every((o) => o.pass)
console.log(JSON.stringify({ allPass, results: out }, null, 2))
await ctx.close(); await b.close()
process.exit(allPass ? 0 : 1)
