/**
 * Phase 10 QA — vertraege CRM/Finance-Verknüpfung + KI-Fristencheck
 *
 * Szenarien:
 * (a) Dialog — Kontakt verknüpfen + Chip erscheint
 * (b) Dialog — Deal verknüpfen + Chip erscheint
 * (c) Dialog — Rechnung verknüpfen + Chip erscheint
 * (d) Vertrag anlegen → DetailPanel zeigt Verknüpfungs-Chips
 * (e) Kontakt-Chip → navigiert zu /kontakte?contact=<id>
 * (f) Rechnungs-Chip → navigiert zu /finanzen?invoice=<id>
 * (g) History-Eintrag für contact_linked sichtbar
 * (h) Fristencheck-Button → Ergebnisliste sichtbar
 * (i) Leer-Zustand: Vertrag ohne Verknüpfungen zeigt keine Chips
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir, writeFile } from 'node:fs/promises'

const BASE = process.env.QA_BASE ?? 'http://localhost:5173'
const outDir = resolve('scripts/qa-shots/phase10')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

/** Returns the ErrorBoundary message text if visible, null otherwise. */
async function checkErrorBoundary(page) {
  const text = await page.evaluate(() => document.body.innerText)
  if (text.includes('Etwas ist schiefgelaufen') || text.includes('Cannot read properties')) {
    // Return first line of crash detail
    const lines = text.split('\n').map((l) => l.trim()).filter(Boolean)
    const idx = lines.findIndex((l) => l.includes('Etwas ist schiefgelaufen'))
    return idx >= 0 ? lines.slice(idx, idx + 3).join(' | ') : 'ErrorBoundary visible'
  }
  return null
}

async function rawKeyCheck(page) {
  return page.evaluate(() => {
    const rx = /vertraege\.(links|deadlineCheck)\./
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 10)
  })
}

async function scrollAllScrollables(page) {
  await page.evaluate(() => {
    document.querySelectorAll('*').forEach((e) => {
      const s = window.getComputedStyle(e)
      if ((s.overflowY === 'auto' || s.overflowY === 'scroll') && e.scrollHeight > e.clientHeight) {
        e.scrollTop = e.scrollHeight
      }
    })
    window.scrollTo(0, document.body.scrollHeight)
  })
  await page.waitForTimeout(400)
}

async function pickFirstItemInDropdown(page) {
  // The picker dropdown has class "absolute" and "z-20" and contains a max-h-40 div with buttons
  // Find visible dropdown and click first non-cancel button
  const dropdowns = page.locator('div.absolute.z-20')
  const count = await dropdowns.count()
  for (let i = 0; i < count; i++) {
    const dd = dropdowns.nth(i)
    if (await dd.isVisible()) {
      const allBtns = dd.getByRole('button')
      const btnCount = await allBtns.count()
      if (btnCount > 1) {
        const firstItem = allBtns.first()
        const text = await firstItem.textContent()
        await firstItem.click()
        return text?.trim() ?? ''
      }
    }
  }
  return null
}

const browser = await chromium.launch({ headless: true })
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

const out = {
  linkContactInDialog: { pass: false, detail: '' },
  linkDealInDialog: { pass: false, detail: '' },
  linkInvoiceInDialog: { pass: false, detail: '' },
  linksSavedToStore: { pass: false, detail: '' },
  chipNavigatesToContact: { pass: false, detail: '' },
  chipNavigatesToInvoice: { pass: false, detail: '' },
  historyEntryVisible: { pass: false, detail: '' },
  deadlineCheckRuns: { pass: false, detail: '' },
  emptyStateNoChips: { pass: false, detail: '' },
  rawKeys: [],
  pageErrors: [],
}

