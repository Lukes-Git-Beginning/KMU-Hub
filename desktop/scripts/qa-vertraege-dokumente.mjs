/**
 * Phase 7 QA — Vertragsdokumente ans Dokumente-Modul gekoppelt
 *
 * Szenarien:
 * (a) DetailPanel eines Vertrags MIT Dokumenten — Liste sichtbar
 * (b) Klick auf Dokument → FilePreviewModal öffnet (Dialog-Titel mit Dateiname)
 * (c) Versions-Button → VersionHistoryPanel offen (Versionseinträge sichtbar)
 * (d) ContractDialog: Picker öffnen, Dokument auswählen, erscheint in Liste
 * (e) Leerer Zustand (Vertrag ohne Dokumente)
 *
 * Jedes Szenario gibt pass: true nur wenn die Kern-Interaktion stattgefunden hat.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/vertraege-dokumente')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

// Raw-key detector: any vertraege.documents.* key visible as text
const rawRe = /vertraege\.(documents\.|history\.action\.(document_))/
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 10)
  }, rawRe.source)
}

/** Scroll the detail panel's scrollable container to the bottom */
async function scrollDetailPanelToBottom(page) {
  await page.evaluate(() => {
    // DetailPanel uses a fixed slide-over — find the scrollable inner container
    const candidates = Array.from(document.querySelectorAll('*')).filter((e) => {
      const style = window.getComputedStyle(e)
      return (
        (style.overflowY === 'auto' || style.overflowY === 'scroll') &&
        e.scrollHeight > e.clientHeight &&
        e.clientHeight > 150
      )
    })
    // Pick the last one (deepest scrollable pane)
    const panel = candidates[candidates.length - 1]
    if (panel) panel.scrollTo({ top: panel.scrollHeight, behavior: 'instant' })
  })
  await page.waitForTimeout(400)
}

const browser = await chromium.launch()
const out = []

