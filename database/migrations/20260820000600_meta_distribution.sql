-- atlas:txmode file

CREATE TYPE meta_connection_status AS ENUM ('CONNECTED','EXPIRING','EXPIRED','ERROR','DISCONNECTED');
CREATE TYPE social_platform AS ENUM ('FACEBOOK','INSTAGRAM');
CREATE TYPE social_post_status AS ENUM ('DRAFT','APPROVAL_REQUIRED','APPROVED','SCHEDULED','PUBLISHING','PUBLISHED','FAILED','PERMANENT_FAILURE','CANCELLED');
CREATE TYPE meta_ad_campaign_status AS ENUM ('DRAFT','APPROVAL_REQUIRED','APPROVED','CREATING','PAUSED','ACTIVE','ARCHIVED','FAILED');
CREATE TYPE meta_action_type AS ENUM ('CREATE_PAUSED','ACTIVATE','RESUME','PAUSE','ARCHIVE','BUDGET_CHANGE');
CREATE TYPE meta_action_status AS ENUM ('PENDING_APPROVAL','APPROVED','QUEUED','PROCESSING','SUCCEEDED','REJECTED','FAILED');

CREATE TABLE meta_oauth_states (
    state_hash text PRIMARY KEY,
    actor_id uuid NOT NULL REFERENCES internal_users(id),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id,client_id) REFERENCES workspaces(id,client_id)
);

CREATE TABLE meta_connections (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    meta_user_id text NOT NULL,
    display_name text NOT NULL DEFAULT '',
    token_ciphertext bytea NOT NULL,
    token_nonce bytea NOT NULL,
    token_type text NOT NULL DEFAULT 'USER',
    scopes text[] NOT NULL DEFAULT '{}',
    token_issued_at timestamptz,
    token_expires_at timestamptz,
    data_access_expires_at timestamptz,
    api_version text NOT NULL,
    status meta_connection_status NOT NULL DEFAULT 'CONNECTED',
    last_validated_at timestamptz,
    last_error_code text,
    last_error_message text,
    disconnected_at timestamptz,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id,client_id) REFERENCES workspaces(id,client_id),
    UNIQUE (id,client_id,workspace_id)
);
CREATE UNIQUE INDEX meta_connections_workspace_active_idx ON meta_connections(workspace_id) WHERE disconnected_at IS NULL;
CREATE INDEX meta_connections_expiry_idx ON meta_connections(token_expires_at) WHERE disconnected_at IS NULL;

CREATE TABLE social_accounts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    connection_id uuid NOT NULL REFERENCES meta_connections(id) ON DELETE CASCADE,
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    platform social_platform NOT NULL,
    provider_account_id text NOT NULL,
    facebook_page_id text,
    instagram_business_id text,
    name text NOT NULL,
    username text,
    picture_url text,
    tasks text[] NOT NULL DEFAULT '{}',
    token_ciphertext bytea NOT NULL,
    token_nonce bytea NOT NULL,
    status meta_connection_status NOT NULL DEFAULT 'CONNECTED',
    last_discovered_at timestamptz NOT NULL DEFAULT now(),
    disconnected_at timestamptz,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id,client_id) REFERENCES workspaces(id,client_id),
    UNIQUE (workspace_id,platform,provider_account_id),
    UNIQUE (id,client_id,workspace_id),
    CONSTRAINT social_account_provider_shape CHECK (
      (platform='FACEBOOK' AND facebook_page_id IS NOT NULL) OR
      (platform='INSTAGRAM' AND facebook_page_id IS NOT NULL AND instagram_business_id IS NOT NULL)
    )
);
CREATE INDEX social_accounts_scope_idx ON social_accounts(client_id,workspace_id,platform,status) WHERE disconnected_at IS NULL;

