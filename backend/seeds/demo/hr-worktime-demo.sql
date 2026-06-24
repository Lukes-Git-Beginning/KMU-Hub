-- HR work-time demo seed (tenant …0001, employee = demo@local.test) — populates
-- zeiterfassung (entries, balance, week status) for mock-exit verification.
-- Idempotent + week-relative: ON CONFLICT refreshes clock-in/out to the CURRENT
-- week so the entries always count toward "this week" on re-run.
BEGIN;

INSERT INTO hr_work_time_entries
  (id, tenant_id, employee_id, clock_in, clock_out, break_minutes, auto_break_deducted, net_work_minutes, status, is_correction, created_at, updated_at)
VALUES
  ('e0000001-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001','6593a043-bc5b-49d1-8531-0c6ade16b3ad',
    date_trunc('week', now()) + interval '8 hours',          date_trunc('week', now()) + interval '16 hours 30 minutes', 30, 0, 480, 'completed', false, now(), now()),
  ('e0000001-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','6593a043-bc5b-49d1-8531-0c6ade16b3ad',
    date_trunc('week', now()) + interval '1 day 8 hours',     date_trunc('week', now()) + interval '1 day 16 hours 30 minutes', 30, 0, 480, 'completed', false, now(), now()),
  ('e0000001-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000001','6593a043-bc5b-49d1-8531-0c6ade16b3ad',
    date_trunc('week', now()) + interval '2 days 8 hours',    date_trunc('week', now()) + interval '2 days 16 hours 15 minutes', 30, 0, 465, 'completed', false, now(), now())
ON CONFLICT (id) DO UPDATE SET
  clock_in = EXCLUDED.clock_in, clock_out = EXCLUDED.clock_out,
  net_work_minutes = EXCLUDED.net_work_minutes, updated_at = now();

COMMIT;

SELECT 'entries_this_week' AS k, COUNT(*) AS v FROM hr_work_time_entries
 WHERE tenant_id='00000000-0000-0000-0000-000000000001' AND employee_id='6593a043-bc5b-49d1-8531-0c6ade16b3ad' AND clock_in >= date_trunc('week', now())
UNION ALL SELECT 'net_minutes_sum', COALESCE(SUM(net_work_minutes),0) FROM hr_work_time_entries
 WHERE tenant_id='00000000-0000-0000-0000-000000000001' AND employee_id='6593a043-bc5b-49d1-8531-0c6ade16b3ad' AND clock_in >= date_trunc('week', now());
