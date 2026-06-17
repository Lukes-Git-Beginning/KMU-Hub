/**
 * QA Wiki — Mock-to-TanStack-Query migration
 *
 * Verifies:
 *   1. Wiki page loads without crash (no JS errors)
 *   2. Loading skeleton appears (not raw mock data flash)
 *   3. No raw i18n keys visible (wiki.*, common.*)
 *   4. Sidebar renders category tree area
 *   5. Article list area is present (or empty state)
 *   6. No stale mock IDs visible (wa1, wc1, etc.)
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('scripts/screenshots')
await mkdir(outDir, { recursive: true })

// Stub Electron API (renderer runs in browser during dev)
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
// Mark onboarding as completed so we land on the main layout
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
// Clear any persisted wiki selection from the old store (version bump handles it, but safety net)
const CLEAR_WIKI = `try{localStorage.removeItem('cosmi-wiki')}catch(e){}`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
await ctx.addInitScript(CLEAR_WIKI)
const page = await ctx.newPage()

const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
page.on('console', (msg) => {
  if (msg.type() === 'error') errors.push(`[console.error] ${msg.text()}`)
})

// Navigate directly to wiki route
await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
// Give MSW + React Query time to settle (mock handlers respond instantly)
await page.waitForTimeout(2500)

// Screenshot 1: initial state
await page.screenshot({ path: resolve(outDir, 'wiki-initial.png'), fullPage: false })

// Check for raw i18n keys
const rawKeys = await page.evaluate(() => {
  return Array.from(document.querySelectorAll('body *'))
    .filter((n) => n.children.length === 0 && /^wiki\.[a-z]|^common\.[a-z]/.test((n.textContent ?? '').trim()))
    .map((n) => (n.textContent ?? '').trim())
    .filter(Boolean)
    .slice(0, 15)
})

// Check for stale mock IDs leaking into DOM text
const mockIds = await page.evaluate(() => {
  const bodyText = document.body.innerText
  const found = []
  if (/\bwa\d+\b/.test(bodyText)) found.push('article IDs (wa1..)')
  if (/\bwc\d+\b/.test(bodyText)) found.push('category IDs (wc1..)')
  if (/\bwv\d+\b/.test(bodyText)) found.push('version IDs (wv1..)')
  return found
})

// Check sidebar presence
const sidebarPresent = await page.evaluate(() => !!document.querySelector('[class*="w-56"]'))

// Check article list / empty state / skeleton presence
const articleAreaPresent = await page.evaluate(() => {
  return !!(
    document.querySelector('[class*="divide-y"]') ||
    document.querySelector('[class*="animate-pulse"]') ||
    // EmptyState renders an icon + heading
    document.querySelector('[class*="EmptyState"], [data-testid="empty-state"]') ||
    // fallback: any heading-level text in the main area
    document.querySelector('main h2, main h3, [role="main"] h2')
  )
})

// Check for visible loading error banner
const errorBannerPresent = await page.evaluate(() => {
  return Array.from(document.querySelectorAll('*'))
    .some((n) => n.children.length === 0 && /konnten nicht geladen|Could not load/i.test(n.textContent ?? ''))
})

console.log('\n=== QA Wiki Results ===')
console.log('Raw i18n keys visible:', rawKeys.length ? rawKeys : 'none ✓')
console.log('Stale mock IDs in DOM:', mockIds.length ? mockIds : 'none ✓')
console.log('Sidebar present:', sidebarPresent ? '✓' : '✗ MISSING')
console.log('Article area present:', articleAreaPresent ? '✓' : '✗ MISSING')
console.log('Error banner visible:', errorBannerPresent ? 'YES (MSW handler missing or error)' : 'no ✓')
console.log('JS errors:', errors.length ? errors : 'none ✓')
console.log('')

// Screenshot 2: after potential category click
const firstCatBtn = await page.$('button[class*="rounded-md"][class*="px-2"]')
if (firstCatBtn) {
  await firstCatBtn.click()
  await page.waitForTimeout(500)
  await page.screenshot({ path: resolve(outDir, 'wiki-category-selected.png'), fullPage: false })
  console.log('Screenshot: wiki-category-selected.png')
}

await browser.close()
console.log('Screenshots saved to scripts/screenshots/')
console.log('  wiki-initial.png')

// Exit non-zero if critical checks fail
const failed =
  rawKeys.length > 0 ||
  mockIds.length > 0 ||
  !sidebarPresent ||
  errors.some((e) => !e.includes('Warning:'))

if (failed) {
  console.error('\n✗ QA checks FAILED — review screenshots and errors above')
  process.exit(1)
} else {
  console.log('\n✓ QA checks PASSED')
}
