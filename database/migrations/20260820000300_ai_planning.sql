CREATE TYPE campaign_status AS ENUM (
    'DRAFT', 'SCRIPT_READY', 'SCRIPT_APPROVED', 'SCENES_GENERATING',
    'SCENE_REVIEW', 'FINAL_RENDERING', 'FINAL_REVIEW', 'APPROVED', 'READY_TO_PUBLISH', 'ARCHIVED'
);

CREATE TYPE campaign_objective AS ENUM (
    'PRODUCT_INTRODUCTION', 'AWARENESS', 'ENGAGEMENT', 'WEBSITE_TRAFFIC',
    'LEAD_GENERATION', 'SALES', 'PROMOTION'
);

CREATE TYPE concept_status AS ENUM ('DRAFT', 'APPROVED', 'REJECTED', 'LOCKED');
CREATE TYPE planning_content_status AS ENUM ('DRAFT', 'APPROVED', 'REJECTED');
CREATE TYPE character_type AS ENUM ('PRESET', 'TRUSTED_GENERATED', 'AUTHORIZED_REAL_PERSON');
CREATE TYPE consent_status AS ENUM ('NOT_REQUIRED', 'PENDING', 'APPROVED', 'REVOKED', 'EXPIRED');
CREATE TYPE generation_operation AS ENUM ('CONCEPTS', 'CONTENT', 'SCRIPT', 'SCENES', 'AUDIT');
CREATE TYPE generation_job_status AS ENUM ('QUEUED', 'RUNNING', 'SUCCEEDED', 'FAILED', 'CANCELLED');
CREATE TYPE provider_request_status AS ENUM ('PENDING', 'SUCCEEDED', 'FAILED');
CREATE TYPE approval_event_type AS ENUM ('REQUESTED', 'APPROVED', 'REJECTED', 'INVALIDATED', 'REVOKED');

CREATE TABLE campaigns (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    brand_id uuid NOT NULL,
    product_id uuid NOT NULL,
    name text NOT NULL,
    status campaign_status NOT NULL DEFAULT 'DRAFT',
    current_version integer NOT NULL DEFAULT 1 CHECK (current_version > 0),
    selected_concept_id uuid,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    archived_at timestamptz,
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id, client_id) REFERENCES workspaces(id, client_id),
    FOREIGN KEY (brand_id, client_id, workspace_id) REFERENCES brands(id, client_id, workspace_id),
    FOREIGN KEY (product_id, client_id, workspace_id) REFERENCES products(id, client_id, workspace_id),
    UNIQUE (id, client_id, workspace_id),
    UNIQUE (workspace_id, name),
    CONSTRAINT campaigns_name_length CHECK (length(name) BETWEEN 2 AND 200),
    CONSTRAINT campaigns_archive_consistency CHECK ((status = 'ARCHIVED') = (archived_at IS NOT NULL))
);
CREATE INDEX campaigns_scope_status_idx ON campaigns (client_id, workspace_id, status, updated_at DESC);

CREATE TABLE campaign_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    campaign_id uuid NOT NULL,
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    version integer NOT NULL CHECK (version > 0),
    objective campaign_objective NOT NULL,
    target_audience text NOT NULL,
    market text NOT NULL,
    country char(2) NOT NULL,
    language char(2) NOT NULL CHECK (language IN ('vi', 'en')),
    social_platform_targets text[] NOT NULL CHECK (cardinality(social_platform_targets) BETWEEN 1 AND 3),
    video_format text NOT NULL CHECK (video_format IN ('INTERVIEW_REVIEW', 'PROBLEM_SOLUTION')),
    duration_seconds integer NOT NULL CHECK (duration_seconds IN (30, 45)),
    aspect_ratio text NOT NULL DEFAULT '9:16' CHECK (aspect_ratio = '9:16'),
    tone text NOT NULL,
    offer text NOT NULL DEFAULT '',
    cta text NOT NULL,
    planned_ads_budget numeric(18,2) CHECK (planned_ads_budget IS NULL OR planned_ads_budget >= 0),
    budget_currency char(3),
    starts_on date,
    ends_on date,
    change_summary text NOT NULL DEFAULT '',
    created_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (campaign_id, client_id, workspace_id) REFERENCES campaigns(id, client_id, workspace_id),
    UNIQUE (campaign_id, version),
    CONSTRAINT campaign_versions_dates CHECK (ends_on IS NULL OR starts_on IS NULL OR ends_on >= starts_on),
    CONSTRAINT campaign_versions_budget_currency CHECK ((planned_ads_budget IS NULL) = (budget_currency IS NULL))
);
CREATE INDEX campaign_versions_scope_idx ON campaign_versions (client_id, workspace_id, campaign_id, version DESC);

