/**
 * QA-Script für das Einkauf-Modul (Playwright, Demo-Mode/MSW).
 *
 * Prüft: Bestellungen-Liste, Lieferanten-Grid, Katalog-Grid, Rahmenverträge,
 * Detail-Panels, Loading-States, Raw-i18n-Keys, Emojis, leere Zustände.
 *
 * Voraussetzung: Dev-Server läuft (`npm run dev`, :5173, --mode demo).
 *   node desktop/scripts/qa-einkauf.mjs
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('scripts/screenshots/einkauf')
await mkdir(outDir, { recursive: true })

// Suppress electronAPI + skip onboarding
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()

const errors = []
page.on('pageerror', (e) => errors.push(String(e)))

const shot = async (name) => {
  await page.screenshot({ path: resolve(outDir, `${name}.png`), fullPage: false })
  console.log(`  shot: ${name}.png`)
}

const clickTabByText = async (needle) => {
  const tabs = await page.$$('button')
  for (const t of tabs) {
    const txt = (await page.evaluate((el) => el.textContent, t)) || ''
    if (txt.includes(needle)) {
      await t.click()
      await page.waitForTimeout(700)
      return true
    }
  }
  return false
}

const checkRawKeys = async (label) => {
  const body = await page.textContent('body')
  const rawKeys = (body || '').match(/einkauf\.[a-z.]+/g) || []
  // Filter out valid translated strings that happen to look like keys
  const suspicious = rawKeys.filter(k => !k.includes(' ') && k.split('.').length >= 3)
  if (suspicious.length > 0) {
    console.warn(`  WARN ${label}: possible raw i18n keys: ${[...new Set(suspicious)].slice(0, 5).join(', ')}`)
  } else {
    console.log(`  i18n OK: ${label}`)
  }
}

const checkEmojis = async (label) => {
  const body = await page.textContent('body')
  // Basic emoji detection (common ranges)
  const emojiRe = /[\u{1F300}-\u{1F9FF}]|\u{2600}-\u{27BF}/u
  if (emojiRe.test(body || '')) {
    console.warn(`  WARN ${label}: possible emoji in UI text`)
  }
}

console.log('\n=== Einkauf QA ===\n')

// ============================================================================
// 01 Initial load — Bestellungen tab
// ============================================================================
await page.goto(`${BASE}/#/einkauf`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500) // MSW + React Query settle
await shot('01_initial_load')

// Check Bestellungen table loaded
try {
  await page.waitForSelector('table tbody tr', { timeout: 6000 })
  await shot('02_bestellungen_table')
  console.log('  OK: Bestellungen table visible')
} catch {
  console.warn('  WARN: No table rows found on Bestellungen tab')
  await shot('02_bestellungen_empty')
}
await checkRawKeys('Bestellungen')
await checkEmojis('Bestellungen')

// ============================================================================
// 02 Status filter
// ============================================================================
try {
  // Click "Entwurf" status filter
  const buttons = await page.$$('button')
  for (const btn of buttons) {
    const txt = await page.evaluate((el) => el.textContent?.trim(), btn)
    if (txt === 'Entwurf' || txt === 'Draft') {
      await btn.click()
      await page.waitForTimeout(600)
      await shot('03_filter_draft')
      console.log('  OK: Status filter Entwurf applied')
      // Reset
      const allBtns = await page.$$('button')
      for (const b of allBtns) {
        const t = await page.evaluate((el) => el.textContent?.trim(), b)
        if (t === 'Alle' || t === 'All') {
          await b.click()
          await page.waitForTimeout(400)
          break
        }
      }
      break
    }
  }
} catch (e) {
  console.warn('  WARN: Status filter click failed:', e.message)
}

// ============================================================================
// 03 Bestellungen Detail Panel
// ============================================================================
try {
  const firstRow = await page.$('table tbody tr')
  if (firstRow) {
    await firstRow.click()
    await page.waitForTimeout(800)
    await shot('04_bestellung_detail_panel')
    console.log('  OK: Order detail panel opened')

    // Close panel
    const closeBtn = await page.$('[aria-label="Close"], button[class*="close"]')
    if (closeBtn) {
      await closeBtn.click()
      await page.waitForTimeout(400)
    } else {
      await page.keyboard.press('Escape')
      await page.waitForTimeout(400)
    }
  }
} catch (e) {
  console.warn('  WARN: Order detail panel failed:', e.message)
}

// ============================================================================
// 04 Lieferanten tab
// ============================================================================
const liefOk = await clickTabByText('Lieferant')
if (liefOk) {
  await page.waitForTimeout(1500)
  await shot('05_lieferanten_grid')
  console.log('  OK: Lieferanten tab loaded')

  // Check grid cards
  const cards = await page.$$('.rounded-lg.border.border-border.bg-card')
  console.log(`  INFO: ${cards.length} supplier cards visible`)

  await checkRawKeys('Lieferanten')
  await checkEmojis('Lieferanten')

  // Open supplier detail
  if (cards.length > 0) {
    await cards[0].click()
    await page.waitForTimeout(800)
    await shot('06_lieferant_detail_panel')
    console.log('  OK: Supplier detail panel opened')

    // Check for ratings section
    const body = await page.textContent('body')
    if (body?.includes('Bewertung') || body?.includes('Rating') || body?.includes('Durchschnitt')) {
      await shot('07_lieferant_ratings')
      console.log('  OK: Supplier ratings section visible')
    } else {
      console.warn('  WARN: No ratings section found in supplier detail')
    }

    // Close
    await page.keyboard.press('Escape')
    await page.waitForTimeout(400)
  }
} else {
  console.warn('  WARN: Lieferanten tab not found')
}

// ============================================================================
// 05 Katalog tab
// ============================================================================
const katOk = await clickTabByText('Katalog')
if (katOk) {
  await page.waitForTimeout(1500)
  await shot('08_katalog_grid')
  console.log('  OK: Katalog tab loaded')

  const cards = await page.$$('.rounded-lg.border.border-border.bg-card')
  console.log(`  INFO: ${cards.length} catalog cards visible`)

  await checkRawKeys('Katalog')
  await checkEmojis('Katalog')

  // Category filter
  try {
    const selects = await page.$$('select')
    if (selects.length > 0) {
      // Pick second option
      await selects[0].selectOption({ index: 1 })
      await page.waitForTimeout(600)
      await shot('09_katalog_category_filter')
      console.log('  OK: Katalog category filter applied')
      // Reset
      await selects[0].selectOption({ index: 0 })
      await page.waitForTimeout(400)
    }
  } catch (e) {
    console.warn('  WARN: Katalog category filter failed:', e.message)
  }

  // Search
  try {
    const searchInput = await page.$('input[placeholder*="Artikel"], input[placeholder*="suchen"]')
    if (searchInput) {
      await searchInput.type('Sicherung')
      await page.waitForTimeout(600)
      await shot('10_katalog_search')
      console.log('  OK: Katalog search working')
      await searchInput.fill('')
      await page.waitForTimeout(400)
    }
  } catch (e) {
    console.warn('  WARN: Katalog search failed:', e.message)
  }
} else {
  console.warn('  WARN: Katalog tab not found')
}

// ============================================================================
// 06 Rahmenverträge tab
// ============================================================================
const rahmenOk = await clickTabByText('Rahmen')
if (rahmenOk) {
  await page.waitForTimeout(1500)
  await shot('11_rahmenvertraege_list')
  console.log('  OK: Rahmenverträge tab loaded')

  await checkRawKeys('Rahmenverträge')
  await checkEmojis('Rahmenverträge')

  // Expand a contract
  try {
    const contractHeaders = await page.$$('button.flex.w-full')
    if (contractHeaders.length > 0) {
      await contractHeaders[0].click()
      await page.waitForTimeout(700)
      await shot('12_rahmenvertrag_expanded')
      console.log('  OK: Rahmenvertrag expanded with items table')

      // Check progress bar and usage
      const usageBar = await page.$('.h-2.rounded-full.bg-secondary')
      if (usageBar) {
        console.log('  OK: Usage progress bar visible')
      } else {
        console.warn('  WARN: No usage progress bar found')
      }
    } else {
      console.warn('  WARN: No contract headers found to expand')
    }
  } catch (e) {
    console.warn('  WARN: Contract expand failed:', e.message)
  }
} else {
  console.warn('  WARN: Rahmenverträge tab not found')
}

// ============================================================================
// 07 New Order Dialog
// ============================================================================
const newOrderBtn = await page.$('button:has-text("Neue Bestellung"), button:has-text("New Order")')
if (newOrderBtn) {
  // First navigate back to Bestellungen
  await clickTabByText('Bestellung')
  await page.waitForTimeout(500)

  const btn = await page.$('button:has-text("Neue Bestellung"), button:has-text("New Order")')
  if (btn) {
    await btn.click()
    await page.waitForTimeout(600)
    await shot('13_new_order_dialog')
    console.log('  OK: New Order dialog opened')

    // Check supplier dropdown populated
    const select = await page.$('select')
    if (select) {
      const options = await page.$$eval('select option', opts => opts.map(o => o.textContent?.trim()))
      if (options.length > 1) {
        console.log(`  OK: Supplier dropdown has ${options.length - 1} options`)
      } else {
        console.warn('  WARN: Supplier dropdown has no options (suppliers not loaded?)')
      }
    }

    await page.keyboard.press('Escape')
    await page.waitForTimeout(400)
  }
}

// ============================================================================
// 08 New Supplier Dialog
// ============================================================================
await clickTabByText('Lieferant')
await page.waitForTimeout(500)

try {
  const addSupBtn = await page.$('button:has-text("Lieferant anlegen"), button:has-text("Add Supplier")')
  if (addSupBtn) {
    await addSupBtn.click()
    await page.waitForTimeout(600)
    await shot('14_new_supplier_dialog')
    console.log('  OK: New Supplier dialog opened')
    await page.keyboard.press('Escape')
    await page.waitForTimeout(400)
  } else {
    console.warn('  WARN: "Lieferant anlegen" button not found')
  }
} catch (e) {
  console.warn('  WARN: New Supplier dialog failed:', e.message)
}

// ============================================================================
// 09 Check for console errors
// ============================================================================
await page.waitForTimeout(500)
const jsErrors = errors.filter(e =>
  !e.includes('ResizeObserver') &&
  !e.includes('Non-Error')
)
if (jsErrors.length > 0) {
  console.warn(`\n  JS ERRORS (${jsErrors.length}):`)
  jsErrors.forEach(e => console.warn(`    ${e.slice(0, 200)}`))
} else {
  console.log('\n  No JS errors')
}

// ============================================================================
// Done
// ============================================================================
await browser.close()

console.log('\n=== Einkauf QA done ===')
console.log(`Screenshots: ${outDir}`)
console.log('\nWICHTIG: Screenshots ansehen auf:')
console.log('  - Raw i18n keys (einkauf.xxx.yyy format in UI)')
console.log('  - Leere Zustaende vs. gefuellte Zustaende')
console.log('  - Loading-Spinners vs. Daten')
console.log('  - Numerische Werte lesbar (nicht undefined/NaN)')
console.log('  - Keine Emojis in UI-Text')
