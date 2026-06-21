// QA — wiki PB-4b: inline [[ / @ autocomplete insertion (dev :5173)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/wiki-pb4b')
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
await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
await page.waitForTimeout(900)
await page.locator('button[title="Bearbeiten"]').first().click()
await page.waitForTimeout(1200)

out.linksBefore = await page.locator('.tiptap-content .wiki-link').count()
out.mentionsBefore = await page.locator('.tiptap-content .wiki-mention').count()

await step('linkSuggest', async () => {
  const editable = page.locator('.tiptap-content').first()
  await editable.click()
  await page.keyboard.press('End')
  await page.keyboard.type(' Siehe ')
  await page.keyboard.type('[[Backup')
  await page.waitForTimeout(700)
  const body = await bodyText()
  out.linkPopupShown = /Artikel verlinken/.test(body)
  out.linkPopupHasMatch = /Backup & Disaster Recovery/.test(body)
  await page.screenshot({ path: resolve(outDir, 'pb4b-1-link-suggest.png'), fullPage: false })
  await page.keyboard.press('Enter')
  await page.waitForTimeout(600)
  out.linksAfter = await page.locator('.tiptap-content .wiki-link').count()
})

await step('mentionSuggest', async () => {
  await page.keyboard.type('@Thom')
  await page.waitForTimeout(700)
  const body = await bodyText()
  out.mentionPopupShown = /Person erwähnen/.test(body)
  await page.screenshot({ path: resolve(outDir, 'pb4b-2-mention-suggest.png'), fullPage: false })
  await page.keyboard.press('Enter')
  await page.waitForTimeout(600)
  out.mentionsAfter = await page.locator('.tiptap-content .wiki-mention').count()
})

await step('escapeCloses', async () => {
  await page.keyboard.type(' [[On')
  await page.waitForTimeout(500)
  out.popupBeforeEsc = /Artikel verlinken/.test(await bodyText())
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)
  out.popupAfterEsc = /Artikel verlinken/.test(await bodyText())
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
