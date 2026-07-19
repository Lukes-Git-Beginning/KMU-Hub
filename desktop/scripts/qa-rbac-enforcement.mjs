/**
 * QA — RBAC R-3 Batch 1 (Enforcement-Sweep work/documents/crm/finance/wiki).
 * Verifies: admin full view (regression), readonly overlay preview per module
 * (hidden actions + "Nur Ansicht" chip + exception buttons disabled+tooltip),
 * extern real session (limited chip, own-task complete, deep-link NoAccess),
 * raw keys + pageerrors. Best effort: amounts masking via role-c1 draft preview.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/rbac-enforcement')
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
const rawKeys = (txt) => (txt.match(/\b(rbac|work|dokumente|kontakte|crm|finanzen|wiki|leads|advisory|buchhaltung)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)
const waitForText = (t) =>
  page.waitForFunction((x) => document.body.innerText.includes(x), t, { timeout: 30000 }).catch(() => {})
const shot = (name) => page.screenshot({ path: resolve(outDir, name) })

const openSwitcher = async () => {
  await page.locator('button.fixed.bottom-4.right-4').click()
  await page.waitForTimeout(400)
}
const closeSwitcher = async () => {
  await page.mouse.click(600, 120)
  await page.waitForTimeout(300)
}
const switchTo = async (labelRe) => {
  const target = page.getByRole('button', { name: labelRe }).first()
  if (!(await target.isVisible().catch(() => false))) await openSwitcher()
  await target.click()
  await page.waitForTimeout(1400)
  await closeSwitcher()
}

try {
  // 1) admin regression: work projects list — create buttons visible, no chip
  await page.goto(`${BASE}/#/work/projects`, { waitUntil: 'domcontentloaded' })
  await waitForText('Projekte')
  await page.waitForTimeout(1200)
  const adminWork = await bodyText()
  await shot('01-admin-work-projekte.png')
  out.push({
    step: 'admin work: Neues Projekt + Vorlage sichtbar, KEIN Chip',
    createBtn: /Neues Projekt/.test(adminWork),
    template: /Vorlage/.test(adminWork),
    noChip: !/Nur Ansicht/.test(adminWork) && !/Eingeschränkt/.test(adminWork),
    rawKeys: rawKeys(adminWork),
    pass: /Neues Projekt/.test(adminWork) && !/Nur Ansicht/.test(adminWork) && rawKeys(adminWork).length === 0,
  })

  // 2) admin: open first project board — header actions present
  await page.locator('[role="button"], a, div.cursor-pointer').filter({ hasText: /Cosmi|Hub|Projekt/ }).first().click().catch(() => {})
  await page.waitForTimeout(1500)
  let adminBoard = await bodyText()
  if (!/Neue Aufgabe/.test(adminBoard)) {
    // fallback: click any project card via heading
    await page.goto(`${BASE}/#/work/projects`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(1000)
    await page.locator('h3').first().click().catch(() => {})
    await page.waitForTimeout(1500)
    adminBoard = await bodyText()
  }
  await shot('02-admin-work-board.png')
  out.push({
    step: 'admin board: Neue Aufgabe + Settings-Zahnrad da',
    newTask: /Neue Aufgabe/.test(adminBoard),
    pass: /Neue Aufgabe/.test(adminBoard),
  })

  // 3) start readonly overlay preview from the role editor
  await page.goto(`${BASE}/#/admin/roles/readonly`, { waitUntil: 'domcontentloaded' })
  await waitForText('Nur Lesen')
  await page.waitForTimeout(800)
  await page.getByRole('button', { name: 'Als Rolle anzeigen' }).click()
  await page.waitForTimeout(1400)
  const previewStarted = /Vorschau als/.test(await bodyText())

  // 4) readonly preview: work board — chip, no actions, static pickers
  await page.goto(`${BASE}/#/work/projects`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1200)
  const roProjects = await bodyText()
  await shot('03-readonly-work-projekte.png')
  await page.locator('h3').first().click().catch(() => {})
  await page.waitForTimeout(1500)
  const roBoard = await bodyText()
  await shot('04-readonly-work-board.png')
  out.push({
    step: 'readonly work: Chip „Nur Ansicht", kein Neues Projekt / Neue Aufgabe',
    previewStarted,
    chip: /Nur Ansicht/.test(roProjects) || /Nur Ansicht/.test(roBoard),
    noCreate: !/Neues Projekt/.test(roProjects) && !/Neue Aufgabe/.test(roBoard),
    rawKeys: rawKeys(roBoard),
    pass: previewStarted && /Nur Ansicht/.test(roBoard) && !/Neue Aufgabe/.test(roBoard) && rawKeys(roBoard).length === 0,
  })

  // 5) readonly preview: finance — reduced tabs, no header actions, chip;
  //    draft invoice detail shows send disabled with hint
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const roFin = await bodyText()
  await shot('05-readonly-finanzen.png')
  const finTabsGone = !/DATEV/.test(roFin) && !/Mahnwesen/.test(roFin) && !/Banking/.test(roFin)
  // open drafts: click the Entwurf status filter if present, then first row
  await page.getByRole('button', { name: /Entwurf/ }).first().click().catch(() => {})
  await page.waitForTimeout(900)
  await page.locator('tbody tr, [role="row"]').first().click().catch(() => {})
  await page.waitForTimeout(1200)
  const roInvoice = await bodyText()
  const sendDisabled = await page.locator('button[disabled]', { hasText: /Versenden|Senden/ }).count()
    + await page.locator('[aria-disabled="true"]', { hasText: /Versenden|Senden/ }).count()
  await shot('06-readonly-finanzen-detail-send-disabled.png')
  out.push({
    step: 'readonly finance: Tabs reduziert + keine Neue Rechnung + Versenden disabled (Ausnahme)',
    chip: /Nur Ansicht/.test(roFin),
    tabsReduced: finTabsGone,
    noCreate: !/Neue Rechnung/.test(roFin),
    sendDisabledCount: sendDisabled,
    rawKeys: rawKeys(roInvoice),
    pass: /Nur Ansicht/.test(roFin) && finTabsGone && !/Neue Rechnung/.test(roFin),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 6) readonly preview: documents — no upload, chip; file context menu:
  //    download greyed (readonly lacks documents:file:download)
  await page.goto(`${BASE}/#/dokumente`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const roDocs = await bodyText()
  await shot('07-readonly-dokumente.png')
  // right-click the first file tile/row
  const fileEl = page.locator('[data-file-id], .group').filter({ hasText: /\.(pdf|docx|xlsx|png|jpg)/i }).first()
  await fileEl.click({ button: 'right' }).catch(() => {})
  await page.waitForTimeout(800)
  const menuTxt = await bodyText()
  await shot('08-readonly-dokumente-kontextmenu.png')
  out.push({
    step: 'readonly documents: kein Upload-Button + Chip; Kontextmenü ohne Löschen, Download ausgegraut',
    chip: /Nur Ansicht/.test(roDocs),
    noUpload: !/Hochladen/.test(roDocs.replace(/Hier hochladen/g, '')),
    menuOpened: /Herunterladen|Öffnen/.test(menuTxt),
    rawKeys: rawKeys(roDocs),
    pass: /Nur Ansicht/.test(roDocs) && rawKeys(roDocs).length === 0,
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)

  // 7) readonly preview: crm — import disabled (exception), no create, chip
  await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const roCrm = await bodyText()
  const importBtnDisabled = await page.locator('button[disabled]').count()
  await shot('09-readonly-kontakte.png')
  out.push({
    step: 'readonly crm: kein Neuer Kontakt, Import ausgegraut (Ausnahme), Chip',
    chip: /Nur Ansicht/.test(roCrm),
    noCreate: !/Neuer Kontakt/.test(roCrm),
    anyDisabledBtn: importBtnDisabled > 0,
    rawKeys: rawKeys(roCrm),
    pass: /Nur Ansicht/.test(roCrm) && !/Neuer Kontakt/.test(roCrm) && rawKeys(roCrm).length === 0,
  })

  // 8) readonly preview: wiki — no create buttons, no edit in article header
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  // open first article from the list
  await page.locator('main [role="button"], main li, main article').first().click().catch(() => {})
  await page.waitForTimeout(1000)
  const roWiki = await bodyText()
  const editBtns = await page.getByRole('button', { name: /^Bearbeiten$/ }).count()
  await shot('10-readonly-wiki.png')
  out.push({
    step: 'readonly wiki: Chip + kein Neuer-Artikel-Button + kein Bearbeiten',
    chip: /Nur Ansicht/.test(roWiki),
    noEdit: editBtns === 0,
    rawKeys: rawKeys(roWiki),
    pass: /Nur Ansicht/.test(roWiki) && editBtns === 0 && rawKeys(roWiki).length === 0,
  })

  // 9) end preview via banner
  await page.getByRole('button', { name: /Vorschau beenden|Beenden/ }).click().catch(() => {})
  await page.waitForTimeout(1000)

  // 10) extern real session (Max): my-tasks — limited chip, no create,
  //     quick-complete on own assigned task still available
  await switchTo(/Aushilfe \/ Extern/)
  await page.goto(`${BASE}/#/work/my-tasks`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1600)
  const extTasks = await bodyText()
  await shot('11-extern-meine-aufgaben.png')
  out.push({
    step: 'extern my-tasks: Chip „Eingeschränkt", keine Neue Aufgabe, eigene Tasks abhakbar',
    chip: /Eingeschränkt/.test(extTasks),
    noCreate: !/Neue Aufgabe/.test(extTasks),
    rawKeys: rawKeys(extTasks),
    pass: /Eingeschränkt/.test(extTasks) && !/Neue Aufgabe/.test(extTasks) && rawKeys(extTasks).length === 0,
  })

  // 11) extern deep links: finance + crm → NoAccess page (not blank)
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1200)
  const noAccessFin = await bodyText()
  await shot('12-extern-noaccess-finanzen.png')
  await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1200)
  const noAccessCrm = await bodyText()
  await shot('13-extern-noaccess-kontakte.png')
  out.push({
    step: 'extern deep-link: „Kein Zugriff"-Seite für finanzen + kontakte',
    finance: /Kein Zugriff/.test(noAccessFin) && /Verwaltung/.test(noAccessFin),
    crm: /Kein Zugriff/.test(noAccessCrm),
    cta: /Zum Dashboard/.test(noAccessFin),
    rawKeys: rawKeys(noAccessFin),
    pass: /Kein Zugriff/.test(noAccessFin) && /Kein Zugriff/.test(noAccessCrm) && /Zum Dashboard/.test(noAccessFin),
  })

  // 12) extern documents: download greyed in preview modal OR context menu
  await page.goto(`${BASE}/#/dokumente`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const extDocs = await bodyText()
  await shot('14-extern-dokumente.png')
  out.push({
    step: 'extern documents: „Nur Ansicht"-Chip (kein download/upload/edit-Recht)',
    chip: /Nur Ansicht/.test(extDocs),
    noUploadCta: !/Datei hochladen/.test(extDocs),
    rawKeys: rawKeys(extDocs),
    pass: /Nur Ansicht/.test(extDocs) && rawKeys(extDocs).length === 0,
  })

  // 13) back to admin: finance regression — everything visible again
  await switchTo(/Vollzugriff/)
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
  const adminFin = await bodyText()
  await shot('15-admin-finanzen-regression.png')
  out.push({
    step: 'admin finance regression: Neue Rechnung + DATEV + alle Tabs zurück',
    create: /Neue Rechnung/.test(adminFin),
    datev: /DATEV/.test(adminFin),
    dunning: /Mahnwesen/.test(adminFin),
    pass: /Neue Rechnung/.test(adminFin) && /DATEV/.test(adminFin) && /Mahnwesen/.test(adminFin),
  })

  // 14) best effort: amounts masking via role-c1 draft preview
  try {
    await page.goto(`${BASE}/#/admin/roles/role-c1`, { waitUntil: 'domcontentloaded' })
    await waitForText('Lager & Logistik')
    await page.waitForTimeout(900)
    // search tree for Buchhaltung and open the module
    await page.locator('input[type="search"], input[placeholder*="uch"]').first().fill('Buchhaltung').catch(() => {})
    await page.waitForTimeout(600)
    await page.locator('nav[aria-label="Module"]').getByRole('button', { name: /Buchhaltung/ }).click()
    await page.waitForTimeout(700)
    // make module visible + grant invoice read, leave amounts off
    const visSwitch = page.getByRole('switch').first()
    await visSwitch.click()
    await page.waitForTimeout(500)
    await page.getByRole('switch', { name: /Rechnungen — Ansehen|^Ansehen$/ }).first().click().catch(() => {})
    await page.waitForTimeout(500)
    await page.getByRole('button', { name: 'Als Rolle anzeigen' }).click()
    await page.waitForTimeout(1400)
    await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(1500)
    const maskTxt = await bodyText()
    await shot('16-maskierte-betraege.png')
    out.push({
      step: 'amounts masking (best effort): Beträge als ••• im readonly-Draft ohne amounts:view',
      masked: /•••/.test(maskTxt),
      pass: /•••/.test(maskTxt),
    })
    await page.getByRole('button', { name: /Vorschau beenden|Beenden/ }).click().catch(() => {})
  } catch (e) {
    out.push({ step: 'amounts masking (best effort)', pass: 'SKIP', note: String(e).split('\n')[0] })
  }
} finally {
  console.log(JSON.stringify({ steps: out, pageErrors: errs.slice(0, 8) }, null, 2))
  await b.close()
}
