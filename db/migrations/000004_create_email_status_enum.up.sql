BEGIN;

CREATE TYPE email_status AS ENUM ('pending', 'sent', 'failed');

COMMIT;
