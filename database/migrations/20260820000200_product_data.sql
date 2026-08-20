-- atlas:txmode file

CREATE TYPE lifecycle_status AS ENUM ('ACTIVE', 'ARCHIVED');
CREATE TYPE content_status AS ENUM ('DRAFT', 'APPROVED', 'REJECTED', 'ARCHIVED');
CREATE TYPE fact_status AS ENUM ('DRAFT', 'APPROVED', 'REJECTED');
CREATE TYPE claim_kind AS ENUM ('APPROVED', 'PROHIBITED');
CREATE TYPE media_asset_type AS ENUM ('IMAGE', 'VIDEO', 'AUDIO', 'LOGO', 'BROCHURE', 'SCREENSHOT', 'SCREEN_RECORDING');
CREATE TYPE upload_status AS ENUM ('PENDING', 'UPLOADING', 'UPLOADED', 'VERIFIED', 'FAILED', 'EXPIRED');

CREATE TABLE clients (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    company_name text NOT NULL,
    contact_name text NOT NULL DEFAULT '',
    contact_email text,
    phone text,
    industry text NOT NULL DEFAULT '',
    market text NOT NULL DEFAULT '',
    internal_notes text NOT NULL DEFAULT '',
    status lifecycle_status NOT NULL DEFAULT 'ACTIVE',
    future_tenant_owner_id text,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    archived_at timestamptz,
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT clients_company_name_length CHECK (length(company_name) BETWEEN 2 AND 200),
    CONSTRAINT clients_contact_email_length CHECK (contact_email IS NULL OR length(contact_email) <= 320),
    CONSTRAINT clients_phone_length CHECK (phone IS NULL OR length(phone) <= 40),
    CONSTRAINT clients_archive_consistency CHECK ((status = 'ARCHIVED') = (archived_at IS NOT NULL))
);
CREATE INDEX clients_status_name_idx ON clients (status, company_name);

CREATE TABLE workspaces (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL REFERENCES clients(id),
    name text NOT NULL,
    slug text NOT NULL,
    timezone text NOT NULL DEFAULT 'Asia/Ho_Chi_Minh',
    status lifecycle_status NOT NULL DEFAULT 'ACTIVE',
    future_tenant_owner_id text,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    archived_at timestamptz,
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (client_id, slug),
    UNIQUE (id, client_id),
    CONSTRAINT workspaces_name_length CHECK (length(name) BETWEEN 2 AND 160),
    CONSTRAINT workspaces_slug_format CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    CONSTRAINT workspaces_archive_consistency CHECK ((status = 'ARCHIVED') = (archived_at IS NOT NULL))
);
CREATE INDEX workspaces_client_status_idx ON workspaces (client_id, status, name);

CREATE TABLE brands (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    name text NOT NULL,
    status lifecycle_status NOT NULL DEFAULT 'ACTIVE',
    current_version integer NOT NULL DEFAULT 1 CHECK (current_version > 0),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    archived_at timestamptz,
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id, client_id) REFERENCES workspaces(id, client_id),
    UNIQUE (id, client_id, workspace_id),
    CONSTRAINT brands_name_length CHECK (length(name) BETWEEN 2 AND 160),
    CONSTRAINT brands_archive_consistency CHECK ((status = 'ARCHIVED') = (archived_at IS NOT NULL))
);
CREATE INDEX brands_scope_status_idx ON brands (client_id, workspace_id, status, name);

