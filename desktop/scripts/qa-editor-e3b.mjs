/**
 * QA — Modul-Editor E-3b (Wertelisten-Panel).
 *
 * Helpdesk-Editor → Wertelisten → ticket_priority: Option umbenennen (Input +
 * In-Panel-Vorschau-Chip + Zähler + Entwurf-Badge), Sichtbarkeit togglen, Reset.
 *
 * Screenshots → .qa-screenshots/editor-e3b/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/editor-e3b')
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

  // Open Wertelisten
  await page.locator('nav button').filter({ hasText: 'Wertelisten' }).first().click()
  await wait(600)
  await shot('01-wertelisten-panel.png')
  {
    const txt = await bodyText()
    // ticket_priority resolved: Rückfrage (tenant), Mittel, Hoch, Kritisch(hidden)
    const setShown = /Rückfrage/.test(txt) && /Hoch/.test(txt)
    const previewShown = /Vorschau/.test(txt)
    const inputs = await page.locator('aside input').count()
    out.push({ step: 'S1 Wertelisten-Panel: Set + Optionen + Vorschau', setShown, previewShown, inputs, pass: setShown && previewShown && inputs >= 4 })
  }

  // Rename option "Hoch" (aside inputs: 0=setname,1=Rückfrage,2=Mittel,3=Hoch,4=Kritisch)
  const optHoch = page.locator('aside input').nth(3)
  const beforeVal = await optHoch.inputValue()
  await optHoch.fill('Dringend')
  await wait(500)
  await shot('02-option-renamed.png')
  {
    const txt = await bodyText()
    out.push({
      step: 'S2 Option umbenannt „Hoch"→„Dringend" → Vorschau-Chip + Zähler',
      beforeVal,
      chipUpdated: /Dringend/.test(txt),
      counterOn: /1 Änderung/.test(txt),
      draftBadge: /Entwurf/.test(txt),
      pass: /Dringend/.test(txt) && /1 Änderung/.test(txt) && /Entwurf/.test(txt),
    })
  }

  // Reset the set
  const reset = page.locator('aside button[aria-label="Zurücksetzen"]').first()
  if ((await reset.count()) > 0) {
    await reset.click()
    await wait(500)
    await shot('03-after-reset.png')
    const txt = await bodyText()
    out.push({ step: 'S3 Reset → „Hoch" zurück, keine Änderungen', pass: /Hoch/.test(txt) && !/Dringend/.test(txt) && /Keine Änderungen/.test(txt) })
  } else {
    out.push({ step: 'S3 Reset-Button nicht gefunden', pass: false })
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
console.log(`\n=== QA Modul-Editor E-3b — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
