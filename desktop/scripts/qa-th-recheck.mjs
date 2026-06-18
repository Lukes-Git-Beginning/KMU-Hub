import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/team-helpdesk-fixes')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const browser = await chromium.launch()
const out = []
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []; page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))

try {
  // F1: current user name in sidebar (bottom-left) + topbar (top-right)
  await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2200)
  const names = await page.evaluate(() => {
    const aside = document.querySelector('aside')
    const sidebarTxt = aside ? aside.textContent : ''
    const sidebarUser = /Stefan Vogel/.test(sidebarTxt) ? 'Stefan Vogel' : (/Markus Weber/.test(sidebarTxt) ? 'Markus Weber' : 'other')
    // topbar = first header/topbar region
    const header = document.querySelector('header') || document.body
    const topTxt = header.textContent || ''
    const topUser = /Stefan Vogel/.test(topTxt) ? 'Stefan Vogel' : (/Markus Weber/.test(topTxt) ? 'Markus Weber' : 'other')
    return { sidebarUser, topUser }
  })
  await page.screenshot({ path: resolve(outDir, 'recheck-01-team-sidebar.png') })
  out.push({ step: 'F1-names', ...names, errs: errs.length })

  // F6: helpdesk ticket → open canned responses popover
  await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2200)
  await page.locator('table tbody tr').first().click({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(900)
  // click the "Vorlagen" trigger (real click for Radix)
  const trigger = page.getByRole('button', { name: /Vorlagen/i }).first()
  await trigger.click({ timeout: 5000 }).catch(async () => {
    await page.getByText('Vorlagen', { exact: false }).first().click({ timeout: 5000 }).catch(() => {})
  })
  await page.waitForTimeout(700)
  await page.screenshot({ path: resolve(outDir, 'recheck-02-canned-popover.png') })
  const pop = await page.evaluate(() => {
    // canned response titles seen in screenshot: Begrüßung, VPN, Passwort-Reset…
    const items = Array.from(document.querySelectorAll('*')).filter((n) => n.children.length === 0 && /Begrüßung|VPN|Passwort-Reset|Ticket geschlossen/.test(n.textContent || ''))
    if (!items.length) return { open: false }
    const rects = items.map((n) => n.getBoundingClientRect())
    const allInView = rects.every((r) => r.right <= window.innerWidth + 1 && r.left >= -1 && r.bottom <= window.innerHeight + 1 && r.top >= -1)
    return { open: true, count: items.length, allInView }
  })
  out.push({ step: 'F6-popover', ...pop, errs: errs.length })

  // F8: hover SLA header → tooltip
  await page.keyboard.press('Escape'); await page.waitForTimeout(400)
  const sla = page.getByText('SLA', { exact: true }).first()
  await sla.hover({ timeout: 5000 }).catch(() => {})
  await page.waitForTimeout(600)
  await page.screenshot({ path: resolve(outDir, 'recheck-03-sla-tooltip.png') })
  const tip = await page.evaluate(() => /Service Level Agreement/.test(document.body.textContent || ''))
  out.push({ step: 'F8-tooltip', tooltipShown: tip, errs: errs.length })
} catch (e) { out.push({ step: 'recheck', error: String(e).split('\n')[0] }) } finally { await ctx.close() }
await browser.close()
console.log(JSON.stringify(out, null, 2))
