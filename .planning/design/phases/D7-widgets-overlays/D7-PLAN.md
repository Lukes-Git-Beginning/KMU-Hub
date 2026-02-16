# D7: Widgets & Overlays — Plan

## Goal

Die globalen Widgets und Overlays aus dem Figma implementieren: TimeTracker,
HelpWidget, ProfileSystem, OnboardingWizard.

## Figma Reference

- `desktop/design-reference/src/app/components/TimeTrackerWidget.tsx`
- `desktop/design-reference/src/app/components/TimeTracker.tsx`
- `desktop/design-reference/src/app/components/HelpWidget.tsx`
- `desktop/design-reference/src/app/components/OnboardingModal.tsx`
- `desktop/design-reference/src/app/components/OnboardingWizard.tsx`
- `desktop/design-reference/src/app/components/ProfileSwitchToast.tsx`
- `desktop/design-reference/src/app/components/ProfileSwitchTransition.tsx`
- `desktop/design-reference/src/app/components/VaultSettings.tsx`
- `desktop/design-reference/src/app/contexts/ProfileContext.tsx`

## Tasks

### 1. TimeTracker Widget
- Fixed top-right (top-20, right-4, z-40)
- Minimiert: Clock-Icon + Zeitanzeige + Expand-Button
- Voll (w-80): Timer HH:MM:SS, Projekt-Dropdown, Task-Input, Start/Stop Button
- Dunkles Theme (#1e293b)
- Start = gruen, Stop = rot

### 2. HelpWidget
- Fixed bottom-right
- Tab-System: Hilfe | Chat (mit Live-Indicator)
- Hilfe-Tab: Suchfeld, Quick Links (Docs, Videos), Artikel-Liste, Support CTA
- Chat-Tab: Mock-Chat mit Typing-Indicator (3 bouncing dots)
- Minimiert: w-80 h-16 | Voll: w-96 h-[600px]
- Emerald Header

### 3. OnboardingWizard
- Modal bei erstem App-Start
- Schritt-fuer-Schritt Einfuehrung
- Abschluss setzt Flag (localStorage)

### 4. Profile System
- ProfileContext mit localStorage Persistence
- Mehrere Arbeitsprofile mit unabhaengigen Configs
- Dashboard-Layout, Module-Reihenfolge, Quick Actions pro Profil
- Switch-Animation (fade + slide)
- ProfileSwitchToast bei Wechsel

### 5. VaultSettings
- Sicherheits-Einstellungen
- Verschluesselungs-Optionen

## Files

| Action | File |
|--------|------|
| NEW | components/widgets/TimeTrackerWidget.tsx |
| NEW | components/widgets/HelpWidget.tsx |
| NEW | components/onboarding/OnboardingWizard.tsx |
| NEW | contexts/ProfileContext.tsx |
| NEW | components/profile/ProfileSwitchTransition.tsx |
| NEW | components/profile/ProfileSwitchToast.tsx |
| MODIFY | App.tsx — Widgets einbinden |
