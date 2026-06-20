// QA — wiki W-2: attachments, category rename/delete, real share token (dev :5174)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/wiki')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const scanRaw = (page) =>
  page.evaluate(() => {
    const all = Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0)
      .map((n) => (n.textContent || '').trim())
      .filter(Boolean)
    return {
      rawKeys: [...new Set(all.filter((t) => /^(wiki|common|shared|moduleSettings|settings)\.[a-zA-Z]/.test(t)))].slice(0, 15),
      doubleBrace: [...new Set(all.filter((t) => /\{\{|\}\}/.test(t)))].slice(0, 10),
      replacementChar: [...new Set(all.filter((t) => /�/.test(t)))].slice(0, 10),
    }
  })

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))

try {
  await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3500)

  // ── A) Attachments: open article that has one ──
  await page.getByText('Onboarding neuer Mitarbeitender', { exact: true }).first().click()
  await page.waitForTimeout(1000)
  let body = await page.evaluate(() => document.body.textContent || '')
  out.attachmentsTitle = /Anhänge/.test(body)
  out.seedAttachment = /onboarding-checkliste-2026\.pdf/.test(body)
  await page.screenshot({ path: resolve(outDir, 'w2-1-attachments.png'), fullPage: false })

  // add an attachment via the hidden file input
  await page.locator('input[type=file]').setInputFiles({
    name: 'qa-spezifikation.pdf',
    mimeType: 'application/pdf',
    buffer: Buffer.from('%PDF-1.4 qa test'),
  })
  await page.waitForTimeout(1000)
  body = await page.evaluate(() => document.body.textContent || '')
  out.addedAttachment = /qa-spezifikation\.pdf/.test(body)
  await page.screenshot({ path: resolve(outDir, 'w2-2-attach-added.png'), fullPage: false })

  // delete the newly added attachment (hover its row, click trash)
  const newRow = page.locator('div.group').filter({ hasText: 'qa-spezifikation.pdf' })
  await newRow.hover()
  await newRow.locator('button').first().click()
  await page.waitForTimeout(900)
  body = await page.evaluate(() => document.body.textContent || '')
  out.deletedAttachment = !/qa-spezifikation\.pdf/.test(body)

  // ── B) Category rename ──
  const allgemein = page.locator('.group').filter({ hasText: 'Allgemein' }).first()
  await allgemein.hover()
  await page.waitForTimeout(300)
  await allgemein.getByRole('button').last().click() // kebab
  await page.waitForTimeout(400)
  await page.getByRole('menuitem', { name: 'Umbenennen' }).click()
  await page.waitForTimeout(400)
  const input = page.getByRole('textbox', { name: 'Umbenennen' })
  await input.fill('Allgemein (QA)')
  await input.press('Enter')
  await page.waitForTimeout(900)
  body = await page.evaluate(() => document.body.textContent || '')
  out.renamedCategory = /Allgemein \(QA\)/.test(body)
  await page.screenshot({ path: resolve(outDir, 'w2-3-cat-renamed.png'), fullPage: false })

  // ── C) Category delete (Datenschutz) ──
  const dsCat = page.locator('.group').filter({ hasText: /Datenschutz/ }).first()
  await dsCat.hover()
  await page.waitForTimeout(300)
  await dsCat.getByRole('button').last().click()
  await page.waitForTimeout(400)
  await page.getByRole('menuitem', { name: 'Löschen' }).click()
  await page.waitForTimeout(500)
  // confirm dialog
  await page.getByRole('button', { name: 'Löschen' }).last().click()
  await page.waitForTimeout(900)
  const catCount = await page.locator('.group').filter({ hasText: /Datenschutz/ }).count()
  out.deletedCategory = catCount === 0
  await page.screenshot({ path: resolve(outDir, 'w2-4-cat-deleted.png'), fullPage: false })

  // ── D) Share dialog: real token ──
  await page.getByText('Onboarding neuer Mitarbeitender', { exact: true }).first().click()
  await page.waitForTimeout(700)
  await page.getByRole('button', { name: 'Teilen' }).first().click()
  await page.waitForTimeout(700)
  out.shareDialogOpen = /Artikel teilen/.test(await page.evaluate(() => document.body.textContent || ''))
  await page.getByRole('button', { name: 'Freigabe-Link erstellen' }).click()
  await page.waitForTimeout(900)
  body = await page.evaluate(() => document.body.textContent || '')
  out.shareTokenLink = /cosmi:\/\/wiki\/share\//.test(body)
  await page.screenshot({ path: resolve(outDir, 'w2-5-share-token.png'), fullPage: false })

  Object.assign(out, await scanRaw(page))
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
