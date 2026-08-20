-- atlas:txmode file

CREATE TYPE scene_generation_status AS ENUM (
    'DRAFT','READY','QUEUED','SUBMITTING','PROVIDER_QUEUED','PROVIDER_PROCESSING',
    'SUCCEEDED','DOWNLOADING','VALIDATING','REVIEW_REQUIRED','APPROVED','REJECTED','FAILED','CANCELLED'
);
CREATE TYPE transcription_status AS ENUM ('QUEUED','PROCESSING','SUCCEEDED','FAILED','NOT_REQUIRED');
CREATE TYPE quality_check_status AS ENUM ('QUEUED','PROCESSING','REVIEW_REQUIRED','PASSED','FAILED');

CREATE TABLE scene_generation_tasks (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    campaign_id uuid NOT NULL REFERENCES campaigns(id),
    scene_id uuid NOT NULL,
    scene_version integer NOT NULL CHECK (scene_version > 0),
    provider text NOT NULL,
    provider_task_id text,
    status scene_generation_status NOT NULL DEFAULT 'QUEUED',
    idempotency_key text NOT NULL,
    attempt_number integer NOT NULL DEFAULT 1 CHECK (attempt_number > 0),
    model text NOT NULL,
    api_version text NOT NULL,
    resolution text NOT NULL,
    aspect_ratio text NOT NULL,
    duration_seconds integer NOT NULL CHECK (duration_seconds BETWEEN 3 AND 15),
    generate_audio boolean NOT NULL,
    scene_hash text NOT NULL,
    prompt_hash text NOT NULL,
    reference_hash text NOT NULL,
    request_hash text NOT NULL,
    sanitized_request jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(sanitized_request) = 'object'),
    sanitized_response jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(sanitized_response) = 'object'),
    provider_output_url text,
    output_asset_id uuid REFERENCES media_assets(id),
    estimated_cost_usd numeric(18,6) NOT NULL DEFAULT 0 CHECK (estimated_cost_usd >= 0),
    actual_cost_usd numeric(18,6) CHECK (actual_cost_usd IS NULL OR actual_cost_usd >= 0),
    usage_tokens bigint CHECK (usage_tokens IS NULL OR usage_tokens >= 0),
    provider_seed bigint,
    provider_fps integer CHECK (provider_fps IS NULL OR provider_fps > 0),
    poll_count integer NOT NULL DEFAULT 0 CHECK (poll_count >= 0),
    next_poll_at timestamptz,
    timeout_at timestamptz NOT NULL,
    error_category text,
    error_code text,
    error_message text,
    cancel_requested_at timestamptz,
    cancel_requested_by uuid REFERENCES internal_users(id),
    submitted_at timestamptz,
    provider_started_at timestamptz,
    provider_completed_at timestamptz,
    downloaded_at timestamptz,
    reviewed_at timestamptz,
    reviewed_by uuid REFERENCES internal_users(id),
    review_notes text NOT NULL DEFAULT '',
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (scene_id, client_id, workspace_id) REFERENCES scenes(id, client_id, workspace_id),
    FOREIGN KEY (scene_id, scene_version) REFERENCES scene_versions(scene_id, version),
    FOREIGN KEY (workspace_id, client_id) REFERENCES workspaces(id, client_id),
    UNIQUE (id, client_id, workspace_id),
    UNIQUE (provider, provider_task_id),
    UNIQUE (idempotency_key, attempt_number),
    CONSTRAINT scene_generation_provider_task_state CHECK (
      status IN ('DRAFT','READY','QUEUED','SUBMITTING','FAILED','CANCELLED') OR provider_task_id IS NOT NULL
    ),
    CONSTRAINT scene_generation_output_state CHECK (
      status NOT IN ('VALIDATING','REVIEW_REQUIRED','APPROVED','REJECTED') OR output_asset_id IS NOT NULL
    )
);
CREATE INDEX scene_generation_scene_idx ON scene_generation_tasks (scene_id, created_at DESC);
CREATE INDEX scene_generation_campaign_idx ON scene_generation_tasks (campaign_id, status, created_at DESC);
CREATE INDEX scene_generation_poll_idx ON scene_generation_tasks (next_poll_at) WHERE status IN ('PROVIDER_QUEUED','PROVIDER_PROCESSING');
CREATE UNIQUE INDEX scene_generation_active_key_idx ON scene_generation_tasks (idempotency_key)
    WHERE status IN ('QUEUED','SUBMITTING','PROVIDER_QUEUED','PROVIDER_PROCESSING','SUCCEEDED','DOWNLOADING','VALIDATING');

