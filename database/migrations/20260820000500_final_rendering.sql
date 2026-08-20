-- atlas:txmode file

CREATE TYPE render_job_status AS ENUM ('QUEUED','BUILDING_MANIFEST','RENDERING','VALIDATING','UPLOADING','REVIEW_REQUIRED','APPROVED','REJECTED','FAILED','CANCELLED');
CREATE TYPE subtitle_format AS ENUM ('SRT','VTT');

CREATE TABLE video_projects (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    campaign_id uuid NOT NULL,
    current_version integer NOT NULL DEFAULT 1 CHECK (current_version > 0),
    selected_render_job_id uuid,
    music_asset_id uuid REFERENCES media_assets(id),
    music_gain_db numeric(6,2) NOT NULL DEFAULT -18 CHECK (music_gain_db BETWEEN -60 AND 0),
    dialogue_ducking_db numeric(6,2) NOT NULL DEFAULT -9 CHECK (dialogue_ducking_db BETWEEN -30 AND 0),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (campaign_id,client_id,workspace_id) REFERENCES campaigns(id,client_id,workspace_id),
    FOREIGN KEY (workspace_id,client_id) REFERENCES workspaces(id,client_id),
    UNIQUE (campaign_id),
    UNIQUE (id,client_id,workspace_id)
);

CREATE TABLE video_project_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    video_project_id uuid NOT NULL REFERENCES video_projects(id) ON DELETE CASCADE,
    version integer NOT NULL CHECK (version > 0),
    headline text NOT NULL,
    lower_third text NOT NULL DEFAULT '',
    show_price boolean NOT NULL DEFAULT true,
    show_discount_code boolean NOT NULL DEFAULT true,
    show_cta boolean NOT NULL DEFAULT true,
    show_website boolean NOT NULL DEFAULT true,
    show_phone boolean NOT NULL DEFAULT true,
    show_qr_code boolean NOT NULL DEFAULT true,
    show_disclaimer boolean NOT NULL DEFAULT true,
    burn_captions boolean NOT NULL DEFAULT true,
    project_hash text NOT NULL,
    change_summary text NOT NULL DEFAULT '',
    created_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (video_project_id,version)
);

CREATE TABLE render_manifests (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    video_project_id uuid NOT NULL REFERENCES video_projects(id) ON DELETE CASCADE,
    video_project_version integer NOT NULL,
    manifest_version integer NOT NULL DEFAULT 1 CHECK (manifest_version > 0),
    manifest_hash text NOT NULL,
    manifest jsonb NOT NULL CHECK (jsonb_typeof(manifest)='object'),
    created_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (video_project_id,video_project_version) REFERENCES video_project_versions(video_project_id,version),
    UNIQUE (video_project_id,video_project_version,manifest_hash)
);

CREATE TABLE render_jobs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    campaign_id uuid NOT NULL REFERENCES campaigns(id),
    video_project_id uuid NOT NULL REFERENCES video_projects(id),
    video_project_version integer NOT NULL,
    render_manifest_id uuid REFERENCES render_manifests(id),
    status render_job_status NOT NULL DEFAULT 'QUEUED',
    idempotency_key text NOT NULL UNIQUE,
    river_job_id bigint,
    output_asset_id uuid REFERENCES media_assets(id),
    thumbnail_storage_key text,
    srt_storage_key text,
    vtt_storage_key text,
    output_hash text,
    renderer_request_id text,
    sanitized_response jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(sanitized_response)='object'),
    error_code text,
    error_message text,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    started_at timestamptz,
    completed_at timestamptz,
    reviewed_at timestamptz,
    reviewed_by uuid REFERENCES internal_users(id),
    review_notes text NOT NULL DEFAULT '',
    created_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id,client_id) REFERENCES workspaces(id,client_id),
    FOREIGN KEY (video_project_id,video_project_version) REFERENCES video_project_versions(video_project_id,version),
    UNIQUE (id,client_id,workspace_id),
    CONSTRAINT render_jobs_output_state CHECK (status NOT IN ('REVIEW_REQUIRED','APPROVED','REJECTED') OR output_asset_id IS NOT NULL)
);
CREATE INDEX render_jobs_campaign_idx ON render_jobs(campaign_id,created_at DESC);

ALTER TABLE video_projects ADD CONSTRAINT video_projects_selected_render_fk FOREIGN KEY (selected_render_job_id) REFERENCES render_jobs(id);

CREATE TABLE video_outputs (
    render_job_id uuid PRIMARY KEY REFERENCES render_jobs(id) ON DELETE CASCADE,
    media_asset_id uuid NOT NULL REFERENCES media_assets(id),
    width integer NOT NULL CHECK (width=1080),
    height integer NOT NULL CHECK (height=1920),
    fps integer NOT NULL CHECK (fps=30),
    duration_ms bigint NOT NULL CHECK (duration_ms > 0),
    codec text NOT NULL CHECK (codec IN ('h264','avc1')),
    audio_codec text,
    file_size_bytes bigint NOT NULL CHECK (file_size_bytes > 0),
    checksum_sha256 text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(metadata)='object'),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subtitle_outputs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    render_job_id uuid NOT NULL REFERENCES render_jobs(id) ON DELETE CASCADE,
    format subtitle_format NOT NULL,
    language char(2) NOT NULL CHECK (language IN ('vi','en')),
    storage_key text NOT NULL UNIQUE,
    checksum_sha256 text NOT NULL,
    cue_count integer NOT NULL CHECK (cue_count >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (render_job_id,format,language)
);
