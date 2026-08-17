/**
 * QA — does the production web bundle actually run in a browser?
 *
 * Deliberately does NOT stub window.electronAPI. Every other qa-*.mjs script
 * injects a no-op Proxy for it, which is exactly what would hide the failures
 * this check exists to find: before the platform.ts work, an unguarded
 * window.electronAPI.auth call threw a TypeError on startup and on login.
 *
 * Usage:
 *   npm run build:web
 *   npx vite preview --config vite.web.config.mts --port 4173 &
 *   node scripts/qa-web-build.mjs [tag]
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir, writeFile } from 'node:fs/promises'

const BASE = process.env.QA_WEB_BASE || 'http://localhost:4173'
const TAG = process.argv[2] || 'baseline'
const outDir = resolve('.qa-screenshots', `web-${TAG}`)
await mkdir(outDir, { recursive: true })

const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1480, height: 1000 } })
const page = await ctx.newPage()

const consoleErrors = []
const pageErrors = []
page.on('console', (m) => {
  if (m.type() === 'error') consoleErrors.push(m.text())
})
page.on('pageerror', (e) => pageErrors.push(String(e)))

await page.goto(BASE, { waitUntil: 'networkidle' })
// The skeleton in index.html is static markup; give React a moment to replace it.
await page.waitForTimeout(2500)
await page.screenshot({ path: resolve(outDir, '00-intro.png'), fullPage: true })

// AuthLayout plays the CosmiLaunch intro over the login form for ~6.6s
// (CosmiLaunch T_TEXT_END), once per browser session. Playwright's "visible"
// is not enough here -- the field has a box the whole time, it is just covered
// by the animation's SVG. Wait until it is the element actually hit at its own
// centre, which is what a user clicking it would need.
let introCleared = true
const introStart = Date.now()
try {
  await page.waitForFunction(
    () => {
      const el = document.querySelector('input[type="password"]')
      if (!el) return false
      const r = el.getBoundingClientRect()
      if (r.height === 0) return false
      return document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2) === el
    },
    { timeout: 20000 },
  )
} catch {
  introCleared = false
}
const introMs = Date.now() - introStart
await page.waitForTimeout(400)

const probe = await page.evaluate(() => {
  const text = document.body.innerText || ''
  return {
    electronApiPresent: typeof window.electronAPI !== 'undefined',
    skeletonStillVisible: !!document.getElementById('app-skeleton'),
    rootHasChildren: (document.getElementById('root')?.children.length ?? 0) > 0,
    hasPasswordField: !!document.querySelector('input[type="password"]'),
    passwordFieldVisible: (() => {
      const el = document.querySelector('input[type="password"]')
      if (!el) return false
      const r = el.getBoundingClientRect()
      return r.width > 0 && r.height > 0
    })(),
    hasEmailField: !!document.querySelector('input[type="email"]'),
    // A blank page renders as a couple of characters at most.
    visibleTextLength: text.trim().length,
    excerpt: text.trim().slice(0, 300),
    // Raw i18n keys would show as e.g. "auth.login.title" in the UI.
    rawKeys: Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0)
      .map((n) => (n.textContent || '').trim())
      .filter((t) => /^[a-z][a-zA-Z]*\.[a-z][a-zA-Z]+\./.test(t))
      .slice(0, 10),
  }
})

await page.screenshot({ path: resolve(outDir, '01-startup.png'), fullPage: true })

// The failure mode we are hunting: an undefined-property read on the bridge.
const bridgeErrors = [...consoleErrors, ...pageErrors].filter((e) =>
  /electronAPI|Cannot read propert/i.test(e),
)

const report = {
  base: BASE,
  /** How long the intro covered the login form, in ms. Desktop pays this once
   *  per app start; on the web a first-time visitor pays it before they can
   *  type. Worth watching, not currently a gate. */
  introBlockedMs: introMs,
  probe,
  consoleErrors: consoleErrors.slice(0, 20),
  pageErrors: pageErrors.slice(0, 20),
  bridgeErrors,
  verdict: {
    appMounted: probe.rootHasChildren && !probe.skeletonStillVisible,
    // Visibility, not mere presence: the intro animation covers the form for a
    // few seconds, and a DOM-only check would pass while the user sees a splash.
    loginVisible: introCleared && probe.passwordFieldVisible,
    noBridgeCrash: bridgeErrors.length === 0,
    noRawKeys: probe.rawKeys.length === 0,
  },
}

await writeFile(resolve(outDir, 'report.json'), JSON.stringify(report, null, 2))
console.log(JSON.stringify(report, null, 2))

await browser.close()

const failed = Object.entries(report.verdict).filter(([, ok]) => !ok)
if (failed.length) {
  console.error('FAILED:', failed.map(([k]) => k).join(', '))
  process.exit(1)
}
console.log('OK — web bundle runs without the Electron bridge')
