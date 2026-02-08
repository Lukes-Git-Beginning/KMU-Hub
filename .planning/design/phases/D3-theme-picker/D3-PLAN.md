# D3: Sidebar Redesign — Plan

> Renamed from "Theme Picker" — Sidebar redesign is higher priority and follows
> directly from the color system. Theme Picker moved to D8.

## Goal

Die Sidebar nach dem Figma-Design umbauen: Badges, Live-Indicator, responsive
Collapse, Branding, und warme Farbpalette.

## Figma Reference

- `desktop/design-reference/src/app/components/Sidebar.tsx`

## Tasks

### 1. Navigation Items
Figma definiert 10 Hauptnavigations-Items:
- Dashboard (LayoutDashboard)
- Projekte (FolderKanban)
- Aufgaben (CheckSquare)
- Dokumente (FileText)
- Nachrichten (MessageSquare) — mit Live-Badge
- Team & CRM (Users)
- Kalender (Calendar)
- E-Mail (Mail)
- Kontakte (Contact)
- Einstellungen (Settings) — am unteren Rand

### 2. Badge-System
- Text-Badge: Zähler (z.B. "3" bei Aufgaben)
- "Live" Badge: Rot pulsierend bei Meetings
- Unread-Dot: Roter Punkt im collapsed State
- Badge-Position: rechts vom Label

### 3. Responsive Collapse
- Mobile: Full-width Drawer mit Overlay Backdrop
- Tablet: Auto-collapse zu w-16 (nur Icons)
- Desktop: Toggle-Button mit Chevron
- Collapsed: Icons zentriert, Tooltip bei Hover

### 4. Branding
- Logo oben: "KMU Digital Hub"
- Subtitle: "Schweizer Loesung"
- Im collapsed State: nur Logo-Icon

### 5. Active State
- Linker Border (teal) fuer aktives Item
- Hintergrund: `--sidebar-active`
- Text: `--sidebar-primary`

### 6. Hover States
- Background: `--sidebar-accent`
- Smooth transition

## Files

| Action | File |
|--------|------|
| MODIFY | components/layout/Sidebar.tsx — Kompletter Umbau |
| POSSIBLY NEW | components/layout/SidebarBranding.tsx |
| POSSIBLY NEW | components/layout/SidebarBadge.tsx |

## Verification

- Navigation zu allen Screens funktioniert
- Badges sichtbar und korrekt positioniert
- Collapse/Expand smooth animiert
- Mobile Drawer oeffnet/schliesst mit Overlay
- Branding sichtbar oben
- Desk-Maximize-Button bleibt funktional
