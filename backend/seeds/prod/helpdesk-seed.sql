-- Prod helpdesk seed — tickets (with description/category/ticket_number from
-- migration 000236) in varied states for echt-Schaltung review on Hetzner.
-- Idempotent: deletes the known seed rows by fixed UUID, then re-inserts with
-- due_at relative to now() so SLA badges stay fresh.
--
-- Run as a DB superuser (bypasses RLS), parameterized with the reviewer user:
--   docker exec -i <pg> psql -U kmuhub -d kmuhub \
--     -v reviewer_id="'<reviewer-user-uuid>'" \
--     -f - < backend/seeds/prod/helpdesk-seed.sql
-- Resolve the reviewer with:  SELECT id, email FROM users;
-- tenant_id defaults to the bootstrap tenant; override with -v tenant_id="'<uuid>'".

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

DELETE FROM ticket_messages WHERE ticket_id::text IN
  ('1ac00001-0000-0000-0000-000000000001','2bc00002-0000-0000-0000-000000000002',
   '3cc00003-0000-0000-0000-000000000003','4dc00004-0000-0000-0000-000000000004',
   '5ec00005-0000-0000-0000-000000000005','6fc00006-0000-0000-0000-000000000006');
DELETE FROM tickets WHERE id::text LIKE '_ac00001-%' OR id::text IN
  ('2bc00002-0000-0000-0000-000000000002','3cc00003-0000-0000-0000-000000000003',
   '4dc00004-0000-0000-0000-000000000004','5ec00005-0000-0000-0000-000000000005',
   '6fc00006-0000-0000-0000-000000000006');

INSERT INTO tickets
  (id, tenant_id, subject, status, priority, assignee_id, requester_id,
   description, category, ticket_number,
   due_at, first_response_at, resolved_at, created_at, updated_at)
VALUES
  ('1ac00001-0000-0000-0000-000000000001', :'tenant_id',
   'Drucker im 2. OG druckt nur leere Seiten','open','high',
   :'reviewer_id', :'reviewer_id',
   'Der Drucker zieht das Papier ein, gibt es aber komplett leer wieder aus. Toner ist neu.','Hardware',1,
   now() - interval '2 hours', now() - interval '20 hours', NULL, now() - interval '1 day', now()),
  ('2bc00002-0000-0000-0000-000000000002', :'tenant_id',
   'VPN-Zugang für neuen Mitarbeiter einrichten','open','urgent',
   NULL, :'reviewer_id',
   'Bitte VPN-Profil für die neue Kollegin im Vertrieb anlegen, Start am Montag.','Zugriff',2,
   now() + interval '3 hours', NULL, NULL, now() - interval '5 hours', now()),
  ('3cc00003-0000-0000-0000-000000000003', :'tenant_id',
   'Rechnung März fehlt im Kundenportal','pending','normal',
   :'reviewer_id', :'reviewer_id',
   'Kunde meldet, dass die Rechnung 2026-03 nicht zum Download bereitsteht.','Abrechnung',3,
   now() + interval '2 days', now() - interval '6 hours', NULL, now() - interval '2 days', now()),
  ('4dc00004-0000-0000-0000-000000000004', :'tenant_id',
   'Passwort-Zurücksetzung Outlook','solved','low',
   :'reviewer_id', :'reviewer_id',
   'Outlook fragt nach erneuter Anmeldung, Passwort wurde zurückgesetzt.','Zugriff',4,
   NULL, now() - interval '3 days', now() - interval '2 days', now() - interval '3 days', now()),
  ('5ec00005-0000-0000-0000-000000000005', :'tenant_id',
   'Onboarding-Laptop bestellen','closed','normal',
   :'reviewer_id', :'reviewer_id',
   'Standard-Laptop für Onboarding bestellt, Lieferung bestätigt.','Beschaffung',5,
   NULL, now() - interval '8 days', now() - interval '6 days', now() - interval '8 days', now()),
  ('6fc00006-0000-0000-0000-000000000006', :'tenant_id',
   'Monitor flackert sporadisch','open','normal',
   NULL, :'reviewer_id',
   'Der zweite Monitor flackert unregelmäßig, Kabeltausch hat nicht geholfen.','Hardware',6,
   now() + interval '1 day', NULL, NULL, now() - interval '3 hours', now());

INSERT INTO ticket_messages (id, ticket_id, author_id, body, internal, tenant_id, created_at)
VALUES
  ('a1d00001-0000-0000-0000-000000000001','1ac00001-0000-0000-0000-000000000001',
   :'reviewer_id','Der Drucker zieht das Papier ein, gibt es aber komplett leer wieder aus. Toner ist neu.',false, :'tenant_id', now() - interval '1 day'),
  ('a1d00001-0000-0000-0000-000000000002','1ac00001-0000-0000-0000-000000000001',
   :'reviewer_id','Bitte einmal Tonereinheit entnehmen und den Schutzstreifen prüfen — ich komme gegen 14 Uhr vorbei.',false, :'tenant_id', now() - interval '20 hours'),
  ('a3d00003-0000-0000-0000-000000000003','3cc00003-0000-0000-0000-000000000003',
   :'reviewer_id','Kunde meldet, dass die Rechnung 2026-03 nicht zum Download bereitsteht.',false, :'tenant_id', now() - interval '2 days'),
  ('a3d00003-0000-0000-0000-000000000004','3cc00003-0000-0000-0000-000000000003',
   :'reviewer_id','Intern: Sync-Job für das Portal hing, neu angestoßen. Warte auf Bestätigung.',true, :'tenant_id', now() - interval '6 hours');

-- Keep the per-tenant ticket counter ahead of the seeded numbers so app-created
-- tickets continue the sequence without colliding.
INSERT INTO helpdesk_ticket_counters (tenant_id, next_number)
VALUES (:'tenant_id', 7)
ON CONFLICT (tenant_id) DO UPDATE SET next_number = GREATEST(helpdesk_ticket_counters.next_number, 7);

COMMIT;

SELECT 'tickets' AS k, count(*) AS v FROM tickets WHERE tenant_id = :'tenant_id'
UNION ALL SELECT 'open', count(*) FROM tickets WHERE tenant_id = :'tenant_id' AND status='open'
UNION ALL SELECT 'messages', count(*) FROM ticket_messages WHERE tenant_id = :'tenant_id';
