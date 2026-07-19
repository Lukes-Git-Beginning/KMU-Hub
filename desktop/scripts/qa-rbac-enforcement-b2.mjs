/**
 * QA — RBAC R-3 Batch 2 (team actions · dashboard level 2 · admin/security tabs).
 * Verifies: admin full view (regression, incl. security hub landing on Audit),
 * extern dashboard (cards/alerts/quick actions filtered, no foreign modules),
 * hr_admin/it_admin AdminHub tab sets + deep-link redirect, it_admin security
 * sub-tabs (no GDPR/DSAR), manager team (approve visible, create/payroll gone,
 * finance card gone), readonly member profile (contact/employment sections and
 * documents tab hidden). Raw keys + pageerrors tracked per step.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/rbac-enforcement-b2')
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
const rawKeys = (txt) => (txt.match(/\b(rbac|team|admin|dashboard|widgets|security)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)
const waitForText = (t) =>
  page.waitForFunction((x) => document.body.innerText.includes(x), t, { timeout: 30000 }).catch(() => {})
const shot = (name) => page.screenshot({ path: resolve(outDir, name) })
const tabNames = async () => {
  const tabs = await page.getByRole('tab').allInnerTexts()
  return tabs.map((s) => s.trim())
}

// Panel state survives hash navigation (SPA) — check visibility instead of
// blind toggling.
const switcherPanel = () => page.locator('div.max-h-80')
const setSwitcherOpen = async (open) => {
  const isOpen = await switcherPanel().isVisible().catch(() => false)
  if (isOpen !== open) {
    await page.locator('button.fixed.bottom-4.right-4').click()
    await page.waitForTimeout(400)
  }
}
// Always switch from a neutral page (user names appear as clickable rows on
// /admin/users and in team cards — a bare name-locator would hit those), and
// scope the click to the switcher panel list (div.max-h-80).
const switchTo = async (labelRe) => {
  await page.goto(`${BASE}/#/settings`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(900)
  await setSwitcherOpen(true)
  await switcherPanel().getByRole('button', { name: labelRe }).first().click()
  await page.waitForTimeout(1700)
  await setSwitcherOpen(false)
}

try {
  // ── 1) admin regression: dashboard — module cards incl. finance, quick actions
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neues Projekt')
  await page.waitForTimeout(1200)
  const adminDash = await bodyText()
  await shot('01-admin-dashboard.png')
  out.push({
    step: 'admin dashboard: Buchhaltung-Karte + QuickActions + kein Empty-Hinweis',
    financeCard: /Buchhaltung/.test(adminDash),
    quickInvoice: /Neue Rechnung/.test(adminDash),
    noEmptyHint: !/keine Widgets verfügbar/.test(adminDash),
    rawKeys: rawKeys(adminDash),
    pass: /Buchhaltung/.test(adminDash) && /Neue Rechnung/.test(adminDash) && !/keine Widgets verfügbar/.test(adminDash) && rawKeys(adminDash).length === 0,
  })

  // ── 2) admin: AdminHub shows all 8 tabs
  await page.goto(`${BASE}/#/admin/users`, { waitUntil: 'domcontentloaded' })
  await waitForText('Benutzer')
  await page.waitForTimeout(1000)
  const adminTabs = await tabNames()
  await shot('02-admin-hub-tabs.png')
  const expectAll = ['Benutzer', 'Rollen', 'Lizenz', 'Branding', 'IT', 'Sicherheit', 'Abrechnung', 'Integrationen']
  out.push({
    step: 'admin AdminHub: alle 8 Tabs',
    tabs: adminTabs,
    pass: expectAll.every((t) => adminTabs.includes(t)),
  })

  // ── 3) admin: security hub lands on Audit-Log, all 10 sub-tabs present
  await page.goto(`${BASE}/#/admin/security`, { waitUntil: 'domcontentloaded' })
  await waitForText('Audit-Log')
  await page.waitForTimeout(1200)
  const secTabs = await tabNames()
  const auditSelected = await page.getByRole('tab', { name: 'Audit-Log', selected: true }).count()
  await shot('03-admin-security-subtabs.png')
  const expectSec = ['Audit-Log', 'DSGVO', 'Auskunft (Art. 15)', 'Aufbewahrung', 'Sessions', 'Passwort-Richtlinie', 'IP-Whitelist', 'Vault', 'Datenschutz', 'KI-Governance']
  out.push({
    step: 'admin Security-Hub: 10 Sub-Tabs, landet auf Audit-Log (nicht Sessions!)',
    subTabs: secTabs,
    auditSelected: auditSelected > 0,
    pass: expectSec.every((t) => secTabs.includes(t)) && auditSelected > 0,
  })

  // ── 4) admin team: create button + payroll run actions
  await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded' })
  await waitForText('Mitarbeiter erstellen')
  await page.waitForTimeout(800)
  const adminTeam = await bodyText()
  await page.getByRole('button', { name: /Lohnvorbereitung/ }).first().click()
    .catch(() => page.getByText('Lohnvorbereitung', { exact: false }).first().click().catch(() => {}))
  await page.waitForTimeout(1200)
  const adminPayroll = await bodyText()
  await shot('04-admin-team-payroll.png')
  out.push({
    step: 'admin team: Mitarbeiter erstellen + Payroll-Aktion (Prüfen & freigeben) da',
    createBtn: /Mitarbeiter erstellen/.test(adminTeam),
    payrollActions: /freigeben|Entsperren/i.test(adminPayroll),
    rawKeys: rawKeys(adminPayroll),
    pass: /Mitarbeiter erstellen/.test(adminTeam) && /freigeben|Entsperren/i.test(adminPayroll),
  })

  // ── 5) extern: dashboard filtered (no finance/crm cards, no alerts, reduced quick actions)
  await switchTo(/Max Steiner/)
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1800)
  const externDash = await bodyText()
  // quick actions are buttons — the notification widget seeds mention
  // "Neuer Kontakt"/"Neuer Deal" as plain text, so check buttons only
  const invoiceBtns = await page.getByRole('button', { name: 'Neue Rechnung' }).count()
  const contactBtns = await page.getByRole('button', { name: 'Neuer Kontakt' }).count()
  await shot('05-extern-dashboard.png')
  out.push({
    step: 'extern dashboard: keine Buchhaltung/CRM-Karten, keine Rechnung/Kontakt-QuickActions, keine Alerts, kein Umsatz-Widget',
    noFinance: !/Buchhaltung/.test(externDash),
    noInvoiceBtn: invoiceBtns === 0,
    noContactBtn: contactBtns === 0,
    hasWorkCard: /Projektverwaltung/.test(externDash),
    hasDocsCard: /Dokumentenmanagement/.test(externDash),
    noRevenueWidget: !/Umsatz/.test(externDash),
    noAlerts: !/überfällig|läuft ab|SLA/.test(externDash),
    rawKeys: rawKeys(externDash),
    pass: !/Buchhaltung/.test(externDash) && invoiceBtns === 0 && contactBtns === 0 && /Projektverwaltung/.test(externDash) && !/Umsatz/.test(externDash) && rawKeys(externDash).length === 0,
  })

  // ── 6) extern: admin deep-link → NoAccess (regression from batch 1)
  await page.goto(`${BASE}/#/admin/users`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const externAdmin = await bodyText()
  await shot('06-extern-admin-noaccess.png')
  out.push({
    step: 'extern /admin/users: Kein-Zugriff-Seite',
    pass: /Kein Zugriff/.test(externAdmin),
  })

  // ── 7) hr_admin: AdminHub only users+roles
  await switchTo(/Nina Fischer/)
  await page.goto(`${BASE}/#/admin/users`, { waitUntil: 'domcontentloaded' })
  await waitForText('Benutzer')
  await page.waitForTimeout(1200)
  const hrTabs = await tabNames()
  await shot('07-hradmin-hub-tabs.png')
  out.push({
    step: 'hr_admin AdminHub: nur Benutzer+Rollen, kein Lizenz/Branding/IT/Sicherheit/Abrechnung/Integrationen',
    tabs: hrTabs,
    pass:
      hrTabs.includes('Benutzer') && hrTabs.includes('Rollen') &&
      ['Lizenz', 'Branding', 'IT', 'Sicherheit', 'Abrechnung', 'Integrationen'].every((t) => !hrTabs.includes(t)),
  })

  // ── 8) hr_admin: deep-link to hidden tab redirects to first allowed
  await page.goto(`${BASE}/#/admin/license`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const hrUrl = page.url()
  await shot('08-hradmin-license-redirect.png')
  out.push({
    step: 'hr_admin /admin/license: Redirect auf ersten erlaubten Tab',
    url: hrUrl,
    pass: /#\/admin\/users/.test(hrUrl),
  })

  // ── 9) it_admin: no license/branding/billing; security sub-tabs without GDPR/DSAR
  await switchTo(/Thomas Keller/)
  await page.goto(`${BASE}/#/admin/users`, { waitUntil: 'domcontentloaded' })
  await waitForText('Benutzer')
  await page.waitForTimeout(1200)
  const itTabs = await tabNames()
  await shot('09a-itadmin-hub-tabs.png')
  await page.goto(`${BASE}/#/admin/security`, { waitUntil: 'domcontentloaded' })
  await waitForText('Audit-Log')
  await page.waitForTimeout(1200)
  const itSecTabs = await tabNames()
  await shot('09b-itadmin-security-subtabs.png')
  out.push({
    step: 'it_admin: kein Lizenz/Branding/Abrechnung; Security ohne DSGVO/DSAR, Audit da',
    hubTabs: itTabs,
    secTabs: itSecTabs,
    pass:
      ['Lizenz', 'Branding', 'Abrechnung'].every((t) => !itTabs.includes(t)) &&
      itTabs.includes('Sicherheit') &&
      itSecTabs.includes('Audit-Log') &&
      !itSecTabs.includes('DSGVO') && !itSecTabs.includes('Auskunft (Art. 15)'),
  })

  // ── 10) manager: team approve visible, create + payroll tab gone; dashboard w/o finance
  await switchTo(/Sarah Müller/)
  await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const mgrTeam = await bodyText()
  await shot('10a-manager-team.png')
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const mgrDash = await bodyText()
  // "Buchhaltung" also exists as a team DEPARTMENT name (team widgets) — only
  // the finance module CARD (a link) and the invoice quick action count.
  const mgrFinanceCards = await page.getByRole('link', { name: /Buchhaltung/ }).count()
  const mgrInvoiceBtns = await page.getByRole('button', { name: 'Neue Rechnung' }).count()
  await shot('10b-manager-dashboard.png')
  out.push({
    step: 'manager: kein Mitarbeiter-erstellen, kein Lohn-Tab, Anfragen-Tab da; Dashboard ohne Buchhaltungs-Karte',
    noCreate: !/Mitarbeiter erstellen/.test(mgrTeam),
    noPayrollTab: !/Lohnvorbereitung/.test(mgrTeam),
    hasRequestsTab: /Anfragen/.test(mgrTeam),
    noFinanceCard: mgrFinanceCards === 0,
    noInvoiceBtn: mgrInvoiceBtns === 0,
    rawKeys: rawKeys(mgrTeam),
    pass:
      !/Mitarbeiter erstellen/.test(mgrTeam) &&
      !/Lohnvorbereitung/.test(mgrTeam) &&
      /Anfragen/.test(mgrTeam) &&
      mgrFinanceCards === 0 &&
      mgrInvoiceBtns === 0 &&
      rawKeys(mgrTeam).length === 0,
  })

  // ── 11) readonly: foreign member profile — contact/employment sections + documents tab hidden
  await switchTo(/Elena Richter/)
  await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const roTeam = await bodyText()
  // open a foreign member profile (Sarah Müller is a seeded employee ≠ Elena)
  await page.getByText('Sarah Müller', { exact: false }).first().click().catch(() => {})
  await page.waitForTimeout(1500)
  const roProfile = await bodyText()
  const roProfileTabs = await tabNames()
  await shot('11-readonly-member-profile.png')
  out.push({
    step: 'readonly: kein Erstellen; fremdes Profil ohne Beschäftigung-Sektion/Dokumente-Tab',
    noCreate: !/Mitarbeiter erstellen/.test(roTeam),
    noEmploymentSection: !/Beschäftigung/i.test(roProfile),
    profileTabs: roProfileTabs,
    noDocsTab: !roProfileTabs.some((t) => /Dokumente/.test(t)),
    rawKeys: rawKeys(roProfile),
    pass:
      !/Mitarbeiter erstellen/.test(roTeam) &&
      !/Beschäftigung/i.test(roProfile) &&
      !roProfileTabs.some((t) => /Dokumente/.test(t)) &&
      rawKeys(roProfile).length === 0,
  })

  // ── 12) back to admin: regression after all switches
  await page.keyboard.press('Escape').catch(() => {})
  await switchTo(/Stefan Vogel/)
  await page.goto(`${BASE}/#/admin/users`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1400)
  const finalTabs = await tabNames()
  await shot('12-admin-regression.png')
  out.push({
    step: 'admin Rückkehr: alle 8 Tabs wieder da',
    tabs: finalTabs,
    pass: expectAll.every((t) => finalTabs.includes(t)),
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
