// Schluss-QA S-5: konsolidierter Hub (10 Sub-Tabs), Legacy-Redirect, DE+EN.
import { chromium } from 'playwright'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const FE = process.env.QA_FE || 'http://localhost:5173'
const outDir = resolve('.qa-screenshots')
const ELECTRON_STUB = `
  const noop = () => Promise.resolve(undefined)
  const handler = { get: (_t, prop) => (prop === 'then' ? undefined : new Proxy(noop, handler)), apply: () => Promise.resolve(undefined) }
  const fallback = new Proxy(noop, handler)
  const authApi = { getStoredTokens: async () => null, storeTokens: async () => {}, clearTokens: async () => {} }
  window.electronAPI = new Proxy({}, { get: (_t, p) => (p === 'auth' ? authApi : p === 'then' ? undefined : fallback[p]) })
`
const suppress = (lang) => `
  try {
    const K='cosmi-ui'; const r=localStorage.getItem(K); const p=r?JSON.parse(r):{state:{},version:0}
    p.state={...(p.state||{}),onboardingCompleted:true}; localStorage.setItem(K,JSON.stringify(p))
    localStorage.setItem('cosmi-locale', JSON.stringify({ state: { locale: '${lang}' }, version: 0 }))
  } catch(e){}
`
const TABS = ['audit', 'gdpr', 'dsar', 'retention', 'sessions', 'password-policy', 'ip-whitelist', 'vault', 'privacy', 'ai']

async function run(lang) {
  await mkdir(outDir, { recursive: true })
  const browser = await chromium.launch()
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 }, reducedMotion: 'reduce' })
  await ctx.addInitScript(ELECTRON_STUB)
  await ctx.addInitScript(suppress(lang))
  await ctx.addInitScript(`try { sessionStorage.setItem('cosmi:launch-played', '1') } catch (e) {}`)
  const page = await ctx.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(String(e)))
  const crashed = []
  const rawKeys = new Set()
  const hardcoded = new Set()

  // Legacy redirect check (only DE run)
  if (lang === 'de') {
    await page.goto(`${FE}/#/admin/security-legacy`, { waitUntil: 'domcontentloaded', timeout: 30000 })
    await page.waitForTimeout(2500)
    const url = page.url()
    console.log(`  legacy redirect → ${url.includes('/admin/security') && !url.includes('legacy') ? 'OK (/admin/security)' : 'FAIL: ' + url}`)
  }

  await page.goto(`${FE}/#/admin/security`, { waitUntil: 'domcontentloaded', timeout: 30000 })
  await page.waitForTimeout(2500)

  for (const key of TABS) {
    const ok = await page.evaluate((k) => { const el = document.querySelector(`#security-tab-${k}`); if (el) { el.click(); return true } return false }, key)
    await page.waitForTimeout(800)
    if (!ok) { console.log(`  [warn] ${lang} tab not found: ${key}`); continue }
    if (lang === 'de' || ['audit', 'vault', 'sessions', 'ip-whitelist'].includes(key)) {
      await page.waitForTimeout(300)
      await page.screenshot({ path: resolve(outDir, `final-${lang}-${key}.png`) })
    }
    const body = await page.evaluate(() => document.body.innerText)
    if (/schiefgelaufen|went wrong/i.test(body)) crashed.push(key)
    // raw i18n keys
    ;(body.match(/\b(security|gdpr|audit|ipAccess|vault|password|session|admin)\.[a-zA-Z.]{3,}/g) || []).forEach((m) => rawKeys.add(m))
    // leftover hardcoded english strings that should be translated
    if (lang === 'de') {
      ;['My Sessions', 'All Sessions', ' Secrets', 'Office network', 'no expiry', 'Enter a password to test'].forEach((s) => { if (body.includes(s)) hardcoded.add(`${key}:${s}`) })
    }
  }
  console.log(`  [${lang}] crashes: ${crashed.length ? crashed.join(',') : 'keine'} | raw-keys: ${rawKeys.size ? [...rawKeys].slice(0, 8).join(',') : 'keine'} | hardcoded: ${hardcoded.size ? [...hardcoded].join(',') : 'keine'} | pageerrors: ${errors.length}`)
  await browser.close()
  return { crashed: crashed.length, rawKeys: rawKeys.size, hardcoded: hardcoded.size }
}

console.log('=== SCHLUSS-QA ===')
const de = await run('de')
const en = await run('en')
console.log('\n=== ZUSAMMENFASSUNG ===')
console.log('DE:', JSON.stringify(de))
console.log('EN:', JSON.stringify(en))
