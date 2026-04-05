import { IDS } from './shared-ids'
import { hoursAgo, daysAgo, minutesAgo } from './date-helpers'

// ---------------------------------------------------------------------------
// Email Account
// ---------------------------------------------------------------------------

export const mockEmailAccount = {
  account: {
    id: IDS.emailAccounts.main,
    email: 'stefan.mueller@techvision.de',
    name: 'Stefan Müller',
    provider: 'imap',
    imap_host: 'imap.techvision.de',
    smtp_host: 'smtp.techvision.de',
    is_active: true,
    last_sync_at: minutesAgo(5),
    created_at: daysAgo(180),
  },
}

// ---------------------------------------------------------------------------
// Email Folders
// ---------------------------------------------------------------------------

export const mockEmailFolders = {
  folders: [
    {
      id: IDS.emailFolders.inbox,
      name: 'Posteingang',
      type: 'inbox',
      unread_count: 12,
      total_count: 247,
      account_id: IDS.emailAccounts.main,
    },
    {
      id: IDS.emailFolders.sent,
      name: 'Gesendet',
      type: 'sent',
      unread_count: 0,
      total_count: 189,
      account_id: IDS.emailAccounts.main,
    },
    {
      id: IDS.emailFolders.drafts,
      name: 'Entwürfe',
      type: 'drafts',
      unread_count: 0,
      total_count: 2,
      account_id: IDS.emailAccounts.main,
    },
    {
      id: IDS.emailFolders.trash,
      name: 'Papierkorb',
      type: 'trash',
      unread_count: 0,
      total_count: 34,
      account_id: IDS.emailAccounts.main,
    },
    {
      id: IDS.emailFolders.archive,
      name: 'Archiv',
      type: 'archive',
      unread_count: 0,
      total_count: 1520,
      account_id: IDS.emailAccounts.main,
    },
  ],
}

// ---------------------------------------------------------------------------
// Email Messages
// ---------------------------------------------------------------------------

