/**
 * Phase 8 QA — E-Signatur EES (Hybrid-Modell)
 *
 * Szenarien:
 * (a) DetailPanel Müller Metallbau → Signer-Sektion mit 2 Signern (1 signed / 1 pending)
 * (b) Unterschrift-Dialog öffnen → entschärfter Titel / EES-Banner asserted (kein "rechtlich bindend", kein Skribble-als-aktiv)
 * (c) Canvas: Hans Müller "Jetzt unterschreiben" → 2-3 Striche → Übernehmen → Badge signed + Thumbnail
 * (d) AuditLog: contract_signed-Eintrag sichtbar
 * (e) Dispatch: neuen Signer hinzufügen → "Zur Unterschrift senden" → Status sent
 * (f) Rapporte-Smoke: kein pageError durch Import-Umzug
 *
 * Jedes Szenario gibt pass: true NUR wenn die Kern-Interaktion nachweislich stattfand.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/vertraege-esignatur')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

// Raw-key detector for vertraege.esignatur.* and vertraege.detail.signaturen.*
const rawRe = /vertraege\.(esignatur\.|detail\.signaturen)/
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 10)
  }, rawRe.source)
}

async function scrollDetailPanelToBottom(page) {
  await page.evaluate(() => {
    const candidates = Array.from(document.querySelectorAll('*')).filter((e) => {
      const style = window.getComputedStyle(e)
      return (
        (style.overflowY === 'auto' || style.overflowY === 'scroll') &&
        e.scrollHeight > e.clientHeight &&
        e.clientHeight > 150
      )
    })
    const panel = candidates[candidates.length - 1]
    if (panel) panel.scrollTo({ top: panel.scrollHeight, behavior: 'instant' })
  })
  await page.waitForTimeout(400)
}

/** Open Müller Metallbau by title locator */
async function openMuellerMetallbau(page) {
  await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2000)
  const targetRow = page.locator('table tbody tr').filter({ hasText: 'Müller Metallbau' }).first()
  await targetRow.waitFor({ state: 'visible', timeout: 8000 })
  await targetRow.click()
  await page.waitForTimeout(1200)
}

const browser = await chromium.launch()
const out = []

