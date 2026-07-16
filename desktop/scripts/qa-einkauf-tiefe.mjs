/**
 * QA — Einkauf Demo-Tiefe (Branchen-Block #6).
 * Verifies: order row → DetailModal incl. positions (list query carried no
 * lines — was always empty!), PO PDF export, supplier back chain, approval
 * banner → real submit, orders CSV, SortMenu, supplier rating form, goods
 * receipt with visible articles, catalog cart → bundled orders per supplier
 * (recomputed totals ≠ 0), framework-contract call-off (+used_value),
 * settings panel, raw keys + pageerrors.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/einkauf-tiefe')
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
const rawKeys = (txt) => (txt.match(/\b(einkauf|shared|common|moduleSettings)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)

try {
  await page.goto(`${BASE}/#/einkauf`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4500)
  await page.screenshot({ path: resolve(outDir, '1-bestellungen.png') })

  // 1) Order row → DetailModal with real positions (list query has no lines!)
  await page.locator('tbody tr[role="button"]').first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(900)
  await page.screenshot({ path: resolve(outDir, '2-order-modal.png') })
  const d1 = await dialogText()
  out.push({
    step: 'order row → detail modal with positions',
    hasTimeline: /Entwurf/.test(d1) && /Geliefert/.test(d1),
    hasPositions: !/Keine Positionen/.test(d1) && /Stück|Menge|St\.|x /.test(d1 + ' '),
    pass: /Bestellte Positionen|Positionen/.test(d1) && !/Keine Positionen/.test(d1),
  })

  // 2) PO PDF export from the modal
  const [pdfDl] = await Promise.all([
    page.waitForEvent('download', { timeout: 6000 }),
    page.locator('[role="dialog"]').getByRole('button', { name: /Bestell-PDF/ }).click(),
  ])
  out.push({
    step: 'po pdf export (was dead exportPO endpoint)',
    filename: pdfDl.suggestedFilename(),
    pass: /^PO-.*\.pdf$/.test(pdfDl.suggestedFilename()),
  })

  // 3) Supplier link → supplier modal → back chain returns to order modal
  await page.locator('[role="dialog"]').getByRole('heading', { level: 4 }).filter({ hasText: 'Lieferant' }).first().waitFor({ timeout: 4000 }).catch(() => {})
  const orderTitle = await page.locator('[role="dialog"] h3').first().innerText()
  await page.locator('[role="dialog"] button').filter({ hasText: /GmbH|AG|Elektronik/ }).first().click()
  await page.waitForTimeout(900)
  const dSup = await dialogText()
  const hasBackBtn = await page.locator('[role="dialog"] button[aria-label="Zurück"]').count()
  await page.screenshot({ path: resolve(outDir, '3-supplier-via-backchain.png') })
  if (hasBackBtn > 0) await page.locator('[role="dialog"] button[aria-label="Zurück"]').click()
  await page.waitForTimeout(700)
  const backTitle = await page.locator('[role="dialog"] h3').first().innerText().catch(() => '')
  out.push({
    step: 'order → supplier modal → back chain',
    supplierModal: /Bewertungen|Bestellungen \(/.test(dSup),
    backBtn: hasBackBtn > 0,
    backReturnsToOrder: backTitle === orderTitle,
    pass: hasBackBtn > 0 && backTitle === orderTitle,
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 4) Approval banner on expensive draft → real submit (was toast-only)
  await page.getByRole('button', { name: 'Entwurf', exact: true }).click()
  await page.waitForTimeout(600)
  // sort by amount desc so the >threshold draft is first
  await page.getByRole('button', { name: /Sortier/i }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitemradio', { name: 'Betrag' }).click()
  await page.waitForTimeout(200)
  await page.getByRole('button', { name: /Sortier/i }).first().click()
  await page.waitForTimeout(300)
  await page.getByRole('menuitemradio', { name: /Absteigend/ }).click()
  await page.waitForTimeout(400)
  await page.locator('tbody tr[role="button"]').first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(700)
  const d4 = await dialogText()
  await page.screenshot({ path: resolve(outDir, '4-draft-approval-banner.png') })
  const hadBanner = /Genehmigung erforderlich/.test(d4)
  await page.locator('[role="dialog"]').getByRole('button', { name: 'Genehmigen' }).click()
  await page.waitForTimeout(1200)
  const bodyAfterSubmit = await bodyText()
  await page.screenshot({ path: resolve(outDir, '5-nach-freigabe-submit.png') })
  out.push({
    step: 'approval banner → real submitPO (was dead toast)',
    hadBanner,
    submitted: /genehmigt|Eingereicht/i.test(bodyAfterSubmit),
    pass: hadBanner && /genehmigt|Eingereicht/i.test(bodyAfterSubmit),
  })

  // 5) Orders CSV export
  await page.getByRole('button', { name: 'Alle', exact: true }).click()
  await page.waitForTimeout(500)
  const [csvDl] = await Promise.all([
    page.waitForEvent('download', { timeout: 6000 }),
    page.getByRole('button', { name: /CSV exportieren/ }).click(),
  ])
  out.push({
    step: 'orders csv export',
    filename: csvDl.suggestedFilename(),
    pass: /^bestellungen-.*\.csv$/.test(csvDl.suggestedFilename()),
  })

  // 6) Goods receipt dialog shows real articles (was always empty)
  await page.getByRole('button', { name: 'Bestellt', exact: true }).click()
  await page.waitForTimeout(600)
  await page.locator('tbody tr[role="button"]').first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(700)
  await page.locator('[role="dialog"]').getByRole('button', { name: /Wareneingang/ }).click()
  await page.waitForTimeout(900)
  const d6 = await dialogText()
  await page.screenshot({ path: resolve(outDir, '6-wareneingang-artikel.png') })
  const weInput = page.locator('[role="dialog"] input[type="number"]').first()
  await weInput.fill('5')
  await page.locator('[role="dialog"]').getByRole('button', { name: /buchen/i }).click()
  await page.waitForTimeout(1200)
  const bodyAfterWE = await bodyText()
  out.push({
    step: 'goods receipt: articles visible + partial booking',
    hasArticles: !/Keine Positionen/.test(d6) && /Bereits erhalten|erhalten/i.test(d6),
    booked: /Wareneingang.*gebucht|gebucht/i.test(bodyAfterWE),
    pass: !/Keine Positionen/.test(d6) && /gebucht/i.test(bodyAfterWE),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 7) Supplier tab: card → modal → inline rating form (unused hook wired)
  await page.getByRole('button', { name: /Lieferanten \(/ }).click()
  await page.waitForTimeout(800)
  await page.locator('div[role="button"]').filter({ hasText: /GmbH|AG|Elektronik/ }).first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(700)
  await page.locator('[role="dialog"]').getByText('Bewertung abgeben').click()
  await page.waitForTimeout(400)
  await page.locator('[role="dialog"] input[type="text"]').last().fill('QA-Testbewertung')
  await page.screenshot({ path: resolve(outDir, '7-supplier-rating-form.png') })
  await page.locator('[role="dialog"]').getByRole('button', { name: 'Bewertung speichern' }).click()
  await page.waitForTimeout(1200)
  const bodyAfterRating = await bodyText()
  out.push({
    step: 'supplier rating form → save (unused hook wired)',
    pass: /Bewertung für .* gespeichert/.test(bodyAfterRating),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 8) Catalog cart → bundled orders per supplier, totals recomputed
  await page.getByRole('button', { name: /Katalog \(/ }).click()
  await page.waitForTimeout(900)
  const cartBtns = page.getByRole('button', { name: 'In den Warenkorb' })
  await cartBtns.nth(0).click()
  await page.waitForTimeout(300)
  await cartBtns.nth(2).click()
  await page.waitForTimeout(400)
  await page.getByRole('button', { name: /Warenkorb/ }).first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(600)
  const d8 = await dialogText()
  await page.screenshot({ path: resolve(outDir, '8-cart-dialog.png') })
  await page.locator('[role="dialog"]').getByRole('button', { name: /Bestellungen erstellen|Bestellung erstellen/ }).click()
  await page.waitForTimeout(2500)
  const bodyAfterCart = await bodyText()
  await page.screenshot({ path: resolve(outDir, '9-bestellungen-nach-cart.png') })
  const newPoVisible = bodyAfterCart.includes(`PO-${YEAR}-011`)
  out.push({
    step: 'catalog cart → orders per supplier (was dead toast)',
    grouped: /GmbH/.test(d8),
    newPoVisible,
    noZeroTotal: !/PO-\d+-01[12]\s[\s\S]{0,80}?\s0,00/.test(bodyAfterCart),
    pass: newPoVisible && !/PO-\d+-01[12]\s[\s\S]{0,80}?\s0,00/.test(bodyAfterCart),
  })

  // 9) Framework contract: calls list + new call-off raises used_value
  await page.getByRole('button', { name: /Rahmenverträge \(/ }).click()
  await page.waitForTimeout(800)
  await page.getByRole('button', { name: /Jahresvertrag Elektromaterial/ }).click()
  await page.waitForTimeout(700)
  const bodyContract = await bodyText()
  const hadCallsList = /Bisherige Abrufe \(2\)/i.test(bodyContract)
  await page.screenshot({ path: resolve(outDir, '10-contract-expanded.png') })
  await page.getByRole('button', { name: /Neuen Abruf erstellen/ }).click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(400)
  await page.locator('[role="dialog"] input[type="number"]').fill('500')
  await page.locator('[role="dialog"] textarea').fill('QA-Abruf')
  await page.screenshot({ path: resolve(outDir, '11-call-dialog.png') })
  await page.locator('[role="dialog"]').getByRole('button', { name: 'Abruf anlegen' }).click()
  await page.waitForTimeout(1500)
  const bodyAfterCall = await bodyText()
  out.push({
    step: 'contract call-off (unused hook wired) + used_value',
    hadCallsList,
    callsNow3: /Bisherige Abrufe \(3\)/i.test(bodyAfterCall),
    usedRaised: /13\.000/.test(bodyAfterCall),
    pass: hadCallsList && /Bisherige Abrufe \(3\)/i.test(bodyAfterCall) && /13\.000/.test(bodyAfterCall),
  })

  // 10) Settings panel registered (personal + tenant)
  await page.evaluate(() => {
    const el = Array.from(document.querySelectorAll('button, a, [role="button"]')).find((e) => /Modul-Einstellung/.test(e.textContent || ''))
    if (el) el.click()
  })
  try {
    await page.getByText('Freigabegrenze', { exact: false }).first().waitFor({ state: 'visible', timeout: 12000 })
  } catch { /* fällt auf Text-Assertion zurück */ }
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '12-settings-panel.png') })
  const txt10 = await bodyText()
  out.push({
    step: 'settings panel registered',
    hasPersonal: /Standard-Tab|Standard-Statusfilter/.test(txt10),
    hasTenant: /Freigabegrenze|Bestellnummern-Präfix/.test(txt10),
    pass: /Standard-Statusfilter/.test(txt10) && /Freigabegrenze/.test(txt10),
  })

  // 11) raw keys + pageerrors
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