CREATE TABLE social_posts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    campaign_id uuid NOT NULL REFERENCES campaigns(id),
    social_account_id uuid NOT NULL,
    platform social_platform NOT NULL,
    media_asset_id uuid NOT NULL REFERENCES media_assets(id),
    caption text NOT NULL,
    scheduled_at timestamptz,
    idempotency_key text NOT NULL UNIQUE,
    status social_post_status NOT NULL DEFAULT 'DRAFT',
    content_hash text NOT NULL,
    provider_post_id text,
    public_url text,
    provider_request_id text,
    error_category text,
    error_code text,
    error_message text,
    published_at timestamptz,
    reviewed_at timestamptz,
    reviewed_by uuid REFERENCES internal_users(id),
    review_notes text NOT NULL DEFAULT '',
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id,client_id) REFERENCES workspaces(id,client_id),
    FOREIGN KEY (social_account_id,client_id,workspace_id) REFERENCES social_accounts(id,client_id,workspace_id),
    UNIQUE (id,client_id,workspace_id),
    CONSTRAINT social_post_schedule_future CHECK (scheduled_at IS NULL OR scheduled_at > created_at),
    CONSTRAINT social_post_published_shape CHECK (status <> 'PUBLISHED' OR (provider_post_id IS NOT NULL AND published_at IS NOT NULL))
);
CREATE INDEX social_posts_campaign_idx ON social_posts(campaign_id,created_at DESC);
CREATE INDEX social_posts_schedule_idx ON social_posts(scheduled_at) WHERE status='SCHEDULED';

CREATE TABLE publish_jobs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    social_post_id uuid NOT NULL REFERENCES social_posts(id) ON DELETE CASCADE,
    idempotency_key text NOT NULL UNIQUE,
    river_job_id bigint,
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    last_error_retryable boolean,
    safe_response jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(safe_response)='object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (social_post_id)
);

CREATE TABLE meta_businesses (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    connection_id uuid NOT NULL REFERENCES meta_connections(id) ON DELETE CASCADE,
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    provider_business_id text NOT NULL,
    name text NOT NULL,
    verification_status text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id,client_id) REFERENCES workspaces(id,client_id),
    UNIQUE (workspace_id,provider_business_id),
    UNIQUE (id,client_id,workspace_id)
);

CREATE TABLE meta_ad_accounts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    meta_business_id uuid REFERENCES meta_businesses(id) ON DELETE CASCADE,
    connection_id uuid NOT NULL REFERENCES meta_connections(id) ON DELETE CASCADE,
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    provider_ad_account_id text NOT NULL,
    name text NOT NULL,
    currency char(3) NOT NULL,
    timezone_name text NOT NULL,
    account_status integer,
    provider_spend_cap_minor bigint,
    amount_spent_minor bigint,
    last_synced_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id,client_id) REFERENCES workspaces(id,client_id),
    UNIQUE (workspace_id,provider_ad_account_id),
    UNIQUE (id,client_id,workspace_id)
);

CREATE TABLE meta_pixels (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    meta_ad_account_id uuid NOT NULL REFERENCES meta_ad_accounts(id) ON DELETE CASCADE,
    provider_pixel_id text NOT NULL,
    name text NOT NULL,
    conversion_event text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (meta_ad_account_id,provider_pixel_id)
);

CREATE TABLE meta_audiences (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    meta_ad_account_id uuid NOT NULL REFERENCES meta_ad_accounts(id) ON DELETE CASCADE,
    provider_audience_id text NOT NULL,
    name text NOT NULL,
    audience_type text NOT NULL,
    subtype text,
    approximate_count bigint,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (meta_ad_account_id,provider_audience_id)
);

CREATE TABLE meta_ad_guardrails (
    workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    client_id uuid NOT NULL,
    workspace_spend_cap_minor bigint NOT NULL CHECK (workspace_spend_cap_minor > 0),
    default_campaign_spend_cap_minor bigint NOT NULL CHECK (default_campaign_spend_cap_minor > 0),
    maximum_budget_increase_percent numeric(6,2) NOT NULL DEFAULT 20 CHECK (maximum_budget_increase_percent BETWEEN 0 AND 100),
    currency char(3) NOT NULL,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id,client_id) REFERENCES workspaces(id,client_id),
    CHECK (default_campaign_spend_cap_minor <= workspace_spend_cap_minor)
);

