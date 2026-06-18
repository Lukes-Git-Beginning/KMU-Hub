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

function rawKeys(page) {
  return page.evaluate(() => {
    const rx = /(\{\{|team\.page\.|team\.selfService\.|team\.orgChart\.|helpdesk\.|glossary\.)[a-zA-Z.]*\}?\}?/
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => (n.textContent || '').trim())
      .slice(0, 10)
  })
}

async function shot(page, name) { await page.screenshot({ path: resolve(outDir, name) }) }

async function newPage() {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e).split('\n')[0]))
  return { ctx, page, errs }
}

async function clickTab(page, label) {
  const tab = page.getByRole('button', { name: label, exact: false }).first()
  await tab.click({ timeout: 5000 }).catch(async () => {
    await page.getByText(label, { exact: false }).first().click({ timeout: 5000 })
  })
  await page.waitForTimeout(900)
}

// ============ TEAM ============
{
  const { ctx, page, errs } = await newPage()
  try {
    await page.goto(`${BASE}/#/team`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2200)
    await shot(page, 'team-01-members.png')
    // Sidebar user name (F1)
    const sidebarName = await page.evaluate(() => {
      const el = Array.from(document.querySelectorAll('*')).find((n) => /Stefan (Vogel|Müller)/.test(n.textContent || '') && (n.textContent || '').length < 40)
      return el ? el.textContent.trim() : null
    })
    out.push({ step: 'team-load', sidebarName, rawKeys: await rawKeys(page), errs: errs.length })

    // Self-Service → Gehaltsabrechnungen (F2)
    await clickTab(page, 'Self-Service')
    await shot(page, 'team-02-selfservice-profile.png')
    const ssName = await page.evaluate(() => {
      const h = Array.from(document.querySelectorAll('h2')).find((n) => /Stefan/.test(n.textContent || ''))
      return h ? h.textContent.trim() : null
    })
    // click salary tab inside self-service
    await page.getByText('Gehaltsabrechnungen', { exact: false }).first().click({ timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(700)
    await shot(page, 'team-03-salary-list.png')
    // click first salary row → preview
    await page.locator('table tbody tr').first().click({ timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(700)
    await shot(page, 'team-04-salary-preview.png')
    const previewOpen = await page.evaluate(() => /Demo-Vorschau/.test(document.body.textContent || ''))
    out.push({ step: 'selfservice', ssName, salaryPreviewOpen: previewOpen, errs: errs.length })
    // close
    await page.keyboard.press('Escape'); await page.waitForTimeout(400)

    // Organigramm → click person → DetailModal (F3)
    await clickTab(page, 'Organigramm')
    await page.waitForTimeout(800)
    await page.locator('.bg-card.p-3.cursor-pointer, [class*="cursor-pointer"]').first().click({ timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(800)
    await shot(page, 'team-05-orgchart-modal.png')
    const orgModal = await page.locator('[role="dialog"]').count()
    out.push({ step: 'orgchart', dialogOpen: orgModal, errs: errs.length })
    await page.keyboard.press('Escape'); await page.waitForTimeout(400)

    // Schulungen → catalog → click training → detail modal (F5)
    await clickTab(page, 'Schulungen')
    await page.waitForTimeout(800)
    await shot(page, 'team-06-trainings-catalog.png')
    await page.locator('[role="button"]').filter({ hasText: 'Erste Hilfe' }).first().click({ timeout: 5000 }).catch(async () => {
      await page.getByText('Erste Hilfe', { exact: false }).first().click({ timeout: 5000 }).catch(() => {})
    })
    await page.waitForTimeout(800)
    await shot(page, 'team-07-training-detail.png')
    const trainingDetail = await page.evaluate(() => /Lernziele|Unterlagen|Teilnehmer/.test(document.body.textContent || ''))
    out.push({ step: 'training-detail', hasDepthSections: trainingDetail, rawKeys: await rawKeys(page), errs: errs.length })
    await page.keyboard.press('Escape'); await page.waitForTimeout(400)
    // Add training dialog
    await page.getByText('Schulung anlegen', { exact: false }).first().click({ timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(700)
    await shot(page, 'team-08-training-add.png')
    const addFields = await page.evaluate(() => ['Beschreibung', 'Lernziele', 'Unterlagen'].filter((f) => document.body.textContent.includes(f)))
    out.push({ step: 'training-add', deepFields: addFields, errs: errs.length })
  } catch (e) { out.push({ step: 'team', error: String(e).split('\n')[0] }) } finally { await ctx.close() }
}

// ============ HELPDESK ============
{
  const { ctx, page, errs } = await newPage()
  try {
    await page.goto(`${BASE}/#/helpdesk`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2200)
    await shot(page, 'hd-01-list.png')
    out.push({ step: 'hd-load', rawKeys: await rawKeys(page), errs: errs.length })

    // open first ticket → modal (F7 wider)
    await page.locator('table tbody tr').first().click({ timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(900)
    await shot(page, 'hd-02-ticket-modal.png')
    const dialogW = await page.evaluate(() => {
      const d = document.querySelector('[role="dialog"]')
      return d ? Math.round(d.getBoundingClientRect().width) : null
    })
    // open canned responses popover (F6)
    await page.getByText('Vorlagen', { exact: false }).first().click({ timeout: 5000 }).catch(() => {})
    await page.waitForTimeout(700)
    await shot(page, 'hd-03-canned-popover.png')
    const popoverInViewport = await page.evaluate(() => {
      const inputs = Array.from(document.querySelectorAll('[role="dialog"] input, body > div input'))
      const search = Array.from(document.querySelectorAll('input')).find((i) => i.placeholder && /such/i.test(i.placeholder))
      if (!search) return null
      const r = search.getBoundingClientRect()
      return { right: Math.round(r.right), withinViewport: r.right <= window.innerWidth && r.left >= 0 }
    })
    out.push({ step: 'hd-ticket', dialogWidth: dialogW, cannedPopover: popoverInViewport, errs: errs.length })
  } catch (e) { out.push({ step: 'helpdesk', error: String(e).split('\n')[0] }) } finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
