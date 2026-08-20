-- atlas:txmode file

CREATE TYPE internal_user_role AS ENUM ('ADMIN', 'OPERATOR', 'REVIEWER');
CREATE TYPE internal_user_status AS ENUM ('ACTIVE', 'DISABLED');
CREATE TYPE idempotency_status AS ENUM ('PROCESSING', 'COMPLETED', 'FAILED');

CREATE TABLE internal_users (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    email text NOT NULL,
    display_name text NOT NULL,
    password_hash text NOT NULL,
    role internal_user_role NOT NULL,
    status internal_user_status NOT NULL DEFAULT 'ACTIVE',
    requires_password_change boolean NOT NULL DEFAULT true,
    failed_login_attempts integer NOT NULL DEFAULT 0 CHECK (failed_login_attempts >= 0),
    locked_until timestamptz,
    last_login_at timestamptz,
    password_changed_at timestamptz NOT NULL DEFAULT now(),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT internal_users_email_length CHECK (length(email) BETWEEN 3 AND 320),
    CONSTRAINT internal_users_display_name_length CHECK (length(display_name) BETWEEN 2 AND 120)
);
CREATE UNIQUE INDEX internal_users_email_unique ON internal_users (lower(email));
CREATE INDEX internal_users_status_role_idx ON internal_users (status, role);

CREATE TABLE sessions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    internal_user_id uuid NOT NULL REFERENCES internal_users(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    csrf_hash bytea NOT NULL,
    ip_address inet,
    user_agent text NOT NULL DEFAULT '',
    expires_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    revoke_reason text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sessions_expiry_after_creation CHECK (expires_at > created_at),
    CONSTRAINT sessions_user_agent_length CHECK (length(user_agent) <= 1000)
);
CREATE INDEX sessions_user_active_idx ON sessions (internal_user_id, expires_at) WHERE revoked_at IS NULL;
CREATE INDEX sessions_expiry_idx ON sessions (expires_at) WHERE revoked_at IS NULL;

CREATE TABLE idempotency_keys (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    namespace text NOT NULL,
    key_hash bytea NOT NULL,
    request_hash bytea NOT NULL,
    status idempotency_status NOT NULL DEFAULT 'PROCESSING',
    response_status integer,
    response_body jsonb,
    locked_until timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    CONSTRAINT idempotency_keys_namespace_length CHECK (length(namespace) BETWEEN 1 AND 120),
    CONSTRAINT idempotency_keys_response_status CHECK (response_status IS NULL OR response_status BETWEEN 100 AND 599),
    UNIQUE (namespace, key_hash)
);
CREATE INDEX idempotency_keys_expiry_idx ON idempotency_keys (expires_at);

CREATE TABLE audit_logs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    actor_internal_user_id uuid REFERENCES internal_users(id) ON DELETE SET NULL,
    action text NOT NULL,
    entity_type text,
    entity_id uuid,
    client_id uuid,
    workspace_id uuid,
    request_id text NOT NULL,
    ip_address inet,
    user_agent text NOT NULL DEFAULT '',
    outcome text NOT NULL CHECK (outcome IN ('SUCCESS', 'FAILURE', 'DENIED')),
    reason text,
    before_data jsonb,
    after_data jsonb,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT audit_logs_action_length CHECK (length(action) BETWEEN 1 AND 160),
    CONSTRAINT audit_logs_request_id_length CHECK (length(request_id) BETWEEN 1 AND 200)
);
CREATE INDEX audit_logs_actor_time_idx ON audit_logs (actor_internal_user_id, occurred_at DESC);
CREATE INDEX audit_logs_entity_time_idx ON audit_logs (entity_type, entity_id, occurred_at DESC);
CREATE INDEX audit_logs_workspace_time_idx ON audit_logs (workspace_id, occurred_at DESC);

CREATE TABLE webhook_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    provider text NOT NULL,
    provider_event_id text NOT NULL,
    event_type text NOT NULL,
    signature_valid boolean NOT NULL,
    payload_hash bytea NOT NULL,
    sanitized_payload jsonb,
    received_at timestamptz NOT NULL DEFAULT now(),
    processed_at timestamptz,
    processing_error text,
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    UNIQUE (provider, provider_event_id)
);
CREATE INDEX webhook_events_unprocessed_idx ON webhook_events (received_at) WHERE processed_at IS NULL;

CREATE TABLE feature_flags (
    key text PRIMARY KEY,
    description text NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    safe_config jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_by uuid REFERENCES internal_users(id) ON DELETE SET NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT feature_flags_key_format CHECK (key ~ '^[a-z][a-z0-9_.-]{1,99}$')
);
