// QA — wiki PB-5: post-cleanup health, version diff (block→html), edit round-trip
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/wiki-pb5')
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

// Health: list renders after deleting the dead extensions.
out.listRenders = /Willkommen im Cosmi-Wiki/.test(await bodyText())
await page.screenshot({ path: resolve(outDir, 'pb5-0-list.png'), fullPage: false })

// Edit round-trip: type into a block, save, confirm it persisted.
await step('roundTrip', async () => {
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(900)
  await page.locator('button[title="Bearbeiten"]').first().click()
  await page.waitForTimeout(1200)
  const editable = page.locator('.tiptap-content').first()
  await editable.click()
  await page.keyboard.press('End')
  await page.keyboard.type(' Hinzugefuegter QA-Satz PB5.')
  await page.waitForTimeout(400)
  await page.getByText('Speichern', { exact: true }).first().click()
  await page.waitForTimeout(1200)
  out.editPersisted = /Hinzugefuegter QA-Satz PB5\./.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'pb5-1-roundtrip.png'), fullPage: false })
})

// Version history: the diff must show content (adaptVersion now projects rows→html).
await step('versionDiff', async () => {
  await page.locator('button[title="Versionshistorie"]').first().click()
  await page.waitForTimeout(900)
  out.versionPanelOpen = /Versionen/.test(await bodyText())
  // Reveal a version's diff against the current article.
  const compare = page.getByText('Mit aktueller Version vergleichen', { exact: false }).first()
  if (await compare.count()) {
    await compare.click()
    await page.waitForTimeout(800)
  }
  const body = await bodyText()
  out.diffLabels = /Hinzugefügt|Entfernt|Keine Textunterschiede/.test(body)
  out.diffHasText = await page.evaluate(() => {
    const nodes = document.querySelectorAll('ins, del, [class*="diff"], .bg-success-light, .bg-destructive')
    let chars = 0
    nodes.forEach((n) => (chars += (n.textContent || '').length))
    return chars
  })
  await page.screenshot({ path: resolve(outDir, 'pb5-2-versions.png'), fullPage: false })
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
