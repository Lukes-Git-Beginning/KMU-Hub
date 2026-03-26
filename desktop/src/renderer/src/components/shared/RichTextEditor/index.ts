import { lazy } from 'react'

export { RichTextEditor } from './RichTextEditor'
export type { RichTextEditorProps } from './RichTextEditor'

export const LazyRichTextEditor = lazy(() => import('./RichTextEditor'))
