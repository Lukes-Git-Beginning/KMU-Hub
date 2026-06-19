import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'
const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/skeletons')
await mkdir(outDir, { recursive: true })
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

// module route → folder name in /modules/<folder>/ (used to delay its chunk)
const TARGETS = [
  { path: 'finanzen', folder: 'finanzen' },
  { path: 'helpdesk', folder: 'helpdesk' },
  { path: 'team', folder: 'team' },
  { path: 'vertraege', folder: 'vertraege' },
  { path: 'dokumente', folder: 'dokumente' },
  { path: 'mails', folder: 'mails' },
  { path: 'fuhrpark', folder: 'fuhrpark' },
  { path: 'notifications', folder: 'notifications' },
  { path: 'work', folder: 'work' },
  { path: 'kalender', folder: 'kalender' },
  { path: 'profil', folder: 'profil' },
]
const browser = await chromium.launch()
const out = []
for (const tgt of TARGETS) {
  // --- Pass A: skeleton (delay the module chunk so the Suspense fallback stays) ---
  {
    const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
    await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
    const page = await ctx.newPage()
    try {
      await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
      await page.waitForTimeout(1600)
      await page.route((url) => url.href.includes(`/modules/${tgt.folder}/`), async (route) => {
        await new Promise((r) => setTimeout(r, 2800))
        await route.continue()
      })
      await page.evaluate((p) => { window.location.hash = '#/' + p }, tgt.path)
      await page.waitForTimeout(800)
      const hasSkel = await page.evaluate(() => !!document.querySelector('[aria-busy="true"]'))
      await page.screenshot({ path: resolve(outDir, `${tgt.path}-1-skeleton.png`) })
      out.push({ path: tgt.path, skeletonCaught: hasSkel })
    } catch (e) { out.push({ path: tgt.path, error: String(e).split('\n')[0] }) } finally { await ctx.close() }
  }
  // --- Pass B: real loaded layout (no delay) ---
  {
    const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
    await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
    const page = await ctx.newPage()
    try {
      await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
      await page.waitForTimeout(1600)
      await page.evaluate((p) => { window.location.hash = '#/' + p }, tgt.path)
      await page.waitForTimeout(2600)
      await page.screenshot({ path: resolve(outDir, `${tgt.path}-2-real.png`) })
    } catch (e) { /* noted in pass A */ } finally { await ctx.close() }
  }
}
await browser.close()
console.log(JSON.stringify(out, null, 2))
