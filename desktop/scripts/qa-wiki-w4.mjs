// QA — wiki W-4: server search, view counts, tags/pins, author lookup (dev :5174)
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

const bodyText = () => page.evaluate(() => document.body.textContent || '')

try {
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)

  // ── A) Demo depth in the list: view counts + tags + pins ──
  let body = await bodyText()
  out.listHasViewCounts = /342|211|158/.test(body)
  out.listHasTags = /Onboarding|DATEV|Finanzen/.test(body)
  out.pinIcons = await page.locator('.lucide-pin').count()
  await page.screenshot({ path: resolve(outDir, 'w4-1-list-depth.png'), fullPage: false })

  // ── B) Server search (debounced) ──
  await page.getByPlaceholder('Wiki durchsuchen...').fill('Backup')
  await page.waitForTimeout(1100)
  body = await bodyText()
  out.searchCountLabel = /Treffer/.test(body)
  out.searchFoundBackup = /Backup & Disaster Recovery/.test(body)
  // a non-matching article should be filtered out of the result list
  const dsgvoVisible = await page.getByText('DSGVO: Einwilligungen', { exact: false }).count()
  out.searchFiltered = dsgvoVisible === 0
  await page.screenshot({ path: resolve(outDir, 'w4-2-search.png'), fullPage: false })
  await page.getByPlaceholder('Wiki durchsuchen...').fill('')
  await page.waitForTimeout(600)

  // ── C) Author lookup + view count in detail header ──
  await page.getByText('Angebots- und Rechnungsprozess', { exact: true }).first().click()
  await page.waitForTimeout(1000)
  body = await bodyText()
  out.authorName = /Markus Weber/.test(body)
  out.headerViewCount = /\d+ Aufrufe/.test(body)
  await page.screenshot({ path: resolve(outDir, 'w4-3-detail-author.png'), fullPage: false })

  // ── D) Pin toggle ──
  const header = page.locator('div.border-b', { has: page.locator('h2.text-lg') }).first()
  await header.getByRole('button').last().click()
  await page.waitForTimeout(400)
  await page.getByRole('menuitem', { name: /Anpinnen/ }).click()
  await page.waitForTimeout(700)
  out.pinnedToast = /Angepinnt/.test(await bodyText())
  out.headerPinIconAfter = await page.locator('h2.text-lg ~ *, .border-b .lucide-pin').count()
  await page.screenshot({ path: resolve(outDir, 'w4-4-pinned.png'), fullPage: false })

  // ── E) Version editor names resolved ──
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(800)
  await page.getByRole('button', { name: 'Versionshistorie' }).first().click()
  await page.waitForTimeout(800)
  body = await bodyText()
  out.versionEditorName = /Stefan Vogel|Jana Köhler/.test(body)
  out.noRawUserId = !/usr-e\d|usr-00\d/.test(body)
  await page.screenshot({ path: resolve(outDir, 'w4-5-version-names.png'), fullPage: false })

  out.rawKeys = await page.evaluate(() => {
    const all = Array.from(document.querySelectorAll('body *')).filter((n) => n.children.length === 0).map((n) => (n.textContent || '').trim())
    return [...new Set(all.filter((t) => /^(wiki|common|shared)\.[a-zA-Z]/.test(t)))].slice(0, 12)
  })
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'w4-error.png'), fullPage: false }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
