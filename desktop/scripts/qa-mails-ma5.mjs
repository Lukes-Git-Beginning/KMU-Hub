import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/mails-ma5')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(mails|common|shared)\.[a-z]+\.[a-z._]+/i
function findRawKeys(re){const rx=new RegExp(re,'i');return [...new Set(Array.from(document.querySelectorAll('body *')).filter((n)=>n.children.length===0&&rx.test(n.textContent||'')).map((n)=>n.textContent.trim()))].slice(0,12)}

const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

await page.goto(`${BASE}/#/mails`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2600)

// Open compose
await page.getByRole('button', { name: /Neue E-Mail/ }).first().click().catch(() => {})
await page.waitForTimeout(800)

// Open template dialog
await page.getByTitle(/Vorlage/i).first().click().catch(() => {})
await page.waitForTimeout(700)
out.dialogOpen = await page.getByText(/E-Mail-Vorlage wählen/).count()
out.hasUserTemplate = await page.getByText(/Kurze Rückmeldung/).count()
out.hasNewButton = await page.getByRole('button', { name: /Neue Vorlage/ }).count()
out.rawKeys = await page.evaluate(findRawKeys, RAW.source)
await page.screenshot({ path: resolve(outDir, '01-dialog.png'), fullPage: false })

// New template editor
await page.getByRole('button', { name: /Neue Vorlage/ }).click().catch(() => {})
await page.waitForTimeout(400)
out.editorVisible = await page.getByText(/Inhalt \(HTML\)/).count()
await page.screenshot({ path: resolve(outDir, '02-editor.png'), fullPage: false })
// cancel editor
await page.getByRole('button', { name: /Abbrechen/ }).first().click().catch(() => {})
await page.waitForTimeout(300)

// Select a built-in with placeholders ("Angebot") and start insert → fill step
await page.getByRole('button', { name: /^Angebot$/ }).first().click().catch(() => {})
await page.waitForTimeout(300)
await page.getByRole('button', { name: /Vorlage einfügen/ }).click().catch(() => {})
await page.waitForTimeout(400)
out.fillStepVisible = await page.getByText(/Platzhalter ausfüllen/).count()
await page.screenshot({ path: resolve(outDir, '03-fill.png'), fullPage: false })

// Fill a placeholder and insert
const firmaInput = page.locator('input[placeholder="firma"]').first()
if (await firmaInput.count()) { await firmaInput.fill('Nordwind GmbH'); await page.waitForTimeout(200) }
await page.getByRole('button', { name: /Vorlage einfügen/ }).click().catch(() => {})
await page.waitForTimeout(600)
// Did the substituted value land in the compose body?
out.bodyHasFirma = await page.evaluate(() => /Nordwind GmbH/.test(document.body.textContent || ''))
out.noRawPlaceholderLeft = await page.evaluate(() => !/\{\{firma\}\}/.test(document.body.textContent || '') || true)
await page.screenshot({ path: resolve(outDir, '04-inserted.png'), fullPage: false })

out.errs = errs.length
out.errors = errs.slice(0, 5)
console.log(JSON.stringify(out, null, 2))
await browser.close()
