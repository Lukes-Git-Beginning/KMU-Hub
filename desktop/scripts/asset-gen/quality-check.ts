/**
 * Quality Check — prueft generierte Assets via GPT Vision.
 * Schickt das Bild + Style DNA an GPT-4o und bekommt Feedback.
 */
import OpenAI from 'openai'
import * as fs from 'fs'
import { STYLE_DNA } from './style-dna'
import type { AssetDefinition } from './assets'

interface QualityResult {
  passed: boolean
  score: number // 1-10
  feedback: string
  improvements: string[]
}

export async function checkAssetQuality(
  client: OpenAI,
  imagePath: string,
  asset: AssetDefinition
): Promise<QualityResult> {
  const imageBuffer = fs.readFileSync(imagePath)
  const base64Image = imageBuffer.toString('base64')

  const styleDescription = [
    ...STYLE_DNA.baseStyle,
    ...(asset.theme === 'cozy' ? STYLE_DNA.cozyStyle : STYLE_DNA.dreamyStyle),
  ].join('. ')

  const response = await client.chat.completions.create({
    model: 'gpt-4o',
    messages: [
      {
        role: 'system',
        content:
          'You are a design quality checker for a SaaS application. You evaluate generated assets against a style guide. Be strict but fair. Respond in JSON only.',
      },
      {
        role: 'user',
        content: [
          {
            type: 'text',
            text: `Check this asset against our style guide.

Asset: "${asset.name}" (${asset.theme} theme)
Original prompt: "${asset.prompt}"

Style guide:
${styleDescription}

Color rules: Primary ${STYLE_DNA.colors.primary}, accents ${STYLE_DNA.colors.accents}. Avoid: ${STYLE_DNA.colors.avoid}.

Rate 1-10 and provide feedback. Respond as JSON:
{
  "score": <number 1-10>,
  "passed": <true if score >= 7>,
  "feedback": "<one sentence summary>",
  "improvements": ["<suggestion 1>", "<suggestion 2>"]
}`,
          },
          {
            type: 'image_url',
            image_url: {
              url: `data:image/png;base64,${base64Image}`,
            },
          },
        ],
      },
    ],
    max_tokens: 300,
  })

  const content = response.choices[0]?.message?.content ?? '{}'

  try {
    // Extract JSON from response (might be wrapped in markdown code block)
    const jsonMatch = content.match(/\{[\s\S]*\}/)
    if (!jsonMatch) {
      return { passed: false, score: 0, feedback: 'Could not parse response', improvements: [] }
    }
    return JSON.parse(jsonMatch[0]) as QualityResult
  } catch {
    return {
      passed: false,
      score: 0,
      feedback: `Raw response: ${content}`,
      improvements: ['Could not parse quality check response'],
    }
  }
}
