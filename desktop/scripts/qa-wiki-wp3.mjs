// QA — wiki WP-3: article identity (cover + icon) (dev :5174)
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
      icuLeak: [...new Set(all.filter((t) => /\{count|plural,/.test(t)))].slice(0, 10),
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

try {
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)

  // ── A) List: icons / initials next to titles ──
  out.listIconBadges = await page.locator('.flex-1 span.inline-flex.bg-primary\\/10, span.bg-primary\\/10').count()
  await page.screenshot({ path: resolve(outDir, 'wp3-1-list-icons.png'), fullPage: false })

  // ── B) Read head: cover banner + icon (wart-001 has aurora + Rocket) ──
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(1000)
  out.readCoverPresent = await page.evaluate(() => {
    // header cover is a div with a background gradient, height 96px (h-24)
    return Array.from(document.querySelectorAll('div')).some((d) => {
      const s = getComputedStyle(d)
      return /gradient/.test(s.backgroundImage) && Math.round(d.getBoundingClientRect().height) >= 80 && Math.round(d.getBoundingClientRect().height) <= 120
    })
  })
  await page.screenshot({ path: resolve(outDir, 'wp3-2-read-cover.png'), fullPage: false })

  // ── C) Edit head: identity bar with cover + icon + change controls ──
  await page.locator('button[title="Bearbeiten"]').first().click()
  await page.waitForTimeout(1000)
  out.editCoverBanner = await page.evaluate(() =>
    Array.from(document.querySelectorAll('div')).some((d) => {
      const s = getComputedStyle(d)
      return /gradient/.test(s.backgroundImage) && Math.round(d.getBoundingClientRect().height) >= 130 && Math.round(d.getBoundingClientRect().height) <= 160
    }),
  )
  await page.screenshot({ path: resolve(outDir, 'wp3-3-edit-identity.png'), fullPage: false })

  // ── D) Open the icon picker ──
  // hover the cover to reveal change controls, then open icon picker via the lg icon
  await page.locator('.wiki-title-input').first().hover()
  await page.waitForTimeout(300)
  const lgIcon = page.locator('button[aria-label]').filter({ hasText: '' }).first()
  // click the big icon button (aria-label = icon label) to open the picker
  await page.locator('button:has(> span.h-14)').first().click().catch(() => {})
  await page.waitForTimeout(500)
  out.iconPickerOpen = await page.evaluate(() => /Symbol/.test(document.body.textContent || ''))
  await page.screenshot({ path: resolve(outDir, 'wp3-4-icon-picker.png'), fullPage: false })

  Object.assign(out, await scanRaw(page))

  // ── E) An article without a cover: the add controls show ──
  await page.keyboard.press('Escape')
  await page.getByRole('button', { name: 'Abbrechen' }).first().click().catch(() => {})
  await page.waitForTimeout(500)
  await page.getByText('Backup & Disaster Recovery', { exact: true }).first().click()
  await page.waitForTimeout(800)
  await page.locator('button[title="Bearbeiten"]').first().click()
  await page.waitForTimeout(800)
  out.addControlsVisible = await page.evaluate(() => /Titelbild|Symbol/.test(document.body.textContent || ''))
  await page.screenshot({ path: resolve(outDir, 'wp3-5-add-controls.png'), fullPage: false })
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'wp3-error.png'), fullPage: false }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
