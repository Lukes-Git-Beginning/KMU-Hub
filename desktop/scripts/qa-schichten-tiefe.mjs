/**
 * QA — Schichten Demo-Tiefe (Branchen-Block #4).
 * Verifies: occupied grid cell → ShiftDetailModal (was an info toast), swap
 * form inside the modal, real PDF + CSV downloads (PDF was a toast stub),
 * SortMenu, template edit dialog (was a dead button), apply-template dialog,
 * settings panel (personal + tenant), raw keys + pageerrors.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/schichten-tiefe')
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
const rawKeys = (txt) => (txt.match(/\b(schichten|shared|common|moduleSettings)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)

try {
  await page.goto(`${BASE}/#/schichten`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4500)
  await page.screenshot({ path: resolve(outDir, '1-wochenplan.png') })

  // 1) Occupied cell → ShiftDetailModal (was toast.info)
  const occupied = page.locator('div[role="button"]').filter({ hasText: /Frühschicht|Spätschicht|Nachtschicht/ })
  const occupiedCount = await occupied.count()
  await occupied.first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '2-shift-detail-modal.png') })
  const d1 = await dialogText()
  out.push({
    step: 'occupied cell → detail modal',
    occupiedCells: occupiedCount,
    hasEmployee: /Mitarbeiter/.test(d1),
    hasTime: /Schichtzeit/.test(d1),
    hasStatus: /Veröffentlicht|Entwurf/.test(d1),
    hasActions: /Zuweisung entfernen/.test(d1),
    pass: /Mitarbeiter/.test(d1) && /Schichtzeit/.test(d1) && /Zuweisung entfernen/.test(d1),
  })

  // 2) Swap form inside the modal (wires previously unused createSwapRequest)
  await page.getByRole('button', { name: 'Tausch beantragen' }).click()
  await page.waitForTimeout(400)
  await page.screenshot({ path: resolve(outDir, '3-swap-form.png') })
  const d2 = await dialogText()
  out.push({
    step: 'swap form in modal',
    hasSelect: /Tauschen mit/.test(d2),
    hasReason: /Begründung/.test(d2),
    pass: /Tauschen mit/.test(d2) && /Begründung/.test(d2),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 3) Real PDF download (was toast stub)
  const [pdfDownload] = await Promise.all([
    page.waitForEvent('download', { timeout: 6000 }),
    page.getByRole('button', { name: 'PDF-Export' }).click(),
  ])
  out.push({
    step: 'pdf export → real download',
    filename: pdfDownload.suggestedFilename(),
    pass: /^dienstplan-kw\d+-.*\.pdf$/.test(pdfDownload.suggestedFilename()),
  })

  // 4) CSV export (new)
  const [csvDownload] = await Promise.all([
    page.waitForEvent('download', { timeout: 6000 }),
    page.getByRole('button', { name: 'CSV-Export' }).click(),
  ])
  out.push({
    step: 'csv export → real download',
    filename: csvDownload.suggestedFilename(),
    pass: /^schichten-kw\d+-.*\.csv$/.test(csvDownload.suggestedFilename()),
  })

  // 5) SortMenu: sort by weekly hours desc — grid re-renders without errors
  await page.getByRole('button', { name: /Sortier/i }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitemradio', { name: 'Wochenstunden' }).click()
  await page.waitForTimeout(200)
  await page.getByRole('button', { name: /Sortier/i }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitemradio', { name: /Absteigend/ }).click()
  await page.waitForTimeout(400)
  await page.screenshot({ path: resolve(outDir, '4-sortmenu.png') })
  out.push({ step: 'sortmenu: hours desc', pass: true })

  // 6) Template edit dialog (was a dead toast button)
  await page.getByRole('button', { name: /Vorlagen/ }).first().click()
  await page.waitForTimeout(800)
  const tplCard = page.locator('div.rounded-lg.border').filter({ hasText: 'Frühschicht' }).first()
  await tplCard.locator('button').last().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitem', { name: 'Bearbeiten' }).click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(400)
  await page.screenshot({ path: resolve(outDir, '5-template-edit.png') })
  const tplName = await page.locator('[role="dialog"] input[type="text"]').inputValue()
  const d6 = await dialogText()
  out.push({
    step: 'template edit dialog (was dead button)',
    prefilledName: tplName,
    hasEditTitle: /Vorlage bearbeiten/.test(d6),
    pass: /Vorlage bearbeiten/.test(d6) && tplName.length > 0,
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 7) Apply-template dialog (wires previously unused applyTemplate endpoint)
  await tplCard.locator('button').last().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitem', { name: 'Auf Woche anwenden' }).click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(400)
  await page.screenshot({ path: resolve(outDir, '6-apply-template.png') })
  const d7 = await dialogText()
  out.push({
    step: 'apply template dialog',
    hasWeeks: /Aktuelle Woche/.test(d7) && /Nächste Woche/.test(d7),
    pass: /Aktuelle Woche/.test(d7),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 8) Settings panel registered (personal + tenant)
  await page.evaluate(() => {
    const el = Array.from(document.querySelectorAll('button, a, [role="button"]')).find((e) => /Modul-Einstellung/.test(e.textContent || ''))
    if (el) el.click()
  })
  try {
    await page.getByText('Standard-Ansicht', { exact: false }).first().waitFor({ state: 'visible', timeout: 12000 })
  } catch { /* fällt auf Text-Assertion zurück */ }
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '7-settings-panel.png') })
  const txt8 = await bodyText()
  out.push({
    step: 'settings panel registered',
    hasPersonal: /Standard-Ansicht|Zuschlags-Badges/.test(txt8),
    hasTenant: /Schichttausch erlauben|Max\. Wochenstunden/.test(txt8),
    pass: /Standard-Ansicht/.test(txt8) && /Schichttausch erlauben/.test(txt8),
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
