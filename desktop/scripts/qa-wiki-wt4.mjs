// QA — wiki WT-4: attachment download + image preview, share token management (dev :5174)
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
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, acceptDownloads: true })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const bodyText = () => page.evaluate(() => document.body.textContent || '')
const tokenCount = async () => {
  const m = (await bodyText()).match(/Aktive Freigaben\s*\((\d+)\)/)
  return m ? Number(m[1]) : -1
}

try {
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)

  // ── A) Attachments: image preview + download buttons ──
  await page.getByText('Onboarding neuer Mitarbeitender', { exact: true }).first().click()
  await page.waitForTimeout(1200)
  out.imageThumb = await page.locator('img[alt="onboarding-ablauf.svg"]').count()
  out.downloadBtns = await page.getByRole('button', { name: 'Herunterladen' }).count()
  await page.screenshot({ path: resolve(outDir, 'wt4-1-attachments.png'), fullPage: false })

  // download the PDF (generated demo blob) and the image (data URL)
  try {
    const [dlPdf] = await Promise.all([
      page.waitForEvent('download', { timeout: 5000 }),
      page.getByRole('button', { name: 'Herunterladen' }).first().click(),
    ])
    out.pdfDownloadName = dlPdf.suggestedFilename()
  } catch (e) { out.pdfDownloadErr = String(e).split('\n')[0] }
  try {
    const [dlImg] = await Promise.all([
      page.waitForEvent('download', { timeout: 5000 }),
      page.getByRole('button', { name: 'Herunterladen' }).nth(1).click(),
    ])
    out.imgDownloadName = dlImg.suggestedFilename()
  } catch (e) { out.imgDownloadErr = String(e).split('\n')[0] }
  Object.assign(out, await scanRaw(page))

  // ── B) Share dialog: token list + internal link ──
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(1000)
  await page.getByRole('button', { name: 'Teilen' }).first().click()
  await page.waitForTimeout(800)
  out.tokensInitial = await tokenCount()
  out.internalLink = await page.evaluate(() => {
    const dlg = document.querySelector('[role="dialog"]')
    return /\?a=wart-001/.test(dlg?.textContent || '')
  })
  out.openInApp = await page.getByRole('button', { name: 'Im Wiki öffnen' }).count()
  await page.screenshot({ path: resolve(outDir, 'wt4-2-share-dialog.png'), fullPage: false })

  // create a token
  await page.getByRole('button', { name: 'Freigabe-Link erstellen' }).click()
  await page.waitForTimeout(1000)
  out.tokensAfterCreate = await tokenCount()
  out.createToast = /Freigabe erstellt/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'wt4-3-share-created.png'), fullPage: false })

  // revoke one token
  await page.getByRole('button', { name: 'Widerrufen' }).first().click()
  await page.waitForTimeout(1000)
  out.tokensAfterRevoke = await tokenCount()
  out.revokeToast = /Freigabe widerrufen/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'wt4-4-share-revoked.png'), fullPage: false })

  // ── C) Narrow ──
  await page.keyboard.press('Escape')
  await page.setViewportSize({ width: 820, height: 900 })
  await page.waitForTimeout(600)
  await page.getByText('Onboarding neuer Mitarbeitender', { exact: true }).first().click()
  await page.waitForTimeout(800)
  await page.screenshot({ path: resolve(outDir, 'wt4-5-narrow.png'), fullPage: false })
  out.narrow = await scanRaw(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'wt4-error.png'), fullPage: false }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
