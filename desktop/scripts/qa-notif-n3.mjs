/**
 * QA — notifications N-3: row click → shared DetailModal.
 * Verifies: clicking a row opens a centred modal with actor/avatar, module +
 * priority, full body and footer actions; "Öffnen" navigates to the deep link;
 * pin/mark-read work; keyboard (Enter) opens; system actor shows a system avatar.
 * Sub-terminal → :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/notif-n3')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW_RE = /(notifications\.[a-z]|\{\{)/

async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('[role="dialog"] *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim()).slice(0, 8)
  }, RAW_RE.source)
}
const clickCard = async (loc) => loc.evaluate((el) => el.click()).catch(() => loc.click({ timeout: 4000 }))

const browser = await chromium.launch()
const out = []
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
const page = await ctx.newPage(); const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error') errs.push('console: ' + m.text()) })

try {
  await page.goto(`${BASE}/#/notifications`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)

  // 1) open first card → modal
  await clickCard(page.locator('.space-y-2 .cursor-pointer').first())
  await page.waitForTimeout(900)
  const dialog = page.locator('[role="dialog"]')
  const dialogOpen = await dialog.count()
  const footerBtns = {
    open: await page.locator('[role="dialog"] button:has-text("Öffnen")').count(),
    markRead: await page.locator('[role="dialog"] button:has-text("Als gelesen markieren")').count(),
    pin: await page.locator('[role="dialog"] button:has-text("Anpinnen")').count(),
    dismiss: await page.locator('[role="dialog"] button:has-text("Ignorieren")').count(),
  }
  const actorVisible = await page.locator('[role="dialog"]:has-text("Thomas Meier")').count()
  const metaVon = await page.locator('[role="dialog"]:has-text("Von")').count()
  await page.screenshot({ path: resolve(outDir, '0-modal-deal.png') })
  out.push({ check: 'modal-open', dialogOpen, footerBtns, actorVisible, metaVon, rawKeys: await rawKeys(page) })

  // 2) "Öffnen" navigates to deep link (/pipeline for the deal)
  await page.locator('[role="dialog"] button:has-text("Öffnen")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(1000)
  const urlAfterOpen = page.url()
  await page.screenshot({ path: resolve(outDir, '1-after-open.png') })
  out.push({ check: 'deep-link-nav', urlAfterOpen, navigated: /pipeline/.test(urlAfterOpen) })

  // back to center
  await page.goto(`${BASE}/#/notifications`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2000)

  // 3) system actor + urgent pill: open the security alert
  await clickCard(page.locator('.cursor-pointer:has-text("Sicherheitswarnung")').first())
  await page.waitForTimeout(800)
  const urgentPill = await page.locator('[role="dialog"]:has-text("Dringend")').count()
  const secBadge = await page.locator('[role="dialog"]:has-text("Sicherheit")').count()
  const systemActor = await page.locator('[role="dialog"]:has-text("System")').count()
  await page.screenshot({ path: resolve(outDir, '2-modal-system.png') })
  out.push({ check: 'system-modal', urgentPill, secBadge, systemActor })

  // 4) pin inside modal → button flips to "Lösen"
  await page.locator('[role="dialog"] button:has-text("Anpinnen")').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(600)
  const becameUnpin = await page.locator('[role="dialog"] button:has-text("Lösen")').count()
  await page.screenshot({ path: resolve(outDir, '3-modal-pinned.png') })
  out.push({ check: 'modal-pin', becameUnpin })

  // close (Escape)
  await page.keyboard.press('Escape')
  await page.waitForTimeout(600)
  const closedAfterEsc = await page.locator('[role="dialog"]').count()

  // 5) keyboard open: focus first card, press Enter
  await page.locator('.space-y-2 .cursor-pointer').first().focus().catch(() => {})
  await page.keyboard.press('Enter')
  await page.waitForTimeout(700)
  const keyboardOpened = await page.locator('[role="dialog"]').count()
  out.push({ check: 'close+keyboard', closedAfterEsc, keyboardOpened })

  out.push({ pageErrors: errs.slice(0, 5) })
} catch (e) {
  out.push({ error: String(e).split('\n')[0], pageErrors: errs.slice(0, 5) })
} finally {
  await ctx.close(); await browser.close()
}

console.log(JSON.stringify(out, null, 2))
