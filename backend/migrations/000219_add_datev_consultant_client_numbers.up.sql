-- R3-7c-4: the DATEV EXTF Buchungsstapel header left Beraternummer and
-- Mandantennummer empty, so DATEV could not assign the imported batch to a
-- client. Persist both on company_settings (tenant-level), read by the DATEV
-- exporter. Plain nullable-with-default column additions — no RLS change
-- (tenant_id untouched). Empty default keeps existing behaviour until configured.
ALTER TABLE company_settings ADD COLUMN datev_berater_nr TEXT NOT NULL DEFAULT '';
ALTER TABLE company_settings ADD COLUMN datev_mandant_nr TEXT NOT NULL DEFAULT '';
