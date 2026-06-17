/**
 * QA — helpdesk H-8 i18n + KB stateful save against :5174.
 *  A) Switch language to EN → SLA texts are English (no German hardcodes), no raw keys.
 *  B) KB body edit is stateful: edit → save → reload → content persists.
 * Screenshots: .qa-screenshots/helpdesk-i18n/
 *
 * Usage: node scripts/qa-helpdesk-i18n.mjs
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots', 'helpdesk-i18n')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const WIPE = `try{if(!sessionStorage.getItem('qa-hd-wiped')){localStorage.removeItem('cosmi-helpdesk');sessionStorage.setItem('qa-hd-wiped','1')}}catch(e){}`
const EN = `try{localStorage.setItem('cosmi-locale',JSON.stringify({state:{locale:'en'},version:0}))}catch(e){}`

const scan = (page) => page.evaluate(() => {
  const all = Array.from(document.querySelectorAll('body *')).filter((n) => n.children.length === 0)
    .map((n) => (n.textContent || '').trim()).filter(Boolean)
  return {
    rawKeys: all.filter((t) => /^(helpdesk|common|shared|moduleSettings|settings)\.[a-zA-Z]/.test(t)).slice(0, 12),
    doubleBrace: all.filter((t) => /\{\{|\}\}/.test(t)).slice(0, 12),
  }
})
const slaTexts = (page) => page.evaluate(() =>
  Array.from(document.querySelectorAll('tbody tr')).map((tr) => (tr.querySelectorAll('td')[6]?.textContent || '').trim()))

const out = []
const browser = await chromium.launch()

// ── A) English SLA texts + no raw keys ────────────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1480, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(WIPE); await ctx.addInitScript(EN)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(3500)
    const sla = await slaTexts(page)
    // open a ticket too, to scan the modal SLA timer + actions
    const row = page.locator('tbody tr').first()
    await row.evaluate((el) => el.click()).catch(async () => { await row.click() })
    await page.waitForTimeout(900)
    const modalText = await page.evaluate(() => document.querySelector('[role="dialog"]')?.textContent || '')
    await page.screenshot({ path: resolve(outDir, '0-english.png'), fullPage: true })
    out.push({
      step: 'english',
      slaEnglish: sla.filter((s) => /left|overdue/.test(s)).length,
      slaGermanLeftovers: sla.filter((s) => /übrig|überfällig/.test(s)),
      modalEnglishActions: /Escalate|Merge|Assign|Change status/i.test(modalText),
      ...(await scan(page)),
      pageErrors: errs.slice(0, 4),
    })
  } catch (e) { out.push({ step: 'english', error: String(e).split('\n')[0], pageErrors: errs.slice(0, 4) }) }
  finally { await ctx.close() }
}

// ── B) KB body stateful save survives reload ──────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1480, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB); await ctx.addInitScript(WIPE)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  const MARK = 'QA-KB-EDIT-PERSIST-12345'
  try {
    await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(3500)
    await page.getByText('Wissensdatenbank', { exact: false }).first().click()
    await page.waitForTimeout(700)
    await page.locator('button').filter({ hasText: /VPN-Verbindung einrichten/ }).first().click()
    await page.waitForTimeout(700)
    await page.getByRole('button', { name: 'Bearbeiten' }).click()
    await page.waitForTimeout(500)
    const editor = page.locator('[contenteditable="true"]').first()
    await editor.click()
    await editor.type(' ' + MARK)
    await page.waitForTimeout(300)
    await page.getByRole('button', { name: 'Speichern' }).click()
    await page.waitForTimeout(700)
    const afterSave = await page.getByText(MARK).count()
    await page.screenshot({ path: resolve(outDir, '1-kb-saved.png') })

    // reload → reopen the same article → mark persists
    await page.reload({ waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(3000)
    await page.getByText('Wissensdatenbank', { exact: false }).first().click()
    await page.waitForTimeout(700)
    await page.locator('button').filter({ hasText: /VPN-Verbindung einrichten/ }).first().click()
    await page.waitForTimeout(700)
    const afterReload = await page.getByText(MARK).count()
    await page.screenshot({ path: resolve(outDir, '2-kb-reloaded.png') })
    out.push({ step: 'kb-save', afterSave, afterReload, pageErrors: errs.slice(0, 4) })
  } catch (e) { out.push({ step: 'kb-save', error: String(e).split('\n')[0], pageErrors: errs.slice(0, 4) }) }
  finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
