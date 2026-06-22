import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/db5-columns')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(document|blocks|wiki|common)\.[a-z]+\.[a-z._]+/i

function findRawKeys(reSource) {
  const rx = new RegExp(reSource, 'i')
  return [
    ...new Set(
      Array.from(document.querySelectorAll('body *'))
        .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
        .map((n) => n.textContent.trim()),
    ),
  ].slice(0, 12)
}

const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 }, deviceScaleFactor: 2 })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

await page.goto(`${BASE}/#/wiki?a=wart-001`, { waitUntil: 'domcontentloaded' })
await page.getByRole('heading', { name: /Willkommen im Cosmi-Wiki/ }).first().waitFor({ timeout: 15000 }).catch(() => {})

// Edit mode → the two-column row whose left column stacks two blocks.
const editBtn = page.locator('button[title="Bearbeiten"]').first()
await editBtn.click().catch(() => {})
await page.waitForTimeout(1800)

const stackHint = page.getByText(/ordnest du sie per Pfeil neu/).first()
await stackHint.scrollIntoViewIfNeeded().catch(() => {})
await page.waitForTimeout(400)

out.moveUpButtons = await page.locator('button[aria-label="Block nach oben"]').count()
out.moveDownButtons = await page.locator('button[aria-label="Block nach unten"]').count()

// Hover the stacked paragraph block so its control cluster becomes visible.
const stackedBlock = stackHint.locator('xpath=ancestor::div[contains(@class,"group/block")][1]')
await stackedBlock.hover().catch(async () => {
  await stackHint.hover().catch(() => {})
})
await page.waitForTimeout(400)
out.edit = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, '01-edit-controls.png'), fullPage: false })

// Zoomed capture of the two-column row.
const colsRow = page.locator('div.flex.gap-3').filter({ hasText: 'ordnest du sie per Pfeil neu' }).first()
await colsRow.screenshot({ path: resolve(outDir, 'zoom-columns.png') }).catch(() => {})

out.errors = errs.slice(0, 6)
console.log(JSON.stringify(out, null, 2))
await browser.close()
