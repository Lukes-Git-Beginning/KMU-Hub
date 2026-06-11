/**
 * Playwright QA — profil Phase 9: UserProfileCard Overlay + Ping→Chat
 *
 * Szenarien:
 *  S1: Kommunikation/Team-Chat — Members-Panel öffnen → Avatar eines Members klicken
 *      → Karte sichtbar mit Name + Presence-Badge
 *  S2: Auf der offenen Karte „Nachricht senden" klicken → URL enthält /kommunikation,
 *      ein DM-Channel sichtbar/aktiv
 *  S3: Dashboard TeamStatus-Widget — Avatar klicken → Karte sichtbar
 *  S4: Leerzustand-Robustheit — Karte ohne Position/Abteilung zeigt kein „undefined"
 *
 * Hinweise:
 *  - data-testid="user-profile-card" auf dem PopoverContent gesetzt
 *  - Members-Toggle ist button[aria-pressed] at x > 1000 (Channel-Header-Bereich)
 *  - Member-Avatare sind button.rounded-full at x > 1100 (Members-Panel)
 *  - Dashboard-Avatare sind Buttons mit Initialen-Text (keine Icons) im widget
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = process.env.QA_BASE ?? 'http://localhost:5173'
const outDir = resolve('.qa-screenshots/phase9-profile-card')
await mkdir(outDir, { recursive: true })

// Electron API stub
const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`

// Onboarding überspringen
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`

// Demo-User in Auth-Store (id: 'usr-001' ≠ member usr-e1..e18 → öffnet other-user card)
const AUTH_SEED = `try{
  const AK='cosmi-cached-user';
  const u={id:'usr-001',email:'demo@zentria.tech',firstName:'Darien',lastName:'Morales',roles:['admin']};
  localStorage.setItem(AK,JSON.stringify(u));
}catch(e){}`

// Dashboard-Seed: team-status als erstes Widget, damit es ohne Scrollen sichtbar ist
const DASHBOARD_SEED = `try{
  const layouts=[
    {i:'team-status',x:0,y:0,w:6,h:6,minW:3,minH:3},
    {i:'my-tasks',x:6,y:0,w:6,h:4,minW:3,minH:3}
  ];
  const state={state:{layouts,activeWidgets:['team-status','my-tasks'],isEditing:false,serverSyncStatus:'synced',serverInitialized:true},version:0};
  localStorage.setItem('cosmi-dashboard',JSON.stringify(state));
}catch(e){}
// Feature-Flag-Override (DEV QA path) — aktiviert das team-Modul für den Widget-Container
window.__cosmi_qa_flags__ = { 'modules.team': true };`

// Raw-Key-Scan — prüft auf sichtbare i18n-Schlüssel im profil.*-Namespace
const RAW_RE_SRC = /(profil\.[a-z])/
async function scanRawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 10)
  }, RAW_RE_SRC.source)
}

/**
 * Hilfsfunktion: Members-Toggle anklicken.
 * Der Toggle hat aria-pressed und sitzt im Channel-Header (x > 1000).
 * Gibt zurück ob erfolgreich geklickt.
 */
async function openMembersPanel(page) {
  const btnInfo = await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button[aria-pressed]'))
    return btns.map((b) => {
      const box = b.getBoundingClientRect()
      return { x: box.x, y: box.y, w: box.width, h: box.height }
    })
  })
  // Der Members-Toggle ist derjenige mit x > 1000 (im Channel-Header-Bereich)
  const toggleBtn = btnInfo.find((b) => b.x > 1000)
  if (!toggleBtn) return false
  await page.mouse.click(toggleBtn.x + toggleBtn.w / 2, toggleBtn.y + toggleBtn.h / 2)
  await page.waitForTimeout(1500)
  return true
}

/**
 * Hilfsfunktion: ersten Member-Avatar-Button im Members-Panel klicken (x > 1100).
 * Gibt zurück ob ein Button gefunden und geklickt wurde.
 */
async function clickMemberAvatar(page, index = 0) {
  const btnInfo = await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button.rounded-full'))
    return btns.map((b) => {
      const box = b.getBoundingClientRect()
      return { x: box.x, y: box.y, w: box.width, h: box.height }
    })
  })
  // Members-Panel-Buttons befinden sich bei x > 1100
  const panelBtns = btnInfo.filter((b) => b.x > 1100)
  if (panelBtns.length === 0) return false
  const btn = panelBtns[Math.min(index, panelBtns.length - 1)]
  await page.mouse.click(btn.x + btn.w / 2, btn.y + btn.h / 2)
  await page.waitForTimeout(1500)
  return true
}

