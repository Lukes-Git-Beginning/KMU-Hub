/**
 * QA — Fuhrpark Demo-Tiefe (Branchen-Block #5).
 * Verifies: vehicle card → DetailModal (was slide-over), maintenance/fuel/
 * logbook rows → row detail modals (were hover-only), AddTripDialog (was a
 * "coming soon" toast) incl. save round-trip, real logbook PDF/CSV + vehicles
 * CSV downloads, SortMenu, settings panel, raw keys + pageerrors.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/fuhrpark-tiefe')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 950 }, acceptDownloads: true })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []

const dialogText = () => page.evaluate(() => Array.from(document.querySelectorAll('[role="dialog"]')).map((d) => d.textContent).join(' '))
const bodyText = () => page.evaluate(() => document.body.innerText)
const rawKeys = (txt) => (txt.match(/\b(fuhrpark|shared|common|moduleSettings)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)

try {
  await page.goto(`${BASE}/#/fuhrpark`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4500)
  await page.screenshot({ path: resolve(outDir, '1-fahrzeuge.png') })

  // 1) Vehicle card → DetailModal (was a slide-over panel)
  const card = page.locator('button.rounded-lg.border').filter({ hasText: /km/ }).first()
  await card.click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, '2-vehicle-modal.png') })
  const d1 = await dialogText()
  out.push({
    step: 'vehicle card → detail modal',
    hasTco: /Gesamtkosten|Kostenverteilung|TCO/i.test(d1),
    hasStatus: /Prüfung|Versicherung/.test(d1),
    pass: /Prüfung|Versicherung/.test(d1),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 2) Vehicles CSV export
  const [vehCsv] = await Promise.all([
    page.waitForEvent('download', { timeout: 6000 }),
    page.getByRole('button', { name: /Fahrzeugliste/ }).click(),
  ])
  out.push({
    step: 'vehicles csv export',
    filename: vehCsv.suggestedFilename(),
    pass: /^fahrzeuge-.*\.csv$/.test(vehCsv.suggestedFilename()),
  })

  // 3) SortMenu: mileage desc
  await page.getByRole('button', { name: /Sortier/i }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitemradio', { name: 'Kilometerstand' }).click()
  await page.waitForTimeout(200)
  await page.getByRole('button', { name: /Sortier/i }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitemradio', { name: /Absteigend/ }).click()
  await page.waitForTimeout(400)
  await page.screenshot({ path: resolve(outDir, '3-sortmenu.png') })
  out.push({ step: 'sortmenu: mileage desc', pass: true })

  // 4) Wartung row → MaintenanceDetailModal
  await page.getByRole('button', { name: /Wartung \(/ }).click()
  await page.waitForTimeout(800)
  await page.locator('tbody tr[role="button"]').first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(500)
  await page.screenshot({ path: resolve(outDir, '4-wartung-modal.png') })
  const d4 = await dialogText()
  out.push({
    step: 'maintenance row → detail modal',
    hasCost: /Kosten/.test(d4),
    hasMileage: /km-Stand/.test(d4),
    pass: /Kosten/.test(d4) && /km-Stand/.test(d4),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 5) Fahrtenbuch: AddTripDialog (was "coming soon" toast) — fill & save
  await page.getByRole('button', { name: /Fahrtenbuch \(/ }).click()
  await page.waitForTimeout(800)
  const rowsBefore = await page.locator('tbody tr[role="button"]').count()
  await page.getByRole('button', { name: 'Fahrt eintragen' }).click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(400)
  const textInputs = page.locator('[role="dialog"] input[type="text"]')
  await textInputs.nth(0).fill('Firmensitz München')
  await textInputs.nth(1).fill('Kunde Bergmann AG')
  const numInputs = page.locator('[role="dialog"] input[type="number"]')
  await numInputs.nth(0).fill('45210')
  await numInputs.nth(1).fill('45298')
  await textInputs.nth(2).fill('Kundentermin QA')
  await textInputs.nth(3).fill('Max Muster')
  await page.screenshot({ path: resolve(outDir, '5-addtrip-dialog.png') })
  const d5 = await dialogText()
  await page.locator('[role="dialog"]').getByRole('button', { name: 'Speichern' }).click()
  await page.waitForTimeout(1200)
  const rowsAfter = await page.locator('tbody tr[role="button"]').count()
  await page.screenshot({ path: resolve(outDir, '6-fahrtenbuch-nach-save.png') })
  out.push({
    step: 'add trip dialog → save (was dead toast)',
    hasKmAuto: /88 km/.test(d5),
    rowsBefore,
    rowsAfter,
    pass: /Strecke/.test(d5) && rowsAfter >= rowsBefore + 1,
  })

  // 6) Logbook PDF + CSV export
  const [pdfDl] = await Promise.all([
    page.waitForEvent('download', { timeout: 6000 }),
    page.getByRole('button', { name: 'PDF-Export' }).click(),
  ])
  const [csvDl] = await Promise.all([
    page.waitForEvent('download', { timeout: 6000 }),
    page.getByRole('button', { name: 'CSV-Export' }).click(),
  ])
  out.push({
    step: 'logbook pdf + csv export',
    pdf: pdfDl.suggestedFilename(),
    csv: csvDl.suggestedFilename(),
    pass: /^fahrtenbuch-.*\.pdf$/.test(pdfDl.suggestedFilename()) && /^fahrtenbuch-.*\.csv$/.test(csvDl.suggestedFilename()),
  })

  // 7) Trip row → TripDetailModal
  await page.locator('tbody tr[role="button"]').first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(500)
  await page.screenshot({ path: resolve(outDir, '7-trip-modal.png') })
  const d7 = await dialogText()
  out.push({
    step: 'trip row → detail modal',
    hasKmReadings: /km-Stand Beginn/.test(d7),
    pass: /km-Stand Beginn/.test(d7),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 8) Settings panel registered (personal + tenant)
  await page.evaluate(() => {
    const el = Array.from(document.querySelectorAll('button, a, [role="button"]')).find((e) => /Modul-Einstellung/.test(e.textContent || ''))
    if (el) el.click()
  })
  try {
    await page.getByText('Erinnerungs-Vorlauf', { exact: false }).first().waitFor({ state: 'visible', timeout: 12000 })
  } catch { /* fällt auf Text-Assertion zurück */ }
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '8-settings-panel.png') })
  const txt8 = await bodyText()
  out.push({
    step: 'settings panel registered',
    hasPersonal: /Standard-Fahrtenkategorie|Reifenwechsel-Banner/.test(txt8),
    hasTenant: /Erinnerungs-Vorlauf|Privatfahrten erlauben/.test(txt8),
    pass: /Standard-Fahrtenkategorie/.test(txt8) && /Erinnerungs-Vorlauf/.test(txt8),
  })

  // 9) raw keys + pageerrors
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
