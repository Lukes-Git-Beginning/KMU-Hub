// QA — wiki WT-3: change notes + version diff/preview (dev :5174)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/wiki')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const scanRaw = (page) =>
  page.evaluate(() => {
    const all = Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0)
      .map((n) => (n.textContent || '').trim())
      .filter(Boolean)
    return {
      rawKeys: [...new Set(all.filter((t) => /^(wiki|common|shared)\.[a-zA-Z]/.test(t)))].slice(0, 15),
      doubleBrace: [...new Set(all.filter((t) => /\{\{|\}\}/.test(t)))].slice(0, 10),
      replacementChar: [...new Set(all.filter((t) => /�/.test(t)))].slice(0, 10),
    }
  })

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const bodyText = () => page.evaluate(() => document.body.textContent || '')

try {
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)

  // ── A) Open article + version history → change notes visible ──
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(1000)
  await page.getByRole('button', { name: 'Versionshistorie' }).first().click()
  await page.waitForTimeout(900)
  const bodyV = await bodyText()
  out.noteAbteilung = /Abteilungsstruktur ergänzt/.test(bodyV)
  out.noteEinleitung = /Einleitung neu formuliert/.test(bodyV)
  await page.screenshot({ path: resolve(outDir, 'wt3-1-versions-notes.png'), fullPage: false })

  // ── B) Compare an older version → diff with legend ──
  await page.getByRole('button', { name: 'Mit aktueller Version vergleichen' }).first().click()
  await page.waitForTimeout(700)
  const bodyD = await bodyText()
  out.diffLegendAdded = /Hinzugefügt/.test(bodyD)
  out.diffLegendRemoved = /Entfernt/.test(bodyD)
  out.diffShowsOldText = /Überarbeitet mit Abteilungsstruktur/.test(bodyD)
  await page.screenshot({ path: resolve(outDir, 'wt3-2-diff.png'), fullPage: false })
  Object.assign(out, await scanRaw(page))

  // ── C) Edit with a change note → new version carries the note ──
  await page.getByRole('button', { name: 'Versionshistorie' }).first().click() // close panel
  await page.waitForTimeout(400)
  await page.getByRole('button', { name: 'Bearbeiten' }).first().click()
  await page.waitForTimeout(900)
  const editor = page.locator('.ProseMirror, .tiptap-content').first()
  await editor.click()
  await page.keyboard.press('End')
  await page.keyboard.type(' Diese Zeile kam per QA hinzu.')
  await page.waitForTimeout(300)
  out.changeNoteField = await page.getByLabel('Was wurde geändert?').count()
  await page.getByLabel('Was wurde geändert?').fill('QA-Änderungsnotiz')
  await page.getByRole('button', { name: 'Speichern' }).first().click()
  await page.waitForTimeout(1200)
  out.savedToast = /Artikel gespeichert/.test(await bodyText())
  await page.getByRole('button', { name: 'Versionshistorie' }).first().click()
  await page.waitForTimeout(800)
  out.newNoteVisible = /QA-Änderungsnotiz/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'wt3-3-new-version-note.png'), fullPage: false })

  // ── D) Narrow ──
  await page.setViewportSize({ width: 820, height: 900 })
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, 'wt3-4-narrow.png'), fullPage: false })
  out.narrow = await scanRaw(page)
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'wt3-error.png'), fullPage: false }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
