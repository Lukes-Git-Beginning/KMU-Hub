// QA work W-5: Auslastung + Guest view now served from MSW (project-scoped),
// not hardcoded fictional mock data.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const ELECTRON_STUB = `const noop=()=>Promise.resolve(undefined);const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve(undefined)};window.electronAPI=new Proxy(noop,h)`
const SUPPRESS = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(work|common|shared)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}
async function bodyText(page) {
  return page.evaluate(() => document.body.innerText.replace(/\n{2,}/g, '\n'))
}

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 960 } })
await ctx.addInitScript(ELECTRON_STUB)
await ctx.addInitScript(SUPPRESS)
const page = await ctx.newPage()
const errors = []
page.on('pageerror', (e) => errors.push(String(e)))
const out = {}

try {
  // ---- Auslastung view ----
  await page.goto(`${BASE}/#/work/projects/prj-001`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(3500)
  await page.locator('button[title="Auslastung"]').first().evaluate((el) => el.click())
  await page.waitForTimeout(1500)
  const aTxt = await bodyText(page)
  out.auslastung = {
    rawKeys: await scanRawKeys(page),
    hasRealMember: aTxt.includes('Stefan Müller') || aTxt.includes('Julia Weber'),
    hasFictionalMock: aTxt.includes('Anna Müller') || aTxt.includes('Lukas Braun') || aTxt.includes('Julia Hoffmann'),
    hasTeamMembersLabel: aTxt.includes('Teammitglieder') || aTxt.includes('Auslastung'),
  }
  await page.screenshot({ path: resolve(outDir, 'w5-1-auslastung.png') })

  // ---- Guest project view (real project, not the fictional mock) ----
  await page.goto(`${BASE}/#/work/projects/prj-001/guest`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(3000)
  const gTxt = await bodyText(page)
  out.guest = {
    rawKeys: await scanRawKeys(page),
    hasRealProject: gTxt.includes('Cosmi v2.0'),
    hasFictionalMock: gTxt.includes('Website Redesign 2026'),
    hasMilestones: /Go-Live|Entwicklung Kern|Konzept & Setup/.test(gTxt),
    hasStatusUpdates: /umgesetzt|API-Endpoints|Meilenstein/.test(gTxt),
  }
  await page.screenshot({ path: resolve(outDir, 'w5-2-guest.png') })

  // guest for a second project to prove it is project-scoped
  await page.goto(`${BASE}/#/work/projects/prj-002/guest`, { waitUntil: 'domcontentloaded', timeout: 25000 })
  await page.waitForTimeout(2500)
  const g2 = await bodyText(page)
  out.guest2 = {
    hasProject2: g2.includes('Website Relaunch'),
    notProject1: !g2.includes('Cosmi v2.0'),
  }
  await page.screenshot({ path: resolve(outDir, 'w5-3-guest-proj2.png') })
} catch (err) {
  out.fatal = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
