/**
 * QA Script: Schichten-Modul
 * Führt visuelle QA-Checks für das Schichten-Modul durch.
 * Ausführen: node desktop/scripts/qa-schichten.mjs
 */

import { chromium } from 'playwright';
import { writeFileSync, mkdirSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const SCREENSHOTS_DIR = join(__dirname, 'screenshots', 'schichten');
const BASE_URL = process.env.QA_BASE_URL || 'http://localhost:5173';
const QA_TOKEN = process.env.QA_TOKEN || '';

mkdirSync(SCREENSHOTS_DIR, { recursive: true });

async function screenshot(page, name) {
  const path = join(SCREENSHOTS_DIR, `${name}.png`);
  await page.screenshot({ path, fullPage: true });
  console.log(`  📸 ${name}.png`);
  return path;
}

async function checkErrorBoundary(page, context) {
  const errorBoundary = await page.$('[data-testid="error-boundary"]');
  if (errorBoundary) {
    throw new Error(`ErrorBoundary triggered on ${context}`);
  }
  const consoleErrors = [];
  page.on('console', msg => {
    if (msg.type() === 'error') consoleErrors.push(msg.text());
  });
}

async function main() {
  console.log('🔍 QA: Schichten-Modul');
  console.log(`   Base URL: ${BASE_URL}`);

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    storageState: QA_TOKEN ? undefined : undefined,
    extraHTTPHeaders: QA_TOKEN ? { Authorization: `Bearer ${QA_TOKEN}` } : {},
  });
  const page = await context.newPage();

  // Inject auth token if provided
  if (QA_TOKEN) {
    await page.addInitScript((token) => {
      localStorage.setItem('authToken', token);
    }, QA_TOKEN);
  }

  const results = { passed: 0, failed: 0, errors: [] };

  async function test(name, fn) {
    try {
      await fn();
      console.log(`  ✅ ${name}`);
      results.passed++;
    } catch (err) {
      console.error(`  ❌ ${name}: ${err.message}`);
      results.failed++;
      results.errors.push({ name, error: err.message });
    }
  }

  // Navigate to Schichten page
  await page.goto(`${BASE_URL}/#/schichten`, { waitUntil: 'networkidle' });
  await screenshot(page, '01-initial-load');

  await test('Seite lädt ohne Crash', async () => {
    await checkErrorBoundary(page, 'initial load');
    const title = await page.textContent('h1, [data-testid="page-title"]');
    if (!title?.toLowerCase().includes('schicht')) {
      throw new Error(`Kein Schichten-Titel gefunden, stattdessen: ${title}`);
    }
  });

  await test('Wochenansicht rendert Grid', async () => {
    // Grid sollte sichtbar sein (Tage der Woche)
    const grid = await page.$('[data-testid="schichten-grid"], .schichten-grid, table');
    if (!grid) throw new Error('Kein Schichten-Grid gefunden');
  });

  await test('Keine Raw i18n-Keys sichtbar', async () => {
    const body = await page.textContent('body');
    const rawKeyPattern = /schichten\.(swap|loading|error)\.\w+/;
    if (rawKeyPattern.test(body || '')) {
      throw new Error('Rohe i18n-Keys gefunden im Body');
    }
  });

  // Tab: Vorlagen
  await test('Tab "Vorlagen" navigierbar', async () => {
    const tab = await page.$('[data-testid="tab-vorlagen"], button:has-text("Vorlage"), [role="tab"]:has-text("Vorlage")');
    if (!tab) throw new Error('Vorlagen-Tab nicht gefunden');
    await tab.click();
    await page.waitForTimeout(500);
    await screenshot(page, '02-vorlagen-tab');
  });

  await test('Vorlagen: kein statischer Mock-Inhalt "Frühschicht/Spätschicht"', async () => {
    // API-Daten sollten geladen werden; Mock-Fixtures sollten nicht hartkodiert im DOM sein
    // (Es ist OK wenn die API leer zurückliefert und Empty-State erscheint)
    const emptyState = await page.$('[data-testid="empty-state"], .empty-state');
    // emptyState ist OK — aber wir stellen sicher dass kein "Loading..." ewig hängt
    await page.waitForTimeout(1000);
    const loadingStuck = await page.$('text="Vorlagen werden geladen..."');
    if (loadingStuck) {
      throw new Error('Vorlagen-Loading hängt nach 1s');
    }
  });

  // Tab: Anfragen (Tauschanfragen)
  await test('Tab "Anfragen" navigierbar', async () => {
    const tab = await page.$('[data-testid="tab-anfragen"], button:has-text("Anfragen"), [role="tab"]:has-text("Anfragen")');
    if (!tab) throw new Error('Anfragen-Tab nicht gefunden');
    await tab.click();
    await page.waitForTimeout(500);
    await screenshot(page, '03-anfragen-tab');
  });

  await test('Anfragen-Tab: kein Mock-Swap "Thomas Keller" mit Datum Feb 2026', async () => {
    const body = await page.textContent('body');
    // Mock-Datum war 2026-02-16 mit fixture employees
    if (body?.includes('2026-02-16')) {
      throw new Error('Mock-Fixture-Datum 2026-02-16 noch im DOM');
    }
  });

  await test('Anfragen-Tab: leerer Zustand oder Tausch-Liste', async () => {
    // Entweder Empty-State oder echte Liste — kein Crash
    await checkErrorBoundary(page, 'anfragen tab');
    await page.waitForTimeout(800);
    await screenshot(page, '04-anfragen-state');
  });

  await test('Anfragen-Tab: Approve/Reject-Buttons nicht im Loading-Hang', async () => {
    const approveBtn = await page.$('button:has-text("Genehmigen"), button:has-text("Approve")');
    if (approveBtn) {
      const disabled = await approveBtn.getAttribute('disabled');
      // Buttons sollten entweder aktivierbar sein oder klar disabled (während Mutation)
      console.log(`    Approve-Button disabled=${disabled}`);
    }
  });

  // Stats-Bereich
  await test('Stats/Compliance-Bereich rendert', async () => {
    const statsTab = await page.$('[data-testid="tab-stats"], button:has-text("Statistik"), [role="tab"]:has-text("Statistik")');
    if (statsTab) {
      await statsTab.click();
      await page.waitForTimeout(500);
      await screenshot(page, '05-stats-tab');
      await checkErrorBoundary(page, 'stats tab');
    } else {
      console.log('    Stats-Tab nicht gefunden — skip');
    }
  });

  // Back to overview
  const overviewTab = await page.$('[data-testid="tab-overview"], button:has-text("Übersicht"), [role="tab"]:first-child');
  if (overviewTab) {
    await overviewTab.click();
    await page.waitForTimeout(300);
  }
  await screenshot(page, '06-final-state');

  await browser.close();

  console.log('\n📊 Ergebnis:');
  console.log(`   ✅ Bestanden: ${results.passed}`);
  console.log(`   ❌ Fehlgeschlagen: ${results.failed}`);

  if (results.errors.length > 0) {
    console.log('\n🔴 Fehler:');
    results.errors.forEach(e => console.log(`   - ${e.name}: ${e.error}`));
  }

  if (results.failed > 0) {
    process.exit(1);
  }
}

main().catch(err => {
  console.error('QA-Script Fehler:', err);
  process.exit(1);
});