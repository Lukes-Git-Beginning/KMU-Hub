/**
 * QA — Customization v1.2 Final Verification (2-Punkt-Beweis).
 *
 * Punkt 1: Bootstrap-Merge wirkt auf Sidebar
 *   → Sidebar zeigt "Patienten" für contacts (Vendor-Seed layout.navItems.contacts = Patienten)
 *
 * Punkt 2: Tenant-Live-Preview OHNE Reload
 *   → layout.navItems.contacts umbenennen zu "Klienten"
 *   → Sidebar wechselt SOFORT auf "Klienten" (via i18next overlay + changeLanguage re-render)
 *
 * Screenshots → .qa-screenshots/customization-v12-final/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

// Dev-Server: versuche zuerst 5174 (vite.qa.config), dann 5173/src/renderer
const BASE_5174 = 'http://localhost:5174'
const BASE_5173 = 'http://localhost:5173/src/renderer'
let BASE = BASE_5174

const outDir = resolve('.qa-screenshots/customization-v12-final')
await mkdir(outDir, { recursive: true })

// Electron-API stub + Onboarding-Skip + Launch-Skip
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

// Detect working base URL
{
  let detected = false
  try {
    const r = await fetch(BASE_5174)
    if (r.ok || r.status < 500) { BASE = BASE_5174; detected = true }
  } catch {}
  if (!detected) {
    try {
      const r = await fetch(BASE_5173 + '/index.html')
      if (r.ok) { BASE = BASE_5173; detected = true }
    } catch {}
  }
  console.log(`Using base: ${BASE}`)
}

const b = await chromium.launch({ headless: true })
const ctx = await b.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []

const bodyText = () => page.evaluate(() => document.body.innerText)
const navText = () => page.evaluate(() => {
  const nav = document.querySelector('nav, [data-tour="sidebar"], aside')
  return nav ? nav.innerText : ''
})
const shot = (name) => page.screenshot({ path: resolve(outDir, name), fullPage: false })
const wait = (ms) => page.waitForTimeout(ms)
const waitForText = (x, timeout = 20000) =>
  page.waitForFunction((t) => document.body.innerText.includes(t), x, { timeout }).catch(() => {})

try {
  // ── SETUP: Login / Demo-Mode ─────────────────────────────────────────────────
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await wait(2000)

  // Falls Login-Screen → Demo-Login klicken
  const loginText = await bodyText()
  if (/Anmelden|Login|Sign in/i.test(loginText) && !/Dashboard/i.test(loginText)) {
    // Versuche Demo-Button
    const demoBtn = page.locator('button').filter({ hasText: /Demo|Continue|Fortfahren/i }).first()
    if (await demoBtn.count() > 0) {
      await demoBtn.click()
      await wait(2000)
    } else {
      // Standard-Login mit admin
      const emailInput = page.locator('input[type="email"], input[name="email"]').first()
      const pwInput = page.locator('input[type="password"]').first()
      if (await emailInput.count() > 0) {
        await emailInput.fill('admin@example.com')
        await pwInput.fill('password')
        await page.locator('button[type="submit"]').click()
        await wait(2000)
      }
    }
  }

  // ── PUNKT 1: Bootstrap-Merge wirkt auf Sidebar ───────────────────────────────
  // Gehe zur Hauptseite und prüfe die Sidebar
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await wait(3000)

  let sidebarTxt = await navText()
  let fullTxt = await bodyText()
  await shot('01-sidebar-initial.png')

  // Suche "Patienten" (Vendor-Override aktiv) vs "Kontakte" (kein Merge)
  const hasPatienten = /Patienten/.test(sidebarTxt) || /Patienten/.test(fullTxt.slice(0, 2000))
  const hasKontakte = /Kontakte/.test(sidebarTxt) || /Kontakte/.test(fullTxt.slice(0, 2000))

  // Auch in der ganzen Seite (Sidebar kann in aside/nav sein)
  const sidebarEl = page.locator('[data-tour="sidebar"], aside').first()
  const sidebarElText = await sidebarEl.innerText().catch(() => sidebarTxt)
  const patientenInSidebar = /Patienten/.test(sidebarElText)
  const kontakteInSidebar = /Kontakte/.test(sidebarElText)

  out.push({
    step: 'P1 Bootstrap-Merge: Sidebar zeigt "Patienten" (Vendor-Override layout.navItems.contacts)',
    patientenInSidebar,
    kontakteInSidebar,
    sidebarSnippet: sidebarElText.slice(0, 400),
    pass: patientenInSidebar,
    verdict: patientenInSidebar
      ? 'PASS — Vendor-Seed wirkt auf Sidebar'
      : kontakteInSidebar
        ? 'FAIL — Sidebar zeigt "Kontakte" (Vendor-Merge greift nicht)'
        : 'FAIL — weder Patienten noch Kontakte gefunden (Sidebar leer / anders strukturiert)',
  })

  // ── PUNKT 2: Live-Preview ohne Reload ────────────────────────────────────────
  // Gehe zu Begriffe-Tab
  await page.goto(`${BASE}/#/admin/anpassungen/begriffe`, { waitUntil: 'domcontentloaded' })
  await waitForText('Begriffe', 20000)
  await wait(2500)

  await shot('02a-begriffe-initial.png')
  const begriffeInitial = await bodyText()

  // Welcher Wert steht aktuell für layout.navItems.contacts in der Liste?
  // Die Zeile zeigt den effectiveValue (Patienten wenn vendor aktiv)
  const currentContactsValue = (() => {
    // Suche nach dem Muster "Patienten ... Von Zentria" oder "Kontakte ... Standard"
    if (/Patienten/.test(begriffeInitial)) return 'Patienten (vendor)'
    if (/Kontakte/.test(begriffeInitial)) return 'Kontakte (default)'
    return 'unbekannt'
  })()

  out.push({
    step: 'P2-Setup: Aktueller Wert für layout.navItems.contacts im Editor',
    currentContactsValue,
    pass: true,
    note: 'Info-Schritt, kein Pass/Fail',
  })

  // Screenshot der Sidebar VOR dem Edit (damit wir den Vorher-Zustand festhalten)
  await shot('02b-sidebar-before-edit.png')

  // Klick auf Edit-Button für layout.navItems.contacts
  const navContactsEditBtn = page.locator('[aria-label="layout.navItems.contacts bearbeiten"]')
  const navContactsBtnCount = await navContactsEditBtn.count()

  if (navContactsBtnCount > 0) {
    // Scroll zur Navigation-Gruppe (unten in der Liste)
    await navContactsEditBtn.scrollIntoViewIfNeeded()
    await wait(300)
    await navContactsEditBtn.click()
    await wait(700)
    await shot('02c-edit-open.png')

    const navInput = page.locator('input[placeholder]').first()
    const navInputVisible = await navInput.isVisible().catch(() => false)

    if (navInputVisible) {
      await navInput.fill('Klienten')
      await wait(400)
      await shot('02d-draft-filled.png')

      // Speichern
      const saveBtn = page.locator('button').filter({ hasText: /Speichern|Save/i }).last()
      await saveBtn.click()

      // Warte auf Live-Preview (React Query invalidate → fetch → applyLabelOverlay → i18n.changeLanguage re-render)
      await wait(3000)

      const txtAfter = await bodyText()
      await shot('02e-after-save.png')

      // Prüfe die Sidebar (data-tour="sidebar" oder aside)
      const sidebarAfterEl = page.locator('[data-tour="sidebar"], aside').first()
      const sidebarAfterText = await sidebarAfterEl.innerText().catch(() => '')
      await shot('02f-sidebar-post-save.png')

      const klientenInSidebar = /Klienten/.test(sidebarAfterText)
      const patientenGoneFromSidebar = !/Patienten/.test(sidebarAfterText)

      out.push({
        step: 'P2 Live-Preview: Sidebar zeigt "Klienten" OHNE Reload nach Speichern',
        klientenInSidebar,
        patientenGoneFromSidebar,
        sidebarAfterSnippet: sidebarAfterText.slice(0, 400),
        pass: klientenInSidebar,
        verdict: klientenInSidebar
          ? 'PASS — Live-Preview wirkt: Sidebar zeigt sofort "Klienten"'
          : patientenGoneFromSidebar
            ? 'PARTIAL — "Patienten" weg, aber "Klienten" noch nicht in Sidebar-Element (prüfe Screenshot)'
            : 'FAIL — Sidebar zeigt noch "Patienten" (kein Live-Preview)',
      })
    } else {
      out.push({
        step: 'P2 SETUP FAIL: Edit-Button geklickt, aber kein Input erschienen',
        pass: false,
      })
      await shot('02-fail-no-input.png')
    }
  } else {
    // Fallback: Scrolle zur Navigation-Gruppe und nutze data-qa-Buttons
    await waitForText('Navigation', 5000)
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
    await wait(800)
    await shot('02-nav-scrolled.png')

    const allEditBtns = page.locator('[data-qa="label-edit-btn"]')
    const editCount = await allEditBtns.count()

    out.push({
      step: 'P2 SETUP: aria-label-Button nicht gefunden, Fallback',
      editCount,
      note: 'Kein Button mit aria-label "layout.navItems.contacts bearbeiten" — Key evtl. anders',
      pass: false,
    })
  }

} catch (e) {
  out.push({ step: 'FATAL', error: String(e), stack: String(e.stack).slice(0, 500), pass: false })
  await shot('fatal-error.png').catch(() => {})
} finally {
  await b.close()
}

// ── Ergebnis ──────────────────────────────────────────────────────────────────

const total = out.length
const passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Customization v1.2 Final — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  const icon = s.pass ? 'PASS' : (s.step.includes('Info') ? 'INFO' : 'FAIL')
  console.log(`${icon}  ${s.step}`)
  const { step: _, pass: __, ...details } = s
  console.log('     ', JSON.stringify(details, null, 2).replace(/\n/g, '\n      '))
})

if (errs.length > 0) {
  console.log('\nPage-Errors:')
  errs.forEach((e) => console.log(' ', e))
}

console.log('\nScreenshots:')
const pngNames = [
  '01-sidebar-initial.png',
  '02a-begriffe-initial.png',
  '02b-sidebar-before-edit.png',
  '02c-edit-open.png',
  '02d-draft-filled.png',
  '02e-after-save.png',
  '02f-sidebar-post-save.png',
]
pngNames.forEach((f) => console.log(' ', resolve(outDir, f)))
