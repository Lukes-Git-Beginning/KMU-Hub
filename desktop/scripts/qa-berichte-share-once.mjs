import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

// Share links are shown once: the new link appears after "Link erzeugen" with a
// copy button and a hint, the list carries no token, and "1 Jahr" replaced the
// "never" option (the server caps every link at 365 days).
const BASE = process.env.QA_BASE || 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/berichte-share-once')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(berichte|common|shared)\.[a-z]+\.[a-z._]+/i

function findRawKeys(re) {
  const rx = new RegExp(re, 'i')
  return [
    ...new Set(
      Array.from(document.querySelectorAll('body *'))
        .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
        .map((n) => n.textContent.trim()),
    ),
  ].slice(0, 12)
}

const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({
  viewport: { width: 1440, height: 1000 },
  permissions: ['clipboard-read', 'clipboard-write'],
})
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
page.setDefaultTimeout(8000)
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
// LaunchOverlay intercepts every click until it unmounts.
await page.locator('.cosmi-startup').waitFor({ state: 'detached', timeout: 20000 }).catch(() => {})
const tab = page.getByRole('button', { name: /^Berichte$/ }).first()
if (await tab.count()) await tab.click().catch(() => {})
await page.waitForTimeout(1000)

await page.getByRole('button').filter({ hasText: /Verkaufsbericht Q2 2026/ }).first().click().catch(() => {})
await page.waitForTimeout(1500)
await page.getByRole('button', { name: /^Teilen$/ }).first().click().catch(() => {})
await page.waitForTimeout(400)
await page.getByRole('button', { name: /Externen Link erzeugen/ }).first().click().catch(() => {})
await page.waitForTimeout(700)

out.dialogOpen = await page.getByText(/Externer Link/).count()
out.oneYearOption = await page.getByRole('button', { name: /^1 Jahr$/ }).count()
out.neverOption = await page.getByText(/^Unbegrenzt$/).count()
await page.screenshot({ path: resolve(outDir, '1-empty.png') })

// pick 1 year and create
await page.getByRole('button', { name: /^1 Jahr$/ }).first().click().catch(() => {})
await page.getByRole('button', { name: /^Link erzeugen$/ }).first().click().catch(() => {})
await page.waitForTimeout(900)
out.freshPanel = await page.getByText(/^Neuer Link$/).count()
out.onceHint = await page.getByText(/nur jetzt angezeigt/).count()
out.urlShown = await page.getByText(/cosmi:\/\/share\/report\/tok-/).count()
out.listRowCreatedOn = await page.getByText(/Erstellt am/).count()
out.expiresShown = await page.getByText(/Läuft ab am/).count()
await page.getByRole('button', { name: /^Kopieren$/ }).first().click().catch(() => {})
await page.waitForTimeout(300)
out.clipboard = await page.evaluate(() => navigator.clipboard.readText()).catch((e) => String(e))
out.dialogRawKeys = await page.evaluate(findRawKeys, RAW.source)
await page.screenshot({ path: resolve(outDir, '2-created.png') })

// close and reopen: the link must not come back
await page.getByRole('button', { name: /^Fertig$/ }).first().click().catch(() => {})
await page.waitForTimeout(500)
await page.getByRole('button', { name: /^Teilen$/ }).first().click().catch(() => {})
await page.waitForTimeout(400)
await page.getByRole('button', { name: /Externen Link erzeugen/ }).first().click().catch(() => {})
await page.waitForTimeout(900)
out.reopenFreshPanel = await page.getByText(/^Neuer Link$/).count()
out.reopenUrlShown = await page.getByText(/cosmi:\/\/share\/report/).count()
out.reopenListRow = await page.getByText(/Erstellt am/).count()
await page.screenshot({ path: resolve(outDir, '3-reopened.png') })

// revoke it
await page.getByRole('button', { name: /^Widerrufen$/ }).first().click().catch(() => {})
await page.waitForTimeout(900)
out.afterRevokeEmpty = await page.getByText(/Noch keine aktiven Links/).count()
await page.screenshot({ path: resolve(outDir, '4-revoked.png') })

out.errs = errs.length
out.errDetail = errs.slice(0, 3)
console.log(JSON.stringify(out, null, 2))
await browser.close()
