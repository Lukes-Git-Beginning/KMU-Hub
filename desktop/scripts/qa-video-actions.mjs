/**
 * QA — Video action buttons (dead-button fix).
 * Header "Meeting starten"/"Neuer Anruf" + CallHistory call-back buttons now open
 * the reused MeetingFormDialog; call-back seeds the title with the contact name.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/video-actions')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
// Skip the CosmiLaunch startup splash (sessionStorage flag) so it never overlays QA.
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []

try {
  await page.goto(`${BASE}/#/video`, { waitUntil: 'domcontentloaded' })
  // Let the CosmiLaunch dev animation finish (clears by ~4s) before interacting.
  await page.waitForTimeout(4500)
  const startBtn = page.getByRole('button', { name: /Meeting starten/ })
  await startBtn.waitFor({ state: 'visible', timeout: 10000 })
  await page.screenshot({ path: resolve(outDir, '1-video-page.png') })

  // 1) Header "Meeting starten" opens the dialog
  await startBtn.click()
  await page.waitForTimeout(1500)
  await page.screenshot({ path: resolve(outDir, '2-meeting-starten-dialog.png') })
  const dlgText = await page.evaluate(() => document.querySelector('[role="dialog"]')?.textContent || '')
  const anyModal = await page.evaluate(() => {
    const d = document.querySelector('[role="dialog"], [data-state="open"]')
    return { hasRoleDialog: !!document.querySelector('[role="dialog"]'), dataStateOpen: !!document.querySelector('[data-state="open"]'), inputs: document.querySelectorAll('input').length, sample: d ? (d.textContent || '').slice(0, 40) : '' }
  })
  out.push({ step: 'header meeting-starten', ...anyModal, pass: /Neues Meeting|Titel|Teilnehmer/.test(dlgText) })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(500)

  // 2) History tab → call-back button opens dialog with prefilled title
  await page.getByRole('button', { name: /Anrufverlauf/ }).click()
  await page.waitForTimeout(1200)
  await page.screenshot({ path: resolve(outDir, '3-history-tab.png') })
  // first call-back button (video) — labelled "Videoanruf"
  const cbBtn = page.locator('button[aria-label="Videoanruf"]').first()
  const clicked = (await cbBtn.count()) > 0
  await cbBtn.click({ force: true })
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(600)
  const titleVal = await page.evaluate(() => {
    const dlg = document.querySelector('[role="dialog"]')
    const input = dlg?.querySelector('input:not([type="date"]):not([type="time"])')
    return input ? input.value : ''
  })
  await page.screenshot({ path: resolve(outDir, '4-callback-prefilled.png') })
  out.push({ step: 'callback prefilled-title', clicked, titlePrefilled: titleVal.length > 0, title: titleVal, pass: clicked && titleVal.length > 0 })
} catch (e) {
  out.push({ step: 'error', pass: false, error: String(e).split('\n')[0] })
}

out.push({ step: 'pageerrors', errors: errs.slice(0, 8), pass: errs.length === 0 })
await ctx.close(); await b.close()
const allPass = out.every((s) => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== VIDEO-ACTIONS QA: ${allPass ? 'ALL PASS' : 'REVIEW'} ===`)
out.forEach((s) => console.log(`  ${s.pass ? '✓' : '✗'} ${s.step}`))
