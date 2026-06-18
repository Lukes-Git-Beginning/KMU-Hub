BEGIN;
DELETE FROM permissions WHERE name IN (
    'produktion:bom:read','produktion:bom:write',
    'produktion:machine:read','produktion:machine:write',
    'produktion:quality:read','produktion:quality:write',
    'produktion:workstep:read','produktion:workstep:write'
);
COMMIT;
