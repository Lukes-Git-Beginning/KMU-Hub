/**
 * wiki block registry — wires the shared document engine into the wiki module.
 *
 * Phase B (PB-1): the core blocks only (long-form text, heading, bullet, callout,
 * image, divider). The long-form text block leads the insert menu because a wiki
 * article is mostly prose; the special elements sit between the text. Report-only
 * blocks (chart/kpi/cover/table/page break) are intentionally absent.
 *
 * Wiki-specific blocks (toggle, code, simple table, attachment) land in the next
 * batch as additional definitions appended here — the engine and every other
 * surface stay untouched.
 */
import { buildRegistry, createCoreBlockDefs, type BlockRegistry, type BlockTypeDef } from '@/components/shared/document'

// Core blocks, keyed so we can order the insert menu deliberately.
const core = Object.fromEntries(createCoreBlockDefs().map((d) => [d.type, d])) as Record<
  string,
  BlockTypeDef
>

/**
 * Insert-menu order: prose first (the writer's default), then structure and the
 * special inline elements. Columns/layout come from the engine itself.
 */
export const wikiBlockRegistry: BlockRegistry = buildRegistry([
  core.text,
  core.heading,
  core.bullet,
  core.callout,
  core.image,
  core.divider,
])