CREATE TABLE brand_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    brand_id uuid NOT NULL REFERENCES brands(id),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    version integer NOT NULL CHECK (version > 0),
    logo_asset_ids uuid[] NOT NULL DEFAULT '{}',
    primary_color text,
    secondary_color text,
    background_color text,
    heading_font text,
    body_font text,
    tone_of_voice text NOT NULL DEFAULT '',
    primary_language text NOT NULL CHECK (primary_language IN ('vi', 'en')),
    target_audience text NOT NULL DEFAULT '',
    main_message text NOT NULL DEFAULT '',
    default_cta text NOT NULL DEFAULT '',
    website text,
    phone_number text,
    preferred_terminology text[] NOT NULL DEFAULT '{}',
    prohibited_terminology text[] NOT NULL DEFAULT '{}',
    default_disclaimer text NOT NULL DEFAULT '',
    default_video_style text NOT NULL DEFAULT '',
    default_music_style text NOT NULL DEFAULT '',
    change_summary text NOT NULL DEFAULT '',
    created_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (brand_id, client_id, workspace_id) REFERENCES brands(id, client_id, workspace_id),
    UNIQUE (brand_id, version),
    CONSTRAINT brand_colors_hex CHECK (
      (primary_color IS NULL OR primary_color ~ '^#[0-9A-Fa-f]{6}$') AND
      (secondary_color IS NULL OR secondary_color ~ '^#[0-9A-Fa-f]{6}$') AND
      (background_color IS NULL OR background_color ~ '^#[0-9A-Fa-f]{6}$')
    )
);
CREATE INDEX brand_versions_scope_idx ON brand_versions (client_id, workspace_id, brand_id, version DESC);

CREATE TABLE products (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    brand_id uuid REFERENCES brands(id),
    name text NOT NULL,
    sku text NOT NULL,
    model text NOT NULL DEFAULT '',
    category text NOT NULL,
    vertical_key text NOT NULL,
    status content_status NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'APPROVED', 'ARCHIVED')),
    current_version integer NOT NULL DEFAULT 1 CHECK (current_version > 0),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    archived_at timestamptz,
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id, client_id) REFERENCES workspaces(id, client_id),
    UNIQUE (id, client_id, workspace_id),
    UNIQUE (workspace_id, sku),
    CONSTRAINT products_name_length CHECK (length(name) BETWEEN 2 AND 200),
    CONSTRAINT products_sku_length CHECK (length(sku) BETWEEN 1 AND 100),
    CONSTRAINT products_vertical_key_format CHECK (vertical_key ~ '^[a-z][a-z0-9-]{1,79}$'),
    CONSTRAINT products_archive_consistency CHECK ((status = 'ARCHIVED') = (archived_at IS NOT NULL))
);
CREATE INDEX products_scope_status_idx ON products (client_id, workspace_id, status, name);

CREATE TABLE product_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    product_id uuid NOT NULL,
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    version integer NOT NULL CHECK (version > 0),
    short_description text NOT NULL DEFAULT '',
    long_description text NOT NULL DEFAULT '',
    features text[] NOT NULL DEFAULT '{}',
    benefits text[] NOT NULL DEFAULT '{}',
    differentiators text[] NOT NULL DEFAULT '{}',
    intended_audience text NOT NULL DEFAULT '',
    currency char(3),
    regular_price numeric(18,2) CHECK (regular_price IS NULL OR regular_price >= 0),
    sale_price numeric(18,2) CHECK (sale_price IS NULL OR sale_price >= 0),
    discount_code text,
    offer_valid_from timestamptz,
    offer_valid_until timestamptz,
    variants jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(variants) = 'array'),
    change_summary text NOT NULL DEFAULT '',
    created_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (product_id, client_id, workspace_id) REFERENCES products(id, client_id, workspace_id),
    UNIQUE (product_id, version),
    CONSTRAINT product_versions_price_order CHECK (sale_price IS NULL OR regular_price IS NULL OR sale_price <= regular_price),
    CONSTRAINT product_versions_offer_order CHECK (offer_valid_until IS NULL OR offer_valid_from IS NULL OR offer_valid_until > offer_valid_from)
);
CREATE INDEX product_versions_scope_idx ON product_versions (client_id, workspace_id, product_id, version DESC);

