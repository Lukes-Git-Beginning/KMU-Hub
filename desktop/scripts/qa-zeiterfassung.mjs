// QA script for Profil > Zeiterfassung tab
// Route: http://localhost:5173/#/profil → click "Zeiterfassung" tab
//
// NOTE: Do NOT run this script — it is for human QA or CI reference only.
// Run via: node desktop/scripts/qa-zeiterfassung.mjs

import { chromium } from 'playwright'

const BASE_URL = process.env.QA_BASE_URL ?? 'http://localhost:5173'

async function main() {
  const browser = await chromium.launch({ headless: false })
  const ctx = await browser.newContext()
  await ctx.addInitScript(() => {
    localStorage.setItem('cosmi-onboarding-complete', 'true')
    localStorage.setItem('cosmi-demo-mode', 'true')
  })
  const page = await ctx.newPage()

  await page.goto(`${BASE_URL}/#/profil`)
  await page.waitForLoadState('networkidle')

  // Click Zeiterfassung tab
  await page.getByRole('tab', { name: /zeiterfassung/i }).click()
  await page.waitForTimeout(500)

  // Screenshot: Overview
  await page.screenshot({ path: 'desktop/scripts/screenshots/zeiterfassung-overview.png', fullPage: true })

  // Click Today sub-view
  await page.getByRole('button', { name: /heute|today/i }).first().click()
  await page.waitForTimeout(300)
  await page.screenshot({ path: 'desktop/scripts/screenshots/zeiterfassung-today.png', fullPage: true })

  // Click Week sub-view
  await page.getByRole('button', { name: /woche|week/i }).first().click()
  await page.waitForTimeout(300)
  await page.screenshot({ path: 'desktop/scripts/screenshots/zeiterfassung-week.png', fullPage: true })

  // Click Categories sub-view
  await page.getByRole('button', { name: /kategori|categor/i }).first().click()
  await page.waitForTimeout(300)
  await page.screenshot({ path: 'desktop/scripts/screenshots/zeiterfassung-categories.png', fullPage: true })

  // Click Team sub-view
  await page.getByRole('button', { name: /team/i }).first().click()
  await page.waitForTimeout(300)
  await page.screenshot({ path: 'desktop/scripts/screenshots/zeiterfassung-team.png', fullPage: true })

  await browser.close()
}

main().catch(console.error)
