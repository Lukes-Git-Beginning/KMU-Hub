// QA finanzen Banking-Fixes: Verbindungs-Dialog (Bank wählen → Login →
// verbunden), klickbare Transaktions-Zeile → Detail-Modal, manuelle Zuordnung.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const ELECTRON_STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUPPRESS = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(finanzen|common|shared)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}
async function dialogText(page) {
  return page.evaluate(() => {
    const d = document.querySelector('[role="dialog"]')
    return d ? d.innerText.replace(/\n{2,}/g, '\n').slice(0, 1200) : null
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
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(3000)
  await page.getByRole('button', { name: /^Banking/ }).first().click({ timeout: 6000 })
  await page.waitForTimeout(800)

  // ---- Connect dialog ----
  const conn = {}
  try {
    conn.connectBtnBefore = await page.getByRole('button', { name: /^Verbinden$/ }).count()
    await page.getByRole('button', { name: /^Verbinden$/ }).first().click({ timeout: 4000 })
    await page.waitForTimeout(600)
    conn.dialogOpen = !!(await page.locator('[role="dialog"]').count())
    conn.selectStepText = (await dialogText(page))?.slice(0, 200)
    await page.screenshot({ path: resolve(outDir, 'bank-connect-select.png') })
    // choose a bank (first row in dialog)
    await page.locator('[role="dialog"] button:has-text("Sparkasse")').first().click({ timeout: 4000 }).catch(async () => {
      await page.locator('[role="dialog"] button').nth(1).click({ timeout: 4000 })
    })
    await page.waitForTimeout(500)
    conn.loginStep = (await dialogText(page))?.includes('Anmeldename')
    await page.screenshot({ path: resolve(outDir, 'bank-connect-login.png') })
    conn.loginRawKeys = await scanRawKeys(page)
    // fill credentials
    await page.locator('[role="dialog"] input[type="text"]').first().fill('demo-user')
    await page.locator('[role="dialog"] input[type="password"]').first().fill('12345')
    await page.waitForTimeout(200)
    await page.getByRole('button', { name: /Sicher verbinden/ }).click({ timeout: 4000 })
    await page.waitForTimeout(600)
    conn.connectingShown = (await dialogText(page))?.includes('Verbindung wird hergestellt')
    await page.screenshot({ path: resolve(outDir, 'bank-connect-connecting.png') })
    await page.waitForTimeout(2200)
    conn.dialogClosed = !(await page.locator('[role="dialog"]').count())
    // sparkasse should now be connected → balance visible, no more Verbinden btn
    conn.connectBtnAfter = await page.getByRole('button', { name: /^Verbinden$/ }).count()
    await page.screenshot({ path: resolve(outDir, 'bank-connect-after.png') })
  } catch (e) { conn.error = String(e).split('\n')[0] }
  out.connect = conn

  // ---- Transaction row clickable → detail modal ----
  const det = {}
  try {
    const row = page.locator('div[role="button"].cursor-pointer').first()
    det.rowCount = await row.count()
    await row.click({ timeout: 4000 })
    await page.waitForTimeout(700)
    det.dialogOpen = !!(await page.locator('[role="dialog"]').count())
    const dt = await dialogText(page)
    det.isTxDetail = dt ? dt.includes('Transaktionsdetails') : false
    det.text = dt
    det.rawKeys = await scanRawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'bank-tx-detail.png') })
    await page.keyboard.press('Escape')
    await page.waitForTimeout(400)
  } catch (e) { det.error = String(e).split('\n')[0] }
  out.txDetail = det

  // ---- Manual assignment: open an 'Offen' credit tx detail, expect invoice list ----
  const man = {}
  try {
    await page.getByRole('button', { name: /^Offen \(/ }).first().click({ timeout: 4000 }).catch(() => {})
    await page.waitForTimeout(500)
    // click the search (manual match) button on first unmatched credit row
    const searchBtn = page.locator('button[title="Manuell zuordnen"]').first()
    man.manualBtnCount = await searchBtn.count()
    if (man.manualBtnCount) {
      await searchBtn.click({ timeout: 4000 })
      await page.waitForTimeout(700)
      const dt = await dialogText(page)
      man.hasAssignSection = dt ? dt.includes('Rechnung zuordnen') : false
      man.text = dt
      await page.screenshot({ path: resolve(outDir, 'bank-manual-assign.png') })
    }
  } catch (e) { man.error = String(e).split('\n')[0] }
  out.manualAssign = man
} catch (err) {
  out.fatal = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
