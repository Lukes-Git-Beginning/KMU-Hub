/**
 * QA — formulare FT-3b: conversion rate + per-page drop-off.
 *  1) Eventanmeldung (multi-page, 210 share-link views) → Auswertung shows the
 *     conversion block AND the per-page drop-off funnel.
 *  2) Kundenfeedback (single page) shows conversion but NO drop-off funnel.
 *  No raw keys, 0 pageErrors. Sub-terminal → BASE :5174.
 */
import { chromium } from 'playwright'
import { resolve } from 'node:path'
import { mkdir } from 'node:fs/promises'

const BASE = 'http://localhost:5174'
const outDir = resolve('.qa-screenshots/formulare-ft3b')
await mkdir(outDir, { recursive: true })

const STUB = `const noop=()=>Promise.resolve();const h={get:(_t,p)=>p==='then'?undefined:new Proxy(noop,h),apply:()=>Promise.resolve()};window.electronAPI=new Proxy(noop,h)`
const ONB = `try{const K='cosmi-ui';const r=localStorage.getItem(K);const p=r?JSON.parse(r):{state:{},version:0};p.state={...(p.state||{}),onboardingCompleted:true};localStorage.setItem(K,JSON.stringify(p))}catch(e){}`
const RAW_RE = /(formulare\.[a-z][a-zA-Z.]+|moduleSettings\.[a-z]|\{\{|, plural,)/

async function rawKeys(page) {
  return page.evaluate((src) => {
    const rx = new RegExp(src)
    return Array.from(document.querySelectorAll('body *'))
      .filter((n) => n.children.length === 0 && rx.test(n.textContent || ''))
      .map((n) => n.textContent.trim())
      .slice(0, 12)
  }, RAW_RE.source)
}

async function openEvaluation(page, formName) {
  await page.goto(`${BASE}/#/formulare`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3200)
  await page.locator(`[role="button"]:has-text("${formName}")`).first().click({ timeout: 6000 })
  await page.waitForTimeout(700)
  await page.locator('[role="dialog"]').last().locator('button:has-text("Auswertung")').first().click({ timeout: 5000 })
  await page.waitForTimeout(2000)
}

const browser = await chromium.launch()
const out = []
const errs = []

// ── 1) Multi-page form: conversion + drop-off ──
{
  const ctx = await browser.newContext({ viewport: { width: 1100, height: 1300 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('dropoff: ' + String(e)))
  page.on('console', (m) => { if (m.type() === 'error') errs.push('dropoff console: ' + m.text()) })
  await openEvaluation(page, 'Eventanmeldung')
  const dlg = page.locator('[role="dialog"]').last()
  const text = await dlg.textContent()
  await dlg.screenshot({ path: resolve(outDir, '1-event-conversion-dropoff.png') })
  out.push({
    check: 'event-conversion-dropoff',
    hasConversion: /Conversion-Rate/.test(text || ''),
    hasViews: /Aufrufe?/.test(text || ''),
    hasDropoff: /Abbruch pro Seite/.test(text || ''),
    hasPageRows: /Seite 1/.test(text || '') && /Seite 2/.test(text || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

// ── 2) Single-page form: conversion, but no drop-off ──
{
  const ctx = await browser.newContext({ viewport: { width: 1100, height: 1100 } })
  await ctx.addInitScript(STUB); await ctx.addInitScript(ONB)
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errs.push('single: ' + String(e)))
  await openEvaluation(page, 'Kundenfeedback')
  const dlg = page.locator('[role="dialog"]').last()
  const text = await dlg.textContent()
  await dlg.screenshot({ path: resolve(outDir, '2-feedback-conversion-only.png') })
  out.push({
    check: 'feedback-conversion-only',
    hasConversion: /Conversion-Rate/.test(text || ''),
    noDropoff: !/Abbruch pro Seite/.test(text || ''),
    rawKeys: await rawKeys(page),
  })
  await ctx.close()
}

console.log(JSON.stringify({ results: out, pageErrors: errs }, null, 2))
await browser.close()
