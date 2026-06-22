import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/mails-ma9')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(mails|common|shared|moduleSettings)\.[a-z]+\.[a-z._]+/i
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

// Open the module settings overlay (bottom-left)
await page.getByText(/Modul-Einstellung/).first().click().catch(() => {})
await page.waitForTimeout(1000)
out.overlayOpen = await page.evaluate(() => /Mail-Einstellungen|Allgemein|Server & Compliance/.test(document.body.textContent || ''))
out.hasPersonal = await page.evaluate(() => /Standard-Konto|Signatur/.test(document.body.textContent || ''))
out.hasTenant = await page.evaluate(() => /IMAP|Automatische Antwort|Externe Bilder/.test(document.body.textContent || ''))
out.rawKeys = await page.evaluate(findRawKeys, RAW.source)
await page.screenshot({ path: resolve(outDir, '01-settings-panel.png'), fullPage: true })

out.errs = errs.length
out.errors = errs.slice(0, 6)
console.log(JSON.stringify(out, null, 2))
await browser.close()
