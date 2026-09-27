ALTER TABLE ministry_group_requests
  ADD COLUMN submission_round BIGINT UNSIGNED NOT NULL DEFAULT 1;

ALTER TABLE ministry_shares
  ADD COLUMN submission_round BIGINT UNSIGNED NOT NULL DEFAULT 1;
