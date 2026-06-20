import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/r0-berichte')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(berichte|common)\.[a-z]+\.[a-z._]+/i

function findRawKeys(re) {
  const rx = new RegExp(re, 'i')
  return [
    ...new Set(
      Array.from(document.querySelectorAll('body *'))
        .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
        .map((n) => n.textContent.trim()),
    ),
  ].slice(0, 12)
}

const browser = await chromium.launch()
const out = []
for (const w of [1440, 820]) {
  const ctx = await browser.newContext({ viewport: { width: w, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  const result = { w }
  try {
    await page.goto(`${BASE}/#/berichte`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2000)
    // Switch to the "Berichte" tab (library)
    const tab = page.getByRole('button', { name: /^Berichte$/ }).first()
    if (await tab.count()) {
      await tab.click().catch(() => {})
      await page.waitForTimeout(1200)
    }
    result.library = {
      rawKeys: await page.evaluate(findRawKeys, RAW.source),
      errs: errs.length,
    }
    await page.screenshot({ path: resolve(outDir, `library-${w}.png`) })

    // Open a seeded document -> editor shell
    const card = page
      .locator('[role="button"]')
      .filter({ hasText: /Verkaufsbericht|Monatsbericht|Helpdesk-Auslastung/ })
      .first()
    if (await card.count()) {
      await card.click().catch(() => {})
      await page.waitForTimeout(1500)
      result.editor = {
        rawKeys: await page.evaluate(findRawKeys, RAW.source),
        errs: errs.length,
      }
      await page.screenshot({ path: resolve(outDir, `editor-${w}.png`) })
    } else {
      result.editor = { error: 'no card found' }
    }
  } catch (e) {
    result.error = String(e).split('\n')[0]
  } finally {
    await ctx.close()
  }
  out.push(result)
}
await browser.close()
console.log(JSON.stringify(out, null, 2))
