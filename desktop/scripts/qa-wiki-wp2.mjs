// QA — wiki WP-2: slash menu + rich blocks (callout/code/toggle/figure) (dev :5174)
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
      icuLeak: [...new Set(all.filter((t) => /\{count|plural,/.test(t)))].slice(0, 10),
      replacementChar: [...new Set(all.filter((t) => /�/.test(t)))].slice(0, 10),
    }
  })

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

try {
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)

  // ── A) Read mode: rich blocks render (callouts/code/toggle/figure) ──
  await page.getByText('Angebots- und Rechnungsprozess', { exact: true }).first().click()
  await page.waitForTimeout(1200)
  out.calloutCount = await page.locator('.wiki-callout').count()
  out.calloutVariants = await page.evaluate(() =>
    Array.from(document.querySelectorAll('.wiki-callout')).map((c) => c.getAttribute('data-variant')),
  )
  out.detailsCount = await page.locator('details.wiki-details').count()
  out.figureCount = await page.locator('figure.wiki-figure').count()
  out.codeHljsSpans = await page.locator('pre code span').count()
  out.calloutIconRendered = await page.evaluate(() => {
    const c = document.querySelector('.wiki-callout')
    if (!c) return false
    const bg = getComputedStyle(c, '::before').maskImage || getComputedStyle(c, '::before').webkitMaskImage
    return !!bg && bg !== 'none'
  })
  await page.screenshot({ path: resolve(outDir, 'wp2-1-read-blocks.png'), fullPage: true })

  // ── B) Edit mode: open the slash menu ──
  await page.locator('button[title="Bearbeiten"]').first().click()
  await page.waitForTimeout(1000)
  // place cursor at end, add a fresh line, then trigger the slash menu
  await page.locator('.wiki-canvas.tiptap-content').first().click()
  await page.keyboard.press('Control+End')
  await page.keyboard.press('Enter')
  await page.keyboard.type('/')
  await page.waitForTimeout(700)
  out.slashMenuOpen = await page.evaluate(() =>
    /Block einfügen/.test(document.body.textContent || ''),
  )
  out.slashItemCount = await page.evaluate(() => {
    const menus = Array.from(document.querySelectorAll('div')).filter((d) =>
      /Block einfügen/.test(d.textContent || ''),
    )
    return menus.length ? menus[menus.length - 1].querySelectorAll('button').length : 0
  })
  await page.screenshot({ path: resolve(outDir, 'wp2-2-slash-menu.png'), fullPage: false })

  // ── C) Filter the slash menu by typing "code" ──
  await page.keyboard.type('code')
  await page.waitForTimeout(500)
  await page.screenshot({ path: resolve(outDir, 'wp2-3-slash-filtered.png'), fullPage: false })

  // ── D) Insert the code block via Enter ──
  await page.keyboard.press('Enter')
  await page.waitForTimeout(600)
  out.codeBlockInserted = await page.locator('.wiki-canvas pre').count()
  await page.screenshot({ path: resolve(outDir, 'wp2-4-inserted-code.png'), fullPage: false })

  Object.assign(out, await scanRaw(page))

  // ── E) Narrow viewport read mode ──
  await page.getByRole('button', { name: 'Abbrechen' }).first().click().catch(() => {})
  await page.waitForTimeout(500)
  await page.setViewportSize({ width: 880, height: 1000 })
  await page.waitForTimeout(500)
  await page.screenshot({ path: resolve(outDir, 'wp2-5-narrow-read.png'), fullPage: true })
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'wp2-error.png'), fullPage: false }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
