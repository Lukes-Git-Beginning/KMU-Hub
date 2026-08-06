-- Reverse of 000295. Options first: the composite FK cascades, but dropping in
-- dependency order keeps the intent readable.

BEGIN;

DROP TABLE IF EXISTS customization_value_set_options;
DROP TABLE IF EXISTS customization_value_sets;

COMMIT;
