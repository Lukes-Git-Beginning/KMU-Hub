/**
 * V-1 QA — Vertrag-Detail: Slide-over → zentriertes DetailModal
 *
 * Szenarien:
 * (a) Tabellenzeile klicken → zentriertes Modal (kein Slide-over, kein flush-right)
 * (b) Alle Sektionen im Modal-Body vorhanden (Details, Laufzeit, Wert, Konditionen,
 *     Erinnerungen, Notizen, Dokumente, KI-Fristencheck, Änderungshistorie)
 * (c) Header sticky: nach Scroll bis Bodenende bleibt der Schließen-Button oben
 * (d) Keyboard: Zeile fokussieren + Enter → Modal öffnet
 *
 * Sub-Terminal: läuft gegen :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/vertraege-modal')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const TARGET = 'Büro-Mietvertrag München' // v-1: rich (Dokumente, Reminder, Notizen, Audit)
const SECTIONS = ['Vertragsdetails', 'Laufzeit', 'Wert', 'Konditionen', 'Erinnerungen', 'Notizen', 'Dokumente', 'KI-Fristencheck', 'Änderungshistorie']

// Raw-key detector: any vertraege.*/shared.* key visible as text
const rawRe = /vertraege\.[a-z]|shared\.detailPanel\./i
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src, 'i')
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => (n.textContent || '').trim())
      .slice(0, 10)
  }, rawRe.source)
}

async function openTarget(page) {
  await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2000)
  const row = page.locator('table tbody tr').filter({ hasText: TARGET }).first()
  await row.waitFor({ state: 'visible', timeout: 8000 })
  return row
}

const browser = await chromium.launch()
const out = []

// ─── (a) zentriertes Modal + (b) Sektionen ────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('console: ' + m.text()) })
  try {
    const row = await openTarget(page)
    await row.click()
    await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
    await page.waitForTimeout(700)

    const dialog = page.locator('[role="dialog"]').first()
    const box = await dialog.boundingBox()
    const vw = 1440
    // zentriert: linker Rand deutlich rechts von 0, rechter Rand deutlich links von vw
    // (Slide-over wäre flush-right: x ≈ vw - panelWidth, rechter Rand == vw)
    const centered = !!box && box.x > 120 && box.x + box.width < vw - 120

    const bodyText = await dialog.evaluate((el) => el.textContent || '')
    const sectionsFound = SECTIONS.filter((s) => bodyText.includes(s))
    const sectionsMissing = SECTIONS.filter((s) => !bodyText.includes(s))

    // Header: Titel = Vertragstitel, Subtitle = Typ-Label · Vertragsnr
    const headerOk = bodyText.includes(TARGET) && bodyText.includes('MV-2024-001') && /Mietvertrag/.test(bodyText)

    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'a-modal-open.png'), fullPage: false })
    out.push({
      step: 'a-modal-centered+sections',
      centered,
      box: box ? { x: Math.round(box.x), w: Math.round(box.width), right: Math.round(box.x + box.width) } : null,
      headerOk,
      sectionsFound,
      sectionsMissing,
      pass: centered && headerOk && sectionsMissing.length === 0 && rk.length === 0 && errs.length === 0,
      rawKeys: rk,
      pageErrors: errs.slice(0, 5),
    })
  } catch (e) {
    out.push({ step: 'a-modal-centered+sections', pass: false, error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── (c) Sticky Header beim Scrollen ──────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    const row = await openTarget(page)
    await row.click()
    await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
    await page.waitForTimeout(700)

    // Schließen-Button Position VOR dem Scrollen
    const closeBtn = page.locator('[role="dialog"] button[aria-label="Schließen"]').first()
    const beforeBox = await closeBtn.boundingBox()

    // Das tatsächlich scrollbare Element im Dialog finden und bis unten scrollen
    const scrolledTop = await page.evaluate(() => {
      const dialog = document.querySelector('[role="dialog"]')
      if (!dialog) return 0
      const cands = Array.from(dialog.querySelectorAll('*')).filter(
        (e) => e.scrollHeight > e.clientHeight + 20 && e.clientHeight > 150,
      )
      cands.sort((a, b) => (b.scrollHeight - b.clientHeight) - (a.scrollHeight - a.clientHeight))
      const vp = cands[0]
      if (!vp) return 0
      vp.scrollTo({ top: vp.scrollHeight, behavior: 'instant' })
      return vp.scrollTop
    })
    await page.waitForTimeout(500)

    const afterBox = await closeBtn.boundingBox()
    // Header sticky = Close-Button hat sich NICHT mit weggescrollt (gleiche y-Position, weiterhin oben)
    const stickyOk = !!beforeBox && !!afterBox &&
      Math.abs(beforeBox.y - afterBox.y) < 4 && afterBox.y < 160

    const scrolledToAudit = scrolledTop > 100

    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'c-scrolled-sticky-header.png'), fullPage: false })
    out.push({
      step: 'c-sticky-header',
      beforeY: beforeBox ? Math.round(beforeBox.y) : null,
      afterY: afterBox ? Math.round(afterBox.y) : null,
      scrolledToAudit,
      pass: stickyOk && scrolledToAudit && rk.length === 0,
      rawKeys: rk,
      pageErrors: errs.slice(0, 5),
    })
  } catch (e) {
    out.push({ step: 'c-sticky-header', pass: false, error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── (d) Keyboard: Zeile fokussieren + Enter ──────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    const row = await openTarget(page)
    await row.focus()
    await page.keyboard.press('Enter')
    let opened = false
    try {
      await page.waitForSelector('[role="dialog"]', { timeout: 4000 })
      opened = true
    } catch { opened = false }
    await page.waitForTimeout(500)

    const titleVisible = await page.evaluate((t) =>
      (document.querySelector('[role="dialog"]')?.textContent || '').includes(t), TARGET)

    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'd-keyboard-open.png'), fullPage: false })
    out.push({
      step: 'd-keyboard-enter',
      opened,
      titleVisible,
      pass: opened && titleVisible && rk.length === 0,
      rawKeys: rk,
      pageErrors: errs.slice(0, 5),
    })
  } catch (e) {
    out.push({ step: 'd-keyboard-enter', pass: false, error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

await browser.close()

// ─── Summary ──────────────────────────────────────────────────────
const allPass = out.every((s) => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== V-1 QA RESULT: ${allPass ? 'ALL PASS' : 'FAIL'} ===`)
out.forEach((s) => console.log(`  ${s.pass ? '✓' : '✗'} ${s.step}`))
process.exit(allPass ? 0 : 1)