CREATE TABLE scene_generation_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    generation_task_id uuid NOT NULL REFERENCES scene_generation_tasks(id) ON DELETE CASCADE,
    from_status scene_generation_status,
    to_status scene_generation_status NOT NULL,
    actor_id uuid REFERENCES internal_users(id),
    source text NOT NULL CHECK (source IN ('API','WORKER','PROVIDER_WEBHOOK','PROVIDER_POLL','SYSTEM')),
    provider_request_id text,
    safe_detail text NOT NULL DEFAULT '',
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
    occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX scene_generation_events_task_idx ON scene_generation_events (generation_task_id, occurred_at);

CREATE TABLE provider_webhook_deliveries (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    provider text NOT NULL,
    provider_task_id text NOT NULL,
    payload_hash text NOT NULL,
    request_id text NOT NULL,
    signature_valid boolean NOT NULL,
    status_code integer,
    processed_at timestamptz,
    processing_error text,
    received_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_task_id, payload_hash)
);

CREATE TABLE scene_transcriptions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    generation_task_id uuid NOT NULL REFERENCES scene_generation_tasks(id) ON DELETE CASCADE,
    status transcription_status NOT NULL DEFAULT 'QUEUED',
    provider text NOT NULL,
    model text NOT NULL,
    language char(2) CHECK (language IS NULL OR language IN ('vi','en')),
    transcript text NOT NULL DEFAULT '',
    segments jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(segments) = 'array'),
    transcript_hash text,
    provider_request_id text,
    input_tokens bigint CHECK (input_tokens IS NULL OR input_tokens >= 0),
    output_tokens bigint CHECK (output_tokens IS NULL OR output_tokens >= 0),
    actual_cost_usd numeric(18,6) CHECK (actual_cost_usd IS NULL OR actual_cost_usd >= 0),
    error_code text,
    error_message text,
    completed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (generation_task_id)
);

CREATE TABLE scene_quality_checks (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    generation_task_id uuid NOT NULL REFERENCES scene_generation_tasks(id) ON DELETE CASCADE,
    status quality_check_status NOT NULL DEFAULT 'QUEUED',
    deterministic_pass boolean,
    transcript_pass boolean,
    video_decodes boolean,
    duration_pass boolean,
    resolution_pass boolean,
    audio_stream_present boolean,
    silence_warning boolean,
    transcript_diff jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(transcript_diff) = 'object'),
    findings jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(findings) = 'array'),
    character_count_review integer CHECK (character_count_review IS NULL OR character_count_review >= 0),
    duplicate_character_review boolean,
    duplicate_product_review boolean,
    product_color_mismatch boolean,
    blur_or_low_quality_warning boolean,
    crop_warning boolean,
    subtitle_overflow boolean,
    logo_overlap boolean,
    cta_safe_zone_violation boolean,
    human_notes text NOT NULL DEFAULT '',
    reviewed_by uuid REFERENCES internal_users(id),
    reviewed_at timestamptz,
    completed_at timestamptz,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (generation_task_id)
);

CREATE TABLE scene_generation_edits (
    generation_task_id uuid PRIMARY KEY REFERENCES scene_generation_tasks(id) ON DELETE CASCADE,
    trim_start_ms bigint NOT NULL DEFAULT 0 CHECK (trim_start_ms >= 0),
    trim_end_ms bigint CHECK (trim_end_ms IS NULL OR trim_end_ms > 0),
    mute_audio boolean NOT NULL DEFAULT false,
    transition text NOT NULL DEFAULT 'CUT' CHECK (transition IN ('CUT','CROSSFADE','FADE_TO_BLACK')),
    replacement_asset_id uuid REFERENCES media_assets(id),
    attached_product_asset_ids uuid[] NOT NULL DEFAULT '{}',
    subtitle_preview boolean NOT NULL DEFAULT true,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    updated_by uuid REFERENCES internal_users(id),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT scene_generation_edit_trim CHECK (trim_end_ms IS NULL OR trim_end_ms > trim_start_ms)
);

ALTER TABLE scenes ADD COLUMN selected_generation_task_id uuid REFERENCES scene_generation_tasks(id);

ALTER TABLE approvals DROP CONSTRAINT approvals_entity_type_check;
ALTER TABLE approvals ADD CONSTRAINT approvals_entity_type_check CHECK (
  entity_type IN ('CAMPAIGN','CONCEPT','CONTENT_VARIANT','SCRIPT','SCENE','SCENE_GENERATION','FINAL_RENDER','SOCIAL_POST','AD_CAMPAIGN','BUDGET_CHANGE','RECOMMENDATION')
);