CREATE TABLE ad_campaigns (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    campaign_id uuid NOT NULL REFERENCES campaigns(id),
    meta_ad_account_id uuid NOT NULL,
    social_account_id uuid NOT NULL,
    meta_pixel_id uuid REFERENCES meta_pixels(id),
    name text NOT NULL,
    objective text NOT NULL,
    buying_type text NOT NULL DEFAULT 'AUCTION',
    daily_budget_minor bigint CHECK (daily_budget_minor IS NULL OR daily_budget_minor > 0),
    lifetime_budget_minor bigint CHECK (lifetime_budget_minor IS NULL OR lifetime_budget_minor > 0),
    campaign_spend_cap_minor bigint NOT NULL CHECK (campaign_spend_cap_minor > 0),
    currency char(3) NOT NULL,
    starts_at timestamptz,
    ends_at timestamptz,
    audience jsonb NOT NULL CHECK (jsonb_typeof(audience)='object'),
    placements text[] NOT NULL DEFAULT '{}',
    destination_url text NOT NULL,
    utm_parameters jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(utm_parameters)='object'),
    conversion_event text,
    provider_campaign_id text,
    status meta_ad_campaign_status NOT NULL DEFAULT 'DRAFT',
    campaign_hash text NOT NULL,
    last_error_code text,
    last_error_message text,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id,client_id) REFERENCES workspaces(id,client_id),
    FOREIGN KEY (meta_ad_account_id,client_id,workspace_id) REFERENCES meta_ad_accounts(id,client_id,workspace_id),
    FOREIGN KEY (social_account_id,client_id,workspace_id) REFERENCES social_accounts(id,client_id,workspace_id),
    UNIQUE (id,client_id,workspace_id),
    CONSTRAINT ad_campaign_budget_mode CHECK ((daily_budget_minor IS NOT NULL)::int + (lifetime_budget_minor IS NOT NULL)::int = 1),
    CONSTRAINT ad_campaign_dates CHECK (ends_at IS NULL OR starts_at IS NULL OR ends_at > starts_at),
    CONSTRAINT ad_campaign_provider_state CHECK (status IN ('DRAFT','APPROVAL_REQUIRED','APPROVED','CREATING','FAILED') OR provider_campaign_id IS NOT NULL)
);
CREATE INDEX ad_campaigns_campaign_idx ON ad_campaigns(campaign_id,created_at DESC);

