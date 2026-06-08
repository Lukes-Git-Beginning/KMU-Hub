// QA Kommunikation Phase 5: @mention + /slash in composer, call buttons,
// collision banner, settings sections (presence/notifications/channels/
// canned/webhooks/retention). Screenshots + raw-key scan + pageErrors.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>(p==='then'?undefined:new Proxy(noop,h)),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

async function scanRawKeys(page) {
  const text = await page.evaluate(() => document.body.innerText)
  return [...new Set([...text.matchAll(/\b(kommunikation|chat|moduleSettings|settings|common)\.[a-zA-Z][a-zA-Z0-9.]+\b/g)].map((m) => m[0]))]
}
async function shot(page, name) { await page.screenshot({ path: resolve(outDir, name), fullPage: true }) }

await mkdir(outDir, { recursive: true })
const browser = await chromium.launch()
const out = {}
const errors = []
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
page.on('pageerror', (e) => errors.push(String(e)))

try {
  await page.goto(`${BASE}/#/kommunikation?bereich=posteingang`, { waitUntil: 'domcontentloaded', timeout: 20000 })
  await page.waitForTimeout(2500)

  // Select a conversation
  await page.getByText('Peter Gruber').first().click({ timeout: 6000 }).catch((e) => { out.selectErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1000)

  // Call buttons present
  out.audioCallBtn = await page.getByTitle('Audioanruf').first().isVisible().catch(() => false)
  out.videoCallBtn = await page.getByTitle('Videoanruf').first().isVisible().catch(() => false)

  // Collision banner — scan all conversations for at least one banner
  out.collisionSeen = false
  for (const name of ['Peter Gruber', 'Anna Schneider', 'Michael Brunner', 'Maria Huber', 'Werner Koch', 'Sophie Lang']) {
    await page.getByText(name).first().click({ timeout: 3000 }).catch(() => {})
    await page.waitForTimeout(400)
    const seen = await page.getByText(/bearbeitet diese Konversation/).first().isVisible().catch(() => false)
    if (seen) { out.collisionSeen = true; await shot(page, 'synergy-collision.png'); break }
  }

  // Reselect a conversation for composer tests
  await page.getByText('Peter Gruber').first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(600)

  // Slash-command palette: type "/" in the reply composer
  const reply = page.getByPlaceholder('Antwort schreiben…').or(page.locator('textarea').last())
  await reply.first().click({ timeout: 4000 }).catch((e) => { out.replyClickErr = String(e).split('\n')[0] })
  await reply.first().fill('/').catch(() => {})
  await page.waitForTimeout(500)
  out.slashPaletteVisible = await page.getByText('/giphy').first().isVisible().catch(() => false)
  await shot(page, 'synergy-slash.png')

  // @mention autocomplete: type "@"
  await reply.first().fill('@').catch(() => {})
  await page.waitForTimeout(500)
  out.mentionVisible = await page.getByText('Mitarbeiter').first().isVisible().catch(() => false)
  await shot(page, 'synergy-mention.png')
  await reply.first().fill('').catch(() => {})

  // Settings sections
  await page.getByText(/Modul-Einstellungen/i).first().click({ timeout: 5000 }).catch((e) => { out.settingsOpenErr = String(e).split('\n')[0] })
  await page.waitForTimeout(1000)
  await page.getByText(/^Kommunikation$/).first().click({ timeout: 4000 }).catch(() => {})
  await page.waitForTimeout(900)
  out.secPresence = await page.getByText('Eigener Status').first().isVisible().catch(() => false)
  out.secNotifications = await page.getByText(/^Benachrichtigungen$/).first().isVisible().catch(() => false)
  out.secChannels = await page.getByText(/^Kanäle$/).first().isVisible().catch(() => false)
  out.secCanned = await page.getByText('Textbausteine').first().isVisible().catch(() => false)
  out.secWebhooks = await page.getByText(/^Webhooks$/).first().isVisible().catch(() => false)
  out.secRetention = await page.getByText('Aufbewahrung').first().isVisible().catch(() => false)
  await shot(page, 'synergy-settings.png')
  out.settingsRawKeys = await scanRawKeys(page)

  // Scroll settings to retention + screenshot bottom
  await page.mouse.wheel(0, 1600)
  await page.waitForTimeout(500)
  await shot(page, 'synergy-settings-bottom.png')
} catch (err) {
  out.error = String(err).split('\n')[0]
}
out.pageErrors = errors.slice(0, 8)
await ctx.close()
await browser.close()
console.log(JSON.stringify(out, null, 2))