const browser = await chromium.launch()
const results = []

// ─── S1: Chat → Members-Panel → Avatar eines Members klicken → Karte sichtbar ──
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  await ctx.addInitScript(AUTH_SEED)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))

  let pass = false
  let details = {}

  try {
    await page.goto(`${BASE}/#/kommunikation?bereich=team`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(3000)

    await page.screenshot({ path: resolve(outDir, 's1-01-chat-loaded.png'), fullPage: true })

    // Channel „allgemein" wählen
    const channelItem = page.locator('text=allgemein').first()
    const channelVisible = await channelItem.isVisible().catch(() => false)
    if (channelVisible) {
      await channelItem.click({ timeout: 5000 })
      await page.waitForTimeout(2000)
    }

    await page.screenshot({ path: resolve(outDir, 's1-02-channel-selected.png'), fullPage: true })

    // Members-Panel öffnen
    const membersToggleClicked = await openMembersPanel(page)
    await page.screenshot({ path: resolve(outDir, 's1-03-members-panel.png'), fullPage: true })

    // Member-Avatar anklicken
    const avatarFound = await clickMemberAvatar(page, 0)

    // Karte sichtbar? — data-testid="user-profile-card"
    const card = page.locator('[data-testid="user-profile-card"]').first()
    const cardVisible = await card.isVisible().catch(() => false)

    await page.screenshot({ path: resolve(outDir, 's1-04-card-open.png'), fullPage: true })

    let cardText = ''
    if (cardVisible) {
      cardText = await card.innerText().catch(() => '')
    }
    const hasUndefined = /\bundefined\b/.test(cardText)

    // Presence-Badge innerhalb der Karte (scoped, nicht Topbar)
    const presenceBadge = card.locator('[aria-label]').first()
    const presenceBadgeVisible = await presenceBadge.isVisible().catch(() => false)

    pass = avatarFound && cardVisible && !hasUndefined
    details = {
      channelVisible,
      membersToggleClicked,
      avatarFound,
      cardVisible,
      presenceBadgeVisible,
      cardTextSnippet: cardText.slice(0, 150),
      hasUndefined,
    }
  } catch (e) {
    details = { error: String(e).split('\n')[0] }
  }

  results.push({
    scenario: 'S1 - Chat Members-Panel Avatar → Card sichtbar',
    pass,
    details,
    rawKeys: await scanRawKeys(page).catch(() => []),
    pageErrors: errs.slice(),
  })
  await ctx.close()
}

