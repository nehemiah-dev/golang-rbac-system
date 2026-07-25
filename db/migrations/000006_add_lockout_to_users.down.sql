BEGIN;

ALTER TABLE users
    DROP COLUMN failed_attempts,
    DROP COLUMN locked_until;

COMMIT;
