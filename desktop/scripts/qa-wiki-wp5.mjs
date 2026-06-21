// QA — wiki WP-5: template CRUD + preview, empty-canvas joy moment (dev :5174)
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
      icuLeak: [...new Set(all.filter((t) => /\{count|plural,|\{name\}|\{min/.test(t)))].slice(0, 10),
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

  // ── A) Empty-canvas joy moment (empty draft article) ──
  await page.getByText('Reisekostenrichtlinie (Entwurf)', { exact: true }).first().click()
  await page.waitForTimeout(900)
  const emptyBody = await bodyText()
  out.emptyTitle = /Ein leeres Blatt/.test(emptyBody)
  out.emptyHasSvg = await page.locator('svg[role="img"]').count()
  await page.screenshot({ path: resolve(outDir, 'wp5-1-empty-canvas.png'), fullPage: false })

  // ── B) New-article dialog with template list + live preview ──
  await page.locator('button[title="Neuer Artikel"]').first().click()
  await page.waitForTimeout(800)
  out.dialogTemplates = await page.evaluate(() =>
    /Meeting-Protokoll/.test(document.body.textContent || '') &&
    /Projekt-Steckbrief/.test(document.body.textContent || ''),
  )
  // pick a template → preview renders
  await page.getByText('Post-Mortem', { exact: true }).first().click()
  await page.waitForTimeout(500)
  out.previewRendered = await page.evaluate(() =>
    /Post-Mortem:/.test(document.body.textContent || ''),
  )
  await page.screenshot({ path: resolve(outDir, 'wp5-2-template-dialog.png'), fullPage: false })

  // ── C) Open the template manager ──
  await page.getByText('Vorlagen verwalten', { exact: true }).first().click()
  await page.waitForTimeout(700)
  out.managerOpen = /Erstelle und bearbeite/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'wp5-3-manager-list.png'), fullPage: false })

  // ── D) Create a new template (form) ──
  await page.getByText('Neue Vorlage', { exact: true }).first().click()
  await page.waitForTimeout(600)
  out.formShown = /Name, Beschreibung/.test(await bodyText())
  await page.locator('input').first().fill('Sprint-Retrospektive')
  await page.waitForTimeout(200)
  await page.screenshot({ path: resolve(outDir, 'wp5-4-template-form.png'), fullPage: false })

  Object.assign(out, await scanRaw(page))
} catch (err) {
  out.error = String(err).split('\n')[0]
  await page.screenshot({ path: resolve(outDir, 'wp5-error.png'), fullPage: false }).catch(() => {})
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
