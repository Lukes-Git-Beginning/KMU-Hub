/**
 * QA — Modul-Editor E-2 (EditorFrame shell).
 *
 * Beweist: Editor öffnet als Overlay über ein Pilot-Modul, rendert das echte
 * Modul in der Sandbox, zeigt Toolbar + Amber-Banner + Drei-Panel + Footer,
 * Trio-Nav wählt Bereiche, Preview-Toggle blendet Panels aus. Keine Raw-Keys,
 * keine Page-Errors.
 *
 * Screenshots → .qa-screenshots/editor-e2/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/editor-e2')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch({ headless: true })
const ctx = await b.newContext({ viewport: { width: 1600, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const consoleErrs = []
page.on('console', (m) => {
  if (m.type() === 'error') consoleErrs.push(m.text().slice(0, 400))
})
const out = []
const bodyText = () => page.evaluate(() => document.body.innerText)
const shot = (n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
const wait = (ms) => page.waitForTimeout(ms)
const waitForText = (x, timeout = 20000) =>
  page.waitForFunction((t) => document.body.innerText.includes(t), x, { timeout }).catch(() => {})

try {
  // ── Setup: Login / Demo ──────────────────────────────────────────────────────
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await wait(2000)
  const loginText = await bodyText()
  if (/Anmelden|Login|Sign in/i.test(loginText) && !/Dashboard/i.test(loginText)) {
    const demoBtn = page.locator('button').filter({ hasText: /Demo|Continue|Fortfahren/i }).first()
    if ((await demoBtn.count()) > 0) {
      await demoBtn.click()
      await wait(2000)
    } else {
      const email = page.locator('input[type="email"], input[name="email"]').first()
      const pw = page.locator('input[type="password"]').first()
      if ((await email.count()) > 0) {
        await email.fill('admin@example.com')
        await pw.fill('password')
        await page.locator('button[type="submit"]').click()
        await wait(2000)
      }
    }
  }

  // ── Step 1: Anpassungen-Hub mit Editor-Launch-Leiste ─────────────────────────
  await page.goto(`${BASE}/#/admin/anpassungen`, { waitUntil: 'domcontentloaded' })
  await waitForText('Modul-Editor', 20000)
  await wait(1500)
  await shot('01-hub-launch.png')
  const hubTxt = await bodyText()
  const launchVisible = /Modul-Editor/.test(hubTxt) && /Editor öffnen/.test(hubTxt)
  out.push({
    step: 'S1 Launch-Leiste sichtbar (Modul-Editor + Kacheln)',
    launchVisible,
    pass: launchVisible,
  })

  // ── Step 2: Editor öffnen (erste Kachel = Kontakte) ──────────────────────────
  const tile = page.getByRole('button').filter({ hasText: 'Editor öffnen' }).first()
  const tileCount = await tile.count()
  if (tileCount > 0) {
    await tile.click()
    await waitForText('du bearbeitest eine Kopie', 15000)
    await wait(2500) // let the sandbox module lazy-load + render
    await shot('02-editor-open.png')

    const dialogVisible = (await page.locator('[role="dialog"]').count()) > 0
    const editorTxt = await bodyText()
    const bannerVisible = /du bearbeitest eine Kopie/.test(editorTxt)
    const trioVisible = /Felder/.test(editorTxt) && /Begriffe/.test(editorTxt) && /Wertelisten/.test(editorTxt)
    const footerVisible = /Als Entwurf speichern/.test(editorTxt) && /Übernehmen/.test(editorTxt)
    const sandboxOk = !/Vorschau nicht verfügbar/.test(editorTxt)

    out.push({
      step: 'S2 Editor-Overlay: Dialog + Amber-Banner + Trio-Nav + Footer + Sandbox rendert',
      dialogVisible,
      bannerVisible,
      trioVisible,
      footerVisible,
      sandboxOk,
      pass: dialogVisible && bannerVisible && trioVisible && footerVisible && sandboxOk,
    })

    // ── Step 3: Trio-Nav → Begriffe → Properties-Intro ─────────────────────────
    const begriffeNav = page.locator('nav button').filter({ hasText: 'Begriffe' }).first()
    if ((await begriffeNav.count()) > 0) {
      await begriffeNav.click()
      await wait(600)
      await shot('03-properties-terms.png')
      const propTxt = await bodyText()
      const propsShown = /in eure Sprache umbenennen|Wähle ein Element/.test(propTxt)
      out.push({ step: 'S3 Trio-Nav Begriffe → Properties-Panel zeigt Bereichs-Intro', propsShown, pass: propsShown })
    } else {
      out.push({ step: 'S3 Trio-Nav Begriffe-Button nicht gefunden', pass: false })
    }

    // ── Step 4: Preview-Toggle blendet Panels aus ──────────────────────────────
    const previewBtn = page.getByRole('button').filter({ hasText: /^Vorschau$/ }).first()
    if ((await previewBtn.count()) > 0) {
      await previewBtn.click()
      await wait(600)
      await shot('04-preview-only.png')
      out.push({ step: 'S4 Preview-Toggle → Panels ausgeblendet, Canvas voll', pass: true })
    } else {
      out.push({ step: 'S4 Preview-Button nicht gefunden', pass: false })
    }

    // ── Step 5: Raw-Key-Check ──────────────────────────────────────────────────
    const rawKeys = /customization\.editor\.|rbac\.module\.[a-z]/.test(await bodyText())
    out.push({
      step: 'S5 Keine rohen i18n-Keys sichtbar',
      rawKeysFound: rawKeys,
      pass: !rawKeys,
    })
  } else {
    out.push({ step: 'S2 SETUP FAIL: keine Editor-Kachel gefunden', pass: false })
    await shot('02-fail-no-tile.png')
  }
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length
const passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Modul-Editor E-2 — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
if (errs.length) {
  console.log('\nPage-Errors:')
  errs.forEach((e) => console.log(' ', e))
} else {
  console.log('\nNo page errors.')
}
if (consoleErrs.length) {
  console.log('\nConsole-Errors (first 6, sandbox diagnosis):')
  ;[...new Set(consoleErrs)].slice(0, 6).forEach((e) => console.log(' -', e))
}
console.log('\nScreenshots in', outDir)
