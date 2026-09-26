SET @previous_mobile_view_default = (
  SELECT COLUMN_DEFAULT FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'group_members'
    AND COLUMN_NAME = 'mobile_view_mode'
);
SET @reset_mobile_view = IF(
  @previous_mobile_view_default = 'masonry',
  'UPDATE group_members SET mobile_view_mode = ''stacked'' WHERE mobile_view_mode = ''masonry''',
  'SELECT 1'
);
PREPARE reset_mobile_view FROM @reset_mobile_view;
EXECUTE reset_mobile_view;
DEALLOCATE PREPARE reset_mobile_view;
SET @set_mobile_view_default = IF(
  @previous_mobile_view_default = 'stacked',
  'SELECT 1',
  'ALTER TABLE group_members ALTER COLUMN mobile_view_mode SET DEFAULT ''stacked'''
);
PREPARE set_mobile_view_default FROM @set_mobile_view_default;
EXECUTE set_mobile_view_default;
DEALLOCATE PREPARE set_mobile_view_default;
