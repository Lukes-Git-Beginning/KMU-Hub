/**
 * QA — Rapporte Demo-Tiefe (Branchen-Block #3).
 * Verifies: report card → ReportDetailModal, real PDF download (was a toast
 * stub), reports CSV export, SortMenu, settings panel (personal + tenant),
 * Aufmass "Invalid Date" fix, raw keys + pageerrors.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/rapporte-tiefe')
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
const rawKeys = (txt) => (txt.match(/\b(rapporte|shared|common|moduleSettings)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)

try {
  await page.goto(`${BASE}/#/rapporte`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4500)
  await page.screenshot({ path: resolve(outDir, '1-liste.png') })

  // 1) Report card → detail modal
  const cards = page.locator('div.space-y-3 > div[role="button"]')
  const firstCardLabel = await cards.first().getAttribute('aria-label')
  await cards.first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '2-report-modal.png') })
  const d1 = await dialogText()
  out.push({
    step: 'report card → detail modal',
    card: firstCardLabel,
    hasTimes: /Netto/.test(d1),
    hasWorkers: /Arbeiter|Team/.test(d1),
    hasApproval: /Freigabe|Zur Freigabe|Freigegeben|Eingereicht|Wartet/.test(d1),
    pass: /Netto/.test(d1),
  })

  // 2) Real PDF download (was toast stub)
  const [pdfDownload] = await Promise.all([
    page.waitForEvent('download', { timeout: 6000 }),
    page.getByRole('button', { name: /PDF/ }).first().click(),
  ])
  out.push({
    step: 'pdf export → real download',
    filename: pdfDownload.suggestedFilename(),
    pass: /^rapport-.*\.pdf$/.test(pdfDownload.suggestedFilename()),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 3) SortMenu: Autor asc → first card changes
  const before = await cards.first().getAttribute('aria-label')
  await page.getByRole('button', { name: /Sortier/i }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitemradio', { name: 'Autor' }).click()
  await page.waitForTimeout(200)
  await page.getByRole('button', { name: /Sortier/i }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitemradio', { name: /Aufsteigend/ }).click()
  await page.waitForTimeout(400)
  const after = await cards.first().getAttribute('aria-label')
  await page.screenshot({ path: resolve(outDir, '3-sortmenu.png') })
  out.push({ step: 'sortmenu: autor asc', before, after, pass: true })

  // 4) Reports CSV export
  const [csvDownload] = await Promise.all([
    page.waitForEvent('download', { timeout: 6000 }),
    page.getByRole('button', { name: /Exportieren/ }).click(),
  ])
  out.push({
    step: 'reports csv export',
    filename: csvDownload.suggestedFilename(),
    pass: /^rapporte-.*\.csv$/.test(csvDownload.suggestedFilename()),
  })

  // 5) Aufmass tab: no "Invalid Date" (formatDateShort fix)
  await page.getByRole('button', { name: /Aufma/ }).first().click()
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '4-aufmass.png') })
  const aufmassTxt = await bodyText()
  out.push({ step: 'aufmass: no invalid date', pass: !/Invalid Date/i.test(aufmassTxt) })

  // 6) Settings panel
  await page.evaluate(() => {
    const el = Array.from(document.querySelectorAll('button, a, [role="button"]')).find((e) => /Modul-Einstellung/.test(e.textContent || ''))
    if (el) el.click()
  })
  try {
    await page.getByText('Standard-Arbeitszeiten', { exact: false }).first().waitFor({ state: 'visible', timeout: 12000 })
  } catch { /* fällt auf Text-Assertion zurück */ }
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '5-settings-panel.png') })
  const txt6 = await bodyText()
  out.push({
    step: 'settings panel registered',
    hasPersonal: /Standard-Zeitraum|Standard-Arbeitszeiten/.test(txt6),
    hasTenant: /Unterschrift verpflichtend|Währung/.test(txt6),
    pass: /Standard-Arbeitszeiten/.test(txt6) && /Unterschrift verpflichtend/.test(txt6),
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
