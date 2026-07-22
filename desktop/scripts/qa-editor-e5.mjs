/**
 * QA — Modul-Editor E-5 (Deploy-Dialog).
 *
 * Änderung machen → „Übernehmen" → Dialog (Zusammenfassung + Jetzt/Terminiert +
 * Termin-Picker + Ankündigung) → „Jetzt übernehmen" schließt den Editor.
 *
 * Screenshots → .qa-screenshots/editor-e5/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/editor-e5')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch({ headless: true })
const ctx = await b.newContext({ viewport: { width: 1280, height: 900 } })
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
  await wait(2000)
  const lt = await bodyText()
  if (/Anmelden|Login|Sign in/i.test(lt) && !/Dashboard/i.test(lt)) {
    const d = page.locator('button').filter({ hasText: /Demo|Continue|Fortfahren/i }).first()
    if ((await d.count()) > 0) { await d.click(); await wait(2000) }
  }

  await page.goto(`${BASE}/#/editor-window?module=helpdesk`, { waitUntil: 'domcontentloaded' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2000)

  // Make a change (Begriffe → rename first label) so isDirty
  await page.locator('nav button').filter({ hasText: 'Begriffe' }).first().click()
  await wait(500)
  await page.locator('aside input').first().fill('Support-Center')
  await wait(500)

  // Open deploy dialog (dispatchEvent: footer button is overlapped by the dev
  // ProfileSwitcher, so a coordinate click lands on the wrong element)
  await page.getByRole('button', { name: /Übernehmen/ }).first().dispatchEvent('click')
  await waitForText('Änderungen übernehmen', 8000)
  await wait(400)
  await shot('01-deploy-now.png')
  {
    const txt = await bodyText()
    out.push({
      step: 'S1 Deploy-Dialog: Titel + Zusammenfassung + Modi + Ankündigung',
      title: /Änderungen übernehmen/.test(txt),
      summary: /1 Änderung/.test(txt),
      affects: /Betrifft alle Nutzer/.test(txt),
      modes: /Jetzt/.test(txt) && /Terminiert/.test(txt),
      announce: /Ankündigung/.test(txt),
      pass: /Änderungen übernehmen/.test(txt) && /1 Änderung/.test(txt) && /Jetzt/.test(txt) && /Terminiert/.test(txt),
    })
  }

  // Switch to Terminiert → datetime picker
  await page.getByRole('button', { name: /Terminiert/ }).first().dispatchEvent('click')
  await wait(400)
  await shot('02-deploy-scheduled.png')
  {
    const dt = await page.locator('input[type="datetime-local"]').count()
    const confirmScheduled = /Rollout planen/.test(await bodyText())
    out.push({ step: 'S2 Terminiert → Termin-Picker + „Rollout planen"', datetimeInputs: dt, confirmScheduled, pass: dt >= 1 && confirmScheduled })
  }

  // Back to Jetzt (mode button's name includes its hint), then deploy.
  // NOTE: in the QA browser the electronAPI stub makes window.close() a no-op
  // (real Electron closes the window), so we verify the deploy fired via its
  // success toast rather than the editor-close navigation.
  await page.getByRole('button', { name: /Sofort für alle live/ }).first().dispatchEvent('click')
  await wait(300)
  await page.getByRole('button', { name: /Jetzt übernehmen/ }).first().dispatchEvent('click')
  await wait(400)
  await shot('03-after-deploy.png')
  {
    const txt = await bodyText()
    out.push({
      step: 'S3 „Jetzt übernehmen" → Deploy feuert (Erfolgs-Toast)',
      appliedToast: /übernommen/.test(txt),
      pass: /übernommen/.test(txt),
    })
  }

  const raw = /customization\.editor\.[a-z]|customization\.labels\.provenance\.[a-z]/.test(await bodyText())
  out.push({ step: 'S4 Keine rohen Keys', rawKeysFound: raw, pass: !raw })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Modul-Editor E-5 — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
