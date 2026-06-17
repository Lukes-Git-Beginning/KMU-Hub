/**
 * V-4 QA — E-Signatur „Senden": realistischer Demo-Rücklauf
 *
 * (a) Vertrag mit Signern → E-Signatur-Dialog → „Zur Unterschrift senden":
 *     offener Signer wird „Gesendet", Demo-Hinweis erscheint, Dialog bleibt offen,
 *     „Rücklauf simulieren" verfügbar.
 * (b) „Rücklauf simulieren" → Signer durchläuft Angesehen → Unterschrieben,
 *     Statusverlauf/Timeline aktualisiert.
 * (c) Audit-Log im Detail-Modal zeigt versendet / geöffnet / unterzeichnet.
 *
 * Sub-Terminal: :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/vertraege-esign')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

const CONTRACT = 'Müller Metallbau Rahmenvertrag' // v-5: Lukas Brunner (signed), Hans Müller (pending)

function clickButtonByText(page, text) {
  return page.evaluate((t) => {
    const btn = Array.from(document.querySelectorAll('button')).find(
      (b) => (b.textContent || '').trim().includes(t),
    )
    if (btn) { btn.click(); return true }
    return false
  }, text)
}

async function rawKeys(page) {
  return page.evaluate(() => {
    const rx = /vertraege\.(esignatur|history|detail)\.|\{\{|\}\}/
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => (n.textContent || '').trim())
      .slice(0, 8)
  })
}

const browser = await chromium.launch()
const out = []
const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
await ctx.addInitScript(STUB)
await ctx.addInitScript(ONB)
const page = await ctx.newPage()
const errs = []
page.on('pageerror', (e) => errs.push(String(e)))
page.on('console', (m) => { if (m.type() === 'error') errs.push('console: ' + m.text()) })

try {
  await page.goto(`${BASE}/#/vertraege`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2000)

  // Detail-Modal öffnen
  const row = page.locator('table tbody tr').filter({ hasText: CONTRACT }).first()
  await row.waitFor({ state: 'visible', timeout: 8000 })
  await row.click()
  await page.waitForSelector('[role="dialog"]', { timeout: 6000 })
  await page.waitForTimeout(600)

  // Footer „Unterschrift" → E-Signatur-Dialog
  await clickButtonByText(page, 'Unterschrift')
  await page.waitForFunction(
    () => (document.body.textContent || '').includes('Digitale Unterschrift'),
    { timeout: 6000 },
  )
  await page.waitForTimeout(600)

  // ── (a) Senden ───────────────────────────────────────────────────
  const sentClicked = await clickButtonByText(page, 'Zur Unterschrift senden')
  await page.waitForTimeout(1000)
  const afterSend = await page.evaluate(() => {
    const text = document.body.textContent || ''
    return {
      hansGesendet: /Hans Müller/.test(text) && /Gesendet/.test(text),
      demoHint: /Demo:.*simuliert/.test(text),
      simulateBtn: Array.from(document.querySelectorAll('button')).some((b) =>
        /Rücklauf simulieren/.test(b.textContent || ''),
      ),
    }
  })
  const rkSend = await rawKeys(page)
  await page.screenshot({ path: resolve(outDir, 'a-after-send.png'), fullPage: false })
  out.push({
    step: 'a-dispatch',
    sentClicked,
    ...afterSend,
    pass: sentClicked && afterSend.hansGesendet && afterSend.demoHint && afterSend.simulateBtn && rkSend.length === 0,
    rawKeys: rkSend,
    pageErrors: errs.slice(0, 4),
  })

  // ── (b) Rücklauf simulieren ──────────────────────────────────────
  const simClicked = await clickButtonByText(page, 'Rücklauf simulieren')
  await page.waitForTimeout(2600) // 1.5s Timer + Puffer
  const afterSim = await page.evaluate(() => {
    const text = document.body.textContent || ''
    // Hans-Zeile: Status sollte jetzt „Unterschrieben" sein
    const rows = Array.from(document.querySelectorAll('tr'))
    const hansRow = rows.find((r) => /Hans Müller/.test(r.textContent || ''))
    return {
      hansSigned: !!hansRow && /Unterschrieben/.test(hansRow.textContent || ''),
      timelineHasSigned: /signiert|unterzeichnet|Unterschrieben/i.test(text),
    }
  })
  const rkSim = await rawKeys(page)
  await page.screenshot({ path: resolve(outDir, 'b-after-simulate.png'), fullPage: false })
  out.push({
    step: 'b-simulated-return',
    simClicked,
    ...afterSim,
    pass: simClicked && afterSim.hansSigned && rkSim.length === 0,
    rawKeys: rkSim,
    pageErrors: errs.slice(0, 4),
  })

  // ── (c) Audit-Log im Detail-Modal ────────────────────────────────
  // E-Signatur-Dialog schließen (Abbrechen) → zurück zum Detail-Modal
  await clickButtonByText(page, 'Abbrechen')
  await page.waitForTimeout(800)
  // Detail-Modal bis zum Audit-Log scrollen
  await page.evaluate(() => {
    const dialog = document.querySelector('[role="dialog"]')
    const cands = Array.from(dialog?.querySelectorAll('*') || []).filter(
      (e) => e.scrollHeight > e.clientHeight + 20 && e.clientHeight > 150,
    )
    cands.sort((a, b) => (b.scrollHeight - b.clientHeight) - (a.scrollHeight - a.clientHeight))
    cands[0]?.scrollTo({ top: cands[0].scrollHeight, behavior: 'instant' })
  })
  await page.waitForTimeout(500)
  const auditText = await page.evaluate(() => document.querySelector('[role="dialog"]')?.textContent || '')
  const auditOk = /Zur Unterschrift versendet/.test(auditText) &&
    /Vertrag geöffnet/.test(auditText) &&
    /Vertrag unterzeichnet/.test(auditText)
  const rkAudit = await rawKeys(page)
  await page.screenshot({ path: resolve(outDir, 'c-audit-log.png'), fullPage: false })
  out.push({
    step: 'c-audit-log',
    auditOk,
    pass: auditOk && rkAudit.length === 0,
    rawKeys: rkAudit,
    pageErrors: errs.slice(0, 4),
  })
} catch (e) {
  out.push({ step: 'error', pass: false, error: String(e).split('\n')[0] })
} finally {
  await ctx.close()
}

await browser.close()

const allPass = out.every((s) => s.pass === true)
console.log(JSON.stringify(out, null, 2))
console.log(`\n=== V-4 QA RESULT: ${allPass ? 'ALL PASS' : 'FAIL'} ===`)
out.forEach((s) => console.log(`  ${s.pass ? '✓' : '✗'} ${s.step}`))
process.exit(allPass ? 0 : 1)