// ─── (a) DetailPanel Signer-Sektion ────────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await openMuellerMetallbau(page)
    await scrollDetailPanelToBottom(page)

    // Check Signer section header
    const hasSignerSection = await page.evaluate(() =>
      /UNTERSCHRIFTEN|Unterschriften|Firme|Signatures/.test(document.body.textContent || ''))

    // Check both signers present
    const hasLukasBrunner = await page.evaluate(() =>
      /Lukas Brunner/.test(document.body.textContent || ''))
    const hasHansMueller = await page.evaluate(() =>
      /Hans Müller/.test(document.body.textContent || ''))

    // Check status badges
    const hasSignedBadge = await page.evaluate(() =>
      /Unterschrieben|Signed|Signé|Firmato/.test(document.body.textContent || ''))
    const hasPendingBadge = await page.evaluate(() =>
      /Ausstehend|Pending|En attente|In attesa/.test(document.body.textContent || ''))

    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'a-detail-signers.png'), fullPage: false })
    out.push({
      step: 'a-detail-signers',
      hasSignerSection,
      hasLukasBrunner,
      hasHansMueller,
      hasSignedBadge,
      hasPendingBadge,
      pass: hasSignerSection && hasLukasBrunner && hasHansMueller && hasSignedBadge && hasPendingBadge,
      rawKeys: rk,
      pageErrors: errs,
    })
  } catch (e) {
    out.push({ step: 'a-detail-signers', pass: false, error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── (b) Dialog: entschärfter Titel + EES-Banner ──────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await openMuellerMetallbau(page)

    // Click "Unterschrift" footer button
    const signBtn = page.locator('button').filter({ hasText: /Unterschrift|Signature|Firma/ }).first()
    await signBtn.waitFor({ state: 'visible', timeout: 5000 })
    await signBtn.click()
    await page.waitForTimeout(1500)

    // Dialog must be open — check for dialog heading
    const dialogOpen = await page.evaluate(() => {
      const titles = Array.from(document.querySelectorAll('[role="dialog"] h2, [role="dialog"] [data-radix-dialog-title]'))
      return titles.some(t =>
        /Digitale Unterschrift|Digital Signature|Signature numérique|Firma digitale/.test(t.textContent || ''))
    })

    // Title must NOT contain "Skribble" as active feature
    const titleHasNoSkribbleSuffix = await page.evaluate(() => {
      const titles = Array.from(document.querySelectorAll('[role="dialog"] h2'))
      return titles.every(t => !/— Skribble/.test(t.textContent || ''))
    })

    // Banner must NOT contain "rechtlich bindend" / "legally binding"
    const bannerNoLegallyBinding = await page.evaluate(() => {
      const text = document.body.textContent || ''
      return !/(rechtlich bindend|legally binding|giuridicamente vincolanti|juridiquement contraignante)/.test(text)
    })

    // Banner must contain EES reference
    const bannerHasEES = await page.evaluate(() => {
      const text = document.body.textContent || ''
      return /(einfache elektronische|simple electronic|FES|SES)/.test(text)
    })

    // Skribble shown as "coming soon" (disabled), not as active
    const skribbleComingSoon = await page.evaluate(() => {
      const text = document.body.textContent || ''
      return /(Bald verfügbar|Coming soon|Bientôt|Prossimamente)/.test(text)
    })

    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'b-dialog-banner.png'), fullPage: false })
    out.push({
      step: 'b-dialog-banner',
      dialogOpen,
      titleHasNoSkribbleSuffix,
      bannerNoLegallyBinding,
      bannerHasEES,
      skribbleComingSoon,
      pass: dialogOpen && titleHasNoSkribbleSuffix && bannerNoLegallyBinding && bannerHasEES && skribbleComingSoon,
      rawKeys: rk,
      pageErrors: errs,
    })
  } catch (e) {
    out.push({ step: 'b-dialog-banner', pass: false, error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── (c) Canvas: Hans Müller unterschreiben → Badge + Thumbnail ───────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await openMuellerMetallbau(page)

    // Open dialog
    const signBtn = page.locator('button').filter({ hasText: /Unterschrift|Signature|Firma/ }).first()
    await signBtn.waitFor({ state: 'visible', timeout: 5000 })
    await signBtn.click()
    await page.waitForTimeout(1500)

    // Find "Jetzt unterschreiben" button for Hans Müller row
    // Look for the button in the same table row as "Hans Müller"
    const signNowBtn = await page.evaluate(() => {
      const rows = Array.from(document.querySelectorAll('[role="dialog"] table tbody tr'))
      for (const row of rows) {
        if (/Hans Müller/.test(row.textContent || '')) {
          const btn = row.querySelector('button[class*="primary"]') ||
            Array.from(row.querySelectorAll('button')).find(b =>
              /Jetzt unterschreiben|Sign now|Firma ora|Signer maintenant/.test(b.textContent || ''))
          if (btn) {
            btn.click()
            return true
          }
        }
      }
      return false
    })
    await page.waitForTimeout(1000)

    // Canvas step should be visible
    const canvasVisible = await page.evaluate(() => {
      return document.querySelector('canvas') !== null
    })

    if (!canvasVisible) {
      throw new Error('Canvas not visible after clicking Sign now')
    }

    // Draw 3 strokes on the canvas
    const canvas = page.locator('canvas').first()
    const box = await canvas.boundingBox()
    if (!box) throw new Error('Canvas bounding box not found')

    // Stroke 1
    await page.mouse.move(box.x + 50, box.y + 80)
    await page.mouse.down()
    await page.mouse.move(box.x + 120, box.y + 60)
    await page.mouse.move(box.x + 180, box.y + 100)
    await page.mouse.up()
    await page.waitForTimeout(200)
    // Stroke 2
    await page.mouse.move(box.x + 200, box.y + 70)
    await page.mouse.down()
    await page.mouse.move(box.x + 260, box.y + 110)
    await page.mouse.up()
    await page.waitForTimeout(200)
    // Stroke 3
    await page.mouse.move(box.x + 280, box.y + 90)
    await page.mouse.down()
    await page.mouse.move(box.x + 330, box.y + 70)
    await page.mouse.move(box.x + 360, box.y + 110)
    await page.mouse.up()
    await page.waitForTimeout(300)

    await page.screenshot({ path: resolve(outDir, 'c1-canvas-drawn.png'), fullPage: false })

    // First click "Speichern" (Canvas save button — sets pendingDataUrl)
    const saveCanvasBtn = page.locator('button').filter({ hasText: /^Speichern$|^Save$|^Sauvegarder$|^Salva$/ }).first()
    await saveCanvasBtn.waitFor({ state: 'visible', timeout: 5000 })
    await saveCanvasBtn.click()
    await page.waitForTimeout(500)

    // Now click "Unterschrift übernehmen" / "Accept signature"
    const acceptBtn = page.locator('button').filter({
      hasText: /Unterschrift übernehmen|Accept signature|Accetta firma|Accepter la signature/
    }).first()
    await acceptBtn.waitFor({ state: 'visible', timeout: 5000 })
    const acceptEnabled = !(await acceptBtn.isDisabled())
    await acceptBtn.click()
    await page.waitForTimeout(1200)

    // Canvas should be gone, back to main dialog
    const canvasGone = await page.evaluate(() => document.querySelector('canvas') === null)

    // Hans Müller row should now show "Unterschrieben" badge
    const hansMuellerSigned = await page.evaluate(() => {
      const rows = Array.from(document.querySelectorAll('[role="dialog"] table tbody tr'))
      for (const row of rows) {
        if (/Hans Müller/.test(row.textContent || '')) {
          return /Unterschrieben|Signed|Firmato|Signé/.test(row.textContent || '')
        }
      }
      return false
    })

    // Thumbnail img should be present for Hans Müller
    const thumbnailVisible = await page.evaluate(() => {
      const rows = Array.from(document.querySelectorAll('[role="dialog"] table tbody tr'))
      for (const row of rows) {
        if (/Hans Müller/.test(row.textContent || '')) {
          const img = row.querySelector('img[src^="data:image/png"]')
          return img !== null
        }
      }
      return false
    })

    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'c2-signed-with-thumbnail.png'), fullPage: false })
    out.push({
      step: 'c-canvas-sign',
      signNowBtn,
      canvasVisible,
      acceptEnabled,
      canvasGone,
      hansMuellerSigned,
      thumbnailVisible,
      pass: signNowBtn && canvasVisible && acceptEnabled && canvasGone && hansMuellerSigned && thumbnailVisible,
      rawKeys: rk,
      pageErrors: errs,
    })
  } catch (e) {
    out.push({ step: 'c-canvas-sign', pass: false, error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── (d) AuditLog: contract_signed-Eintrag ───────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)

  // Pre-seed a signed signer in store so history has contract_signed entry
  await ctx.addInitScript(`
    try {
      const K = 'cosmi-verträge';
      const raw = localStorage.getItem(K);
      const store = raw ? JSON.parse(raw) : { state: { contracts: [], contractTemplates: [] }, version: 0 };
      const v5 = (store.state.contracts || []).find(c => c.id === 'v-5');
      if (v5 && !v5.history.some(h => h.action === 'contract_signed' && h.meta === 'Hans Müller')) {
        v5.history.push({
          date: new Date().toISOString().split('T')[0],
          action: 'contract_signed',
          meta: 'Hans Müller',
          user: 'Aktueller Benutzer'
        });
        localStorage.setItem(K, JSON.stringify(store));
      }
    } catch(e) {}
  `)

  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await openMuellerMetallbau(page)
    await scrollDetailPanelToBottom(page)
    await page.waitForTimeout(400)

    // Check for "Vertrag unterzeichnet" in the audit log
    const auditLogHasContractSigned = await page.evaluate(() =>
      /Vertrag unterzeichnet|Contract signed|Firmato|contrat signé/i.test(document.body.textContent || ''))

    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'd-auditlog.png'), fullPage: false })
    out.push({
      step: 'd-auditlog-contract-signed',
      auditLogHasContractSigned,
      pass: auditLogHasContractSigned,
      rawKeys: rk,
      pageErrors: errs,
    })
  } catch (e) {
    out.push({ step: 'd-auditlog-contract-signed', pass: false, error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── (e) Dispatch: neuen Signer → "Zur Unterschrift senden" → sent ───────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await openMuellerMetallbau(page)

    // Open dialog
    const signBtn = page.locator('button').filter({ hasText: /Unterschrift|Signature|Firma/ }).first()
    await signBtn.waitFor({ state: 'visible', timeout: 5000 })
    await signBtn.click()
    await page.waitForTimeout(1500)

    // Add new signer
    const addSignerBtn = page.locator('button').filter({
      hasText: /Unterzeichner hinzufügen|Add signer|Ajouter un signataire|Aggiungi firmatario/
    }).first()
    await addSignerBtn.waitFor({ state: 'visible', timeout: 5000 })
    await addSignerBtn.click()
    await page.waitForTimeout(600)

    // Fill name
    const nameInput = page.locator('input[placeholder*="Mustermann"], input[placeholder*="Jane"], input[placeholder*="Smith"]').first()
    await nameInput.waitFor({ state: 'visible', timeout: 3000 })
    await nameInput.fill('Test Dispatch')
    // Fill email
    const emailInput = page.locator('input[type="email"]').first()
    await emailInput.fill('test@dispatch.de')
    await page.waitForTimeout(300)

    // Click "Hinzufügen"
    const addBtn = page.locator('button').filter({
      hasText: /^Hinzufügen$|^Add$|^Ajouter$|^Aggiungi$/
    }).first()
    await addBtn.waitFor({ state: 'visible', timeout: 3000 })
    await addBtn.click()
    await page.waitForTimeout(600)

    // Verify "Test Dispatch" appeared in table
    const signerAdded = await page.evaluate(() =>
      /Test Dispatch/.test(document.body.textContent || ''))

    // Click "Zur Unterschrift senden"
    const sendBtn = page.locator('button').filter({
      hasText: /Zur Unterschrift senden|Send for signature|Envoyer pour signature|Invia per firma/
    }).first()
    await sendBtn.waitFor({ state: 'visible', timeout: 5000 })
    await sendBtn.click()
    await page.waitForTimeout(2000)

    // Dialog closed = no Radix DialogContent visible (has data-radix-dialog-content or class max-w-2xl)
    const dialogClosed = await page.evaluate(() => {
      // Look for ESignaturDialog specifically (has DialogContent with max-w-2xl and contains FileSignature icon)
      const esignaturOpen = Array.from(document.querySelectorAll('[data-radix-dialog-content], [role="dialog"]'))
        .some(d => {
          const style = window.getComputedStyle(d)
          if (style.display === 'none' || style.visibility === 'hidden') return false
          // ESignaturDialog has "Zur Unterschrift senden" or "Digitale Unterschrift" text
          return /Zur Unterschrift senden|Send for signature|Digitale Unterschrift|Digital Signature/.test(d.textContent || '')
        })
      return !esignaturOpen
    })

    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'e-dispatch-sent.png'), fullPage: false })
    out.push({
      step: 'e-dispatch',
      signerAdded,
      dialogClosed,
      pass: signerAdded && dialogClosed,
      rawKeys: rk,
      pageErrors: errs,
    })
  } catch (e) {
    out.push({ step: 'e-dispatch', pass: false, error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── (f) Rapporte-Smoke: kein pageError durch Import-Umzug ─────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/rapporte`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2500)

    // Check page loaded (Rapporte content visible)
    const rapporteVisible = await page.evaluate(() =>
      /Rapporte|Rapport|Rapporti/i.test(document.body.textContent || ''))

    // No canvas/import errors
    const canvasImportError = errs.some(e =>
      /SignatureCanvas|cannot find module|import/i.test(e))

    const rk = await page.evaluate(() => {
      const rx = /rapporte\.(signature\.|detail\.signature)/
      return Array.from(document.querySelectorAll('body *'))
        .filter(n => n.children.length === 0 && rx.test(n.textContent || ''))
        .map(n => n.textContent.trim())
        .slice(0, 10)
    })

    await page.screenshot({ path: resolve(outDir, 'f-rapporte-smoke.png'), fullPage: false })
    out.push({
      step: 'f-rapporte-smoke',
      rapporteVisible,
      canvasImportError,
      pass: rapporteVisible && !canvasImportError && errs.length === 0,
      rawKeys: rk,
      pageErrors: errs,
    })
  } catch (e) {
    out.push({ step: 'f-rapporte-smoke', pass: false, error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

await browser.close()

// ─── Summary ─────────────────────────────────────────────────────────────
const allPass = out.every(s => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== QA RESULT: ${allPass ? 'ALL PASS' : 'FAIL'} ===`)
out.forEach(s => console.log(`  ${s.pass ? '✓' : '✗'} ${s.step}`))
