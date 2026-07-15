/**
 * QA — Video Demo-Tiefe.
 * Verifies: CallHistory row → CallHistoryDetailModal (notes, recording,
 * participants, call-back + download buttons), and the Settings tab device
 * controls + real camera preview (fake media device → active <video>).
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/video-tiefe')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

// Fake media device so getUserMedia yields an active preview stream.
const b = await chromium.launch({
  args: ['--use-fake-device-for-media-stream', '--use-fake-ui-for-media-stream'],
})
const ctx = await b.newContext({ viewport: { width: 1440, height: 950 }, permissions: ['camera', 'microphone'] })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []

const bodyText = () => page.evaluate(() => document.body.innerText)
const rawKeys = (txt) => (txt.match(/\b(video|shared|common)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)

try {
  await page.goto(`${BASE}/#/video`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4500)
  await page.getByRole('button', { name: /Anrufverlauf/ }).click()
  await page.waitForTimeout(1000)
  await page.screenshot({ path: resolve(outDir, '1-history.png') })

  // 1) Row click → detail modal (Anna Müller: notes + recording + company + phone)
  await page.getByRole('button', { name: /Anna Müller/ }).first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, '2-detail-recording.png') })
  const d1 = await page.evaluate(() => document.querySelector('[role="dialog"]')?.textContent || '')
  out.push({
    step: 'detail: recording call',
    hasRecording: /Aufzeichnung verfügbar/.test(d1),
    hasCallBack: /Zurückrufen/.test(d1),
    hasDownload: /Protokoll herunterladen/.test(d1),
    hasNote: /SEO-Paket/.test(d1),
    pass: /Aufzeichnung verfügbar/.test(d1) && /Zurückrufen/.test(d1) && /Protokoll herunterladen/.test(d1) && /SEO-Paket/.test(d1),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 2) Row click → detail modal (Meier AG: participants)
  await page.getByRole('button', { name: /Meier AG/ }).first().click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(500)
  await page.screenshot({ path: resolve(outDir, '3-detail-participants.png') })
  const d2 = await page.evaluate(() => document.querySelector('[role="dialog"]')?.textContent || '')
  out.push({
    step: 'detail: participants',
    hasParticipants: /Teilnehmer/.test(d2),
    hasNames: /Peter Koch/.test(d2) && /Michael Berg/.test(d2),
    pass: /Teilnehmer/.test(d2) && /Peter Koch/.test(d2),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 3) Settings tab → device controls + camera placeholder (idle)
  await page.getByRole('button', { name: /Einstellungen/ }).first().click()
  await page.waitForTimeout(800)
  await page.screenshot({ path: resolve(outDir, '4-settings-idle.png') })
  const startBtn = page.getByRole('button', { name: /Vorschau starten/ })
  const hasStart = (await startBtn.count()) > 0
  out.push({ step: 'settings: device controls + preview button', hasStart, pass: hasStart })

  // 4) Start camera preview → active <video> (fake device)
  if (hasStart) {
    await startBtn.click()
    await page.waitForTimeout(1500)
    await page.screenshot({ path: resolve(outDir, '5-settings-preview-active.png') })
    const active = await page.evaluate(() => {
      const v = document.querySelector('video')
      return !!v && !v.classList.contains('invisible') && v.readyState >= 2
    })
    out.push({ step: 'settings: live preview active', active, pass: active })
  }

  // Raw-key scan across the visited surfaces
  const leaks = rawKeys(await bodyText())
  out.push({ step: 'no raw i18n keys', leaks, pass: leaks.length === 0 })
} catch (e) {
  out.push({ step: 'error', pass: false, error: String(e).split('\n')[0] })
}

out.push({ step: 'pageerrors', errors: errs.slice(0, 8), pass: errs.length === 0 })
await ctx.close(); await b.close()
const allPass = out.every((s) => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== VIDEO-TIEFE QA: ${allPass ? 'ALL PASS' : 'REVIEW'} ===`)
out.forEach((s) => console.log(`  ${s.pass ? '✓' : '✗'} ${s.step}`))
