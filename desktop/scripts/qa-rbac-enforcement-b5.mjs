/**
 * QA — RBAC R-3 Batch 5 (close-out: berichte · formulare · automatisierung +
 * standard-module mini catalogues kommunikation/kalender/zeiterfassung/
 * infrastructure).
 * Verifies: admin regression per module (tabs, header actions, no chip),
 * berichte editor edit-own + lifecycle/publish + share gating, KPI dashboard
 * follows module visibility (finance KPIs/chart hidden without finance view),
 * DATEV privacy split (readonly sees it, it_admin/member do not), member
 * automatisierung module removed (nav + deep link), zeiterfassung team/export
 * gating, kalender booking tab gating, extern NoAccess.
 * Raw keys + pageerrors tracked.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/rbac-enforcement-b5')
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
const rawKeys = (txt) => (txt.match(/\b(rbac|berichte|formulare|automatisierung|zeiterfassung|kalender|kommunikation|infrastructure)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)
const waitForText = (t) =>
  page.waitForFunction((x) => document.body.innerText.includes(x), t, { timeout: 30000 }).catch(() => {})
const shot = (name) => page.screenshot({ path: resolve(outDir, name) })
const btn = (name, exact = false) => page.getByRole('button', { name, exact })

// Panel state survives hash navigation (SPA) — check visibility, don't blind-toggle.
const switcherPanel = () => page.locator('div.max-h-80')
const setSwitcherOpen = async (open) => {
  const isOpen = await switcherPanel().isVisible().catch(() => false)
  if (isOpen !== open) {
    await page.locator('button.fixed.bottom-4.right-4').click()
    await page.waitForTimeout(400)
  }
}
// Switch only from /settings (user names are clickable rows elsewhere — b2 learning).
const switchTo = async (labelRe) => {
  await page.goto(`${BASE}/#/settings`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(900)
  await setSwitcherOpen(true)
  await switcherPanel().getByRole('button', { name: labelRe }).first().click()
  await page.waitForTimeout(1700)
  await setSwitcherOpen(false)
}

try {
  // ── 1) admin berichte: 4 Tabs + Neuer Bericht + Umsatz-KPI + kein Chip
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neuer Bericht')
  await page.waitForTimeout(1200)
  const berAdmin = await bodyText()
  const adminDatevTab = await btn('DATEV', true).count()
  const adminGeplantTab = await btn(/^Geplant \(/).count()
  await shot('01-admin-berichte-dashboard.png')
  out.push({
    step: 'admin berichte: Dashboard/Berichte/Geplant/DATEV-Tabs + Neuer Bericht + Umsatz-KPI + Umsatzverlauf, kein Chip',
    create: /Neuer Bericht/.test(berAdmin),
    datevTab: adminDatevTab > 0,
    geplantTab: adminGeplantTab > 0,
    financeKpi: /Umsatz \(MTD\)/.test(berAdmin),
    heroChart: /Umsatzverlauf/.test(berAdmin),
    noChip: !/Nur Ansicht|Eingeschränkt/.test(berAdmin),
    rawKeys: rawKeys(berAdmin),
    pass: /Neuer Bericht/.test(berAdmin) && adminDatevTab > 0 && adminGeplantTab > 0 && /Umsatz \(MTD\)/.test(berAdmin) && !/Nur Ansicht/.test(berAdmin) && rawKeys(berAdmin).length === 0,
  })

  // ── 2) admin berichte draft-editor: Bearbeiten-Toggle + Lifecycle + Teilen
  await btn('Berichte', true).click()
  await page.waitForTimeout(900)
  await btn(/Helpdesk-Auslastung KW 24/).first().click()
  await page.waitForTimeout(1200)
  const edAdmin = await bodyText()
  const adminEditToggle = await btn('Bearbeiten', true).count()
  const adminMarkFinal = await btn('Als fertig markieren').count()
  const adminShare = await btn('Teilen', true).count()
  await shot('02-admin-berichte-editor.png')
  out.push({
    step: 'admin berichte draft-editor: Bearbeiten-Toggle + Als fertig markieren + Teilen-Menü',
    edit: adminEditToggle > 0,
    markFinal: adminMarkFinal > 0,
    share: adminShare > 0,
    rawKeys: rawKeys(edAdmin),
    pass: adminEditToggle > 0 && adminMarkFinal > 0 && adminShare > 0 && rawKeys(edAdmin).length === 0,
  })

  // ── 3) admin formulare + automatisierung: Tabs + Creates
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neues Formular')
  await page.waitForTimeout(900)
  const formAdmin = await bodyText()
  await shot('03a-admin-formulare.png')
  await page.goto(`${BASE}/#/automatisierung`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neue Automatisierung')
  await page.waitForTimeout(900)
  const autoAdmin = await bodyText()
  await shot('03b-admin-automatisierung.png')
  out.push({
    step: 'admin formulare/automatisierung: alle Tabs + Neues Formular + Neue Automatisierung, kein Chip',
    formTabs: /Meine Formulare \(/.test(formAdmin) && /Eingänge \(/.test(formAdmin) && /Vorlagen \(/.test(formAdmin),
    formCreate: /Neues Formular/.test(formAdmin),
    autoTabs: /Meine Automatisierungen/.test(autoAdmin) && /Vorlagen/.test(autoAdmin) && /Protokoll/.test(autoAdmin),
    autoCreate: /Neue Automatisierung/.test(autoAdmin),
    noChips: !/Nur Ansicht/.test(formAdmin) && !/Nur Ansicht/.test(autoAdmin),
    rawKeys: [...rawKeys(formAdmin), ...rawKeys(autoAdmin)],
    pass: /Neues Formular/.test(formAdmin) && /Meine Formulare \(/.test(formAdmin) && /Neue Automatisierung/.test(autoAdmin) && /Protokoll/.test(autoAdmin) && rawKeys(formAdmin).length === 0 && rawKeys(autoAdmin).length === 0,
  })

  // ── 4) admin zeiterfassung: Team-View da + Export in Auswertungen
  await page.goto(`${BASE}/#/zeiterfassung`, { waitUntil: 'domcontentloaded' })
  await waitForText('Auswertungen')
  await page.waitForTimeout(900)
  const ztAdminTeamTab = await page.getByRole('tab', { name: 'Team', exact: true }).count()
  await page.getByRole('tab', { name: 'Auswertungen', exact: true }).click()
  await page.waitForTimeout(1100)
  const ztAdminAnalytics = await bodyText()
  const ztAdminExport = await btn('Export', true).count()
  await shot('04-admin-zeiterfassung-auswertungen.png')
  out.push({
    step: 'admin zeiterfassung: Team-View sichtbar + Export-Button in Auswertungen',
    teamTab: ztAdminTeamTab > 0,
    exportBtn: ztAdminExport > 0,
    rawKeys: rawKeys(ztAdminAnalytics),
    pass: ztAdminTeamTab > 0 && ztAdminExport > 0 && rawKeys(ztAdminAnalytics).length === 0,
  })

  // ── 5) it_admin: berichte OHNE DATEV/Umsatz (Finance-Privacy), automatisierung VOLL (IT-Domäne)
  await switchTo(/Thomas Keller/)
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const berIt = await bodyText()
  const itDatevTab = await btn('DATEV', true).count()
  await shot('05a-itadmin-berichte.png')
  await page.goto(`${BASE}/#/automatisierung`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neue Automatisierung')
  const autoIt = await bodyText()
  await shot('05b-itadmin-automatisierung.png')
  out.push({
    step: 'it_admin: berichte ohne DATEV-Tab + ohne Umsatz-KPI/Umsatzverlauf (kein finance view); automatisierung voll (Neue Automatisierung)',
    noDatev: itDatevTab === 0,
    noFinanceKpi: !/Umsatz \(MTD\)/.test(berIt),
    noHeroChart: !/Umsatzverlauf/.test(berIt),
    autoCreate: /Neue Automatisierung/.test(autoIt),
    rawKeys: [...rawKeys(berIt), ...rawKeys(autoIt)],
    pass: itDatevTab === 0 && !/Umsatz \(MTD\)/.test(berIt) && !/Umsatzverlauf/.test(berIt) && /Neue Automatisierung/.test(autoIt) && rawKeys(berIt).length === 0,
  })

  // ── 6) readonly berichte: Chip + DATEV DA (Steuerberater!) + Geplant WEG + kein Export
  await switchTo(/Elena Richter/)
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const berRo = await bodyText()
  const roDatevTab = await btn('DATEV', true).count()
  const roGeplantTab = await btn(/^Geplant \(/).count()
  await shot('06a-readonly-berichte.png')
  await btn('DATEV', true).click()
  await page.waitForTimeout(1400)
  const datevRo = await bodyText()
  await shot('06b-readonly-berichte-datev.png')
  out.push({
    step: 'readonly berichte: Chip, kein Neuer Bericht, Geplant-Tab WEG, DATEV-Tab DA (BWA sichtbar) aber DATEV Export WEG',
    chip: /Nur Ansicht/.test(berRo),
    noCreate: !/Neuer Bericht/.test(berRo),
    noGeplant: roGeplantTab === 0,
    datevTab: roDatevTab > 0,
    bwaVisible: /BWA/.test(datevRo),
    noExport: !/DATEV Export/.test(datevRo),
    rawKeys: [...rawKeys(berRo), ...rawKeys(datevRo)],
    pass: /Nur Ansicht/.test(berRo) && !/Neuer Bericht/.test(berRo) && roGeplantTab === 0 && roDatevTab > 0 && /BWA/.test(datevRo) && !/DATEV Export/.test(datevRo) && rawKeys(datevRo).length === 0,
  })

  // ── 7) readonly formulare + automatisierung: reads bleiben, Mutationen weg
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const formRo = await bodyText()
  await shot('07a-readonly-formulare.png')
  await page.goto(`${BASE}/#/automatisierung`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const autoRo = await bodyText()
  const roAutoVorlagen = await page.getByRole('tab', { name: 'Vorlagen' }).count()
  await shot('07b-readonly-automatisierung.png')
  out.push({
    step: 'readonly formulare/automatisierung: Listen sichtbar, kein Neues Formular / Neue Automatisierung, Vorlagen-Tab (Templates) WEG, Protokoll DA',
    formList: /Meine Formulare \(/.test(formRo),
    formNoCreate: !/Neues Formular/.test(formRo),
    autoNoCreate: !/Neue Automatisierung/.test(autoRo),
    autoNoTemplates: roAutoVorlagen === 0,
    autoLog: /Protokoll/.test(autoRo),
    rawKeys: [...rawKeys(formRo), ...rawKeys(autoRo)],
    pass: /Meine Formulare \(/.test(formRo) && !/Neues Formular/.test(formRo) && !/Neue Automatisierung/.test(autoRo) && roAutoVorlagen === 0 && /Protokoll/.test(autoRo) && rawKeys(autoRo).length === 0,
  })

  // ── 8) readonly zeiterfassung + kalender: Team/Export/Terminbuchung weg
  await page.goto(`${BASE}/#/zeiterfassung`, { waitUntil: 'domcontentloaded' })
  await waitForText('Auswertungen')
  await page.waitForTimeout(900)
  const ztRoTeamTab = await page.getByRole('tab', { name: 'Team', exact: true }).count()
  await page.getByRole('tab', { name: 'Auswertungen', exact: true }).click()
  await page.waitForTimeout(1100)
  const ztRoExport = await btn('Export', true).count()
  await shot('08a-readonly-zeiterfassung.png')
  await page.goto(`${BASE}/#/kalender`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const kalRoBooking = await btn('Terminbuchung').count()
  await shot('08b-readonly-kalender.png')
  out.push({
    step: 'readonly zeiterfassung: Team-View WEG + Export WEG; kalender: Terminbuchung-Tab WEG',
    noTeam: ztRoTeamTab === 0,
    noExport: ztRoExport === 0,
    noBookingTab: kalRoBooking === 0,
    pass: ztRoTeamTab === 0 && ztRoExport === 0 && kalRoBooking === 0,
  })

  // ── 9) member berichte: create bleibt, Finance-KPIs weg, DATEV/Geplant weg, edit-own im Editor
  await switchTo(/Markus Weber/)
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neuer Bericht')
  await page.waitForTimeout(1400)
  const berMem = await bodyText()
  const memDatevTab = await btn('DATEV', true).count()
  const memGeplantTab = await btn(/^Geplant \(/).count()
  await shot('09-member-berichte-dashboard.png')
  out.push({
    step: 'member berichte: Neuer Bericht DA, Umsatz-KPI + Umsatzverlauf WEG (kein finance view), Helpdesk-KPI DA, DATEV + Geplant WEG',
    create: /Neuer Bericht/.test(berMem),
    noFinanceKpi: !/Umsatz \(MTD\)/.test(berMem),
    noHeroChart: !/Umsatzverlauf/.test(berMem),
    helpdeskKpi: /Offene Tickets/.test(berMem),
    noDatev: memDatevTab === 0,
    noGeplant: memGeplantTab === 0,
    rawKeys: rawKeys(berMem),
    pass: /Neuer Bericht/.test(berMem) && !/Umsatz \(MTD\)/.test(berMem) && /Offene Tickets/.test(berMem) && memDatevTab === 0 && memGeplantTab === 0 && rawKeys(berMem).length === 0,
  })

  // ── 10) member berichte editor: EIGENER Draft editierbar (ohne Lifecycle), FREMDER final read-only
  await btn('Berichte', true).click()
  await page.waitForTimeout(900)
  await btn(/Helpdesk-Auslastung KW 24/).first().click()
  await page.waitForTimeout(1200)
  const memOwn = await bodyText()
  const memOwnEdit = await btn('Bearbeiten', true).count()
  const memOwnFinal = await btn('Als fertig markieren').count()
  const memOwnShare = await btn('Teilen', true).count()
  await shot('10a-member-berichte-own-draft.png')
  // Same-hash goto does not remount (editor is state-based) — bounce via dashboard.
  await page.goto(`${BASE}/#/dashboard`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(700)
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1200)
  await btn('Berichte', true).click()
  await page.waitForTimeout(900)
  await btn(/Monatsbericht Juni 2026/).first().click()
  await page.waitForTimeout(1200)
  const memForeign = await bodyText()
  const memForeignEdit = await btn('Bearbeiten', true).count()
  const memForeignRelease = await btn(/Freigeben/).count()
  await shot('10b-member-berichte-foreign-final.png')
  out.push({
    step: 'member editor: eigener Draft MIT Bearbeiten, OHNE Als fertig markieren/Teilen; fremder final-Bericht OHNE Bearbeiten/Freigeben',
    ownEdit: memOwnEdit > 0,
    ownNoLifecycle: memOwnFinal === 0,
    ownNoShare: memOwnShare === 0,
    foreignNoEdit: memForeignEdit === 0,
    foreignNoRelease: memForeignRelease === 0,
    rawKeys: [...rawKeys(memOwn), ...rawKeys(memForeign)],
    pass: memOwnEdit > 0 && memOwnFinal === 0 && memOwnShare === 0 && memForeignEdit === 0 && memForeignRelease === 0 && rawKeys(memForeign).length === 0,
  })

  // ── 11) member: automatisierung KOMPLETT WEG (Nav + Deep-Link), formulare ohne Designer
  const memAutoNav = await page.locator('a[href="#/automatisierung"]').count()
  await page.goto(`${BASE}/#/automatisierung`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const autoMem = await bodyText()
  await shot('11a-member-automatisierung-noaccess.png')
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const formMem = await bodyText()
  await shot('11b-member-formulare.png')
  out.push({
    step: 'member: automatisierung Nav-Eintrag WEG + Deep-Link → Kein Zugriff; formulare: Eingänge-Tab DA, kein Neues Formular',
    noNav: memAutoNav === 0,
    noAccess: /Kein Zugriff/.test(autoMem),
    formSubTab: /Eingänge \(/.test(formMem),
    formNoCreate: !/Neues Formular/.test(formMem),
    rawKeys: rawKeys(formMem),
    pass: memAutoNav === 0 && /Kein Zugriff/.test(autoMem) && /Eingänge \(/.test(formMem) && !/Neues Formular/.test(formMem) && rawKeys(formMem).length === 0,
  })

  // ── 12) extern: berichte + formulare Deep-Links → NoAccess (Ebene 1 hält)
  await switchTo(/Max Steiner/)
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const extBer = await bodyText()
  await shot('12a-extern-berichte-noaccess.png')
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const extForm = await bodyText()
  await shot('12b-extern-formulare-noaccess.png')
  out.push({
    step: 'extern: /berichte + /formulare → Kein-Zugriff-Seite',
    pass: /Kein Zugriff/.test(extBer) && /Kein Zugriff/.test(extForm),
  })

  // ── 13) admin Rückkehr: Regression nach allen Switches
  await switchTo(/Stefan Vogel/)
  await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neuer Bericht')
  const berFinal = await bodyText()
  await page.goto(`${BASE}/#/automatisierung`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neue Automatisierung')
  const autoFinal = await bodyText()
  await shot('13-admin-regression.png')
  out.push({
    step: 'admin Rückkehr: Neuer Bericht + Neue Automatisierung wieder da',
    pass: /Neuer Bericht/.test(berFinal) && /Neue Automatisierung/.test(autoFinal),
  })
} finally {
  await b.close()
}

let allPass = true
for (const step of out) {
  if (!step.pass) allPass = false
  console.log(JSON.stringify(step))
}
console.log(`pageerrors: ${JSON.stringify(errs.slice(0, 5))}`)
console.log(allPass && errs.length === 0 ? 'ALL PASS' : 'FAILURES — see above')
process.exit(allPass && errs.length === 0 ? 0 : 1)
