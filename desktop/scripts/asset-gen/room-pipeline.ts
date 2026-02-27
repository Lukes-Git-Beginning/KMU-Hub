/**
 * Room Scene Pipeline — Multi-step image editing approach.
 *
 * Instead of generating room scenes from text prompts (unreliable composition),
 * we start from a high-quality reference image and modify it step by step:
 *
 *   Step 1: CLEAN — Remove all objects/UI/decorations from reference → clean base
 *   Step 2: THEME — Transform the clean base into each theme variant
 *   Step 3: FURNITURE — Generate shelf/furniture overlays as transparent PNGs
 *   Step 4: DECO — Generate individual decoration objects as transparent PNGs
 *
 * Uses OpenAI images.edit API (inpainting) for Steps 1-2,
 * and images.generate for Steps 3-4 (transparent PNG objects).
 *
 * Usage:
 *   npx tsx scripts/asset-gen/room-pipeline.ts --step clean
 *   npx tsx scripts/asset-gen/room-pipeline.ts --step theme --theme dreamy
 *   npx tsx scripts/asset-gen/room-pipeline.ts --step theme --all
 *   npx tsx scripts/asset-gen/room-pipeline.ts --step furniture --theme cozy
 *   npx tsx scripts/asset-gen/room-pipeline.ts --step deco --theme cozy --item lamp
 *   npx tsx scripts/asset-gen/room-pipeline.ts --dry-run (show prompts only)
 */
import 'dotenv/config'
import OpenAI, { toFile } from 'openai'
import * as fs from 'fs'
import * as path from 'path'

const ASSETS_DIR = path.resolve(__dirname, '../../src/renderer/assets/desk')
const DESIGN_REF_DIR = path.resolve(__dirname, '../../design-reference')
const PIPELINE_DIR = path.resolve(__dirname, '../../src/renderer/assets/desk/_pipeline')

const apiKey = process.env.OPENAI_API_KEY
if (!apiKey) {
  console.error('ERROR: OPENAI_API_KEY not found in .env')
  process.exit(1)
}
const client = new OpenAI({ apiKey })
const dryRun = process.argv.includes('--dry-run')
const generateAll = process.argv.includes('--all')
const directMode = process.argv.includes('--direct')

function getArg(flag: string): string | null {
  const idx = process.argv.indexOf(flag)
  return idx !== -1 ? process.argv[idx + 1] : null
}

const step = getArg('--step')
const themeArg = getArg('--theme')

// ── Reference Images ─────────────────────────────────────────────

const REF_V1 = path.join(DESIGN_REF_DIR, 'Vorlage V1-Cozy.webp')
const REF_V3 = path.join(DESIGN_REF_DIR, 'Vorlage V3 - Cozy.png')

// V3 has the best composition (large window, correct proportions)
const REF_PRIMARY = REF_V3

// ── Theme Definitions ────────────────────────────────────────────

interface ThemeTransform {
  id: string
  name: string
  prompt: string
}

