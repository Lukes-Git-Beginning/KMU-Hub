-- fix-gdpr-anonymized-label-counter-race: GetNextAnonymizedLabel used COUNT(*)+1,
-- so two concurrent erasures read the same count and got the same label. A
-- sequence hands out each value exactly once. The label counter is global across
-- tenants by design (it always was); it is not a tenant table, so no RLS.
CREATE SEQUENCE gdpr_anonymized_label_seq;

-- Continue after the highest label already issued so historic labels never repeat.
DO $$
DECLARE
    highest bigint;
BEGIN
    SELECT COALESCE(MAX(substring(anonymized_label FROM '#(\d+)$')::bigint), 0)
      INTO highest
      FROM gdpr_erasure_log;
    IF highest > 0 THEN
        PERFORM setval('gdpr_anonymized_label_seq', highest);
    END IF;
END $$;
