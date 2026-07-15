/**
 * QA — Inventar Demo-Tiefe (Branchen-Pilot).
 * Verifies: item row → ItemDetailModal (stock bar, movements, attachments),
 * location card → LocationDetailModal → item back-chain, real "Neue Inventur"
 * dialog + count workflow (Ist input → finish counting), SortMenu, CSV export
 * (real download), and the registered settings panel (personal + tenant).
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/inventar-tiefe')
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
const rawKeys = (txt) => (txt.match(/\b(inventar|shared|common|moduleSettings)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)

try {
  await page.goto(`${BASE}/#/inventar`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4500)
  await page.screenshot({ path: resolve(outDir, '1-artikel-tab.png') })

  // 1) Item row (role=button) → centered detail modal
  const firstRowName = await page.locator('table tbody tr[role="button"] td:nth-child(2)').first().innerText()
  await page.locator('table tbody tr[role="button"]').first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '2-item-modal.png') })
  const d1 = await dialogText()
  out.push({
    step: 'item row → detail modal',
    hasTitle: d1.includes(firstRowName),
    hasStockBar: /Bestand vs Mindestbestand/.test(d1),
    hasMovements: /Letzte Bewegungen/.test(d1),
    hasAttachments: /Anhänge/.test(d1),
    hasFooter: /Bestandsbewegung/.test(d1),
    pass: d1.includes(firstRowName) && /Bestand vs Mindestbestand/.test(d1) && /Anhänge/.test(d1) && /Bestandsbewegung/.test(d1),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 2) SortMenu: sort by Bestand desc → first row changes
  await page.getByRole('button', { name: /Sortier/i }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitemradio', { name: 'Bestand' }).click()
  await page.waitForTimeout(200)
  await page.getByRole('button', { name: /Sortier/i }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitemradio', { name: /Absteigend/ }).click()
  await page.waitForTimeout(400)
  const sortedFirst = await page.locator('table tbody tr[role="button"] td:nth-child(2)').first().innerText()
  await page.screenshot({ path: resolve(outDir, '3-sortmenu-bestand-desc.png') })
  out.push({
    step: 'sortmenu: bestand desc',
    before: firstRowName,
    after: sortedFirst,
    pass: sortedFirst !== firstRowName,
  })

  // 3) CSV export → real download
  const [download] = await Promise.all([
    page.waitForEvent('download', { timeout: 6000 }),
    page.getByRole('button', { name: /Exportieren/ }).click(),
  ])
  out.push({
    step: 'items csv export',
    filename: download.suggestedFilename(),
    pass: /^inventar-artikel-.*\.csv$/.test(download.suggestedFilename()),
  })

  // 4) Location card → location modal → item click → item modal with back
  await page.getByRole('button', { name: /^Lagerorte/ }).click()
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '4-lagerorte-tab.png') })
  const cardName = await page.locator('[role="button"][aria-label]').first().getAttribute('aria-label')
  await page.locator(`[role="button"][aria-label="${cardName}"]`).first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(500)
  await page.screenshot({ path: resolve(outDir, '5-location-modal.png') })
  const d2 = await dialogText()
  const locPass = /Artikel an diesem Lagerort/.test(d2)
  out.push({ step: 'location card → modal', location: cardName, hasItemsList: locPass, pass: locPass })

  // click first item inside the location modal → item modal with back arrow
  const modalItem = page.locator('[role="dialog"] [role="button"]').first()
  const hasModalItem = (await modalItem.count()) > 0
  if (hasModalItem) {
    await modalItem.click()
    await page.waitForTimeout(600)
    await page.screenshot({ path: resolve(outDir, '6-item-from-location.png') })
    const d3 = await dialogText()
    const backBtn = page.locator('[role="dialog"] button[aria-label]').first()
    const backLabel = await backBtn.getAttribute('aria-label')
    out.push({
      step: 'location → item back-chain',
      hasStockBar: /Bestand vs Mindestbestand/.test(d3),
      backLabel,
      pass: /Bestand vs Mindestbestand/.test(d3) && !!backLabel,
    })
    await backBtn.click()
    await page.waitForTimeout(500)
    const d4 = await dialogText()
    out.push({ step: 'back → location modal again', pass: /Artikel an diesem Lagerort/.test(d4) })
  }
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 5) Neue Inventur: real dialog + create + count workflow
  await page.getByRole('button', { name: /^Inventur/ }).click()
  await page.waitForTimeout(500)
  await page.getByRole('button', { name: /Neue Inventur starten/ }).click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(400)
  await page.screenshot({ path: resolve(outDir, '7-inventur-dialog.png') })
  const d5 = await dialogText()
  out.push({
    step: 'neue inventur dialog (no toast stub)',
    hasFields: /Stichtag/.test(d5) && /Lagerort/.test(d5) && /Zählliste/.test(d5),
    pass: /Stichtag/.test(d5) && /Zählliste/.test(d5),
  })
  await page.locator('[role="dialog"] input[type="text"]').first().fill('QA Testinventur')
  await page.getByRole('button', { name: /Inventur anlegen/ }).click()
  await page.waitForTimeout(800)
  const created = /QA Testinventur/.test(await bodyText())
  out.push({ step: 'session created + in list', pass: created })

  // expand new session → Ist input → commit → finish counting
  if (created) {
    await page.getByRole('button', { name: /QA Testinventur/ }).click()
    await page.waitForTimeout(500)
    await page.screenshot({ path: resolve(outDir, '8-inventur-counting.png') })
    const istInput = page.locator('input[placeholder="Ist"]').first()
    const hasIst = (await istInput.count()) > 0
    if (hasIst) {
      await istInput.fill('7')
      await page.keyboard.press('Enter')
      await page.waitForTimeout(800)
      const finishBtn = page.getByRole('button', { name: /Zählung abschließen/ })
      const finishEnabled = (await finishBtn.count()) > 0 && (await finishBtn.first().isEnabled())
      if (finishEnabled) {
        await finishBtn.first().click()
        await page.waitForTimeout(800)
      }
      await page.screenshot({ path: resolve(outDir, '9-inventur-review.png') })
      const txt = await bodyText()
      out.push({
        step: 'count + finish → review status',
        hasIstInput: hasIst,
        finishEnabled,
        hasReviewBadge: /Prüfung/.test(txt),
        hasBookButton: /Differenzen buchen/.test(txt),
        pass: hasIst && finishEnabled && /Differenzen buchen/.test(txt),
      })
    } else {
      out.push({ step: 'count + finish → review status', hasIstInput: false, pass: false })
    }
  }

  // 6) Settings overlay → inventar panel (personal + tenant)
  await page.evaluate(() => {
    const el = Array.from(document.querySelectorAll('button, a, [role="button"]')).find((e) => /Modul-Einstellung/.test(e.textContent || ''))
    if (el) el.click()
  })
  try {
    await page.getByText('Team-Vorgaben', { exact: false }).first().waitFor({ state: 'visible', timeout: 12000 })
  } catch { /* fällt auf Text-Assertion zurück */ }
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '10-settings-panel.png') })
  const txt6 = await bodyText()
  out.push({
    step: 'settings panel registered',
    hasPersonal: /Standard-Tab|Tabellendichte/.test(txt6),
    hasTenant: /Standard-Einheit|Barcode-Format|Negativbestände/.test(txt6),
    pass: /Standard-Tab|Tabellendichte/.test(txt6) && /Standard-Einheit|Barcode-Format/.test(txt6),
  })

  // 7) raw keys + pageerrors
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
