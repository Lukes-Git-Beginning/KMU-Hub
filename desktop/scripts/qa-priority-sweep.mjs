// QA Welle 5: task priority medium->normal against the REAL backend (localbackend, :5173).
// Verifies the work module renders with the normalized priority: PriorityBadges show
// "Normal" (not a raw key, never "medium"), the priority filter pill labeled "Normal"
// (value 'normal') filters tasks, and no JS errors / ErrorBoundary crash.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = 'http://localhost:5173'
const GW = 'http://localhost:8080'
const outDir = resolve('.qa-screenshots/priority-sweep-w5')
await mkdir(outDir, { recursive: true })

const loginRes = await fetch(`${GW}/api/v1/auth/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email: 'demo@local.test', password: 'DemoPass123!' }),
})
const loginJson = await loginRes.json()
const ACCESS = loginJson.access_token
const REFRESH = loginJson.refresh_token
console.log('[auth] login status', loginRes.status, 'token?', !!ACCESS)

const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const TOKENS = { accessToken: ${JSON.stringify(ACCESS)}, refreshToken: ${JSON.stringify(REFRESH)} }
  const authApi = { getStoredTokens: async () => TOKENS, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const SUPPRESS_ONBOARDING = `
  try {
    const KEY = 'cosmi-ui'
    const raw = localStorage.getItem(KEY)
    const parsed = raw ? JSON.parse(raw) : { state: {}, version: 0 }
    parsed.state = { ...(parsed.state || {}), onboardingCompleted: true }
    localStorage.setItem(KEY, JSON.stringify(parsed))
  } catch (e) {}
`

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS_ONBOARDING)
await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}`)

const page = await ctx.newPage()
page.setDefaultTimeout(15000)
const errors = []
page.on('pageerror', (e) => errors.push('PAGEERR: ' + String(e).slice(0, 160)))
page.on('console', (m) => { if (m.type() === 'error') errors.push('CONSOLE: ' + m.text().slice(0, 160)) })
const shot = (n) => page.screenshot({ path: resolve(outDir, `${n}.png`), fullPage: false }).catch(() => {})

const scan = async (label) => {
  const r = await page.evaluate(() => {
    const txt = document.body.innerText
    // compact PriorityBadges render the label only in the title attribute
    const titles = Array.from(document.querySelectorAll('[title]')).map((e) => e.getAttribute('title'))
    const titleJoin = titles.join(' | ')
    const both = txt + ' ' + titleJoin
    return {
      rawKeys: (both.match(/\bwork\.[a-z][a-zA-Z.]+/g) || []).slice(0, 8),
      mediumLeak: /\bmedium\b/i.test(both),
      crash: /Something went wrong|Etwas ist schief|ErrorBoundary/i.test(txt),
      titleLabels: titles.filter((tt) => /Dringend|Hoch|Normal|Niedrig/.test(tt || '')).slice(0, 12),
      labels: {
        Dringend: (both.match(/Dringend/g) || []).length,
        Hoch: (both.match(/Hoch/g) || []).length,
        Normal: (both.match(/\bNormal\b/g) || []).length,
        Niedrig: (both.match(/Niedrig/g) || []).length,
      },
    }
  })
  console.log(`[${label}]`, JSON.stringify(r))
  return r
}

try {
  await page.goto(`${FE}/`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  const email = page.locator('input[type=email]')
  if (await email.count().catch(() => 0)) {
    try {
      await email.waitFor({ state: 'visible', timeout: 5000 })
      await email.fill('demo@local.test')
      await page.locator('input[type=password]').fill('DemoPass123!')
      await page.locator('input[type=password]').press('Enter')
      await page.waitForTimeout(3500)
    } catch { /* already authed */ }
  }
  await shot('00-after-login')

  // --- Work / My Tasks ---
  await page.goto(`${FE}/#/work/my-tasks`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  await shot('10-mytasks-all')
  await scan('mytasks-all')

  // Click the "Normal" priority filter pill and capture the filtered set.
  const normalPill = page.getByRole('button', { name: /^Normal$/ }).first()
  const pillCount = await normalPill.count().catch(() => 0)
  console.log('[filter] Normal pill present:', pillCount > 0)
  if (pillCount > 0) {
    await normalPill.click()
    await page.waitForTimeout(2500)
    await shot('11-mytasks-normal-filter')
    await scan('mytasks-normal-filter')
  }

  // --- A project's task views (richer task set, PriorityBadges + filter pills) ---
  await page.goto(`${FE}/#/work/projects`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  await shot('20-projects')
  // open the "Website Relaunch" project (11 tasks)
  const projCard = page.getByText('Website Relaunch').first()
  if (await projCard.count().catch(() => 0)) {
    await projCard.click().catch(() => {})
    await page.waitForTimeout(3500)
    await shot('21-project-board')
    await scan('project-board')
    // switch to the list view if a tab/control exists
    const listTab = page.getByRole('tab', { name: /Liste|List/ }).or(page.getByRole('button', { name: /^Liste$/ })).first()
    if (await listTab.count().catch(() => 0)) {
      await listTab.click().catch(() => {})
      await page.waitForTimeout(2000)
      await shot('22-project-list')
      await scan('project-list')
    }
    // open the priority filter (pill bar or dropdown) and pick "Normal"
    const prioFilter = page.getByRole('button', { name: /Priorit/ }).first()
    if (await prioFilter.count().catch(() => 0)) {
      await prioFilter.click().catch(() => {})
      await page.waitForTimeout(800)
    }
    const normalOpt = page.getByRole('button', { name: /^Normal$/ }).or(page.getByText(/^Normal$/)).first()
    if (await normalOpt.count().catch(() => 0)) {
      await normalOpt.click().catch(() => {})
      await page.waitForTimeout(2000)
      await shot('23-project-normal-filter')
      await scan('project-normal-filter')
    }
  }

  console.log('\nJS errors:', errors.length ? errors : 'none')
} catch (e) {
  console.error('SCRIPT ERROR:', String(e).slice(0, 300))
  await shot('99-error')
} finally {
  await browser.close()
}
console.log('Screenshots:', outDir)
