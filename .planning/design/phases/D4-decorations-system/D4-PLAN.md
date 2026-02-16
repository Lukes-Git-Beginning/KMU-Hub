# D4: Header Redesign — Plan

> Renamed from "Decorations System" — Header redesign follows Sidebar and is
> critical for the Figma design. Decorations moved to D8.

## Goal

Den Header nach dem Figma-Design umbauen: Globale Suche, DailyPlanner,
Sprachumschalter, ProfileSwitcher, und erweitertes NotificationCenter.

## Figma Reference

- `desktop/design-reference/src/app/components/Header.tsx`
- `desktop/design-reference/src/app/components/SearchBar.tsx`
- `desktop/design-reference/src/app/components/DailyPlannerWidget.tsx`
- `desktop/design-reference/src/app/components/NotificationCenter.tsx`
- `desktop/design-reference/src/app/components/ProfileSwitcher.tsx`
- `desktop/design-reference/src/app/components/ProfileMenu.tsx`

## Tasks

### 1. SearchBar (globale Suche)
- Such-Input mit Magnifying Glass Icon
- Dropdown mit Ergebnissen in 6 Kategorien (Projekte, Aufgaben, Dokumente, etc.)
- Top 3 Ergebnisse pro Kategorie
- Kategorie-Icon + farbiger Hintergrund
- Keyboard: Enter navigiert, Escape schliesst
- Responsive: flex-grow, max-w-2xl

### 2. DailyPlanner Widget
- Check-Icon Button mit Badge (Anzahl High-Priority Tasks)
- Dropdown (w-96) mit dunklem Theme
- Add Task Form + Priority Select
- Tabs: Heute | Morgen | Spaeter
- Task-Items: Checkbox, Text, Reminder, Move, Delete

### 3. Language Switcher
- Dropdown mit Flaggen-Icons: DE, FR, IT, EN
- Hidden on mobile

### 4. NotificationCenter (erweitert)
- Bell-Icon mit pulsierendem Dot wenn ungelesen
- Dropdown (w-96) mit Notification-Liste
- Sortierung: Pinned first, Unread first
- Typ-basierte Farben (mail=blue, task=purple, team=green, etc.)
- Pin/Unpin + Delete per Rechtsklick
- "Mark all as read" Link
- Footer: "Alle Benachrichtigungen anzeigen" Link

### 5. ProfileSwitcher
- Icon + Profilname + Dropdown Chevron
- Wechsel zwischen Arbeitsprofilen
- Keyboard Shortcuts (Ctrl+Shift+1-9)
- Create New / Manage Profiles Links

### 6. ProfileMenu
- Avatar + Name/Rolle + Dropdown
- Links zu: Profil, Einstellungen, Support
- TimeTracker Zugang

### 7. Layout
- Height: h-16
- Reihenfolge: [MenuBtn(mobile)] SearchBar | DailyPlanner | Language | Notifications | ProfileSwitcher | ProfileMenu
- Background: `--header-background`
- Border: bottom `--header-border`

## Files

| Action | File |
|--------|------|
| MODIFY | components/layout/Header.tsx — Kompletter Umbau |
| NEW | components/header/SearchBar.tsx |
| NEW | components/header/DailyPlannerWidget.tsx |
| NEW | components/header/LanguageSwitcher.tsx |
| NEW | components/header/NotificationCenter.tsx |
| NEW | components/header/ProfileSwitcher.tsx |
| NEW | components/header/ProfileMenu.tsx |
| NEW | contexts/ProfileContext.tsx (wenn nicht von Luke) |

## Verification

- Suche oeffnet Dropdown mit Ergebnissen
- DailyPlanner zeigt Tasks mit Prioritaeten
- Sprache wechselbar (zumindest UI-seitig)
- Notifications: Bell + Dropdown + Pin/Unpin
- Profile: Wechsel zwischen Profilen
- Responsive: Elemente verstecken sich auf kleineren Screens
- Desk-Maximize-Button bleibt erhalten
