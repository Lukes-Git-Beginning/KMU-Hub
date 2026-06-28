-- Helpdesk demo seed (tenant …0001) — populates tickets + messages in varied
-- states/priorities for echt-Schaltung QA (list filters, SLA badges, detail view).
-- Idempotent: clears prior demo rows, then re-inserts. due_at relative to now() so
-- the overdue SLA badge always shows. requester/assignee = the two demo users.
-- NOTE: ticket IDs use DIVERSE first-4-hex prefixes on purpose — the FE derives the
-- display ticket number (HD-YYYY-NNNN) from the first 4 hex chars of the UUID, so
-- same-prefix IDs would all render the same number. (Real tickets get random UUIDs.)
BEGIN;

DELETE FROM ticket_messages WHERE id::text LIKE '11d00001-%' OR ticket_id::text IN
  ('1ac00001-0000-0000-0000-000000000001','2bc00002-0000-0000-0000-000000000002',
   '3cc00003-0000-0000-0000-000000000003','4dc00004-0000-0000-0000-000000000004',
   '5ec00005-0000-0000-0000-000000000005','6fc00006-0000-0000-0000-000000000006');
DELETE FROM tickets WHERE id::text LIKE '11c00001-%';

-- u_demo  = demo@local.test (admin, login user) = 6593a043-…
-- u_review = demo-review@zentria.tech           = 11111111-…
INSERT INTO tickets
  (id, tenant_id, subject, status, priority, assignee_id, requester_id, due_at, first_response_at, resolved_at, created_at, updated_at)
VALUES
  ('1ac00001-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001',
   'Drucker im 2. OG druckt nur leere Seiten','open','high',
   '6593a043-bc5b-49d1-8531-0c6ade16b3ad','11111111-1111-1111-1111-111111111111',
   now() - interval '2 hours', now() - interval '20 hours', NULL, now() - interval '1 day', now()),
  ('2bc00002-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001',
   'VPN-Zugang für neuen Mitarbeiter einrichten','open','urgent',
   NULL,'11111111-1111-1111-1111-111111111111',
   now() + interval '3 hours', NULL, NULL, now() - interval '5 hours', now()),
  ('3cc00003-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000001',
   'Rechnung März fehlt im Kundenportal','pending','normal',
   '6593a043-bc5b-49d1-8531-0c6ade16b3ad','11111111-1111-1111-1111-111111111111',
   now() + interval '2 days', now() - interval '6 hours', NULL, now() - interval '2 days', now()),
  ('4dc00004-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000001',
   'Passwort-Zurücksetzung Outlook','solved','low',
   '6593a043-bc5b-49d1-8531-0c6ade16b3ad','11111111-1111-1111-1111-111111111111',
   NULL, now() - interval '3 days', now() - interval '2 days', now() - interval '3 days', now()),
  ('5ec00005-0000-0000-0000-000000000005','00000000-0000-0000-0000-000000000001',
   'Onboarding-Laptop bestellen','closed','normal',
   '11111111-1111-1111-1111-111111111111','6593a043-bc5b-49d1-8531-0c6ade16b3ad',
   NULL, now() - interval '8 days', now() - interval '6 days', now() - interval '8 days', now()),
  ('6fc00006-0000-0000-0000-000000000006','00000000-0000-0000-0000-000000000001',
   'Monitor flackert sporadisch','open','normal',
   NULL,'6593a043-bc5b-49d1-8531-0c6ade16b3ad',
   now() + interval '1 day', NULL, NULL, now() - interval '3 hours', now());

INSERT INTO ticket_messages (id, ticket_id, author_id, body, internal, tenant_id, created_at)
VALUES
  ('a1d00001-0000-0000-0000-000000000001','1ac00001-0000-0000-0000-000000000001',
   '11111111-1111-1111-1111-111111111111','Der Drucker zieht das Papier ein, gibt es aber komplett leer wieder aus. Toner ist neu.',false,'00000000-0000-0000-0000-000000000001', now() - interval '1 day'),
  ('a1d00001-0000-0000-0000-000000000002','1ac00001-0000-0000-0000-000000000001',
   '6593a043-bc5b-49d1-8531-0c6ade16b3ad','Bitte einmal Tonereinheit entnehmen und den Schutzstreifen prüfen — ich komme gegen 14 Uhr vorbei.',false,'00000000-0000-0000-0000-000000000001', now() - interval '20 hours'),
  ('a3d00003-0000-0000-0000-000000000003','3cc00003-0000-0000-0000-000000000003',
   '11111111-1111-1111-1111-111111111111','Kunde meldet, dass die Rechnung 2026-03 nicht zum Download bereitsteht.',false,'00000000-0000-0000-0000-000000000001', now() - interval '2 days'),
  ('a3d00003-0000-0000-0000-000000000004','3cc00003-0000-0000-0000-000000000003',
   '6593a043-bc5b-49d1-8531-0c6ade16b3ad','Intern: Sync-Job für das Portal hing, neu angestoßen. Warte auf Bestätigung.',true,'00000000-0000-0000-0000-000000000001', now() - interval '6 hours');

COMMIT;

SELECT 'tickets' AS k, count(*) AS v FROM tickets WHERE tenant_id='00000000-0000-0000-0000-000000000001'
UNION ALL SELECT 'open', count(*) FROM tickets WHERE tenant_id='00000000-0000-0000-0000-000000000001' AND status='open'
UNION ALL SELECT 'messages', count(*) FROM ticket_messages WHERE tenant_id='00000000-0000-0000-0000-000000000001';
