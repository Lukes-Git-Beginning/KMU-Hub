// QA — wiki Phase B PB-2: frameless long-form editor + rich bubble menu (dev :5173)
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/wiki-pb2')
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
      rawKeys: [...new Set(all.filter((t) => /^(wiki|common|shared|document)\.[a-zA-Z]/.test(t)))].slice(0, 15),
      doubleBrace: [...new Set(all.filter((t) => /\{\{|\}\}/.test(t)))].slice(0, 10),
    }
  })

const out = {}
const browser = await chromium.launch()
const ctx = await browser.newContext({ viewport: { width: 1500, height: 920 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
page.setDefaultTimeout(8000)
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
const bodyText = () => page.evaluate(() => document.body.textContent || '')
const step = async (name, fn) => {
  try {
    await fn()
  } catch (err) {
    out[`ERR_${name}`] = String(err).split('\n')[0]
  }
}

await page.goto(`${BASE}/#/wiki`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500)

await step('enterEditor', async () => {
  await page.getByText('Willkommen im Cosmi-Wiki', { exact: true }).first().click()
  await page.waitForTimeout(1000)
  await page.locator('button[title="Bearbeiten"]').first().click()
  await page.waitForTimeout(1200)
  // Frameless: text blocks should NOT sit inside bordered RichTextEditor cards.
  out.framelessTextBlocks = await page.evaluate(() => {
    const editors = Array.from(document.querySelectorAll('.tiptap-content'))
    const boxed = editors.filter((e) => {
      const card = e.closest('.rounded-lg')
      return card && getComputedStyle(card).borderTopWidth !== '0px'
    })
    return { total: editors.length, boxed: boxed.length }
  })
  await page.screenshot({ path: resolve(outDir, 'pb2-1-frameless-editor.png'), fullPage: true })
})

await step('bubbleMenu', async () => {
  // Double-click a word inside the first paragraph to raise the bubble menu.
  const para = page.locator('.tiptap-content p').first()
  await para.dblclick()
  await page.waitForTimeout(700)
  out.bubbleVisible = await page.locator('div.z-50.shadow-md').count()
  // The rich bubble must offer heading + list toggles (H1/H2/H3, bullet, ordered).
  out.bubbleButtons = await page.locator('div.z-50.shadow-md button').count()
  await page.screenshot({ path: resolve(outDir, 'pb2-2-bubble-menu.png'), fullPage: false })
})

await step('headingToggle', async () => {
  // Hover a heading block to reveal its H1/H2 level toggle.
  const heading = page.locator('.group\\/wh').first()
  await heading.hover()
  await page.waitForTimeout(500)
  out.headingToggleVisible = await page.locator('.group\\/wh button[aria-label="H1"]').first().isVisible()
  await page.screenshot({ path: resolve(outDir, 'pb2-3-heading-toggle.png'), fullPage: false })
})

await step('readerUnchanged', async () => {
  // Cancel out and confirm the reader still renders correctly.
  await page.getByText('Abbrechen', { exact: true }).first().click()
  await page.waitForTimeout(800)
  out.readerHeadings = /Was du hier findest/.test(await bodyText())
  await page.screenshot({ path: resolve(outDir, 'pb2-4-reader.png'), fullPage: true })
})

await step('scanRaw', async () => Object.assign(out, await scanRaw(page)))

out.pageErrors = errs.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