const inboxMessages = [
  {
    id: 'em-001',
    subject: 'Angebot: IT-Infrastruktur Modernisierung',
    from: { name: 'Klaus Gruber', email: 'k.gruber@gruber-maschinenbau.de' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: minutesAgo(12),
    snippet:
      'Sehr geehrter Herr Müller, vielen Dank für das ausführliche Gespräch gestern. Anbei sende ich Ihnen unser aktualisiertes Angebot...',
    body_text:
      'Sehr geehrter Herr Müller,\n\nvielen Dank für das ausführliche Gespräch gestern. Anbei sende ich Ihnen unser aktualisiertes Angebot für die IT-Infrastruktur Modernisierung.\n\nDie wichtigsten Punkte:\n- Migration auf Cloud-basierte Infrastruktur\n- Einführung von Cosmi als zentrales CRM\n- Schulung der Mitarbeiter (2 Tage Onsite)\n\nGesamtvolumen: CHF 48\'000 (exkl. MwSt.)\n\nBitte lassen Sie mich wissen, falls Sie Rueckfragen haben.\n\nMit freundlichen Grüßen\nKlaus Gruber\nGeschaeftsfuehrer\nGruber Maschinenbau GmbH',
    body_html:
      '<p>Sehr geehrter Herr Müller,</p><p>vielen Dank für das ausführliche Gespräch gestern. Anbei sende ich Ihnen unser aktualisiertes Angebot für die IT-Infrastruktur Modernisierung.</p><p><strong>Die wichtigsten Punkte:</strong></p><ul><li>Migration auf Cloud-basierte Infrastruktur</li><li>Einführung von Cosmi als zentrales CRM</li><li>Schulung der Mitarbeiter (2 Tage Onsite)</li></ul><p>Gesamtvolumen: CHF 48\'000 (exkl. MwSt.)</p><p>Bitte lassen Sie mich wissen, falls Sie Rueckfragen haben.</p><p>Mit freundlichen Grüßen<br/>Klaus Gruber<br/>Geschaeftsfuehrer<br/>Gruber Maschinenbau GmbH</p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: false,
    is_starred: true,
    has_attachments: true,
    attachments: [
      {
        id: 'att-001',
        filename: 'Angebot_IT-Modernisierung_2026.pdf',
        size: 245000,
        content_type: 'application/pdf',
      },
    ],
    thread_id: 'th-001',
  },
  {
    id: 'em-002',
    subject: 'RE: Terminbestaetigung Freitag',
    from: { name: 'Sabine Schneider', email: 's.schneider@rhein-consulting.de' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: minutesAgo(45),
    snippet:
      'Hallo Herr Müller, der Termin am Freitag um 10:00 Uhr passt perfekt. Ich bringe Herrn Dr. Weber mit...',
    body_text:
      'Hallo Herr Müller,\n\nder Termin am Freitag um 10:00 Uhr passt perfekt. Ich bringe Herrn Dr. Weber aus der Rechtsabteilung mit, da wir auch die AVV-Thematik besprechen moechten.\n\nBitte reservieren Sie einen Besprechungsraum für ca. 2 Stunden.\n\nViele Grüße\nSabine Schneider\nRhein Consulting GmbH',
    body_html:
      '<p>Hallo Herr Müller,</p><p>der Termin am Freitag um 10:00 Uhr passt perfekt. Ich bringe Herrn Dr. Weber aus der Rechtsabteilung mit, da wir auch die AVV-Thematik besprechen moechten.</p><p>Bitte reservieren Sie einen Besprechungsraum für ca. 2 Stunden.</p><p>Viele Grüße<br/>Sabine Schneider<br/>Rhein Consulting GmbH</p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: false,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-002',
  },
  {
    id: 'em-003',
    subject: 'Rechnung #2026-089',
    from: { name: 'Buchhaltung Hetzner', email: 'billing@hetzner.com' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: hoursAgo(1.5),
    snippet:
      'Ihre Rechnung für den Monat März 2026 liegt bereit. Betrag: EUR 89,40. Zahlungsziel: 14 Tage...',
    body_text:
      'Sehr geehrter Herr Müller,\n\nIhre Rechnung für den Monat März 2026 liegt bereit.\n\nBetrag: EUR 89,40\nZahlungsziel: 14 Tage\nKundennummer: HZ-2024-48291\n\nDie Rechnung finden Sie im Anhang sowie in Ihrem Kundenpanel.\n\nMit freundlichen Grüßen\nHetzner Online GmbH',
    body_html:
      '<p>Sehr geehrter Herr Müller,</p><p>Ihre Rechnung für den Monat März 2026 liegt bereit.</p><table><tr><td>Betrag:</td><td>EUR 89,40</td></tr><tr><td>Zahlungsziel:</td><td>14 Tage</td></tr><tr><td>Kundennummer:</td><td>HZ-2024-48291</td></tr></table><p>Die Rechnung finden Sie im Anhang sowie in Ihrem Kundenpanel.</p><p>Mit freundlichen Grüßen<br/>Hetzner Online GmbH</p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: false,
    is_starred: false,
    has_attachments: true,
    attachments: [
      {
        id: 'att-002',
        filename: 'Rechnung_HZ-2026-089.pdf',
        size: 98000,
        content_type: 'application/pdf',
      },
    ],
    thread_id: 'th-003',
  },
  {
    id: 'em-004',
    subject: 'Einladung: Quartals-Review Q1 2026',
    from: { name: 'Julia Hoffmann', email: 'julia.hoffmann@techvision.de' },
    to: [
      { name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' },
      { name: 'Markus Weber', email: 'markus.weber@techvision.de' },
    ],
    date: hoursAgo(2),
    snippet:
      'Hallo zusammen, hiermit lade ich euch zum Quartals-Review ein. Datum: Freitag, 14:00 Uhr...',
    body_text:
      'Hallo zusammen,\n\nhiermit lade ich euch zum Quartals-Review Q1 2026 ein.\n\nDatum: Freitag, 14:00 Uhr\nOrt: Grosser Besprechungsraum / Video\nDauer: ca. 90 Minuten\n\nAgenda:\n1. Umsatz-Update (Markus)\n2. Produkt-Roadmap (Stefan)\n3. Team & Hiring (Julia)\n4. Ausblick Q2\n\nBitte bereitet eure Teile vor.\n\nViele Grüße\nJulia',
    body_html:
      '<p>Hallo zusammen,</p><p>hiermit lade ich euch zum Quartals-Review Q1 2026 ein.</p><p><strong>Datum:</strong> Freitag, 14:00 Uhr<br/><strong>Ort:</strong> Grosser Besprechungsraum / Video<br/><strong>Dauer:</strong> ca. 90 Minuten</p><p><strong>Agenda:</strong></p><ol><li>Umsatz-Update (Markus)</li><li>Produkt-Roadmap (Stefan)</li><li>Team &amp; Hiring (Julia)</li><li>Ausblick Q2</li></ol><p>Bitte bereitet eure Teile vor.</p><p>Viele Grüße<br/>Julia</p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: false,
    is_starred: true,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-004',
  },
  {
    id: 'em-005',
    subject: 'Bewerbung: Senior Go-Entwickler',
    from: { name: 'Tobias Keller', email: 'tobias.keller@protonmail.com' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: hoursAgo(3),
    snippet:
      'Sehr geehrter Herr Müller, mit grossem Interesse habe ich Ihre Stellenausschreibung gelesen...',
    body_text:
      'Sehr geehrter Herr Müller,\n\nmit grossem Interesse habe ich Ihre Stellenausschreibung für einen Senior Go-Entwickler gelesen. Mit ueber 8 Jahren Erfahrung in Go, Microservices und Kubernetes bringe ich die idealen Voraussetzungen mit.\n\nMeine Highlights:\n- 5 Jahre Go-Entwicklung (davon 3 Jahre in Führungsrolle)\n- Erfahrung mit gRPC, PostgreSQL, Redis\n- Open-Source-Contributor (github.com/tkeller)\n\nMeine vollstaendigen Unterlagen finden Sie im Anhang.\n\nIch freue mich auf ein Gespräch!\n\nMit freundlichen Grüßen\nTobias Keller',
    body_html:
      '<p>Sehr geehrter Herr Müller,</p><p>mit grossem Interesse habe ich Ihre Stellenausschreibung für einen Senior Go-Entwickler gelesen. Mit ueber 8 Jahren Erfahrung in Go, Microservices und Kubernetes bringe ich die idealen Voraussetzungen mit.</p><p><strong>Meine Highlights:</strong></p><ul><li>5 Jahre Go-Entwicklung (davon 3 Jahre in Führungsrolle)</li><li>Erfahrung mit gRPC, PostgreSQL, Redis</li><li>Open-Source-Contributor (github.com/tkeller)</li></ul><p>Meine vollstaendigen Unterlagen finden Sie im Anhang.</p><p>Ich freue mich auf ein Gespräch!</p><p>Mit freundlichen Grüßen<br/>Tobias Keller</p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: false,
    is_starred: false,
    has_attachments: true,
    attachments: [
      {
        id: 'att-003',
        filename: 'Lebenslauf_Tobias_Keller.pdf',
        size: 320000,
        content_type: 'application/pdf',
      },
      {
        id: 'att-004',
        filename: 'Zeugnisse_Keller.pdf',
        size: 1200000,
        content_type: 'application/pdf',
      },
    ],
    thread_id: 'th-005',
  },
  {
    id: 'em-006',
    subject: 'DSGVO-Audit: Ergebnisbericht',
    from: { name: 'Dr. Andrea Roth', email: 'a.roth@datenschutz-partner.ch' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: hoursAgo(4),
    snippet:
      'Sehr geehrter Herr Müller, anbei der Ergebnisbericht des DSGVO-Audits vom 18. März...',
    body_text:
      'Sehr geehrter Herr Müller,\n\nanbei der Ergebnisbericht des DSGVO-Audits vom 18. März 2026.\n\nZusammenfassung:\n- 42 von 45 Kriterien erfuellt\n- 3 kleinere Maengel identifiziert (Details im Bericht)\n- Empfehlung: Zertifizierung beantragen\n\nDie 3 offenen Punkte betreffen:\n1. Aufbewahrungsfristen für Logdaten\n2. Cookie-Banner Formulierung\n3. Auskunftsrecht-Prozess Dokumentation\n\nBitte beheben Sie diese bis zum 15. April.\n\nMit freundlichen Grüßen\nDr. Andrea Roth\nDatenschutz Partner AG',
    body_html:
      '<p>Sehr geehrter Herr Müller,</p><p>anbei der Ergebnisbericht des DSGVO-Audits vom 18. März 2026.</p><p><strong>Zusammenfassung:</strong></p><ul><li>42 von 45 Kriterien erfuellt</li><li>3 kleinere Maengel identifiziert (Details im Bericht)</li><li>Empfehlung: Zertifizierung beantragen</li></ul><p>Bitte beheben Sie diese bis zum 15. April.</p><p>Mit freundlichen Grüßen<br/>Dr. Andrea Roth<br/>Datenschutz Partner AG</p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: true,
    is_starred: true,
    has_attachments: true,
    attachments: [
      {
        id: 'att-005',
        filename: 'DSGVO_Audit_Bericht_2026.pdf',
        size: 890000,
        content_type: 'application/pdf',
      },
    ],
    thread_id: 'th-006',
  },
  {
    id: 'em-007',
    subject: 'Feedback: Demo-Praesentation letzte Woche',
    from: { name: 'Hans Weber', email: 'h.weber@helvetia-software.ch' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: hoursAgo(5),
    snippet:
      'Hallo Herr Müller, nochmals vielen Dank für die ausführliche Demo. Unser Team war beeindruckt...',
    body_text:
      'Hallo Herr Müller,\n\nnochmals vielen Dank für die ausführliche Demo letzte Woche. Unser Team war beeindruckt von der Integrationsfaehigkeit und dem EU-Hosting-Konzept.\n\nWir moechten gerne in die nächste Phase gehen und haetten folgende Fragen:\n1. Wie sieht das Onboarding konkret aus?\n2. Können wir eine Testumgebung bekommen?\n3. Gibt es Mengenrabatte ab 50 Usern?\n\nKönnten wir nächste Woche einen Folgetermin vereinbaren?\n\nBeste Grüße\nHans Weber\nCTO, Helvetia Software AG',
    body_html:
      '<p>Hallo Herr Müller,</p><p>nochmals vielen Dank für die ausführliche Demo letzte Woche. Unser Team war beeindruckt von der Integrationsfaehigkeit und dem EU-Hosting-Konzept.</p><p>Wir moechten gerne in die nächste Phase gehen und haetten folgende Fragen:</p><ol><li>Wie sieht das Onboarding konkret aus?</li><li>Können wir eine Testumgebung bekommen?</li><li>Gibt es Mengenrabatte ab 50 Usern?</li></ol><p>Könnten wir nächste Woche einen Folgetermin vereinbaren?</p><p>Beste Grüße<br/>Hans Weber<br/>CTO, Helvetia Software AG</p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-007',
  },
  {
    id: 'em-008',
    subject: 'GitHub Actions: Build fehlgeschlagen - main',
    from: { name: 'GitHub', email: 'noreply@github.com' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: hoursAgo(6),
    snippet: 'Build #847 on main has failed. Click here for details...',
    body_text:
      'Build #847 on branch main has failed.\n\nCommit: fa17fc3 - feat: add deployment auto-rollback\nAuthor: Stefan Müller\nFailed job: test-integration\n\nError: connection refused on port 5432\n\nView details: https://github.com/Lukes-Git-Beginning/KMU-Hub/actions/runs/847',
    body_html:
      '<p><strong>Build #847</strong> on branch <code>main</code> has failed.</p><p>Commit: <code>fa17fc3</code> - feat: add deployment auto-rollback<br/>Author: Stefan Müller<br/>Failed job: <code>test-integration</code></p><p>Error: <code>connection refused on port 5432</code></p><p><a href="#">View details</a></p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-008',
  },
  {
    id: 'em-009',
    subject: 'Vertragsverlaengerung Bavaria Elektro',
    from: { name: 'Maria Huber', email: 'm.huber@bavaria-elektro.de' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: hoursAgo(8),
    snippet:
      'Sehr geehrter Herr Müller, wir moechten unseren Vertrag gerne um ein weiteres Jahr verlaengern...',
    body_text:
      'Sehr geehrter Herr Müller,\n\nwir moechten unseren Vertrag gerne um ein weiteres Jahr verlaengern. Die Zusammenarbeit war bisher ausgezeichnet und das Tool hat sich bei uns fest etabliert.\n\nKönnen Sie uns ein Angebot für die Verlaengerung zukommen lassen? Idealerweise mit den neuen Modulen (Chat + Video).\n\nMit freundlichen Grüßen\nMaria Huber\nIT-Leiterin\nBavaria Elektro GmbH',
    body_html:
      '<p>Sehr geehrter Herr Müller,</p><p>wir moechten unseren Vertrag gerne um ein weiteres Jahr verlaengern. Die Zusammenarbeit war bisher ausgezeichnet und das Tool hat sich bei uns fest etabliert.</p><p>Können Sie uns ein Angebot für die Verlaengerung zukommen lassen? Idealerweise mit den neuen Modulen (Chat + Video).</p><p>Mit freundlichen Grüßen<br/>Maria Huber<br/>IT-Leiterin<br/>Bavaria Elektro GmbH</p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: true,
    is_starred: true,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-009',
  },
  {
    id: 'em-010',
    subject: 'Neue Datenschutzrichtlinie ab April 2026',
    from: { name: 'BITKOM Newsletter', email: 'newsletter@bitkom.org' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: hoursAgo(10),
    snippet:
      'Ab dem 1. April 2026 treten neue EU-Regelungen für Cloud-Dienste in Kraft...',
    body_text:
      'Sehr geehrte Damen und Herren,\n\nab dem 1. April 2026 treten neue EU-Regelungen für Cloud-Dienste in Kraft. Wir fassen die wichtigsten Änderungen zusammen:\n\n1. Verschaerfte Anforderungen an Datenlokalisierung\n2. Neue Meldepflichten bei Sicherheitsvorfaellen\n3. Erweiterte Rechte für Betroffene\n\nMehr Informationen unter: https://bitkom.org/datenschutz-2026\n\nMit freundlichen Grüßen\nBITKOM e.V.',
    body_html:
      '<p>Sehr geehrte Damen und Herren,</p><p>ab dem 1. April 2026 treten neue EU-Regelungen für Cloud-Dienste in Kraft.</p><ol><li>Verschaerfte Anforderungen an Datenlokalisierung</li><li>Neue Meldepflichten bei Sicherheitsvorfaellen</li><li>Erweiterte Rechte für Betroffene</li></ol>',
    folder_id: IDS.emailFolders.inbox,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-010',
  },
  {
    id: 'em-011',
    subject: 'RE: Serverraum Zugang — neuer Schluessel',
    from: { name: 'Thomas Braun', email: 'thomas.braun@techvision.de' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: hoursAgo(12),
    snippet:
      'Hey Stefan, der neue Schluessel liegt bei mir am Schreibtisch. Hol ihn dir wenn du Zeit hast...',
    body_text:
      'Hey Stefan,\n\nder neue Schluessel für den Serverraum liegt bei mir am Schreibtisch. Hol ihn dir wenn du Zeit hast.\n\nDer alte Schluessel muss bis Freitag zurückgegeben werden.\n\nGruss\nThomas',
    body_html:
      '<p>Hey Stefan,</p><p>der neue Schluessel für den Serverraum liegt bei mir am Schreibtisch. Hol ihn dir wenn du Zeit hast.</p><p>Der alte Schluessel muss bis Freitag zurückgegeben werden.</p><p>Gruss<br/>Thomas</p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-011',
  },
  {
    id: 'em-012',
    subject: 'Anfrage: Partnerschaft / Reseller-Programm',
    from: { name: 'Lukas Steiner', email: 'l.steiner@alpen-logistik.at' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: hoursAgo(14),
    snippet:
      'Guten Tag Herr Müller, wir sind ein Logistikunternehmen aus Salzburg und suchen eine CRM-Loesung...',
    body_text:
      'Guten Tag Herr Müller,\n\nwir sind ein Logistikunternehmen aus Salzburg mit 120 Mitarbeitern und suchen eine CRM-Loesung mit EU-Hosting. Ihr Produkt wurde uns von Gruber Maschinenbau empfohlen.\n\nHaben Sie ein Reseller-/Partnerschafts-Programm? Wir wuerden gerne mehrere unserer Kunden ebenfalls an Sie vermitteln.\n\nKönnten wir einen Termin für ein Erstgespräch vereinbaren?\n\nMit freundlichen Grüßen\nLukas Steiner\nGeschaeftsfuehrer\nAlpen Logistik GmbH',
    body_html:
      '<p>Guten Tag Herr Müller,</p><p>wir sind ein Logistikunternehmen aus Salzburg mit 120 Mitarbeitern und suchen eine CRM-Loesung mit EU-Hosting. Ihr Produkt wurde uns von Gruber Maschinenbau empfohlen.</p><p>Haben Sie ein Reseller-/Partnerschafts-Programm?</p><p>Mit freundlichen Grüßen<br/>Lukas Steiner<br/>Geschaeftsfuehrer<br/>Alpen Logistik GmbH</p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: false,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-012',
  },
  {
    id: 'em-013',
    subject: 'Meeting-Notizen: Sprint Retrospektive',
    from: { name: 'Jan Schulze', email: 'jan.schulze@techvision.de' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: daysAgo(1),
    snippet:
      'Hi Stefan, hier die Zusammenfassung der Retro von gestern. Positiv: Deployment-Pipeline...',
    body_text:
      'Hi Stefan,\n\nhier die Zusammenfassung der Sprint-Retro:\n\nPositiv:\n- Deployment-Pipeline laeuft stabil\n- Neue Test-Suite spart viel Zeit\n- Team-Stimmung ist gut\n\nVerbesserungen:\n- Code-Reviews dauern zu lange\n- Dokumentation hinkt hinterher\n- Mehr Pair-Programming gewünscht\n\nAction Items:\n- Review-SLA: max. 24h (Stefan)\n- Docs-Sprint nächste Woche einplanen (Jan)\n- Pair-Programming Slots im Kalender (alle)\n\nGruss\nJan',
    body_html:
      '<p>Hi Stefan,</p><p>hier die Zusammenfassung der Sprint-Retro:</p><h3>Positiv:</h3><ul><li>Deployment-Pipeline laeuft stabil</li><li>Neue Test-Suite spart viel Zeit</li><li>Team-Stimmung ist gut</li></ul><h3>Verbesserungen:</h3><ul><li>Code-Reviews dauern zu lange</li><li>Dokumentation hinkt hinterher</li><li>Mehr Pair-Programming gewünscht</li></ul>',
    folder_id: IDS.emailFolders.inbox,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-013',
  },
  {
    id: 'em-014',
    subject: 'Bexio API: Neue Endpunkte verfügbar',
    from: { name: 'Bexio Developer', email: 'developer@bexio.com' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: daysAgo(1),
    snippet:
      'Sehr geehrter Entwickler, wir freuen uns Ihnen mitzuteilen, dass neue API-Endpunkte...',
    body_text:
      'Sehr geehrter Entwickler,\n\nwir freuen uns, Ihnen mitzuteilen, dass folgende neue API-Endpunkte ab sofort verfügbar sind:\n\n- POST /v4/invoices/bulk - Massenrechnungserstellung\n- GET /v4/reports/revenue - Umsatzberichte\n- PATCH /v4/contacts/merge - Kontakte zusammenfuehren\n\nDokumentation: https://docs.bexio.com/v4\n\nBreaking Changes: Keine\n\nMit freundlichen Grüßen\nBexio Developer Team',
    body_html:
      '<p>Sehr geehrter Entwickler,</p><p>wir freuen uns, Ihnen mitzuteilen, dass folgende neue API-Endpunkte ab sofort verfügbar sind:</p><ul><li><code>POST /v4/invoices/bulk</code> - Massenrechnungserstellung</li><li><code>GET /v4/reports/revenue</code> - Umsatzberichte</li><li><code>PATCH /v4/contacts/merge</code> - Kontakte zusammenfuehren</li></ul>',
    folder_id: IDS.emailFolders.inbox,
    is_read: false,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-014',
  },
  {
    id: 'em-015',
    subject: 'Einladung: Swiss Tech Summit 2026',
    from: { name: 'Swiss Tech Events', email: 'events@swisstechsummit.ch' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: daysAgo(1),
    snippet:
      'Sie sind eingeladen zum Swiss Tech Summit am 15.-16. Mai 2026 in Zürich...',
    body_text:
      'Sehr geehrter Herr Müller,\n\nSie sind eingeladen zum Swiss Tech Summit am 15.-16. Mai 2026 in Zürich.\n\nHighlights:\n- Keynote: Die Zukunft von B2B-SaaS in Europa\n- Workshop: DSGVO-konforme Architektur\n- Networking Dinner mit 200+ CTOs\n\nEarly-Bird-Tickets: CHF 490 (bis 15. April)\n\nAnmeldung: https://swisstechsummit.ch/register\n\nMit freundlichen Grüßen\nSwiss Tech Events AG',
    body_html:
      '<p>Sehr geehrter Herr Müller,</p><p>Sie sind eingeladen zum Swiss Tech Summit am 15.-16. Mai 2026 in Zürich.</p><p><strong>Highlights:</strong></p><ul><li>Keynote: Die Zukunft von B2B-SaaS in Europa</li><li>Workshop: DSGVO-konforme Architektur</li><li>Networking Dinner mit 200+ CTOs</li></ul><p>Early-Bird-Tickets: CHF 490 (bis 15. April)</p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: false,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-015',
  },
  {
    id: 'em-016',
    subject: 'Lizenzverlängerung JetBrains All Products Pack',
    from: { name: 'JetBrains Sales', email: 'sales@jetbrains.com' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: daysAgo(2),
    snippet:
      'Ihre JetBrains-Lizenz laeuft in 30 Tagen ab. Verlaengern Sie jetzt mit 20% Rabatt...',
    body_text:
      'Sehr geehrter Herr Müller,\n\nIhre JetBrains All Products Pack Lizenz (8 Plaetze) laeuft am 25. April 2026 ab.\n\nVerlaengerungspreis: EUR 4.792 (20% Treuerabatt)\n\nBitte verlaengern Sie rechtzeitig, um eine Unterbrechung zu vermeiden.\n\nJetBrains Sales Team',
    body_html:
      '<p>Sehr geehrter Herr Müller,</p><p>Ihre JetBrains All Products Pack Lizenz (8 Plaetze) laeuft am 25. April 2026 ab.</p><p>Verlaengerungspreis: <strong>EUR 4.792</strong> (20% Treuerabatt)</p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-016',
  },
  {
    id: 'em-017',
    subject: 'Support-Ticket #891: 2FA Problem geloest',
    from: { name: 'Lisa Werner', email: 'lisa.werner@techvision.de' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: daysAgo(2),
    snippet:
      'Hi Stefan, das 2FA-Problem von heute Morgen ist geloest. War ein Timeout beim SMS-Provider...',
    body_text:
      'Hi Stefan,\n\ndas 2FA-Problem (Ticket #891) ist geloest. War ein Timeout beim SMS-Provider Twilio.\n\nRoot Cause: Provider hatte kurzzeitige Störung in der EU-Region.\nLoesung: Fallback auf TOTP (Authenticator-App) als Alternative konfiguriert.\n\nEmpfehlung: Generell auf TOTP als primaere 2FA-Methode umstellen.\n\nGruss\nLisa',
    body_html:
      '<p>Hi Stefan,</p><p>das 2FA-Problem (Ticket #891) ist geloest. War ein Timeout beim SMS-Provider Twilio.</p><p><strong>Root Cause:</strong> Provider hatte kurzzeitige Störung in der EU-Region.<br/><strong>Loesung:</strong> Fallback auf TOTP (Authenticator-App) als Alternative konfiguriert.</p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-017',
  },
  {
    id: 'em-018',
    subject: 'Angebot Büromoebel — Ergonomische Arbeitsplaetze',
    from: { name: 'Petra Schwarz', email: 'p.schwarz@officewelt.de' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: daysAgo(2),
    snippet:
      'Sehr geehrter Herr Müller, wie telefonisch besprochen sende ich Ihnen unser Angebot...',
    body_text:
      'Sehr geehrter Herr Müller,\n\nwie telefonisch besprochen sende ich Ihnen unser Angebot für 6 ergonomische Arbeitsplaetze:\n\n- 6x Steh-Sitz-Tisch (160x80): EUR 890/Stück\n- 6x Ergonomischer Bürostuhl: EUR 650/Stück\n- Lieferung & Aufbau: EUR 480\n\nGesamtpreis: EUR 9.720 (zzgl. MwSt.)\nLieferzeit: 2-3 Wochen\n\nMit freundlichen Grüßen\nPetra Schwarz\nOffice Welt GmbH',
    body_html:
      '<p>Sehr geehrter Herr Müller,</p><p>wie telefonisch besprochen sende ich Ihnen unser Angebot für 6 ergonomische Arbeitsplaetze.</p><table><tr><td>6x Steh-Sitz-Tisch</td><td>EUR 890/Stück</td></tr><tr><td>6x Bürostuhl</td><td>EUR 650/Stück</td></tr><tr><td>Lieferung & Aufbau</td><td>EUR 480</td></tr></table><p><strong>Gesamt: EUR 9.720 (zzgl. MwSt.)</strong></p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: false,
    is_starred: false,
    has_attachments: true,
    attachments: [
      {
        id: 'att-006',
        filename: 'Angebot_OfficeWelt_2026.pdf',
        size: 450000,
        content_type: 'application/pdf',
      },
    ],
    thread_id: 'th-018',
  },
  {
    id: 'em-019',
    subject: 'RE: API-Dokumentation für Bexio-Integration',
    from: { name: 'Felix Richter', email: 'felix.richter@techvision.de' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: daysAgo(3),
    snippet:
      'Hey Stefan, die API-Docs für die Bexio-Integration sind fertig. Link im Wiki...',
    body_text:
      'Hey Stefan,\n\ndie API-Docs für die Bexio-Integration sind fertig. Link:\nhttps://wiki.techvision.de/bexio-integration\n\nAbgedeckt:\n- OAuth2 Flow\n- Kontakt-Sync (bidirektional)\n- Rechnungs-Import\n- Webhook-Empfang\n\nReview bitte bis Donnerstag.\n\nGruss\nFelix',
    body_html:
      '<p>Hey Stefan,</p><p>die API-Docs für die Bexio-Integration sind fertig.</p><ul><li>OAuth2 Flow</li><li>Kontakt-Sync (bidirektional)</li><li>Rechnungs-Import</li><li>Webhook-Empfang</li></ul><p>Review bitte bis Donnerstag.</p>',
    folder_id: IDS.emailFolders.inbox,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-019',
  },
  {
    id: 'em-020',
    subject: 'Willkommen bei Hetzner Cloud — Ihr Server ist bereit',
    from: { name: 'Hetzner Cloud', email: 'cloud@hetzner.com' },
    to: [{ name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' }],
    date: daysAgo(3),
    snippet:
      'Ihr neuer Cloud-Server CPX42 wurde erfolgreich erstellt. IP: 178.104.38.195...',
    body_text:
      'Sehr geehrter Herr Müller,\n\nIhr neuer Cloud-Server wurde erfolgreich erstellt.\n\nServer: CPX42 (8 vCPU, 16 GB RAM, 240 GB SSD)\nStandort: Nürnberg (nbg1)\nIP: 178.104.38.195\nOS: Ubuntu 24.04\n\nZugangsdaten wurden separat per SMS versendet.\n\nHetzner Online GmbH',
    body_html:
      '<p>Sehr geehrter Herr Müller,</p><p>Ihr neuer Cloud-Server wurde erfolgreich erstellt.</p><table><tr><td>Server:</td><td>CPX42 (8 vCPU, 16 GB RAM, 240 GB SSD)</td></tr><tr><td>Standort:</td><td>Nürnberg (nbg1)</td></tr><tr><td>IP:</td><td>178.104.38.195</td></tr><tr><td>OS:</td><td>Ubuntu 24.04</td></tr></table>',
    folder_id: IDS.emailFolders.inbox,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-020',
  },
]

const sentMessages = [
  {
    id: 'em-s-001',
    subject: 'RE: Angebot: IT-Infrastruktur Modernisierung',
    from: { name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' },
    to: [{ name: 'Klaus Gruber', email: 'k.gruber@gruber-maschinenbau.de' }],
    date: hoursAgo(1),
    snippet:
      'Sehr geehrter Herr Gruber, vielen Dank für das Angebot. Wir prüfen es intern und melden uns...',
    body_text:
      'Sehr geehrter Herr Gruber,\n\nvielen Dank für das aktualisierte Angebot. Wir prüfen es intern und melden uns bis Ende der Woche.\n\nMit freundlichen Grüßen\nStefan Müller\nTechVision GmbH',
    body_html:
      '<p>Sehr geehrter Herr Gruber,</p><p>vielen Dank für das aktualisierte Angebot. Wir prüfen es intern und melden uns bis Ende der Woche.</p><p>Mit freundlichen Grüßen<br/>Stefan Müller<br/>TechVision GmbH</p>',
    folder_id: IDS.emailFolders.sent,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-001',
  },
  {
    id: 'em-s-002',
    subject: 'Terminvorschlag: Demo Helvetia Software',
    from: { name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' },
    to: [{ name: 'Hans Weber', email: 'h.weber@helvetia-software.ch' }],
    date: hoursAgo(4),
    snippet:
      'Hallo Herr Weber, gerne schlage ich folgende Termine für die Demo vor...',
    body_text:
      'Hallo Herr Weber,\n\ngerne schlage ich folgende Termine für die erweiterte Demo vor:\n\n- Dienstag, 10:00-12:00 Uhr\n- Mittwoch, 14:00-16:00 Uhr\n\nWir können Ihnen auch direkt eine Testumgebung einrichten.\n\nViele Grüße\nStefan Müller',
    body_html:
      '<p>Hallo Herr Weber,</p><p>gerne schlage ich folgende Termine vor:</p><ul><li>Dienstag, 10:00-12:00 Uhr</li><li>Mittwoch, 14:00-16:00 Uhr</li></ul><p>Wir können Ihnen auch direkt eine Testumgebung einrichten.</p><p>Viele Grüße<br/>Stefan Müller</p>',
    folder_id: IDS.emailFolders.sent,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-007',
  },
  {
    id: 'em-s-003',
    subject: 'RE: Bewerbung: Senior Go-Entwickler',
    from: { name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' },
    to: [{ name: 'Tobias Keller', email: 'tobias.keller@protonmail.com' }],
    date: hoursAgo(2),
    snippet:
      'Sehr geehrter Herr Keller, vielen Dank für Ihre Bewerbung. Wir wuerden Sie gerne...',
    body_text:
      'Sehr geehrter Herr Keller,\n\nvielen Dank für Ihre aussagekraeftige Bewerbung. Ihr Profil passt sehr gut zu unseren Anforderungen.\n\nWuerden Sie nächste Woche Zeit für ein erstes Videogespräch haben? Wir nutzen unser eigenes Tool (Cosmi Video) — Sie erhalten einen Link per Einladung.\n\nMit freundlichen Grüßen\nStefan Müller\nCTO, TechVision GmbH',
    body_html:
      '<p>Sehr geehrter Herr Keller,</p><p>vielen Dank für Ihre aussagekraeftige Bewerbung. Ihr Profil passt sehr gut zu unseren Anforderungen.</p><p>Wuerden Sie nächste Woche Zeit für ein erstes Videogespräch haben?</p><p>Mit freundlichen Grüßen<br/>Stefan Müller<br/>CTO, TechVision GmbH</p>',
    folder_id: IDS.emailFolders.sent,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-005',
  },
  {
    id: 'em-s-004',
    subject: 'Vertragsverlaengerung — Angebot Bavaria Elektro',
    from: { name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' },
    to: [{ name: 'Maria Huber', email: 'm.huber@bavaria-elektro.de' }],
    date: hoursAgo(6),
    snippet:
      'Sehr geehrte Frau Huber, vielen Dank für Ihr Vertrauen. Anbei das Verlaengerungsangebot...',
    body_text:
      'Sehr geehrte Frau Huber,\n\nvielen Dank für Ihr Vertrauen in Cosmi! Anbei das Verlaengerungsangebot inklusive der neuen Chat- und Video-Module.\n\nZusammenfassung:\n- Basis-Lizenz (25 User): EUR 12.000/Jahr\n- Chat & Video Add-On: EUR 3.600/Jahr\n- Support Premium: EUR 2.400/Jahr\n- Treuerabatt: -10%\n\nGesamtpreis: EUR 16.200/Jahr (exkl. MwSt.)\n\nMit freundlichen Grüßen\nStefan Müller',
    body_html:
      '<p>Sehr geehrte Frau Huber,</p><p>vielen Dank für Ihr Vertrauen in Cosmi! Anbei das Verlaengerungsangebot inklusive der neuen Chat- und Video-Module.</p><table><tr><td>Basis-Lizenz (25 User)</td><td>EUR 12.000/Jahr</td></tr><tr><td>Chat & Video</td><td>EUR 3.600/Jahr</td></tr><tr><td>Support Premium</td><td>EUR 2.400/Jahr</td></tr><tr><td>Treuerabatt</td><td>-10%</td></tr></table><p><strong>Gesamt: EUR 16.200/Jahr (exkl. MwSt.)</strong></p>',
    folder_id: IDS.emailFolders.sent,
    is_read: true,
    is_starred: false,
    has_attachments: true,
    attachments: [
      {
        id: 'att-007',
        filename: 'Angebot_BavariaElektro_Verlaengerung_2026.pdf',
        size: 185000,
        content_type: 'application/pdf',
      },
    ],
    thread_id: 'th-009',
  },
  {
    id: 'em-s-005',
    subject: 'Team-Update: Sprint 42 Zusammenfassung',
    from: { name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' },
    to: [
      { name: 'Team', email: 'team@techvision.de' },
    ],
    date: daysAgo(1),
    snippet:
      'Hi Team, hier die Zusammenfassung von Sprint 42: 28 von 32 Story Points erledigt...',
    body_text:
      'Hi Team,\n\nhier die Zusammenfassung von Sprint 42:\n\nErledigt: 28 von 32 Story Points\nHighlights:\n- Deployment-Pipeline mit Auto-Rollback\n- WebSocket Connection Indicator\n- 162 neue Unit Tests\n\nCarry-over:\n- E2E-Test-Stabilisierung (4 SP)\n\nGute Arbeit alle zusammen!\n\nStefan',
    body_html:
      '<p>Hi Team,</p><p>hier die Zusammenfassung von Sprint 42:</p><p><strong>Erledigt:</strong> 28 von 32 Story Points</p><p><strong>Highlights:</strong></p><ul><li>Deployment-Pipeline mit Auto-Rollback</li><li>WebSocket Connection Indicator</li><li>162 neue Unit Tests</li></ul>',
    folder_id: IDS.emailFolders.sent,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-021',
  },
]

const draftMessages = [
  {
    id: 'em-d-001',
    subject: 'Antwort: Partnerschaft Alpen Logistik',
    from: { name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' },
    to: [{ name: 'Lukas Steiner', email: 'l.steiner@alpen-logistik.at' }],
    date: hoursAgo(1),
    snippet:
      'Sehr geehrter Herr Steiner, vielen Dank für Ihr Interesse an einer Partnerschaft...',
    body_text:
      'Sehr geehrter Herr Steiner,\n\nvielen Dank für Ihr Interesse an einer Partnerschaft mit TechVision. Wir arbeiten gerade an einem Reseller-Programm und wuerden uns freuen, Sie als einen der ersten Partner...',
    body_html:
      '<p>Sehr geehrter Herr Steiner,</p><p>vielen Dank für Ihr Interesse an einer Partnerschaft mit TechVision. Wir arbeiten gerade an einem Reseller-Programm und wuerden uns freuen, Sie als einen der ersten Partner...</p>',
    folder_id: IDS.emailFolders.drafts,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: 'th-012',
  },
  {
    id: 'em-d-002',
    subject: 'Blog-Post: Warum EU-Hosting für KMUs unverzichtbar ist',
    from: { name: 'Stefan Müller', email: 'stefan.mueller@techvision.de' },
    to: [],
    date: daysAgo(2),
    snippet:
      'In einer zunehmend digitalisierten Welt stehen KMUs vor einer wichtigen Entscheidung...',
    body_text:
      'In einer zunehmend digitalisierten Welt stehen KMUs vor einer wichtigen Entscheidung: Wo werden ihre Daten gespeichert?\n\nDie Antwort sollte einfach sein: In der EU. Hier sind 5 Gruende warum...\n\n1. DSGVO-Konformitaet\n2. Datensouveraenitaet\n3. Latenz\n4. Vertrauen der Kunden\n5. Rechtssicherheit',
    body_html:
      '<p>In einer zunehmend digitalisierten Welt stehen KMUs vor einer wichtigen Entscheidung: Wo werden ihre Daten gespeichert?</p><p>Die Antwort sollte einfach sein: <strong>In der EU.</strong></p>',
    folder_id: IDS.emailFolders.drafts,
    is_read: true,
    is_starred: false,
    has_attachments: false,
    attachments: [],
    thread_id: null,
  },
]

export const mockEmailMessages: Record<
  string,
  { messages: Array<Record<string, unknown>>; total: number }
> = {
  [IDS.emailFolders.inbox]: { messages: inboxMessages, total: inboxMessages.length },
  [IDS.emailFolders.sent]: { messages: sentMessages, total: sentMessages.length },
  [IDS.emailFolders.drafts]: { messages: draftMessages, total: draftMessages.length },
  [IDS.emailFolders.trash]: { messages: [], total: 0 },
  [IDS.emailFolders.archive]: { messages: [], total: 0 },
}

// Flat lookup for single-message requests
const allMessages = [...inboxMessages, ...sentMessages, ...draftMessages]
export const mockEmailMessagesById: Record<string, Record<string, unknown>> =
  Object.fromEntries(allMessages.map((m) => [m.id, m]))

// ---------------------------------------------------------------------------
// Signatures
// ---------------------------------------------------------------------------

export const mockSignatures = {
  signatures: [
    {
      id: 'sig-001',
      name: 'Standard',
      html_content:
        '<p>Mit freundlichen Grüßen<br/><strong>Stefan Müller</strong><br/>CTO, TechVision GmbH<br/>stefan.mueller@techvision.de<br/>+49 89 123 456 78</p>',
      is_default: true,
    },
    {
      id: 'sig-002',
      name: 'Kurz',
      html_content:
        '<p>Viele Grüße<br/>Stefan Müller | TechVision</p>',
      is_default: false,
    },
  ],
}
