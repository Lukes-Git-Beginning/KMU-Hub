// QA — wiki WP-1: focused editorial editor shell + look (dev :5174)
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
const bodyText = () => page.evaluate(() => document.body.textContent || '')

try {
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)

  // ── A) Read mode: editorial typography (Playfair headings, 65ch) ──
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(1200)
  out.readHasSerifHeading = await page.evaluate(() => {
    const h = document.querySelector('.wiki-canvas h2')
    if (!h) return false
    return getComputedStyle(h).fontFamily.includes('Playfair')
  })
  out.readMeasure = await page.evaluate(() => {
    const c = document.querySelector('.wiki-canvas')
    return c ? Math.round(c.getBoundingClientRect().width) : -1
  })
  await page.screenshot({ path: resolve(outDir, 'wp1-1-read.png'), fullPage: false })

  // ── B) Edit mode: frameless canvas, big editorial title, sticky toolbar ──
  await page.locator('button[title="Bearbeiten"]').first().click()
  await page.waitForTimeout(1000)
  out.titleInputPresent = await page.locator('.wiki-title-input').count()
  out.titleValue = await page.locator('.wiki-title-input').first().inputValue().catch(() => null)
  out.titleFontSerif = await page.evaluate(() => {
    const t = document.querySelector('.wiki-title-input')
    return t ? getComputedStyle(t).fontFamily.includes('Playfair') : false
  })
  out.stickyToolbar = await page.locator('.wiki-toolbar-sticky').count()
  // The frameless canvas: editor content should not sit inside a bordered card.
  out.editorMinHeight = await page.evaluate(() => {
    const e = document.querySelector('.wiki-canvas.tiptap-content')
    return e ? Math.round(e.getBoundingClientRect().height) : -1
  })
  await page.screenshot({ path: resolve(outDir, 'wp1-2-edit.png'), fullPage: false })

  // ── C) Type a heading in the editor to confirm serif rendering ──
  await page.locator('.wiki-canvas.tiptap-content').first().click()
  await page.waitForTimeout(300)
  await page.screenshot({ path: resolve(outDir, 'wp1-3-edit-focus.png'), fullPage: false })
  Object.assign(out, await scanRaw(page))

  // ── D) Narrow viewport ──
  await page.setViewportSize({ width: 860, height: 900 })
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, 'wp1-4-narrow.png'), fullPage: false })
  out.narrow = await scanRaw(page)

  // ── E) Cancel back to read mode ──
  await page.getByRole('button', { name: 'Abbrechen' }).first().click().catch(() => {})
  await page.waitForTimeout(600)
  await page.setViewportSize({ width: 1440, height: 900 })
  await page.waitForTimeout(400)
  await page.screenshot({ path: resolve(outDir, 'wp1-5-read-again.png'), fullPage: false })
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'wp1-error.png'), fullPage: false }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