const THEME_TRANSFORMS: ThemeTransform[] = [
  {
    id: 'cozy',
    name: 'Cozy (Gemuetlich)',
    prompt: `Transform this clean room scene into a warm Scandinavian home office. Change ONLY materials, textures, colors, lighting, and the outdoor view. Do NOT move, resize, or reshape any element.

Wall: warm cream/off-white plaster with subtle trowel marks, soft mottling, matte finish.
Desk surface: honey oak with beautiful detailed grain, fine fibers, satin varnish sheen. Keep the slight drawer edge hint at the very bottom.
Window frame edge: thin light oak, minimal modern profile. The window is an architectural wall opening, NOT a picture frame.
Lighting: warm afternoon sunlight, golden bounce light on wall and desk, soft bloom.
Outside view: lush rolling hills, wildflowers, mature trees, soft blue sky, atmospheric depth.
Style: premium illustrated, Makoto Shinkai atmosphere, sharp and richly textured.

COMPOSITION LOCK — do NOT change these:
- Window position and size: exactly as in the input (76% wide, top at 8%, bottom at 73%)
- Wall gap between window and desk: keep the visible wall strip (~5%)
- Desk area: bottom 22%, keep drawer edge visible
- Keep walls and desk completely empty — no objects, no shelves, no decorations.`,
  },
  {
    id: 'dreamy',
    name: 'Dreamy (Vertraeumt)',
    prompt: `Transform this clean room scene into an enchanted elven tower study. Change ONLY materials, textures, colors, lighting, and the outdoor view. Do NOT move, resize, or reshape any element.

Wall: pale lavender-tinted plaster over stone, subtle magical shimmer/iridescence, fine micro-texture — the wall should feel like it belongs in a magical tower.
Desk surface: smooth pale stone sill/ledge (limestone) with faint speckle and polished sheen. Keep the slight drawer/ledge edge hint at the very bottom.
Window frame edge: thin brushed silver or pale stone trim, minimal, slightly rounded. The window is an architectural wall opening, NOT a picture frame.
Lighting: soft ethereal twilight glow, lavender and warm peach light, gentle bloom, subtle sparkly dust motes in the air.
Outside view: magical twilight landscape — fantasy sky with stars and wispy clouds, distant mountains above a sea of clouds, aurora-like light ribbons, atmospheric haze.
The whole room should feel MAGICAL, not just purple-tinted. Like an elven scholar's study.
Style: premium illustrated, dreamy but SHARP (dreamy = magical atmosphere, NOT blurry).

COMPOSITION LOCK — do NOT change these:
- Window position and size: exactly as in the input (76% wide, top at 8%, bottom at 73%)
- Wall gap between window and desk: keep the visible wall strip (~5%)
- Desk area: bottom 22%, keep ledge edge visible
- Keep walls and desk completely empty — no objects, no shelves, no decorations.`,
  },
  {
    id: 'nature',
    name: 'Nature (Waldhuette)',
    prompt: `Transform this clean room scene into a forest ranger's cabin study. Change ONLY materials, textures, colors, lighting, and the outdoor view. Do NOT move, resize, or reshape any element.

Wall: warm vertical timber boards or lightly weathered pine planks, visible seams, knots, subtle wear — refined forest cabin interior.
Desk surface: medium-toned rustic wood with visible grain, slight dents, matte-oil finish, warm amber highlights. Keep the slight drawer edge hint at the very bottom.
Window frame edge: thin dark-stained wood, slim profile — handcrafted cabin joinery. The window is an architectural wall opening, NOT a picture frame.
Lighting: warm amber-golden light filtering through trees outside, soft volumetric sun rays, warm bounce light on wood surfaces.
Outside view: ancient deciduous forest in golden hour — massive old trees, sunlight filtering through canopy creating god-rays, ferns and moss on forest floor, misty depth.
Style: premium illustrated, warm and grounded, rich wood textures.

COMPOSITION LOCK — do NOT change these:
- Window position and size: exactly as in the input (76% wide, top at 8%, bottom at 73%)
- Wall gap between window and desk: keep the visible wall strip (~5%)
- Desk area: bottom 22%, keep drawer edge visible
- Keep walls and desk completely empty — no objects, no shelves, no decorations.`,
  },
  {
    id: 'raumstation',
    name: 'Raumstation (Space Station)',
    prompt: `Transform this clean room scene into a luxury space station observation deck workstation. Change ONLY materials, textures, colors, lighting, and the outdoor view. Do NOT move, resize, or reshape any element.

Wall: sleek sci-fi panels, matte graphite composite with subtle seams and tiny fasteners, micro-scratches, soft ambient occlusion in panel gaps, subtle blue-white LED accent strips along panel edges.
Desk surface: dark composite ledge with fine texture, slight satin sheen, clean edges — advanced material with subtle blue LED reflection. Keep the slight ledge edge hint at the very bottom.
Window frame edge: thin titanium/aluminum viewport bezel, single minimal border with subtle inner glow. The viewport is an architectural opening, NOT a picture frame.
Lighting: cool blue-white ambient LED lighting from panel edges, cinematic contrast, subtle reflections. Nebula colors from outside reflecting faintly on desk.
Outside view: deep space panorama — stunning starfield, colorful nebula (soft purple, blue, teal), a planet partially visible at edge with atmospheric glow, cosmic dust clouds.
NOT military/aggressive — premium civilian space station, like a research institute in orbit.
Style: premium illustrated, sleek and awe-inspiring.

COMPOSITION LOCK — do NOT change these:
- Viewport position and size: exactly as in the input (76% wide, top at 8%, bottom at 73%)
- Wall gap between viewport and desk: keep the visible wall/panel strip (~5%)
- Desk area: bottom 22%, keep ledge edge visible
- Keep walls and desk completely empty — no objects, no equipment, no decorations.`,
  },
  {
    id: 'atelier',
    name: 'Atelier (Kuenstler-Werkstatt)',
    prompt: `Transform this clean room scene into a Montmartre artist's loft studio. Change ONLY materials, textures, colors, lighting, and the outdoor view. Do NOT move, resize, or reshape any element.

Wall: aged exposed brick — each brick individually visible with warm terracotta/salmon tones, white mortar lines, chips, mortar irregularity, slight warping, century-old patina. Warm sun grazing highlights on brick edges.
Desk surface: warm worn reclaimed wood with visible grain, slight dustiness, soft sheen — beautiful aged timber character. Keep the slight drawer edge hint at the very bottom.
Window frame edge: thin black metal frame, minimal modern profile. The window is an architectural loft opening, NOT a picture frame.
Lighting: warm golden hour sunset streaming through, casting beautiful warm light across brick and desk, rich shadows, warm bounce on brick, soft bloom.
Outside view: European city rooftops at golden hour — terracotta and slate rooftops, ornate chimneys, church spire in mid-distance, warm sunset sky (orange/pink/gold), atmospheric haze.
Style: premium illustrated, romantic and inspiring, rich textures.

COMPOSITION LOCK — do NOT change these:
- Window position and size: exactly as in the input (76% wide, top at 8%, bottom at 73%)
- Wall gap between window and desk: keep the visible wall strip (~5%)
- Desk area: bottom 22%, keep drawer edge visible
- Keep walls and desk completely empty — no objects, no easels, no decorations.`,
  },
]

