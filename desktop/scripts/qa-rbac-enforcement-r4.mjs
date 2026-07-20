/**
 * QA — RBAC R-4 (HR data category depth).
 * Verifies: admin regression (full list, drawers, edit pencils, offboard entry
 * inside the profile only), hr_admin change-request inbox (old/new card),
 * it_admin full directory but ZERO drawers (market-gap promise), manager
 * reporting-line scope (sees report's file, not the boss's, never salary),
 * member own scope (self-service with propose flow + pending lock + own
 * payslips, foreign drawers closed, restricted directory), extern NoAccess.
 * Raw keys + pageerrors tracked.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/rbac-enforcement-r4')
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
const rawKeys = (txt) => (txt.match(/\b(rbac|team|api)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)
const waitForText = (t) =>
  page.waitForFunction((x) => document.body.innerText.includes(x), t, { timeout: 30000 }).catch(() => {})
const shot = (name) => page.screenshot({ path: resolve(outDir, name) })

const switcherPanel = () => page.locator('div.max-h-80')
const setSwitcherOpen = async (open) => {
  const isOpen = await switcherPanel().isVisible().catch(() => false)
  if (isOpen !== open) {
    await page.locator('button.fixed.bottom-4.right-4').click()
    await page.waitForTimeout(400)
  }
}
const switchTo = async (labelRe) => {
  await page.goto(`${BASE}/#/settings`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(900)
  await setSwitcherOpen(true)
  await switcherPanel().getByRole('button', { name: labelRe }).first().click()
  await page.waitForTimeout(1700)
  await setSwitcherOpen(false)
}
const gotoTeam = async () => {
  await page.goto(`${BASE}/#/dashboard`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(600)
  await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
}
const openProfile = async (name) => {
  await page.getByText(name, { exact: true }).first().click()
  await page.waitForTimeout(1400)
}
const closeDialog = async () => {
  await page.keyboard.press('Escape')
  await page.waitForTimeout(600)
}

try {
  // ── 1) admin: full list, no restricted hint
  await gotoTeam()
  await waitForText('Mitglieder')
  await waitForText('Stefan Vogel')
  await page.waitForTimeout(1500)
  const adminList = await bodyText()
  await shot('01-admin-team-list.png')
  out.push({
    step: 'admin list: all employees, no restricted hint',
    seesLeaf: /Jonas Schmitt/.test(adminList) && /Petra Zimmermann/.test(adminList),
    noHint: !/Eingeschränkte Ansicht/.test(adminList),
    rawKeys: rawKeys(adminList),
    pass: /Jonas Schmitt/.test(adminList) && !/Eingeschränkte Ansicht/.test(adminList) && rawKeys(adminList).length === 0,
  })

  // ── 2) admin: Markus profile — drawers + pencils + offboard entry
  await openProfile('Markus Weber')
  const adminProfile = await bodyText()
  const editPencils = await page.locator('button[aria-label*="earbeiten"], button:has(svg.lucide-pencil)').count()
  await shot('02-admin-profile-markus.png')
  out.push({
    step: 'admin profile: contact+employment+absence drawers, edit pencils, actions menu',
    contact: /Kontakt/.test(adminProfile),
    employment: /Beschäftigung|Anstellung/.test(adminProfile),
    absences: /Abwesenheiten/.test(adminProfile),
    pencils: editPencils,
    rawKeys: rawKeys(adminProfile),
    pass: /Kontakt/.test(adminProfile) && editPencils > 0 && rawKeys(adminProfile).length === 0,
  })

  // ── 3) admin: offboard dialog on Martin Wolf (has report Jonas → successor required)
  await closeDialog()
  await gotoTeam()
  await openProfile('Martin Wolf')
  const menuBtn = page.locator('button[aria-haspopup="menu"]').last()
  await menuBtn.click().catch(() => {})
  await page.waitForTimeout(500)
  await page.getByText('Austritt einleiten').first().click().catch(() => {})
  await page.waitForTimeout(900)
  let offb = await bodyText()
  await shot('03a-admin-offboard-step1.png')
  const step1ok = /Letzter Arbeitstag/.test(offb) && /Austrittsart/.test(offb)
  // advance to confirmation step if a next/confirm control exists
  const nextBtn = page.getByRole('button', { name: /Weiter|Austritt bestätigen|Fortfahren/ }).first()
  await nextBtn.click().catch(() => {})
  await page.waitForTimeout(800)
  offb = await bodyText()
  await shot('03b-admin-offboard-step2.png')
  out.push({
    step: 'admin offboard dialog (Martin Wolf): fields + consequences + dependents warning + successor select',
    step1: step1ok,
    consequences: /Login wird am|Lizenzplatz|Rollen und Rechte/.test(offb),
    dependents: /berichtet an diese Person|berichten an diese Person/.test(offb),
    successor: /Verantwortung übernimmt/.test(offb),
    rawKeys: rawKeys(offb),
    pass: step1ok && /Lizenzplatz/.test(offb) && /Verantwortung übernimmt/.test(offb) && rawKeys(offb).length === 0,
  })
  await closeDialog(); await closeDialog()

  // ── 4) hr_admin (Nina): change-request inbox with Markus' pending card
  await switchTo(/Nina/)
  await gotoTeam()
  await page.getByText(/Anfragen \(/).last().click().catch(() => {})
  await page.waitForTimeout(1400)
  const inbox = await bodyText()
  await shot('04-hradmin-inbox.png')
  out.push({
    step: 'hr_admin inbox: pending card from Markus with old/new + approve/reject',
    title: /Profil-Änderungsanträge/.test(inbox),
    markus: /Markus Weber/.test(inbox),
    oldNew: /Alt/.test(inbox) && /Neu/.test(inbox),
    actions: /Genehmigen/.test(inbox) && /Ablehnen/.test(inbox),
    rawKeys: rawKeys(inbox),
    pass: /Profil-Änderungsanträge/.test(inbox) && /Genehmigen/.test(inbox) && rawKeys(inbox).length === 0,
  })

  // ── 5) it_admin (Thomas): full directory, ZERO drawers, no offboard
  await switchTo(/Thomas/)
  await gotoTeam()
  const itList = await bodyText()
  const itFull = /Jonas Schmitt/.test(itList) && !/Eingeschränkte Ansicht/.test(itList)
  await openProfile('Markus Weber')
  const itProfile = await bodyText()
  await shot('05-itadmin-profile-markus.png')
  out.push({
    step: 'it_admin: full directory but NO drawers on foreign profile (market-gap promise), no offboard',
    fullDirectory: itFull,
    noContact: !/Notfallkontakt/.test(itProfile),
    noSalaryTab: !/Lohndaten|Gehalt/.test(itProfile),
    noOffboard: !/Austritt einleiten/.test(itProfile),
    rawKeys: rawKeys(itProfile),
    pass: itFull && !/Notfallkontakt/.test(itProfile) && !/Austritt einleiten/.test(itProfile) && rawKeys(itProfile).length === 0,
  })
  await closeDialog()

  // ── 6) manager (Sarah): restricted list; report's drawers open, boss closed, never salary
  await switchTo(/Sarah/)
  await gotoTeam()
  const mgrList = await bodyText()
  await shot('06a-manager-list.png')
  await openProfile('Tim Hartmann')
  const mgrReport = await bodyText()
  await shot('06b-manager-profile-tim.png')
  await closeDialog()
  await openProfile('Stefan Vogel')
  const mgrBoss = await bodyText()
  await shot('06c-manager-profile-stefan.png')
  await closeDialog()
  out.push({
    step: 'manager: restricted directory hint; Tim (report) drawers open without salary; Stefan (boss) drawers closed',
    hint: /Eingeschränkte Ansicht/.test(mgrList),
    seesTim: /Tim Hartmann/.test(mgrList),
    hidesLeafOutside: !/Jonas Schmitt/.test(mgrList),
    timContact: /Kontakt/.test(mgrReport),
    timNoSalary: !/Lohndaten|Bezüge/.test(mgrReport),
    timNoPayslipDocs: !/Gehaltsabrechnung_/.test(mgrReport),
    bossClosed: !/Notfallkontakt/.test(mgrBoss),
    rawKeys: rawKeys(mgrList).concat(rawKeys(mgrReport)),
    pass: /Eingeschränkte Ansicht/.test(mgrList) && /Kontakt/.test(mgrReport) && !/Jonas Schmitt/.test(mgrList) && !/Notfallkontakt/.test(mgrBoss) && !/Gehaltsabrechnung_/.test(mgrReport),
  })

  // ── 7) member (Markus): self-service own data + propose + pending lock + payslips
  await switchTo(/Markus/)
  await gotoTeam()
  const memList = await bodyText()
  await shot('07a-member-list.png')
  const ssTab = page.getByRole('tab', { name: /Self|Selbst/i }).first()
  if (await ssTab.count()) { await ssTab.click() } else { await page.getByText(/Self-Service|Selbstservice/i).first().click().catch(() => {}) }
  await page.waitForTimeout(1400)
  const ss = await bodyText()
  await shot('07b-member-selfservice.png')
  await page.getByText('Gehaltsabrechnungen', { exact: true }).first().click().catch(() => {})
  await page.waitForTimeout(1200)
  const payslips = await bodyText()
  await shot('07c-member-payslips.png')
  out.push({
    step: 'member self-service: own profile (Markus), propose buttons, pending lock on mobile, own payslips visible',
    restrictedList: /Eingeschränkte Ansicht/.test(memList),
    ownName: /Markus Weber/.test(ss),
    propose: /Ändern/.test(ss),
    pendingLock: /Änderung ausstehend/.test(ss),
    payslips: /Brutto|Netto|abrechnung/.test(payslips),
    rawKeys: rawKeys(ss).concat(rawKeys(payslips)),
    pass: /Markus Weber/.test(ss) && /Änderung ausstehend/.test(ss) && rawKeys(ss).length === 0,
  })

  // ── 8) member: foreign profile drawers closed
  await gotoTeam()
  await waitForText('Laura Neumann')
  await openProfile('Laura Neumann')
  const memForeign = await bodyText()
  await shot('08-member-profile-laura.png')
  out.push({
    step: 'member foreign profile (Laura): personal/job/salary drawers closed, no pencils, no offboard',
    noContact: !/Notfallkontakt/.test(memForeign),
    noSalary: !/Lohndaten|Bezüge/.test(memForeign),
    noOffboard: !/Austritt einleiten/.test(memForeign),
    rawKeys: rawKeys(memForeign),
    pass: !/Notfallkontakt/.test(memForeign) && !/Austritt einleiten/.test(memForeign) && rawKeys(memForeign).length === 0,
  })
  await closeDialog()

  // ── 9) extern (Max): /#/team deep link → NoAccess
  await switchTo(/Max/)
  await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const ext = await bodyText()
  await shot('09-extern-team-noaccess.png')
  out.push({
    step: 'extern: /#/team deep link shows NoAccess view',
    noAccess: /Kein Zugriff|kein Zugriff/.test(ext),
    noList: !/Jonas Schmitt/.test(ext),
    rawKeys: rawKeys(ext),
    pass: /ein Zugriff/.test(ext) && !/Jonas Schmitt/.test(ext),
  })
} finally {
  console.log(JSON.stringify({ steps: out, pageerrors: errs.slice(0, 10) }, null, 2))
  const failed = out.filter((s) => !s.pass)
  console.log(`\n${out.length} steps, ${failed.length} FAILED, ${errs.length} pageerrors`)
  await b.close()
}
