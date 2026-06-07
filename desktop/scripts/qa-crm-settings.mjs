import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/crm-settings')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const rawRe = /(crm\.settings\.|moduleSettings\.)/
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 10)
  }, rawRe.source)
}

async function openSettingsOverlay(page) {
  await page.locator('a:has-text("Einstellungen"), button:has-text("Einstellungen")').first().click({ timeout: 8000 })
  await page.waitForTimeout(800)
}

const browser = await chromium.launch()
const out = []

// 1) From Kontakte → CRM preselected (context), pipeline + custom fields render
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage(); const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))
  try {
    await page.goto(`${BASE}/#/kontakte`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(1600)
    // sidebar button checks: admin gone, settings renamed
    const sidebar = await page.evaluate(() => ({
      hasAdmin: document.body.textContent.includes('Administration'),
      hasModulSettings: document.body.textContent.includes('Modul-Einstellungen'),
    }))
    out.push({ step: 'sidebar', ...sidebar })
    await openSettingsOverlay(page)
    const sections = await page.evaluate(() =>
      ['Persönliche Ansicht', 'Pipeline-Phasen', 'Eigene Felder', 'Tag-Verwaltung'].filter((s) => document.body.textContent.includes(s)))
    const scopeGroups = await page.evaluate(() =>
      ['Persönlich', 'Für alle'].filter((g) => document.body.textContent.includes(g)))
    out.push({ step: 'scope-groups', groups: scopeGroups })
    const stages = await page.evaluate(() =>
      ['Lead', 'Qualifiziert', 'Angebot', 'Gewonnen'].filter((s) => document.body.textContent.includes(s)))
    const rk = await rawKeys(page)
    await page.screenshot({ path: resolve(outDir, `crm-panel.png`) })
    out.push({ step: 'crm-panel', sections, stagesVisible: stages, rawKeys: rk, errs: errs.length })

    // 2) Add a new stage
    await page.locator('[role="dialog"] button:has-text("Neue Phase")').first().click({ timeout: 6000 })
    await page.waitForTimeout(400)
    await page.locator('input[placeholder="Phasenname"]').fill('QA Testphase')
    await page.locator('[role="dialog"] button:has-text("Phase hinzufügen")').click()
    await page.waitForTimeout(900)
    const added = await page.evaluate(() => document.body.textContent.includes('QA Testphase'))
    await page.screenshot({ path: resolve(outDir, `crm-panel-added.png`) })
    out.push({ step: 'add-stage', stageAdded: added, errs: errs.length })
  } catch (e) {
    out.push({ step: 'crm-panel', error: String(e).split('\n')[0] })
  } finally { await ctx.close() }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
