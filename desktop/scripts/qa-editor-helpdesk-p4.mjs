/**
 * QA — Helpdesk-Pilot P4 (Editor-Pivot): Chrome / Kontext-Inspektor.
 *
 * Beweist: der leere Properties-Panel-Zustand erklärt jetzt die Edit-in-place-
 * Interaktion (statt „wähle links…") — das Werkzeug ist selbsterklärend.
 *   S1 Editor offen → Panel zeigt „So passt du an" + 3 Schritte.
 *   S2 Keine rohen i18n-Keys, keine Page-Errors.
 *
 * Screenshots → .qa-screenshots/editor-helpdesk-p4/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/editor-helpdesk-p4')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch({ headless: true })
const ctx = await b.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []
const bodyText = () => page.evaluate(() => document.body.innerText)
const shot = (n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
const wait = (ms) => page.waitForTimeout(ms)
const waitForText = (x, timeout = 20000) =>
  page.waitForFunction((t) => document.body.innerText.includes(t), x, { timeout }).catch(() => {})

try {
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await wait(2500)
  await page.evaluate(() => { window.location.hash = '#/editor-window?module=helpdesk' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2500)
  await shot('01-inspector.png')
  {
    const txt = await bodyText()
    const hasTitle = /So passt du an/.test(txt)
    const hasStep1 = /um ihn umzubenennen/.test(txt)
    const hasStep2 = /Doppelklick/.test(txt)
    out.push({ step: 'S1 Kontext-Inspektor „So passt du an" + Schritte', hasTitle, hasStep1, hasStep2, pass: hasTitle && hasStep1 && hasStep2 })
  }

  const raw = /customization\.editor\.(inspector|props)\.[a-z]/i.test(await bodyText())
  out.push({ step: 'S2 Keine rohen Keys / Page-Errors', rawKeysFound: raw, errCount: errs.length, pass: !raw && errs.length === 0 })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Helpdesk-Pilot P4 — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