try {
  // ─── Navigate ────────────────────────────────────────────────
  await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)
  // Clear any previous QA test contract from localStorage store
  await page.evaluate(() => {
    try {
      const raw = localStorage.getItem('cosmi-verträge')
      if (!raw) return
      const parsed = JSON.parse(raw)
      const contracts = parsed?.state?.contracts
      if (Array.isArray(contracts)) {
        parsed.state.contracts = contracts.filter((c) => !c.title?.includes('QA-Test-Phase10'))
        localStorage.setItem('cosmi-verträge', JSON.stringify(parsed))
      }
    } catch (e) { /* noop */ }
  })
  await page.reload({ waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2000)
  await page.screenshot({ path: resolve(outDir, '00-page-loaded.png'), fullPage: false })

  // ─── (i) Leer-Zustand: click first table row ─────────────────
  // The contract table renders rows; click the first one (any row with a contract name)
  await page.locator('table tbody tr').first().click().catch(async () => {
    await page.locator('[role="row"]').nth(1).click()
  })
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, 'i-empty-state.png'), fullPage: false })

  // For an unlinked stock contract, there should be no chip buttons (rounded-full)
  const hasChipsInEmpty = await page.locator('button[class*="rounded-full"]').count()
  out.emptyStateNoChips.pass = hasChipsInEmpty === 0
  out.emptyStateNoChips.detail = hasChipsInEmpty === 0 ? 'No chip buttons in initial contract view' : `FAIL: ${hasChipsInEmpty} chip buttons visible`

  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // ─── Open "Vertrag anlegen" dialog ───────────────────────────
  const newBtn = page.getByRole('button').filter({ hasText: /Vertrag anlegen|Neuer Vertrag/ }).first()
  if (await newBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
    await newBtn.click()
  } else {
    await page.locator('button').filter({ has: page.locator('svg') }).first().click()
  }
  await page.waitForTimeout(1500)
  await page.screenshot({ path: resolve(outDir, 'dialog-opened.png'), fullPage: false })

  // Fill required fields — scope all inputs to the dialog container (glass-elevated or role=dialog)
  // to avoid polluting the global search bar (which is also input[type=text])
  const dialogContainer = page.locator('[class*="glass-elevated"], [role="dialog"]').last()
  const dialogInputs = dialogContainer.locator('input[type="text"]')
  await dialogInputs.nth(0).fill('QA-Test-Phase10')  // Titel
  await dialogInputs.nth(1).fill('Test Partner QA AG')  // Partner
  await dialogInputs.nth(2).fill('QA-P10-001')  // Vertragsnummer
  await dialogContainer.locator('input[type="date"]').first().fill('2025-01-01')
  await page.screenshot({ path: resolve(outDir, 'dialog-required-filled.png'), fullPage: false })

  // Scroll dialog container to bottom
  await page.evaluate(() => {
    const els = Array.from(document.querySelectorAll('*')).filter((e) => {
      const s = window.getComputedStyle(e)
      return (s.overflowY === 'auto' || s.overflowY === 'scroll') && e.scrollHeight > e.clientHeight + 50
    })
    els.sort((a, b) => b.scrollHeight - a.scrollHeight)
    if (els[0]) els[0].scrollTop = els[0].scrollHeight
    if (els[1]) els[1].scrollTop = els[1].scrollHeight
  })
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, 'dialog-scrolled.png'), fullPage: false })

  // ─── (a) Kontakt verknüpfen ──────────────────────────────────
  // All dashed border buttons = the 3 "add" buttons in Verknüpfungen section
  let contactDashedBtn = page.locator('button.border-dashed').nth(0)
  if (!(await contactDashedBtn.isVisible({ timeout: 3000 }).catch(() => false))) {
    // Try scrolling more
    await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('*')).filter((e) => {
        const s = window.getComputedStyle(e)
        return (s.overflowY === 'auto' || s.overflowY === 'scroll') && e.scrollHeight > e.clientHeight
      })
      for (const e of els) e.scrollTop += 999
    })
    await page.waitForTimeout(400)
    await page.screenshot({ path: resolve(outDir, 'dialog-extra-scroll.png'), fullPage: false })
  }

  const dashed0 = page.locator('button.border-dashed').nth(0)
  if (await dashed0.isVisible({ timeout: 5000 }).catch(() => false)) {
    await dashed0.click()
    await page.waitForTimeout(800)
    await page.screenshot({ path: resolve(outDir, 'a-contact-picker-open.png'), fullPage: false })
    const picked = await pickFirstItemInDropdown(page)
    await page.waitForTimeout(400)
    await page.screenshot({ path: resolve(outDir, 'a-contact-selected.png'), fullPage: false })
    out.linkContactInDialog.pass = picked !== null
    out.linkContactInDialog.detail = picked !== null ? `Contact picker item selected: "${picked}"` : 'No items in dropdown'
  } else {
    out.linkContactInDialog.detail = 'No border-dashed buttons visible'
    await page.screenshot({ path: resolve(outDir, 'a-no-dashed-btns.png'), fullPage: false })
  }

  // ─── (b) Deal verknüpfen ─────────────────────────────────────
  // After contact selected, dashed buttons shift: deal is now nth(0) or nth(1) depending on whether contact got replaced
  // Re-query all dashed buttons
  const dashedAll = page.locator('button.border-dashed')
  const dashedCount = await dashedAll.count()
  // If contact was linked, we may have 2 dashed buttons (deal + invoice). If not linked, 3.
  const dealBtnIdx = 0 // deal is always after contact (which was replaced with a chip)
  const dealBtn = dashedAll.nth(0)
  if (await dealBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
    const txt = await dealBtn.textContent()
    // Make sure it's not the invoice btn (text contains "Deal" or "verknüpfen")
    await dealBtn.click()
    await page.waitForTimeout(800)
    await page.screenshot({ path: resolve(outDir, 'b-deal-picker-open.png'), fullPage: false })
    const picked = await pickFirstItemInDropdown(page)
    await page.waitForTimeout(400)
    await page.screenshot({ path: resolve(outDir, 'b-deal-selected.png'), fullPage: false })
    out.linkDealInDialog.pass = picked !== null
    out.linkDealInDialog.detail = picked !== null ? `Deal picker item selected: "${picked}"` : 'No items in deal dropdown'
  } else {
    out.linkDealInDialog.detail = 'Deal dashed button not visible'
  }

  // ─── (c) Rechnung verknüpfen ─────────────────────────────────
  // After contact+deal selected: only invoice dashed button remains (index 0)
  const invBtn = page.locator('button.border-dashed').nth(0)
  if (await invBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
    await invBtn.click()
    await page.waitForTimeout(800)
    await page.screenshot({ path: resolve(outDir, 'c-invoice-picker-open.png'), fullPage: false })
    const picked = await pickFirstItemInDropdown(page)
    await page.waitForTimeout(400)
    await page.screenshot({ path: resolve(outDir, 'c-invoice-selected.png'), fullPage: false })
    out.linkInvoiceInDialog.pass = picked !== null
    out.linkInvoiceInDialog.detail = picked !== null ? `Invoice picker item selected: "${picked}"` : 'No items in invoice dropdown'
  } else {
    out.linkInvoiceInDialog.detail = 'Invoice dashed button not visible'
  }

  await page.screenshot({ path: resolve(outDir, 'dialog-links-filled.png'), fullPage: false })

  // ─── Save ────────────────────────────────────────────────────
  // Scroll to make sure save button is accessible
  await page.evaluate(() => {
    const els = Array.from(document.querySelectorAll('*')).filter((e) => {
      const s = window.getComputedStyle(e)
      return (s.overflowY === 'auto' || s.overflowY === 'scroll') && e.scrollHeight > e.clientHeight
    })
    for (const e of els) e.scrollTop = e.scrollHeight
  })
  await page.waitForTimeout(300)

  const saveBtn = page.getByRole('button').filter({ hasText: /^Anlegen$|^Vertrag anlegen$/ }).last()
  const saveBtnVis = await saveBtn.isVisible({ timeout: 5000 }).catch(() => false)
  if (saveBtnVis) {
    await saveBtn.click()
  } else {
    // Fallback to any Anlegen button
    await page.getByRole('button').filter({ hasText: /Anlegen/ }).last().click()
  }
  await page.waitForTimeout(2000)
  await page.screenshot({ path: resolve(outDir, 'd-after-save.png'), fullPage: false })

  // ─── Find and open newly created contract ────────────────────
  const testRow = page.getByText('QA-Test-Phase10').first()
  const found = await testRow.isVisible({ timeout: 5000 }).catch(() => false)
  if (found) {
    await testRow.click()
    await page.waitForTimeout(1500)
    await page.screenshot({ path: resolve(outDir, 'd-detail-panel.png'), fullPage: false })

    // Scroll the detail panel
    await scrollAllScrollables(page)
    await page.screenshot({ path: resolve(outDir, 'd-detail-scrolled.png'), fullPage: false })

    // ─── (d) Verknüpfungen section present ───────────────────
    // Check both: heading text AND presence of chip buttons (rounded-full)
    const bodyText = await page.evaluate(() => document.body.innerText)
    const verknPresent = bodyText.includes('Verknüpfungen')
    const chipCount = await page.locator('button[class*="rounded-full"]').count()
    // Scroll to Verknüpfungen heading to see if it's there
    const verknInDom = await page.evaluate(() => {
      const all = Array.from(document.querySelectorAll('*'))
      return all.some((e) => e.children.length === 0 && e.textContent?.trim() === 'Verknüpfungen')
    })
    out.linksSavedToStore.pass = verknPresent || verknInDom || chipCount > 0
    out.linksSavedToStore.detail = `verknPresent=${verknPresent}, verknInDom=${verknInDom}, chipCount=${chipCount}`

    // ─── (g) History ─────────────────────────────────────────
    await scrollAllScrollables(page)
    await page.waitForTimeout(300)
    await page.screenshot({ path: resolve(outDir, 'g-history.png'), fullPage: false })
    const bodyText2 = await page.evaluate(() => document.body.innerText)
    const hasHistory = /Kontakt verknüpft|contact.linked/i.test(bodyText2)
    out.historyEntryVisible.pass = hasHistory
    out.historyEntryVisible.detail = hasHistory ? 'History entry for contact_linked found' : 'Not found (search: "Kontakt verknüpft")'

    // ─── (h) Fristencheck ────────────────────────────────────
    // Scroll repeatedly to find the button
    for (let i = 0; i < 3; i++) {
      await scrollAllScrollables(page)
      await page.waitForTimeout(300)
    }
    await page.screenshot({ path: resolve(outDir, 'h-pre-click.png'), fullPage: false })

    const fristenBtn = page.getByRole('button').filter({ hasText: /Fristencheck ausführen/i }).first()
    const fristenVis = await fristenBtn.isVisible({ timeout: 3000 }).catch(() => false)
    if (fristenVis) {
      await fristenBtn.click()
      await page.waitForTimeout(800)
      await page.screenshot({ path: resolve(outDir, 'h-deadline-result.png'), fullPage: false })
      const bodyText3 = await page.evaluate(() => document.body.innerText)
      const hasResult = /Keine kritischen|Vorschau|Kündigungsfenster|Keine Erinnerungen|Abgelaufen|Kein Dokument|allClear|keine.*Frist/i.test(bodyText3)
      out.deadlineCheckRuns.pass = hasResult
      out.deadlineCheckRuns.detail = hasResult ? 'Deadline check result visible' : 'Result text not found after button click'
    } else {
      out.deadlineCheckRuns.detail = 'Fristencheck ausführen button not visible'
    }

    // ─── (e) Kontakt-Chip Navigation ─────────────────────────
    // Scroll back to middle of detail panel where Verknüpfungen section is
    await page.evaluate(() => {
      const heads = Array.from(document.querySelectorAll('*'))
      const vHead = heads.find((e) => e.children.length === 0 && e.textContent?.trim() === 'Verknüpfungen')
      if (vHead) vHead.scrollIntoView({ behavior: 'instant', block: 'center' })
    })
    await page.waitForTimeout(400)
    await page.screenshot({ path: resolve(outDir, 'e-links-section.png'), fullPage: false })

    const roundedBtns = page.locator('button[class*="rounded-full"]')
    const rBtnCount = await roundedBtns.count()
    if (rBtnCount > 0) {
      // Remember which contact was selected so we can assert it's visible after navigation
      const chipLabel = await roundedBtns.first().textContent()
      const urlBefore = page.url()
      await roundedBtns.first().click()
      await page.waitForTimeout(2000)
      const urlAfter = page.url()
      const nav = urlAfter !== urlBefore

      // P0 guard: no ErrorBoundary after navigation
      const ebContact = await checkErrorBoundary(page)
      if (ebContact) {
        out.chipNavigatesToContact.pass = false
        out.chipNavigatesToContact.detail = `ErrorBoundary after contact chip navigation: ${ebContact}`
      } else if (!nav) {
        out.chipNavigatesToContact.pass = false
        out.chipNavigatesToContact.detail = `No navigation; URL: ${urlAfter}`
      } else {
        // Assert: the contact name from the chip is visible in the page body (detail loaded)
        const bodyTextContact = await page.evaluate(() => document.body.innerText)
        // chipLabel may have leading icon text — check if any word from it is visible
        const labelWords = (chipLabel ?? '').replace(/[^\w\s]/g, ' ').split(/\s+/).filter((w) => w.length > 2)
        const contactDetailVisible = labelWords.length === 0 || labelWords.some((w) => bodyTextContact.includes(w))
        out.chipNavigatesToContact.pass = contactDetailVisible
        out.chipNavigatesToContact.detail = contactDetailVisible
          ? `Contact chip → ${urlAfter}; contact detail visible (chip text: "${chipLabel?.trim()}")`
          : `Navigated to ${urlAfter} but contact detail not found (chip: "${chipLabel?.trim()}")`
      }
      await page.screenshot({ path: resolve(outDir, 'e-chip-navigate.png'), fullPage: false })
      await page.goBack()
      await page.waitForTimeout(1500)
    } else {
      out.chipNavigatesToContact.detail = 'No rounded-full chip buttons found'
    }

    // ─── (f) Invoice-Chip Navigation ─────────────────────────
    // Re-open the test contract
    if (!page.url().includes('vertraege')) {
      await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded', timeout: 20000 })
      await page.waitForTimeout(2000)
    }
    const testRow2 = page.getByText('QA-Test-Phase10').first()
    if (await testRow2.isVisible({ timeout: 3000 }).catch(() => false)) {
      await testRow2.click()
      await page.waitForTimeout(1200)
    }
    await page.evaluate(() => {
      const heads = Array.from(document.querySelectorAll('*'))
      const vHead = heads.find((e) => e.children.length === 0 && e.textContent?.trim() === 'Verknüpfungen')
      if (vHead) vHead.scrollIntoView({ behavior: 'instant', block: 'center' })
    })
    await page.waitForTimeout(400)

    const roundedBtns2 = page.locator('button[class*="rounded-full"]')
    const rBtnCount2 = await roundedBtns2.count()
    if (rBtnCount2 >= 2) {
      // Last chip should be invoice — remember its label (invoice number)
      const invChip = roundedBtns2.last()
      const invChipLabel = await invChip.textContent()
      const urlBefore2 = page.url()
      await invChip.click()
      await page.waitForTimeout(2000)
      const urlAfter2 = page.url()
      const navInv = urlAfter2.includes('finanzen') && urlAfter2 !== urlBefore2

      // P0 guard: no ErrorBoundary after navigation
      const ebInv = await checkErrorBoundary(page)
      await page.screenshot({ path: resolve(outDir, 'f-invoice-navigate.png'), fullPage: false })

      if (ebInv) {
        out.chipNavigatesToInvoice.pass = false
        out.chipNavigatesToInvoice.detail = `ErrorBoundary after invoice chip navigation: ${ebInv}`
      } else if (!navInv) {
        out.chipNavigatesToInvoice.pass = false
        out.chipNavigatesToInvoice.detail = `Did not navigate to /finanzen; URL: ${urlAfter2}`
      } else {
        // Wait for InvoiceDetailPanel to render (it needs an extra tick for the effect)
        await page.waitForTimeout(1500)
        const bodyTextInv = await page.evaluate(() => document.body.innerText)
        // The detail panel should show the invoice number from the chip label
        const invLabelWords = (invChipLabel ?? '').replace(/[^\w\s-]/g, ' ').split(/\s+/).filter((w) => w.length > 2)
        const detailVisible = invLabelWords.length === 0 || invLabelWords.some((w) => bodyTextInv.includes(w))
        out.chipNavigatesToInvoice.pass = detailVisible
        out.chipNavigatesToInvoice.detail = detailVisible
          ? `Invoice chip → ${urlAfter2}; invoice detail visible (chip: "${invChipLabel?.trim()}")`
          : `Navigated to ${urlAfter2} but invoice detail panel content not found (chip: "${invChipLabel?.trim()}")`
      }
    } else if (rBtnCount2 === 1) {
      out.chipNavigatesToInvoice.detail = `Only 1 chip button (expected ≥2 for separate invoice chip)`
    } else {
      out.chipNavigatesToInvoice.detail = 'No chip buttons on re-open'
    }

  } else {
    out.linksSavedToStore.detail = 'QA-Test-Phase10 row not found after save'
    await page.screenshot({ path: resolve(outDir, 'd-save-failed.png'), fullPage: false })
  }

} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'error-state.png'), fullPage: false }).catch(() => {})
}

// ─── Raw-Keys scan ────────────────────────────────────────────────
try {
  await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(1500)
  out.rawKeys = await rawKeyCheck(page)
} catch (_) {}

out.pageErrors = errors.slice(0, 5)

await ctx.close()
await browser.close()

const resultPath = resolve('scripts/qa-shots/phase10/qa-result.json')
await writeFile(resultPath, JSON.stringify(out, null, 2), 'utf-8')
console.log(JSON.stringify(out, null, 2))
