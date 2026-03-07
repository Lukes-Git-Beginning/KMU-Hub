import { useState, useEffect } from 'react'
import { Bold, Italic, Heading2, List, ListOrdered, Link, Code } from 'lucide-react'

// ---------------------------------------------------------------------------
// Toolbar button
// ---------------------------------------------------------------------------

function ToolbarBtn({
  icon: Icon,
  title,
  onClick,
}: {
  icon: typeof Bold
  title: string
  onClick: () => void
}) {
  return (
    <button
      onClick={onClick}
      className="rounded p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
      title={title}
    >
      <Icon className="h-3.5 w-3.5" />
    </button>
  )
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiEditorProps {
  content: string
  onChange: (content: string) => void
  onSave: () => void
  onCancel: () => void
}

export function WikiEditor({ content, onChange, onSave, onCancel }: WikiEditorProps) {
  const [text, setText] = useState(content)

   
  useEffect(() => {
    setText(content)
  }, [content])

  const handleChange = (value: string) => {
    setText(value)
    onChange(value)
  }

  // Simple toolbar actions (insert HTML tags at cursor)
  const insertTag = (before: string, after: string) => {
    const sel = `${before}Text${after}`
    handleChange(text + sel)
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 's' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault()
      onSave()
    }
    if (e.key === 'Escape') {
      onCancel()
    }
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {/* Toolbar */}
      <div className="flex items-center gap-0.5 border-b border-border px-3 py-1.5">
        <ToolbarBtn icon={Bold} title="Fett" onClick={() => insertTag('<strong>', '</strong>')} />
        <ToolbarBtn icon={Italic} title="Kursiv" onClick={() => insertTag('<em>', '</em>')} />
        <div className="mx-1 h-4 w-px bg-border" />
        <ToolbarBtn icon={Heading2} title="Ueberschrift" onClick={() => insertTag('<h3>', '</h3>')} />
        <ToolbarBtn icon={List} title="Aufzaehlung" onClick={() => insertTag('<ul><li>', '</li></ul>')} />
        <ToolbarBtn icon={ListOrdered} title="Nummerierte Liste" onClick={() => insertTag('<ol><li>', '</li></ol>')} />
        <div className="mx-1 h-4 w-px bg-border" />
        <ToolbarBtn icon={Link} title="Link" onClick={() => insertTag('<a href="">', '</a>')} />
        <ToolbarBtn icon={Code} title="Code" onClick={() => insertTag('<code>', '</code>')} />

        <div className="flex-1" />

        <span className="text-[10px] text-muted-foreground mr-2">
          Ctrl+S speichern · Esc abbrechen
        </span>
      </div>

      {/* Editor textarea */}
      <textarea
        value={text}
        onChange={(e) => handleChange(e.target.value)}
        onKeyDown={handleKeyDown}
        className="flex-1 w-full bg-transparent px-4 py-3 text-sm text-foreground outline-none resize-none font-mono leading-relaxed placeholder:text-muted-foreground"
        placeholder="HTML-Inhalt bearbeiten..."
        spellCheck
      />

      {/* Footer actions */}
      <div className="flex items-center justify-end gap-2 border-t border-border px-4 py-2">
        <button
          onClick={onCancel}
          className="h-8 rounded-md border border-border px-3 text-xs text-foreground hover:bg-accent transition-colors"
        >
          Abbrechen
        </button>
        <button
          onClick={onSave}
          className="h-8 rounded-md bg-primary px-3 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          Speichern
        </button>
      </div>
    </div>
  )
}
