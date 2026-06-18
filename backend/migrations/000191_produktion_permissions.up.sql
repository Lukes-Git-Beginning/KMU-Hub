-- 000191 Produktion erweiterte Permissions (BOMs, Maschinen, QualityChecks, WorkSteps)

BEGIN;

INSERT INTO permissions (name, description) VALUES
    ('produktion:bom:read',          'Stücklisten lesen'),
    ('produktion:bom:write',         'Stücklisten erstellen/bearbeiten'),
    ('produktion:machine:read',      'Maschinen lesen'),
    ('produktion:machine:write',     'Maschinen erstellen/bearbeiten'),
    ('produktion:quality:read',      'Qualitätsprüfungen lesen'),
    ('produktion:quality:write',     'Qualitätsprüfungen erstellen'),
    ('produktion:workstep:read',     'Arbeitsschritte lesen'),
    ('produktion:workstep:write',    'Arbeitsschritte erstellen/bearbeiten')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
  AND p.name IN (
    'produktion:bom:read','produktion:bom:write',
    'produktion:machine:read','produktion:machine:write',
    'produktion:quality:read','produktion:quality:write',
    'produktion:workstep:read','produktion:workstep:write'
  )
ON CONFLICT DO NOTHING;

COMMIT;
