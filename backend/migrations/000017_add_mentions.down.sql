DROP TABLE IF EXISTS message_mentions;
DROP TYPE IF EXISTS mention_type;
DELETE FROM permissions WHERE resource = 'mentions' AND action = 'read';