// ─── (a) DetailPanel mit Dokumenten ────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2000)

    // Click the row containing "Büro-Mietvertrag München" — has file-005 + file-007
    const targetRow = page.locator('table tbody tr').filter({ hasText: 'Büro-Mietvertrag München' }).first()
    await targetRow.waitFor({ state: 'visible', timeout: 8000 })
    await targetRow.click()
    await page.waitForTimeout(1200)

    // Scroll detail panel to reveal the Dokumente section
    await scrollDetailPanelToBottom(page)

    // Count document entries in the detail panel (file name buttons)
    const fileCount = await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll('button'))
      return btns.filter(b =>
        /Vertrag_Gruber_Maschinenbau|NDA_Rhein_Consulting/.test(b.textContent || '')
      ).length
    })

    const hasDokumenteSection = await page.evaluate(() =>
      /DOKUMENTE/i.test(document.body.textContent || '') ||
      /Dokumente/.test(document.body.textContent || ''))
    const hasDocumentFile = fileCount >= 1

    const rk = await rawKeys(page)

    await page.screenshot({ path: resolve(outDir, 'a-detail-with-documents.png'), fullPage: false })
    out.push({
      step: 'a-detail-with-documents',
      hasDokumenteSection,
      hasDocumentFile,
      fileCount,
      pass: hasDokumenteSection && hasDocumentFile,
      rawKeys: rk,
      pageErrors: errs,
    })
  } catch (e) {
    out.push({ step: 'a-detail-with-documents', pass: false, error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── (b) Klick auf Dokument → FilePreviewModal ──────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2000)

    // Open the row with documents
    const targetRow = page.locator('table tbody tr').filter({ hasText: 'Büro-Mietvertrag München' }).first()
    await targetRow.waitFor({ state: 'visible', timeout: 8000 })
    await targetRow.click()
    await page.waitForTimeout(1200)

    // Scroll to Dokumente section
    await scrollDetailPanelToBottom(page)

    // Click the file name button (Vertrag_Gruber_Maschinenbau)
    const fileBtn = page.locator('button').filter({ hasText: 'Vertrag_Gruber_Maschinenbau' }).first()
    await fileBtn.waitFor({ state: 'visible', timeout: 5000 })
    const docClicked = await fileBtn.isVisible()
    await fileBtn.click()

    // Wait for the FilePreviewModal to appear.
    // The modal has class max-w-4xl (FilePreviewModal DialogContent).
    // We wait for an h2 anywhere in the page that contains the filename.
    await page.waitForFunction(() => {
      const h2s = Array.from(document.querySelectorAll('h2, [data-radix-dialog-title]'))
      return h2s.some(el => /Vertrag_Gruber_Maschinenbau/.test(el.textContent || ''))
    }, { timeout: 6000 })
    await page.waitForTimeout(500)

    // Verify the filename h2 exists
    const modalTitle = await page.evaluate(() => {
      const h2s = Array.from(document.querySelectorAll('h2'))
      const el = h2s.find(h => /Vertrag_Gruber_Maschinenbau/.test(h.textContent || ''))
      return el?.textContent?.trim() ?? ''
    })
    const modalOpen = /Vertrag_Gruber_Maschinenbau/.test(modalTitle)

    const rk = await rawKeys(page)

    await page.screenshot({ path: resolve(outDir, 'b-preview-modal.png'), fullPage: false })
    out.push({
      step: 'b-preview-modal',
      docClicked,
      modalOpen,
      modalTitle: modalTitle.slice(0, 80),
      pass: docClicked && modalOpen,
      rawKeys: rk,
      pageErrors: errs,
    })
  } catch (e) {
    out.push({ step: 'b-preview-modal', pass: false, error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── (c) Versions-Button → VersionHistoryPanel ─────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2000)

    // Open the row with documents
    const targetRow = page.locator('table tbody tr').filter({ hasText: 'Büro-Mietvertrag München' }).first()
    await targetRow.waitFor({ state: 'visible', timeout: 8000 })
    await targetRow.click()
    await page.waitForTimeout(1200)

    // Scroll to Dokumente section
    await scrollDetailPanelToBottom(page)

    // Hover the first document row to make the versions button visible, then click
    const docRow = page.locator('[class*="group"]').filter({ hasText: 'Vertrag_Gruber_Maschinenbau' }).first()
    await docRow.waitFor({ state: 'visible', timeout: 5000 })
    await docRow.hover()
    await page.waitForTimeout(300)

    // Click the versions button (aria-label contains "Version" or the button inside the group)
    const versionsClicked = await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll('button[aria-label]'))
      const btn = btns.find(b => {
        const label = b.getAttribute('aria-label') || ''
        return /[Vv]ersion/.test(label) || /Versionen/.test(label)
      })
      if (btn) {
        // Force visible and click
        btn.style.opacity = '1'
        btn.click()
        return true
      }
      // Fallback: find by VersionIcon (History lucide) inside a group button
      const allBtns = Array.from(document.querySelectorAll('button'))
      const historyBtn = allBtns.find(b => {
        const svg = b.querySelector('svg')
        return svg && b.closest('[class*="group"]') && !b.textContent?.trim()
      })
      if (historyBtn) {
        historyBtn.style.opacity = '1'
        historyBtn.click()
        return true
      }
      return false
    })
    await page.waitForTimeout(1800)

    // Verify VersionHistoryPanel appeared: look for version numbers or "Versionshistorie"
    const panelOpen = await page.evaluate(() => {
      const text = document.body.textContent || ''
      return /Versionshistorie|Aktuelle Version|Ursprungsversion|Version 1|Version 2|dokumente\.version/.test(text) &&
        !/dokumente\.version\.title/.test(text) // raw key guard
    })
    const versionsVisible = await page.evaluate(() => {
      // Check for version number badges in VersionHistoryPanel
      const divs = Array.from(document.querySelectorAll('div'))
      return divs.some(d => /^[12]$/.test(d.textContent?.trim() || '') && d.className.includes('rounded-full'))
    })
    const rk = await rawKeys(page)

    await page.screenshot({ path: resolve(outDir, 'c-versions-panel.png'), fullPage: false })
    out.push({
      step: 'c-versions-panel',
      versionsClicked,
      panelOpen: panelOpen || versionsVisible,
      pass: versionsClicked && (panelOpen || versionsVisible),
      rawKeys: rk,
      pageErrors: errs,
    })
  } catch (e) {
    out.push({ step: 'c-versions-panel', pass: false, error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── (d) ContractDialog: Picker öffnen, Dokument auswählen ─────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2000)

    // Click "Vertrag anlegen" button (i18n key: vertraege.page.buttonNew)
    const newBtn = page.locator('button').filter({ hasText: /Vertrag anlegen|New Contract|Contrat|Nuovo contratto/ }).first()
    await newBtn.waitFor({ state: 'visible', timeout: 5000 })
    const newBtnFound = await newBtn.isVisible()
    await newBtn.click()
    await page.waitForTimeout(1000)

    // Dialog should be open — scroll to the Dokumente section
    await page.evaluate(() => {
      const dialog = document.querySelector('[class*="overflow-y-auto"][class*="rounded-xl"]') ||
        document.querySelector('.glass-elevated.overflow-y-auto')
      if (dialog) dialog.scrollTo({ top: dialog.scrollHeight, behavior: 'instant' })
    })
    await page.waitForTimeout(500)

    // Click "Aus Dokumenten wählen" / "Aus Bibliothek wählen" button
    // i18n key: vertraege.documents.pickFromLibrary
    const pickerBtnLocator = page.locator('button').filter({
      hasText: /Aus Dokumenten|Aus Bibliothek|Pick from|Choisir|Dalla libreria/
    }).first()
    await pickerBtnLocator.waitFor({ state: 'visible', timeout: 5000 })
    const pickerClicked = await pickerBtnLocator.isVisible()
    await pickerBtnLocator.click()
    await page.waitForTimeout(800)

    // Verify the picker dropdown appeared (search input visible)
    const pickerInput = page.locator('input[placeholder*="suchen"], input[placeholder*="Search"], input[placeholder*="Recherch"], input[placeholder*="Cerca"]').first()
    const pickerVisible = await pickerInput.isVisible().catch(() => false)

    // Click the first file in the picker list
    const pickerList = page.locator('[class*="max-h-48"]').first()
    const firstFileBtn = pickerList.locator('button').first()
    let fileSelected = false
    let selectedFileName = ''
    try {
      await firstFileBtn.waitFor({ state: 'visible', timeout: 3000 })
      selectedFileName = (await firstFileBtn.textContent() || '').trim()
      await firstFileBtn.click()
      fileSelected = true
    } catch {
      // Picker list might not have files visible yet
    }
    await page.waitForTimeout(800)

    // Verify the selected file appears as a chip/entry in the document list
    const docAppeared = await page.evaluate((name) => {
      // Look for a document chip with the file name OR any chip in the bg-secondary/30 container
      const spans = Array.from(document.querySelectorAll('[class*="secondary/30"] span, [class*="secondary/30"] div'))
      if (name && spans.some(s => s.textContent?.includes(name.slice(0, 15)))) return true
      // Fallback: any chip row with a FileText icon and a remove X button
      const rows = Array.from(document.querySelectorAll('div[class*="secondary/30"]'))
      return rows.some(r => r.querySelector('svg') && r.querySelector('button'))
    }, selectedFileName)

    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'd-picker-dialog.png'), fullPage: false })
    out.push({
      step: 'd-picker-dialog',
      newBtn: newBtnFound,
      pickerClicked,
      pickerVisible,
      fileSelected,
      selectedFileName: selectedFileName.slice(0, 40),
      docAppeared,
      pass: newBtnFound && pickerClicked && pickerVisible && fileSelected && docAppeared,
      rawKeys: rk,
      pageErrors: errs,
    })
  } catch (e) {
    out.push({ step: 'd-picker-dialog', pass: false, error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── (e) Leerer Zustand (Vertrag ohne Dokumente) ─────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)

  await ctx.addInitScript(`
    try {
      const K = 'cosmi-verträge';
      const raw = localStorage.getItem(K);
      const store = raw ? JSON.parse(raw) : { state: { contracts: [], contractTemplates: [] }, version: 0 };
      const emptyContract = {
        id: 'v-qa-nodocs',
        contractNumber: 'QA-NODOCS-001',
        title: 'QA No Documents',
        partner: 'QA Partner GmbH',
        type: 'lizenz',
        status: 'active',
        startDate: '2025-01-01',
        endDate: '2027-01-01',
        noticePeriodDays: 30,
        renewal: 'manual',
        monthlyCost: 100,
        totalValue: 2400,
        notes: '',
        documents: [],
        history: [],
        currency: 'EUR'
      };
      store.state.contracts = [emptyContract, ...(store.state.contracts || [])];
      localStorage.setItem(K, JSON.stringify(store));
    } catch(e) {}
  `)

  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2000)

    const emptyRow = page.locator('table tbody tr').filter({ hasText: 'QA No Documents' }).first()
    await emptyRow.waitFor({ state: 'visible', timeout: 8000 })
    await emptyRow.click()
    await page.waitForTimeout(1200)

    // Scroll to document section
    await scrollDetailPanelToBottom(page)

    const hasEmptyState = await page.evaluate(() =>
      /Keine Dokumente|No documents|Nessun documento|Aucun document/.test(document.body.textContent || ''))
    const hasDocumentSection = await page.evaluate(() =>
      /DOKUMENTE|Dokumente/.test(document.body.textContent || ''))
    const rk = await rawKeys(page)

    await page.screenshot({ path: resolve(outDir, 'e-empty-state.png'), fullPage: false })
    out.push({
      step: 'e-empty-state',
      hasEmptyState,
      hasDocumentSection,
      pass: hasEmptyState && hasDocumentSection,
      rawKeys: rk,
      pageErrors: errs,
    })
  } catch (e) {
    out.push({ step: 'e-empty-state', pass: false, error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

await browser.close()

// ─── Summary ─────────────────────────────────────────────────────
const allPass = out.every(s => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== QA RESULT: ${allPass ? 'ALL PASS' : 'FAIL'} ===`)
out.forEach(s => console.log(`  ${s.pass ? '✓' : '✗'} ${s.step}`))
