import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/db2-code')
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
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 }, deviceScaleFactor: 2 })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

// Backup article carries a bash code block.
await page.goto(`${BASE}/#/wiki?a=wart-004`, { waitUntil: 'domcontentloaded' })
// Wait for the actual article body (not the skeleton) to settle.
await page.getByRole('heading', { name: /Backup & Disaster Recovery/ }).first().waitFor({ timeout: 15000 }).catch(() => {})
await page.locator('code.hljs').first().waitFor({ timeout: 15000 }).catch(() => {})
await page.waitForTimeout(600)
await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
await page.waitForTimeout(500)

// Count highlighted tokens to prove lowlight actually ran.
out.hljsSpans = await page.locator('code.hljs span[class^="hljs-"]').count()
out.codeText = (await page.locator('code.hljs').first().textContent().catch(() => '') || '').slice(0, 40)
out.read = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, '01-read-code.png'), clip: { x: 760, y: 60, width: 680, height: 940 } })

// Edit mode → language dropdown + monospace editor.
const editBtn = page.locator('button[title="Bearbeiten"]').first()
await editBtn.click().catch(() => {})
await page.waitForTimeout(1600)
await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
await page.waitForTimeout(500)
out.langSelectCount = await page.locator('select').count()
out.edit = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, '02-edit-code.png'), clip: { x: 700, y: 60, width: 740, height: 940 } })

out.errors = errs.slice(0, 6)
console.log(JSON.stringify(out, null, 2))
await browser.close()
