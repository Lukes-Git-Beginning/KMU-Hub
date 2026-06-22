/**
 * Code highlighting for the document code block. Reuses the project's lowlight
 * (the same engine TipTap's code blocks run on) and renders its hast tree to
 * React spans carrying `hljs-*` classes — styled globally by the Cosmi token
 * theme in styles/wiki-content.css, so a highlighted block tracks the active
 * theme in every surface (wiki + berichte) without extra CSS.
 */
import { createLowlight, common } from 'lowlight'
import { Fragment, type ReactNode } from 'react'

const lowlight = createLowlight(common)

type HastNode =
  | { type: 'text'; value: string }
  | { type: 'element'; tagName: string; properties?: { className?: string[] | string }; children: HastNode[] }
  | { type: 'root'; children: HastNode[] }

function renderNodes(nodes: HastNode[], keyPrefix: string): ReactNode[] {
  return nodes.map((node, i) => {
    const key = `${keyPrefix}-${i}`
    if (node.type === 'text') return <Fragment key={key}>{node.value}</Fragment>
    if (node.type === 'element') {
      const cls = node.properties?.className
      const className = Array.isArray(cls) ? cls.join(' ') : cls
      return (
        <span key={key} className={className}>
          {renderNodes(node.children, key)}
        </span>
      )
    }
    return null
  })
}

/**
 * Highlight `code` for `language` as React nodes. Falls back to plain text for
 * empty input or an unknown language, so the block never throws.
 */
export function highlightToReact(code: string, language: string): ReactNode {
  if (!code) return null
  if (language && language !== 'plaintext' && lowlight.registered(language)) {
    try {
      const tree = lowlight.highlight(language, code)
      return renderNodes(tree.children as HastNode[], 'h')
    } catch {
      // Unknown grammar edge case — fall through to plain text.
    }
  }
  return code
}

/** Languages offered in the code-block dropdown (a curated subset of lowlight `common`). */
export const CODE_LANGUAGES: { value: string; label: string }[] = [
  { value: 'plaintext', label: 'Text' },
  { value: 'bash', label: 'Bash' },
  { value: 'javascript', label: 'JavaScript' },
  { value: 'typescript', label: 'TypeScript' },
  { value: 'json', label: 'JSON' },
  { value: 'python', label: 'Python' },
  { value: 'go', label: 'Go' },
  { value: 'sql', label: 'SQL' },
  { value: 'yaml', label: 'YAML' },
  { value: 'xml', label: 'HTML / XML' },
  { value: 'css', label: 'CSS' },
  { value: 'rust', label: 'Rust' },
  { value: 'java', label: 'Java' },
  { value: 'php', label: 'PHP' },
  { value: 'markdown', label: 'Markdown' },
]