CREATE TABLE product_vertical_data (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    product_id uuid NOT NULL,
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    vertical_key text NOT NULL,
    schema_version integer NOT NULL CHECK (schema_version > 0),
    data jsonb NOT NULL CHECK (jsonb_typeof(data) = 'object'),
    data_hash text NOT NULL,
    validated_at timestamptz NOT NULL,
    created_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (product_id, client_id, workspace_id) REFERENCES products(id, client_id, workspace_id),
    UNIQUE (product_id, schema_version, data_hash)
);

CREATE TABLE product_facts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    product_id uuid NOT NULL,
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    fact_key text NOT NULL,
    label text NOT NULL,
    exact_value text NOT NULL,
    normalized_value jsonb,
    unit text,
    source_name text NOT NULL,
    source_excerpt text NOT NULL DEFAULT '',
    source_asset_id uuid,
    status fact_status NOT NULL DEFAULT 'DRAFT',
    locked_value boolean NOT NULL DEFAULT false,
    effective_from timestamptz,
    expires_at timestamptz,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    approved_by uuid REFERENCES internal_users(id),
    approved_at timestamptz,
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (product_id, client_id, workspace_id) REFERENCES products(id, client_id, workspace_id),
    UNIQUE (product_id, fact_key),
    CONSTRAINT product_facts_key_format CHECK (fact_key ~ '^[a-z][a-z0-9_.-]{1,99}$'),
    CONSTRAINT product_facts_effective_order CHECK (expires_at IS NULL OR effective_from IS NULL OR expires_at > effective_from),
    CONSTRAINT product_facts_approval_consistency CHECK ((status = 'APPROVED') = (approved_at IS NOT NULL))
);
CREATE INDEX product_facts_truth_idx ON product_facts (product_id, status, locked_value, fact_key);

CREATE TABLE product_fact_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    product_fact_id uuid NOT NULL REFERENCES product_facts(id) ON DELETE CASCADE,
    version bigint NOT NULL CHECK (version > 0),
    label text NOT NULL,
    exact_value text NOT NULL,
    normalized_value jsonb,
    unit text,
    source_name text NOT NULL,
    source_excerpt text NOT NULL DEFAULT '',
    source_asset_id uuid,
    locked_value boolean NOT NULL,
    change_summary text NOT NULL DEFAULT '',
    created_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (product_fact_id, version)
);

CREATE TABLE product_claims (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    product_id uuid NOT NULL,
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    claim_kind claim_kind NOT NULL,
    claim_text text NOT NULL,
    rationale text NOT NULL DEFAULT '',
    status fact_status NOT NULL DEFAULT 'DRAFT',
    effective_from timestamptz,
    expires_at timestamptz,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    approved_by uuid REFERENCES internal_users(id),
    approved_at timestamptz,
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (product_id, client_id, workspace_id) REFERENCES products(id, client_id, workspace_id),
    CONSTRAINT product_claims_text_length CHECK (length(claim_text) BETWEEN 3 AND 2000),
    CONSTRAINT product_claims_approval_consistency CHECK ((status = 'APPROVED') = (approved_at IS NOT NULL))
);
CREATE INDEX product_claims_truth_idx ON product_claims (product_id, status, claim_kind);

CREATE TABLE product_claim_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    product_claim_id uuid NOT NULL REFERENCES product_claims(id) ON DELETE CASCADE,
    version bigint NOT NULL CHECK (version > 0),
    claim_text text NOT NULL,
    rationale text NOT NULL DEFAULT '',
    change_summary text NOT NULL DEFAULT '',
    created_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (product_claim_id, version)
);

CREATE TABLE product_claim_sources (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    claim_id uuid NOT NULL REFERENCES product_claims(id) ON DELETE CASCADE,
    fact_id uuid REFERENCES product_facts(id),
    media_asset_id uuid,
    evidence_excerpt text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT product_claim_sources_reference CHECK (fact_id IS NOT NULL OR media_asset_id IS NOT NULL),
    UNIQUE NULLS NOT DISTINCT (claim_id, fact_id, media_asset_id)
);

