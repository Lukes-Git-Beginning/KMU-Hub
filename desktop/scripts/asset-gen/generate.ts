/**
 * Asset Generator — Hauptscript.
 *
 * Nutzung:
 *   npx tsx scripts/asset-gen/generate.ts                       → alle Assets (~49 Bilder)
 *   npx tsx scripts/asset-gen/generate.ts deco-succulent        → ein bestimmtes Asset
 *   npx tsx scripts/asset-gen/generate.ts --theme cozy          → alle Cozy-Assets
 *   npx tsx scripts/asset-gen/generate.ts --category decoration → nur Deko-Objekte
 *   npx tsx scripts/asset-gen/generate.ts --category background → nur Hintergruende
 *   npx tsx scripts/asset-gen/generate.ts --category preview    → nur Previews
 *   npx tsx scripts/asset-gen/generate.ts --dark-only           → nur Dark-Mode Hintergruende
 *   npx tsx scripts/asset-gen/generate.ts --light-only          → nur Light-Mode Hintergruende
 *   npx tsx scripts/asset-gen/generate.ts --skip-check          → ohne Quality-Check
 *   npx tsx scripts/asset-gen/generate.ts --dry-run             → zeigt Prompts ohne zu generieren
 */
import 'dotenv/config'
import OpenAI from 'openai'
import * as fs from 'fs'
import * as path from 'path'
import {
  ASSETS,
  ALL_ASSET_KEYS,
  COZY_ASSET_KEYS,
  DREAMY_ASSET_KEYS,
  NATURE_ASSET_KEYS,
  INDUSTRIAL_ASSET_KEYS,
  BACKGROUND_KEYS,
  DECORATION_KEYS,
  PREVIEW_KEYS,
  LIGHT_BG_KEYS,
  DARK_BG_KEYS,
} from './assets'
import type { AssetDefinition } from './assets'
import { buildObjectPrompt, buildBackgroundPrompt, buildPreviewPrompt } from './style-dna'
import { checkAssetQuality } from './quality-check'

// --- Config ---
const MAX_RETRIES = 3
const OUTPUT_BASE = path.resolve(__dirname, '../../src/renderer/assets/desk')

// --- OpenAI Client ---
const apiKey = process.env.OPENAI_API_KEY
if (!apiKey) {
  console.error('ERROR: OPENAI_API_KEY not found in .env')
  console.error('Create a desktop/.env file with: OPENAI_API_KEY=sk-...')
  process.exit(1)
}
const client = new OpenAI({ apiKey })

// --- CLI Args ---
const args = process.argv.slice(2)
const skipCheck = args.includes('--skip-check')
const dryRun = args.includes('--dry-run')
const darkOnly = args.includes('--dark-only')
const lightOnly = args.includes('--light-only')

const themeArg = args.indexOf('--theme')
const themeFilter = themeArg !== -1 ? args[themeArg + 1] : null

const catArg = args.indexOf('--category')
const categoryFilter = catArg !== -1 ? args[catArg + 1] : null

// --- Resolve target assets ---
function resolveTargetKeys(): string[] {
  // Specific asset keys passed directly
  const specificKeys = args.filter((a) => !a.startsWith('--') && ASSETS[a])
  if (specificKeys.length > 0) return specificKeys

  let keys = ALL_ASSET_KEYS

  // Filter by theme
  if (themeFilter) {
    const themeMap: Record<string, string[]> = {
      cozy: COZY_ASSET_KEYS,
      dreamy: DREAMY_ASSET_KEYS,
      nature: NATURE_ASSET_KEYS,
      industrial: INDUSTRIAL_ASSET_KEYS,
    }
    keys = themeMap[themeFilter] ?? keys
  }

  // Filter by category
  if (categoryFilter) {
    const catMap: Record<string, string[]> = {
      background: BACKGROUND_KEYS,
      decoration: DECORATION_KEYS,
      preview: PREVIEW_KEYS,
    }
    const catKeys = catMap[categoryFilter]
    if (catKeys) {
      keys = keys.filter((k) => catKeys.includes(k))
    }
  }

  // Filter dark/light only (for backgrounds)
  if (darkOnly) {
    keys = keys.filter((k) => DARK_BG_KEYS.includes(k) || ASSETS[k].category !== 'background')
  }
  if (lightOnly) {
    keys = keys.filter((k) => LIGHT_BG_KEYS.includes(k) || ASSETS[k].category !== 'background')
  }

  return keys
}

const targetKeys = resolveTargetKeys()

// --- Build the full prompt based on asset category ---
function buildFullPrompt(asset: AssetDefinition): string {
  let stylePrompt: string
  switch (asset.category) {
    case 'background':
      stylePrompt = buildBackgroundPrompt(asset.theme, asset.isDark ?? false)
      break
    case 'preview':
      stylePrompt = buildPreviewPrompt(asset.theme)
      break
    case 'decoration':
    default:
      stylePrompt = buildObjectPrompt(asset.theme)
      break
  }
  return `${asset.prompt}\n\nStyle requirements: ${stylePrompt}`
}

