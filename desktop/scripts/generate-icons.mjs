/**
 * Generate all icon derivatives from high-res branding source images.
 *
 * Usage:  node scripts/generate-icons.mjs
 *
 * Source images (in src/renderer/assets/branding/):
 *   - Cosmi-ohne-Schriftzug-Transparent.png  (1280x853, icon only)
 *   - Orbit-Ohne-Schriftzug-Transparent.png  (1280x853, icon only)
 *   - zentria-logo-transparent 64px.png      (64x64)
 *   - zentria-logo-transparent 128px.png     (128x128)
 *   - zentria-logo.png                       (539x475, already transparent)
 */

import sharp from 'sharp'
import { copyFileSync, mkdirSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const root = resolve(__dirname, '..')

const BRANDING = resolve(root, 'src/renderer/assets/branding')
const BUILD = resolve(root, 'build')
const RESOURCES = resolve(root, 'src/main/resources')
const PUBLIC = resolve(root, 'src/renderer/public')

/**
 * Trim transparent pixels from a PNG, then resize to target with padding.
 * This ensures the icon is centered and not clipped at edges.
 */
async function trimAndResize(srcPath, size) {
  return sharp(srcPath)
    .trim()
    .resize(size, size, {
      fit: 'contain',
      background: { r: 0, g: 0, b: 0, alpha: 0 },
    })
    .png()
    .toBuffer()
}

async function generateCosmiIcons() {
  const src = resolve(BRANDING, 'Cosmi-ohne-Schriftzug-Transparent.png')

  const targets = [
    { path: resolve(BRANDING, 'cosmi-icon-64.png'), size: 64 },
    { path: resolve(BRANDING, 'cosmi-icon-128.png'), size: 128 },
    { path: resolve(RESOURCES, 'tray-icon.png'), size: 16 },
    { path: resolve(RESOURCES, 'tray-icon@2x.png'), size: 32 },
    { path: resolve(PUBLIC, 'favicon.png'), size: 32 },
    { path: resolve(BUILD, 'icon.png'), size: 256 },
    { path: resolve(BUILD, 'icon-512.png'), size: 512 },
    // Larger icon for BrowserWindow taskbar (Windows needs 256px+)
    { path: resolve(RESOURCES, 'app-icon.png'), size: 256 },
  ]

  for (const { path, size } of targets) {
    mkdirSync(dirname(path), { recursive: true })
    const buf = await trimAndResize(src, size)
    await sharp(buf).toFile(path)
    console.log(`  ✓ ${path} (${size}x${size})`)
  }
}

async function generateOrbitIcons() {
  const src = resolve(BRANDING, 'Orbit-Ohne-Schriftzug-Transparent.png')

  const targets = [
    { path: resolve(BRANDING, 'orbit-icon-64.png'), size: 64 },
    { path: resolve(BRANDING, 'orbit-icon-128.png'), size: 128 },
  ]

  for (const { path, size } of targets) {
    const buf = await trimAndResize(src, size)
    await sharp(buf).toFile(path)
    console.log(`  ✓ ${path} (${size}x${size})`)
  }
}

async function generateZentriaIcons() {
  // Copy user-provided pixel-perfect icons
  const copies = [
    { src: 'zentria-logo-transparent 64px.png', dst: 'zentria-icon-64.png' },
    { src: 'zentria-logo-transparent 128px.png', dst: 'zentria-icon-128.png' },
  ]
  for (const { src, dst } of copies) {
    copyFileSync(resolve(BRANDING, src), resolve(BRANDING, dst))
    console.log(`  ✓ ${dst} (copy of ${src})`)
  }

  // zentria-logo.png already has transparent bg → use as zentria-logo-transparent
  copyFileSync(
    resolve(BRANDING, 'zentria-logo.png'),
    resolve(BRANDING, 'zentria-logo-transparent.png'),
  )
  console.log('  ✓ zentria-logo-transparent.png (copy of zentria-logo.png)')
}

async function main() {
  console.log('Generating Cosmi icon derivatives...')
  await generateCosmiIcons()

  console.log('\nGenerating Orbit icon derivatives...')
  await generateOrbitIcons()

  console.log('\nGenerating Zentria icon derivatives...')
  await generateZentriaIcons()

  console.log('\nDone! All icons generated.')
}

main().catch((err) => {
  console.error('Failed:', err)
  process.exit(1)
})
