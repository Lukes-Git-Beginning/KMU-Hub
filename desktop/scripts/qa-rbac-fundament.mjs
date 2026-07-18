/**
 * QA — RBAC R-1 Fundament.
 * Verifies: level-1 nav gating per role (default-deny), profile switcher with
 * the 7 presets + multi-role combo, permissions reload on switch, effective-
 * rights view (scope badges, multi-role provenance), settings overlay gating,
 * raw keys + pageerrors.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/rbac-fundament')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []

const bodyText = () => page.evaluate(() => document.body.innerText)
const navText = async () => {
  const t = await page.evaluate(() =>
    Array.from(document.querySelectorAll('nav, aside')).map((n) => n.innerText).join(' '),
  )
  return t || bodyText()
}
const rawKeys = (txt) => (txt.match(/\b(rbac|profil|devTools|moduleSettings)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)
// Wait until a text appears (guards against slow cold-transform first loads).
const waitForText = (t) =>
  page.waitForFunction((x) => document.body.innerText.includes(x), t, { timeout: 30000 }).catch(() => {})

const openSwitcher = async () => {
  await page.locator('button.fixed.bottom-4.right-4').click()
  await page.waitForTimeout(400)
}
// The panel closes on backdrop click only (no Esc handler) — click mid-left.
const closeSwitcher = async () => {
  await page.mouse.click(600, 120)
  await page.waitForTimeout(300)
}
const switchTo = async (labelRe) => {
  const target = page.getByRole('button', { name: labelRe }).first()
  if (!(await target.isVisible().catch(() => false))) await openSwitcher()
  await target.click()
  await page.waitForTimeout(1200)
  await closeSwitcher()
}

try {
  // 1) Admin default: full nav (finance + infrastructure visible)
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await waitForText('Buchhaltung')
  await page.waitForTimeout(800)
  const navAdmin = await navText()
  await page.screenshot({ path: resolve(outDir, '01-admin-nav.png') })
  // (Infrastruktur wird vom Business-Profil-Filter gehalten — kein RBAC-Thema)
  out.push({
    step: 'admin: full nav (Buchhaltung + Team + Kontakte sichtbar)',
    finance: /Buchhaltung/.test(navAdmin),
    team: /\bTeam\b/.test(navAdmin),
    contacts: /Kontakte/.test(navAdmin),
    pass: /Buchhaltung/.test(navAdmin) && /\bTeam\b/.test(navAdmin) && /Kontakte/.test(navAdmin),
  })

  // 2) Profile switcher: 8 demo identities incl. multi-role combo
  await openSwitcher()
  await page.waitForTimeout(400)
  await page.screenshot({ path: resolve(outDir, '02-switcher.png') })
  const swTxt = await bodyText()
  out.push({
    step: 'switcher: 7 presets + Kombi (Teamleiter + HR-Admin)',
    hasExtern: /Aushilfe \/ Extern/.test(swTxt),
    hasReadonly: /Nur Lesen/.test(swTxt),
    hasCombo: /Teamleiter \+ HR-Admin/.test(swTxt),
    rawKeys: rawKeys(swTxt),
    pass: /Aushilfe \/ Extern/.test(swTxt) && /Nur Lesen/.test(swTxt) && /Teamleiter \+ HR-Admin/.test(swTxt) && rawKeys(swTxt).length === 0,
  })

  // 3) Switch to Aushilfe/Extern → nav shrinks to dashboard/work/documents(+settings/notifications)
  await switchTo(/Aushilfe \/ Extern/)
  await waitForText('Projekte')
  const navExtern = await navText()
  await page.screenshot({ path: resolve(outDir, '03-extern-nav.png') })
  // (Dokumente/Wiki fehlen in JEDER Rolle im frischen Kontext — vorbestehender
  //  Optional-Module-Filter des Business-Profils, kein RBAC-Thema.)
  out.push({
    step: 'extern: default-deny nav (keine Buchhaltung/Kontakte/Team/Kalender)',
    noFinance: !/Buchhaltung/.test(navExtern),
    noContacts: !/Kontakte/.test(navExtern),
    noTeam: !/\bTeam\b/.test(navExtern),
    noCalendar: !/Kalender/.test(navExtern),
    hasWork: /Projekte/.test(navExtern) && /Aufgaben/.test(navExtern),
    pass: !/Buchhaltung/.test(navExtern) && !/Kontakte/.test(navExtern) && !/\bTeam\b/.test(navExtern) && /Projekte/.test(navExtern),
  })

  // 4) Extern: effective-rights view (few grants, scope badges, no finance module)
  await page.goto(`${BASE}/#/profil`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  await page.getByRole('tab', { name: 'Berechtigungen' }).click()
  await page.waitForTimeout(900)
  const rightsExtern = await bodyText()
  await page.screenshot({ path: resolve(outDir, '04-extern-rechte.png') })
  out.push({
    step: 'extern: Effektive Rechte (klein, Scope-Badges, kein Herkunfts-Badge)',
    title: /Effektive Rechte/.test(rightsExtern),
    hasBeAssigned: /Zugewiesen bekommen/.test(rightsExtern),
    scopeBadge: /\bTeam\b/.test(rightsExtern),
    noFinance: !/Buchhaltung/.test(rightsExtern.split('Effektive Rechte')[1] ?? rightsExtern),
    rawKeys: rawKeys(rightsExtern),
    pass: /Effektive Rechte/.test(rightsExtern) && /Zugewiesen bekommen/.test(rightsExtern) && rawKeys(rightsExtern).length === 0,
  })

  // 5) Extern: settings overlay — module entries follow module visibility,
  //    admin entries hidden entirely
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  await page.getByText(/Modul-Einstell/).first().click()
  await page.waitForTimeout(900)
  const ovl = await bodyText()
  await page.screenshot({ path: resolve(outDir, '05-extern-settings-overlay.png') })
  // Overlay entries are exact-name buttons; the dashboard cards behind the
  // overlay carry extra text (counts) so exact matching isolates the overlay.
  const crmEntryCount = await page.getByRole('button', { name: 'Kontakte', exact: true }).count()
  const docsEntryCount = await page.getByRole('button', { name: 'Dokumente', exact: true }).count()
  out.push({
    step: 'extern: settings overlay (nur sichtbare Module, keine Admin-Einträge)',
    overlayOpen: /Modul-Einstellungen/.test(ovl),
    hasDocsEntry: docsEntryCount > 0,
    noCrmEntry: crmEntryCount === 0,
    noBranding: !/Branding/.test(ovl),
    noSecurity: !/Sicherheit\b/.test(ovl),
    pass: /Modul-Einstellungen/.test(ovl) && docsEntryCount > 0 && crmEntryCount === 0 && !/Branding/.test(ovl),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)

  // 6) Multi-role combo (Laura): rights view shows both role chips + provenance badges
  await switchTo(/Teamleiter \+ HR-Admin/)
  await page.goto(`${BASE}/#/profil`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  await page.getByRole('tab', { name: 'Berechtigungen' }).click()
  await page.waitForTimeout(900)
  const rightsCombo = await bodyText()
  await page.screenshot({ path: resolve(outDir, '06-kombi-rechte.png') })
  const provenanceCount = await page.evaluate(() =>
    document.body.innerText.match(/HR-Admin/g)?.length ?? 0,
  )
  out.push({
    step: 'kombi: beide Rollen-Chips + Herkunfts-Badges (Union-Auflösung)',
    bothRoles: /Teamleiter/.test(rightsCombo) && /HR-Admin/.test(rightsCombo),
    provenanceBadges: provenanceCount > 2,
    hasSalary: /Gehaltsdaten/.test(rightsCombo),
    pass: /Teamleiter/.test(rightsCombo) && /HR-Admin/.test(rightsCombo) && provenanceCount > 2 && /Gehaltsdaten/.test(rightsCombo),
  })

  // 7) Combo (hr_admin): team nav visible, finance still hidden
  const navCombo = await navText()
  out.push({
    step: 'kombi: Team sichtbar, Buchhaltung bleibt versteckt',
    hasTeam: /\bTeam\b/.test(navCombo),
    noFinance: !/Buchhaltung/.test(navCombo),
    pass: /\bTeam\b/.test(navCombo) && !/Buchhaltung/.test(navCombo),
  })

  // 8) Back to admin: full nav restored (roundtrip, store refetch works)
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1200)
  await switchTo(/Vollzugriff/)
  await waitForText('Buchhaltung')
  const navBack = await navText()
  await page.screenshot({ path: resolve(outDir, '07-admin-restored.png') })
  out.push({
    step: 'roundtrip: admin wiederhergestellt (Buchhaltung + Team + Kontakte zurück)',
    pass: /Buchhaltung/.test(navBack) && /\bTeam\b/.test(navBack) && /Kontakte/.test(navBack),
  })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).split('\n')[0], pass: false })
  await page.screenshot({ path: resolve(outDir, '99-fatal.png') }).catch(() => {})
}

console.log(JSON.stringify({ steps: out, pageErrors: errs.slice(0, 5), allPass: out.every((s) => s.pass) }, null, 2))
await b.close()
process.exit(out.every((s) => s.pass) && errs.length === 0 ? 0 : 1)