// ── Helpers ───────────────────────────────────────────────────────

function ensureDir(dir: string) {
  fs.mkdirSync(dir, { recursive: true })
}

function loadFile(filePath: string): Buffer {
  if (!fs.existsSync(filePath)) {
    console.error(`File not found: ${filePath}`)
    process.exit(1)
  }
  return fs.readFileSync(filePath)
}

// ── Step 1: CLEAN — Edit reference to remove all objects ─────────

async function stepClean(): Promise<void> {
  console.log('\n══════════════════════════════════════════════')
  console.log('  STEP 1: CLEAN — Removing objects from reference')
  console.log('  Using images.edit with gpt-image-1 + input_fidelity=high')
  console.log('══════════════════════════════════════════════\n')

  const refPath = REF_PRIMARY
  console.log(`Reference: ${path.basename(refPath)}`)

  if (!fs.existsSync(refPath)) {
    console.error(`Reference not found: ${refPath}`)
    process.exit(1)
  }

  const prompt = `Recreate this exact room scene but with ALL objects and decorations completely REMOVED. Remove: the laptop, phone, glasses, coffee cups, mugs, pens, pencils, books, notebooks, the shelves and everything on them, all plants and flowers in pots, all small items on the desk. Also remove any UI overlay, text, or software interface.

CRITICAL COMPOSITION (percentage of total image):
- Window: centered horizontally, 76% of image width (12% margin on each side)
- Window top edge: at 8% from top of image
- Window bottom edge: at 73% from top (NOT lower — leave room for wall gap)
- Wall gap: a clearly visible 5% strip of wall between window bottom (73%) and desk top (78%)
- Desk surface: fills the bottom 22% of the image (from 78% to 100%)
- Desk should be slightly zoomed out — show the full desk surface with a hint of the front drawer edge visible at the very bottom, like a real desk viewed from slightly above

WINDOW STYLE:
- The window must be an ARCHITECTURAL WALL OPENING — a clean rectangular cutout in the wall, NOT a picture frame, NOT a framed artwork
- Thin white or light gray frame (2-3px visual weight), flush with the wall surface
- The outdoor view fills the entire opening edge-to-edge

COLOR RULES — NO YELLOW:
- Window frame: thin WHITE or light gray, NOT golden, NOT yellow
- Wall surface: neutral cool-white, NOT cream, NOT warm, NOT yellow-tinted
- Overall tone: fresh, cool, neutral — zero warm/golden cast
- Only the desk wood and outdoor greenery should have warmth

OUTDOOR VIEW through window: keep the garden/countryside landscape VIVID and sharp — lush hills, trees, clear sky, atmospheric depth. Not milky or faded.

Result: completely bare room — cool white walls, architectural window opening (73% from top), visible wall gap, then empty desk with drawer edge hint at bottom. Premium illustrated quality, sharp textures.`

  console.log('Prompt:', prompt)

  if (dryRun) {
    console.log('\n[DRY RUN] Skipping image generation.')
    return
  }

  console.log('\nGenerating clean base (gpt-image-1, input_fidelity=high)...')
  console.log('This may take 30-90 seconds...')

  // Wrap file with explicit MIME type (SDK doesn't detect .webp correctly)
  const refBuffer = fs.readFileSync(refPath)
  const mime = refPath.endsWith('.webp') ? 'image/webp' : 'image/png'
  const fileName = path.basename(refPath).replace('.webp', '.png')
  const imageInput = await toFile(refBuffer, fileName, { type: mime })

  const response = await client.images.edit({
    model: 'gpt-image-1',
    image: [imageInput],
    prompt,
    size: '1536x1024',
    quality: 'high',
    input_fidelity: 'high',
  } as Parameters<typeof client.images.edit>[0])

  const imageData = response.data?.[0]
  if (!imageData) throw new Error('No image data received')

  let buffer: Buffer
  if ('b64_json' in imageData && imageData.b64_json) {
    buffer = Buffer.from(imageData.b64_json, 'base64')
  } else if ('url' in imageData && imageData.url) {
    const res = await fetch(imageData.url)
    buffer = Buffer.from(await res.arrayBuffer())
  } else {
    throw new Error('No usable image data')
  }

  ensureDir(PIPELINE_DIR)
  const outPath = path.join(PIPELINE_DIR, 'clean-base.png')
  fs.writeFileSync(outPath, buffer)
  const sizeKB = Math.round(buffer.length / 1024)
  console.log(`\nSaved: ${outPath} (${sizeKB} KB)`)
}

