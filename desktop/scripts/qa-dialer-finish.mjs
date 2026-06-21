// QA — dialer finish pass on merged main (dev :5173): surfaces + contact modal + scan
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/dialer-finish')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1500, height: 940 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
page.setDefaultTimeout(9000)
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const bodyText = () => page.evaluate(() => document.body.textContent || '')
const rawKeysOf = () =>
  page.evaluate(() => {
    const all = Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0)
      .map((n) => (n.textContent || '').trim())
    return [...new Set(all.filter((t) => /^(dialer|common|shared|moduleSettings)\.[a-zA-Z]/.test(t)))].slice(0, 10)
  })
const step = async (name, fn) => {
  try { await fn() } catch (err) { out[`ERR_${name}`] = String(err).split('\n')[0] }
}

// 1) Every nav surface renders cleanly on merged main.
const surfaces = [
  ['campaigns', '/#/dialer/campaigns'],
  ['detail', '/#/dialer/campaigns/dlr-camp-001'],
  ['dashboard', '/#/dialer/dashboard'],
  ['supervisor', '/#/dialer/supervisor'],
  ['settings', '/#/dialer/settings'],
]
const allRaw = []
for (const [name, url] of surfaces) {
  await page.goto(`${BASE}${url}`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2600)
  const rk = await rawKeysOf()
  allRaw.push(...rk)
  out[`${name}_nan`] = /NaN/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, `finish-${name}.png`), fullPage: false })
}
out.rawKeysAcrossSurfaces = [...new Set(allRaw)]

// 2) D-3: a contact queue row opens a detail modal (projektweiter Standard).
await step('contactModal', async () => {
  await page.goto(`${BASE}/#/dialer/campaigns/dlr-camp-001`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2600)
  const row = page.locator('tbody tr, [role="button"]').filter({ hasText: /\+?\d|@|GmbH|AG|Frau|Herr/ }).first()
  if (await row.count()) {
    await row.click()
    await page.waitForTimeout(900)
  }
  const body = await bodyText()
  out.modalOpened = await page.locator('[role="dialog"], .fixed').filter({ hasText: /Anruf|Verlauf|Historie|Kontakt|Telefon/ }).count()
  out.modalHasInfo = /Anruf|Verlauf|Historie|Telefon|CRM/i.test(body)
  await page.screenshot({ path: resolve(outDir, 'finish-contact-modal.png'), fullPage: false })
})

// 3) Cross-check: wiki still renders (both modules now share main).
await step('wikiSanity', async () => {
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2600)
  out.wikiRenders = /Willkommen im Cosmi-Wiki/.test(await bodyText())
})

out.pageErrors = errs.slice(0, 10)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
