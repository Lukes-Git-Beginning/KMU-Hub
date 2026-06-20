// QA — wiki W-3: @mention + [[wikilink]] autocomplete (dev :5174)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/wiki')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

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

  out.step = 'open-article'
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(800)
  out.step = 'click-edit'
  await page.getByRole('button', { name: 'Bearbeiten' }).first().click({ timeout: 8000 })
  await page.waitForTimeout(1000)

  out.step = 'focus-editor'
  const editor = page.locator('.tiptap-content')
  out.editorCount = await editor.count()
  await editor.click({ timeout: 8000 })
  await page.keyboard.press('End')
  out.step = 'type-mention'

  // ── @mention ──
  await page.keyboard.type(' @Ste', { delay: 60 })
  await page.waitForTimeout(700)
  out.mentionDropdown = /Person erwähnen/.test(await page.evaluate(() => document.body.textContent || ''))
  out.mentionHasStefan = await page.getByText('Stefan Vogel', { exact: false }).count()
  await page.screenshot({ path: resolve(outDir, 'w3-1-mention-dropdown.png'), fullPage: false })
  await page.keyboard.press('Enter')
  await page.waitForTimeout(500)
  out.mentionInserted = await page.locator('.wiki-mention').count()

  // ── [[wikilink]] ──
  await page.keyboard.type(' [[Onboarding', { delay: 50 })
  await page.waitForTimeout(700)
  out.wikilinkDropdown = /Artikel verknüpfen/.test(await page.evaluate(() => document.body.textContent || ''))
  await page.screenshot({ path: resolve(outDir, 'w3-2-wikilink-dropdown.png'), fullPage: false })
  await page.keyboard.press('Enter')
  await page.waitForTimeout(500)
  out.wikilinkInserted = await page.locator('.wiki-link').count()
  await page.screenshot({ path: resolve(outDir, 'w3-3-editor-tokens.png'), fullPage: false })

  // ── Save ──
  await page.getByRole('button', { name: 'Speichern' }).first().click()
  await page.waitForTimeout(1200)
  out.readMention = await page.locator('.wiki-mention').count()
  out.readWikilink = await page.locator('.wiki-link').count()
  await page.screenshot({ path: resolve(outDir, 'w3-4-read-tokens.png'), fullPage: false })

  // ── Click the wikilink → opens the target article ──
  const titleBefore = await page.locator('h2.text-lg').first().innerText().catch(() => '')
  await page.locator('.wiki-link').first().click()
  await page.waitForTimeout(900)
  const titleAfter = await page.locator('h2.text-lg').first().innerText().catch(() => '')
  out.titleBefore = titleBefore
  out.titleAfter = titleAfter
  out.wikilinkNavigated = /Onboarding/.test(titleAfter)
  await page.screenshot({ path: resolve(outDir, 'w3-5-navigated.png'), fullPage: false })

  out.rawKeys = await page.evaluate(() => {
    const all = Array.from(document.querySelectorAll('body *')).filter((n) => n.children.length === 0).map((n) => (n.textContent || '').trim())
    return [...new Set(all.filter((t) => /^(wiki|common|shared)\.[a-zA-Z]/.test(t)))].slice(0, 12)
  })
} catch (err) {
  out.error = String(err).split('\n')[0]
  out.step = out.step || 'unknown'
  await page.screenshot({ path: resolve(outDir, 'w3-error.png'), fullPage: false }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
