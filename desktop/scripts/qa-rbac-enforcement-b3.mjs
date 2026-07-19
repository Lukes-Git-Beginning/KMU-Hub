/**
 * QA — RBAC R-3 Batch 3 (industry modules: inventar · einkauf · produktion ·
 * vertraege · helpdesk).
 * Verifies: admin regression per module (create buttons, tabs, exports, no
 * chip), einkauf send-exception (draft modal: enabled for admin, disabled+
 * tooltip for readonly), readonly "Nur Ansicht" chip + hidden mutations,
 * extern deep-link → NoAccess, member helpdesk requester model (sees own
 * tickets only, no stats tab), member produktion (no create, Laufkarte export
 * stays), member inventar (no create/export). Raw keys + pageerrors tracked.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/rbac-enforcement-b3')
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
const rawKeys = (txt) => (txt.match(/\b(rbac|inventar|einkauf|produktion|vertraege|helpdesk)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)
const waitForText = (t) =>
  page.waitForFunction((x) => document.body.innerText.includes(x), t, { timeout: 30000 }).catch(() => {})
const shot = (name) => page.screenshot({ path: resolve(outDir, name) })

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
  // ── 1) admin inventar: create + 4 tabs + export, kein Chip
  await page.goto(`${BASE}/#/inventar`, { waitUntil: 'domcontentloaded' })
  await waitForText('Artikel hinzufügen')
  await page.waitForTimeout(1000)
  const invAdmin = await bodyText()
  await shot('01-admin-inventar.png')
  out.push({
    step: 'admin inventar: Artikel hinzufügen + 4 Tabs + CSV-Export, kein Nur-Ansicht-Chip',
    create: /Artikel hinzufügen/.test(invAdmin),
    tabs: /Artikel \(/.test(invAdmin) && /Lagerorte \(/.test(invAdmin) && /Bewegungen \(/.test(invAdmin) && /Inventur \(/.test(invAdmin),
    exportBtn: /CSV exportieren|exportieren/i.test(invAdmin),
    noChip: !/Nur Ansicht/.test(invAdmin),
    rawKeys: rawKeys(invAdmin),
    pass: /Artikel hinzufügen/.test(invAdmin) && /Inventur \(/.test(invAdmin) && !/Nur Ansicht/.test(invAdmin) && rawKeys(invAdmin).length === 0,
  })

  // ── 2) admin einkauf: Neue Bestellung + Draft-Modal mit AKTIVEM Senden-Button
  await page.goto(`${BASE}/#/einkauf`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neue Bestellung')
  await page.waitForTimeout(1000)
  const ekAdmin = await bodyText()
  await page.locator('tr', { hasText: 'Entwurf' }).first().click().catch(() => {})
  await page.waitForTimeout(1200)
  const sendEnabled = await page.getByRole('button', { name: 'An Lieferant senden' }).isEnabled().catch(() => false)
  await shot('02-admin-einkauf-draft-modal.png')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(500)
  out.push({
    step: 'admin einkauf: Neue Bestellung + 4 Tabs; Draft-Detail hat AKTIVEN Senden-Button + Stornieren',
    create: /Neue Bestellung/.test(ekAdmin),
    tabs: /Bestellungen \(/.test(ekAdmin) && /Lieferanten \(/.test(ekAdmin) && /Katalog \(/.test(ekAdmin) && /Rahmenverträge \(/.test(ekAdmin),
    sendEnabled,
    rawKeys: rawKeys(ekAdmin),
    pass: /Neue Bestellung/.test(ekAdmin) && sendEnabled && rawKeys(ekAdmin).length === 0,
  })

  // ── 3) admin produktion + vertraege: create buttons da
  await page.goto(`${BASE}/#/produktion`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neuer Auftrag')
  await page.waitForTimeout(900)
  const prodAdmin = await bodyText()
  await shot('03a-admin-produktion.png')
  await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
  await waitForText('Vertrag anlegen')
  await page.waitForTimeout(900)
  const vertAdmin = await bodyText()
  await shot('03b-admin-vertraege.png')
  out.push({
    step: 'admin produktion/vertraege: Neuer Auftrag + Vertrag anlegen da, keine Chips',
    prodCreate: /Neuer Auftrag/.test(prodAdmin),
    prodTabs: /Aufträge \(/.test(prodAdmin) && /Maschinen \(/.test(prodAdmin),
    vertCreate: /Vertrag anlegen/.test(vertAdmin),
    noChips: !/Nur Ansicht/.test(prodAdmin) && !/Nur Ansicht/.test(vertAdmin),
    rawKeys: [...rawKeys(prodAdmin), ...rawKeys(vertAdmin)],
    pass: /Neuer Auftrag/.test(prodAdmin) && /Vertrag anlegen/.test(vertAdmin) && !/Nur Ansicht/.test(prodAdmin) && rawKeys(prodAdmin).length === 0 && rawKeys(vertAdmin).length === 0,
  })

  // ── 4) admin helpdesk: alle Tickets + Vorlagen + Statistiken-Tab
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neues Ticket')
  await page.waitForTimeout(1000)
  const hdAdmin = await bodyText()
  const adminStatsTab = await page.getByRole('button', { name: /^Statistik$/ }).count()
  await shot('04-admin-helpdesk.png')
  out.push({
    step: 'admin helpdesk: Neues Ticket + Vorlagen + Statistik-Tab, fremdes Ticket sichtbar',
    create: /Neues Ticket/.test(hdAdmin),
    canned: /Vorlagen/.test(hdAdmin),
    statsTab: adminStatsTab > 0,
    foreignTicket: /Drucker im 2\. OG/.test(hdAdmin),
    rawKeys: rawKeys(hdAdmin),
    pass: /Neues Ticket/.test(hdAdmin) && /Vorlagen/.test(hdAdmin) && adminStatsTab > 0 && /Drucker im 2\. OG/.test(hdAdmin) && rawKeys(hdAdmin).length === 0,
  })

  // ── 5) readonly inventar: Chip, keine Mutationen, kein Export
  await switchTo(/Elena Richter/)
  await page.goto(`${BASE}/#/inventar`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const invRo = await bodyText()
  await shot('05-readonly-inventar.png')
  out.push({
    step: 'readonly inventar: Nur-Ansicht-Chip, kein Artikel hinzufügen, kein Export, keine Neue Inventur',
    chip: /Nur Ansicht/.test(invRo),
    noCreate: !/Artikel hinzufügen/.test(invRo),
    noExport: !/CSV exportieren/.test(invRo),
    noNewInventur: !/Neue Inventur starten/.test(invRo),
    tabsStay: /Inventur \(/.test(invRo),
    rawKeys: rawKeys(invRo),
    pass: /Nur Ansicht/.test(invRo) && !/Artikel hinzufügen/.test(invRo) && !/CSV exportieren/.test(invRo) && !/Neue Inventur starten/.test(invRo) && /Inventur \(/.test(invRo) && rawKeys(invRo).length === 0,
  })

  // ── 6) readonly einkauf: Senden-AUSNAHME = disabled + Tooltip, Rest versteckt
  await page.goto(`${BASE}/#/einkauf`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1400)
  const ekRo = await bodyText()
  await page.locator('tr', { hasText: 'Entwurf' }).first().click().catch(() => {})
  await page.waitForTimeout(1200)
  const roSendBtn = page.getByRole('button', { name: 'An Lieferant senden' })
  const roSendVisible = await roSendBtn.isVisible().catch(() => false)
  const roSendDisabled = roSendVisible ? !(await roSendBtn.isEnabled().catch(() => true)) : false
  const roModal = await bodyText()
  await shot('06-readonly-einkauf-draft-modal.png')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(500)
  out.push({
    step: 'readonly einkauf: kein Neue Bestellung/Lieferant anlegen; Draft-Modal: Senden sichtbar aber DISABLED, Stornieren/PDF versteckt',
    noCreate: !/Neue Bestellung/.test(ekRo),
    noAddSupplier: !/Lieferant anlegen/.test(ekRo),
    chip: /Nur Ansicht/.test(ekRo),
    sendVisible: roSendVisible,
    sendDisabled: roSendDisabled,
    noCancel: !/Stornieren/.test(roModal),
    noPdf: !/Bestell-PDF/.test(roModal),
    rawKeys: rawKeys(roModal),
    pass: !/Neue Bestellung/.test(ekRo) && /Nur Ansicht/.test(ekRo) && roSendVisible && roSendDisabled && !/Stornieren/.test(roModal) && rawKeys(roModal).length === 0,
  })

  // ── 7) readonly produktion + vertraege: keine Creates, Chips da
  await page.goto(`${BASE}/#/produktion`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1400)
  const prodRo = await bodyText()
  await shot('07a-readonly-produktion.png')
  await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1400)
  const vertRo = await bodyText()
  await shot('07b-readonly-vertraege.png')
  out.push({
    step: 'readonly produktion/vertraege: keine Create-Buttons, keine Exporte, Chips da',
    noProdCreate: !/Neuer Auftrag/.test(prodRo),
    noProdExport: !/Aufträge exportieren|CSV/.test(prodRo),
    prodChip: /Nur Ansicht/.test(prodRo),
    noVertCreate: !/Vertrag anlegen/.test(vertRo),
    vertChip: /Nur Ansicht/.test(vertRo),
    rawKeys: [...rawKeys(prodRo), ...rawKeys(vertRo)],
    pass: !/Neuer Auftrag/.test(prodRo) && /Nur Ansicht/.test(prodRo) && !/Vertrag anlegen/.test(vertRo) && /Nur Ansicht/.test(vertRo) && rawKeys(vertRo).length === 0,
  })

  // ── 8) extern: helpdesk + inventar deep-links → NoAccess (Ebene 1 hält)
  await switchTo(/Max Steiner/)
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const extHd = await bodyText()
  await shot('08a-extern-helpdesk-noaccess.png')
  await page.goto(`${BASE}/#/produktion`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const extProd = await bodyText()
  await shot('08b-extern-produktion-noaccess.png')
  out.push({
    step: 'extern: /helpdesk + /produktion → Kein-Zugriff-Seite',
    pass: /Kein Zugriff/.test(extHd) && /Kein Zugriff/.test(extProd),
  })

  // ── 9) member helpdesk: Requester-Modell — nur eigene Tickets, kein Stats-Tab
  await switchTo(/Markus Weber/)
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const hdMember = await bodyText()
  const memberStatsTab = await page.getByRole('button', { name: /^Statistik$/ }).count()
  await shot('09-member-helpdesk.png')
  out.push({
    step: 'member helpdesk: eigene Tickets (Requester/Assignee) sichtbar, fremde weg; Neues Ticket da, Vorlagen + Statistik weg',
    ownRequester: /Neuer Mitarbeiter – Zugänge einrichten/.test(hdMember),
    ownRequester2: /Bildschirm flackert/.test(hdMember),
    ownAssignee: /WLAN im Sitzungszimmer/.test(hdMember),
    noForeign: !/Drucker im 2\. OG/.test(hdMember),
    create: /Neues Ticket/.test(hdMember),
    noCanned: !/Vorlagen\b/.test(hdMember),
    noStats: memberStatsTab === 0,
    rawKeys: rawKeys(hdMember),
    pass: /Neuer Mitarbeiter – Zugänge einrichten/.test(hdMember) && !/Drucker im 2\. OG/.test(hdMember) && /Neues Ticket/.test(hdMember) && memberStatsTab === 0 && rawKeys(hdMember).length === 0,
  })

  // ── 10) member produktion: kein Create, aber Laufkarte/Export bleibt (Werker)
  await page.goto(`${BASE}/#/produktion`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1400)
  const prodMember = await bodyText()
  const memberExportBtn = await page.getByRole('button', { name: /CSV/ }).count()
  await shot('10-member-produktion.png')
  out.push({
    step: 'member produktion: kein Neuer Auftrag/Neue Maschine, Export (Laufkarte-Werkerfall) DA, alle 4 Tabs',
    noCreate: !/Neuer Auftrag/.test(prodMember),
    exportStays: memberExportBtn > 0,
    tabs: /Aufträge \(/.test(prodMember) && /Maschinen \(/.test(prodMember),
    rawKeys: rawKeys(prodMember),
    pass: !/Neuer Auftrag/.test(prodMember) && memberExportBtn > 0 && /Aufträge \(/.test(prodMember) && rawKeys(prodMember).length === 0,
  })

  // ── 11) member inventar: kein Create/Export, Bewegung erfassbar (Zeilenmenü bleibt)
  await page.goto(`${BASE}/#/inventar`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1400)
  const invMember = await bodyText()
  await shot('11-member-inventar.png')
  out.push({
    step: 'member inventar: kein Artikel hinzufügen, kein CSV-Export, keine Neue Inventur; Tabs bleiben',
    noCreate: !/Artikel hinzufügen/.test(invMember),
    noExport: !/CSV exportieren/.test(invMember),
    noNewInventur: !/Neue Inventur starten/.test(invMember),
    tabsStay: /Artikel \(/.test(invMember) && /Inventur \(/.test(invMember),
    rawKeys: rawKeys(invMember),
    pass: !/Artikel hinzufügen/.test(invMember) && !/CSV exportieren/.test(invMember) && /Artikel \(/.test(invMember) && rawKeys(invMember).length === 0,
  })

  // ── 12) admin Rückkehr: Regression nach allen Switches
  await switchTo(/Stefan Vogel/)
  await page.goto(`${BASE}/#/inventar`, { waitUntil: 'domcontentloaded' })
  await waitForText('Artikel hinzufügen')
  await page.waitForTimeout(900)
  const invFinal = await bodyText()
  await page.goto(`${BASE}/#/einkauf`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neue Bestellung')
  const ekFinal = await bodyText()
  await shot('12-admin-regression.png')
  out.push({
    step: 'admin Rückkehr: Artikel hinzufügen + Neue Bestellung wieder da',
    pass: /Artikel hinzufügen/.test(invFinal) && /Neue Bestellung/.test(ekFinal),
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
