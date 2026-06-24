-- Dialer demo seed (tenant …0001) — populates the Supervisor dashboard + contact
-- call history so the real backend renders non-empty during mock-exit verification.
-- Idempotent: ON CONFLICT refreshes time columns to "today" so calls_today /
-- agent-active-today stay truthful on re-runs. Agent = demo@local.test (login user).
BEGIN;

-- Call outcomes (one appointment, one positive, one negative) ----------------
INSERT INTO dialer_call_outcomes (id, tenant_id, label, color, is_positive, is_callback, is_appointment, sort_order, is_active, created_at) VALUES
  ('d0000001-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001','Termin vereinbart','#16a34a', true,  false, true,  1, true, now()),
  ('d0000001-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','Interesse',        '#2563eb', true,  false, false, 2, true, now()),
  ('d0000001-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000001','Kein Interesse',   '#dc2626', false, false, false, 3, true, now())
ON CONFLICT (id) DO NOTHING;

-- Campaign --------------------------------------------------------------------
INSERT INTO dialer_campaigns (id, tenant_id, name, description, status, mode, settings, created_by, assigned_agent_ids, contact_count, completed_count, created_at, updated_at, started_at) VALUES
  ('d0000002-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001','Q3 Nachfass-Kampagne','Nachfassen offener Angebote aus Q2','active','preview','{}',
   '6593a043-bc5b-49d1-8531-0c6ade16b3ad',
   ARRAY['6593a043-bc5b-49d1-8531-0c6ade16b3ad','11111111-1111-1111-1111-111111111111']::uuid[],
   4, 3, now(), now(), now())
ON CONFLICT (id) DO NOTHING;

-- Campaign contacts (denormalized contact_name drives the recent-calls feed) --
INSERT INTO dialer_campaign_contacts (id, campaign_id, contact_id, position, status, outcome_id, call_count, last_called_at, created_at, updated_at, tenant_id) VALUES
  ('d0000003-0000-0000-0000-000000000001','d0000002-0000-0000-0000-000000000001','33333333-0000-0000-0000-000000000001',1,'completed','d0000001-0000-0000-0000-000000000001',2, now(), now(), now(),'00000000-0000-0000-0000-000000000001'),
  ('d0000003-0000-0000-0000-000000000002','d0000002-0000-0000-0000-000000000001','33333333-0000-0000-0000-000000000002',2,'completed','d0000001-0000-0000-0000-000000000002',1, now(), now(), now(),'00000000-0000-0000-0000-000000000001'),
  ('d0000003-0000-0000-0000-000000000003','d0000002-0000-0000-0000-000000000001','33333333-0000-0000-0000-000000000003',3,'completed','d0000001-0000-0000-0000-000000000003',1, now(), now(), now(),'00000000-0000-0000-0000-000000000001'),
  ('d0000003-0000-0000-0000-000000000004','d0000002-0000-0000-0000-000000000001','33333333-0000-0000-0000-000000000004',4,'pending',  NULL,                                  1, now(), now(), now(),'00000000-0000-0000-0000-000000000001')
ON CONFLICT (id) DO NOTHING;

-- The campaign contacts get the denormalized contact name from crm.
UPDATE dialer_campaign_contacts cc
SET notes = COALESCE(cc.notes, '')
WHERE cc.campaign_id = 'd0000002-0000-0000-0000-000000000001';

-- Call sessions (TODAY) — refresh created_at to now() on re-run so they stay "today".
INSERT INTO dialer_call_sessions (id, campaign_contact_id, agent_id, outcome_id, duration_seconds, notes, created_at, updated_at, tenant_id) VALUES
  ('d0000004-0000-0000-0000-000000000001','d0000003-0000-0000-0000-000000000001','6593a043-bc5b-49d1-8531-0c6ade16b3ad','d0000001-0000-0000-0000-000000000001',245,'Termin für nächste Woche fixiert', now(), now(),'00000000-0000-0000-0000-000000000001'),
  ('d0000004-0000-0000-0000-000000000002','d0000003-0000-0000-0000-000000000002','6593a043-bc5b-49d1-8531-0c6ade16b3ad','d0000001-0000-0000-0000-000000000002',132,'Rückruf in 2 Wochen gewünscht', now(), now(),'00000000-0000-0000-0000-000000000001'),
  ('d0000004-0000-0000-0000-000000000003','d0000003-0000-0000-0000-000000000003','11111111-1111-1111-1111-111111111111','d0000001-0000-0000-0000-000000000003', 48,'Kein Bedarf aktuell', now(), now(),'00000000-0000-0000-0000-000000000001'),
  ('d0000004-0000-0000-0000-000000000004','d0000003-0000-0000-0000-000000000004','6593a043-bc5b-49d1-8531-0c6ade16b3ad','d0000001-0000-0000-0000-000000000001',310,'Zweittermin vereinbart', now(), now(),'00000000-0000-0000-0000-000000000001'),
  ('d0000004-0000-0000-0000-000000000005','d0000003-0000-0000-0000-000000000001','11111111-1111-1111-1111-111111111111','d0000001-0000-0000-0000-000000000002', 95,'Erstkontakt, Interesse vorhanden', now(), now(),'00000000-0000-0000-0000-000000000001')
ON CONFLICT (id) DO UPDATE SET created_at = now(), updated_at = now();

-- Agent status log (TODAY) — drives GetActiveAgentIDsForTenant (DISTINCT user_id).
INSERT INTO dialer_agent_status_log (id, user_id, campaign_id, status, changed_at, tenant_id) VALUES
  ('d0000005-0000-0000-0000-000000000001','6593a043-bc5b-49d1-8531-0c6ade16b3ad','d0000002-0000-0000-0000-000000000001','available', now(),'00000000-0000-0000-0000-000000000001'),
  ('d0000005-0000-0000-0000-000000000002','11111111-1111-1111-1111-111111111111','d0000002-0000-0000-0000-000000000001','on_call',   now(),'00000000-0000-0000-0000-000000000001')
ON CONFLICT (id) DO UPDATE SET changed_at = now();

COMMIT;

-- Verification helper output
SELECT 'calls_today' AS k, COUNT(*) AS v FROM dialer_call_sessions WHERE tenant_id='00000000-0000-0000-0000-000000000001' AND created_at >= CURRENT_DATE
UNION ALL SELECT 'appointments_today', COUNT(s.id) FROM dialer_call_sessions s JOIN dialer_call_outcomes o ON o.id=s.outcome_id WHERE s.tenant_id='00000000-0000-0000-0000-000000000001' AND s.created_at>=CURRENT_DATE AND o.is_appointment
UNION ALL SELECT 'active_agents_today', COUNT(DISTINCT user_id) FROM dialer_agent_status_log WHERE tenant_id='00000000-0000-0000-0000-000000000001' AND changed_at>=CURRENT_DATE;