CREATE TABLE provider_requests (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    campaign_id uuid REFERENCES campaigns(id),
    provider text NOT NULL,
    operation generation_operation NOT NULL,
    model text NOT NULL,
    prompt_version text NOT NULL,
    provider_request_id text,
    input_hash text NOT NULL,
    status provider_request_status NOT NULL DEFAULT 'PENDING',
    started_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    latency_ms bigint CHECK (latency_ms IS NULL OR latency_ms >= 0),
    input_tokens bigint CHECK (input_tokens IS NULL OR input_tokens >= 0),
    output_tokens bigint CHECK (output_tokens IS NULL OR output_tokens >= 0),
    estimated_cost_usd numeric(18,6) CHECK (estimated_cost_usd IS NULL OR estimated_cost_usd >= 0),
    actual_cost_usd numeric(18,6) CHECK (actual_cost_usd IS NULL OR actual_cost_usd >= 0),
    error_code text,
    error_message text,
    created_by uuid REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id, client_id) REFERENCES workspaces(id, client_id)
);
CREATE INDEX provider_requests_campaign_idx ON provider_requests (campaign_id, created_at DESC);

CREATE TABLE provider_outputs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    provider_request_id uuid NOT NULL REFERENCES provider_requests(id) ON DELETE CASCADE,
    output_hash text NOT NULL,
    normalized_output jsonb NOT NULL CHECK (jsonb_typeof(normalized_output) = 'object'),
    validation_errors jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(validation_errors) = 'array'),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider_request_id, output_hash)
);

CREATE TABLE campaign_concepts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    campaign_id uuid NOT NULL,
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    title text NOT NULL,
    video_format text NOT NULL CHECK (video_format IN ('INTERVIEW_REVIEW', 'PROBLEM_SOLUTION')),
    status concept_status NOT NULL DEFAULT 'DRAFT',
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    current_version integer NOT NULL DEFAULT 1 CHECK (current_version > 0),
    prompt_version text NOT NULL,
    model text NOT NULL,
    request_id text NOT NULL,
    output_hash text NOT NULL,
    estimated_cost_usd numeric(18,6) NOT NULL DEFAULT 0 CHECK (estimated_cost_usd >= 0),
    locked_at timestamptz,
    locked_by uuid REFERENCES internal_users(id),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid REFERENCES internal_users(id),
    updated_by uuid REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (campaign_id, client_id, workspace_id) REFERENCES campaigns(id, client_id, workspace_id),
    UNIQUE (id, client_id, workspace_id),
    CONSTRAINT campaign_concepts_title_length CHECK (length(title) BETWEEN 2 AND 200),
    CONSTRAINT campaign_concepts_lock_consistency CHECK ((status = 'LOCKED') = (locked_at IS NOT NULL))
);
CREATE INDEX campaign_concepts_campaign_idx ON campaign_concepts (campaign_id, status, created_at);

CREATE TABLE campaign_concept_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    concept_id uuid NOT NULL REFERENCES campaign_concepts(id) ON DELETE CASCADE,
    version integer NOT NULL CHECK (version > 0),
    title text NOT NULL,
    video_format text NOT NULL,
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    output_hash text NOT NULL,
    change_summary text NOT NULL DEFAULT '',
    created_by uuid REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (concept_id, version)
);

ALTER TABLE campaigns ADD CONSTRAINT campaigns_selected_concept_fk
    FOREIGN KEY (selected_concept_id, client_id, workspace_id) REFERENCES campaign_concepts(id, client_id, workspace_id);

