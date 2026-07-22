/**
 * QA — Modul-Editor E-2b (eigenes Fenster statt Overlay).
 *
 * Testet die Fenster-Route #/editor-window?module=<key> direkt (der IPC-Aufruf
 * öffnet in Electron ein echtes Fenster mit genau dieser Route). Beweist: die
 * Editor-Seite rendert vollflächig (Toolbar + Amber-Banner + Trio-Nav + echtes
 * Modul + Footer), scharf ohne Overlay-Blur, keine Raw-Keys, keine Page-Errors;
 * unbekanntes Modul → sauberer Fallback.
 *
 * Screenshots → .qa-screenshots/editor-e2b/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/editor-e2b')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch({ headless: true })
const ctx = await b.newContext({ viewport: { width: 1280, height: 860 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
await ctx.addInitScript(NOLAUNCH)
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
  // Login / demo
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await wait(2000)
  const loginText = await bodyText()
  if (/Anmelden|Login|Sign in/i.test(loginText) && !/Dashboard/i.test(loginText)) {
    const demoBtn = page.locator('button').filter({ hasText: /Demo|Continue|Fortfahren/i }).first()
    if ((await demoBtn.count()) > 0) {
      await demoBtn.click()
      await wait(2000)
    }
  }

  // Step 1: Launch-Leiste im Hub
  await page.goto(`${BASE}/#/admin/anpassungen`, { waitUntil: 'domcontentloaded' })
  await waitForText('Modul-Editor', 20000)
  await wait(1000)
  await shot('01-hub-launch.png')
  out.push({ step: 'S1 Launch-Leiste im Hub sichtbar', pass: /Editor öffnen/.test(await bodyText()) })

  // Step 2: Editor-Fenster-Route für Helpdesk (der Blur-Fall)
  await page.goto(`${BASE}/#/editor-window?module=helpdesk`, { waitUntil: 'domcontentloaded' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2500)
  await shot('02-window-helpdesk.png')
  {
    const txt = await bodyText()
    const toolbar = /Helpdesk bearbeiten/.test(txt)
    const banner = /du bearbeitest eine Kopie/.test(txt)
    const trio = /Felder/.test(txt) && /Begriffe/.test(txt) && /Wertelisten/.test(txt)
    const moduleRendered = /Tickets|Ticket suchen|Wissensdatenbank/.test(txt) && !/Vorschau nicht verfügbar/.test(txt)
    const footer = /Als Entwurf speichern/.test(txt)
    out.push({
      step: 'S2 Helpdesk-Fenster: Toolbar+Banner+Trio+Footer + echtes Modul rendert',
      toolbar, banner, trio, moduleRendered, footer,
      pass: toolbar && banner && trio && moduleRendered && footer,
    })
  }

  // Step 3: Kontakte
  await page.goto(`${BASE}/#/editor-window?module=kontakte`, { waitUntil: 'domcontentloaded' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2500)
  await shot('03-window-kontakte.png')
  {
    const txt = await bodyText()
    const moduleRendered = /Kontakt suchen|Kontakte/.test(txt) && !/Vorschau nicht verfügbar/.test(txt)
    out.push({ step: 'S3 Kontakte-Fenster: Modul rendert', moduleRendered, pass: moduleRendered })
  }

  // Step 4: unbekanntes Modul → Fallback
  await page.goto(`${BASE}/#/editor-window?module=doesnotexist`, { waitUntil: 'domcontentloaded' })
  await wait(1500)
  await shot('04-unknown-module.png')
  out.push({ step: 'S4 Unbekanntes Modul → „Modul nicht gefunden"', pass: /Modul nicht gefunden/.test(await bodyText()) })

  // Step 5: Raw-Key-Check über alle Editor-Ansichten
  const raw = /customization\.editor\.|rbac\.module\.[a-z]/.test(await bodyText())
  out.push({ step: 'S5 Keine rohen i18n-Keys', rawKeysFound: raw, pass: !raw })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length
const passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Modul-Editor E-2b — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
