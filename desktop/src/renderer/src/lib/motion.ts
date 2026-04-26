/**
 * Motion Tokens — TS spiegel der CSS Custom Properties in styles/animations.css.
 * Use für motion/react Komponenten und inline-styles.
 * Quelle: Emil Kowalski Production-Werte (Sonner/Vaul/Linear-Stack), Cosmi-adaptiert.
 *
 * Regel: Animationen ausschliesslich auf `transform` / `opacity` (GPU).
 *        Nie `width / height / margin / padding`.
 */

export const easing = {
  /** Entry / responsive — Default für reveals, hover-ins */
  outUI: [0.23, 1, 0.32, 1] as const,
  /** On-screen movement (rare, eher für layout-shifts) */
  inOut: [0.77, 0, 0.175, 1] as const,
  /** iOS-style drawers (Sheet, BottomSheet) */
  drawer: [0.32, 0.72, 0, 1] as const,
  /** Page transitions */
  page: [0.22, 1, 0.36, 1] as const,
} as const

/** Durations in seconds (motion lib convention) */
export const duration = {
  instant: 0.1,
  press: 0.14,
  tooltip: 0.16,
  fast: 0.2,
  dropdown: 0.22,
  base: 0.28,
  modal: 0.3,
  drawer: 0.4,
  success: 0.5,
} as const

/** Spring configs — sparsam einsetzen (Sonner/Apple-Linse) */
export const spring = {
  /** Subtiler Bounce für Joy-Moments (Empty-States, Success) */
  soft: { type: 'spring', duration: 0.5, bounce: 0.2 } as const,
  /** Stiffer für Drawer/Sheet */
  drawer: { type: 'spring', duration: 0.4, bounce: 0.1 } as const,
} as const

/** Stagger-Step in seconds — zwischen sibling-reveals */
export const staggerStep = 0.04

/**
 * Motion Variants — fertige presets fuer haeufige Patterns.
 * Beispiel: <motion.div variants={variants.fadeUp} initial="hidden" animate="show" />
 */
export const variants = {
  fadeIn: {
    hidden: { opacity: 0 },
    show: { opacity: 1, transition: { duration: duration.fast, ease: easing.outUI } },
  },
  fadeUp: {
    hidden: { opacity: 0, y: 8 },
    show: { opacity: 1, y: 0, transition: { duration: duration.base, ease: easing.outUI } },
  },
  scaleIn: {
    /** Niemals scale(0) — Emil's Regel: start bei 0.95 + opacity 0 */
    hidden: { opacity: 0, scale: 0.95 },
    show: { opacity: 1, scale: 1, transition: { duration: duration.fast, ease: easing.outUI } },
  },
  scaleInBounce: {
    hidden: { opacity: 0, scale: 0.92 },
    show: { opacity: 1, scale: 1, transition: spring.soft },
  },
  pageEnter: {
    hidden: { opacity: 0, y: 12 },
    show: { opacity: 1, y: 0, transition: { duration: duration.base, ease: easing.page } },
  },
} as const

/**
 * Asymmetric Press — slow press, snappy release (Sonner-Pattern).
 * Use für hold-to-confirm Aktionen (Delete, Bestaetigen).
 */
export const asymmetricPress = {
  press: { duration: 2, ease: 'linear' } as const,
  release: { duration: duration.fast, ease: easing.outUI } as const,
} as const