CREATE TABLE campaign_content_variants (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    campaign_id uuid NOT NULL,
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    variant_key text NOT NULL,
    platform text NOT NULL,
    content text NOT NULL,
    status planning_content_status NOT NULL DEFAULT 'DRAFT',
    current_version integer NOT NULL DEFAULT 1 CHECK (current_version > 0),
    content_hash text NOT NULL,
    prompt_version text NOT NULL,
    model text NOT NULL,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    approved_at timestamptz,
    approved_by uuid REFERENCES internal_users(id),
    created_by uuid REFERENCES internal_users(id),
    updated_by uuid REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (campaign_id, client_id, workspace_id) REFERENCES campaigns(id, client_id, workspace_id),
    UNIQUE (id, client_id, workspace_id),
    UNIQUE (campaign_id, variant_key),
    CONSTRAINT campaign_content_variant_key CHECK (variant_key ~ '^[a-z][a-z0-9_]{1,79}$'),
    CONSTRAINT campaign_content_approval_consistency CHECK ((status = 'APPROVED') = (approved_at IS NOT NULL))
);
CREATE INDEX campaign_content_campaign_idx ON campaign_content_variants (campaign_id, variant_key);

CREATE TABLE campaign_content_variant_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    content_variant_id uuid NOT NULL REFERENCES campaign_content_variants(id) ON DELETE CASCADE,
    version integer NOT NULL CHECK (version > 0),
    content text NOT NULL,
    content_hash text NOT NULL,
    change_summary text NOT NULL DEFAULT '',
    created_by uuid REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (content_variant_id, version)
);

CREATE TABLE characters (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid,
    workspace_id uuid,
    name text NOT NULL,
    provider text NOT NULL,
    provider_asset_id text,
    character_type character_type NOT NULL,
    gender_presentation text NOT NULL DEFAULT '',
    approximate_age_range text NOT NULL DEFAULT '',
    appearance_description text NOT NULL,
    wardrobe text NOT NULL DEFAULT '',
    gesture_style text NOT NULL DEFAULT '',
    default_role text NOT NULL DEFAULT '',
    supported_languages text[] NOT NULL DEFAULT '{}',
    consent_status consent_status NOT NULL,
    preview_asset_id uuid REFERENCES media_assets(id),
    status lifecycle_status NOT NULL DEFAULT 'ACTIVE',
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id, client_id) REFERENCES workspaces(id, client_id),
    UNIQUE (id, client_id, workspace_id),
    CONSTRAINT characters_scope_pair CHECK ((client_id IS NULL) = (workspace_id IS NULL)),
    CONSTRAINT characters_name_length CHECK (length(name) BETWEEN 2 AND 160),
    CONSTRAINT characters_consent_guard CHECK (
      (character_type = 'PRESET' AND consent_status = 'NOT_REQUIRED') OR
      (character_type = 'TRUSTED_GENERATED' AND consent_status IN ('NOT_REQUIRED','APPROVED')) OR
      (character_type = 'AUTHORIZED_REAL_PERSON' AND consent_status IN ('PENDING','APPROVED','REVOKED','EXPIRED'))
    )
);
CREATE INDEX characters_scope_status_idx ON characters (client_id, workspace_id, status, name);

CREATE TABLE character_assets (
    character_id uuid NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    media_asset_id uuid NOT NULL REFERENCES media_assets(id),
    purpose text NOT NULL CHECK (purpose IN ('PREVIEW', 'REFERENCE_IMAGE', 'REFERENCE_VIDEO', 'VOICE_REFERENCE')),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (character_id, media_asset_id, purpose)
);

CREATE TABLE character_consents (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    character_id uuid NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    status consent_status NOT NULL,
    artifact_asset_id uuid REFERENCES media_assets(id),
    subject_name text NOT NULL,
    granted_at timestamptz,
    expires_at timestamptz,
    revoked_at timestamptz,
    notes text NOT NULL DEFAULT '',
    recorded_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT character_consents_dates CHECK (expires_at IS NULL OR granted_at IS NULL OR expires_at > granted_at)
);

CREATE TABLE scripts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    campaign_id uuid NOT NULL,
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    status planning_content_status NOT NULL DEFAULT 'DRAFT',
    current_version integer NOT NULL DEFAULT 1 CHECK (current_version > 0),
    script_hash text NOT NULL,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    approved_at timestamptz,
    approved_by uuid REFERENCES internal_users(id),
    created_by uuid REFERENCES internal_users(id),
    updated_by uuid REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (campaign_id, client_id, workspace_id) REFERENCES campaigns(id, client_id, workspace_id),
    UNIQUE (id, client_id, workspace_id),
    UNIQUE (campaign_id),
    CONSTRAINT scripts_approval_consistency CHECK ((status = 'APPROVED') = (approved_at IS NOT NULL))
);

