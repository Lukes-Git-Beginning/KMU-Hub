/**
 * Room Scene Generator with GPT-5.2 Art Director.
 *
 * Flow:
 *   1. Load reference images (layout + style + optional theme-specific)
 *   2. GPT-5.2 Art Director analyzes references + briefing → optimized prompt
 *   3. GPT-Image-1.5 generates the room scene with that prompt
 *   4. GPT-5.2 + Vision reviews the result
 *
 * Usage:
 *   npx tsx scripts/asset-gen/test-room-scene.ts --theme cozy
 *   npx tsx scripts/asset-gen/test-room-scene.ts --all              (all 5 themes)
 *   npx tsx scripts/asset-gen/test-room-scene.ts --all --skip-review
 *   npx tsx scripts/asset-gen/test-room-scene.ts --theme dreamy --dry-run
 */
import 'dotenv/config'
import OpenAI from 'openai'
import * as fs from 'fs'
import * as path from 'path'

const ASSETS_DIR = path.resolve(__dirname, '../../src/renderer/assets/desk')
const DESIGN_REF_DIR = path.resolve(__dirname, '../../design-reference')

const apiKey = process.env.OPENAI_API_KEY
if (!apiKey) {
  console.error('ERROR: OPENAI_API_KEY not found in .env')
  process.exit(1)
}
const client = new OpenAI({ apiKey })
const dryRun = process.argv.includes('--dry-run')
const skipReview = process.argv.includes('--skip-review')
const generateAll = process.argv.includes('--all')

function getThemeArg(): string | null {
  const idx = process.argv.indexOf('--theme')
  return idx !== -1 ? process.argv[idx + 1] : null
}

// ── Theme Briefs ─────────────────────────────────────────────────

interface ThemeConfig {
  id: string
  name: string
  brief: string
  extraRefImage?: string // additional reference image filename in design-reference/
  extraRefCaption?: string
  wallMaterial: string
  deskMaterial: string
  windowFrame: string
}

const THEMES: ThemeConfig[] = [
  {
    id: 'cozy',
    name: 'Cozy (Gemuetlich)',
    wallMaterial: 'warm beige/cream plaster with subtle linen texture',
    deskMaterial: 'light oak wood with visible grain',
    windowFrame: 'light oak or warm off-white, minimal thickness',
    brief: `THEME: Cozy (Gemuetlich)
- Warm home office with natural materials
- Light oak wood, ceramic, linen, cotton textures
- Muted earth tones with subtle teal accents
- Warm beige and cream base colors
- Feeling of soft afternoon sunlight through the window
- Window view: Sunny garden with flowers, gentle green hills, blue sky with light clouds
- Handcrafted artisan quality, slightly imperfect warmth
- Overall: hygge, warm, inviting, calm — a place you want to work in all day`,
  },
  {
    id: 'dreamy',
    name: 'Dreamy (Vertraeumt)',
    wallMaterial: 'soft lavender/lilac plaster with subtle shimmer',
    deskMaterial: 'light birch wood with a faint pearlescent sheen',
    windowFrame: 'white with subtle silver/pearl accent, elegant thin frame',
    extraRefImage: 'Farben Vibe inspo - Dreamy.webp',
    extraRefCaption: '(Above: COLOR/VIBE reference for Dreamy theme — lavender/purple palette, floating orbs, ethereal golden leaves, magical atmosphere. Use this for the color direction and magical mood.)',
    brief: `THEME: Dreamy (Vertraeumt)
- Magical, ethereal workspace with soft fantasy elements
- Lavender, lilac, soft purple, pearl white, touches of gold
- Soft birch wood desk, walls with subtle shimmer/iridescence
- Feeling of twilight — soft purple sky with gentle glow
- Window view: Magical landscape — floating crystal formations, soft luminous clouds, distant glowing mountains, golden leaves drifting, ethereal mist, aurora-like light ribbons in the sky
- Dreamy particle effects in the view (subtle floating lights)
- Overall: magical, serene, wonder, fantasy — like working in an enchanted tower`,
  },
  {
    id: 'nature',
    name: 'Nature (Natur)',
    wallMaterial: 'warm stone/earth plaster with subtle moss texture at edges',
    deskMaterial: 'dark walnut wood with rich natural grain, slightly rough-hewn',
    windowFrame: 'dark wood frame, rustic but refined',
    brief: `THEME: Nature (Natur)
- Forest cabin workspace, grounded and organic
- Dark walnut wood, stone, moss, earth tones
- Deep greens, warm browns, forest floor colors
- Warm amber/golden light filtering through trees
- Window view: Dense deciduous forest, sunlight filtering through canopy, forest floor with ferns and moss, maybe a small stream or woodland path visible, misty depth between trees
- Feeling of being in a woodland retreat
- Overall: grounded, organic, peaceful, connected to nature — like a ranger station in an old forest`,
  },
  {
    id: 'raumstation',
    name: 'Raumstation (Space Station)',
    wallMaterial: 'sleek dark gray metallic panels with subtle blue LED edge lighting',
    deskMaterial: 'dark carbon fiber or matte black composite surface with subtle reflections',
    windowFrame: 'thin brushed steel/titanium frame with subtle blue-white glow at edges',
    brief: `THEME: Raumstation (Space Station)
- Futuristic space station workspace, high-tech but livable
- Dark grays, deep navy, brushed steel, subtle blue/cyan LED accents
- Carbon fiber desk, metallic wall panels (not industrial-dirty, but sleek sci-fi)
- Cool blue-white ambient lighting, reflections on surfaces
- Window view: Deep space panorama — stars, distant nebula with soft purple/blue colors, maybe a planet or moon partially visible at the edge, the curvature of a ringed planet, cosmic dust clouds
- NOT Star Wars/aggressive — more like ISS luxury upgrade, clean and premium
- Overall: futuristic, awe-inspiring, calm vastness — like working on an observation deck in space`,
  },
  {
    id: 'atelier',
    name: 'Atelier (Kuenstler-Werkstatt)',
    wallMaterial: 'warm exposed brick (soft terracotta/salmon tones) with white mortar',
    deskMaterial: 'thick reclaimed wood plank surface with paint splatters and character',
    windowFrame: 'black industrial steel frame, thin and elegant (loft-style)',
    brief: `THEME: Atelier (Kuenstler-Werkstatt)
- Creative artist loft workspace, bohemian and inspiring
- Exposed brick walls (warm terracotta), reclaimed wood, black steel accents
- Warm cream, terracotta, sage green, muted mustard accents
- Large industrial-style window (loft aesthetic)
- Natural light flooding in, warm golden hour feeling
- Window view: European city rooftops — old terracotta/slate roofs, chimneys, church spire in distance, warm sunset/golden hour sky, a few birds, maybe distant hills
- The view should feel like Paris/Vienna/Prague rooftops
- Overall: creative, bohemian, inspiring, urban — like working in a Montmartre studio`,
  },
]