// ── Step 2: THEME — Transform clean base into themed variant ─────

async function stepTheme(themeId: string): Promise<void> {
  const theme = THEME_TRANSFORMS.find((t) => t.id === themeId)
  if (!theme) {
    console.error(`Unknown theme: "${themeId}". Available: ${THEME_TRANSFORMS.map((t) => t.id).join(', ')}`)
    process.exit(1)
  }

  const mode = directMode ? 'DIRECT (V3 with furniture)' : 'CLEAN BASE'
  console.log(`\n══════════════════════════════════════════════`)
  console.log(`  STEP 2: THEME — Transforming to "${theme.name}"`)
  console.log(`  Mode: ${mode}`)
  console.log(`  Using images.edit with gpt-image-1 + input_fidelity=high`)
  console.log(`══════════════════════════════════════════════\n`)

  // Direct mode: use V3 reference with furniture intact
  // Normal mode: use clean base (stripped)
  let inputPath: string
  if (directMode) {
    inputPath = REF_PRIMARY
  } else {
    const basePath = path.join(PIPELINE_DIR, 'clean-base.png')
    inputPath = fs.existsSync(basePath) ? basePath : REF_PRIMARY
  }

  if (!fs.existsSync(inputPath)) {
    console.error(`Base image not found: ${inputPath}\nRun --step clean first (or use --direct).`)
    process.exit(1)
  }

  // In direct mode, prepend furniture-preservation instructions
  let finalPrompt = theme.prompt
  if (directMode) {
    const directPrefix = `IMPORTANT: This image contains furniture (shelves, desk items, plants, objects). Keep ALL furniture and structural elements exactly where they are. Do NOT remove, add, or move any furniture. Only change: wall materials/textures, desk surface material, window frame style, outdoor view/lighting, and overall color atmosphere.\n\n`
    // Replace the "COMPOSITION LOCK" section with a direct-mode version
    finalPrompt = directPrefix + theme.prompt
      .replace(/Keep walls and desk completely empty[^.]*\./g, 'Keep all existing furniture and objects in their exact positions.')
      .replace(/no objects[^.]*\./g, 'preserve all existing objects.')
  }

  console.log(`Input: ${path.basename(inputPath)}`)
  console.log('Theme prompt:', finalPrompt.slice(0, 200) + '...')

  if (dryRun) {
    console.log('\nFull prompt:', finalPrompt)
    console.log('\n[DRY RUN] Skipping image generation.')
    return
  }

  console.log(`\nGenerating "${theme.name}" (gpt-image-1, input_fidelity=high)...`)
  console.log('This may take 30-90 seconds...')

  const inputBuffer = fs.readFileSync(inputPath)
  const inputMime = inputPath.endsWith('.webp') ? 'image/webp' : 'image/png'
  const inputFileName = path.basename(inputPath).replace('.webp', '.png')
  const imageInput = await toFile(inputBuffer, inputFileName, { type: inputMime })

  const response = await client.images.edit({
    model: 'gpt-image-1',
    image: [imageInput],
    prompt: finalPrompt,
    size: '1536x1024',
    quality: 'high',
    input_fidelity: 'high',
  } as Parameters<typeof client.images.edit>[0])

  const imageData = response.data?.[0]
  if (!imageData) throw new Error('No image data received')

  let buffer: Buffer
  if ('b64_json' in imageData && imageData.b64_json) {
    buffer = Buffer.from(imageData.b64_json, 'base64')
  } else if ('url' in imageData && imageData.url) {
    const res = await fetch(imageData.url)
    buffer = Buffer.from(await res.arrayBuffer())
  } else {
    throw new Error('No usable image data')
  }

  const outDir = directMode
    ? path.join(PIPELINE_DIR, 'direct', theme.id)
    : path.join(ASSETS_DIR, theme.id)
  ensureDir(outDir)
  const outPath = path.join(outDir, 'room-scene-light.png')
  fs.writeFileSync(outPath, buffer)
  const sizeKB = Math.round(buffer.length / 1024)
  console.log(`\nSaved: ${outPath} (${sizeKB} KB)`)
}

