-- Prod inbox seed — unified-inbox messages across channels (email/chat/notification)
-- plus a few canned responses, for echt-Schaltung review on Hetzner.
-- Idempotent via the UNIQUE (user_id, channel, source_id) index and canned PK.
--
-- inbox_messages is USER-scoped: user_id MUST be the reviewer, or the inbox stays
-- empty for them. Run as a DB superuser (bypasses RLS):
--   docker exec -i <pg> psql -U kmuhub -d kmuhub \
--     -v reviewer_id=<reviewer-user-uuid> \
--     -f - < backend/seeds/prod/inbox-seed.sql
-- Pass the UUID WITHOUT quotes — the script quotes it via :'reviewer_id' itself.
-- Resolve the reviewer with:  SELECT id, email FROM users;
-- tenant_id defaults to the bootstrap tenant; override with -v tenant_id=<uuid>.

\if :{?reviewer_id}
\else
  \echo '>>> ERROR: pass -v reviewer_id="''<user-uuid>''" (the logged-in reviewer).'
  \quit
\endif
\if :{?tenant_id}
\else
  \set tenant_id '00000000-0000-0000-0000-000000000001'
\endif

BEGIN;

INSERT INTO inbox_messages
  (id, user_id, channel, source_id, sender_name, sender_email, subject, preview, is_read, is_starred, tags, received_at, tenant_id)
VALUES
  ('b1000001-0000-0000-0000-000000000001', :'reviewer_id',
   'email','demo-msg-1','Sabine Brandt','sabine.brandt@kunde.example','Angebot 2026-0312 – Rückfrage',
   'Guten Tag, vielen Dank für das Angebot. Eine Frage zur Lieferzeit der Position 4 …',false,true,
   '{Vertrieb}', now() - interval '40 minutes', :'tenant_id'),
  ('b1000001-0000-0000-0000-000000000002', :'reviewer_id',
   'email','demo-msg-2','Thomas Keller','t.keller@lieferant.example','Lieferavis KW 27',
   'Anbei das Lieferavis für die kommende Woche. Bitte um kurze Bestätigung des Termins.',false,false,
   '{Einkauf}', now() - interval '2 hours', :'tenant_id'),
  ('b1000001-0000-0000-0000-000000000003', :'reviewer_id',
   'chat','demo-msg-3','Nico Weber',NULL,'Frage zum Sprint-Board',
   'Kannst du kurz auf das Ticket HD-2026 schauen? Der Kunde wartet auf Rückmeldung.',true,false,
   '{Intern}', now() - interval '5 hours', :'tenant_id'),
  ('b1000001-0000-0000-0000-000000000004', :'reviewer_id',
   'notification','demo-msg-4','System',NULL,'Rechnung 2026-0291 wurde bezahlt',
   'Der offene Betrag von 1.840,00 € wurde dem Konto gutgeschrieben.',true,false,
   '{Finanzen}', now() - interval '1 day', :'tenant_id'),
  ('b1000001-0000-0000-0000-000000000005', :'reviewer_id',
   'email','demo-msg-5','Petra Lindner','p.lindner@partner.example','Terminvorschlag Workshop',
   'Hätten Sie am Donnerstag um 10 Uhr Zeit für den Onboarding-Workshop?',false,false,
   '{Vertrieb}', now() - interval '3 hours', :'tenant_id'),
  ('b1000001-0000-0000-0000-000000000006', :'reviewer_id',
   'chat','demo-msg-6','Sabine Brandt',NULL,'Re: Angebot 2026-0312',
   'Perfekt, dann warten wir auf die aktualisierte Version. Danke!',true,false,
   '{Vertrieb}', now() - interval '2 days', :'tenant_id')
ON CONFLICT (user_id, channel, source_id) DO UPDATE SET
  subject = EXCLUDED.subject, preview = EXCLUDED.preview, is_read = EXCLUDED.is_read,
  is_starred = EXCLUDED.is_starred, received_at = EXCLUDED.received_at, updated_at = now();

-- Canned responses (tenant-scoped reply templates).
INSERT INTO inbox_canned_responses (id, tenant_id, name, body)
VALUES
  ('c1a00001-0000-0000-0000-000000000001', :'tenant_id', 'Eingangsbestätigung',
   'Vielen Dank für Ihre Nachricht. Wir haben Ihr Anliegen erhalten und melden uns zeitnah bei Ihnen zurück.'),
  ('c1a00001-0000-0000-0000-000000000002', :'tenant_id', 'Rückfrage Unterlagen',
   'Könnten Sie uns zur weiteren Bearbeitung bitte die zugehörigen Unterlagen zukommen lassen? Vielen Dank.'),
  ('c1a00001-0000-0000-0000-000000000003', :'tenant_id', 'Abschluss',
   'Wir betrachten Ihr Anliegen damit als erledigt. Bei weiteren Fragen stehen wir gerne zur Verfügung.')
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name, body = EXCLUDED.body, updated_at = now();

COMMIT;

SELECT 'inbox_messages' AS k, count(*) AS v FROM inbox_messages WHERE user_id = :'reviewer_id'
UNION ALL SELECT 'unread', count(*) FROM inbox_messages WHERE user_id = :'reviewer_id' AND is_read=false
UNION ALL SELECT 'canned', count(*) FROM inbox_canned_responses WHERE tenant_id = :'tenant_id';
