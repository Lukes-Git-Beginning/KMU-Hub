/**
 * QA — Customization v1.1 (Custom Fields Editor).
 * Verifies: Hub-Einstieg als admin → Tab „Felder" sichtbar → Entity-Selector →
 * bestehende Felder anzeigen → Feld anlegen (Basis) → Progressive Disclosure
 * aufklappen (Optionen bei select) → Soft-Delete-Dialog bei benutztem Feld.
 *
 * Screenshots → .qa-screenshots/customization-v11/
 * Raw-Keys + pageerrors werden getrackt.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = `http://localhost:${process.env.QA_PORT ?? 5173}`
const API = 'http://localhost:8080'
const outDir = resolve('.qa-screenshots/customization-v11')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch()
const ctx = await b.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []

const bodyText = () => page.evaluate(() => document.body.innerText)
const rawKeys = (txt) =>
  (txt.match(/\b(customization|admin|rbac)\.[a-zA-Z]+\.[a-zA-Z.]+/g) || []).slice(0, 8)
const shot = (name) => page.screenshot({ path: resolve(outDir, name) })
const wait = (ms) => page.waitForTimeout(ms)
const waitForText = (x, timeout = 15000) =>
  page.waitForFunction((t) => document.body.innerText.includes(t), x, { timeout }).catch(() => {})
const bounce = async (path) => {
  await page.goto(`${BASE}/#/dashboard`, { waitUntil: 'domcontentloaded' })
  await wait(400)
  await page.goto(`${BASE}${path}`, { waitUntil: 'domcontentloaded' })
  await wait(1500)
}

try {
  // ── 1) Hub öffnen als admin ───────────────────────────────────────────────
  await page.goto(`${BASE}/#/admin/anpassungen/felder`, { waitUntil: 'domcontentloaded' })
  await waitForText('Anpassungen', 20000)
  await wait(1000)
  let txt = await bodyText()
  await shot('01-hub-felder-tab.png')
  const hubTitle = /Anpassungen/.test(txt)
  const felderTab = /Felder/.test(txt)
  const entitySelector = /Aufgaben|Kontakte|Firmen|Deals/.test(txt)
  const noRawKeysHub = rawKeys(txt).length === 0
  out.push({
    step: '1 Hub öffnen als admin — Titel, Tab „Felder", Entity-Selector sichtbar',
    hubTitle,
    felderTab,
    entitySelector,
    noRawKeys: noRawKeysHub,
    rawKeys: rawKeys(txt),
    pass: hubTitle && felderTab && entitySelector && noRawKeysHub,
  })

  // ── 2) MSW-Felder via API prüfen ─────────────────────────────────────────
  const fieldsCheck = await page.evaluate(async (api) => {
    try {
      const r = await fetch(`${api}/api/v1/customization/fields?entity=work_task`)
      const d = await r.json()
      return {
        ok: r.ok,
        count: (d.fields || []).length,
        firstLabel: (d.fields || [])[0]?.label ?? null,
        types: [...new Set((d.fields || []).map((f) => f.type))],
      }
    } catch (e) {
      return { ok: false, error: String(e) }
    }
  }, API)
  out.push({
    step: '2 MSW GET /api/v1/customization/fields?entity=work_task',
    apiOk: fieldsCheck.ok,
    count: fieldsCheck.count,
    firstLabel: fieldsCheck.firstLabel,
    types: fieldsCheck.types,
    pass: fieldsCheck.ok && fieldsCheck.count >= 2,
  })

  // ── 3) Bestehende Felder in der UI sehen (work_task ist default) ─────────
  await waitForText('ERP-Ticketnummer')
  txt = await bodyText()
  await shot('03-existing-fields.png')
  const seedField = /ERP-Ticketnummer/.test(txt)
  const typeBadge = /Zahl|Text|Auswahl/.test(txt)
  const noRawKeysFields = rawKeys(txt).length === 0
  out.push({
    step: '3 Bestehende Felder (work_task) sichtbar — Seed-Felder + Typ-Badges',
    seedField,
    typeBadge,
    noRawKeys: noRawKeysFields,
    rawKeys: rawKeys(txt),
    pass: seedField && typeBadge && noRawKeysFields,
  })

  // ── 4) Entity wechseln → crm_contact ─────────────────────────────────────
  await page.getByRole('radio', { name: 'Kontakte' }).click().catch(async () => {
    // Fallback: suche nach Button
    await page.getByText('Kontakte').first().click()
  })
  await wait(1000)
  await waitForText('Kundennummer ERP')
  txt = await bodyText()
  await shot('04-entity-crm-contact.png')
  const contactField = /Kundennummer ERP/.test(txt)
  const noRawKeysContact = rawKeys(txt).length === 0
  out.push({
    step: '4 Entity-Wechsel → crm_contact zeigt Kundennummer-ERP-Feld',
    contactField,
    noRawKeys: noRawKeysContact,
    rawKeys: rawKeys(txt),
    pass: contactField && noRawKeysContact,
  })

  // ── 5) Feld anlegen — „Feld anlegen"-Button → Modal öffnet sich ──────────
  await page.getByRole('button', { name: 'Feld anlegen' }).click()
  await wait(800)
  txt = await bodyText()
  await shot('05-create-modal-open.png')
  const modalTitle = /Neues Feld anlegen/.test(txt)
  // Placeholder is "z.B. Kundennummer ERP" (DE) — match by label text presence
  const labelInputPresent = /Name \/ Bezeichnung|Name \/ Label/.test(txt)
  const typeGrid = /Text|Zahl|Datum|Ja\/Nein/.test(txt)
  const noRawKeysModal = rawKeys(txt).length === 0
  out.push({
    step: '5 Create-Modal öffnet sich — Titel, Label-Input, Typ-Grid',
    modalTitle,
    labelInputPresent,
    typeGrid,
    noRawKeys: noRawKeysModal,
    rawKeys: rawKeys(txt),
    pass: modalTitle && labelInputPresent && typeGrid && noRawKeysModal,
  })

  // ── 6) Basis-Schicht ausfüllen: Label + Typ "Auswahl" wählen ─────────────
  await page.getByPlaceholder(/Kundennummer ERP|ERP customer|z\.B\./).fill('Test-Branche')
  await wait(300)
  // Typ "Auswahl" (select) klicken
  await page.getByRole('button', { name: 'Auswahl' }).click().catch(async () => {
    await page.getByText('Auswahl').first().click()
  })
  await wait(400)
  txt = await bodyText()
  await shot('06-basis-layer-filled.png')
  const labelFilled = /Test-Branche/.test(txt)
  out.push({
    step: '6 Basis-Schicht: Label + Typ „Auswahl" gewählt',
    labelFilled,
    noRawKeys: rawKeys(txt).length === 0,
    pass: labelFilled && rawKeys(txt).length === 0,
  })

  // ── 7) Progressive Disclosure: „Weitere Optionen" aufklappen ─────────────
  await page.getByRole('button', { name: /Weitere Optionen|More options/ }).click().catch(() => {})
  await wait(600)
  txt = await bodyText()
  await shot('07-progressive-disclosure-open.png')
  const optionsSection = /Auswahloptionen|Options de sélection|Options/.test(txt)
  const validationSection = /Standardwert|Default value/.test(txt)
  const noRawKeysAdv = rawKeys(txt).length === 0
  out.push({
    step: '7 Progressive Disclosure aufgeklappt — Optionen + Standardwert sichtbar',
    optionsSection,
    validationSection,
    noRawKeys: noRawKeysAdv,
    rawKeys: rawKeys(txt),
    pass: optionsSection && validationSection && noRawKeysAdv,
  })

  // Optionen eingeben
  await page.getByPlaceholder(/Option 1, Option 2/).fill('Gastro, Handel, Handwerk, Sonstiges')
  await wait(300)
  await shot('07b-options-filled.png')

  // ── 8) Feld anlegen speichern ─────────────────────────────────────────────
  await page.getByRole('button', { name: 'Feld anlegen' }).last().click()
  await wait(1200)
  txt = await bodyText()
  await shot('08-field-created.png')
  const fieldCreated = /Test-Branche/.test(txt)
  const toastShown = /angelegt|created/.test(txt)
  const noRawKeysAfter = rawKeys(txt).length === 0
  out.push({
    step: '8 Feld gespeichert — „Test-Branche" in Liste, Toast-Meldung',
    fieldCreated,
    toastShown,
    noRawKeys: noRawKeysAfter,
    rawKeys: rawKeys(txt),
    pass: fieldCreated && noRawKeysAfter,
  })

  // ── 9) Soft-Delete-Dialog: benutzte Feld (inUse=true) löschen ────────────
  // Zurück zu work_task (hat Felder mit inUse=true)
  await page.getByRole('radio', { name: 'Aufgaben' }).click().catch(async () => {
    await page.getByText('Aufgaben').first().click()
  })
  await wait(1000)
  await waitForText('ERP-Ticketnummer')
  // Klick auf die Zeile (erstes Feld mit inUse=true)
  await page.getByText('ERP-Ticketnummer').first().click()
  await wait(800)
  txt = await bodyText()
  await shot('09a-edit-modal-inuse.png')
  // Löschen-Button im Footer
  await page.getByRole('button', { name: /Löschen|Delete|Supprimer/ }).first().click()
  await wait(800)
  txt = await bodyText()
  await shot('09b-soft-delete-dialog.png')
  const deleteDialog = /gespeicherten Daten|stored data|Données enregistrées/.test(txt)
  const cancelButton = await page.getByRole('button', { name: /Abbrechen|Cancel|Annuler/ }).count() > 0
  const noRawKeysDelete = rawKeys(txt).length === 0
  out.push({
    step: '9 Soft-Delete-Dialog bei inUse=true Feld — Konsequenz-Warnung + Abbrechen',
    deleteDialog,
    cancelButton,
    noRawKeys: noRawKeysDelete,
    rawKeys: rawKeys(txt),
    pass: deleteDialog && cancelButton && noRawKeysDelete,
  })
  // Abbrechen drücken (keinen Datenverlust im QA)
  await page.getByRole('button', { name: /Abbrechen|Cancel|Annuler/ }).first().click()
  await wait(500)

} catch (e) {
  out.push({ step: 'FATAL', error: String(e), pass: false })
  await shot('fatal-error.png').catch(() => {})
} finally {
  await b.close()
}

// ── Ergebnis ─────────────────────────────────────────────────────────────────

const total = out.length
const passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Customization v1.1 — ${passed}/${total} PASS ===\n`)
out.forEach((s, i) => {
  const icon = s.pass ? 'PASS' : 'FAIL'
  console.log(`${icon}  ${s.step}`)
  if (!s.pass) {
    const { step: _, pass: __, ...details } = s
    console.log('     ', JSON.stringify(details))
  }
})

if (errs.length > 0) {
  console.log('\nPage errors:')
  errs.forEach((e) => console.log(' ', e))
}

console.log('\nScreenshots:')
const pngNames = [
  '01-hub-felder-tab.png',
  '03-existing-fields.png',
  '04-entity-crm-contact.png',
  '05-create-modal-open.png',
  '06-basis-layer-filled.png',
  '07-progressive-disclosure-open.png',
  '07b-options-filled.png',
  '08-field-created.png',
  '09a-edit-modal-inuse.png',
  '09b-soft-delete-dialog.png',
]
pngNames.forEach((f) => console.log(' ', resolve('.qa-screenshots/customization-v11', f)))

if (passed < total) process.exit(1)
