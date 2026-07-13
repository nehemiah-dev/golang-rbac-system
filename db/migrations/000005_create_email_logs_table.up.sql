BEGIN;

CREATE TABLE email_logs (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    template       VARCHAR(255) NOT NULL,
    status         email_status NOT NULL DEFAULT 'pending',
    delivery_meta  JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_email_logs_user_id ON email_logs (user_id);
CREATE INDEX idx_email_logs_status ON email_logs (status);

CREATE TRIGGER trg_email_logs_updated_at
BEFORE UPDATE ON email_logs
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMIT;
