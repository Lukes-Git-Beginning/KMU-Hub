/**
 * One-off: readonly (Elena) opens a FOREIGN member profile via the card menu —
 * verify contact/employment sections + documents tab are hidden in the modal.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()

// switch to Elena (readonly) from settings
await page.goto(`${BASE}/#/settings`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(1200)
await page.locator('button.fixed.bottom-4.right-4').click()
await page.waitForTimeout(400)
await page.locator('div.max-h-80').getByRole('button', { name: /Elena Richter/ }).first().click()
await page.waitForTimeout(1700)
// close the switcher panel (its backdrop overlay intercepts clicks)
if (await page.locator('div.max-h-80').isVisible().catch(() => false)) {
  await page.locator('button.fixed.bottom-4.right-4').click()
  await page.waitForTimeout(400)
}

await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(1600)
// open the item menu of the FIRST card (Andrea Keller ≠ Elena) → "Profil ansehen"
await page.locator('.grid button[aria-haspopup="menu"]').first().click()
await page.waitForTimeout(500)
await page.getByRole('menuitem', { name: /Profil ansehen/ }).first().click()
await page.waitForTimeout(1800)

const txt = await page.evaluate(() => document.body.innerText)
const tabs = (await page.getByRole('tab').allInnerTexts()).map((s) => s.trim())
await page.screenshot({ path: resolve('.qa-screenshots/rbac-enforcement-b2/13-readonly-profile-modal.png') })
console.log(JSON.stringify({
  modalOpen: tabs.length > 0,
  tabs,
  noDocsTab: !tabs.some((t) => /Dokumente/.test(t)),
  noEmployment: !/Beschäftigung/i.test(txt),
  noContactSection: !/Notfallkontakt/i.test(txt),
  menuHasDeactivate: /Deaktivieren/.test(txt),
}))
await b.close()