CREATE TABLE script_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    script_id uuid NOT NULL REFERENCES scripts(id) ON DELETE CASCADE,
    version integer NOT NULL CHECK (version > 0),
    hook text NOT NULL,
    introduction text NOT NULL DEFAULT '',
    problem text NOT NULL DEFAULT '',
    product_solution text NOT NULL DEFAULT '',
    product_features text[] NOT NULL DEFAULT '{}',
    benefits text[] NOT NULL DEFAULT '{}',
    cta text NOT NULL,
    closing text NOT NULL DEFAULT '',
    approximate_duration_seconds integer NOT NULL CHECK (approximate_duration_seconds IN (30,45)),
    character_roles jsonb NOT NULL CHECK (jsonb_typeof(character_roles) = 'object'),
    spoken_language char(2) NOT NULL CHECK (spoken_language IN ('vi','en')),
    script_hash text NOT NULL,
    prompt_version text NOT NULL,
    model text NOT NULL,
    change_summary text NOT NULL DEFAULT '',
    created_by uuid REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (script_id, version)
);

CREATE TABLE script_dialogue_turns (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    script_version_id uuid NOT NULL REFERENCES script_versions(id) ON DELETE CASCADE,
    turn_order integer NOT NULL CHECK (turn_order > 0),
    character_role text NOT NULL,
    dialogue text NOT NULL,
    estimated_duration_ms bigint NOT NULL CHECK (estimated_duration_ms > 0),
    UNIQUE (script_version_id, turn_order)
);

CREATE TABLE scenes (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    campaign_id uuid NOT NULL,
    script_id uuid NOT NULL,
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    scene_key text NOT NULL,
    scene_order integer NOT NULL CHECK (scene_order > 0),
    status planning_content_status NOT NULL DEFAULT 'DRAFT',
    current_version integer NOT NULL DEFAULT 1 CHECK (current_version > 0),
    scene_hash text NOT NULL,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    approved_at timestamptz,
    approved_by uuid REFERENCES internal_users(id),
    created_by uuid REFERENCES internal_users(id),
    updated_by uuid REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (campaign_id, client_id, workspace_id) REFERENCES campaigns(id, client_id, workspace_id),
    FOREIGN KEY (script_id, client_id, workspace_id) REFERENCES scripts(id, client_id, workspace_id),
    UNIQUE (id, client_id, workspace_id),
    UNIQUE (campaign_id, scene_key),
    UNIQUE (campaign_id, scene_order),
    CONSTRAINT scenes_key_format CHECK (scene_key ~ '^scene-[0-9]{2,3}$'),
    CONSTRAINT scenes_approval_consistency CHECK ((status = 'APPROVED') = (approved_at IS NOT NULL))
);
CREATE INDEX scenes_campaign_order_idx ON scenes (campaign_id, scene_order);

CREATE TABLE scene_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    scene_id uuid NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
    version integer NOT NULL CHECK (version > 0),
    duration_seconds integer NOT NULL CHECK (duration_seconds BETWEEN 3 AND 15),
    generation_method text NOT NULL CHECK (generation_method IN ('seedance','product_footage','still_image')),
    speaker_character_id uuid REFERENCES characters(id),
    listener_character_id uuid REFERENCES characters(id),
    dialogue text NOT NULL DEFAULT '',
    speaker_action text NOT NULL DEFAULT '',
    listener_action text NOT NULL DEFAULT '',
    camera text NOT NULL,
    environment text NOT NULL,
    product_placement text NOT NULL,
    expected_cost_usd numeric(18,6) NOT NULL DEFAULT 0 CHECK (expected_cost_usd >= 0),
    seedance_prompt text NOT NULL DEFAULT '',
    scene_hash text NOT NULL,
    change_summary text NOT NULL DEFAULT '',
    created_by uuid REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (scene_id, version),
    CONSTRAINT scene_versions_character_pair CHECK (
      generation_method <> 'seedance' OR
      (speaker_character_id IS NOT NULL AND listener_character_id IS NOT NULL AND speaker_character_id <> listener_character_id)
    )
);

CREATE TABLE scene_assets (
    scene_version_id uuid NOT NULL REFERENCES scene_versions(id) ON DELETE CASCADE,
    media_asset_id uuid NOT NULL REFERENCES media_assets(id),
    role text NOT NULL CHECK (role IN ('PRODUCT_REFERENCE','CHARACTER_REFERENCE','AUDIO_REFERENCE','VIDEO_REFERENCE')),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (scene_version_id, media_asset_id, role)
);

