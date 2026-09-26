SET @add_mobile_view_mode = IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE()
     AND TABLE_NAME = 'group_members'
     AND COLUMN_NAME = 'mobile_view_mode') = 0,
  'ALTER TABLE group_members ADD COLUMN mobile_view_mode VARCHAR(16) NOT NULL DEFAULT ''masonry'' AFTER member_name',
  'SELECT 1'
);
PREPARE add_mobile_view_mode FROM @add_mobile_view_mode;
EXECUTE add_mobile_view_mode;
DEALLOCATE PREPARE add_mobile_view_mode;