// ── Helpers ──────────────────────────────────────────────────────

function loadImageBase64(filePath: string): string | null {
  if (!fs.existsSync(filePath)) {
    console.warn(`Reference image not found: ${filePath}`)
    return null
  }
  return fs.readFileSync(filePath).toString('base64')
}

function getMimeType(filePath: string): string {
  if (filePath.endsWith('.webp')) return 'image/webp'
  if (filePath.endsWith('.png')) return 'image/png'
  if (filePath.endsWith('.jpg') || filePath.endsWith('.jpeg')) return 'image/jpeg'
  return 'image/png'
}

// ── Art Director System Prompt ───────────────────────────────────

function buildArtDirectorSystem(theme: ThemeConfig): string {
  return `You are the Art Director for KMU Hub, a CRM desktop application. Your job is to write the PERFECT image generation prompt for our room scene backgrounds.

## The Concept
The app uses a "desk workspace" metaphor. The user looks at a PAINTED ROOM with a LARGE WINDOW. The functional app UI is positioned INSIDE the window area using CSS. Everything around the window (walls, desk, shelves, decorations) creates atmosphere.

## 5-Layer Architecture
- Layer 1 (your job): ROOM SCENE — the full painted room background (wall + window + view + desk)
- Layer 2: Furniture overlays (transparent PNGs, positioned on top — NOT your job)
- Layer 3: Decorations (small objects at mount points — NOT your job)
- Layer 4: Mount points (data only)
- Layer 5: UI skin (CSS variables)

## Your Task
You receive reference images and a theme description. You must output a DETAILED prompt for GPT-Image that will generate the room scene.

## Critical Rules
- The window must be centered and VERY LARGE (roughly 75-80% width of the image) — zoom out enough so the window dominates the wall. The side walls should be narrow strips (10-12% each side)
- The window has NO muntins, NO crossbars, NO grid lines inside — it is ONE clean open rectangle. Only a subtle thin frame around the outer edge.
- The UI will sit INSIDE the window — so the window area should show the VIEW (landscape, space, city, etc.)
- The walls on either side of the window must be COMPLETELY EMPTY — no shelves, no artwork, no decorations, no objects. Just clean wall surface. (Shelves and decorations are added as separate transparent PNG overlays in a different layer)
- The desk surface spans the bottom ~20-25% of the image and must be COMPLETELY EMPTY — no cups, no notebooks, no objects at all. Just a clean desk surface. (Objects are added separately)
- Frontal perspective — looking straight at the wall
- The desk must be COMPLETELY EMPTY — no mugs, pens, notebooks, books, plants, nothing. Just the surface material.
- The walls must be COMPLETELY EMPTY — no shelves, frames, artwork, hooks, nothing. Just wall texture.

## Theme-Specific Materials
- Wall: ${theme.wallMaterial}
- Desk: ${theme.deskMaterial}
- Window frame: ${theme.windowFrame}

## Art Style (VERY IMPORTANT)
- Modern, warm, inviting — like a high-end interior design rendering
- A touch realistic but still slightly stylized/illustrated — NOT pure watercolor/gouache, NOT overly painterly
- Think: modern digital illustration with realistic lighting and materials but soft atmosphere
- NOT cold or clinical — inviting and atmospheric
- NOT photorealistic either — still has artistic charm
- Rich detail in materials (wood grain, wall texture, surface reflections) but soft overall mood
- Theme-appropriate lighting: warm for cozy/nature/atelier, cool ambient for raumstation, soft twilight for dreamy

## About the Reference Images
- The LAYOUT reference is a rough concept sketch from the Cozy theme. The yellow frame and blue UI frame were just annotations — IGNORE those colors
- Use the STYLE reference for the art quality and rendering approach
- Adapt the materials, colors, and mood to match the THEME DESCRIPTION — do NOT copy the Cozy colors for other themes

## Output Format
Output ONLY the image generation prompt. No explanations, no markdown, just the prompt text.`
}

