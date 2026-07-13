BEGIN;

DROP TRIGGER IF EXISTS trg_email_logs_updated_at ON email_logs;
DROP TABLE IF EXISTS email_logs;

COMMIT;