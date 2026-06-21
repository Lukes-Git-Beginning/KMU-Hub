/**
 * Shared lowlight instance for the wiki code blocks (WP-2).
 *
 * The TipTap editor highlights code live via CodeBlockLowlight's ProseMirror
 * decorations, but those decorations are NOT part of the serialised HTML — the
 * stored document only carries `<pre><code class="language-x">raw</code></pre>`.
 * So the read view (which renders stored HTML through DOMPurify) re-highlights
 * each code block here, turning the raw code into `hljs-*` spans the wiki CSS
 * then colours. No extra dependency: a tiny hast→HTML walker does the job.
 */
import { createLowlight, common } from 'lowlight'

export const lowlight = createLowlight(common)

/** The languages we surface in the editor's code-block language picker. */
export const CODE_LANGUAGES: { value: string; label: string }[] = [
  { value: 'plaintext', label: 'Text' },
  { value: 'typescript', label: 'TypeScript' },
  { value: 'javascript', label: 'JavaScript' },
  { value: 'json', label: 'JSON' },
  { value: 'bash', label: 'Bash' },
  { value: 'python', label: 'Python' },
  { value: 'go', label: 'Go' },
  { value: 'sql', label: 'SQL' },
  { value: 'css', label: 'CSS' },
  { value: 'xml', label: 'HTML / XML' },
  { value: 'yaml', label: 'YAML' },
  { value: 'markdown', label: 'Markdown' },
]

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

interface HastNode {
  type: string
  tagName?: string
  value?: string
  properties?: { className?: string[] | string }
  children?: HastNode[]
}

/** Serialise a lowlight hast tree to an HTML string (text + className spans). */
function hastToHtml(node: HastNode): string {
  if (node.type === 'text') return escapeHtml(node.value ?? '')
  const inner = (node.children ?? []).map(hastToHtml).join('')
  if (node.type === 'element' && node.tagName) {
    const cls = node.properties?.className
    const classAttr = Array.isArray(cls)
      ? ` class="${cls.join(' ')}"`
      : typeof cls === 'string'
        ? ` class="${cls}"`
        : ''
    return `<${node.tagName}${classAttr}>${inner}</${node.tagName}>`
  }
  return inner
}

/**
 * Re-highlight every `<pre><code>` block inside an HTML string so the read view
 * shows the same syntax colours as the editor. Unknown languages fall back to
 * auto-detection; failures leave the block as escaped plain text.
 */
export function highlightCodeBlocks(html: string): string {
  if (!html || (!html.includes('<pre') && !html.includes('<code'))) return html
  const doc = new DOMParser().parseFromString(html, 'text/html')
  doc.querySelectorAll('pre code').forEach((code) => {
    const text = code.textContent ?? ''
    if (!text.trim()) return
    const langClass = Array.from(code.classList).find((c) => c.startsWith('language-'))
    const lang = langClass?.replace('language-', '')
    try {
      const tree =
        lang && lang !== 'plaintext' && lowlight.registered(lang)
          ? lowlight.highlight(lang, text)
          : lowlight.highlightAuto(text)
      code.innerHTML = hastToHtml(tree as unknown as HastNode)
      code.classList.add('hljs')
    } catch {
      /* leave the block as-is (browser-escaped plain text) */
    }
  })
  return doc.body.innerHTML
}
