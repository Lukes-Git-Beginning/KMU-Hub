/**
 * Shared document engine — the common block-authoring substrate for every Cosmi
 * surface where users write/compose documents (berichte, wiki, …).
 *
 *   import { DocumentBlockEditor, DocumentReader, buildRegistry,
 *            createCoreBlockDefs } from '@/components/shared/document'
 */
export * from './types'
export * from './block-registry'
export * from './blocks/core-types'
export { createCoreBlockDefs } from './blocks/CoreBlocks'
export { DocumentBlockEditor } from './DocumentBlockEditor'
export type { QuickInsert, DocumentBlockEditorProps } from './DocumentBlockEditor'
export { DocumentReader } from './DocumentReader'
export type { DocReaderSettings, DocumentReaderProps } from './DocumentReader'
