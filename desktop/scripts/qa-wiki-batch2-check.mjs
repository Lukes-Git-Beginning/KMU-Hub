// Independent visual gegencheck for wiki Batch 2 (WP-1..WP-5) on the MAIN clone (:5173).
// Focus: does the editor/reader now match berichte level? Run with dev server on :5173.
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/wiki-batch2-check')
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
const has = async (loc) => (await loc.count().catch(() => 0)) > 0

try {
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  await page.screenshot({ path: resolve(outDir, '0-wiki-home.png') })

  // Reader (WP-4): editorial typography + TOC
  const art = page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first()
  if (await has(art)) { await art.click(); await page.waitForTimeout(1300) }
  out.readSerif = await page.evaluate(() => {
    const h = document.querySelector('.wiki-canvas h2, .wiki-canvas h1')
    return h ? getComputedStyle(h).fontFamily.includes('Playfair') : false
  })
  await page.screenshot({ path: resolve(outDir, '1-reader.png') })

  // Editor (WP-1): frameless canvas + big editorial title
  const editBtn = page.locator('button[title="Bearbeiten"]').first()
  if (await has(editBtn)) { await editBtn.click(); await page.waitForTimeout(1000) }
  out.titleSerif = await page.evaluate(() => {
    const t = document.querySelector('.wiki-title-input')
    return t ? getComputedStyle(t).fontFamily.includes('Playfair') : false
  })
  out.stickyToolbar = await page.locator('.wiki-toolbar-sticky').count()
  await page.screenshot({ path: resolve(outDir, '2-editor.png') })

  // Slash menu (WP-2): type "/" in the canvas
  const canvas = page.locator('.wiki-canvas.tiptap-content, .wiki-canvas [contenteditable="true"]').first()
  if (await has(canvas)) {
    await canvas.click()
    await page.waitForTimeout(300)
    await page.keyboard.press('End')
    await page.keyboard.press('Enter')
    await page.keyboard.type('/')
    await page.waitForTimeout(900)
  }
  out.slashMenuVisible = await page.evaluate(() =>
    /Callout|Code|Trenner|Bild|Umschalt|Toggle|Hinweis|Überschrift/i.test(document.body.innerText),
  )
  await page.screenshot({ path: resolve(outDir, '3-slash-menu.png') })

  Object.assign(out, await scanRaw(page))
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'ERROR.png') }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
