/**
 * QA — dashboard F6/F7: drag placeholder colour + no text selection.
 *  F6: react-grid-placeholder background must NOT be red (rgb(255,0,0)).
 *  F7: draggable grid items must have user-select: none.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/dashboard-grid')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const seed = `try{localStorage.setItem('cosmi-dashboard',JSON.stringify({state:{scope:'personal',personalActiveWidgets:['my-tasks','birthdays','team-status','kpi-revenue'],personalLayouts:[{i:'my-tasks',x:0,y:0,w:6,h:4,minW:3,minH:3},{i:'birthdays',x:6,y:0,w:6,h:4,minW:3,minH:3},{i:'team-status',x:0,y:4,w:6,h:4,minW:3,minH:3},{i:'kpi-revenue',x:6,y:4,w:6,h:3,minW:3,minH:2}],teamActiveWidgets:[],teamLayouts:[]},version:2}))}catch(e){}`

const browser = await chromium.launch()
const out = []
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(seed)
const page = await ctx.newPage(); const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

try {
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2500)

  // enter edit mode
  const editBtn = page.locator('button:has-text("Anpassen"), button:has-text("Bearbeiten"), button:has-text("Dashboard anpassen")').first()
  if (await editBtn.isVisible().catch(() => false)) { await editBtn.click({ timeout: 5000 }); await page.waitForTimeout(800) }
  await page.evaluate(() => { const el = document.querySelector('.layout'); if (el) el.scrollIntoView() })
  await page.waitForTimeout(400)

  // F7: user-select on draggable items
  const userSelect = await page.evaluate(() => {
    const item = document.querySelector('.react-grid-item.react-draggable')
    return item ? getComputedStyle(item).userSelect || getComputedStyle(item).webkitUserSelect : 'no-item'
  })

  // F6: drag a widget to surface the placeholder, read its background
  const handle = page.locator('.widget-drag-handle').first()
  const box = await handle.boundingBox()
  let placeholder = null
  if (box) {
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
    await page.mouse.down()
    await page.mouse.move(box.x + 200, box.y + 160, { steps: 8 })
    await page.waitForTimeout(350)
    placeholder = await page.evaluate(() => {
      const ph = document.querySelector('.react-grid-placeholder')
      if (!ph) return null
      const s = getComputedStyle(ph)
      return { background: s.backgroundColor, borderRadius: s.borderRadius, border: s.borderColor }
    })
    await page.screenshot({ path: resolve(outDir, '1-dragging-placeholder.png') })
    await page.mouse.up()
    await page.waitForTimeout(300)
  }

  const isRed = placeholder ? /rgba?\(\s*255,\s*0,\s*0/.test(placeholder.background) : null
  out.push({
    userSelect,
    userSelectOk: userSelect === 'none',
    placeholder,
    placeholderNotRed: placeholder ? !isRed : null,
    pageErrors: errs.slice(0, 3),
  })
} catch (e) { out.push({ error: String(e).split('\n')[0], pageErrors: errs.slice(0, 3) }) }
finally { await ctx.close(); await browser.close() }

console.log(JSON.stringify(out, null, 2))
