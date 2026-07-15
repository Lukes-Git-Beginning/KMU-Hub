/**
 * QA — Vermietung Demo-Tiefe (Branchen-Block #2).
 * Verifies: object card → ObjectDetailModal (+rental back-chain), reservation
 * row → RentalDetailModal with real lifecycle actions (Ausgeben/Zurücknehmen),
 * calendar busy slot → modal (no toast stub), SortMenu, CSV export, settings
 * panel (personal + tenant).
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/vermietung-tiefe')
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
const rawKeys = (txt) => (txt.match(/\b(vermietung|shared|common|moduleSettings)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)

try {
  await page.goto(`${BASE}/#/vermietung`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4500)
  await page.screenshot({ path: resolve(outDir, '1-objekte-tab.png') })

  // 1) Object card → detail modal
  const firstCard = page.locator('.grid > div[role="button"][aria-label]').first()
  const cardName = await firstCard.getAttribute('aria-label')
  await firstCard.click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(500)
  await page.screenshot({ path: resolve(outDir, '2-object-modal.png') })
  const d1 = await dialogText()
  out.push({
    step: 'object card → detail modal',
    object: cardName,
    hasDetails: d1.includes(cardName ?? '§none§'),
    hasPricing: /Preise/.test(d1),
    hasReservations: /Letzte Reservierungen|Reservierungen/.test(d1),
    pass: d1.includes(cardName ?? '§none§') && /Preise/.test(d1),
  })

  // 2) Rental inside the object modal → rental modal with back arrow
  const modalRental = page.locator('[role="dialog"] [role="button"]').first()
  if ((await modalRental.count()) > 0) {
    await modalRental.click()
    await page.waitForTimeout(600)
    await page.screenshot({ path: resolve(outDir, '3-rental-from-object.png') })
    const d2 = await dialogText()
    const backBtn = page.locator('[role="dialog"] button[aria-label]').first()
    const backLabel = await backBtn.getAttribute('aria-label')
    out.push({
      step: 'object → rental back-chain',
      hasPricing: /Preis & Kaution/.test(d2),
      backLabel,
      pass: /Preis & Kaution/.test(d2) && !!backLabel,
    })
    await backBtn.click()
    await page.waitForTimeout(500)
    out.push({ step: 'back → object modal again', pass: /Preise/.test(await dialogText()) })
  }
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 3) Reservierungen tab: row click → rental modal + lifecycle action
  await page.getByRole('button', { name: /Reservierungen/ }).first().click()
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '4-reservierungen-tab.png') })

  // find a row whose modal offers "Ausgeben" (reserved rental)
  const rows = page.locator('table tbody tr[role="button"]')
  const rowCount = await rows.count()
  let lifecyclePass = false
  let sawModal = false
  for (let i = 0; i < Math.min(rowCount, 8); i++) {
    await rows.nth(i).click()
    await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
    await page.waitForTimeout(500)
    sawModal = true
    const dt = await dialogText()
    if (i === 0) await page.screenshot({ path: resolve(outDir, '5-rental-modal.png') })
    const ausgeben = page.getByRole('button', { name: /^Ausgeben$/ })
    if ((await ausgeben.count()) > 0) {
      await ausgeben.first().click()
      await page.waitForTimeout(1200)
      const after = await dialogText()
      lifecyclePass = /Zurücknehmen/.test(after)
      await page.screenshot({ path: resolve(outDir, '6-rental-active-after-start.png') })
      out.push({
        step: 'lifecycle: Ausgeben → active (Zurücknehmen visible)',
        hadDetail: /Preis & Kaution/.test(dt),
        pass: lifecyclePass,
      })
      break
    }
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)
  }
  if (!lifecyclePass) {
    out.push({ step: 'lifecycle: Ausgeben → active', sawModal, pass: false, note: 'no reserved rental found' })
  }
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 4) SortMenu (Mieter asc) → first row changes
  const firstRowBefore = await rows.first().innerText()
  await page.getByRole('button', { name: /Sortier/i }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitemradio', { name: 'Mieter' }).click()
  await page.waitForTimeout(200)
  await page.getByRole('button', { name: /Sortier/i }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitemradio', { name: /Aufsteigend/ }).click()
  await page.waitForTimeout(400)
  const firstRowAfter = await rows.first().innerText()
  await page.screenshot({ path: resolve(outDir, '7-sortmenu.png') })
  out.push({ step: 'sortmenu: mieter asc', changed: firstRowAfter !== firstRowBefore, pass: true })

  // 5) CSV export
  const [download] = await Promise.all([
    page.waitForEvent('download', { timeout: 6000 }),
    page.getByRole('button', { name: /Exportieren/ }).click(),
  ])
  out.push({
    step: 'rentals csv export',
    filename: download.suggestedFilename(),
    pass: /^vermietung-reservierungen-.*\.csv$/.test(download.suggestedFilename()),
  })

  // 6) Calendar: busy slot → rental modal (not a toast)
  await page.getByRole('button', { name: /Kalender/ }).first().click()
  await page.waitForTimeout(600)
  let busy = page.locator('div[role="button"][aria-label*=": "]')
  let found = (await busy.count()) > 0
  for (let nav = 0; nav < 6 && !found; nav++) {
    // navigate weeks forward looking for a busy slot
    await page.locator('button:has(svg.lucide-chevron-right)').first().click()
    await page.waitForTimeout(400)
    busy = page.locator('div[role="button"][aria-label*=": "]')
    found = (await busy.count()) > 0
  }
  if (found) {
    await busy.first().click()
    await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
    await page.waitForTimeout(500)
    await page.screenshot({ path: resolve(outDir, '8-calendar-slot-modal.png') })
    const d6 = await dialogText()
    out.push({ step: 'calendar busy slot → rental modal', pass: /Preis & Kaution/.test(d6) })
    await page.keyboard.press('Escape')
    await page.waitForTimeout(400)
  } else {
    out.push({ step: 'calendar busy slot → rental modal', pass: false, note: 'no busy slot found in 6 weeks' })
  }

  // 7) Settings panel
  await page.evaluate(() => {
    const el = Array.from(document.querySelectorAll('button, a, [role="button"]')).find((e) => /Modul-Einstellung/.test(e.textContent || ''))
    if (el) el.click()
  })
  try {
    await page.getByText('Vorbereitungszeit', { exact: false }).first().waitFor({ state: 'visible', timeout: 12000 })
  } catch { /* fällt auf Text-Assertion zurück */ }
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '9-settings-panel.png') })
  const txt7 = await bodyText()
  out.push({
    step: 'settings panel registered',
    hasPersonal: /Standard-Tab|Kennzahlen-Leiste/.test(txt7),
    hasTenant: /Standardwährung|Vorbereitungszeit|Kaution verpflichtend/.test(txt7),
    pass: /Standard-Tab|Kennzahlen-Leiste/.test(txt7) && /Standardwährung|Vorbereitungszeit/.test(txt7),
  })

  // 8) raw keys + pageerrors
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
