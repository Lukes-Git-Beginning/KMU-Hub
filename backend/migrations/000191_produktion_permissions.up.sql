-- 000191 Produktion erweiterte Permissions (BOMs, Maschinen, QualityChecks, WorkSteps)

BEGIN;

INSERT INTO permissions (name, resource, action) VALUES
    ('produktion:bom:read',       'produktion:bom',      'read'),
    ('produktion:bom:write',      'produktion:bom',      'write'),
    ('produktion:machine:read',   'produktion:machine',  'read'),
    ('produktion:machine:write',  'produktion:machine',  'write'),
    ('produktion:quality:read',   'produktion:quality',  'read'),
    ('produktion:quality:write',  'produktion:quality',  'write'),
    ('produktion:workstep:read',  'produktion:workstep', 'read'),
    ('produktion:workstep:write', 'produktion:workstep', 'write')
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
