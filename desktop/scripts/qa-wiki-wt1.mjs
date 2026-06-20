// QA — wiki WT-1: category hierarchy (recursive tree) + move article (dev :5174)
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
      rawKeys: [...new Set(all.filter((t) => /^(wiki|common|shared)\.[a-zA-Z]/.test(t)))].slice(0, 15),
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
const bodyText = () => page.evaluate(() => document.body.textContent || '')

try {
  // ── A) Overview — recursive tree shows the subcategory nested ──
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)
  out.hasAllgemein = await page.getByText('Allgemein', { exact: true }).count()
  out.hasOnboardingSub = await page.getByText('Onboarding', { exact: true }).count()
  // Onboarding (wcat-004) is a child of Allgemein (wcat-001) — verify indentation
  out.onboardingIndent = await page.evaluate(() => {
    const nodes = Array.from(document.querySelectorAll('aside, div'))
    const el = Array.from(document.querySelectorAll('span'))
      .find((s) => s.textContent?.trim() === 'Onboarding')
    let row = el
    while (row && !(row.style && row.style.paddingLeft)) row = row.parentElement
    return row?.style?.paddingLeft ?? 'none'
  })
  await page.screenshot({ path: resolve(outDir, 'wt1-1-tree.png'), fullPage: false })
  Object.assign(out, await scanRaw(page))

  // ── B) Collapse the parent → subcategory hides ──
  out.collapseBtnCount = await page.getByRole('button', { name: 'Einklappen' }).count()
  await page.getByRole('button', { name: 'Einklappen' }).first().click()
  await page.waitForTimeout(700)
  out.onboardingAfterCollapse = await page.getByText('Onboarding', { exact: true }).count()
  await page.screenshot({ path: resolve(outDir, 'wt1-2-collapsed.png'), fullPage: false })
  // re-expand
  await page.getByRole('button', { name: 'Ausklappen' }).first().click()
  await page.waitForTimeout(500)

  // ── C) Open article → move dialog ──
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(1200)
  // The header's overflow menu is the button right after the "Teilen" action.
  await page.locator('button[title="Teilen"]').locator('xpath=following-sibling::button').first().click()
  await page.waitForTimeout(500)
  out.hasMoveItem = await page.getByRole('menuitem', { name: 'Verschieben' }).count()
  await page.getByRole('menuitem', { name: 'Verschieben' }).click()
  await page.waitForTimeout(700)
  out.moveDialogOpen = await page.getByRole('dialog').getByText('Artikel verschieben').count()
  out.moveShowsCurrent = await page.getByRole('dialog').getByText('Aktuell').count()
  await page.screenshot({ path: resolve(outDir, 'wt1-3-move-dialog.png'), fullPage: false })

  // pick a different category + confirm
  await page.getByRole('dialog').getByText('Prozesse & Workflows', { exact: true }).click()
  await page.waitForTimeout(300)
  await page.getByRole('dialog').getByRole('button', { name: 'Verschieben' }).click()
  await page.waitForTimeout(1200)
  const body = await bodyText()
  out.moveToast = /Artikel verschoben/.test(body)
  await page.screenshot({ path: resolve(outDir, 'wt1-4-after-move.png'), fullPage: false })

  // ── D) Narrow ──
  await page.setViewportSize({ width: 760, height: 900 })
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, 'wt1-5-narrow.png'), fullPage: false })
  out.narrow = await scanRaw(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'wt1-error.png'), fullPage: false }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
