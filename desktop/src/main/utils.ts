import { join } from 'path'

/**
 * Resolve a file path inside `src/main/resources/` (dev) or the
 * packaged `resources/` directory (production).
 */
export function getResourcePath(filename: string): string {
  if (process.env.ELECTRON_RENDERER_URL) {
    // Development — files at their source location
    return join(__dirname, '../../src/main/resources', filename)
  }
  // Production — electron-builder copies extraResources here
  return join(process.resourcesPath, 'resources', filename)
}
