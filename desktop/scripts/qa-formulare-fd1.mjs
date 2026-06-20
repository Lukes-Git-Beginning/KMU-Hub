/**
 * QA — formulare FD-1: real share link (token URL, clipboard, channels).
 *  1) Active form → Teilen opens the share dialog (channel + settings).
 *  2) "Link erstellen" generates a real forms.zentria.tech/r/{token} URL.
 *  3) Embed channel yields an <iframe …> snippet.
 *  4) navigator.clipboard.writeText fires ("Kopiert" feedback).
 *  No raw keys, 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-fd1')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW_RE = /(formulare\.[a-z][a-zA-Z.]+|moduleSettings\.[a-z]|\{\{)/

async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 12)
  }, RAW_RE.source)
}

async function openShareFor(page, title) {
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator(`[role="button"]:has-text("${title}")`).first().click({ timeout: 6000 })
  await page.waitForTimeout(800)
  await page.locator('[role="dialog"]').last().locator('button:has-text("Teilen")').first().click({ timeout: 5000 })
  await page.waitForTimeout(700)
}

const browser = await chromium.launch()
const out = []
const errs = []

// ── 1+2) link channel ──────────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.grantPermissions(['clipboard-read', 'clipboard-write'], { origin: BASE })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('link: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('link console: ' + m.text()) })
  await openShareFor(page, 'Kundenfeedback')
  const dlg = page.locator('[role="dialog"]').last()
  await page.screenshot({ path: resolve(outDir, '1-share-stage1.png') })
  const stage1Text = await dlg.textContent().catch(() => '')
  await dlg.locator('button:has-text("Link erstellen")').first().click({ timeout: 5000 })
  await page.waitForTimeout(900)
  const url = await dlg.locator('input[readonly]').first().inputValue().catch(() => '')
  const copyBtnText = await dlg.locator('button:has-text("Kopiert"), button:has-text("Kopieren")').first().textContent().catch(() => '')
  // Read clipboard to confirm the real write happened
  const clip = await page.evaluate(() => navigator.clipboard.readText().catch(() => '')).catch(() => '')
  await page.screenshot({ path: resolve(outDir, '2-share-stage2-link.png') })
  out.push({
    check: 'link-channel',
    hasChannelPicker: /Kanal/.test(stage1Text || ''),
    url,
    urlOk: /forms\.zentria\.tech\/r\//.test(url || ''),
    clipboardMatches: clip === url && !!url,
    copyBtnText: (copyBtnText || '').trim(),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 3) embed channel → iframe snippet ───────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.grantPermissions(['clipboard-read', 'clipboard-write'], { origin: BASE })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('embed: ' + String(e)))
  await openShareFor(page, 'Kontaktanfrage')
  const dlg = page.locator('[role="dialog"]').last()
  await dlg.locator('button:has-text("Einbettung")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(300)
  await dlg.locator('button:has-text("Link erstellen")').first().click({ timeout: 5000 })
  await page.waitForTimeout(900)
  const snippet = await dlg.locator('textarea[readonly]').first().inputValue().catch(() => '')
  await page.screenshot({ path: resolve(outDir, '3-share-embed.png') })
  out.push({
    check: 'embed-channel',
    snippetStart: (snippet || '').slice(0, 40),
    isIframe: /^<iframe /.test(snippet || ''),
  })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
