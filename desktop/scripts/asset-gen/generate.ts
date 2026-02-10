/**
 * Asset Generator — Hauptscript.
 *
 * Nutzung:
 *   npx tsx scripts/asset-gen/generate.ts                  → alle Assets
 *   npx tsx scripts/asset-gen/generate.ts cozy-plant       → ein bestimmtes Asset
 *   npx tsx scripts/asset-gen/generate.ts --theme cozy     → alle Cozy-Assets
 *   npx tsx scripts/asset-gen/generate.ts --skip-check     → ohne Quality-Check
 */
import 'dotenv/config'
import OpenAI from 'openai'
import * as fs from 'fs'
import * as path from 'path'
import { ASSETS, ALL_ASSET_KEYS, COZY_ASSET_KEYS, DREAMY_ASSET_KEYS } from './assets'
import type { AssetDefinition } from './assets'
import { buildStylePrompt } from './style-dna'
import { checkAssetQuality } from './quality-check'

// --- Config ---
const MAX_RETRIES = 3
const OUTPUT_BASE = path.resolve(__dirname, '../../src/renderer/assets/desk')

// --- OpenAI Client ---
const apiKey = process.env.OPENAI_API_KEY
if (!apiKey) {
  console.error('ERROR: OPENAI_API_KEY not found in .env')
  process.exit(1)
}
const client = new OpenAI({ apiKey })

// --- CLI Args ---
const args = process.argv.slice(2)
const skipCheck = args.includes('--skip-check')
const themeArg = args.indexOf('--theme')
const themeFilter = themeArg !== -1 ? args[themeArg + 1] : null

// Figure out which assets to generate
let targetKeys: string[]
if (themeFilter === 'cozy') {
  targetKeys = COZY_ASSET_KEYS
} else if (themeFilter === 'dreamy') {
  targetKeys = DREAMY_ASSET_KEYS
} else {
  const specificKeys = args.filter((a) => !a.startsWith('--') && ASSETS[a])
  targetKeys = specificKeys.length > 0 ? specificKeys : ALL_ASSET_KEYS
}

// --- Generate one asset ---
async function generateAsset(key: string, asset: AssetDefinition): Promise<boolean> {
  const fullPrompt = `${asset.prompt}\n\nStyle requirements: ${buildStylePrompt(asset.theme)}`
  const outputPath = path.join(OUTPUT_BASE, `${asset.outputFile}.png`)

  fs.mkdirSync(path.dirname(outputPath), { recursive: true })

  for (let attempt = 1; attempt <= MAX_RETRIES; attempt++) {
    console.log(`\n${'='.repeat(50)}`)
    console.log(`Generating: ${asset.name} (${key}) — attempt ${attempt}/${MAX_RETRIES}`)
    console.log(`Theme: ${asset.theme}`)

    try {
      console.log('Calling OpenAI Image API...')
      const response = await client.images.generate({
        model: 'gpt-image-1',
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

      // Save image — gpt-image-1 returns b64_json by default
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
      console.log(`Saved: ${outputPath}`)

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
  console.log(`Assets to generate: ${targetKeys.length}`)
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
  for (const r of results) {
    console.log(`  [${r.success ? 'OK' : 'FAIL'}] ${r.key}`)
  }

  const ok = results.filter((r) => r.success).length
  console.log(`\n${ok}/${results.length} assets generated successfully.`)
}

main().catch((err) => {
  console.error('Fatal error:', err)
  process.exit(1)
})