// --- Generate one asset ---
async function generateAsset(key: string, asset: AssetDefinition): Promise<boolean> {
  const fullPrompt = buildFullPrompt(asset)
  const outputPath = path.join(OUTPUT_BASE, `${asset.outputFile}.png`)

  fs.mkdirSync(path.dirname(outputPath), { recursive: true })

  if (dryRun) {
    console.log(`\n${'='.repeat(50)}`)
    console.log(`[DRY RUN] ${asset.name} (${key})`)
    console.log(`Category: ${asset.category} | Theme: ${asset.theme}${asset.isDark ? ' (DARK)' : ''}`)
    console.log(`Size: ${asset.size} | Output: ${asset.outputFile}.png`)
    console.log(`Animation: ${asset.animation?.cssClass ?? 'none'}`)
    console.log(`Prompt (${fullPrompt.length} chars):`)
    console.log(fullPrompt.slice(0, 300) + '...')
    return true
  }

  for (let attempt = 1; attempt <= MAX_RETRIES; attempt++) {
    console.log(`\n${'='.repeat(50)}`)
    console.log(`Generating: ${asset.name} (${key}) — attempt ${attempt}/${MAX_RETRIES}`)
    console.log(`Category: ${asset.category} | Theme: ${asset.theme}${asset.isDark ? ' (DARK)' : ''}`)
    console.log(`Size: ${asset.size}`)

    try {
      console.log('Calling OpenAI Image API...')
      const response = await client.images.generate({
        model: 'gpt-image-1.5',
        prompt: fullPrompt,
        n: 1,
        size: asset.size,
        quality: 'high',
      })

      const imageData = response.data?.[0]
      if (!imageData) {
        console.error('No image data received')
        continue
      }

      // Save image
      let imageBuffer: Buffer
      if ('b64_json' in imageData && imageData.b64_json) {
        imageBuffer = Buffer.from(imageData.b64_json, 'base64')
      } else if ('url' in imageData && imageData.url) {
        const fetchRes = await fetch(imageData.url)
        const arrayBuf = await fetchRes.arrayBuffer()
        imageBuffer = Buffer.from(arrayBuf)
      } else {
        console.error('No usable image data in response')
        continue
      }

      fs.writeFileSync(outputPath, imageBuffer)
      const fileSizeKB = Math.round(imageBuffer.length / 1024)
      console.log(`Saved: ${outputPath} (${fileSizeKB} KB)`)

      // Quality check (optional)
      if (!skipCheck) {
        console.log('Running quality check...')
        const quality = await checkAssetQuality(client, outputPath, asset)
        console.log(`Score: ${quality.score}/10 — ${quality.passed ? 'PASSED' : 'FAILED'}`)
        console.log(`Feedback: ${quality.feedback}`)

        if (!quality.passed) {
          console.log('Improvements needed:')
          quality.improvements.forEach((imp) => console.log(`  - ${imp}`))
          if (attempt < MAX_RETRIES) {
            console.log('Retrying...')
            continue
          }
          console.log('Max retries reached. Keeping best attempt.')
        }
      }

      console.log(`Done: ${asset.name}`)
      return true
    } catch (error: any) {
      console.error(`Error: ${error.message ?? error}`)
      if (attempt === MAX_RETRIES) {
        console.error(`FAILED after ${MAX_RETRIES} attempts: ${key}`)
        return false
      }
    }
  }

  return false
}

// --- Main ---
async function main() {
  console.log('========================================')
  console.log('  KMU Hub — AI Asset Generator')
  console.log('========================================')
  console.log(`Mode: ${dryRun ? 'DRY RUN (no API calls)' : 'GENERATE'}`)
  console.log(`Assets to generate: ${targetKeys.length}`)
  console.log(`  - Backgrounds: ${targetKeys.filter((k) => ASSETS[k].category === 'background').length}`)
  console.log(`  - Decorations: ${targetKeys.filter((k) => ASSETS[k].category === 'decoration').length}`)
  console.log(`  - Previews: ${targetKeys.filter((k) => ASSETS[k].category === 'preview').length}`)
  console.log(`Quality check: ${skipCheck ? 'SKIP' : 'ON'}`)
  console.log(`Output: ${OUTPUT_BASE}`)
  console.log()

  const results: { key: string; success: boolean }[] = []

  for (const key of targetKeys) {
    const success = await generateAsset(key, ASSETS[key])
    results.push({ key, success })
  }

  // Summary
  console.log('\n========================================')
  console.log('  RESULTS')
  console.log('========================================')

  const categories = ['background', 'decoration', 'preview'] as const
  for (const cat of categories) {
    const catResults = results.filter((r) => ASSETS[r.key].category === cat)
    if (catResults.length === 0) continue
    console.log(`\n  ${cat.toUpperCase()}S:`)
    for (const r of catResults) {
      const icon = r.success ? 'OK' : 'FAIL'
      const dark = ASSETS[r.key].isDark ? ' (dark)' : ''
      console.log(`    [${icon}] ${r.key}${dark}`)
    }
  }

  const ok = results.filter((r) => r.success).length
  console.log(`\n${ok}/${results.length} assets generated successfully.`)

  if (!dryRun && ok > 0) {
    console.log(`\nAssets saved to: ${OUTPUT_BASE}`)
    console.log('Next step: Update desktop/src/renderer/src/config/desk-asset-urls.ts with imports.')
  }
}

main().catch((err) => {
  console.error('Fatal error:', err)
  process.exit(1)
})
