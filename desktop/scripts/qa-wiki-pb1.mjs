// QA — wiki Phase B PB-1: block-document engine switch (dev :5173)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/wiki-pb1')
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
      rawKeys: [...new Set(all.filter((t) => /^(wiki|common|shared|document)\.[a-zA-Z]/.test(t)))].slice(0, 15),
      doubleBrace: [...new Set(all.filter((t) => /\{\{|\}\}/.test(t)))].slice(0, 10),
      icuLeak: [...new Set(all.filter((t) => /\{count|plural,|\{min|\{x\}|\{y\}/.test(t)))].slice(0, 10),
      replacementChar: [...new Set(all.filter((t) => /�/.test(t)))].slice(0, 10),
    }
  })

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1500, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
page.setDefaultTimeout(8000)
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const bodyText = () => page.evaluate(() => document.body.textContent || '')
const step = async (name, fn) => {
  try {
    await fn()
  } catch (err) {
    out[`ERR_${name}`] = String(err).split('\n')[0]
  }
}

await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500)
await page.screenshot({ path: resolve(outDir, 'pb1-0-list.png'), fullPage: false })

// ── Reader: showcase article (image + callouts + columns + divider) ──
await step('readerShowcase', async () => {
  await page.getByText('Angebots- und Rechnungsprozess', { exact: true }).first().click()
  await page.waitForTimeout(1400)
  const body = await bodyText()
  out.reader_showsHeadings = /Angebot erstellen/.test(body) && /Auftrag & Rechnung/.test(body)
  out.reader_showsCalloutTitles = /Rabatt-Freigabe/.test(body) && /Vor dem Export/.test(body)
  out.reader_hasImage = await page.locator('figure img, img[src^="data:image"]').count()
  out.reader_hasHr = await page.locator('hr').count()
  await page.screenshot({ path: resolve(outDir, 'pb1-1-reader-showcase.png'), fullPage: true })
})

// ── Reader: columns demo (two callouts side by side in welcome) ──
await step('readerColumns', async () => {
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(1000)
  const body = await bodyText()
  out.reader_welcomeCallouts = /Für Leser schreiben/.test(body) && /Statt kopieren: verlinken/.test(body)
  await page.screenshot({ path: resolve(outDir, 'pb1-2-reader-columns.png'), fullPage: true })
})

// ── Editor: enter edit mode via the header pencil ──
await step('editor', async () => {
  await page.locator('button[title="Bearbeiten"]').first().click()
  await page.waitForTimeout(1200)
  const body = await bodyText()
  out.editor_addBlockButton = /Block einfügen/.test(body)
  out.editor_blockCount = await page.locator('.group\\/block').count()
  await page.screenshot({ path: resolve(outDir, 'pb1-3-editor.png'), fullPage: true })
})

// ── Editor: open the insert menu ──
await step('insertMenu', async () => {
  await page.getByText('Block einfügen', { exact: true }).first().click()
  await page.waitForTimeout(500)
  const body = await bodyText()
  out.insertMenu_hasHeading = /Überschrift/.test(body)
  out.insertMenu_hasCallout = /Hinweis/.test(body)
  out.insertMenu_hasImage = /Bild/.test(body)
  out.insertMenu_hasDivider = /Trenner/.test(body)
  out.insertMenu_hasColumns = /Spalten/.test(body)
  await page.screenshot({ path: resolve(outDir, 'pb1-4-insert-menu.png'), fullPage: false })
})

// ── Empty draft → joy-moment empty canvas, then enter editor from it ──
await step('emptyCanvas', async () => {
  // Full reload to leave the previous article's edit mode (hash nav alone keeps
  // the in-memory React state, so the editor would otherwise stay open).
  await page.reload({ waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2800)
  // Locate the draft via search (it sits low in the list otherwise).
  const search = page.getByPlaceholder(/Such|Search/i).first()
  await search.fill('Reisekosten')
  await page.waitForTimeout(1000)
  await page.getByText('Reisekostenrichtlinie', { exact: false }).first().click()
  await page.waitForTimeout(1000)
  const body = await bodyText()
  out.emptyCanvasShown = /leeres Blatt|Inhalt verfassen/.test(body)
  await page.screenshot({ path: resolve(outDir, 'pb1-5-empty.png'), fullPage: false })
  await page.getByText('Inhalt verfassen', { exact: true }).first().click()
  await page.waitForTimeout(900)
  out.emptyEditor_addBlock = /Block einfügen/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'pb1-5b-empty-editor.png'), fullPage: false })
})

// ── Search still works against block content ──
await step('search', async () => {
  const search = page.getByPlaceholder(/Such|Search/i).first()
  if (await search.count()) {
    await search.fill('Mahnwesen')
    await page.waitForTimeout(1000)
    out.search_findsBlockText = /Angebots|Rechnung/.test(await bodyText())
    await page.screenshot({ path: resolve(outDir, 'pb1-6-search.png'), fullPage: false })
    await search.fill('')
  }
})

await step('scanRaw', async () => Object.assign(out, await scanRaw(page)))
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
