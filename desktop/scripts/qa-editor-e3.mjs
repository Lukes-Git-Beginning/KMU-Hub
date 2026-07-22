/**
 * QA — Modul-Editor E-3 (Begriffe-Panel + Live-Vorschau).
 *
 * Der Kernbeweis der Pipeline: im Helpdesk-Editor-Fenster den Modul-Begriff
 * umbenennen → Sandbox-Header UND Toolbar-Titel aktualisieren live → Footer
 * aktiviert sich (Zähler + Buttons) → Reset stellt wieder her.
 *
 * Screenshots → .qa-screenshots/editor-e3/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/editor-e3')
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
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await wait(2000)
  const loginText = await bodyText()
  if (/Anmelden|Login|Sign in/i.test(loginText) && !/Dashboard/i.test(loginText)) {
    const demoBtn = page.locator('button').filter({ hasText: /Demo|Continue|Fortfahren/i }).first()
    if ((await demoBtn.count()) > 0) { await demoBtn.click(); await wait(2000) }
  }

  // Open Helpdesk editor window
  await page.goto(`${BASE}/#/editor-window?module=helpdesk`, { waitUntil: 'domcontentloaded' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2500)
  await shot('01-editor-open.png')

  // Open Begriffe section
  const begriffeNav = page.locator('nav button').filter({ hasText: 'Begriffe' }).first()
  await begriffeNav.click()
  await wait(600)
  await shot('02-begriffe-panel.png')
  const panelInputs = page.locator('aside input')
  const inputCount = await panelInputs.count()
  out.push({ step: 'S1 Begriffe-Panel rendert Label-Inputs', inputCount, pass: inputCount >= 2 })

  // Rename the module label (first input = rbac.module.helpdesk = "Helpdesk")
  const firstInput = panelInputs.first()
  const before = await firstInput.inputValue()
  await firstInput.fill('Support-Center')
  await wait(600) // debounce 120ms + re-render
  await shot('03-live-preview.png')
  {
    const txt = await bodyText()
    // Sandbox module header + toolbar both read t('rbac.module.helpdesk')
    const previewUpdated = /Support-Center/.test(txt)
    const toolbarUpdated = /Support-Center bearbeiten/.test(txt)
    const counterOn = /1 Änderung/.test(txt)
    const draftBadge = /Entwurf/.test(txt)
    const applyDisabled = await page.getByRole('button', { name: /Übernehmen/ }).isDisabled().catch(() => true)
    out.push({
      step: 'S2 Live-Vorschau: Begriff umbenannt → Header+Toolbar live, Footer aktiv',
      before, previewUpdated, toolbarUpdated, counterOn, draftBadge, applyEnabled: !applyDisabled,
      pass: previewUpdated && toolbarUpdated && counterOn && draftBadge && !applyDisabled,
    })
  }

  // Reset the row → back to "Helpdesk"
  const resetBtn = page.locator('aside button[aria-label="Zurücksetzen"]').first()
  if ((await resetBtn.count()) > 0) {
    await resetBtn.click()
    await wait(600)
    await shot('04-after-reset.png')
    const txt = await bodyText()
    out.push({
      step: 'S3 Reset → Begriff wieder „Helpdesk", keine Änderungen',
      helpdeskBack: /Helpdesk bearbeiten/.test(txt),
      noChanges: /Keine Änderungen/.test(txt),
      pass: /Helpdesk bearbeiten/.test(txt) && /Keine Änderungen/.test(txt),
    })
  } else {
    out.push({ step: 'S3 Reset-Button nicht gefunden', pass: false })
  }

  // Raw-key check (only MY editor keys must be translated; mono key captions are intentional)
  const raw = /customization\.editor\.[a-z]|customization\.labels\.provenance\.[a-z]/.test(await bodyText())
  out.push({ step: 'S4 Keine rohen Editor-/Provenance-Keys', rawKeysFound: raw, pass: !raw })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length
const passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Modul-Editor E-3 — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
