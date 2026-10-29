import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

// Invite dialog against the role shapes: the member preset is preselected by
// key (server role ids are UUIDs), its description shows, a duplicate address
// gets the "account or pending invitation" message, a fresh one the sent toast.
const BASE = process.env.QA_BASE || 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/admin-invite-honest')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW = /\b(admin|rbac|common)\.[a-z]+\.[a-z._]+/i

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
const out = {}
const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
page.setDefaultTimeout(8000)
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
await page.goto(`${BASE}/#/admin/users`, { waitUntil: 'domcontentloaded' })
await page.locator('.cosmi-startup').waitFor({ state: 'detached', timeout: 20000 }).catch(() => {})
await page.waitForTimeout(800)

await page.getByRole('button', { name: /einladen/i }).first().click()
await page.waitForTimeout(600)
const dialog = page.getByRole('dialog')
out.dialogOpen = await dialog.count()
out.checkedRoles = await dialog.locator('[role="checkbox"][aria-checked="true"]').allInnerTexts()
out.memberHint = await dialog.getByText(/Mitarbeiter|Standard/).count()
out.rawKeys = await page.evaluate(findRawKeys, RAW.source)
await page.screenshot({ path: resolve(outDir, '1-open.png') })

// duplicate: an address the demo directory already has
const existing = await page.evaluate(() => {
  const m = document.body.innerText.match(/[\w.+-]+@[\w-]+\.[\w.]+/)
  return m ? m[0] : ''
})
out.duplicateAddress = existing
await dialog.getByRole('textbox').first().fill(existing || 'stefan.vogel@zentria.tech')
await dialog.getByRole('button', { name: /Einladung senden/ }).click()
await page.waitForTimeout(900)
out.duplicateToast = await page.getByText(/schon ein Konto oder eine offene Einladung/).count()
await page.screenshot({ path: resolve(outDir, '2-duplicate.png') })

// fresh address
await dialog.getByRole('textbox').first().fill('neu.kollegin@firma.de')
await dialog.getByRole('button', { name: /Einladung senden/ }).click()
await page.waitForTimeout(900)
out.sentToast = await page.getByText(/Einladung an neu.kollegin@firma.de gesendet/).count()
out.dialogClosed = (await page.getByRole('dialog').count()) === 0
await page.screenshot({ path: resolve(outDir, '3-sent.png') })

out.errs = errs.length
out.errDetail = errs.slice(0, 3)
console.log(JSON.stringify(out, null, 2))
await browser.close()
