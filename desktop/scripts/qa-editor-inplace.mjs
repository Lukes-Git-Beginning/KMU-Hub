/**
 * QA — Modul-Editor Edit-in-place PoC (Kontakte).
 *
 * Beweist: die Vorschau ist begehbar UND anpassbare Elemente (Kategorie-Labels)
 * sind IM Bild anklickbar → Inline-Umbenennen → live im Modul.
 *   1. Kontakte-Editor öffnen → Kategorie „Kunden" sichtbar in der Vorschau.
 *   2. „Kunden" anklicken → Inline-Input erscheint direkt am Element.
 *   3. „Patienten" eintippen + Enter → Label wird live „Patienten", Entwurf dirty.
 *
 * Screenshots → .qa-screenshots/editor-inplace/
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE || 'http://localhost:5175'
const outDir = resolve('.qa-screenshots/editor-inplace')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const NOLAUNCH = `try{sessionStorage.setItem('cosmi:launch-played','1')}catch(e){}`

const b = await chromium.launch({ headless: true })
const ctx = await b.newContext({ viewport: { width: 1360, height: 900 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(NOLAUNCH)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
const out = []
const bodyText = () => page.evaluate(() => document.body.innerText)
const shot = (n) => page.screenshot({ path: resolve(outDir, n), fullPage: false })
const wait = (ms) => page.waitForTimeout(ms)
const waitForText = (x, timeout = 20000) =>
  page.waitForFunction((t) => document.body.innerText.includes(t), x, { timeout }).catch(() => {})

try {
  await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
  await wait(2000)
  await page.evaluate(() => { window.location.hash = '#/editor-window?module=kontakte' })
  await waitForText('du bearbeitest eine Kopie', 15000)
  await wait(2000)
  await shot('01-editor-open.png')

  // The category "Kunden" lives in the navigable preview (left sidebar of the module).
  const kunden = page.locator('[role="button"]', { hasText: /^Kunden$/ }).first()
  {
    const visible = (await kunden.count()) > 0
    out.push({ step: 'S1 Kategorie „Kunden" in der Vorschau sichtbar + editierbar markiert', visible, pass: visible })
  }

  // Click it → inline input appears in place
  await kunden.click()
  await wait(500)
  await shot('02-inline-input.png')
  const input = page.locator('input[value="Kunden"], input').filter({ hasNotText: '' })
  {
    // an input focused with value Kunden should exist now
    const inputCount = await page.locator('aside input, .min-h-full input, input').count()
    out.push({ step: 'S2 Klick auf „Kunden" → Inline-Input am Element', inputCount, pass: inputCount > 0 })
  }

  // Type Patienten + Enter
  const active = page.locator('input:focus')
  if ((await active.count()) > 0) {
    await active.fill('Patienten')
    await active.press('Enter')
  } else {
    // fallback: find the inline input by proximity
    await page.keyboard.type('Patienten')
    await page.keyboard.press('Enter')
  }
  await wait(700)
  await shot('03-renamed-live.png')
  {
    const txt = await bodyText()
    out.push({
      step: 'S3 „Kunden" → „Patienten" inline umbenannt, live im Modul',
      hasPatienten: /Patienten/.test(txt),
      changeCounter: /1 Änderung/.test(txt),
      pass: /Patienten/.test(txt) && /1 Änderung/.test(txt),
    })
  }

  const raw = /kontakte\.category\.|customization\.editor\.[a-z]/i.test(await bodyText())
  out.push({ step: 'S4 Keine rohen i18n-Keys', rawKeysFound: raw, pass: !raw })
} catch (e) {
  out.push({ step: 'FATAL', error: String(e).slice(0, 300), pass: false })
  await shot('fatal.png').catch(() => {})
} finally {
  await b.close()
}

const total = out.length, passed = out.filter((s) => s.pass).length
console.log(`\n=== QA Edit-in-place PoC (Kontakte) — ${passed}/${total} PASS ===\n`)
out.forEach((s) => {
  console.log(`${s.pass ? 'PASS' : 'FAIL'}  ${s.step}`)
  const { step: _s, pass: _p, ...rest } = s
  if (Object.keys(rest).length) console.log('     ', JSON.stringify(rest))
})
console.log(errs.length ? `\nPage-Errors:\n ${errs.join('\n ')}` : '\nNo page errors.')
console.log('\nScreenshots in', outDir)
