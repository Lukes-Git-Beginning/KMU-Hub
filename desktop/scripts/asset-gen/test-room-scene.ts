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
    wallMaterial: 'warm off-white Scandinavian plaster wall, subtle trowel marks, soft mottling, clean matte finish with gentle light gradients',
    deskMaterial: 'light ash/oak desktop with fine grain fibers, satin varnish sheen, subtle edge wear, visible wood pores',
    windowFrame: 'light oak slim frame or painted white slim frame, minimal modern profile, thin bezel',
    brief: `THEME: Cozy (Gemuetlich) — A Sunlit Scandinavian Home Office

WORLD FANTASY: You are working in a beautiful Scandinavian-inspired home office. Afternoon sunlight streams in through the window, casting warm golden light across natural materials. Everything feels handcrafted, warm, and lived-in — like a Danish architect's personal workspace.

MATERIALS & TEXTURES (render with rich detail):
- Walls: Cream/warm white plaster with subtle organic texture variations — not flat paint, but plaster with depth and warmth. Soft sunlight creates gentle light/shadow gradients across the wall surface.
- Desk: Honey oak wood — the grain must be BEAUTIFUL and detailed. Individual wood fibers, subtle knots, warm golden-brown tones. The surface has a soft satin finish with gentle light reflections.
- Window frame: Thin, elegant oak frame. Minimal thickness — just a refined border.

COLOR PALETTE: Warm cream (#e8e3dd), honey oak, soft golden light, with the outdoor view providing greens and sky blue as accent colors.

LIGHTING: Warm afternoon sunlight from the window, creating a gentle golden glow on the desk surface and soft light gradients on the walls. Subtle volumetric light atmosphere.

WINDOW VIEW: Idyllic garden landscape — lush green rolling hills with wildflowers (subtle poppies, daisies), mature trees with detailed foliage, gentle blue sky with soft white clouds. The view should feel like countryside in Tuscany or southern Germany. LUSH, detailed, inviting.

MOOD: Hygge, warmth, calm productivity. "I never want to leave this desk."`,
  },
  {
    id: 'dreamy',
    name: 'Dreamy (Vertraeumt)',
    wallMaterial: 'pale lavender-tinted plaster over stone, subtle magical shimmer (very subtle), fine micro-texture, gentle mottling',
    deskMaterial: 'smooth pale stone sill/ledge (limestone) with faint speckle and soft polished sheen — enchanted tower ledge, not ordinary wood',
    windowFrame: 'thin brushed silver or pale stone edge trim, minimal profile, slightly rounded corners — architectural tower opening, not picture frame',
    extraRefImage: 'Farben Vibe inspo - Dreamy.webp',
    extraRefCaption: '(Above: COLOR/VIBE reference for Dreamy theme — lavender/purple palette, floating orbs, ethereal golden leaves, magical atmosphere. Use this for the color direction and magical mood.)',
    brief: `THEME: Dreamy (Vertraeumt) — An Enchanted Tower Study

WORLD FANTASY: You are working in a study at the top of an enchanted tower. This is where an elven scholar or a gentle wizard would work. Everything has a soft magical quality — surfaces shimmer faintly, the light has an ethereal purple-gold quality, and the world outside the window is a magical landscape. This is NOT just a purple room — it is a MAGICAL workspace.

MATERIALS & TEXTURES (render with rich detail):
- Walls: Soft lavender plaster that seems to have a faint inner luminescence — like moonstone or opal. Subtle shimmer particles catch the light. The texture has depth and a dreamy, almost liquid quality.
- Desk: Pale silvery-white birch wood with an opalescent sheen. The wood grain has a pearlescent quality, as if enchanted. Smooth, luminous, otherworldly.
- Window frame: Thin, elegant, silver-white — almost like it's made of crystallized light. Subtle inner glow along the edges.

COLOR PALETTE: Lavender, soft lilac, pearl white, touches of rose gold and warm amber. The overall tone is cool purple with warm golden accents from the magical light.

LIGHTING: Soft ethereal twilight glow — the light comes from the magical sky outside and seems to have a warm purple-gold quality. Surfaces catch this light with subtle iridescent reflections. Gentle, dreamy, never harsh.

WINDOW VIEW: Magical twilight landscape — a vast ethereal sky in purple and rose-gold gradients, luminous clouds, distant crystal formations or floating islands catching golden light, aurora-like ribbons of light across the sky, tiny floating particles of light (like fireflies or magic dust). The landscape below shows mystical terrain — maybe a silver lake reflecting the sky, enchanted forests with glowing canopy.

MOOD: Wonder, serenity, magic. "I'm working in a fairytale." The image must feel SHARP and richly detailed despite the dreamy mood — dreamy means magical atmosphere, NOT blurry/out-of-focus.`,
  },
  {
    id: 'nature',
    name: 'Nature (Natur)',
    wallMaterial: 'warm cabin interior: vertical timber boards or lightly weathered pine planks, visible seams, knots, subtle wear — refined forest cabin',
    deskMaterial: 'medium-toned rustic wood ledge with visible grain, slight dents, matte-oil finish, warm amber highlights where light catches',
    windowFrame: 'thin dark-stained wood frame, slim profile, clean rectangular opening — handcrafted cabin joinery',
    brief: `THEME: Nature (Natur) — A Forest Ranger's Cabin Study

WORLD FANTASY: You are working in a beautifully crafted cabin deep in an ancient forest. This is the study of a forest ranger or nature scholar — refined but deeply connected to the surrounding woods. Through the window, an old-growth forest glows with filtered sunlight. The cabin materials are natural, rich, and grounding.

MATERIALS & TEXTURES (render with rich detail):
- Walls: Natural stone with warm earth tones — think refined cabin interior. Subtle texture variations, tiny moss traces where wall meets ceiling or at edges. The stone has depth, individual stones or texture patterns visible. Warm amber/brown undertones.
- Desk: Dark walnut wood — RICH, deep grain patterns with amber highlights where light catches the surface. The wood has character: subtle variations in tone, visible growth rings, a polished but natural finish.
- Window frame: Dark stained wood, thin and elegant — handcrafted quality, like fine cabin joinery.

COLOR PALETTE: Deep forest greens, warm walnut browns, amber/golden light tones, moss accents. Rich and earthy without being dark or muddy.

LIGHTING: Warm amber-golden light filtering through the forest canopy outside, creating beautiful dappled light patterns. The light enters through the window and creates a warm glow on the desk surface. Volumetric light rays visible in the forest view.

WINDOW VIEW: Ancient deciduous forest in golden hour — massive old trees with detailed bark and lush canopy, sunlight filtering through leaves creating god-rays, forest floor with ferns and moss, a gentle woodland path disappearing into misty depth. Maybe a small stream visible. The forest should feel OLD, LUSH, and MAGICAL in a natural way.

MOOD: Grounded, peaceful, connected to nature. "Working in perfect harmony with the forest."`,
  },
  {
    id: 'raumstation',
    name: 'Raumstation (Space Station)',
    wallMaterial: 'sleek sci-fi wall panels, matte graphite composite with subtle seams and tiny fasteners, micro-scratches, soft ambient occlusion in panel gaps',
    deskMaterial: 'dark composite ledge with fine texture, slight satin sheen, clean edges — advanced material, subtle blue LED reflection',
    windowFrame: 'thin titanium/aluminum viewport bezel, single minimal border, subtle inner glow reflection — NOT nested triple-border',
    brief: `THEME: Raumstation (Space Station) — An Orbital Observatory Deck

WORLD FANTASY: You are working at a personal workstation on a luxury space station's observation deck. This is NOT military/aggressive sci-fi — think premium civilian space station, like a research institute in orbit. The interior is sleek, clean, and beautifully engineered. Through the viewport, the majesty of deep space unfolds.

MATERIALS & TEXTURES (render with rich detail):
- Walls: Spacecraft interior panels — dark gunmetal gray with precision panel lines/seams visible. Subtle blue-white LED accent strips along panel edges create soft ambient glow. The panels have a machined metal quality — brushed texture, subtle reflections. Think premium spacecraft interior design.
- Desk: Dark matte composite — carbon fiber or advanced material with ultra-smooth finish. Subtle reflections of the blue ambient lighting on the surface. The material looks advanced and expensive.
- Window frame: Thin titanium/steel bezel with subtle blue-white luminous edge line — like a high-tech viewport frame.

COLOR PALETTE: Dark gunmetal gray, deep navy, brushed steel silver, with blue-white and cyan LED accents providing the color highlights. The space view adds purple nebula colors and warm star light.

LIGHTING: Cool blue-white ambient LED lighting from panel edges. The deep space view provides additional subtle illumination — nebula colors reflecting faintly on the desk surface. Clean, cool, but not cold — premium and calming.

WINDOW VIEW: Deep space panorama — a stunning vista of stars, a colorful nebula (soft purple, blue, teal gradients), maybe a planet or moon partially visible at the edge with atmospheric glow, distant cosmic dust clouds lit by starlight. The view should inspire AWE and serenity. Stars should be crisp points of light, the nebula should have beautiful color depth.

MOOD: Awe, calm vastness, premium technology. "Working at the edge of the cosmos." NOT Star Wars/aggressive — more Interstellar/The Expanse luxury aesthetic.`,
  },
  {
    id: 'atelier',
    name: 'Atelier (Kuenstler-Werkstatt)',
    wallMaterial: 'aged Paris loft brick wall, varied bricks with chips and mortar irregularity, warm sun grazing highlights, realistic depth and patina',
    deskMaterial: 'warm worn wooden ledge with visible grain, slight dustiness, soft sheen — reclaimed timber character',
    windowFrame: 'thin black metal window frame, minimal modern profile — industrial loft architectural opening',
    brief: `THEME: Atelier (Kuenstler-Werkstatt) — A Montmartre Artist's Loft

WORLD FANTASY: You are working in a beautiful artist's loft in an old European building — think top floor of a Parisian or Viennese building from the 1890s, converted into a creative workspace. Warm exposed brick, large industrial windows, and through them a stunning view of historic rooftops bathed in golden sunset light.

MATERIALS & TEXTURES (render with rich detail):
- Walls: Exposed brick — each brick individually visible with warm terracotta/salmon tones, white mortar lines, subtle age variations. The brick has character: slightly different colors between bricks, some darker, some lighter. Century-old patina. Rich, warm, characterful.
- Desk: Thick reclaimed wood plank — beautiful aged patina, warm honey-brown tones. The wood grain is prominent and detailed, with the character of reclaimed timber. Smooth from years of use, warm under the light.
- Window frame: Thin black steel, industrial loft style — elegant and minimal profile. The frame should feel architectural and refined.

COLOR PALETTE: Warm terracotta/brick, honey wood, black steel accents, golden sunset light. The view adds warm oranges, slate roof grays, and sky gradients.

LIGHTING: Warm golden hour sunlight streaming through the window, casting beautiful warm light across the brick wall and desk. The light creates rich shadows and warm highlights on the textured surfaces. This is the most dramatically lit theme — sunset atmosphere.

WINDOW VIEW: European city rooftops at golden hour — terracotta and slate rooftops receding into the distance, ornate chimneys, a church spire or dome in the mid-distance, warm sunset sky with orange/pink/gold gradients, a few birds in flight. The view should feel like looking out over Paris, Vienna, or Prague from an attic studio. Romantic, warm, inspiring.

MOOD: Creative, bohemian, inspiring, romantic. "I could paint masterpieces at this desk."`,
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
The app uses a "desk workspace" metaphor. The user sees a beautifully illustrated ROOM with a LARGE WINDOW. A software UI is displayed INSIDE the window area (via CSS overlay). Everything around the window — walls, desk surface, window frame — creates immersive atmosphere.

## 5-Layer Architecture
- Layer 1 (YOUR JOB): ROOM SCENE — the full illustrated room background (wall + window + view + desk surface)
- Layer 2–5: Furniture, decorations, UI — added separately, NOT your concern

## Your Task
Write a DETAILED image generation prompt that produces a premium-quality room scene. You receive reference images for layout and style quality.

## EXACT COMPOSITION (MANDATORY — every theme MUST match this layout)
These proportions are non-negotiable. Every theme must produce images with identical spatial layout:

- CAMERA: Frontal view, straight at the wall. Slight distance (1.5m back from desk) — natural working perspective, NOT pressed against desk.
- IMAGE SIZE: 1536x1024 pixels (landscape).
- EXACT PIXEL LAYOUT (non-negotiable):
  - WALL OPENING (not a framed window!): left edge at x=184px (12% from left), right edge at x=1352px (12% from right), top edge at y=82px (8% from top), bottom edge at y=799px (78% from top).
  - This means: wall opening is 1168px wide × 717px tall, centered horizontally.
  - LEFT WALL STRIP: 0–184px (narrow, ~12% of width)
  - RIGHT WALL STRIP: 1352–1536px (narrow, ~12% of width)
  - CEILING STRIP: 0–82px (thin, ~8% of height)
  - DESK SURFACE: 799–1024px (bottom 22% of image, ~225px tall)
- The opening in the wall is NOT a window with a frame. It is a CLEAN RECTANGULAR OPENING cut directly into the wall. Think of it as an architectural wall cutout — the wall simply ends and the view begins. There may be a very thin edge/reveal where the wall meets the opening (2-4px, just ambient occlusion/shadow), but NO decorative frame, NO wooden frame, NO picture frame, NO molding, NO border. The transition from wall to view should be almost seamless — just a subtle shadow line.
- WINDOW INTERIOR: ONE clean uninterrupted rectangle showing the view. NO muntins, NO crossbars, NO grid, NO divided panes.
- WALLS: COMPLETELY EMPTY — no shelves, no artwork, no decorations, no hooks. Clean uninterrupted surface with rich micro-texture. (Decorations overlaid separately.)
- DESK: COMPLETELY EMPTY flat surface — no objects, no ledges, no raised edges, no rear board. Clean stage, no props. Just bare surface material extending to the wall. (Objects added separately.)

## Theme-Specific Materials
- Wall: ${theme.wallMaterial}
- Desk: ${theme.deskMaterial}
- Window frame: ${theme.windowFrame}

## Art Style (CRITICAL — this defines the quality bar)
Premium illustrated interior background, cinematic background art quality:

- Style: Makoto Shinkai background art meets modern interior design visualization
- SHARP, DETAILED — material study quality: visible wood pores/fibers, plaster trowel marks, micro-texture variations, subtle edge wear, contact shadows, ambient occlusion
- Lighting is KEY — soft global illumination, gentle bloom, motivated light direction, color bounce from outside view onto interior surfaces, atmospheric perspective
- NOT photorealistic — illustrated/artistic but with AAA game concept painting detail
- NOT flat, NOT blurry, NOT cheap. Every surface has depth, variation, specular response

## Theme Identity (IMPORTANT)
Each theme is a COMPLETELY DIFFERENT WORLD, not a "color swap":
- Since walls/desk must be empty, immersion comes from: material language, lighting color + bounce, architectural detailing (frame reveal, sill material)
- Commit fully to the theme fantasy

## About the Reference Images
- LAYOUT reference: spatial arrangement only — IGNORE colors, frame thickness, style quality
- STYLE reference: art quality baseline — aim HIGHER in detail and richness
- Adapt everything to the CURRENT THEME — never copy Cozy materials for other themes

## OUTPUT RULES (CRITICAL)
Your output prompt MUST follow this exact structure and order:
1. **Composition + layout** (hard constraints — window size, desk height, thin frame)
2. **Materials** (wall, frame, desk with micro-detail keywords)
3. **Lighting & atmosphere** (bounce light, bloom, volumetric where appropriate)
4. **Outside view** (foreground/midground/background depth, atmospheric perspective)
5. **Style anchor** ("premium illustrated interior background, cinematic background art, Makoto Shinkai atmosphere")
6. **DO NOT block** (explicit negatives — MUST include all of these):
   - "no furniture, no chair, no lamp, no shelves, no frames, no posters, no plants, no books, no objects, no props"
   - "no window muntins, no crossbars, no grid, no divided panes, no French window, no sash window"
   - "no text, no logos, no UI, no people"
   - "not a picture frame, not a framed artwork, not ornate molding, not chunky frame"
   - "no curtains, no blinds, no extra windows"

Output ONLY the prompt. No explanations, no markdown. Target length: **150-220 words** (concise but specific — too long and the model averages constraints).`
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
