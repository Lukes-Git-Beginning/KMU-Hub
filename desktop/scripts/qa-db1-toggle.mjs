import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/db1-toggle')
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
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

// Open the welcome article directly via deep link (#/wiki?a=<id>).
await page.goto(`${BASE}/#/wiki?a=wart-001`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2800)

// Scroll to the FAQ toggles near the bottom of the article.
await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
await page.waitForTimeout(500)
const faq = page.getByText('Häufige Fragen').first()
await faq.scrollIntoViewIfNeeded().catch(() => {})
await page.waitForTimeout(400)
out.read = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, '01-read-toggles.png'), fullPage: false })

// Expand the collapsed toggle (the second FAQ item, default closed).
const collapsed = page.getByRole('button', { name: /Wie oft wird ein Artikel geprüft/ }).first()
await collapsed.click().catch(() => {})
await page.waitForTimeout(500)
await page.screenshot({ path: resolve(outDir, '02-read-expanded.png'), fullPage: false })

// Enter edit mode (header edit button, title = common.edit).
const editBtn = page.locator('button[title="Bearbeiten"]').first()
await editBtn.click().catch(() => {})
await page.waitForTimeout(1600)
await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
await page.waitForTimeout(400)
out.edit = { rawKeys: await page.evaluate(findRawKeys, RAW.source), errs: errs.length }
await page.screenshot({ path: resolve(outDir, '03-edit-toggle.png'), fullPage: false })

// Open the +Block insert menu and confirm the new "Aufklappen" entry is offered.
const addBlock = page.getByRole('button', { name: /Block einfügen/ }).first()
await addBlock.click().catch(() => {})
await page.waitForTimeout(600)
out.pickerHasToggle = await page.getByRole('button', { name: /^Aufklappen$/ }).count()
await page.screenshot({ path: resolve(outDir, '04-picker.png'), fullPage: false })

out.errors = errs.slice(0, 6)
console.log(JSON.stringify(out, null, 2))
await browser.close()
