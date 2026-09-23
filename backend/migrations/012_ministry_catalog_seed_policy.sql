SET @add_auto_seed_ministry_catalog = IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'study_groups'
     AND COLUMN_NAME = 'auto_seed_ministry_catalog') = 0,
  'ALTER TABLE study_groups ADD COLUMN auto_seed_ministry_catalog TINYINT NOT NULL DEFAULT 1',
  'SELECT 1'
);
PREPARE add_auto_seed_ministry_catalog FROM @add_auto_seed_ministry_catalog;
EXECUTE add_auto_seed_ministry_catalog;
DEALLOCATE PREPARE add_auto_seed_ministry_catalog;
