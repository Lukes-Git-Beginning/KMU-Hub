import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/db10-wiki-sweep')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(document|blocks|wiki|common)\.[a-z]+\.[a-z._]+/i

function findRawKeys(reSource) {
  const rx = new RegExp(reSource, 'i')
  return [
    ...new Set(
      Array.from(document.querySelectorAll('body *'))
        .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
        .map((n) => n.textContent.trim()),
    ),
  ].slice(0, 12)
}

const browser = await chromium.launch()
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1200 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

const articles = [
  ['wart-001', 'Willkommen im Cosmi-Wiki'],
  ['wart-004', 'Backup & Disaster Recovery'],
  ['wart-005', 'DSGVO'],
  ['wart-006', 'CRM'],
]
out.perArticle = {}
for (const [id, name] of articles) {
  await page.goto(`${BASE}/#/wiki?a=${id}`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2600)
  const raw = await page.evaluate(findRawKeys, RAW.source)
  out.perArticle[id] = { name, rawKeys: raw, errs: errs.length }
  await page.screenshot({ path: resolve(outDir, `${id}.png`), fullPage: true })
}

out.totalErrs = errs.length
out.errors = errs.slice(0, 6)
console.log(JSON.stringify(out, null, 2))
await browser.close()
