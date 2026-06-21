// QA — wiki PB-4: link/mention preview popover + jump (dev :5173)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/wiki-pb4')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1500, height: 940 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
page.setDefaultTimeout(8000)
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const bodyText = () => page.evaluate(() => document.body.textContent || '')
const step = async (name, fn) => {
  try { await fn() } catch (err) { out[`ERR_${name}`] = String(err).split('\n')[0] }
}

await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500)

await step('openWelcome', async () => {
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(1000)
  out.hasWikiLink = await page.locator('[data-wiki-link]').count()
  out.hasMention = await page.locator('[data-mention-id]').count()
  await page.screenshot({ path: resolve(outDir, 'pb4-0-article.png'), fullPage: false })
})

await step('clickWikiLink', async () => {
  await page.locator('[data-wiki-link]').first().click()
  await page.waitForTimeout(500)
  const body = await bodyText()
  out.popoverShown = await page.locator('div.fixed.shadow-lg, div[style*="position: fixed"]').count()
  out.popoverHasTarget = /Onboarding neuer Mitarbeitender/.test(body)
  out.popoverHasAction = /Artikel öffnen/.test(body)
  await page.screenshot({ path: resolve(outDir, 'pb4-1-link-popover.png'), fullPage: false })
})

await step('jumpToArticle', async () => {
  await page.getByText('Artikel öffnen', { exact: true }).first().click()
  await page.waitForTimeout(1000)
  const body = await bodyText()
  // wart-002 content should now be in view
  out.jumpedToTarget = /Vor dem ersten Tag/.test(body)
  await page.screenshot({ path: resolve(outDir, 'pb4-2-jumped.png'), fullPage: false })
})

await step('clickMention', async () => {
  // Back to the welcome article, then click the @mention.
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(900)
  await page.locator('[data-mention-id]').first().click()
  await page.waitForTimeout(500)
  const body = await bodyText()
  out.mentionPopover = /Teammitglied/.test(body)
  out.mentionAction = /Im Team öffnen/.test(body)
  await page.screenshot({ path: resolve(outDir, 'pb4-3-mention-popover.png'), fullPage: false })
})

// Editing a block that contains a link must preserve the node (PB-4 extensions).
await step('editPreserves', async () => {
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(900)
  await page.locator('button[title="Bearbeiten"]').first().click()
  await page.waitForTimeout(1200)
  out.editorKeepsLink = await page.locator('.tiptap-content .wiki-link').count()
  out.editorKeepsMention = await page.locator('.tiptap-content .wiki-mention').count()
  await page.screenshot({ path: resolve(outDir, 'pb4-4-edit-preserves.png'), fullPage: false })
})

out.pageErrors = errs.slice(0, 8)
out.rawKeys = await page.evaluate(() => {
  const all = Array.from(document.querySelectorAll('body *'))
    .filter((n) => n.children.length === 0)
    .map((n) => (n.textContent || '').trim())
  return [...new Set(all.filter((t) => /^(wiki|common|shared|document)\.[a-zA-Z]/.test(t)))].slice(0, 12)
})
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