CREATE TABLE media_assets (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    brand_id uuid REFERENCES brands(id),
    product_id uuid REFERENCES products(id),
    campaign_id uuid,
    asset_type media_asset_type NOT NULL,
    category text NOT NULL DEFAULT '',
    name text NOT NULL,
    folder text NOT NULL DEFAULT '',
    status content_status NOT NULL DEFAULT 'DRAFT',
    current_version integer NOT NULL DEFAULT 1 CHECK (current_version > 0),
    usage_rights text NOT NULL,
    source_metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(source_metadata) = 'object'),
    expires_at timestamptz,
    temporary_until timestamptz,
    deleted_at timestamptz,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES internal_users(id),
    updated_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id, client_id) REFERENCES workspaces(id, client_id),
    UNIQUE (id, client_id, workspace_id),
    CONSTRAINT media_assets_name_length CHECK (length(name) BETWEEN 1 AND 240)
);
CREATE INDEX media_assets_scope_search_idx ON media_assets (client_id, workspace_id, status, asset_type, name) WHERE deleted_at IS NULL;
CREATE INDEX media_assets_cleanup_idx ON media_assets (temporary_until) WHERE temporary_until IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE media_asset_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    media_asset_id uuid NOT NULL,
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    version integer NOT NULL CHECK (version > 0),
    storage_key text NOT NULL UNIQUE,
    original_filename text NOT NULL,
    mime_type text NOT NULL,
    file_extension text NOT NULL,
    file_size_bytes bigint NOT NULL CHECK (file_size_bytes > 0),
    checksum_sha256 text,
    width integer CHECK (width IS NULL OR width > 0),
    height integer CHECK (height IS NULL OR height > 0),
    duration_ms bigint CHECK (duration_ms IS NULL OR duration_ms > 0),
    codec text,
    bitrate_bps bigint CHECK (bitrate_bps IS NULL OR bitrate_bps > 0),
    thumbnail_storage_key text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
    verified_at timestamptz,
    created_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (media_asset_id, client_id, workspace_id) REFERENCES media_assets(id, client_id, workspace_id),
    UNIQUE (media_asset_id, version)
);

CREATE TABLE media_asset_tags (
    media_asset_id uuid NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    tag text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (media_asset_id, tag),
    CONSTRAINT media_asset_tags_format CHECK (tag ~ '^[[:alnum:]][[:alnum:] _-]{0,79}$')
);
CREATE INDEX media_asset_tags_tag_idx ON media_asset_tags (tag, media_asset_id);

CREATE TABLE media_uploads (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    media_asset_id uuid NOT NULL,
    storage_key text NOT NULL UNIQUE,
    multipart_upload_id text,
    expected_filename text NOT NULL,
    expected_mime_type text NOT NULL,
    expected_extension text NOT NULL,
    expected_size_bytes bigint NOT NULL CHECK (expected_size_bytes > 0),
    status upload_status NOT NULL DEFAULT 'PENDING',
    expires_at timestamptz NOT NULL,
    completed_at timestamptz,
    failure_reason text,
    created_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (media_asset_id, client_id, workspace_id) REFERENCES media_assets(id, client_id, workspace_id),
    CONSTRAINT media_uploads_expiry CHECK (expires_at > created_at)
);
CREATE INDEX media_uploads_cleanup_idx ON media_uploads (status, expires_at);

ALTER TABLE product_facts ADD CONSTRAINT product_facts_source_asset_fk FOREIGN KEY (source_asset_id) REFERENCES media_assets(id);
ALTER TABLE product_fact_versions ADD CONSTRAINT product_fact_versions_source_asset_fk FOREIGN KEY (source_asset_id) REFERENCES media_assets(id);
ALTER TABLE product_claim_sources ADD CONSTRAINT product_claim_sources_media_fk FOREIGN KEY (media_asset_id) REFERENCES media_assets(id);
