# D5: Dashboard — Plan

> Renamed from "Module Styling" — Dashboard is the Startseite und verdient eine
> eigene Phase. Module-Screens werden D6.

## Goal

Das Dashboard nach dem Figma-Design umbauen: ModulesGrid statt Widget-Grid,
Alerts-Bereich, NotificationsFeed mit Tabs, Activity Feed, Quick Stats.

## Figma Reference

- `desktop/design-reference/src/app/screens/Dashboard.tsx`
- `desktop/design-reference/src/app/components/ModulesGrid.tsx`
- `desktop/design-reference/src/app/components/NotificationsFeed.tsx`
- `desktop/design-reference/src/app/components/StatCard.tsx`

## Tasks

### 1. Header-Bereich
- Begruessung: "Guten Morgen/Tag/Abend" (zeitbasiert)
- Subtitle: "Willkommen im KMU Digital Hub"

### 2. Alerts Section
- 2-Spalten Grid (1-Spalte auf Mobile)
- Warning (gelb) und Info (blau) Alert-Boxen
- Farbiger linker Rand + Icon + Text + Action-Link

### 3. NotificationsFeed
- Tabbed Feed: Mails | Kalender | Nachrichten | Projekte | Aufgaben
- View-State Toggle: Minimiert | Halb (3 Items) | Voll
- Avatar + Titel + Beschreibung + Zeit pro Item
- Scrollbar bei vielen Items

### 4. ModulesGrid
- 3-Spalten Grid (responsive: 3→2→1)
- 6 Module: Projekte, Aufgaben, Dokumente, Buchhaltung, Kommunikation, Team & CRM
- Pro Karte: farbiges Icon, Name, Beschreibung, Stats-Counter, Pfeil
- Inactive Overlay fuer deaktivierte Module (z.B. Buchhaltung "Neu")
- View-State Toggle: Minimiert | Halb | Voll
- "Module verwalten" Link

### 5. Two-Column Bottom
- Links (2/3): Letzte Aktivitaeten mit Avatars und Zeitstempeln
- Rechts (1/3): Quick Stats (Progress Bars) + Support CTA (emerald gradient)

### 6. Lukes Widget-System beruecksichtigen
- Luke hat ein react-grid-layout Widget-System gebaut
- Entscheidung: ModulesGrid ALS Widget oder Lukes System ersetzen?
- Empfehlung: Figma-Dashboard als Default-View, Lukes Widgets als Option beibehalten

## Files

| Action | File |
|--------|------|
| NEW or MODIFY | modules/dashboard/DashboardPage.tsx |
| NEW | components/dashboard/ModulesGrid.tsx |
| NEW | components/dashboard/NotificationsFeed.tsx |
| NEW | components/dashboard/AlertsSection.tsx |
| NEW | components/dashboard/ActivityFeed.tsx |
| NEW | components/dashboard/QuickStats.tsx |

## Verification

- Dashboard zeigt alle 5 Sections
- Module-Karten navigieren zu richtigen Seiten
- Alerts sichtbar mit korrekten Farben
- NotificationsFeed Tabs wechselbar
- Activity Feed zeigt Avatars und Zeiten
- Quick Stats Progress Bars sichtbar
- Responsive: Grids passen sich an