CREATE TABLE scene_required_facts (
    scene_version_id uuid NOT NULL REFERENCES scene_versions(id) ON DELETE CASCADE,
    product_fact_id uuid NOT NULL REFERENCES product_facts(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (scene_version_id, product_fact_id)
);

CREATE TABLE approvals (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    campaign_id uuid REFERENCES campaigns(id),
    entity_type text NOT NULL CHECK (entity_type IN ('CAMPAIGN','CONCEPT','CONTENT_VARIANT','SCRIPT','SCENE','FINAL_RENDER','SOCIAL_POST','AD_CAMPAIGN','BUDGET_CHANGE','RECOMMENDATION')),
    entity_id uuid NOT NULL,
    entity_version bigint NOT NULL CHECK (entity_version > 0),
    entity_hash text NOT NULL,
    status planning_content_status NOT NULL DEFAULT 'DRAFT',
    requested_by uuid REFERENCES internal_users(id),
    requested_at timestamptz NOT NULL DEFAULT now(),
    decided_by uuid REFERENCES internal_users(id),
    decided_at timestamptz,
    notes text NOT NULL DEFAULT '',
    invalidated_at timestamptz,
    invalidation_reason text,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id, client_id) REFERENCES workspaces(id, client_id),
    CONSTRAINT approvals_decision_consistency CHECK ((status = 'DRAFT') = (decided_at IS NULL)),
    CONSTRAINT approvals_invalidation_pair CHECK ((invalidated_at IS NULL) = (invalidation_reason IS NULL))
);
CREATE UNIQUE INDEX approvals_active_entity_idx ON approvals (entity_type, entity_id, entity_version) WHERE invalidated_at IS NULL;
CREATE INDEX approvals_campaign_idx ON approvals (campaign_id, requested_at DESC);

CREATE TABLE approval_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    approval_id uuid NOT NULL REFERENCES approvals(id) ON DELETE CASCADE,
    event_type approval_event_type NOT NULL,
    actor_id uuid REFERENCES internal_users(id),
    entity_version bigint NOT NULL CHECK (entity_version > 0),
    entity_hash text NOT NULL,
    notes text NOT NULL DEFAULT '',
    occurred_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE generation_jobs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    campaign_id uuid NOT NULL REFERENCES campaigns(id),
    operation generation_operation NOT NULL,
    status generation_job_status NOT NULL DEFAULT 'QUEUED',
    river_job_id bigint,
    idempotency_key_hash bytea NOT NULL,
    input_hash text NOT NULL,
    estimated_cost_usd numeric(18,6) NOT NULL DEFAULT 0 CHECK (estimated_cost_usd >= 0),
    actual_cost_usd numeric(18,6) CHECK (actual_cost_usd IS NULL OR actual_cost_usd >= 0),
    provider_request_id uuid REFERENCES provider_requests(id),
    output_summary jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(output_summary) = 'object'),
    error_code text,
    error_message text,
    created_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    completed_at timestamptz,
    FOREIGN KEY (workspace_id, client_id) REFERENCES workspaces(id, client_id),
    UNIQUE (campaign_id, operation, idempotency_key_hash)
);
CREATE INDEX generation_jobs_status_idx ON generation_jobs (status, created_at);

CREATE TABLE cost_estimates (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    campaign_id uuid NOT NULL REFERENCES campaigns(id),
    operation generation_operation NOT NULL,
    model text NOT NULL,
    currency char(3) NOT NULL DEFAULT 'USD',
    estimated_input_tokens bigint NOT NULL DEFAULT 0 CHECK (estimated_input_tokens >= 0),
    estimated_output_tokens bigint NOT NULL DEFAULT 0 CHECK (estimated_output_tokens >= 0),
    estimated_video_seconds bigint NOT NULL DEFAULT 0 CHECK (estimated_video_seconds >= 0),
    estimated_cost numeric(18,6) NOT NULL CHECK (estimated_cost >= 0),
    assumptions jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(assumptions) = 'object'),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id, client_id) REFERENCES workspaces(id, client_id)
);
CREATE INDEX cost_estimates_campaign_idx ON cost_estimates (campaign_id, created_at DESC);
