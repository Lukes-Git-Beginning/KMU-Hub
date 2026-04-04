/**
 * Generate all icon derivatives from the high-res branding source images.
 *
 * Usage:  node scripts/generate-icons.mjs
 *
 * Source images (must exist in src/renderer/assets/branding/):
 *   - cosmi-logo-transparent.png  (1280x853, full logo with text)
 *   - zentria-logo.png            (539x475, Z-icon, already transparent)
 *
 * Generated outputs:
 *   - branding/cosmi-icon-64.png      (64x64, icon only)
 *   - branding/cosmi-icon-128.png     (128x128, icon only)
 *   - build/icon.png                  (256x256, Electron installer)
 *   - build/icon-512.png              (512x512, Electron)
 *   - resources/tray-icon.png         (16x16, system tray)
 *   - resources/tray-icon@2x.png      (32x32, system tray HiDPI)
 *   - public/favicon.png              (32x32, browser tab)
 *   - branding/zentria-logo-transparent.png (copy of zentria-logo.png)
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

async function generateCosmiIcons() {
  const src = resolve(BRANDING, 'cosmi-logo-transparent.png')
  const meta = await sharp(src).metadata()
  const { width, height } = meta

  // The icon (hexagon + orbit ring) occupies roughly the top 62% of the image.
  // We crop a square region from the center-top that captures the icon without text.
  const iconHeight = Math.round(height * 0.62)
  const iconSize = Math.min(width, iconHeight)
  const left = Math.round((width - iconSize) / 2)

  // Extract icon region
  const iconBuffer = await sharp(src)
    .extract({ left, top: 0, width: iconSize, height: iconSize })
    .toBuffer()

  // Generate all sizes from the cropped icon
  const targets = [
    { path: resolve(BRANDING, 'cosmi-icon-64.png'), size: 64 },
    { path: resolve(BRANDING, 'cosmi-icon-128.png'), size: 128 },
    { path: resolve(BUILD, 'icon.png'), size: 256 },
    { path: resolve(BUILD, 'icon-512.png'), size: 512 },
    { path: resolve(RESOURCES, 'tray-icon.png'), size: 16 },
    { path: resolve(RESOURCES, 'tray-icon@2x.png'), size: 32 },
    { path: resolve(PUBLIC, 'favicon.png'), size: 32 },
  ]

  for (const { path, size } of targets) {
    mkdirSync(dirname(path), { recursive: true })
    await sharp(iconBuffer)
      .resize(size, size, { fit: 'contain', background: { r: 0, g: 0, b: 0, alpha: 0 } })
      .png()
      .toFile(path)
    console.log(`  ✓ ${path} (${size}x${size})`)
  }
}

async function generateZentriaTransparent() {
  const src = resolve(BRANDING, 'zentria-logo.png')
  const dst = resolve(BRANDING, 'zentria-logo-transparent.png')
  copyFileSync(src, dst)
  console.log(`  ✓ ${dst} (copy of zentria-logo.png)`)
}

async function main() {
  console.log('Generating Cosmi icon derivatives...')
  await generateCosmiIcons()

  console.log('\nGenerating Zentria transparent variant...')
  await generateZentriaTransparent()

  console.log('\nDone! All icons generated.')
}

main().catch((err) => {
  console.error('Failed:', err)
  process.exit(1)
})
