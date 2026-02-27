/**
 * Art Director Consultation — One-off script.
 *
 * Sends all 5 current room scene images + a reference image to GPT-5.2
 * for a detailed critique and prompt-improvement consultation.
 *
 * Usage:
 *   npx tsx scripts/asset-gen/consult-art-director.ts
 *   npx tsx scripts/asset-gen/consult-art-director.ts --ref "Vorlage V1-Cozy.webp"
 *   npx tsx scripts/asset-gen/consult-art-director.ts --save            (save response to file)
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
  console.error('Create desktop/.env with: OPENAI_API_KEY=sk-...')
  process.exit(1)
}
const client = new OpenAI({ apiKey })

const saveToFile = process.argv.includes('--save')

function getRefArg(): string | null {
  const idx = process.argv.indexOf('--ref')
  return idx !== -1 ? process.argv[idx + 1] : null
}

// ── Theme definitions (matching test-room-scene.ts) ─────────────

const THEME_IDS = ['cozy', 'dreamy', 'nature', 'raumstation', 'atelier'] as const
const THEME_LABELS: Record<string, string> = {
  cozy: 'Cozy (Gemuetlich) — Scandinavian Home Office',
  dreamy: 'Dreamy (Vertraeumt) — Enchanted Tower Study',
  nature: 'Nature (Natur) — Forest Ranger Cabin',
  raumstation: 'Raumstation — Space Station Observatory',
  atelier: 'Atelier — Montmartre Artist Loft',
}

// ── Helpers ──────────────────────────────────────────────────────

function loadImageBase64(filePath: string): string | null {
  if (!fs.existsSync(filePath)) {
    console.warn(`  [SKIP] Image not found: ${filePath}`)
    return null
  }
  const sizeKB = Math.round(fs.statSync(filePath).size / 1024)
  console.log(`  [OK]   ${path.basename(filePath)} (${sizeKB} KB)`)
  return fs.readFileSync(filePath).toString('base64')
}

function getMimeType(filePath: string): string {
  if (filePath.endsWith('.webp')) return 'image/webp'
  if (filePath.endsWith('.png')) return 'image/png'
  if (filePath.endsWith('.jpg') || filePath.endsWith('.jpeg')) return 'image/jpeg'
  return 'image/png'
}

// ── Consultation System Prompt ──────────────────────────────────

const SYSTEM_PROMPT = `You are a senior Art Director and technical pipeline architect with deep expertise in:
- AI image generation pipelines (GPT-Image-1, GPT-Image-1.5, images.edit API, images.generate API)
- Digital illustration and concept art (AAA game environments, Makoto Shinkai-style backgrounds)
- Image compositing, layer extraction, and sprite sheet workflows
- UI/UX background art for premium desktop software products

## Context
You are consulting for KMU Hub, a CRM desktop application that uses a "desk workspace" metaphor. The user sees an illustrated room with a large window — the app UI renders inside the window area via CSS overlay. Everything around the window (walls, desk surface, shelves, decorations) creates immersive atmosphere.

There are 5 themes: Cozy, Dreamy, Nature, Raumstation, Atelier. Each needs a room scene background.

## Technical Constraints
- **images.edit** (gpt-image-1): Takes a reference image + prompt. Can modify/restyle. input_fidelity='high' available. Max 1536x1024. No transparent output.
- **images.generate** (gpt-image-1.5): Text-only prompt, no reference image input. Can generate transparent PNGs (background='transparent'). Max 1536x1024.
- The app composites layers: L1=room scene background, L2=furniture overlays, L3=decoration overlays, all via CSS absolute positioning.
- The CSS "window" area (where app UI sits) is positioned via percentage-based insets on top of the room scene image.

Be direct, specific, and actionable. Think like an engineer, not just an artist. We need a PRACTICAL pipeline that minimizes API calls and maximizes visual quality.`

// ── Main ────────────────────────────────────────────────────────

async function main() {
  console.log('='.repeat(60))
  console.log('  ART DIRECTOR CONSULTATION')
  console.log('  Sending 5 room scenes + reference to GPT-5.2')
  console.log('='.repeat(60))

  // ── Load reference image ──
  const refFileName = getRefArg() ?? 'Vorlage V1-Cozy.webp'
  const refPath = path.join(DESIGN_REF_DIR, refFileName)
  console.log(`\nLoading reference image:`)
  const refBase64 = loadImageBase64(refPath)
  if (!refBase64) {
    console.error(`\nFATAL: Reference image not found: ${refPath}`)
    console.error('Available files in design-reference/:')
    fs.readdirSync(DESIGN_REF_DIR)
      .filter((f) => /\.(png|jpg|jpeg|webp)$/i.test(f))
      .forEach((f) => console.error(`  - ${f}`))
    process.exit(1)
  }
  const refMime = getMimeType(refPath)

  // ── Load all 5 room scenes ──
  console.log(`\nLoading room scene images:`)
  const roomScenes: { id: string; base64: string }[] = []

  for (const themeId of THEME_IDS) {
    const scenePath = path.join(ASSETS_DIR, themeId, 'room-scene-light.png')
    const base64 = loadImageBase64(scenePath)
    if (base64) {
      roomScenes.push({ id: themeId, base64 })
    }
  }

  if (roomScenes.length === 0) {
    console.error('\nFATAL: No room scene images found!')
    console.error(`Looked in: ${ASSETS_DIR}/{theme}/room-scene-light.png`)
    process.exit(1)
  }

  console.log(`\nLoaded ${roomScenes.length}/5 room scenes.`)

  // ── Build the message content ──
  const userContent: OpenAI.ChatCompletionContentPart[] = []

  // Reference image first
  userContent.push({
    type: 'text',
    text: `## REFERENCE IMAGE (Quality/Style Target)\nThis is our quality target — "Vorlage" (template). The room scenes should aspire to this level of detail, atmosphere, and material rendering quality. Note the window proportions, frame thickness, and overall composition.`,
  })
  userContent.push({
    type: 'image_url',
    image_url: { url: `data:${refMime};base64,${refBase64}`, detail: 'high' },
  })

  // Each room scene
  for (const scene of roomScenes) {
    const label = THEME_LABELS[scene.id] ?? scene.id
    userContent.push({
      type: 'text',
      text: `\n## CURRENT IMAGE: ${label}`,
    })
    userContent.push({
      type: 'image_url',
      image_url: { url: `data:image/png;base64,${scene.base64}`, detail: 'high' },
    })
  }

  // The consultation request
  userContent.push({
    type: 'text',
    text: `## CONSULTATION REQUEST — PIPELINE ARCHITECTURE

We need your help redesigning our image generation pipeline. Here's the situation:

### CURRENT APPROACH (problematic)
1. Take reference image (V3, the one with furniture/plants/objects)
2. Step 1: Strip all objects → clean base (images.edit, gpt-image-1)
3. Step 2: Transform clean base into each theme (images.edit, gpt-image-1)
4. Step 3: Generate furniture overlays as transparent PNGs (images.generate, gpt-image-1.5)
5. Step 4: Generate decoration objects as transparent PNGs (images.generate, gpt-image-1.5)

**Problems:** Window position shifts at each step despite composition-lock prompts. After 2 transforms the window can be 5-10% off. The CSS overlay that positions the app UI inside the window no longer aligns. Also 4 steps = 4x composition drift + expensive.

### PROPOSED NEW APPROACH
Instead of stripping objects first, keep furniture/shelves from the reference and just restyle:

1. **Theme transform WITH furniture**: Take V3 reference (has shelves, plants, desk items) → "Restyle to [theme], keep all furniture in place, change materials/lighting/view" (1 API call per theme via images.edit)
2. **Theme transform WITHOUT small deco**: Same but "remove small decorative items (plants, cups, books) but keep structural furniture (shelves, desk)" (1 API call per theme)
3. **Extract deco elements**: Diff the "with deco" vs "without deco" images to isolate decoration items as separate transparent PNGs
4. **User toggles**: In the app, users can toggle deco groups on/off, or choose between different deco arrangements per mount point

### SPECIFIC QUESTIONS FOR YOU

1. **Feasibility of the diff approach**: Can we reliably extract decoration elements by pixel-diffing "with deco" vs "without deco" images from GPT-Image-1? The images won't be pixel-identical — how much drift should we expect? What's the best algorithm (simple alpha threshold on diff, color-distance threshold, edge detection, something else)?

2. **Better alternative for deco extraction?** Would it be better to:
   a. Diff two generated images (with/without deco)
   b. Generate deco objects separately on transparent BG (images.generate with background='transparent')
   c. Use images.edit with a mask to inpaint specific regions
   d. Something else entirely?

3. **Minimizing window shift**: The V3 reference has very clear furniture anchoring the composition. If we do a single images.edit transform (reference → themed) instead of two steps (reference → clean → themed), will the window position be more stable? What prompt techniques best preserve spatial layout in images.edit?

4. **Optimal pipeline**: Given our constraints (5 themes, light+dark mode each, toggleable decorations, CSS window overlay must align), what's the most efficient pipeline? Minimize API calls while maximizing quality and composition stability.

5. **Per-theme CSS adjustment**: If the window shifts slightly per theme, should we:
   a. Manually measure window position in each generated image and set per-theme CSS insets
   b. Use a script to auto-detect the window rectangle (edge detection on the generated image)
   c. Use a fixed mask/overlay approach so the window is ALWAYS in the exact same spot
   d. Something else?

6. **Dark mode strategy**: For dark mode variants, should we:
   a. Generate separate dark versions (doubles API calls)
   b. Apply CSS filters (brightness, hue-rotate) to the light versions
   c. Use images.edit to darken an existing light theme image
   d. Something else?

### LOOKING AT THE IMAGES
- The REFERENCE image has the best composition — furniture anchors, clear window, good proportions
- The CURRENT 5 themes were generated with the old strip-then-restyle approach
- Notice how the window shifted and frames got thick/chunky in Cozy, Nature, Atelier
- Raumstation and Dreamy held up better but still drifted

Please give us a concrete, actionable pipeline recommendation. Think engineer, not just artist. We want minimum API calls, maximum composition stability, and toggleable decorations.`,
  })

  // ── Send to GPT-5.2 ──
  console.log('\nSending to GPT-5.2 for consultation...')
  console.log('(This may take 30-90 seconds due to multiple images)\n')

  const startTime = Date.now()

  const response = await client.chat.completions.create({
    model: 'gpt-5.2',
    messages: [
      { role: 'system', content: SYSTEM_PROMPT },
      { role: 'user', content: userContent },
    ],
    max_completion_tokens: 8000,
    temperature: 0.5,
  })

  const elapsed = Math.round((Date.now() - startTime) / 1000)
  const reply = response.choices[0]?.message?.content?.trim() ?? '(empty response)'
  const usage = response.usage

  // ── Output ──
  console.log('='.repeat(60))
  console.log('  ART DIRECTOR CONSULTATION RESPONSE')
  console.log('='.repeat(60))
  console.log()
  console.log(reply)
  console.log()
  console.log('='.repeat(60))
  console.log(`  Time: ${elapsed}s`)
  if (usage) {
    console.log(`  Tokens: ${usage.prompt_tokens} prompt + ${usage.completion_tokens} completion = ${usage.total_tokens} total`)
  }
  console.log('='.repeat(60))

  // ── Optionally save to file ──
  if (saveToFile) {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
    const outPath = path.resolve(__dirname, `../../consultation-${timestamp}.md`)
    const fileContent = [
      `# Art Director Consultation — ${new Date().toISOString().slice(0, 10)}`,
      '',
      `**Model:** GPT-5.2`,
      `**Reference:** ${refFileName}`,
      `**Room Scenes:** ${roomScenes.map((s) => s.id).join(', ')}`,
      `**Tokens:** ${usage?.total_tokens ?? 'unknown'}`,
      `**Time:** ${elapsed}s`,
      '',
      '---',
      '',
      reply,
    ].join('\n')

    fs.writeFileSync(outPath, fileContent, 'utf-8')
    console.log(`\nResponse saved to: ${outPath}`)
  }
}

main().catch((err) => {
  console.error('Error:', err.message ?? err)
  if (err.status) console.error(`HTTP Status: ${err.status}`)
  if (err.code) console.error(`Error Code: ${err.code}`)
  process.exit(1)
})
