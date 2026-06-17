/**
 * V-2 QA — Dokument-Vorschau im Vertrag-Detail rendert echtes PDF (kein 404)
 *
 * Für mehrere Seed-Verträge: Detail öffnen → Dokument klicken → FilePreviewModal.
 * Prüft: /download liefert 200 (kein 404), iframe-src ist blob:-URL, PDF-Viewer da.
 *
 * HEADED (headless Chromium hat keinen PDF-Viewer → iframe bliebe leer).
 * Sub-Terminal: :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/vertraege-preview')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

// Aktive Verträge mit Dokumenten (im „Aktiv"-Tab erreichbar) + erster Dateiname
const CASES = [
  { contract: 'Büro-Mietvertrag München', file: 'Vertrag_Gruber_Maschinenbau' },
  { contract: 'Telekom Business Internet', file: 'SLA_Helvetia_Software' },
  { contract: 'Thomas Berger Arbeitsvertrag', file: 'Arbeitsvertrag_Muster' },
  { contract: 'Allianz Betriebsversicherung', file: 'Datenschutzerklaerung' },
]

const browser = await chromium.launch({ headless: false })
const out = []

for (const c of CASES) {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  const downloadResponses = []
  page.on('pageerror', (e) => errs.push(String(e)))
  page.on('response', (r) => {
    if (/\/documents\/files\/.+\/download/.test(r.url())) {
      downloadResponses.push({ url: r.url(), status: r.status() })
    }
  })
  try {
    await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2000)

    // Detail öffnen
    const row = page.locator('table tbody tr').filter({ hasText: c.contract }).first()
    await row.waitFor({ state: 'visible', timeout: 8000 })
    await row.click()
    await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
    await page.waitForTimeout(600)

    // Dokumente-Sektion scrollen + Dokument-Button klicken
    await page.evaluate(() => {
      const dialog = document.querySelector('[role="dialog"]')
      const cands = Array.from(dialog?.querySelectorAll('*') || []).filter(
        (e) => e.scrollHeight > e.clientHeight + 20 && e.clientHeight > 150,
      )
      cands.sort((a, b) => (b.scrollHeight - b.clientHeight) - (a.scrollHeight - a.clientHeight))
      cands[0]?.scrollTo({ top: cands[0].scrollHeight, behavior: 'instant' })
    })
    await page.waitForTimeout(400)

    const docBtn = page.locator('button').filter({ hasText: c.file }).first()
    await docBtn.waitFor({ state: 'visible', timeout: 5000 })
    await docBtn.click()

    // Auf FilePreviewModal warten (Dateiname als Titel)
    await page.waitForFunction(
      (name) => Array.from(document.querySelectorAll('h2, [role="heading"]'))
        .some((el) => (el.textContent || '').includes(name)),
      c.file,
      { timeout: 8000 },
    )
    await page.waitForTimeout(1500) // PDF-Viewer Zeit zum Rendern geben

    // iframe-src prüfen (blob:-URL = Vorschau geladen)
    const iframeSrc = await page.evaluate(() => {
      const f = document.querySelector('[role="dialog"] iframe')
      return f ? f.getAttribute('src') : null
    })
    const isBlob = !!iframeSrc && iframeSrc.startsWith('blob:')
    const status = downloadResponses.length ? downloadResponses[downloadResponses.length - 1].status : null
    const no404 = downloadResponses.every((r) => r.status === 200)

    await page.screenshot({ path: resolve(outDir, `${c.file}.png`), fullPage: false })
    out.push({
      contract: c.contract,
      file: c.file,
      downloadStatus: status,
      no404,
      iframeBlob: isBlob,
      iframeSrcPrefix: iframeSrc ? iframeSrc.slice(0, 12) : null,
      pass: no404 && isBlob && errs.length === 0,
      pageErrors: errs.slice(0, 3),
    })
  } catch (e) {
    out.push({ contract: c.contract, file: c.file, pass: false, error: String(e).split('\n')[0], downloadResponses })
  } finally {
    await ctx.close()
  }
}

await browser.close()

const allPass = out.every((s) => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== V-2 QA RESULT: ${allPass ? 'ALL PASS' : 'FAIL'} ===`)
out.forEach((s) => console.log(`  ${s.pass ? '✓' : '✗'} ${s.contract} → ${s.file} (status ${s.downloadStatus ?? '?'})`))
process.exit(allPass ? 0 : 1)
