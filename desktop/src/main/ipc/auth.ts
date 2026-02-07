import { ipcMain, safeStorage, app } from 'electron'
import * as fs from 'fs'
import * as path from 'path'

const TOKEN_FILE = 'tokens.enc'

function getTokenPath(): string {
  return path.join(app.getPath('userData'), TOKEN_FILE)
}

export function registerAuthHandlers(): void {
  const encryptionAvailable = safeStorage.isEncryptionAvailable()

  if (!encryptionAvailable) {
    console.warn(
      '[auth] safeStorage encryption not available (Linux without keyring?). ' +
      'Tokens will be stored in plaintext. This is NOT secure for production.'
    )
  }

  ipcMain.handle('auth:get-tokens', async () => {
    const tokenPath = getTokenPath()

    if (!fs.existsSync(tokenPath)) {
      return null
    }

    try {
      const raw = fs.readFileSync(tokenPath)

      if (encryptionAvailable) {
        const decrypted = safeStorage.decryptString(raw)
        return JSON.parse(decrypted)
      }

      // Fallback: plaintext storage
      return JSON.parse(raw.toString('utf-8'))
    } catch (err) {
      console.error('[auth] Failed to read stored tokens:', err)
      return null
    }
  })

  ipcMain.handle('auth:store-tokens', async (_event, tokens: { accessToken: string; refreshToken: string }) => {
    const tokenPath = getTokenPath()
    const json = JSON.stringify(tokens)

    try {
      if (encryptionAvailable) {
        const encrypted = safeStorage.encryptString(json)
        fs.writeFileSync(tokenPath, encrypted)
      } else {
        // Fallback: plaintext storage
        fs.writeFileSync(tokenPath, json, 'utf-8')
      }
    } catch (err) {
      console.error('[auth] Failed to store tokens:', err)
      throw new Error('Failed to store tokens')
    }
  })

  ipcMain.handle('auth:clear-tokens', async () => {
    const tokenPath = getTokenPath()

    try {
      if (fs.existsSync(tokenPath)) {
        fs.unlinkSync(tokenPath)
      }
    } catch (err) {
      console.error('[auth] Failed to clear tokens:', err)
      throw new Error('Failed to clear tokens')
    }
  })
}
