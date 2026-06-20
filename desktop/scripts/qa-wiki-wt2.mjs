// QA — wiki WT-2: editable tags + tag filter + inline title rename (dev :5174)
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
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)

  // ── A) Tag filter from the list ──
  await page.getByRole('button', { name: 'DSGVO', exact: true }).first().click()
  await page.waitForTimeout(700)
  out.filterBannerVisible = await page.locator('button[title="Tag-Filter entfernen"]').count()
  out.filteredHasDSGVO = await page.getByText('DSGVO: Einwilligungen & Auskunftsrechte').count()
  out.filteredHidesBackup = await page.getByText('Backup & Disaster Recovery').count()
  await page.screenshot({ path: resolve(outDir, 'wt2-1-tag-filter.png'), fullPage: false })
  // clear filter
  await page.locator('button[title="Tag-Filter entfernen"]').click()
  await page.waitForTimeout(500)
  out.afterClearShowsBackup = await page.getByText('Backup & Disaster Recovery').count()
  Object.assign(out, await scanRaw(page))

  // ── B) Open article → editable tags ──
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(1200)
  out.addTagBtn = await page.getByRole('button', { name: 'Tag', exact: true }).count()
  await page.getByRole('button', { name: 'Tag', exact: true }).first().click()
  await page.waitForTimeout(400)
  await page.getByPlaceholder('Tag eingeben…').fill('Wichtig')
  await page.waitForTimeout(300)
  await page.getByPlaceholder('Tag eingeben…').press('Enter')
  await page.waitForTimeout(1000)
  const bodyAdd = await bodyText()
  out.tagAddedToast = /Tag hinzugefügt/.test(bodyAdd)
  out.newTagVisible = await page.getByText('Wichtig', { exact: true }).count()
  await page.screenshot({ path: resolve(outDir, 'wt2-2-tag-added.png'), fullPage: false })

  // remove a tag (the freshly added "Wichtig" → click its remove)
  await page.getByRole('button', { name: 'Tag entfernen' }).last().click()
  await page.waitForTimeout(1000)
  out.tagRemovedToast = /Tag entfernt/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'wt2-3-tag-removed.png'), fullPage: false })

  // ── C) Inline title rename (header title button carries title="Titel bearbeiten") ──
  await page.locator('button[title="Titel bearbeiten"]').click()
  await page.waitForTimeout(400)
  const titleInput = page.locator('input[aria-label="Titel bearbeiten"]')
  await titleInput.fill('Willkommen im Cosmi-Wiki (aktualisiert)')
  await titleInput.press('Enter')
  await page.waitForTimeout(1000)
  const bodyTitle = await bodyText()
  out.titleRenamedToast = /Titel aktualisiert/.test(bodyTitle)
  out.newTitleVisible = /Willkommen im Cosmi-Wiki \(aktualisiert\)/.test(bodyTitle)
  await page.screenshot({ path: resolve(outDir, 'wt2-4-title-renamed.png'), fullPage: false })

  // ── D) Narrow ──
  await page.setViewportSize({ width: 760, height: 900 })
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, 'wt2-5-narrow.png'), fullPage: false })
  out.narrow = await scanRaw(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'wt2-error.png'), fullPage: false }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
