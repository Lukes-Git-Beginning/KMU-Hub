// QA — wiki W-1: TipTap editor + version restore (dev :5174)
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
      rawKeys: [...new Set(all.filter((t) => /^(wiki|common|shared|moduleSettings|settings)\.[a-zA-Z]/.test(t)))].slice(0, 15),
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

try {
  // ── A) Wiki overview ──
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  await page.screenshot({ path: resolve(outDir, 'w1-1-overview.png'), fullPage: false })
  Object.assign(out, await scanRaw(page))

  // ── B) Open an article (read view) ──
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(1200)
  const body1 = await page.evaluate(() => document.body.textContent || '')
  out.readShowsContent = /zentrale Wissensbasis/.test(body1)
  await page.screenshot({ path: resolve(outDir, 'w1-2-read.png'), fullPage: false })

  // ── C) Edit mode → TipTap editor ──
  await page.getByRole('button', { name: 'Bearbeiten' }).first().click()
  await page.waitForTimeout(1200)
  out.editorPresent = await page.locator('.tiptap-content, .ProseMirror').count()
  out.toolbarBold = await page.getByRole('button', { name: 'Fett' }).count()
  await page.screenshot({ path: resolve(outDir, 'w1-3-editor.png'), fullPage: false })
  // cancel edit
  await page.getByRole('button', { name: 'Abbrechen' }).first().click()
  await page.waitForTimeout(700)

  // ── D) Version history + restore button ──
  await page.getByRole('button', { name: 'Versionshistorie' }).first().click()
  await page.waitForTimeout(1000)
  const body2 = await page.evaluate(() => document.body.textContent || '')
  out.versionsPanel = /Versionen/.test(body2)
  out.hasRestoreBtn = await page.getByRole('button', { name: 'Wiederherstellen' }).count()
  await page.screenshot({ path: resolve(outDir, 'w1-4-versions.png'), fullPage: false })

  // perform a restore (older version) and confirm content changes
  if (out.hasRestoreBtn > 0) {
    await page.getByRole('button', { name: 'Wiederherstellen' }).last().click()
    await page.waitForTimeout(1200)
    const body3 = await page.evaluate(() => document.body.textContent || '')
    out.restoredChangedContent = /Erste Version|Abteilungsstruktur/.test(body3)
    await page.screenshot({ path: resolve(outDir, 'w1-5-after-restore.png'), fullPage: false })
  }

  // ── E) Narrow width ──
  await page.setViewportSize({ width: 760, height: 900 })
  await page.waitForTimeout(800)
  await page.screenshot({ path: resolve(outDir, 'w1-6-narrow.png'), fullPage: false })
  Object.assign(out, { narrow: await scanRaw(page) })
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
