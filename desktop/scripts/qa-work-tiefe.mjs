// QA work Tiefe-Pass W-1/2/3: centered DetailModal (not slide-over) on task
// click in list + kanban, fully clickable rows, MyTasks standalone open + move.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const ELECTRON_STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUPPRESS = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(work|common|shared)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}
async function dialogInfo(page) {
  return page.evaluate(() => {
    const d = document.querySelector('[role="dialog"]')
    if (!d) return null
    const r = d.getBoundingClientRect()
    const vw = window.innerWidth
    const leftGap = r.left
    const rightGap = vw - r.right
    return {
      text: d.innerText.replace(/\n{2,}/g, '\n').slice(0, 600),
      leftGap: Math.round(leftGap),
      rightGap: Math.round(rightGap),
      centered: Math.abs(leftGap - rightGap) < 90,
      width: Math.round(r.width),
    }
  })
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 960 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const out = {}

try {
  // ---- Projects list -> enter a project ----
  await page.goto(`${BASE}/#/work/projects`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(3500)
  out.projectsRawKeys = await scanRawKeys(page)
  await page.getByText('Cosmi v2.0', { exact: false }).first().click({ timeout: 6000 })
  await page.waitForTimeout(2500)
  out.boardUrl = page.url()
  out.boardRawKeys = await scanRawKeys(page)
  await page.screenshot({ path: resolve(outDir, 'work-2-board.png') })

  // ---- List view: click a task row -> centered modal ----
  const list = {}
  try {
    await page.locator('button[title="Liste"]').first().click({ timeout: 3000 }).catch(() => {})
    await page.waitForTimeout(900)
    const row = page.locator('div[role="button"].cursor-pointer').first()
    list.rowCount = await page.locator('div[role="button"].cursor-pointer').count()
    await row.click({ timeout: 5000 })
    await page.waitForTimeout(900)
    list.dialog = await dialogInfo(page)
    list.rawKeys = await scanRawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'work-3-list-modal.png') })
    await page.keyboard.press('Escape')
    await page.waitForTimeout(500)
  } catch (e) { list.error = String(e).split('\n')[0] }
  out.listModal = list

  // ---- Kanban view: click a card -> centered modal ----
  const kan = {}
  try {
    await page.locator('button[title="Kanban"]').first().click({ timeout: 3000 }).catch(() => {})
    await page.waitForTimeout(1200)
    await page.screenshot({ path: resolve(outDir, 'work-4-kanban.png') })
    const card = page.locator('.cursor-grab').first()
    kan.cardCount = await page.locator('.cursor-grab').count()
    await card.click({ timeout: 5000 })
    await page.waitForTimeout(900)
    kan.dialog = await dialogInfo(page)
    kan.rawKeys = await scanRawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'work-5-kanban-modal.png') })
    await page.keyboard.press('Escape')
    await page.waitForTimeout(500)
  } catch (e) { kan.error = String(e).split('\n')[0] }
  out.kanbanModal = kan

  // ---- MyTasks: standalone create + move-to-project, then standalone modal ----
  const my = {}
  async function createStandalone(title) {
    await page.getByRole('button', { name: /Neue Aufgabe/ }).first().click({ timeout: 5000 })
    await page.waitForTimeout(600)
    await page.locator('input[placeholder="Was ist zu tun?"]').fill(title)
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1200)
  }
  try {
    await page.goto(`${BASE}/#/work/my-tasks`, { waitUntil: 'domcontentloaded', timeout: 25000 })
    await page.waitForTimeout(3000)
    my.rawKeys = await scanRawKeys(page)

    // (1) MOVE test — create a task, move it via popover, expect toast + regroup
    await createStandalone('QA Move-Aufgabe')
    my.hasPersonal = (await page.getByText('Persönlich', { exact: false }).count()) > 0
    await page.screenshot({ path: resolve(outDir, 'work-6-mytasks.png') })
    const moreBtn = page.locator('button[title="In Projekt verschieben"]').first()
    my.moveBtnCount = await page.locator('button[title="In Projekt verschieben"]').count()
    if (my.moveBtnCount) {
      my.topAtBtn = await page.evaluate(() => {
        const b = document.querySelector('button[title="In Projekt verschieben"]')
        if (!b) return null
        const r = b.getBoundingClientRect()
        const el = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2)
        return el === b || b.contains(el) ? 'button' : (el?.tagName + '.' + (el?.className || '').toString().slice(0, 40))
      })
      // JS click bypasses Playwright actionability (topbar clock re-renders make
      // the 20px icon flap "unstable" for the harness; a real user click works).
      await moreBtn.evaluate((el) => el.click())
      await page.waitForTimeout(700)
      await page.screenshot({ path: resolve(outDir, 'work-8-move-popover.png') })
      const projBtn = page.locator('[data-radix-popper-content-wrapper] button').first()
      my.projBtnCount = await projBtn.count()
      if (my.projBtnCount) {
        await projBtn.evaluate((el) => el.click()).catch(() => {})
        await page.waitForTimeout(1000)
      }
      my.afterMoveToast = await page.evaluate(() => {
        const t = document.querySelector('[data-sonner-toast]')
        return t ? t.innerText : null
      })
      await page.screenshot({ path: resolve(outDir, 'work-9-after-move.png') })
    }

    // (2) MODAL test — fresh standalone task, open -> centered modal
    await createStandalone('QA Modal-Aufgabe')
    await page.getByText('QA Modal-Aufgabe').first().click({ timeout: 5000 })
    await page.waitForTimeout(900)
    my.dialog = await dialogInfo(page)
    await page.screenshot({ path: resolve(outDir, 'work-7-standalone-modal.png') })
  } catch (e) { my.error = String(e).split('\n')[0] }
  out.myTasks = my
} catch (err) {
  out.fatal = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
