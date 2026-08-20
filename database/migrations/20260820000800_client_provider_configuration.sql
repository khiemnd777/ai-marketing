-- atlas:txmode file

CREATE TYPE provider_kind AS ENUM ('OPENAI','SEEDANCE','R2','META','RENDERER');

CREATE TABLE client_provider_profiles (
    client_id uuid PRIMARY KEY REFERENCES clients(id) ON DELETE CASCADE,
    demo_mode boolean NOT NULL DEFAULT true,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE provider_configurations (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    provider provider_kind NOT NULL,
    enabled boolean NOT NULL DEFAULT true,
    safe_config jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(safe_config) = 'object'),
    secret_ciphertext bytea NOT NULL,
    secret_nonce bytea NOT NULL,
    configured_secret_fields text[] NOT NULL DEFAULT '{}',
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (client_id, provider),
    UNIQUE (id, client_id),
    CONSTRAINT provider_configurations_secret_fields CHECK (
      configured_secret_fields <@ ARRAY[
        'apiKey','webhookSecret','accessKeyId','secretAccessKey','appSecret'
      ]::text[]
    )
);

CREATE INDEX provider_configurations_client_idx ON provider_configurations (client_id, provider);