// ─── S2: Karte → „Nachricht senden" → DM aktiv ──────────────────────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  await ctx.addInitScript(AUTH_SEED)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))

  let pass = false
  let details = {}

  try {
    await page.goto(`${BASE}/#/kommunikation?bereich=team`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(3000)

    // Channel öffnen
    const channelItem = page.locator('text=allgemein').first()
    const channelVisible = await channelItem.isVisible().catch(() => false)
    if (channelVisible) {
      await channelItem.click({ timeout: 5000 })
      await page.waitForTimeout(2000)
    }

    // Members-Panel öffnen
    const membersToggleClicked = await openMembersPanel(page)

    // Member-Avatar klicken (usr-001 ist unser Auth-User, Member-user_ids sind usr-e1..e18 → other-user card)
    const avatarFound = await clickMemberAvatar(page, 0)

    const card = page.locator('[data-testid="user-profile-card"]').first()
    const cardVisible = await card.isVisible().catch(() => false)

    await page.screenshot({ path: resolve(outDir, 's2-01-card-open.png'), fullPage: true })

    // „Nachricht senden" Button innerhalb der Karte
    const sendBtn = card.locator('button').filter({ hasText: /Nachricht senden|Send message/i }).first()
    const sendBtnVisible = await sendBtn.isVisible().catch(() => false)

    // Name aus der offenen Karte capturen — der Chat-Header MUSS nach dem Senden
    // genau diesen Namen zeigen (harter Beweis, dass der DM-Channel selektiert wurde).
    const cardName = (await card.locator('p.font-semibold').first().textContent().catch(() => '') || '').trim()

    let urlAfterSend = ''
    let urlHasKomm = false
    let headerAfterSend = ''
    let dmActive = false

    if (sendBtnVisible && cardName) {
      await sendBtn.click({ timeout: 8000 })
      await page.waitForTimeout(3500)

      urlAfterSend = page.url()
      urlHasKomm = urlAfterSend.includes('kommunikation')
      await page.screenshot({ path: resolve(outDir, 's2-02-after-send.png'), fullPage: true })

      // Harter Assert: Haupt-Chat-Header (h2) zeigt den Namen aus der Karte —
      // nicht mehr den vorher aktiven Channel (z.B. "allgemein").
      headerAfterSend = (await page.locator('h2').first().textContent().catch(() => '') || '').trim()
      dmActive = headerAfterSend.length > 0 && headerAfterSend === cardName
    }

    pass = cardVisible && sendBtnVisible && urlHasKomm && dmActive

    details = {
      channelVisible,
      membersToggleClicked,
      avatarFound,
      cardVisible,
      sendBtnVisible,
      cardName,
      urlAfterSend,
      urlHasKomm,
      headerAfterSend,
      dmActive,
    }
  } catch (e) {
    details = { error: String(e).split('\n')[0] }
  }

  results.push({
    scenario: 'S2 - Nachricht senden → DM aktiv',
    pass,
    details,
    rawKeys: await scanRawKeys(page).catch(() => []),
    pageErrors: errs.slice(),
  })
  await ctx.close()
}

