/**
 * Playwright QA — profil Phase 6: Notification Quick-Card + Account Info + Settings Shortcuts
 *
 * Szenarien:
 *  (a) Quick-Card sichtbar, Sound-Toggle klickbar + Zustand wechselt, DND im erwarteten Fallback
 *  (b) Account-Info zeigt echte/Demo-Daten — kein hardcoded "Management" / "Januar 2024"
 *  (c) Klick "Alle Benachrichtigungs-Einstellungen" → /settings mit Benachrichtigungen-Tab aktiv
 *  (d) Shortcut Erscheinungsbild → /settings mit Appearance-Tab aktiv
 *
 * Hinweise:
 *  - rawKeys-Scan auf profil.notifications.* + profil.shortcuts.* (keine echten Keys im DOM)
 *  - DND-Bereich: Backend nicht verfügbar im Dev → Switch disabled (Fallback-Text)
 *  - Sound-Toggle: rein lokal, funktioniert immer
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/profil-account')
await mkdir(outDir, { recursive: true })

// Electron API stub — alle IPC-Methoden als no-op Promises
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`

// Onboarding überspringen
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

// Demo-User in Auth-Store setzen damit user !== null (offline/dev Fallback)
const AUTH_SEED = `try{
  const AK='cosmi-cached-user';
  const u={id:'demo-1',email:'demo@zentria.tech',firstName:'Demo',lastName:'User',roles:['employee']};
  localStorage.setItem(AK, JSON.stringify(u));
}catch(e){}`

// Raw-Key-Scan: profil.notifications.* und profil.shortcuts.* sollten NICHT im DOM erscheinen
const RAW_RE = /(profil\.(notifications|shortcuts)\.)/
async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 10)
  }, RAW_RE.source)
}

const browser = await chromium.launch()
const out = []

// ─── Szenario (a) + (b): Quick-Card + Account-Info ──────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  await ctx.addInitScript(AUTH_SEED)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))

  try {
    await page.goto(`${BASE}/#/profil`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2500)

    await page.screenshot({ path: resolve(outDir, '01-profil-overview.png') })

    // (b) Account-Info: hardcoded strings dürfen NICHT erscheinen
    const bodyText = await page.evaluate(() => document.body.innerText)
    const hasManagement = /\bManagement\b/.test(bodyText)
    const hasJanuar2024 = /Januar 2024/.test(bodyText)

    // (a) Notification Quick-Card sichtbar
    // Sound-Toggle via aria-label (wir prüfen mehrere mögliche Übersetzungen)
    const soundToggle = page.locator('button[role="switch"][aria-label="Benachrichtigungstöne"], button[role="switch"][aria-label="Notification sounds"]').first()
    const soundVisible = await soundToggle.isVisible().catch(() => false)

    // Zustand vor dem Klick lesen
    const soundBefore = await soundToggle.getAttribute('aria-checked').catch(() => null)

    if (soundVisible) {
      await soundToggle.click({ timeout: 5000 })
      await page.waitForTimeout(300)
    }
    const soundAfter = await soundToggle.getAttribute('aria-checked').catch(() => null)
    const soundToggled = soundBefore !== soundAfter

    // Toggle zurücksetzen
    if (soundVisible && soundToggled) {
      await soundToggle.click({ timeout: 5000 })
      await page.waitForTimeout(200)
    }

    await page.screenshot({ path: resolve(outDir, '02-profil-quick-card.png') })

    // DND-Bereich: erwartet disabled (Backend nicht erreichbar in Dev)
    const dndSwitch = page.locator('button[role="switch"][aria-label="Bitte nicht stören"], button[role="switch"][aria-label="Do not disturb"]').first()
    const dndVisible = await dndSwitch.isVisible().catch(() => false)
    const dndDisabled = dndVisible ? await dndSwitch.isDisabled().catch(() => false) : null

    out.push({
      step: 'quick-card-and-account',
      soundToggleVisible: soundVisible,
      soundToggled,
      dndVisible,
      dndDisabledWhenBackendDown: dndDisabled,
      hardcodedManagement: hasManagement,
      hardcodedJanuar2024: hasJanuar2024,
      rawKeys: await rawKeys(page),
      pageErrors: errs.slice(),
    })
  } catch (e) {
    out.push({ step: 'quick-card-and-account', error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── Szenario (c): "Alle Benachrichtigungs-Einstellungen" → /settings Notifications-Tab ─
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  await ctx.addInitScript(AUTH_SEED)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))

  try {
    await page.goto(`${BASE}/#/profil`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2000)

    // Klick auf den "Alle Benachrichtigungs-Einstellungen" Link
    const allSettingsLink = page.locator('button[aria-label="Alle Benachrichtigungs-Einstellungen"], a[aria-label="Alle Benachrichtigungs-Einstellungen"], button[aria-label="All notification settings"]').first()
    await allSettingsLink.click({ timeout: 8000 })
    await page.waitForTimeout(1500)

    await page.screenshot({ path: resolve(outDir, '03-settings-notifications-tab.png') })

    // Verifikation: Benachrichtigungen-Tab aktiv
    // Suche nach dem aktiven Tab-Button in der Settings-Sidebar
    const notifTab = page.locator('button.text-primary:has-text("Benachrichtigungen"), button.text-primary:has-text("Notifications")').first()
    const notifTabActive = await notifTab.isVisible().catch(() => false)

    // Alternativ: Matrix-Heading
    const matrixHeading = page.locator('h3:has-text("Modul"), h3:has-text("Module")').first()
    const matrixVisible = await matrixHeading.isVisible().catch(() => false)

    out.push({
      step: 'nav-to-notifications',
      notifTabActive,
      matrixVisible,
      rawKeys: await rawKeys(page),
      pageErrors: errs.slice(),
    })
  } catch (e) {
    out.push({ step: 'nav-to-notifications', error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

// ─── Szenario (d): Shortcut Erscheinungsbild → Settings Appearance-Tab ──────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  await ctx.addInitScript(AUTH_SEED)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))

  try {
    await page.goto(`${BASE}/#/profil`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(2000)

    const appearanceBtn = page.locator('button[aria-label="Erscheinungsbild"], button[aria-label="Appearance"]').first()
    await appearanceBtn.click({ timeout: 8000 })
    await page.waitForTimeout(1500)

    await page.screenshot({ path: resolve(outDir, '04-settings-appearance-tab.png') })

    // Verifikation: Appearance-Tab aktiv — suche nach theme-switcher-Überschrift
    const themeHeading = page.locator('h3:has-text("Farbschema"), h3:has-text("Color scheme"), h3:has-text("Schéma")').first()
    const themeVisible = await themeHeading.isVisible().catch(() => false)

    out.push({
      step: 'nav-to-appearance',
      themeHeadingVisible: themeVisible,
      rawKeys: await rawKeys(page),
      pageErrors: errs.slice(),
    })
  } catch (e) {
    out.push({ step: 'nav-to-appearance', error: String(e).split('\n')[0] })
  } finally {
    await ctx.close()
  }
}

await browser.close()
console.log(JSON.stringify(out, null, 2))
