/**
 * QA-Nachtest R-3 Batch 1 — die drei offenen Punkte aus dem Hauptlauf:
 * 1) admin work: Buttons nach echtem Laden (Kaltstart-Timing)
 * 2) readonly-Preview finance: Rechnungen-Tab → Zeilen-Menü → Versenden grayed
 * 3) readonly-Preview documents: Kontextmenü-Einträge als Text-Dump (Download grayed?)
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
const waitForText = (t, ms = 45000) =>
  page.waitForFunction((x) => document.body.innerText.includes(x), t, { timeout: ms }).catch(() => {})
const shot = (name) => page.screenshot({ path: resolve(outDir, name) })

try {
  // 1) admin work — wait for the actual button, not the skeleton
  await page.goto(`${BASE}/#/work/projects`, { waitUntil: 'domcontentloaded' })
  await waitForText('Neues Projekt')
  await page.waitForTimeout(500)
  const adminWork = await bodyText()
  await shot('01b-admin-work-projekte.png')
  await page.locator('h3').first().click().catch(() => {})
  await waitForText('Neue Aufgabe', 20000)
  const adminBoard = await bodyText()
  await shot('02b-admin-work-board.png')
  out.push({
    step: 'admin work (Nachtest): Buttons nach Laden',
    createBtn: /Neues Projekt/.test(adminWork),
    template: /Vorlage/.test(adminWork),
    newTask: /Neue Aufgabe/.test(adminBoard),
    pass: /Neues Projekt/.test(adminWork) && /Neue Aufgabe/.test(adminBoard),
  })

  // start readonly preview
  await page.goto(`${BASE}/#/admin/roles/readonly`, { waitUntil: 'domcontentloaded' })
  await waitForText('Als Rolle anzeigen')
  await page.getByRole('button', { name: 'Als Rolle anzeigen' }).click()
  await page.waitForTimeout(1400)

  // 2) finance invoices tab → row actions menu → send greyed
  await page.goto(`${BASE}/#/finanzen`, { waitUntil: 'domcontentloaded' })
  await waitForText('Buchhaltung')
  await page.waitForTimeout(800)
  await page.getByRole('button', { name: /Rechnungen/ }).first().click()
  await page.waitForTimeout(1200)
  const invoiceTab = await bodyText()
  await shot('06b-readonly-rechnungen-tab.png')
  // open the first row's three-dot menu (ItemActions trigger)
  const menuBtn = page.locator('table button, [role="row"] button').last()
  await menuBtn.click().catch(() => {})
  await page.waitForTimeout(700)
  const menuState = await page.evaluate(() => {
    const items = Array.from(document.querySelectorAll('[role="menuitem"]'))
    return items.map((el) => ({
      label: el.textContent?.trim(),
      ariaDisabled: el.getAttribute('aria-disabled'),
      dataDisabled: el.hasAttribute('data-disabled'),
      title: el.getAttribute('title'),
    }))
  })
  await shot('06c-readonly-rechnung-menu.png')
  const sendItem = menuState.find((i) => /Versenden|Senden/i.test(i.label ?? ''))
  const editItem = menuState.find((i) => /Bearbeiten/i.test(i.label ?? ''))
  const deleteItem = menuState.find((i) => /Storno|Löschen|Abbrechen/i.test(i.label ?? ''))
  out.push({
    step: 'readonly finance Rechnungen: Menü — Versenden grayed+Hinweis, Edit/Storno WEG',
    menuItems: menuState.map((i) => `${i.label}${i.ariaDisabled === 'true' || i.dataDisabled ? ' [disabled]' : ''}`),
    sendGreyed: sendItem ? sendItem.ariaDisabled === 'true' || sendItem.dataDisabled : 'kein draft in Zeile?',
    sendHint: sendItem?.title ?? null,
    editGone: !editItem,
    deleteGone: !deleteItem,
    pass: !editItem && !deleteItem && (sendItem ? (sendItem.ariaDisabled === 'true' && !!sendItem.title) : true),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)

  // 2b) if no draft row was hit, note the tab text for manual eval
  out.push({ note: 'Rechnungen-Tab enthält Entwurf-Zeilen', hasDraft: /Entwurf/.test(invoiceTab) })

  // 3) documents context menu text dump
  await page.goto(`${BASE}/#/dokumente`, { waitUntil: 'domcontentloaded' })
  await waitForText('Dokumente', 20000)
  await page.waitForTimeout(1200)
  const tile = page.locator('.group').filter({ hasText: /\.(pdf|docx|xlsx|png|jpg)/i }).first()
  await tile.click({ button: 'right' }).catch(() => {})
  await page.waitForTimeout(700)
  const docMenu = await page.evaluate(() => {
    const root = Array.from(document.querySelectorAll('[role="menu"], .fixed.z-50, [data-context-menu]')).pop()
    if (!root) return { found: false, items: [] }
    return {
      found: true,
      items: Array.from(root.querySelectorAll('button, [role="menuitem"]')).map((el) => ({
        label: el.textContent?.trim(),
        ariaDisabled: el.getAttribute('aria-disabled'),
        title: el.getAttribute('title'),
        cls: (el.getAttribute('class') || '').includes('opacity-50'),
      })),
    }
  })
  await shot('08b-readonly-doc-kontextmenu.png')
  const dl = docMenu.items.find((i) => /Herunterladen/i.test(i.label ?? ''))
  out.push({
    step: 'readonly documents Kontextmenü: Download grayed+Tooltip (Ausnahme), Edit-Einträge weg',
    menuFound: docMenu.found,
    items: docMenu.items.map((i) => `${i.label}${i.cls || i.ariaDisabled === 'true' ? ' [grayed]' : ''}`),
    downloadPresent: !!dl,
    downloadGreyed: dl ? dl.cls || dl.ariaDisabled === 'true' : false,
    downloadHint: dl?.title ?? null,
    editGone: !docMenu.items.some((i) => /Umbenennen|Verschieben|Löschen|Freigeben/i.test(i.label ?? '')),
    pass: docMenu.found && !!dl && (dl.cls || dl.ariaDisabled === 'true') && !!dl.title,
  })
} finally {
  console.log(JSON.stringify({ steps: out, pageErrors: errs.slice(0, 8) }, null, 2))
  await b.close()
}
