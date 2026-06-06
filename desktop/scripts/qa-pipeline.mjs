import { chromium } from 'playwright'
import { resolve } from 'node:path'
const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/phase3-qa')
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const browser = await chromium.launch()
const out = []

async function openKanban(page) {
  await page.goto(`${BASE}/#/kontakte/pipeline`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1800)
  // click the pipeline (Columns3) toggle — 2nd icon button in the segmented group
  const toggle = page.locator('button:has(svg.lucide-columns-3)').first()
  if (await toggle.count()) await toggle.click()
  else await page.locator('div.rounded-md.border > button').nth(1).click()
  await page.waitForTimeout(1200)
}

// 1) render kanban @ full + half
for (const [name, w] of [['kanban', 1440], ['kanban', 900]]) {
  const ctx = await browser.newContext({ viewport: { width: w, height: 900 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []; page.on('pageerror', e => errs.push(String(e)))
  try {
    await openKanban(page)
    await page.screenshot({ path: resolve(outDir, `${name}-${w}.png`) })
    const cards = await page.locator('[role="button"]').count()
    out.push({ step: `render-${w}`, cards, errs: errs.length })
  } catch (e) { out.push({ step: `render-${w}`, error: String(e).split('\n')[0] }) } finally { await ctx.close() }
}

// 2) drag a card between columns
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []; page.on('pageerror', e => errs.push(String(e)))
  try {
    await openKanban(page)
    // first deal card (has text + currency). Cards are role=button inside columns.
    const cards = page.locator('div[role="button"]')
    const n = await cards.count()
    const src = cards.first()
    const srcBox = await src.boundingBox()
    // drop target: a point far to the right (next column area)
    const target = { x: srcBox.x + 320, y: srcBox.y + 40 }
    await page.mouse.move(srcBox.x + srcBox.width / 2, srcBox.y + 20)
    await page.mouse.down()
    await page.mouse.move(srcBox.x + srcBox.width / 2 + 40, srcBox.y + 30, { steps: 5 })
    await page.mouse.move(target.x, target.y, { steps: 10 })
    await page.waitForTimeout(300)
    await page.mouse.up()
    await page.waitForTimeout(1200)
    // toast text indicates move
    const toast = await page.evaluate(() => {
      const el = Array.from(document.querySelectorAll('*')).find(n => /verschoben|moved/i.test(n.textContent||'') && (n.textContent||'').length<60)
      return el ? el.textContent.trim() : null
    })
    await page.screenshot({ path: resolve(outDir, 'kanban-after-drag.png') })
    out.push({ step: 'drag', cardsBefore: n, toast, errs: errs.length })
  } catch (e) { out.push({ step: 'drag', error: String(e).split('\n')[0] }) } finally { await ctx.close() }
}

await browser.close(); console.log(JSON.stringify(out, null, 2))
