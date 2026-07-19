/**
 * QA — RBAC R-3 Batch 4 (industry modules: schichten · fuhrpark · vermietung ·
 * rapporte · dialer).
 * Verifies: admin regression per module (header actions, tabs, no chip),
 * rapporte approve UI (submitted report shows Genehmigen/Ablehnen for admin,
 * not for readonly) + author display names, schichten swap own-scope (member
 * sees only his request, no approve buttons, real names instead of raw ids),
 * fuhrpark gps tab privacy (admin only), dialer route guard (nav filter +
 * redirect on forbidden deep link), extern deep-link → NoAccess.
 * Raw keys + pageerrors tracked.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/rbac-enforcement-b4')
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
const rawKeys = (txt) => (txt.match(/\b(rbac|schichten|fuhrpark|vermietung|rapporte|dialer)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)
const waitForText = (t) =>
  page.waitForFunction((x) => document.body.innerText.includes(x), t, { timeout: 30000 }).catch(() => {})
const shot = (name) => page.screenshot({ path: resolve(outDir, name) })
const navLink = (path) => page.locator(`a[href="#${path}"]`)

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
  // ── 1) admin schichten: 4 Tabs + alle 4 Header-Aktionen, kein Chip
  await page.goto(`${BASE}/#/schichten`, { waitUntil: 'domcontentloaded' })
  await waitForText('Schicht zuweisen')
  await page.waitForTimeout(1000)
  const schAdmin = await bodyText()
  await shot('01-admin-schichten.png')
  out.push({
    step: 'admin schichten: Wochenplan/Vorlagen/Tausch-Anfragen/Verfügbarkeit + CSV/PDF/Veröffentlichen/Zuweisen, kein Chip',
    actions: /CSV-Export/.test(schAdmin) && /PDF-Export/.test(schAdmin) && /Woche veröffentlichen/.test(schAdmin) && /Schicht zuweisen/.test(schAdmin),
    tabs: /Wochenplan/.test(schAdmin) && /Vorlagen \(/.test(schAdmin) && /Tausch-Anfragen \(/.test(schAdmin) && /Verfügbarkeit/.test(schAdmin),
    noChip: !/Nur Ansicht|Eingeschränkt/.test(schAdmin),
    rawKeys: rawKeys(schAdmin),
    pass: /Schicht zuweisen/.test(schAdmin) && /Vorlagen \(/.test(schAdmin) && !/Nur Ansicht/.test(schAdmin) && rawKeys(schAdmin).length === 0,
  })

  // ── 2) admin schichten anfragen: Genehmigen-Buttons + echte Namen statt Roh-Ids
  await page.getByRole('button', { name: /Tausch-Anfragen/ }).click()
  await page.waitForTimeout(900)
  const anfAdmin = await bodyText()
  const approveBtns = await page.getByRole('button', { name: 'Genehmigen' }).count()
  await shot('02-admin-schichten-anfragen.png')
  out.push({
    step: 'admin schichten anfragen: alle 3 Anfragen, Genehmigen-Button auf pending, Namen statt usr-Ids',
    allThree: /Jan Schäfer/.test(anfAdmin) && /Felix Krause/.test(anfAdmin) && /Thomas Keller/.test(anfAdmin),
    approveBtn: approveBtns > 0,
    noRawIds: !/usr-e\d/.test(anfAdmin),
    rawKeys: rawKeys(anfAdmin),
    pass: /Jan Schäfer/.test(anfAdmin) && approveBtns > 0 && !/usr-e\d/.test(anfAdmin) && rawKeys(anfAdmin).length === 0,
  })

  // ── 3) admin rapporte: Create + Export + Approve-UI im submitted-Detail
  await page.goto(`${BASE}/#/rapporte`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neuer Tagesbericht')
  await page.waitForTimeout(1000)
  const rapAdmin = await bodyText()
  await page.getByRole('button', { name: /Innenausbau Büro Winterthur/ }).first().click()
  await page.waitForTimeout(1200)
  const rapModal = await bodyText()
  const approveInModal = await page.getByRole('button', { name: 'Genehmigen' }).count()
  const rejectInModal = await page.getByRole('button', { name: /^Ablehnen$/ }).count()
  await shot('03-admin-rapporte-approve-modal.png')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(500)
  out.push({
    step: 'admin rapporte: Neuer Tagesbericht + 3 Tabs + Exportieren; submitted-Detail hat Genehmigen+Ablehnen; Autor = echter Name',
    create: /Neuer Tagesbericht/.test(rapAdmin),
    tabs: /Tagesberichte \(/.test(rapAdmin) && /Aufmass \(/.test(rapAdmin) && /Vorlagen \(/.test(rapAdmin),
    exportBtn: /Exportieren/.test(rapAdmin),
    authorName: /Markus Weber/.test(rapModal),
    approve: approveInModal > 0,
    reject: rejectInModal > 0,
    rawKeys: rawKeys(rapModal),
    pass: /Neuer Tagesbericht/.test(rapAdmin) && approveInModal > 0 && rejectInModal > 0 && /Markus Weber/.test(rapModal) && rawKeys(rapModal).length === 0,
  })

  // ── 4) admin fuhrpark + vermietung: Creates + Tracking-Tab da
  await page.goto(`${BASE}/#/fuhrpark`, { waitUntil: 'domcontentloaded' })
  await waitForText('Fahrzeug hinzufügen')
  await page.waitForTimeout(900)
  const fpAdmin = await bodyText()
  await shot('04a-admin-fuhrpark.png')
  await page.goto(`${BASE}/#/vermietung`, { waitUntil: 'domcontentloaded' })
  await waitForText('Objekt anlegen')
  await page.waitForTimeout(900)
  const vmAdmin = await bodyText()
  await shot('04b-admin-vermietung.png')
  out.push({
    step: 'admin fuhrpark/vermietung: Fahrzeug hinzufügen + Tracking-Tab; Objekt anlegen + Reservierung',
    fpCreate: /Fahrzeug hinzufügen/.test(fpAdmin),
    fpTracking: /Tracking/.test(fpAdmin),
    vmCreateObj: /Objekt anlegen/.test(vmAdmin),
    vmCreateRes: /Reservierung\b/.test(vmAdmin),
    noChips: !/Nur Ansicht/.test(fpAdmin) && !/Nur Ansicht/.test(vmAdmin),
    rawKeys: [...rawKeys(fpAdmin), ...rawKeys(vmAdmin)],
    pass: /Fahrzeug hinzufügen/.test(fpAdmin) && /Tracking/.test(fpAdmin) && /Objekt anlegen/.test(vmAdmin) && rawKeys(fpAdmin).length === 0 && rawKeys(vmAdmin).length === 0,
  })

  // ── 5) admin dialer: 5 Nav-Links + Neue Kampagne
  await page.goto(`${BASE}/#/dialer`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neue Kampagne')
  await page.waitForTimeout(900)
  const dlAdmin = await bodyText()
  const adminLinks = {
    campaigns: await navLink('/dialer/campaigns').count(),
    workspace: await navLink('/dialer/workspace').count(),
    dashboard: await navLink('/dialer/dashboard').count(),
    supervisor: await navLink('/dialer/supervisor').count(),
    settings: await navLink('/dialer/settings').count(),
  }
  await shot('05-admin-dialer.png')
  out.push({
    step: 'admin dialer: alle 5 Nav-Tabs + Neue Kampagne',
    links: adminLinks,
    create: /Neue Kampagne/.test(dlAdmin),
    rawKeys: rawKeys(dlAdmin),
    pass: Object.values(adminLinks).every((c) => c > 0) && /Neue Kampagne/.test(dlAdmin) && rawKeys(dlAdmin).length === 0,
  })

  // ── 6) readonly schichten + fuhrpark: Chip, keine Aktionen, kein Tracking/GPS
  await switchTo(/Elena Richter/)
  await page.goto(`${BASE}/#/schichten`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const schRo = await bodyText()
  await shot('06a-readonly-schichten.png')
  await page.goto(`${BASE}/#/fuhrpark`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const fpRo = await bodyText()
  await shot('06b-readonly-fuhrpark.png')
  out.push({
    step: 'readonly schichten/fuhrpark: Chip, keine Header-Aktionen, Vorlagen-Tab bleibt (read), Tracking-Tab WEG (GPS-Privacy)',
    schChip: /Nur Ansicht/.test(schRo),
    schNoActions: !/Schicht zuweisen/.test(schRo) && !/Woche veröffentlichen/.test(schRo) && !/CSV-Export/.test(schRo),
    schTemplateTabStays: /Vorlagen \(/.test(schRo),
    fpChip: /Nur Ansicht/.test(fpRo),
    fpNoCreate: !/Fahrzeug hinzufügen/.test(fpRo),
    fpNoTracking: !/Tracking/.test(fpRo),
    rawKeys: [...rawKeys(schRo), ...rawKeys(fpRo)],
    pass: /Nur Ansicht/.test(schRo) && !/Schicht zuweisen/.test(schRo) && /Vorlagen \(/.test(schRo) && !/Fahrzeug hinzufügen/.test(fpRo) && !/Tracking/.test(fpRo) && rawKeys(schRo).length === 0,
  })

  // ── 7) readonly rapporte: sieht ALLE, aber kein Create/Export/Approve
  await page.goto(`${BASE}/#/rapporte`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const rapRo = await bodyText()
  await page.getByRole('button', { name: /Innenausbau Büro Winterthur/ }).first().click()
  await page.waitForTimeout(1200)
  const rapRoModal = await bodyText()
  const roApprove = await page.getByRole('button', { name: 'Genehmigen' }).count()
  await shot('07-readonly-rapporte-modal.png')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(500)
  out.push({
    step: 'readonly rapporte: fremde Berichte sichtbar (read all), kein Neuer Tagesbericht, kein Exportieren, Detail OHNE Genehmigen/Löschen/PDF',
    seesForeign: /Rohbau Mehrfamilienhaus Zürich/.test(rapRo),
    chip: /Nur Ansicht/.test(rapRo),
    noCreate: !/Neuer Tagesbericht/.test(rapRo),
    noExport: !/Exportieren/.test(rapRo),
    noApprove: roApprove === 0,
    noDelete: !/Löschen/.test(rapRoModal),
    noPdf: !/PDF-Export/.test(rapRoModal),
    rawKeys: rawKeys(rapRoModal),
    pass: /Rohbau Mehrfamilienhaus Zürich/.test(rapRo) && !/Neuer Tagesbericht/.test(rapRo) && roApprove === 0 && !/PDF-Export/.test(rapRoModal) && rawKeys(rapRoModal).length === 0,
  })

  // ── 8) readonly dialer: nur Kampagnen+Dashboard-Links; Workspace-Deep-Link → Redirect
  await page.goto(`${BASE}/#/dialer`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const dlRo = await bodyText()
  const roLinks = {
    campaigns: await navLink('/dialer/campaigns').count(),
    workspace: await navLink('/dialer/workspace').count(),
    dashboard: await navLink('/dialer/dashboard').count(),
    supervisor: await navLink('/dialer/supervisor').count(),
    settings: await navLink('/dialer/settings').count(),
  }
  await page.goto(`${BASE}/#/dialer/workspace`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const roWsUrl = page.url()
  await shot('08-readonly-dialer.png')
  out.push({
    step: 'readonly dialer: Nav nur Kampagnen+Dashboard, keine Neue Kampagne; /dialer/workspace → Redirect',
    links: roLinks,
    noCreate: !/Neue Kampagne/.test(dlRo),
    redirected: !roWsUrl.includes('/dialer/workspace'),
    rawKeys: rawKeys(dlRo),
    pass: roLinks.campaigns > 0 && roLinks.dashboard > 0 && roLinks.workspace === 0 && roLinks.supervisor === 0 && roLinks.settings === 0 && !/Neue Kampagne/.test(dlRo) && !roWsUrl.includes('/dialer/workspace') && rawKeys(dlRo).length === 0,
  })

  // ── 9) extern: schichten + dialer Deep-Links → NoAccess (Ebene 1 hält)
  await switchTo(/Max Steiner/)
  await page.goto(`${BASE}/#/schichten`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const extSch = await bodyText()
  await shot('09a-extern-schichten-noaccess.png')
  await page.goto(`${BASE}/#/dialer`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const extDl = await bodyText()
  await shot('09b-extern-dialer-noaccess.png')
  out.push({
    step: 'extern: /schichten + /dialer → Kein-Zugriff-Seite',
    pass: /Kein Zugriff/.test(extSch) && /Kein Zugriff/.test(extDl),
  })

  // ── 10) member schichten: swap-own — nur eigene Anfrage, keine Approve-Buttons, kein Vorlagen-Tab
  await switchTo(/Markus Weber/)
  await page.goto(`${BASE}/#/schichten`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const schMember = await bodyText()
  await page.getByRole('button', { name: /Tausch-Anfragen/ }).click()
  await page.waitForTimeout(900)
  const anfMember = await bodyText()
  const memberApprove = await page.getByRole('button', { name: 'Genehmigen' }).count()
  await shot('10-member-schichten-anfragen.png')
  out.push({
    step: 'member schichten: keine Header-Aktionen, Vorlagen-Tab WEG; Anfragen nur EIGENE (Felix↔Markus), Jans Anfrage weg, kein Genehmigen',
    noActions: !/Schicht zuweisen/.test(schMember) && !/Woche veröffentlichen/.test(schMember) && !/CSV-Export/.test(schMember),
    noTemplateTab: !/Vorlagen \(/.test(schMember),
    wochenplanStays: /Wochenplan/.test(schMember),
    ownSwap: /Felix Krause/.test(anfMember) && /Markus Weber/.test(anfMember),
    noForeignSwap: !/Arzttermin/.test(anfMember),
    noApprove: memberApprove === 0,
    rawKeys: rawKeys(anfMember),
    pass: !/Schicht zuweisen/.test(schMember) && !/Vorlagen \(/.test(schMember) && /Felix Krause/.test(anfMember) && !/Arzttermin/.test(anfMember) && memberApprove === 0 && rawKeys(anfMember).length === 0,
  })

  // ── 11) member rapporte: scope-own Liste + Create + Export (Werker-PDF) bleiben
  await page.goto(`${BASE}/#/rapporte`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const rapMember = await bodyText()
  await shot('11-member-rapporte.png')
  out.push({
    step: 'member rapporte: NUR eigene 3 Berichte (Innenausbau/Elektrische/Badezimmer), fremde weg; Neuer Tagesbericht + Exportieren bleiben',
    own1: /Innenausbau Büro Winterthur/.test(rapMember),
    own2: /Elektrische Installation St\. Gallen/.test(rapMember),
    own3: /Badezimmer-Renovation Thun/.test(rapMember),
    noForeign: !/Rohbau Mehrfamilienhaus Zürich/.test(rapMember) && !/Fassadensanierung Bern/.test(rapMember),
    create: /Neuer Tagesbericht/.test(rapMember),
    exportStays: /Exportieren/.test(rapMember),
    rawKeys: rawKeys(rapMember),
    pass: /Innenausbau Büro Winterthur/.test(rapMember) && !/Rohbau Mehrfamilienhaus Zürich/.test(rapMember) && /Neuer Tagesbericht/.test(rapMember) && /Exportieren/.test(rapMember) && rawKeys(rapMember).length === 0,
  })

  // ── 12) member fuhrpark: Fahrer-Basics bleiben, Verwaltung/GPS/Exporte weg
  await page.goto(`${BASE}/#/fuhrpark`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const fpMember = await bodyText()
  await page.getByRole('button', { name: /Fahrtenbuch/ }).click()
  await page.waitForTimeout(800)
  const fpMemberLog = await bodyText()
  await shot('12-member-fuhrpark-fahrtenbuch.png')
  out.push({
    step: 'member fuhrpark: kein Fahrzeug hinzufügen, kein Tracking; Fahrtenbuch: Fahrt eintragen DA, CSV/PDF-Export WEG',
    noCreate: !/Fahrzeug hinzufügen/.test(fpMember),
    noTracking: !/Tracking/.test(fpMember),
    tripCreate: /Fahrt eintragen/.test(fpMemberLog),
    noExport: !/CSV-Export/.test(fpMemberLog) && !/PDF-Export/.test(fpMemberLog),
    rawKeys: rawKeys(fpMemberLog),
    pass: !/Fahrzeug hinzufügen/.test(fpMember) && !/Tracking/.test(fpMember) && /Fahrt eintragen/.test(fpMemberLog) && !/CSV-Export/.test(fpMemberLog) && rawKeys(fpMemberLog).length === 0,
  })

  // ── 13) member vermietung + dialer: Desk-Aktionen bleiben, Verwaltung weg + Routen-Guard
  await page.goto(`${BASE}/#/vermietung`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const vmMember = await bodyText()
  const vmResBtn = await page.getByRole('button', { name: 'Reservierung', exact: true }).count()
  await shot('13a-member-vermietung.png')
  await page.goto(`${BASE}/#/dialer/supervisor`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const dlMemberUrl = page.url()
  const dlMember = await bodyText()
  await shot('13b-member-dialer.png')
  out.push({
    step: 'member vermietung: Reservierung-Button DA, Objekt anlegen + Exportieren WEG; dialer /supervisor → Redirect, keine Neue Kampagne',
    vmResBtn: vmResBtn > 0,
    vmNoObjCreate: !/Objekt anlegen/.test(vmMember),
    vmNoExport: !/Exportieren/.test(vmMember),
    dlRedirected: !dlMemberUrl.includes('/dialer/supervisor'),
    dlNoCreate: !/Neue Kampagne/.test(dlMember),
    rawKeys: [...rawKeys(vmMember), ...rawKeys(dlMember)],
    pass: vmResBtn > 0 && !/Objekt anlegen/.test(vmMember) && !dlMemberUrl.includes('/dialer/supervisor') && !/Neue Kampagne/.test(dlMember) && rawKeys(vmMember).length === 0,
  })

  // ── 14) admin Rückkehr: Regression nach allen Switches
  await switchTo(/Stefan Vogel/)
  await page.goto(`${BASE}/#/schichten`, { waitUntil: 'domcontentloaded' })
  await waitForText('Schicht zuweisen')
  const schFinal = await bodyText()
  await page.goto(`${BASE}/#/dialer`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neue Kampagne')
  const dlFinal = await bodyText()
  await shot('14-admin-regression.png')
  out.push({
    step: 'admin Rückkehr: Schicht zuweisen + Neue Kampagne wieder da',
    pass: /Schicht zuweisen/.test(schFinal) && /Neue Kampagne/.test(dlFinal),
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
