import { IDS } from './shared-ids'
import {
  thisWeekday,
  nextWeekday,
  daysAgo,
} from './date-helpers'

// ---------------------------------------------------------------------------
// Calendars
// ---------------------------------------------------------------------------

export const mockCalendars = {
  calendars: [
    {
      id: IDS.calendars.personal,
      name: 'Mein Kalender',
      color: '#3B82F6',
      owner_id: IDS.users.stefan,
      visible: true,
      is_default: true,
      created_at: daysAgo(180),
    },
    {
      id: IDS.calendars.team,
      name: 'Team Kalender',
      color: '#10B981',
      owner_id: IDS.users.admin,
      visible: true,
      is_default: false,
      created_at: daysAgo(180),
    },
    {
      id: IDS.calendars.meetings,
      name: 'Besprechungsraeume',
      color: '#F59E0B',
      owner_id: IDS.users.admin,
      visible: true,
      is_default: false,
      created_at: daysAgo(180),
    },
  ],
}

// ---------------------------------------------------------------------------
// Events
// ---------------------------------------------------------------------------

export const mockEvents = {
  events: [
    // 1 — Daily Standup (this week Monday)
    {
      id: 'evt-001',
      title: 'Daily Standup',
      description: 'Taegliches Team-Standup: Blocker, Fortschritt, Tagesziele.',
      calendar_id: IDS.calendars.team,
      start_time: thisWeekday(1, 9, 0),
      end_time: thisWeekday(1, 9, 15),
      all_day: false,
      location: 'Konferenzraum A',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.markus, name: 'Markus Weber', email: 'markus.weber@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.julia, name: 'Julia Fischer', email: 'julia.fischer@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.laura, name: 'Laura Braun', email: 'laura.braun@techvision.de', status: 'accepted' as const },
      ],
      organizer_id: IDS.users.stefan,
      color: '#10B981',
      recurring: 'weekly',
      created_at: daysAgo(60),
    },

    // 2 — Sprint Planning (Monday)
    {
      id: 'evt-002',
      title: 'Sprint Planning',
      description: 'Sprint 14 planen: Backlog-Refinement und Story-Point-Schaetzung.',
      calendar_id: IDS.calendars.team,
      start_time: thisWeekday(1, 10, 0),
      end_time: thisWeekday(1, 11, 30),
      all_day: false,
      location: 'Konferenzraum B',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.markus, name: 'Markus Weber', email: 'markus.weber@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.thomas, name: 'Thomas Klein', email: 'thomas.klein@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.julia, name: 'Julia Fischer', email: 'julia.fischer@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.felix, name: 'Felix Hartmann', email: 'felix.hartmann@techvision.de', status: 'pending' as const },
      ],
      organizer_id: IDS.users.stefan,
      color: '#6366F1',
      recurring: 'biweekly',
      created_at: daysAgo(90),
    },

    // 3 — Kundentermin Gruber Maschinenbau (Tuesday)
    {
      id: 'evt-003',
      title: 'Kundentermin Gruber Maschinenbau',
      description: 'Produktdemo und Anforderungsgespräch für CRM-Erweiterung.',
      calendar_id: IDS.calendars.personal,
      start_time: thisWeekday(2, 14, 0),
      end_time: thisWeekday(2, 15, 30),
      all_day: false,
      location: 'Gruber Maschinenbau GmbH, Industriestr. 42, München',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.julia, name: 'Julia Fischer', email: 'julia.fischer@techvision.de', status: 'accepted' as const },
        { user_id: IDS.contacts.gruber, name: 'Hans Gruber', email: 'h.gruber@gruber-maschinenbau.de', status: 'accepted' as const },
      ],
      organizer_id: IDS.users.stefan,
      color: '#3B82F6',
      recurring: null,
      created_at: daysAgo(5),
    },

    // 4 — Daily Standup (Tuesday)
    {
      id: 'evt-004',
      title: 'Daily Standup',
      description: 'Taegliches Team-Standup: Blocker, Fortschritt, Tagesziele.',
      calendar_id: IDS.calendars.team,
      start_time: thisWeekday(2, 9, 0),
      end_time: thisWeekday(2, 9, 15),
      all_day: false,
      location: 'Konferenzraum A',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.markus, name: 'Markus Weber', email: 'markus.weber@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.julia, name: 'Julia Fischer', email: 'julia.fischer@techvision.de', status: 'accepted' as const },
      ],
      organizer_id: IDS.users.stefan,
      color: '#10B981',
      recurring: 'weekly',
      created_at: daysAgo(60),
    },

    // 5 — Design Review (Wednesday)
    {
      id: 'evt-005',
      title: 'Design Review',
      description: 'UI/UX Review der neuen Dashboard-Komponenten mit dem Design-Team.',
      calendar_id: IDS.calendars.team,
      start_time: thisWeekday(3, 11, 0),
      end_time: thisWeekday(3, 12, 0),
      all_day: false,
      location: 'Online (LiveKit)',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.laura, name: 'Laura Braun', email: 'laura.braun@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.sophie, name: 'Sophie Lang', email: 'sophie.lang@techvision.de', status: 'pending' as const },
      ],
      organizer_id: IDS.users.laura,
      color: '#EC4899',
      recurring: 'weekly',
      created_at: daysAgo(30),
    },

    // 6 — Daily Standup (Wednesday)
    {
      id: 'evt-006',
      title: 'Daily Standup',
      description: 'Taegliches Team-Standup: Blocker, Fortschritt, Tagesziele.',
      calendar_id: IDS.calendars.team,
      start_time: thisWeekday(3, 9, 0),
      end_time: thisWeekday(3, 9, 15),
      all_day: false,
      location: 'Konferenzraum A',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.markus, name: 'Markus Weber', email: 'markus.weber@techvision.de', status: 'accepted' as const },
      ],
      organizer_id: IDS.users.stefan,
      color: '#10B981',
      recurring: 'weekly',
      created_at: daysAgo(60),
    },

    // 7 — 1:1 mit Stefan (Thursday)
    {
      id: 'evt-007',
      title: '1:1 mit Stefan',
      description: 'Woechentliches 1:1: Fortschritt, Feedback, Karriereentwicklung.',
      calendar_id: IDS.calendars.personal,
      start_time: thisWeekday(4, 16, 0),
      end_time: thisWeekday(4, 16, 30),
      all_day: false,
      location: 'Büro Stefan',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.markus, name: 'Markus Weber', email: 'markus.weber@techvision.de', status: 'accepted' as const },
      ],
      organizer_id: IDS.users.stefan,
      color: '#3B82F6',
      recurring: 'weekly',
      created_at: daysAgo(45),
    },

    // 8 — Daily Standup (Thursday)
    {
      id: 'evt-008',
      title: 'Daily Standup',
      description: 'Taegliches Team-Standup: Blocker, Fortschritt, Tagesziele.',
      calendar_id: IDS.calendars.team,
      start_time: thisWeekday(4, 9, 0),
      end_time: thisWeekday(4, 9, 15),
      all_day: false,
      location: 'Konferenzraum A',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.julia, name: 'Julia Fischer', email: 'julia.fischer@techvision.de', status: 'accepted' as const },
      ],
      organizer_id: IDS.users.stefan,
      color: '#10B981',
      recurring: 'weekly',
      created_at: daysAgo(60),
    },

    // 9 — Team Lunch (Friday)
    {
      id: 'evt-009',
      title: 'Team Lunch',
      description: 'Gemeinsames Mittagessen — Restaurant Zum Goldenen Hirschen.',
      calendar_id: IDS.calendars.team,
      start_time: thisWeekday(5, 12, 0),
      end_time: thisWeekday(5, 13, 0),
      all_day: false,
      location: 'Restaurant Zum Goldenen Hirschen',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.markus, name: 'Markus Weber', email: 'markus.weber@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.julia, name: 'Julia Fischer', email: 'julia.fischer@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.laura, name: 'Laura Braun', email: 'laura.braun@techvision.de', status: 'pending' as const },
        { user_id: IDS.users.felix, name: 'Felix Hartmann', email: 'felix.hartmann@techvision.de', status: 'accepted' as const },
      ],
      organizer_id: IDS.users.julia,
      color: '#F59E0B',
      recurring: null,
      created_at: daysAgo(3),
    },

    // 10 — Daily Standup (Friday)
    {
      id: 'evt-010',
      title: 'Daily Standup',
      description: 'Taegliches Team-Standup: Blocker, Fortschritt, Tagesziele.',
      calendar_id: IDS.calendars.team,
      start_time: thisWeekday(5, 9, 0),
      end_time: thisWeekday(5, 9, 15),
      all_day: false,
      location: 'Konferenzraum A',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.markus, name: 'Markus Weber', email: 'markus.weber@techvision.de', status: 'accepted' as const },
      ],
      organizer_id: IDS.users.stefan,
      color: '#10B981',
      recurring: 'weekly',
      created_at: daysAgo(60),
    },

    // 11 — Quartals-Review (next Monday)
    {
      id: 'evt-011',
      title: 'Quartals-Review Q1 2026',
      description: 'Quartalszahlen, OKR-Review und Ausblick Q2.',
      calendar_id: IDS.calendars.meetings,
      start_time: nextWeekday(1, 9, 0),
      end_time: nextWeekday(1, 11, 0),
      all_day: false,
      location: 'Konferenzraum A',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.markus, name: 'Markus Weber', email: 'markus.weber@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.thomas, name: 'Thomas Klein', email: 'thomas.klein@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.julia, name: 'Julia Fischer', email: 'julia.fischer@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.laura, name: 'Laura Braun', email: 'laura.braun@techvision.de', status: 'pending' as const },
      ],
      organizer_id: IDS.users.admin,
      color: '#EF4444',
      recurring: null,
      created_at: daysAgo(14),
    },

    // 12 — Messebesuch (multi-day, next week Tue-Thu)
    {
      id: 'evt-012',
      title: 'SaaS Expo München',
      description: 'Messebesuch — Stand besuchen, Networking, Vortraege.',
      calendar_id: IDS.calendars.personal,
      start_time: nextWeekday(2, 8, 0),
      end_time: nextWeekday(4, 18, 0),
      all_day: true,
      location: 'Messe München, Halle B3',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.julia, name: 'Julia Fischer', email: 'julia.fischer@techvision.de', status: 'accepted' as const },
      ],
      organizer_id: IDS.users.stefan,
      color: '#8B5CF6',
      recurring: null,
      created_at: daysAgo(30),
    },

    // 13 — Workshop Datenschutz (next Wednesday, all-day)
    {
      id: 'evt-013',
      title: 'Workshop: DSGVO-Compliance',
      description: 'Ganztaegiger Workshop mit externem Berater zur DSGVO-Konformitaet.',
      calendar_id: IDS.calendars.meetings,
      start_time: nextWeekday(3, 9, 0),
      end_time: nextWeekday(3, 17, 0),
      all_day: true,
      location: 'Konferenzraum A + B',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.thomas, name: 'Thomas Klein', email: 'thomas.klein@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.sarah, name: 'Sarah Hoffmann', email: 'sarah.hoffmann@techvision.de', status: 'pending' as const },
      ],
      organizer_id: IDS.users.thomas,
      color: '#F97316',
      recurring: null,
      created_at: daysAgo(10),
    },

    // 14 — Retro (next Friday)
    {
      id: 'evt-014',
      title: 'Sprint Retrospektive',
      description: 'Was lief gut? Was können wir verbessern? Action Items.',
      calendar_id: IDS.calendars.team,
      start_time: nextWeekday(5, 14, 0),
      end_time: nextWeekday(5, 15, 30),
      all_day: false,
      location: 'Online (LiveKit)',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.markus, name: 'Markus Weber', email: 'markus.weber@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.julia, name: 'Julia Fischer', email: 'julia.fischer@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.felix, name: 'Felix Hartmann', email: 'felix.hartmann@techvision.de', status: 'accepted' as const },
      ],
      organizer_id: IDS.users.stefan,
      color: '#6366F1',
      recurring: 'biweekly',
      created_at: daysAgo(90),
    },

    // 15 — Bewerbungsgespräch (Thursday room booking)
    {
      id: 'evt-015',
      title: 'Bewerbungsgespräch — Senior Frontend',
      description: 'Technisches Interview mit Kandidat:in für Senior Frontend Position.',
      calendar_id: IDS.calendars.meetings,
      start_time: thisWeekday(4, 10, 0),
      end_time: thisWeekday(4, 11, 30),
      all_day: false,
      location: 'Konferenzraum B',
      attendees: [
        { user_id: IDS.users.stefan, name: 'Stefan Müller', email: 'stefan.mueller@techvision.de', status: 'accepted' as const },
        { user_id: IDS.users.laura, name: 'Laura Braun', email: 'laura.braun@techvision.de', status: 'accepted' as const },
      ],
      organizer_id: IDS.users.stefan,
      color: '#14B8A6',
      recurring: null,
      created_at: daysAgo(7),
    },
  ],
  total: 15,
}

// ---------------------------------------------------------------------------
// Resources (room booking)
// ---------------------------------------------------------------------------

export const mockResources = {
  resources: [
    {
      id: 'room-1',
      name: 'Konferenzraum A',
      type: 'meeting_room',
      capacity: 12,
      equipment: ['Beamer', 'Whiteboard', 'Videokonferenz'],
      floor: '2. OG',
      available: true,
      color: '#3B82F6',
    },
    {
      id: 'room-2',
      name: 'Konferenzraum B',
      type: 'meeting_room',
      capacity: 6,
      equipment: ['TV-Display', 'Whiteboard'],
      floor: '2. OG',
      available: true,
      color: '#10B981',
    },
    {
      id: 'room-3',
      name: 'Fokusraum',
      type: 'focus_room',
      capacity: 2,
      equipment: ['Monitor'],
      floor: '1. OG',
      available: true,
      color: '#F59E0B',
    },
    {
      id: 'room-4',
      name: 'Telefonbox',
      type: 'phone_booth',
      capacity: 1,
      equipment: ['Headset'],
      floor: '1. OG',
      available: false,
      color: '#EF4444',
    },
  ],
}