// ── Step 1: Art Director prompt optimization ─────────────────────

async function getArtDirectorPrompt(theme: ThemeConfig): Promise<string> {
  console.log(`\n[Step 1] Art Director (GPT-5.2) creating prompt for "${theme.name}"...`)

  const layoutRef = loadImageBase64(path.join(DESIGN_REF_DIR, 'Vorlage V2- Cozy.png'))
  const styleRef = loadImageBase64(path.join(DESIGN_REF_DIR, 'Vorlage V1-Cozy.webp'))

  const userContent: OpenAI.ChatCompletionContentPart[] = [
    {
      type: 'text',
      text: `Create the image generation prompt for this room scene:\n\n${theme.brief}\n\nImage format: 1536x1024 (landscape). The prompt should be detailed enough to produce a consistent, high-quality result.`,
    },
  ]

  if (layoutRef) {
    userContent.push({
      type: 'image_url',
      image_url: { url: `data:image/png;base64,${layoutRef}`, detail: 'high' },
    })
    userContent.push({
      type: 'text',
      text: '(Above: LAYOUT reference — room concept from Cozy theme. Use for SPATIAL LAYOUT only: window position, desk position, wall proportions. Yellow/blue frames are annotations — ignore those colors.)',
    })
  }

  if (styleRef) {
    const mime = getMimeType('Vorlage V1-Cozy.webp')
    userContent.push({
      type: 'image_url',
      image_url: { url: `data:${mime};base64,${styleRef}`, detail: 'high' },
    })
    userContent.push({
      type: 'text',
      text: '(Above: STYLE reference — target art QUALITY and rendering approach. Adapt colors/mood to the theme description, do NOT copy Cozy colors.)',
    })
  }

  // Add theme-specific reference if available
  if (theme.extraRefImage) {
    const extraRef = loadImageBase64(path.join(DESIGN_REF_DIR, theme.extraRefImage))
    if (extraRef) {
      const mime = getMimeType(theme.extraRefImage)
      userContent.push({
        type: 'image_url',
        image_url: { url: `data:${mime};base64,${extraRef}`, detail: 'high' },
      })
      userContent.push({
        type: 'text',
        text: theme.extraRefCaption ?? '(Above: Additional theme-specific reference.)',
      })
    }
  }

  const response = await client.chat.completions.create({
    model: 'gpt-5.2',
    messages: [
      { role: 'system', content: buildArtDirectorSystem(theme) },
      { role: 'user', content: userContent },
    ],
    max_completion_tokens: 2000,
    temperature: 0.7,
  })

  const prompt = response.choices[0]?.message?.content?.trim() ?? ''
  console.log(`Art Director prompt (${prompt.length} chars):`)
  console.log('---')
  console.log(prompt)
  console.log('---')
  return prompt
}

// ── Step 2: Generate image ───────────────────────────────────────

