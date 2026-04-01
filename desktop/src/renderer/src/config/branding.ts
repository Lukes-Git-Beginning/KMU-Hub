/**
 * Centralized branding asset imports.
 *
 * All logo/icon images are imported here so components never reference
 * asset paths directly — they import from `@/config/branding`.
 */

// ── Cosmi (Software) ──────────────────────────────────────
import cosmiLogo from '@/../assets/branding/cosmi-logo.png'
import cosmiLogoTransparent from '@/../assets/branding/cosmi-logo-transparent.png'
import cosmiIcon64 from '@/../assets/branding/cosmi-icon-64.png'
import cosmiIcon128 from '@/../assets/branding/cosmi-icon-128.png'

// ── Zentria (Company) ──────────────────────────────────────
import zentriaLogo from '@/../assets/branding/zentria-logo.png'
import zentriaLogoTransparent from '@/../assets/branding/zentria-logo-transparent.png'
import zentriaRegisterkarte from '@/../assets/branding/zentria-registerkarte.png'

// ── Orbit (Self-Hosted) ───────────────────────────────────
import orbitLogo from '@/../assets/branding/orbit-logo.png'
import orbitLogoTransparent from '@/../assets/branding/orbit-logo-transparent.png'

export const branding = {
  cosmi: {
    /** Full logo with text — dark background (use on dark panels) */
    logo: cosmiLogo,
    /** Full logo with text — transparent background */
    logoTransparent: cosmiLogoTransparent,
    /** Icon only, 64px — for sidebar, small placements */
    icon64: cosmiIcon64,
    /** Icon only, 128px — for medium placements */
    icon128: cosmiIcon128,
  },
  zentria: {
    /** Z icon — dark background */
    logo: zentriaLogo,
    /** Z icon — checkered/transparent background (note: baked-in pattern) */
    logoTransparent: zentriaLogoTransparent,
    /** Horizontal logo with text — dark background */
    registerkarte: zentriaRegisterkarte,
  },
  orbit: {
    /** Planet logo with text — dark background */
    logo: orbitLogo,
    /** Planet logo with text — transparent background */
    logoTransparent: orbitLogoTransparent,
  },
} as const