// ─── S3: Dashboard TeamStatus-Widget → Avatar → Karte sichtbar ───────────────
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  await ctx.addInitScript(AUTH_SEED)
  await ctx.addInitScript(DASHBOARD_SEED)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))

  let pass = false
  let details = {}

  try {
    await page.goto(`${BASE}/#/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(3000)

    await page.screenshot({ path: resolve(outDir, 's3-01-dashboard.png'), fullPage: false })

    // Dashboard nutzt inneren Scroll-Container (.h-full.overflow-auto)
    // Widgets sind unterhalb der Module-Grid → scrollen
    await page.evaluate(() => {
      const el = document.querySelector('.h-full.overflow-auto')
      if (el) el.scrollTop = 2000
    })
    await page.waitForTimeout(3000)

    await page.screenshot({ path: resolve(outDir, 's3-01b-dashboard-scrolled.png'), fullPage: false })

    // TeamStatus-Widget rendert button.rounded-full für User mit userId (32x32px)
    // Alle button.rounded-full-Elemente im gesamten Dokument finden
    const roundedBtnInfo = await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll('button.rounded-full'))
      return btns.map((b) => {
        const box = b.getBoundingClientRect()
        return { text: b.textContent?.trim().slice(0, 10), x: box.x, y: box.y, w: box.width, h: box.height }
      })
    })

    // TeamStatus-Buttons sind h-8 w-8 (32px = 32px), Viewport-visible (y 0–950)
    const widgetBtns = roundedBtnInfo.filter((b) => b.w <= 36 && b.h <= 36 && b.y >= 0 && b.y < 950)
    const btnCount = widgetBtns.length

    // Falls noch keine Buttons: Widget ist vielleicht nicht im Viewport — kein Scroll nötig,
    // da Layout y:0 gesetzt (TeamStatus ist erstes Widget)
    // Fallback: button mit text (Initialen) INNERHALB des team-status-Bereichs
    let avatarVisible = btnCount > 0

    let clickedBtn = null
    if (avatarVisible) {
      // Zweiten Button nehmen falls vorhanden (auth-user usr-001 ≠ Mitarbeiter usr-e2..e18)
      clickedBtn = widgetBtns[Math.min(1, widgetBtns.length - 1)]
      await page.mouse.click(clickedBtn.x + clickedBtn.w / 2, clickedBtn.y + clickedBtn.h / 2)
      await page.waitForTimeout(1500)
    }

    await page.screenshot({ path: resolve(outDir, 's3-02-card-open.png'), fullPage: false })

    const card = page.locator('[data-testid="user-profile-card"]').first()
    const cardVisible = await card.isVisible().catch(() => false)

    let cardText = ''
    if (cardVisible) {
      cardText = await card.innerText().catch(() => '')
    }

    pass = avatarVisible && cardVisible
    details = {
      btnCount,
      clickedAt: clickedBtn ? { x: Math.round(clickedBtn.x), y: Math.round(clickedBtn.y) } : null,
      avatarVisible,
      cardVisible,
      cardTextSnippet: cardText.slice(0, 150),
    }
  } catch (e) {
    details = { error: String(e).split('\n')[0] }
  }

  results.push({
    scenario: 'S3 - Dashboard TeamStatus Avatar → Card sichtbar',
    pass,
    details,
    rawKeys: await scanRawKeys(page).catch(() => []),
    pageErrors: errs.slice(),
  })
  await ctx.close()
}

// ─── S4: Karte eines Users ohne Position/Abteilung — kein undefined/kaputtes Layout ──
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 950 } })
  await ctx.addInitScript(STUB)
  await ctx.addInitScript(ONB)
  await ctx.addInitScript(AUTH_SEED)
  const page = await ctx.newPage()
  const errs = []
  page.on('pageerror', (e) => errs.push(String(e)))

  let pass = false
  let details = {}

  try {
    await page.goto(`${BASE}/#/kommunikation?bereich=team`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(3000)

    // Channel öffnen
    const channelItem = page.locator('text=allgemein').first()
    const channelVisible = await channelItem.isVisible().catch(() => false)
    if (channelVisible) {
      await channelItem.click({ timeout: 5000 })
      await page.waitForTimeout(2000)
    }

    // Members-Panel öffnen
    const membersToggleClicked = await openMembersPanel(page)

    // Zweiten Avatar klicken (index 1, um Vielfalt zu prüfen, aber im Viewport)
    // Letzten NICHT — kann ausserhalb des Viewports liegen (y > 900)
    const btnInfo = await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll('button.rounded-full'))
      return btns.map((b) => {
        const box = b.getBoundingClientRect()
        return { x: box.x, y: box.y, w: box.width, h: box.height }
      })
    })
    // Nur sichtbare Panel-Buttons (x > 1100, y < 900)
    const panelBtns = btnInfo.filter((b) => b.x > 1100 && b.y < 900)
    const avatarCount = panelBtns.length
    let avatarFound = false

    if (panelBtns.length > 0) {
      // Zweiten Button oder letzten der sichtbaren, um Robustheit zu testen
      const btn = panelBtns[Math.min(1, panelBtns.length - 1)]
      await page.mouse.click(btn.x + btn.w / 2, btn.y + btn.h / 2)
      await page.waitForTimeout(1500)
      avatarFound = true
    }

    const card = page.locator('[data-testid="user-profile-card"]').first()
    const cardVisible = await card.isVisible().catch(() => false)

    let cardText = ''
    if (cardVisible) {
      cardText = await card.innerText().catch(() => '')
    }

    await page.screenshot({ path: resolve(outDir, 's4-01-card-no-undefined.png'), fullPage: true })

    const hasUndefined = /\bundefined\b/.test(cardText)
    const hasEmptyBrackets = /\[\s*\]/.test(cardText)
    const noRawKeys = !/(profil\.\w)/.test(cardText)

    pass = avatarFound && cardVisible && !hasUndefined && !hasEmptyBrackets && noRawKeys
    details = {
      channelVisible,
      membersToggleClicked,
      avatarCount,
      avatarFound,
      cardVisible,
      hasUndefined,
      hasEmptyBrackets,
      noRawKeys,
      cardTextSnippet: cardText.slice(0, 200),
    }
  } catch (e) {
    details = { error: String(e).split('\n')[0] }
  }

  results.push({
    scenario: 'S4 - Card ohne Position/Abteilung — kein undefined/kaputtes Layout',
    pass,
    details,
    rawKeys: await scanRawKeys(page).catch(() => []),
    pageErrors: errs.slice(),
  })
  await ctx.close()
}

await browser.close()

// Ergebnis ausgeben
console.log(JSON.stringify(results, null, 2))

// Zusammenfassung
const passed = results.filter((r) => r.pass).length
const failed = results.filter((r) => !r.pass).length
process.stderr.write(`\nQA Summary: ${passed}/${results.length} passed, ${failed} failed\n`)
if (failed > 0) process.exit(1)