// ── Step 3: FURNITURE — Generate shelf overlays ──────────────────

async function stepFurniture(themeId: string): Promise<void> {
  const theme = THEME_TRANSFORMS.find((t) => t.id === themeId)
  if (!theme) {
    console.error(`Unknown theme: "${themeId}"`)
    process.exit(1)
  }

  console.log(`\n══════════════════════════════════════════════`)
  console.log(`  STEP 3: FURNITURE — Generating shelf for "${theme.name}"`)
  console.log(`══════════════════════════════════════════════\n`)

  // Load the themed room scene as context
  const roomPath = path.join(ASSETS_DIR, theme.id, 'room-scene-light.png')
  if (!fs.existsSync(roomPath)) {
    console.error(`Theme room scene not found: ${roomPath}\nRun --step theme --theme ${themeId} first.`)
    process.exit(1)
  }

  const prompt = `Based on the style and materials of this room scene, generate a wall-mounted shelf that matches the theme perfectly. The shelf should be:
- Viewed from the same frontal perspective as the room
- Rendered in the same illustrated art style
- Made of materials consistent with the room theme
- A single shelf unit (2-3 tiers) suitable for displaying small decorative objects
- On a FULLY TRANSPARENT background (PNG with alpha channel)
- No objects ON the shelf — just the empty shelf structure itself
- Size: roughly 200x300 pixels worth of content in the center of the image

Output on transparent background only.`

  console.log('Prompt:', prompt)

  if (dryRun) {
    console.log('\n[DRY RUN] Skipping image generation.')
    return
  }

  console.log('\nGenerating shelf overlay (this may take 30-60 seconds)...')

  const response = await client.images.generate({
    model: 'gpt-image-1.5',
    prompt,
    n: 1,
    size: '1024x1024',
    quality: 'high',
    background: 'transparent',
  })

  const imageData = response.data?.[0]
  if (!imageData) throw new Error('No image data received')

  let buffer: Buffer
  if ('b64_json' in imageData && imageData.b64_json) {
    buffer = Buffer.from(imageData.b64_json, 'base64')
  } else if ('url' in imageData && imageData.url) {
    const res = await fetch(imageData.url)
    buffer = Buffer.from(await res.arrayBuffer())
  } else {
    throw new Error('No usable image data')
  }

  const outDir = path.join(ASSETS_DIR, theme.id)
  ensureDir(outDir)
  const outPath = path.join(outDir, 'shelf-overlay.png')
  fs.writeFileSync(outPath, buffer)
  const sizeKB = Math.round(buffer.length / 1024)
  console.log(`\nSaved: ${outPath} (${sizeKB} KB)`)
}

// ── Step 4: DECO — Generate individual decoration objects ────────

