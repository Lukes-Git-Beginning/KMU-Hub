// QA — wiki WP-4: editorial reader (breadcrumbs, reading time, TOC scroll-spy) (dev :5174)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/wiki')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const scanRaw = (page) =>
  page.evaluate(() => {
    const all = Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0)
      .map((n) => (n.textContent || '').trim())
      .filter(Boolean)
    return {
      rawKeys: [...new Set(all.filter((t) => /^(wiki|common|shared)\.[a-zA-Z]/.test(t)))].slice(0, 15),
      doubleBrace: [...new Set(all.filter((t) => /\{\{|\}\}/.test(t)))].slice(0, 10),
      icuLeak: [...new Set(all.filter((t) => /\{count|plural,|\{min/.test(t)))].slice(0, 10),
      replacementChar: [...new Set(all.filter((t) => /�/.test(t)))].slice(0, 10),
    }
  })

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1500, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const bodyText = () => page.evaluate(() => document.body.textContent || '')

try {
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)

  // Open the rich onboarding article (multi-level category + many headings)
  await page.getByText('Onboarding neuer Mitarbeitender', { exact: true }).first().click()
  await page.waitForTimeout(1200)

  const body = await bodyText()
  // ── Breadcrumbs (Allgemein › Onboarding) ──
  out.breadcrumbHasArea = /Allgemein/.test(body)
  out.breadcrumbHasCategory = /Onboarding/.test(body)
  // ── Reading time ──
  out.readingTimeShown = /Min\. Lesezeit/.test(body)
  out.noMinLeak = !/\{min\}/.test(body)
  // ── TOC ──
  out.tocPresent = await page.evaluate(() => !!document.querySelector('nav[aria-label]'))
  out.tocLinks = await page.locator('nav[aria-label] button').count()
  out.tocTitleShown = /Inhalt/.test(body)
  await page.screenshot({ path: resolve(outDir, 'wp4-1-reader-toc.png'), fullPage: false })

  // ── TOC jump + scroll-spy ──
  const tocButtons = page.locator('nav[aria-label] button')
  const n = await tocButtons.count()
  if (n >= 3) {
    await tocButtons.nth(n - 1).click() // jump to the last section
    await page.waitForTimeout(900)
    out.activeAfterJump = await page.evaluate(() => {
      const active = document.querySelector('nav[aria-label] button.text-primary, nav[aria-label] button.border-primary')
      return active ? active.textContent.trim() : null
    })
  }
  await page.screenshot({ path: resolve(outDir, 'wp4-2-toc-jumped.png'), fullPage: false })

  Object.assign(out, await scanRaw(page))

  // ── Narrow viewport: TOC hides, content stays readable ──
  await page.setViewportSize({ width: 900, height: 900 })
  await page.waitForTimeout(500)
  out.tocHiddenNarrow = await page.evaluate(() => {
    const nav = document.querySelector('nav[aria-label]')
    if (!nav) return true
    return getComputedStyle(nav.closest('aside') || nav).display === 'none'
  })
  await page.screenshot({ path: resolve(outDir, 'wp4-3-narrow.png'), fullPage: false })
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'wp4-error.png'), fullPage: false }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
