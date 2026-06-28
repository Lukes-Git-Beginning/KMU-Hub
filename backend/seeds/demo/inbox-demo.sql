-- Inbox demo seed (tenant …0001, user demo@local.test) — populates the unified
-- inbox with messages across channels (email/chat/notification) in varied states
-- (unread/read/starred) for kommunikation echt-Schaltung QA. Idempotent via the
-- UNIQUE (user_id, channel, source_id) index → ON CONFLICT refreshes.
BEGIN;

-- user = demo@local.test = 6593a043-bc5b-49d1-8531-0c6ade16b3ad
INSERT INTO inbox_messages
  (id, user_id, channel, source_id, sender_name, sender_email, subject, preview, is_read, is_starred, tags, received_at, tenant_id)
VALUES
  ('b1000001-0000-0000-0000-000000000001','6593a043-bc5b-49d1-8531-0c6ade16b3ad',
   'email','demo-msg-1','Sabine Brandt','sabine.brandt@kunde.example','Angebot 2026-0312 – Rückfrage',
   'Guten Tag, vielen Dank für das Angebot. Eine Frage zur Lieferzeit der Position 4 …',false,true,
   '{Vertrieb}', now() - interval '40 minutes','00000000-0000-0000-0000-000000000001'),
  ('b1000001-0000-0000-0000-000000000002','6593a043-bc5b-49d1-8531-0c6ade16b3ad',
   'email','demo-msg-2','Thomas Keller','t.keller@lieferant.example','Lieferavis KW 27',
   'Anbei das Lieferavis für die kommende Woche. Bitte um kurze Bestätigung des Termins.',false,false,
   '{Einkauf}', now() - interval '2 hours','00000000-0000-0000-0000-000000000001'),
  ('b1000001-0000-0000-0000-000000000003','6593a043-bc5b-49d1-8531-0c6ade16b3ad',
   'chat','demo-msg-3','Nico Weber',NULL,'Frage zum Sprint-Board',
   'Kannst du kurz auf das Ticket HD-2026 schauen? Der Kunde wartet auf Rückmeldung.',true,false,
   '{Intern}', now() - interval '5 hours','00000000-0000-0000-0000-000000000001'),
  ('b1000001-0000-0000-0000-000000000004','6593a043-bc5b-49d1-8531-0c6ade16b3ad',
   'notification','demo-msg-4','System',NULL,'Rechnung 2026-0291 wurde bezahlt',
   'Der offene Betrag von 1.840,00 € wurde dem Konto gutgeschrieben.',true,false,
   '{Finanzen}', now() - interval '1 day','00000000-0000-0000-0000-000000000001'),
  ('b1000001-0000-0000-0000-000000000005','6593a043-bc5b-49d1-8531-0c6ade16b3ad',
   'email','demo-msg-5','Petra Lindner','p.lindner@partner.example','Terminvorschlag Workshop',
   'Hätten Sie am Donnerstag um 10 Uhr Zeit für den Onboarding-Workshop?',false,false,
   '{Vertrieb}', now() - interval '3 hours','00000000-0000-0000-0000-000000000001'),
  ('b1000001-0000-0000-0000-000000000006','6593a043-bc5b-49d1-8531-0c6ade16b3ad',
   'chat','demo-msg-6','Sabine Brandt',NULL,'Re: Angebot 2026-0312',
   'Perfekt, dann warten wir auf die aktualisierte Version. Danke!',true,false,
   '{Vertrieb}', now() - interval '2 days','00000000-0000-0000-0000-000000000001')
ON CONFLICT (user_id, channel, source_id) DO UPDATE SET
  subject = EXCLUDED.subject, preview = EXCLUDED.preview, is_read = EXCLUDED.is_read,
  is_starred = EXCLUDED.is_starred, received_at = EXCLUDED.received_at, updated_at = now();

COMMIT;

SELECT 'inbox_messages' AS k, count(*) AS v FROM inbox_messages WHERE user_id='6593a043-bc5b-49d1-8531-0c6ade16b3ad'
UNION ALL SELECT 'unread', count(*) FROM inbox_messages WHERE user_id='6593a043-bc5b-49d1-8531-0c6ade16b3ad' AND is_read=false;