async function stepDeco(themeId: string, itemName: string): Promise<void> {
  const theme = THEME_TRANSFORMS.find((t) => t.id === themeId)
  if (!theme) {
    console.error(`Unknown theme: "${themeId}"`)
    process.exit(1)
  }

  console.log(`\n══════════════════════════════════════════════`)
  console.log(`  STEP 4: DECO — Generating "${itemName}" for "${theme.name}"`)
  console.log(`══════════════════════════════════════════════\n`)

  // Load the themed room scene as style reference
  const roomPath = path.join(ASSETS_DIR, theme.id, 'room-scene-light.png')
  if (!fs.existsSync(roomPath)) {
    console.error(`Theme room scene not found: ${roomPath}\nRun --step theme --theme ${themeId} first.`)
    process.exit(1)
  }
  console.log(`Style reference: ${path.basename(roomPath)}`)

  const prompt = `Looking at the style, materials, and atmosphere of this room scene: generate a SINGLE ${itemName} as a separate object on a transparent background.

The object must:
- Match the EXACT art style, color palette, and lighting of this room scene
- Be viewed from a slight front-angle perspective (as if sitting on a desk or shelf in this room)
- Have rich textures and detailed material rendering consistent with the room
- Be on a FULLY TRANSPARENT background (PNG with alpha channel)
- Be well-lit with soft shadows matching the room's lighting direction
- Be centered in the image, roughly 60-70% of image area
- Be a single isolated object — nothing else, no surface, no background

Just the ${itemName}, matching this room's style perfectly.`

  console.log('Prompt:', prompt.slice(0, 200) + '...')

  if (dryRun) {
    console.log('\nFull prompt:', prompt)
    console.log('\n[DRY RUN] Skipping image generation.')
    return
  }

  console.log('\nGenerating decoration object (gpt-image-1, input_fidelity=high)...')
  console.log('This may take 30-60 seconds...')

  const roomBuffer = fs.readFileSync(roomPath)
  const imageInput = await toFile(roomBuffer, 'room-scene.png', { type: 'image/png' })

  const response = await client.images.edit({
    model: 'gpt-image-1',
    image: [imageInput],
    prompt,
    size: '1024x1024',
    quality: 'high',
  } as Parameters<typeof client.images.edit>[0])

  const imageData = response.data?.[0]
  if (!imageData) throw new Error('No image data received')

  let buffer: Buffer
  if ('b64_json' in imageData && imageData.b64_json) {
    buffer = Buffer.from(imageData.b64_json, 'base64')
  } else if ('url' in imageData && imageData.url) {
    const res = await fetch(imageData.url)
    buffer = Buffer.from(await res.arrayBuffer())
  } else {
    throw new Error('No usable image data')
  }

  const safeName = itemName.toLowerCase().replace(/[^a-z0-9]+/g, '-')
  const outDir = path.join(ASSETS_DIR, theme.id)
  ensureDir(outDir)
  const outPath = path.join(outDir, `deco-${safeName}.png`)
  fs.writeFileSync(outPath, buffer)
  const sizeKB = Math.round(buffer.length / 1024)
  console.log(`\nSaved: ${outPath} (${sizeKB} KB)`)
}

// ── Main ─────────────────────────────────────────────────────────

async function main() {
  if (!step) {
    console.log(`Room Scene Pipeline — Multi-step image editing

Usage:
  --step clean                         Remove objects from reference → clean base
  --step theme --theme <id>            Transform clean base into theme variant
  --step theme --all                   Generate all 5 theme variants
  --step furniture --theme <id>        Generate shelf overlay for theme
  --step deco --theme <id> --item <n>  Generate decoration object

Options:
  --dry-run    Show prompts without generating

Themes: cozy, dreamy, nature, raumstation, atelier

Recommended order:
  1. --step clean
  2. --step theme --all
  3. --step furniture --theme cozy (etc.)
  4. --step deco --theme cozy --item "desk lamp" (etc.)`)
    process.exit(0)
  }

  switch (step) {
    case 'clean':
      await stepClean()
      break

    case 'theme': {
      if (generateAll) {
        for (const t of THEME_TRANSFORMS) {
          await stepTheme(t.id)
        }
      } else if (themeArg) {
        await stepTheme(themeArg)
      } else {
        console.error('Specify --theme <id> or --all')
        process.exit(1)
      }
      break
    }

    case 'furniture': {
      if (!themeArg) {
        console.error('Specify --theme <id>')
        process.exit(1)
      }
      await stepFurniture(themeArg)
      break
    }

    case 'deco': {
      const item = getArg('--item')
      if (!themeArg || !item) {
        console.error('Specify --theme <id> and --item <name>')
        process.exit(1)
      }
      await stepDeco(themeArg, item)
      break
    }

    default:
      console.error(`Unknown step: "${step}". Use: clean, theme, furniture, deco`)
      process.exit(1)
  }

  console.log('\n══════════════════════════════════════════════')
  console.log('  DONE!')
  console.log('══════════════════════════════════════════════')
}

main().catch((err) => {
  console.error('Error:', err.message ?? err)
  process.exit(1)
})
