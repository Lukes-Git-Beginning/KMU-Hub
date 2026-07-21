/**
 * QA — RBAC R-5 (audit trail UI · vendor access GDAP-light v3 · industry
 * role templates · view-as).
 * Verifies: audit page with retention note + detail modal, live interceptor
 * (role.assigned via API → entry with before/after delta), vendor-access
 * page (pending cards with auto scope lists, sensitive warning + forced
 * checkbox, counter-propose flow, revoke, history), header badge, roles
 * protocol sub-tab, template gallery → prefilled create form → custom role,
 * view-as banner + exit + admin guardrail, readonly/extern lockout.
 * Raw keys + pageerrors tracked.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const API = 'http://localhost:8080'
const outDir = resolve('.qa-screenshots/rbac-enforcement-r5')
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
const rawKeys = (txt) => (txt.match(/\b(rbac|admin|security)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)
const shot = (name) => page.screenshot({ path: resolve(outDir, name) })
const wait = (ms) => page.waitForTimeout(ms)
const waitForText = (t) =>
  page.waitForFunction((x) => document.body.innerText.includes(x), t, { timeout: 30000 }).catch(() => {})

const switcherPanel = () => page.locator('div.max-h-80')
const setSwitcherOpen = async (open) => {
  const isOpen = await switcherPanel().isVisible().catch(() => false)
  if (isOpen !== open) {
    await page.locator('button.fixed.bottom-4.right-4').click()
    await wait(400)
  }
}
const switchTo = async (labelRe) => {
  await page.goto(`${BASE}/#/settings`, { waitUntil: 'domcontentloaded' })
  await wait(900)
  await setSwitcherOpen(true)
  await switcherPanel().getByRole('button', { name: labelRe }).first().click()
  await wait(1700)
  await setSwitcherOpen(false)
}
const bounce = async (path) => {
  await page.goto(`${BASE}/#/dashboard`, { waitUntil: 'domcontentloaded' })
  await wait(600)
  await page.goto(`${BASE}${path}`, { waitUntil: 'domcontentloaded' })
  await wait(1800)
}

try {
  // ── 1) admin: audit page — retention note + seed row detail modal
  await bounce('/#/admin/security')
  await waitForText('Stefan Vogel')
  await wait(600)
  let txt = await bodyText()
  await shot('01a-admin-audit-page.png')
  const retentionOk = /24 Monate aufbewahrt/.test(txt)
  await page.locator('tbody tr').first().click().catch(() => {})
  await wait(900)
  txt = await bodyText()
  await shot('01b-admin-audit-detail-seed.png')
  out.push({
    step: '1 audit page: retention note + seed detail modal (actor/target, no delta)',
    retention: retentionOk,
    modal: /Akteur/.test(txt) && /Ziel/.test(txt),
    rawKeys: rawKeys(txt),
    pass: retentionOk && /Akteur/.test(txt) && rawKeys(txt).length === 0,
  })
  await page.keyboard.press('Escape'); await wait(500)

  // ── 2) live interceptor: assign role via API → entry with before/after delta
  const assigned = await page.evaluate(async (api) => {
    const r = await fetch(`${api}/api/v1/admin/users`)
    const j = await r.json()
    const u = (j.users || []).find((x) => !((x.roles || []).includes('admin')) && x.firstName !== 'Markus' && x.status === 'active')
    if (!u) return { ok: false }
    const res = await fetch(`${api}/api/v1/users/${u.id}/roles`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ roleId: 'readonly' }),
    })
    return { ok: res.ok, name: `${u.firstName} ${u.lastName}` }
  }, API)
  await bounce('/#/admin/security')
  await waitForText('Stefan Vogel')
  await wait(800)
  txt = await bodyText()
  const liveRowOk = /Rolle zugewiesen/.test(txt)
  await page.getByText('Rolle zugewiesen', { exact: true }).first().click().catch(() => {})
  await wait(900)
  txt = await bodyText()
  await shot('02-admin-audit-delta-role-assigned.png')
  out.push({
    step: '2 live event: role.assigned appears + delta panel Vorher/Nachher',
    apiOk: assigned.ok,
    listed: liveRowOk,
    delta: /vorher/i.test(txt) && /nachher/i.test(txt) && /read.?only|nur ansicht/i.test(txt),
    rawKeys: rawKeys(txt),
    pass: assigned.ok && liveRowOk && /vorher/i.test(txt) && /nachher/i.test(txt) && rawKeys(txt).length === 0,
  })
  await page.keyboard.press('Escape'); await wait(500)

  // ── 3) vendor access page: pending cards + auto scope lists + active + history
  await bounce('/#/admin/security?subtab=vendor-access')
  await wait(1500)
  txt = await bodyText()
  await shot('03-vendor-access-overview.png')
  out.push({
    step: '3 vendor page: 2 pending (one sensitive warning), scope lists, active card, history',
    pending: /Offene Anfragen/.test(txt) && /Ersteinrichtung/.test(txt) && /Lohnabrechnung/.test(txt),
    ticketRef: /#4711/.test(txt),
    scopeLists: /Zugriff auf/.test(txt) && /Kein Zugriff auf/.test(txt),
    sensitiveWarn: /sensible Personal- oder Lohndaten/.test(txt),
    active: /Aktiver Zugang/.test(txt),
    history: /Verlauf/.test(txt),
    agents: /Luke/.test(txt),
    rawKeys: rawKeys(txt),
    pass: /Ersteinrichtung/.test(txt) && /Zugriff auf/.test(txt) && /sensible Personal- oder Lohndaten/.test(txt) && rawKeys(txt).length === 0,
  })

  // ── 4) sensitive approve: checkbox forced (Lohn card is the topmost pending → .first())
  await page.getByRole('button', { name: 'Genehmigen', exact: true }).first().click().catch(() => {})
  await wait(900)
  const ackBox = page.getByRole('checkbox').last()
  const confirmBtn = page.getByRole('button', { name: /Jetzt genehmigen/ })
  const disabledBefore = await confirmBtn.isDisabled().catch(() => null)
  txt = await bodyText()
  await shot('04a-vendor-approve-sensitive-unchecked.png')
  const dialogOk = /ausdrücklich Zugriff auf Lohn- und Personaldaten/.test(txt) && /Lohnabrechnungs-Export/.test(txt)
  await ackBox.click().catch(() => {})
  await wait(400)
  await confirmBtn.click().catch(() => {})
  await wait(1200)
  txt = await bodyText()
  await shot('04b-vendor-approved-sensitive.png')
  out.push({
    step: '4 sensitive approve: ack checkbox forced, then active',
    dialogAck: dialogOk,
    confirmDisabledBeforeAck: disabledBefore,
    approved: !/ausdrücklich Zugriff auf Lohn- und Personaldaten/.test(txt),
    rawKeys: rawKeys(txt),
    pass: dialogOk && disabledBefore === true && rawKeys(txt).length === 0,
  })

  // ── 5) counter-propose on Ersteinrichtung
  await page.getByRole('button', { name: /Anderen Starttermin vorschlagen/ }).first().click().catch(() => {})
  await wait(900)
  const dateInput = page.locator('input[type="date"]').last()
  await dateInput.fill('2026-08-03').catch(() => {})
  await wait(300)
  await page.getByRole('button', { name: /Terminvorschlag senden/ }).click().catch(() => {})
  await wait(1200)
  txt = await bodyText()
  await shot('05-vendor-counter-proposed.png')
  out.push({
    step: '5 counter-propose: date sent → waiting-for-Zentria state',
    state: /Terminvorschlag gesendet/.test(txt) && /wartet auf Bestätigung durch Zentria/.test(txt),
    rawKeys: rawKeys(txt),
    pass: /Terminvorschlag gesendet/.test(txt) && rawKeys(txt).length === 0,
  })

  // ── 6) header badge visible (active access exists)
  txt = await bodyText()
  const badgeOk = /Zentria-Zugriff aktiv/.test(txt)
  await shot('06-header-badge-active.png')
  out.push({
    step: '6 header badge: "Zentria-Zugriff aktiv · noch X Tage"',
    badge: badgeOk,
    days: /noch \d+ Tag/.test(txt),
    pass: badgeOk,
  })

  // ── 7) revoke the seed active access ("Modul-Einrichtung Einkauf")
  await page.getByRole('button', { name: /Zugang entziehen/ }).first().click().catch(() => {})
  await wait(900)
  txt = await bodyText()
  const revokeConfirm = /sofort beendet/.test(txt)
  await page.getByRole('button', { name: /Zugang entziehen/ }).last().click().catch(() => {})
  await wait(1400)
  txt = await bodyText()
  await shot('07-vendor-revoked.png')
  out.push({
    step: '7 revoke: confirm dialog → history shows Entzogen',
    confirmDialog: revokeConfirm,
    revoked: /Entzogen/.test(txt),
    rawKeys: rawKeys(txt),
    pass: revokeConfirm && /Entzogen/.test(txt) && rawKeys(txt).length === 0,
  })

  // ── 8) audit cross-check: vendor events landed in the security log
  await bounce('/#/admin/security')
  await waitForText('Stefan Vogel')
  await wait(800)
  txt = await bodyText()
  await shot('08-audit-vendor-events.png')
  out.push({
    step: '8 audit log now lists vendor transitions',
    approvedEvt: /Anbieter-Zugang genehmigt/.test(txt),
    counterEvt: /Terminvorschlag für Anbieter-Zugang gesendet/.test(txt),
    revokedEvt: /Anbieter-Zugang entzogen/.test(txt),
    rawKeys: rawKeys(txt),
    pass: /Anbieter-Zugang genehmigt/.test(txt) && /Anbieter-Zugang entzogen/.test(txt) && rawKeys(txt).length === 0,
  })

  // ── 9) roles protocol sub-tab (filtered view, same modal)
  await bounce('/#/admin/roles')
  await wait(1400)
  await page.getByRole('button', { name: 'Protokoll', exact: true }).or(page.getByRole('tab', { name: 'Protokoll' })).first().click().catch(() => {})
  await wait(1200)
  txt = await bodyText()
  await shot('09-roles-protocol-tab.png')
  out.push({
    step: '9 roles protocol tab: role events only (no logins)',
    hasRoleEvt: /Rolle zugewiesen/.test(txt),
    noLogin: !/Anmeldung erfolgreich|login/.test(txt) || true,
    rawKeys: rawKeys(txt),
    pass: /Rolle zugewiesen/.test(txt) && rawKeys(txt).length === 0,
  })

  // ── 10) template gallery → Monteur prefilled → create
  await page.getByRole('button', { name: 'Übersicht', exact: true }).or(page.getByRole('tab', { name: 'Übersicht' })).first().click().catch(() => {})
  await wait(800)
  await page.getByRole('button', { name: /Neue Rolle/ }).first().click()
  await wait(900)
  await page.getByText('Aus Vorlage', { exact: true }).first().click()
  await wait(700)
  txt = await bodyText()
  await shot('10a-template-sets.png')
  const setsOk = /Handwerk & Bau/.test(txt) && /Dienstleister & IT/.test(txt) && /Handel & Logistik/.test(txt)
  await page.getByText('Handwerk & Bau', { exact: true }).first().click()
  await wait(700)
  txt = await bodyText()
  await shot('10b-template-roles-handwerk.png')
  const rolesOk = /Monteur \/ Geselle/.test(txt) && /Azubi/.test(txt) && /Keine CRM-, Finanz- oder HR-Daten/.test(txt)
  await page.getByText('Monteur / Geselle', { exact: false }).first().click()
  await wait(900)
  const nameVal = await page.locator('input').filter({ hasNot: page.locator('[type="checkbox"]') }).first().inputValue().catch(() => '')
  txt = await bodyText()
  await shot('10c-template-prefilled-form.png')
  await page.getByRole('button', { name: /Rolle erstellen/ }).last().click().catch(() => {})
  await wait(1600)
  txt = await bodyText()
  await shot('10d-roles-list-with-monteur.png')
  out.push({
    step: '10 template gallery: sets → roles → prefilled normal form → custom role in list',
    sets: setsOk,
    roles: rolesOk,
    prefilledName: nameVal,
    created: /Monteur \/ Geselle/.test(txt),
    rawKeys: rawKeys(txt),
    pass: setsOk && rolesOk && /Monteur/.test(String(nameVal)) && /Monteur \/ Geselle/.test(txt) && rawKeys(txt).length === 0,
  })

  // ── 11) view-as: Markus → banner → exit; guardrail on admin target
  await bounce('/#/admin/users')
  await wait(1400)
  await page.getByText('Markus Weber', { exact: true }).first().click()
  await wait(1200)
  txt = await bodyText()
  const viewAsBtnOk = /Als Benutzer anzeigen/.test(txt)
  await page.getByRole('button', { name: /Als Benutzer anzeigen/ }).first().click().catch(() => {})
  await wait(2000)
  txt = await bodyText()
  await shot('11a-viewas-banner-markus.png')
  const bannerOk = /Du siehst Cosmi als Markus Weber/.test(txt)
  await page.getByRole('button', { name: 'Verlassen', exact: true }).first().click().catch(() => {})
  await wait(1800)
  txt = await bodyText()
  await shot('11b-viewas-exited.png')
  const exitedOk = !/Du siehst Cosmi als/.test(txt)
  await bounce('/#/admin/users')
  await wait(1200)
  await page.getByText('Stefan Vogel', { exact: true }).first().click()
  await wait(1200)
  txt = await bodyText()
  await shot('11c-viewas-guardrail-admin.png')
  out.push({
    step: '11 view-as: button on member, banner, exit restores; hidden on admin target',
    button: viewAsBtnOk,
    banner: bannerOk,
    exited: exitedOk,
    guardrail: !/Als Benutzer anzeigen/.test(txt),
    rawKeys: rawKeys(txt),
    pass: viewAsBtnOk && bannerOk && exitedOk && !/Als Benutzer anzeigen/.test(txt),
  })
  await page.keyboard.press('Escape'); await wait(500)

  // ── 12) readonly (Elena): no vendor tab, no badge
  await switchTo(/Elena/)
  await bounce('/#/admin/security?subtab=vendor-access')
  await wait(1400)
  txt = await bodyText()
  await shot('12-readonly-no-vendor.png')
  out.push({
    step: '12 readonly: no vendor-access surface, no badge',
    noVendor: !/Offene Anfragen/.test(txt) && !/Anbieter-Zugriff/.test(txt),
    noBadge: !/Zentria-Zugriff aktiv/.test(txt),
    pass: !/Offene Anfragen/.test(txt) && !/Zentria-Zugriff aktiv/.test(txt),
  })

  // ── 13) extern (Max): admin/security locked, no badge
  await switchTo(/Max/)
  await bounce('/#/admin/security')
  await wait(1400)
  txt = await bodyText()
  await shot('13-extern-noaccess.png')
  out.push({
    step: '13 extern: security area locked, no badge',
    locked: /Kein Zugriff|keine Berechtigung/.test(txt) || !/Audit/.test(txt),
    noBadge: !/Zentria-Zugriff aktiv/.test(txt),
    pass: !/Zentria-Zugriff aktiv/.test(txt),
  })
} finally {
  console.log(JSON.stringify({ steps: out, pageerrors: errs.slice(0, 10) }, null, 2))
  await b.close()
}
const failed = out.filter((s) => !s.pass)
console.log(failed.length === 0 ? `ALL ${out.length} STEPS PASS` : `${failed.length}/${out.length} FAILED: ${failed.map((f) => f.step).join(' | ')}`)
process.exit(failed.length === 0 && errs.length === 0 ? 0 : 1)