async function generateImage(prompt: string): Promise<Buffer> {
  console.log('\n[Step 2] Generating image (GPT-Image-1.5, 1536x1024)...')
  console.log('This may take 30-60 seconds...')

  const response = await client.images.generate({
    model: 'gpt-image-1.5',
    prompt,
    n: 1,
    size: '1536x1024',
    quality: 'high',
  })

  const imageData = response.data?.[0]
  if (!imageData) throw new Error('No image data received')

  if ('b64_json' in imageData && imageData.b64_json) {
    return Buffer.from(imageData.b64_json, 'base64')
  }
  if ('url' in imageData && imageData.url) {
    console.log('Downloading from URL...')
    const res = await fetch(imageData.url)
    return Buffer.from(await res.arrayBuffer())
  }
  throw new Error('No usable image data in response')
}

// ── Step 3: Vision QA review ─────────────────────────────────────

async function reviewImage(imagePath: string, originalPrompt: string, themeName: string): Promise<void> {
  console.log(`\n[Step 3] Art Director reviewing "${themeName}" result...`)

  const base64 = fs.readFileSync(imagePath).toString('base64')

  const response = await client.chat.completions.create({
    model: 'gpt-5.2',
    messages: [
      {
        role: 'system',
        content: 'You are reviewing a generated room scene for a CRM application. Check if it matches the requirements. Be concise.',
      },
      {
        role: 'user',
        content: [
          {
            type: 'text',
            text: `Review this "${themeName}" room scene image. The prompt was:\n\n"${originalPrompt.slice(0, 500)}..."\n\nCheck:\n1. Is there a large window in the center (75-80% width)?\n2. Is the view through the window visible and theme-appropriate?\n3. Is the art style soft/illustrated (not photorealistic)?\n4. Are the walls completely empty (no shelves, no art)?\n5. Is the desk completely empty (no objects)?\n6. Overall quality score (1-10)?\n\nRespond as JSON: { "score": N, "passed": bool, "issues": ["..."], "feedback": "..." }`,
          },
          {
            type: 'image_url',
            image_url: { url: `data:image/png;base64,${base64}`, detail: 'high' },
          },
        ],
      },
    ],
    max_completion_tokens: 500,
  })

  console.log('Review:', response.choices[0]?.message?.content ?? '')
}

// ── Generate one theme ───────────────────────────────────────────

async function generateTheme(theme: ThemeConfig): Promise<void> {
  console.log(`\n${'='.repeat(50)}`)
  console.log(`  Generating: ${theme.name}`)
  console.log(`${'='.repeat(50)}`)

  const optimizedPrompt = await getArtDirectorPrompt(theme)

  if (dryRun) {
    console.log(`\n[DRY RUN] Skipping image generation for ${theme.name}.`)
    return
  }

  const outDir = path.join(ASSETS_DIR, theme.id)
  const outFile = path.join(outDir, 'room-scene-light.png')

  fs.mkdirSync(outDir, { recursive: true })
  const imageBuffer = await generateImage(optimizedPrompt)
  fs.writeFileSync(outFile, imageBuffer)
  const sizeKB = Math.round(imageBuffer.length / 1024)
  console.log(`\nSaved: ${outFile} (${sizeKB} KB)`)

  if (!skipReview) {
    await reviewImage(outFile, optimizedPrompt, theme.name)
  }
}

// ── Main ─────────────────────────────────────────────────────────

async function main() {
  const themeArg = getThemeArg()

  if (!generateAll && !themeArg) {
    console.log('Usage:')
    console.log('  --theme <name>   Generate one theme (cozy, dreamy, nature, raumstation, atelier)')
    console.log('  --all            Generate all 5 themes')
    console.log('  --dry-run        Only show Art Director prompts')
    console.log('  --skip-review    Skip Vision QA review')
    process.exit(0)
  }

  const themesToGen = generateAll
    ? THEMES
    : THEMES.filter((t) => t.id === themeArg)

  if (themesToGen.length === 0) {
    console.error(`Unknown theme: "${themeArg}". Available: ${THEMES.map((t) => t.id).join(', ')}`)
    process.exit(1)
  }

  console.log(`\nGenerating ${themesToGen.length} room scene(s): ${themesToGen.map((t) => t.id).join(', ')}`)

  for (const theme of themesToGen) {
    await generateTheme(theme)
  }

  console.log(`\n${'='.repeat(50)}`)
  console.log('  ALL DONE!')
  console.log(`${'='.repeat(50)}`)
}

main().catch((err) => {
  console.error('Error:', err.message ?? err)
  process.exit(1)
})
