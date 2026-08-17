/**
 * Vendor chunk split, shared by the Electron and the web build.
 *
 * Lives in its own module so the two configs cannot drift apart: a package that
 * only one of them splits out ends up duplicated inside the entry bundle there,
 * which is invisible until someone reads a bundle report.
 *
 * .mjs rather than .ts because package.json has no "type": "module" -- a plain
 * .ts config file gets loaded as CommonJS and cannot import ESM-only plugins.
 */
export const vendorChunks = {
  'vendor-react': ['react', 'react-dom', 'react-router-dom'],
  'vendor-query': [
    '@tanstack/react-query',
    '@tanstack/react-query-persist-client',
    '@tanstack/query-async-storage-persister',
  ],
  'vendor-editor': [
    '@tiptap/react',
    '@tiptap/starter-kit',
    '@tiptap/extension-code-block-lowlight',
    '@tiptap/extension-image',
    '@tiptap/extension-link',
    '@tiptap/extension-placeholder',
    '@tiptap/extension-table',
    '@tiptap/extension-table-cell',
    '@tiptap/extension-table-header',
    '@tiptap/extension-table-row',
    '@tiptap/extension-task-item',
    '@tiptap/extension-task-list',
    '@tiptap/extension-text-align',
    '@tiptap/extension-underline',
    'lowlight',
  ],
  'vendor-video': [
    '@livekit/components-react',
    '@livekit/components-styles',
    'livekit-client',
  ],
  'vendor-workflow': ['@xyflow/react'],
  'vendor-dnd': ['@dnd-kit/core', '@dnd-kit/sortable', '@dnd-kit/utilities'],
  'vendor-ui': ['lucide-react'],
  'vendor-i18n': ['i18next', 'react-i18next', 'i18next-icu'],
  'vendor-dates': ['date-fns'],
  'vendor-state': ['zustand'],
}
