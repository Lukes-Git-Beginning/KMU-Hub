import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/profil-presence')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const rawRe = /(profil\.presence\.|profil\.avatar\.)/
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 10)
  }, rawRe.source)
}

async function presenceDotClass(page) {
  return page.evaluate(() => {
    const avatarBox = document.querySelector('span[class*="ring-2 ring-card"]')
    return avatarBox ? avatarBox.className : null
  })
}

const browser = await chromium.launch()
const out = []

{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/profil`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2000)

    const dotBefore = await presenceDotClass(page)
    const pickerVisible = await page.locator('button[aria-label="Status wählen"]').first().isVisible().catch(() => false)
    const fileInput = await page.evaluate(() => !!document.querySelector('input[type="file"][accept="image/*"]'))
    await page.screenshot({ path: resolve(outDir, 'profil-header.png') })
    out.push({ step: 'header', pickerVisible, fileInputPresent: fileInput, dotBefore, rawKeys: await rawKeys(page), errs: errs.length })

    // Pick "Beschäftigt" → dot turns red, persists across reload
    await page.locator('button[aria-label="Status wählen"]').first().click({ timeout: 6000 })
    await page.waitForTimeout(400)
    await page.screenshot({ path: resolve(outDir, 'picker-open.png') })
    await page.locator('[role="menuitem"]:has-text("Beschäftigt")').first().click({ timeout: 6000 })
    await page.waitForTimeout(400)
    const dotAfter = await presenceDotClass(page)
    await page.screenshot({ path: resolve(outDir, 'status-dnd.png') })

    await page.reload()
    await page.waitForTimeout(2000)
    const dotReload = await presenceDotClass(page)
    const labelReload = await page.locator('button[aria-label="Status wählen"]:has-text("Beschäftigt")').first().isVisible().catch(() => false)
    out.push({
      step: 'pick-persist',
      dndAfterPick: (dotAfter || '').includes('bg-red-500'),
      dndAfterReload: (dotReload || '').includes('bg-red-500'),
      labelPersisted: labelReload,
      errs: errs.length,
    })
  } catch (e) {
    out.push({ step: 'main', error: String(e).split('\n')[0] })
  } finally { await ctx.close() }
}

// Narrow viewport
{
  const ctx = await browser.newContext({ viewport: { width: 760, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/profil`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2000)
    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, 'profil-760.png') })
    out.push({ step: 'narrow-760', rawKeys: rk, errs: errs.length })
  } catch (e) {
    out.push({ step: 'narrow-760', error: String(e).split('\n')[0] })
  } finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
