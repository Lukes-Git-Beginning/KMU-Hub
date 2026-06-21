// QA — wiki PB-3 TOC + edit-mode attachments + file-first image block (dev :5173)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/wiki-pb3')
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
  try {
    await fn()
  } catch (err) {
    out[`ERR_${name}`] = String(err).split('\n')[0]
  }
}

await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500)

// ── PB-3: TOC from block headings ──
await step('toc', async () => {
  await page.getByText('Onboarding neuer Mitarbeitender', { exact: true }).first().click()
  await page.waitForTimeout(1200)
  out.tocPresent = await page.evaluate(() => !!document.querySelector('nav[aria-label]'))
  out.tocLinks = await page.locator('nav[aria-label] button').count()
  out.tocTitleShown = /Inhalt/.test(await bodyText())
  out.readingTimeShown = /Lesezeit/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'pb3-1-toc.png'), fullPage: false })
})

await step('tocJump', async () => {
  const buttons = page.locator('nav[aria-label] button')
  const n = await buttons.count()
  if (n >= 2) {
    await buttons.nth(n - 1).click()
    await page.waitForTimeout(900)
    out.activeAfterJump = await page.evaluate(() => {
      const a = document.querySelector('nav[aria-label] button.text-primary, nav[aria-label] button.border-primary')
      return a ? a.textContent.trim() : null
    })
  }
  await page.screenshot({ path: resolve(outDir, 'pb3-2-toc-jumped.png'), fullPage: false })
})

// ── #6: attachments available in EDIT mode ──
await step('editAttachments', async () => {
  await page.locator('button[title="Bearbeiten"]').first().click()
  await page.waitForTimeout(1200)
  const body = await bodyText()
  out.editHasAttachments = /ANHÄNGE|Anhänge|Anhang/i.test(body)
  await page.screenshot({ path: resolve(outDir, 'pb3-3-edit-attachments.png'), fullPage: true })
})

// ── #7: file-first image block ──
await step('imageBlock', async () => {
  await page.getByText('Block einfügen', { exact: true }).first().click()
  await page.waitForTimeout(500)
  await page.getByRole('button', { name: 'Bild', exact: true }).first().click()
  await page.waitForTimeout(700)
  const body = await bodyText()
  out.imgHasPicker = /Bild auswählen/.test(body)
  out.imgHasFilesHint = /Aus deinen Dateien/.test(body)
  out.imgUrlSecondary = /Bild-Adresse einfügen/.test(body)
  out.imgNoBareUrlLabel = !/Bild-URL/.test(body)
  out.imgHasFileInput = await page.locator('input[type="file"][accept="image/*"]').count()
  await page.screenshot({ path: resolve(outDir, 'pb3-4-image-picker.png'), fullPage: false })
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
