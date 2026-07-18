/**
 * QA — RBAC R-2 Rollen-Baukasten.
 * Verifies: Verwaltung nav entry + builder list (presets/custom/deviation
 * badge), preset editor read-only, custom editor draft + discard + apply
 * (confirm dialog), clone flow → editor, compare modal diff, overlay preview
 * (banner + nav shrink + end), team member roles section + effective-rights
 * modal, A-1 multi-role chips, delete guardrail dialog, raw keys + pageerrors.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const outDir = resolve('.qa-screenshots/rbac-baukasten')
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
const rawKeys = (txt) => (txt.match(/\b(rbac|profil|admin|team)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 6)
const waitForText = (t) =>
  page.waitForFunction((x) => document.body.innerText.includes(x), t, { timeout: 30000 }).catch(() => {})
const shot = (name) => page.screenshot({ path: resolve(outDir, name) })

try {
  // 1) Verwaltung nav entry (admin) → builder list with presets + custom role
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await waitForText('Buchhaltung')
  await page.waitForTimeout(800)
  const navTxt = await page.evaluate(() =>
    Array.from(document.querySelectorAll('nav, aside')).map((n) => n.innerText).join(' '),
  )
  await page.goto(`${BASE}/#/admin/roles`, { waitUntil: 'domcontentloaded' })
  await waitForText('Rollen-Baukasten')
  await page.waitForTimeout(1000)
  const listTxt = await bodyText()
  await shot('01-builder-liste.png')
  out.push({
    step: 'liste: Verwaltung-Nav + 7 Presets + Custom „Lager & Logistik" + Abweichungs-Badge',
    navEntry: /Administration/.test(navTxt),
    presets: /Vollzugriff/.test(listTxt) && /IT-Admin/.test(listTxt) && /Aushilfe \/ Extern/.test(listTxt),
    customRole: /Lager & Logistik/.test(listTxt),
    basedOn: /basiert auf/.test(listTxt),
    deviations: /Abweichung/.test(listTxt),
    rawKeys: rawKeys(listTxt),
    pass:
      /Administration/.test(navTxt) && /Vollzugriff/.test(listTxt) && /Lager & Logistik/.test(listTxt) &&
      /basiert auf/.test(listTxt) && /Abweichung/.test(listTxt) && rawKeys(listTxt).length === 0,
  })

  // 2) Preset editor read-only (Vollzugriff): system banner, switches disabled
  await page.goto(`${BASE}/#/admin/roles/admin`, { waitUntil: 'domcontentloaded' })
  await waitForText('System-Rolle')
  await page.waitForTimeout(800)
  const presetTxt = await bodyText()
  const disabledSwitches = await page.locator('button[role="switch"][disabled]').count()
  await shot('02-editor-preset-readonly.png')
  out.push({
    step: 'editor preset: read-only Banner + disabled Switches + Zwei-Pane',
    banner: /System-Rolle — hier nur ansehen/.test(presetTxt),
    tree: /Standard/.test(presetTxt) && /Branchen/.test(presetTxt) && /Verwaltung/.test(presetTxt),
    switchesDisabled: disabledSwitches > 0,
    pass: /System-Rolle — hier nur ansehen/.test(presetTxt) && disabledSwitches > 0,
  })

  // 3) Custom editor draft: toggle a switch → staged bar appears → discard clears
  await page.goto(`${BASE}/#/admin/roles/role-c1`, { waitUntil: 'domcontentloaded' })
  await waitForText('Lager & Logistik')
  await page.waitForTimeout(800)
  // Focus the documents module in the tree (deviation badge 1: download)
  await page.locator('nav[aria-label="Module"]').getByRole('button', { name: /Dokumente/ }).click()
  await page.waitForTimeout(500)
  await page.getByRole('switch', { name: 'Hochladen' }).click()
  await page.waitForTimeout(500)
  const draftTxt = await bodyText()
  await shot('03-editor-draft.png')
  const hasStagedBar = /1 Änderung/.test(draftTxt)
  await page.getByRole('button', { name: 'Verwerfen' }).click()
  await page.waitForTimeout(400)
  const afterDiscard = await bodyText()
  out.push({
    step: 'editor custom: Draft-Leiste bei Toggle + Verwerfen setzt zurück',
    stagedBar: hasStagedBar,
    deviationDot: /Weicht von der Vorlage ab/.test(draftTxt) || /Abweichung/.test(draftTxt),
    discardClears: !/1 Änderung(?!en)/.test(afterDiscard.replace(/1 Änderungen/g, '')),
    pass: hasStagedBar && !/Änderungen übernehmen/.test(afterDiscard),
  })

  // 4) Apply: toggle again → Übernehmen → confirm dialog → live save toast
  await page.getByRole('switch', { name: 'Hochladen' }).click()
  await page.waitForTimeout(400)
  await page.getByRole('button', { name: 'Änderungen übernehmen' }).click()
  await page.waitForTimeout(500)
  const confirmTxt = await bodyText()
  await shot('04-apply-confirm.png')
  await page.getByRole('button', { name: 'Übernehmen', exact: true }).click()
  await waitForText('Rollen-Rechte gespeichert')
  const savedTxt = await bodyText()
  await shot('05-apply-saved.png')
  out.push({
    step: 'übernehmen: Confirm-Dialog (Konten-Hinweis) + Save-Toast',
    dialog: /Änderungen an .* übernehmen\?/.test(confirmTxt),
    membersHint: /Konten tragen diese Rolle/.test(confirmTxt) || /Konten/.test(confirmTxt),
    savedToast: /Rollen-Rechte gespeichert/.test(savedTxt),
    pass: /übernehmen\?/.test(confirmTxt) && /Rollen-Rechte gespeichert/.test(savedTxt),
  })

  // 5) Clone flow: list → Neue Rolle → create → lands in editor
  await page.goto(`${BASE}/#/admin/roles`, { waitUntil: 'domcontentloaded' })
  await waitForText('Rollen-Baukasten')
  await page.getByRole('button', { name: 'Neue Rolle' }).click()
  await page.waitForTimeout(500)
  await page.locator('#clone-name').fill('QA Testrolle')
  await shot('06-clone-dialog.png')
  await page.getByRole('button', { name: 'Rolle erstellen' }).click()
  await waitForText('QA Testrolle')
  await page.waitForTimeout(800)
  const clonedTxt = await bodyText()
  await shot('07-clone-editor.png')
  out.push({
    step: 'klonen: Dialog → neue Rolle → Editor mit basiert-auf-Badge',
    inEditor: /QA Testrolle/.test(clonedTxt),
    basedOnBadge: /basiert auf/.test(clonedTxt),
    pass: /QA Testrolle/.test(clonedTxt) && /basiert auf/.test(clonedTxt),
  })

  // 6) Compare modal: member vs extern, differences highlighted
  await page.goto(`${BASE}/#/admin/roles`, { waitUntil: 'domcontentloaded' })
  await waitForText('Rollen-Baukasten')
  await page.getByRole('button', { name: 'Vergleichen' }).first().click()
  await waitForText('Rollen vergleichen')
  await page.waitForTimeout(900)
  const compareTxt = await bodyText()
  await shot('08-vergleich.png')
  out.push({
    step: 'vergleich: 2 Rollen + Nur-Unterschiede + Diff-Zeilen mit Scope-Badges',
    open: /Rollen vergleichen/.test(compareTxt),
    diffCount: /Unterschied/.test(compareTxt),
    onlyDiff: /Nur Unterschiede/.test(compareTxt),
    pass: /Rollen vergleichen/.test(compareTxt) && /Unterschied/.test(compareTxt),
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(400)

  // 7) Overlay preview: extern editor → Als Rolle anzeigen → banner + shrunk nav → end
  await page.goto(`${BASE}/#/admin/roles/extern`, { waitUntil: 'domcontentloaded' })
  await waitForText('Aushilfe / Extern')
  await page.waitForTimeout(600)
  await page.getByRole('button', { name: 'Als Rolle anzeigen' }).click()
  await page.waitForTimeout(1200)
  const previewTxt = await bodyText()
  const navPreview = await page.evaluate(() =>
    Array.from(document.querySelectorAll('nav, aside')).map((n) => n.innerText).join(' '),
  )
  await shot('09-preview-banner.png')
  await page.getByRole('button', { name: 'Beenden' }).click()
  await page.waitForTimeout(1000)
  const navAfter = await page.evaluate(() =>
    Array.from(document.querySelectorAll('nav, aside')).map((n) => n.innerText).join(' '),
  )
  await shot('10-preview-ended.png')
  out.push({
    step: 'preview: Banner + Nav schrumpft auf Extern-Sicht + Beenden stellt her',
    banner: /Vorschau als/.test(previewTxt),
    navShrunk: !/Buchhaltung/.test(navPreview) && !/Kontakte/.test(navPreview),
    navRestored: /Buchhaltung/.test(navAfter),
    pass: /Vorschau als/.test(previewTxt) && !/Buchhaltung/.test(navPreview) && /Buchhaltung/.test(navAfter),
  })

  // 8) Team member roles section (Laura, multi-role) + effective-rights modal
  await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded' })
  await waitForText('Laura')
  await page.waitForTimeout(800)
  await page.getByText('Laura Neumann').first().click()
  await waitForText('Rollen verwalten')
  await page.waitForTimeout(600)
  const memberTxt = await bodyText()
  await shot('11-team-rollen-sektion.png')
  await page.getByRole('button', { name: 'Effektive Rechte ansehen' }).click()
  await waitForText('Effektive Rechte — ')
  await page.waitForTimeout(900)
  const effTxt = await bodyText()
  await shot('12-team-effektive-rechte.png')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)
  out.push({
    step: 'team: Rollen & Zugriff-Sektion (2 Chips + Verwalten) + Effektive-Rechte-Modal',
    section: /Rollen & Zugriff/i.test(memberTxt),
    bothChips: /Teamleiter/.test(memberTxt) && /HR-Admin/.test(memberTxt),
    manageBtn: /Rollen verwalten/.test(memberTxt),
    modal: /Effektive Rechte — /.test(effTxt) && /Gehaltsdaten/.test(effTxt),
    rawKeys: rawKeys(effTxt),
    pass:
      /Rollen & Zugriff/i.test(memberTxt) && /Teamleiter/.test(memberTxt) && /HR-Admin/.test(memberTxt) &&
      /Effektive Rechte — /.test(effTxt) && rawKeys(effTxt).length === 0,
  })
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)

  // 9) A-1 multi-role chips (Laura row shows both roles)
  await page.goto(`${BASE}/#/admin/users`, { waitUntil: 'domcontentloaded' })
  await waitForText('Laura Neumann')
  await page.waitForTimeout(600)
  const usersTxt = await bodyText()
  await shot('13-a1-multirole.png')
  await page.getByText('Laura Neumann').first().click()
  await waitForText('Rollen verwalten')
  await page.waitForTimeout(500)
  const userDetailTxt = await bodyText()
  await shot('14-a1-userdetail.png')
  await page.keyboard.press('Escape')
  await page.waitForTimeout(300)
  out.push({
    step: 'A-1: Rollen-Chips in Liste + UserDetail mit Verwalten + Effektive Rechte',
    listChips: /Teamleiter/.test(usersTxt) && /HR-Admin/.test(usersTxt),
    detail: /Rollen verwalten/.test(userDetailTxt) && /Effektive Rechte ansehen/.test(userDetailTxt),
    pass: /Teamleiter/.test(usersTxt) && /Rollen verwalten/.test(userDetailTxt),
  })

  // 10) Delete guardrail dialog (QA Testrolle, 0 members → confirm enabled; cancel)
  await page.goto(`${BASE}/#/admin/roles`, { waitUntil: 'domcontentloaded' })
  await waitForText('QA Testrolle')
  const qaCard = page.locator('div[role="button"]', { hasText: 'QA Testrolle' }).first()
  await qaCard.hover()
  await qaCard.getByRole('button', { name: 'Rollen-Aktionen' }).click()
  await page.waitForTimeout(400)
  await page.getByRole('menuitem', { name: 'Löschen' }).click()
  await waitForText('löschen?')
  const delTxt = await bodyText()
  await shot('15-delete-dialog.png')
  await page.getByRole('button', { name: 'Endgültig löschen' }).click()
  await page.waitForTimeout(800)
  const afterDel = await bodyText()
  out.push({
    step: 'löschen: Confirm-Dialog + Rolle verschwindet aus der Liste',
    dialog: /löschen\?/.test(delTxt),
    gone: !/QA Testrolle/.test(afterDel),
    pass: /löschen\?/.test(delTxt) && !/QA Testrolle/.test(afterDel),
  })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).split('\n')[0], pass: false })
  await page.screenshot({ path: resolve(outDir, '99-fatal.png') }).catch(() => {})
}

console.log(JSON.stringify({ steps: out, pageErrors: errs.slice(0, 5), allPass: out.every((s) => s.pass) }, null, 2))
await b.close()
process.exit(out.every((s) => s.pass) && errs.length === 0 ? 0 : 1)
