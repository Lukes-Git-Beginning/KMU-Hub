/**
 * 4o Image Generation Comparison
 *
 * Uses GPT-4o via the Responses API to generate a room scene
 * for quality comparison against gpt-image-1 edit pipeline.
 */
import 'dotenv/config'
import OpenAI, { toFile } from 'openai'
import * as fs from 'fs'
import * as path from 'path'

const DESIGN_REF_DIR = path.resolve(__dirname, '../../design-reference')
const PIPELINE_DIR = path.resolve(__dirname, '../../src/renderer/assets/desk/_pipeline')

const apiKey = process.env.OPENAI_API_KEY
if (!apiKey) {
  console.error('ERROR: OPENAI_API_KEY not found in .env')
  process.exit(1)
}
const client = new OpenAI({ apiKey })

async function main() {
  console.log('\n══════════════════════════════════════════════')
  console.log('  4o COMPARISON — Image generation with GPT-4o')
  console.log('══════════════════════════════════════════════\n')

  // Load reference image for context
  const refPath = path.join(DESIGN_REF_DIR, 'Vorlage V1-Cozy.webp')
  if (!fs.existsSync(refPath)) {
    console.error(`Reference not found: ${refPath}`)
    process.exit(1)
  }

  const refBuffer = fs.readFileSync(refPath)
  const base64Ref = refBuffer.toString('base64')
  const mimeType = 'image/webp'

  const prompt = `Look at this reference image of an illustrated room scene with a desk, window, and wall.

I need you to recreate this EXACT scene but with ALL objects removed — no laptop, no phone, no cups, no books, no shelf, no plants. Just the bare room: empty wall, window with the outdoor view, empty desk surface.

CRITICAL requirements:
- Match the EXACT composition: same wall proportions, same window size and position, same desk height
- Wall should be NEUTRAL cool-white to light gray — NOT warm, NOT golden, NOT yellowish
- Desk keeps natural wood warmth
- Same premium illustrated art style (Makoto Shinkai-like)
- Same outdoor landscape view through window
- Same soft lighting but neutral/cool overall tone
- Sharp, richly textured, premium quality

Generate this as a single clean room scene image.`

  console.log('Sending to GPT-4o with reference image...')
  console.log('This may take 30-90 seconds...\n')

  try {
    // Use responses API with image generation
    const response = await (client as any).responses.create({
      model: 'gpt-4o',
      input: [
        {
          role: 'user',
          content: [
            {
              type: 'input_image',
              image_url: `data:${mimeType};base64,${base64Ref}`,
            },
            {
              type: 'input_text',
              text: prompt,
            },
          ],
        },
      ],
      tools: [{ type: 'image_generation', size: '1536x1024', quality: 'high' }],
    })

    // Extract generated image from response
    let imageFound = false
    for (const item of response.output) {
      if (item.type === 'image_generation_call' && item.result) {
        const buffer = Buffer.from(item.result, 'base64')
        fs.mkdirSync(PIPELINE_DIR, { recursive: true })
        const outPath = path.join(PIPELINE_DIR, 'clean-base-4o.png')
        fs.writeFileSync(outPath, buffer)
        const sizeKB = Math.round(buffer.length / 1024)
        console.log(`Saved: ${outPath} (${sizeKB} KB)`)
        imageFound = true
        break
      }
    }

    if (!imageFound) {
      console.log('Response output:', JSON.stringify(response.output, null, 2))
      console.error('No image found in response')
    }
  } catch (err: any) {
    console.error('4o Responses API error:', err.message ?? err)
    console.log('\nFalling back to gpt-image-1.5 generate (text-only, no reference)...\n')

    const textOnlyPrompt = `A premium illustrated room scene viewed from a frontal perspective. The scene shows:

- A large arched window taking up about 60% of the wall, centered, with thin white frame
- Through the window: a lush countryside landscape with rolling green hills, a winding river, mature trees, wildflowers, soft blue sky with wispy clouds — illustrated in a Makoto Shinkai style
- Wall: neutral cool-white to light gray plaster, subtle texture, matte finish — NOT warm, NOT golden, NOT yellowish
- Desk surface at the bottom 22% of the image: honey oak wood with beautiful grain detail, satin finish
- Completely empty room — no objects, no decorations, no shelf, nothing on the desk or wall
- Lighting: soft natural daylight, neutral tone, subtle shadows
- Art style: premium illustrated, sharp textures, rich detail, Makoto Shinkai atmosphere
- Aspect ratio matches 1536x1024 exactly`

    console.log('Generating with gpt-image-1.5 (text-only, no reference)...')

    const fallbackResponse = await client.images.generate({
      model: 'gpt-image-1.5',
      prompt: textOnlyPrompt,
      n: 1,
      size: '1536x1024',
      quality: 'high',
    })

    const imageData = fallbackResponse.data?.[0]
    if (!imageData) throw new Error('No image data')

    let buffer: Buffer
    if ('b64_json' in imageData && imageData.b64_json) {
      buffer = Buffer.from(imageData.b64_json, 'base64')
    } else if ('url' in imageData && imageData.url) {
      const res = await fetch(imageData.url)
      buffer = Buffer.from(await res.arrayBuffer())
    } else {
      throw new Error('No usable image data')
    }

    fs.mkdirSync(PIPELINE_DIR, { recursive: true })
    const outPath = path.join(PIPELINE_DIR, 'clean-base-4o.png')
    fs.writeFileSync(outPath, buffer)
    const sizeKB = Math.round(buffer.length / 1024)
    console.log(`Saved: ${outPath} (${sizeKB} KB)`)
  }

  console.log('\n══════════════════════════════════════════════')
  console.log('  DONE!')
  console.log('══════════════════════════════════════════════')
}

main().catch((err) => {
  console.error('Fatal error:', err.message ?? err)
  process.exit(1)
})