CREATE TABLE ad_creatives (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    ad_campaign_id uuid NOT NULL REFERENCES ad_campaigns(id) ON DELETE CASCADE,
    media_asset_id uuid NOT NULL REFERENCES media_assets(id),
    thumbnail_asset_id uuid REFERENCES media_assets(id),
    primary_text_variants jsonb NOT NULL CHECK (jsonb_typeof(primary_text_variants)='array'),
    headline_variants jsonb NOT NULL CHECK (jsonb_typeof(headline_variants)='array'),
    cta_variants jsonb NOT NULL CHECK (jsonb_typeof(cta_variants)='array'),
    preview_spec jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(preview_spec)='object'),
    provider_creative_id text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE ad_sets (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    ad_campaign_id uuid NOT NULL REFERENCES ad_campaigns(id) ON DELETE CASCADE,
    name text NOT NULL,
    audience jsonb NOT NULL CHECK (jsonb_typeof(audience)='object'),
    placements text[] NOT NULL DEFAULT '{}',
    optimization_goal text NOT NULL,
    billing_event text NOT NULL DEFAULT 'IMPRESSIONS',
    provider_ad_set_id text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE ads (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    ad_campaign_id uuid NOT NULL REFERENCES ad_campaigns(id) ON DELETE CASCADE,
    ad_set_id uuid NOT NULL REFERENCES ad_sets(id) ON DELETE CASCADE,
    ad_creative_id uuid NOT NULL REFERENCES ad_creatives(id) ON DELETE CASCADE,
    name text NOT NULL,
    provider_ad_id text,
    status text NOT NULL DEFAULT 'PAUSED' CHECK (status IN ('PAUSED','ACTIVE','ARCHIVED','FAILED')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE meta_ad_actions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    ad_campaign_id uuid NOT NULL REFERENCES ad_campaigns(id) ON DELETE CASCADE,
    action meta_action_type NOT NULL,
    status meta_action_status NOT NULL DEFAULT 'PENDING_APPROVAL',
    requested_budget_minor bigint,
    previous_budget_minor bigint,
    confirmation_text text NOT NULL DEFAULT '',
    action_hash text NOT NULL,
    idempotency_key text NOT NULL UNIQUE,
    river_job_id bigint,
    requested_by uuid NOT NULL REFERENCES internal_users(id),
    reviewed_by uuid REFERENCES internal_users(id),
    review_notes text NOT NULL DEFAULT '',
    safe_response jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(safe_response)='object'),
    error_code text,
    error_message text,
    requested_at timestamptz NOT NULL DEFAULT now(),
    reviewed_at timestamptz,
    completed_at timestamptz,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    UNIQUE (ad_campaign_id,action_hash)
);

CREATE TABLE social_post_metrics_daily (
    social_post_id uuid NOT NULL REFERENCES social_posts(id) ON DELETE CASCADE,
    metric_date date NOT NULL,
    views bigint NOT NULL DEFAULT 0,
    reach bigint NOT NULL DEFAULT 0,
    impressions bigint NOT NULL DEFAULT 0,
    watch_time_ms bigint NOT NULL DEFAULT 0,
    likes bigint NOT NULL DEFAULT 0,
    comments bigint NOT NULL DEFAULT 0,
    shares bigint NOT NULL DEFAULT 0,
    saves bigint NOT NULL DEFAULT 0,
    link_clicks bigint NOT NULL DEFAULT 0,
    provider_response jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(provider_response)='object'),
    synced_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (social_post_id,metric_date)
);

CREATE TABLE ad_campaign_metrics_daily (
    ad_campaign_id uuid NOT NULL REFERENCES ad_campaigns(id) ON DELETE CASCADE,
    metric_date date NOT NULL,
    spend_minor bigint NOT NULL DEFAULT 0,
    impressions bigint NOT NULL DEFAULT 0,
    reach bigint NOT NULL DEFAULT 0,
    clicks bigint NOT NULL DEFAULT 0,
    conversions numeric(18,4) NOT NULL DEFAULT 0,
    leads numeric(18,4) NOT NULL DEFAULT 0,
    purchases numeric(18,4) NOT NULL DEFAULT 0,
    revenue_minor bigint NOT NULL DEFAULT 0,
    frequency numeric(18,6) NOT NULL DEFAULT 0,
    provider_response jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(provider_response)='object'),
    synced_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (ad_campaign_id,metric_date)
);

CREATE TABLE ad_set_metrics_daily (
    ad_set_id uuid NOT NULL REFERENCES ad_sets(id) ON DELETE CASCADE,
    metric_date date NOT NULL,
    spend_minor bigint NOT NULL DEFAULT 0,
    impressions bigint NOT NULL DEFAULT 0,
    clicks bigint NOT NULL DEFAULT 0,
    conversions numeric(18,4) NOT NULL DEFAULT 0,
    provider_response jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(provider_response)='object'),
    synced_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (ad_set_id,metric_date)
);

CREATE TABLE ad_metrics_daily (
    ad_id uuid NOT NULL REFERENCES ads(id) ON DELETE CASCADE,
    metric_date date NOT NULL,
    spend_minor bigint NOT NULL DEFAULT 0,
    impressions bigint NOT NULL DEFAULT 0,
    clicks bigint NOT NULL DEFAULT 0,
    conversions numeric(18,4) NOT NULL DEFAULT 0,
    provider_response jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(provider_response)='object'),
    synced_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (ad_id,metric_date)
);

CREATE TABLE meta_webhook_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    delivery_hash text NOT NULL UNIQUE,
    signature_valid boolean NOT NULL,
    object_type text NOT NULL,
    normalized_events jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(normalized_events)='array'),
    processed_at timestamptz,
    processing_error text,
    received_at timestamptz NOT NULL DEFAULT now()
);
