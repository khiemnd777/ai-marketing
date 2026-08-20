-- Add new schema named "atlas_schema_revisions"
CREATE SCHEMA "atlas_schema_revisions";
-- Add new schema named "public"
CREATE SCHEMA IF NOT EXISTS "public";
-- Set comment to schema: "public"
COMMENT ON SCHEMA "public" IS 'standard public schema';
-- Create "atlas_schema_revisions" table
CREATE TABLE "atlas_schema_revisions"."atlas_schema_revisions" (
  "version" character varying NOT NULL,
  "description" character varying NOT NULL,
  "type" bigint NOT NULL DEFAULT 2,
  "applied" bigint NOT NULL DEFAULT 0,
  "total" bigint NOT NULL DEFAULT 0,
  "executed_at" timestamptz NOT NULL,
  "execution_time" bigint NOT NULL,
  "error" text NULL,
  "error_stmt" text NULL,
  "hash" character varying NOT NULL,
  "partial_hashes" jsonb NULL,
  "operator_version" character varying NOT NULL,
  PRIMARY KEY ("version")
);
-- Create enum type "internal_user_role"
CREATE TYPE "public"."internal_user_role" AS ENUM ('ADMIN', 'OPERATOR', 'REVIEWER');
-- Create enum type "internal_user_status"
CREATE TYPE "public"."internal_user_status" AS ENUM ('ACTIVE', 'DISABLED');
-- Create enum type "idempotency_status"
CREATE TYPE "public"."idempotency_status" AS ENUM ('PROCESSING', 'COMPLETED', 'FAILED');
-- Create enum type "lifecycle_status"
CREATE TYPE "public"."lifecycle_status" AS ENUM ('ACTIVE', 'ARCHIVED');
-- Create enum type "content_status"
CREATE TYPE "public"."content_status" AS ENUM ('DRAFT', 'APPROVED', 'REJECTED', 'ARCHIVED');
-- Create enum type "fact_status"
CREATE TYPE "public"."fact_status" AS ENUM ('DRAFT', 'APPROVED', 'REJECTED');
-- Create enum type "claim_kind"
CREATE TYPE "public"."claim_kind" AS ENUM ('APPROVED', 'PROHIBITED');
-- Create enum type "media_asset_type"
CREATE TYPE "public"."media_asset_type" AS ENUM ('IMAGE', 'VIDEO', 'AUDIO', 'LOGO', 'BROCHURE', 'SCREENSHOT', 'SCREEN_RECORDING');
-- Create enum type "upload_status"
CREATE TYPE "public"."upload_status" AS ENUM ('PENDING', 'UPLOADING', 'UPLOADED', 'VERIFIED', 'FAILED', 'EXPIRED');
-- Create enum type "campaign_status"
CREATE TYPE "public"."campaign_status" AS ENUM ('DRAFT', 'SCRIPT_READY', 'SCRIPT_APPROVED', 'SCENES_GENERATING', 'SCENE_REVIEW', 'FINAL_RENDERING', 'FINAL_REVIEW', 'APPROVED', 'READY_TO_PUBLISH', 'ARCHIVED');
-- Create enum type "campaign_objective"
CREATE TYPE "public"."campaign_objective" AS ENUM ('PRODUCT_INTRODUCTION', 'AWARENESS', 'ENGAGEMENT', 'WEBSITE_TRAFFIC', 'LEAD_GENERATION', 'SALES', 'PROMOTION');
-- Create enum type "concept_status"
CREATE TYPE "public"."concept_status" AS ENUM ('DRAFT', 'APPROVED', 'REJECTED', 'LOCKED');
-- Create enum type "planning_content_status"
CREATE TYPE "public"."planning_content_status" AS ENUM ('DRAFT', 'APPROVED', 'REJECTED');
-- Create enum type "character_type"
CREATE TYPE "public"."character_type" AS ENUM ('PRESET', 'TRUSTED_GENERATED', 'AUTHORIZED_REAL_PERSON');
-- Create enum type "consent_status"
CREATE TYPE "public"."consent_status" AS ENUM ('NOT_REQUIRED', 'PENDING', 'APPROVED', 'REVOKED', 'EXPIRED');
-- Create enum type "generation_operation"
CREATE TYPE "public"."generation_operation" AS ENUM ('CONCEPTS', 'CONTENT', 'SCRIPT', 'SCENES', 'AUDIT');
-- Create enum type "generation_job_status"
CREATE TYPE "public"."generation_job_status" AS ENUM ('QUEUED', 'RUNNING', 'SUCCEEDED', 'FAILED', 'CANCELLED');
-- Create enum type "provider_request_status"
CREATE TYPE "public"."provider_request_status" AS ENUM ('PENDING', 'SUCCEEDED', 'FAILED');
-- Create enum type "approval_event_type"
CREATE TYPE "public"."approval_event_type" AS ENUM ('REQUESTED', 'APPROVED', 'REJECTED', 'INVALIDATED', 'REVOKED');
-- Create enum type "scene_generation_status"
CREATE TYPE "public"."scene_generation_status" AS ENUM ('DRAFT', 'READY', 'QUEUED', 'SUBMITTING', 'PROVIDER_QUEUED', 'PROVIDER_PROCESSING', 'SUCCEEDED', 'DOWNLOADING', 'VALIDATING', 'REVIEW_REQUIRED', 'APPROVED', 'REJECTED', 'FAILED', 'CANCELLED');
-- Create enum type "transcription_status"
CREATE TYPE "public"."transcription_status" AS ENUM ('QUEUED', 'PROCESSING', 'SUCCEEDED', 'FAILED', 'NOT_REQUIRED');
-- Create enum type "quality_check_status"
CREATE TYPE "public"."quality_check_status" AS ENUM ('QUEUED', 'PROCESSING', 'REVIEW_REQUIRED', 'PASSED', 'FAILED');
-- Create enum type "render_job_status"
CREATE TYPE "public"."render_job_status" AS ENUM ('QUEUED', 'BUILDING_MANIFEST', 'RENDERING', 'VALIDATING', 'UPLOADING', 'REVIEW_REQUIRED', 'APPROVED', 'REJECTED', 'FAILED', 'CANCELLED');
-- Create enum type "subtitle_format"
CREATE TYPE "public"."subtitle_format" AS ENUM ('SRT', 'VTT');
-- Create enum type "meta_connection_status"
CREATE TYPE "public"."meta_connection_status" AS ENUM ('CONNECTED', 'EXPIRING', 'EXPIRED', 'ERROR', 'DISCONNECTED');
-- Create enum type "social_platform"
CREATE TYPE "public"."social_platform" AS ENUM ('FACEBOOK', 'INSTAGRAM');
-- Create enum type "social_post_status"
CREATE TYPE "public"."social_post_status" AS ENUM ('DRAFT', 'APPROVAL_REQUIRED', 'APPROVED', 'SCHEDULED', 'PUBLISHING', 'PUBLISHED', 'FAILED', 'PERMANENT_FAILURE', 'CANCELLED');
-- Create enum type "meta_ad_campaign_status"
CREATE TYPE "public"."meta_ad_campaign_status" AS ENUM ('DRAFT', 'APPROVAL_REQUIRED', 'APPROVED', 'CREATING', 'PAUSED', 'ACTIVE', 'ARCHIVED', 'FAILED');
-- Create enum type "meta_action_type"
CREATE TYPE "public"."meta_action_type" AS ENUM ('CREATE_PAUSED', 'ACTIVATE', 'RESUME', 'PAUSE', 'ARCHIVE', 'BUDGET_CHANGE');
-- Create enum type "meta_action_status"
CREATE TYPE "public"."meta_action_status" AS ENUM ('PENDING_APPROVAL', 'APPROVED', 'QUEUED', 'PROCESSING', 'SUCCEEDED', 'REJECTED', 'FAILED');
-- Create enum type "usage_outcome"
CREATE TYPE "public"."usage_outcome" AS ENUM ('SUCCESS', 'FAILURE');
-- Create enum type "recommendation_status"
CREATE TYPE "public"."recommendation_status" AS ENUM ('DRAFT', 'APPROVED', 'REJECTED', 'APPLIED', 'DISMISSED');
-- Create enum type "notification_severity"
CREATE TYPE "public"."notification_severity" AS ENUM ('INFO', 'WARNING', 'CRITICAL');
-- Create "ad_campaign_metrics_daily" table
CREATE TABLE "public"."ad_campaign_metrics_daily" (
  "ad_campaign_id" uuid NOT NULL,
  "metric_date" date NOT NULL,
  "spend_minor" bigint NOT NULL DEFAULT 0,
  "impressions" bigint NOT NULL DEFAULT 0,
  "reach" bigint NOT NULL DEFAULT 0,
  "clicks" bigint NOT NULL DEFAULT 0,
  "conversions" numeric(18,4) NOT NULL DEFAULT 0,
  "leads" numeric(18,4) NOT NULL DEFAULT 0,
  "purchases" numeric(18,4) NOT NULL DEFAULT 0,
  "revenue_minor" bigint NOT NULL DEFAULT 0,
  "frequency" numeric(18,6) NOT NULL DEFAULT 0,
  "provider_response" jsonb NOT NULL DEFAULT '{}',
  "synced_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("ad_campaign_id", "metric_date"),
  CONSTRAINT "ad_campaign_metrics_daily_provider_response_check" CHECK (jsonb_typeof(provider_response) = 'object'::text)
);
-- Create "ad_campaigns" table
CREATE TABLE "public"."ad_campaigns" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "campaign_id" uuid NOT NULL,
  "meta_ad_account_id" uuid NOT NULL,
  "social_account_id" uuid NOT NULL,
  "meta_pixel_id" uuid NULL,
  "name" text NOT NULL,
  "objective" text NOT NULL,
  "buying_type" text NOT NULL DEFAULT 'AUCTION',
  "daily_budget_minor" bigint NULL,
  "lifetime_budget_minor" bigint NULL,
  "campaign_spend_cap_minor" bigint NOT NULL,
  "currency" character(3) NOT NULL,
  "starts_at" timestamptz NULL,
  "ends_at" timestamptz NULL,
  "audience" jsonb NOT NULL,
  "placements" text[] NOT NULL DEFAULT '{}',
  "destination_url" text NOT NULL,
  "utm_parameters" jsonb NOT NULL DEFAULT '{}',
  "conversion_event" text NULL,
  "provider_campaign_id" text NULL,
  "status" "public"."meta_ad_campaign_status" NOT NULL DEFAULT 'DRAFT',
  "campaign_hash" text NOT NULL,
  "last_error_code" text NULL,
  "last_error_message" text NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "created_by" uuid NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "ad_campaigns_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "ad_campaign_budget_mode" CHECK ((((daily_budget_minor IS NOT NULL))::integer + ((lifetime_budget_minor IS NOT NULL))::integer) = 1),
  CONSTRAINT "ad_campaign_dates" CHECK ((ends_at IS NULL) OR (starts_at IS NULL) OR (ends_at > starts_at)),
  CONSTRAINT "ad_campaign_provider_state" CHECK ((status = ANY (ARRAY['DRAFT'::public.meta_ad_campaign_status, 'APPROVAL_REQUIRED'::public.meta_ad_campaign_status, 'APPROVED'::public.meta_ad_campaign_status, 'CREATING'::public.meta_ad_campaign_status, 'FAILED'::public.meta_ad_campaign_status])) OR (provider_campaign_id IS NOT NULL)),
  CONSTRAINT "ad_campaigns_audience_check" CHECK (jsonb_typeof(audience) = 'object'::text),
  CONSTRAINT "ad_campaigns_campaign_spend_cap_minor_check" CHECK (campaign_spend_cap_minor > 0),
  CONSTRAINT "ad_campaigns_daily_budget_minor_check" CHECK ((daily_budget_minor IS NULL) OR (daily_budget_minor > 0)),
  CONSTRAINT "ad_campaigns_lifetime_budget_minor_check" CHECK ((lifetime_budget_minor IS NULL) OR (lifetime_budget_minor > 0)),
  CONSTRAINT "ad_campaigns_utm_parameters_check" CHECK (jsonb_typeof(utm_parameters) = 'object'::text),
  CONSTRAINT "ad_campaigns_version_check" CHECK (version > 0)
);
-- Create index "ad_campaigns_campaign_idx" to table: "ad_campaigns"
CREATE INDEX "ad_campaigns_campaign_idx" ON "public"."ad_campaigns" ("campaign_id", "created_at" DESC);
-- Create "ad_creatives" table
CREATE TABLE "public"."ad_creatives" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "ad_campaign_id" uuid NOT NULL,
  "media_asset_id" uuid NOT NULL,
  "thumbnail_asset_id" uuid NULL,
  "primary_text_variants" jsonb NOT NULL,
  "headline_variants" jsonb NOT NULL,
  "cta_variants" jsonb NOT NULL,
  "preview_spec" jsonb NOT NULL DEFAULT '{}',
  "provider_creative_id" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "ad_creatives_cta_variants_check" CHECK (jsonb_typeof(cta_variants) = 'array'::text),
  CONSTRAINT "ad_creatives_headline_variants_check" CHECK (jsonb_typeof(headline_variants) = 'array'::text),
  CONSTRAINT "ad_creatives_preview_spec_check" CHECK (jsonb_typeof(preview_spec) = 'object'::text),
  CONSTRAINT "ad_creatives_primary_text_variants_check" CHECK (jsonb_typeof(primary_text_variants) = 'array'::text)
);
-- Create "ad_metrics_daily" table
CREATE TABLE "public"."ad_metrics_daily" (
  "ad_id" uuid NOT NULL,
  "metric_date" date NOT NULL,
  "spend_minor" bigint NOT NULL DEFAULT 0,
  "impressions" bigint NOT NULL DEFAULT 0,
  "clicks" bigint NOT NULL DEFAULT 0,
  "conversions" numeric(18,4) NOT NULL DEFAULT 0,
  "provider_response" jsonb NOT NULL DEFAULT '{}',
  "synced_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("ad_id", "metric_date"),
  CONSTRAINT "ad_metrics_daily_provider_response_check" CHECK (jsonb_typeof(provider_response) = 'object'::text)
);
-- Create "ad_recommendations" table
CREATE TABLE "public"."ad_recommendations" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "campaign_id" uuid NULL,
  "ad_campaign_id" uuid NULL,
  "recommendation_type" text NOT NULL,
  "recommendation_hash" text NOT NULL,
  "input_snapshot" jsonb NOT NULL,
  "model" text NOT NULL,
  "output" text NOT NULL,
  "rationale" text NOT NULL,
  "status" "public"."recommendation_status" NOT NULL DEFAULT 'DRAFT',
  "reviewer_id" uuid NULL,
  "review_notes" text NOT NULL DEFAULT '',
  "reviewed_at" timestamptz NULL,
  "action_taken" text NOT NULL DEFAULT '',
  "version" bigint NOT NULL DEFAULT 1,
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "ad_recommendations_recommendation_hash_key" UNIQUE ("recommendation_hash"),
  CONSTRAINT "ad_recommendations_input_snapshot_check" CHECK (jsonb_typeof(input_snapshot) = 'object'::text),
  CONSTRAINT "ad_recommendations_recommendation_type_check" CHECK (recommendation_type = ANY (ARRAY['PAUSE_WEAK_CREATIVE'::text, 'SCALE_WINNER'::text, 'INVESTIGATE_LOW_CTR'::text, 'INVESTIGATE_RISING_CPC'::text, 'INVESTIGATE_HIGH_FREQUENCY'::text, 'INVESTIGATE_CREATIVE_FATIGUE'::text, 'INVESTIGATE_CPA'::text, 'SUGGEST_HOOK'::text, 'SUGGEST_CTA'::text, 'SHORTEN_SCENE'::text, 'LOWEST_COST_TEMPLATE'::text, 'NEXT_CAMPAIGN_DIRECTION'::text])),
  CONSTRAINT "ad_recommendations_version_check" CHECK (version > 0),
  CONSTRAINT "recommendation_review_shape" CHECK ((status = 'DRAFT'::public.recommendation_status) = (reviewed_at IS NULL))
);
-- Create index "ad_recommendations_scope_status_idx" to table: "ad_recommendations"
CREATE INDEX "ad_recommendations_scope_status_idx" ON "public"."ad_recommendations" ("client_id", "workspace_id", "status", "created_at" DESC);
-- Create "ad_set_metrics_daily" table
CREATE TABLE "public"."ad_set_metrics_daily" (
  "ad_set_id" uuid NOT NULL,
  "metric_date" date NOT NULL,
  "spend_minor" bigint NOT NULL DEFAULT 0,
  "impressions" bigint NOT NULL DEFAULT 0,
  "clicks" bigint NOT NULL DEFAULT 0,
  "conversions" numeric(18,4) NOT NULL DEFAULT 0,
  "provider_response" jsonb NOT NULL DEFAULT '{}',
  "synced_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("ad_set_id", "metric_date"),
  CONSTRAINT "ad_set_metrics_daily_provider_response_check" CHECK (jsonb_typeof(provider_response) = 'object'::text)
);
-- Create "ad_sets" table
CREATE TABLE "public"."ad_sets" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "ad_campaign_id" uuid NOT NULL,
  "name" text NOT NULL,
  "audience" jsonb NOT NULL,
  "placements" text[] NOT NULL DEFAULT '{}',
  "optimization_goal" text NOT NULL,
  "billing_event" text NOT NULL DEFAULT 'IMPRESSIONS',
  "provider_ad_set_id" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "ad_sets_audience_check" CHECK (jsonb_typeof(audience) = 'object'::text)
);
-- Create "ads" table
CREATE TABLE "public"."ads" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "ad_campaign_id" uuid NOT NULL,
  "ad_set_id" uuid NOT NULL,
  "ad_creative_id" uuid NOT NULL,
  "name" text NOT NULL,
  "provider_ad_id" text NULL,
  "status" text NOT NULL DEFAULT 'PAUSED',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "ads_status_check" CHECK (status = ANY (ARRAY['PAUSED'::text, 'ACTIVE'::text, 'ARCHIVED'::text, 'FAILED'::text]))
);
-- Create "approval_events" table
CREATE TABLE "public"."approval_events" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "approval_id" uuid NOT NULL,
  "event_type" "public"."approval_event_type" NOT NULL,
  "actor_id" uuid NULL,
  "entity_version" bigint NOT NULL,
  "entity_hash" text NOT NULL,
  "notes" text NOT NULL DEFAULT '',
  "occurred_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "approval_events_entity_version_check" CHECK (entity_version > 0)
);
-- Create "approvals" table
CREATE TABLE "public"."approvals" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "campaign_id" uuid NULL,
  "entity_type" text NOT NULL,
  "entity_id" uuid NOT NULL,
  "entity_version" bigint NOT NULL,
  "entity_hash" text NOT NULL,
  "status" "public"."planning_content_status" NOT NULL DEFAULT 'DRAFT',
  "requested_by" uuid NULL,
  "requested_at" timestamptz NOT NULL DEFAULT now(),
  "decided_by" uuid NULL,
  "decided_at" timestamptz NULL,
  "notes" text NOT NULL DEFAULT '',
  "invalidated_at" timestamptz NULL,
  "invalidation_reason" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "approvals_decision_consistency" CHECK ((status = 'DRAFT'::public.planning_content_status) = (decided_at IS NULL)),
  CONSTRAINT "approvals_entity_type_check" CHECK (entity_type = ANY (ARRAY['CAMPAIGN'::text, 'CONCEPT'::text, 'CONTENT_VARIANT'::text, 'SCRIPT'::text, 'SCENE'::text, 'SCENE_GENERATION'::text, 'FINAL_RENDER'::text, 'SOCIAL_POST'::text, 'AD_CAMPAIGN'::text, 'BUDGET_CHANGE'::text, 'RECOMMENDATION'::text])),
  CONSTRAINT "approvals_entity_version_check" CHECK (entity_version > 0),
  CONSTRAINT "approvals_invalidation_pair" CHECK ((invalidated_at IS NULL) = (invalidation_reason IS NULL))
);
-- Create index "approvals_active_entity_idx" to table: "approvals"
CREATE UNIQUE INDEX "approvals_active_entity_idx" ON "public"."approvals" ("entity_type", "entity_id", "entity_version") WHERE (invalidated_at IS NULL);
-- Create index "approvals_campaign_idx" to table: "approvals"
CREATE INDEX "approvals_campaign_idx" ON "public"."approvals" ("campaign_id", "requested_at" DESC);
-- Create "audit_logs" table
CREATE TABLE "public"."audit_logs" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "actor_internal_user_id" uuid NULL,
  "action" text NOT NULL,
  "entity_type" text NULL,
  "entity_id" uuid NULL,
  "client_id" uuid NULL,
  "workspace_id" uuid NULL,
  "request_id" text NOT NULL,
  "ip_address" inet NULL,
  "user_agent" text NOT NULL DEFAULT '',
  "outcome" text NOT NULL,
  "reason" text NULL,
  "before_data" jsonb NULL,
  "after_data" jsonb NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "occurred_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "audit_logs_action_length" CHECK ((length(action) >= 1) AND (length(action) <= 160)),
  CONSTRAINT "audit_logs_outcome_check" CHECK (outcome = ANY (ARRAY['SUCCESS'::text, 'FAILURE'::text, 'DENIED'::text])),
  CONSTRAINT "audit_logs_request_id_length" CHECK ((length(request_id) >= 1) AND (length(request_id) <= 200))
);
-- Create index "audit_logs_actor_time_idx" to table: "audit_logs"
CREATE INDEX "audit_logs_actor_time_idx" ON "public"."audit_logs" ("actor_internal_user_id", "occurred_at" DESC);
-- Create index "audit_logs_entity_time_idx" to table: "audit_logs"
CREATE INDEX "audit_logs_entity_time_idx" ON "public"."audit_logs" ("entity_type", "entity_id", "occurred_at" DESC);
-- Create index "audit_logs_workspace_time_idx" to table: "audit_logs"
CREATE INDEX "audit_logs_workspace_time_idx" ON "public"."audit_logs" ("workspace_id", "occurred_at" DESC);
-- Create "brand_versions" table
CREATE TABLE "public"."brand_versions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "brand_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "version" integer NOT NULL,
  "logo_asset_ids" uuid[] NOT NULL DEFAULT '{}',
  "primary_color" text NULL,
  "secondary_color" text NULL,
  "background_color" text NULL,
  "heading_font" text NULL,
  "body_font" text NULL,
  "tone_of_voice" text NOT NULL DEFAULT '',
  "primary_language" text NOT NULL,
  "target_audience" text NOT NULL DEFAULT '',
  "main_message" text NOT NULL DEFAULT '',
  "default_cta" text NOT NULL DEFAULT '',
  "website" text NULL,
  "phone_number" text NULL,
  "preferred_terminology" text[] NOT NULL DEFAULT '{}',
  "prohibited_terminology" text[] NOT NULL DEFAULT '{}',
  "default_disclaimer" text NOT NULL DEFAULT '',
  "default_video_style" text NOT NULL DEFAULT '',
  "default_music_style" text NOT NULL DEFAULT '',
  "change_summary" text NOT NULL DEFAULT '',
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "brand_versions_brand_id_version_key" UNIQUE ("brand_id", "version"),
  CONSTRAINT "brand_colors_hex" CHECK (((primary_color IS NULL) OR (primary_color ~ '^#[0-9A-Fa-f]{6}$'::text)) AND ((secondary_color IS NULL) OR (secondary_color ~ '^#[0-9A-Fa-f]{6}$'::text)) AND ((background_color IS NULL) OR (background_color ~ '^#[0-9A-Fa-f]{6}$'::text))),
  CONSTRAINT "brand_versions_primary_language_check" CHECK (primary_language = ANY (ARRAY['vi'::text, 'en'::text])),
  CONSTRAINT "brand_versions_version_check" CHECK (version > 0)
);
-- Create index "brand_versions_scope_idx" to table: "brand_versions"
CREATE INDEX "brand_versions_scope_idx" ON "public"."brand_versions" ("client_id", "workspace_id", "brand_id", "version" DESC);
-- Create "brands" table
CREATE TABLE "public"."brands" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "name" text NOT NULL,
  "status" "public"."lifecycle_status" NOT NULL DEFAULT 'ACTIVE',
  "current_version" integer NOT NULL DEFAULT 1,
  "version" bigint NOT NULL DEFAULT 1,
  "archived_at" timestamptz NULL,
  "created_by" uuid NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "brands_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "brands_archive_consistency" CHECK ((status = 'ARCHIVED'::public.lifecycle_status) = (archived_at IS NOT NULL)),
  CONSTRAINT "brands_current_version_check" CHECK (current_version > 0),
  CONSTRAINT "brands_name_length" CHECK ((length(name) >= 2) AND (length(name) <= 160)),
  CONSTRAINT "brands_version_check" CHECK (version > 0)
);
-- Create index "brands_scope_status_idx" to table: "brands"
CREATE INDEX "brands_scope_status_idx" ON "public"."brands" ("client_id", "workspace_id", "status", "name");
-- Create "campaign_characters" table
CREATE TABLE "public"."campaign_characters" (
  "campaign_id" uuid NOT NULL,
  "character_id" uuid NOT NULL,
  "role" text NOT NULL,
  "selected_by" uuid NOT NULL,
  "selected_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("campaign_id", "role"),
  CONSTRAINT "campaign_characters_campaign_id_character_id_key" UNIQUE ("campaign_id", "character_id"),
  CONSTRAINT "campaign_characters_role_check" CHECK (role = ANY (ARRAY['PRIMARY'::text, 'LISTENER'::text]))
);
-- Create "campaign_concept_versions" table
CREATE TABLE "public"."campaign_concept_versions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "concept_id" uuid NOT NULL,
  "version" integer NOT NULL,
  "title" text NOT NULL,
  "video_format" text NOT NULL,
  "payload" jsonb NOT NULL,
  "output_hash" text NOT NULL,
  "change_summary" text NOT NULL DEFAULT '',
  "created_by" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "campaign_concept_versions_concept_id_version_key" UNIQUE ("concept_id", "version"),
  CONSTRAINT "campaign_concept_versions_payload_check" CHECK (jsonb_typeof(payload) = 'object'::text),
  CONSTRAINT "campaign_concept_versions_version_check" CHECK (version > 0)
);
-- Create "campaign_concepts" table
CREATE TABLE "public"."campaign_concepts" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "campaign_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "title" text NOT NULL,
  "video_format" text NOT NULL,
  "status" "public"."concept_status" NOT NULL DEFAULT 'DRAFT',
  "payload" jsonb NOT NULL,
  "current_version" integer NOT NULL DEFAULT 1,
  "prompt_version" text NOT NULL,
  "model" text NOT NULL,
  "request_id" text NOT NULL,
  "output_hash" text NOT NULL,
  "estimated_cost_usd" numeric(18,6) NOT NULL DEFAULT 0,
  "locked_at" timestamptz NULL,
  "locked_by" uuid NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "created_by" uuid NULL,
  "updated_by" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "campaign_concepts_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "campaign_concepts_current_version_check" CHECK (current_version > 0),
  CONSTRAINT "campaign_concepts_estimated_cost_usd_check" CHECK (estimated_cost_usd >= (0)::numeric),
  CONSTRAINT "campaign_concepts_lock_consistency" CHECK ((status = 'LOCKED'::public.concept_status) = (locked_at IS NOT NULL)),
  CONSTRAINT "campaign_concepts_payload_check" CHECK (jsonb_typeof(payload) = 'object'::text),
  CONSTRAINT "campaign_concepts_title_length" CHECK ((length(title) >= 2) AND (length(title) <= 200)),
  CONSTRAINT "campaign_concepts_version_check" CHECK (version > 0),
  CONSTRAINT "campaign_concepts_video_format_check" CHECK (video_format = ANY (ARRAY['INTERVIEW_REVIEW'::text, 'PROBLEM_SOLUTION'::text]))
);
-- Create index "campaign_concepts_campaign_idx" to table: "campaign_concepts"
CREATE INDEX "campaign_concepts_campaign_idx" ON "public"."campaign_concepts" ("campaign_id", "status", "created_at");
-- Create "campaign_content_variant_versions" table
CREATE TABLE "public"."campaign_content_variant_versions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "content_variant_id" uuid NOT NULL,
  "version" integer NOT NULL,
  "content" text NOT NULL,
  "content_hash" text NOT NULL,
  "change_summary" text NOT NULL DEFAULT '',
  "created_by" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "campaign_content_variant_version_content_variant_id_version_key" UNIQUE ("content_variant_id", "version"),
  CONSTRAINT "campaign_content_variant_versions_version_check" CHECK (version > 0)
);
-- Create "campaign_content_variants" table
CREATE TABLE "public"."campaign_content_variants" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "campaign_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "variant_key" text NOT NULL,
  "platform" text NOT NULL,
  "content" text NOT NULL,
  "status" "public"."planning_content_status" NOT NULL DEFAULT 'DRAFT',
  "current_version" integer NOT NULL DEFAULT 1,
  "content_hash" text NOT NULL,
  "prompt_version" text NOT NULL,
  "model" text NOT NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "approved_at" timestamptz NULL,
  "approved_by" uuid NULL,
  "created_by" uuid NULL,
  "updated_by" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "campaign_content_variants_campaign_id_variant_key_key" UNIQUE ("campaign_id", "variant_key"),
  CONSTRAINT "campaign_content_variants_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "campaign_content_approval_consistency" CHECK ((status = 'APPROVED'::public.planning_content_status) = (approved_at IS NOT NULL)),
  CONSTRAINT "campaign_content_variant_key" CHECK (variant_key ~ '^[a-z][a-z0-9_]{1,79}$'::text),
  CONSTRAINT "campaign_content_variants_current_version_check" CHECK (current_version > 0),
  CONSTRAINT "campaign_content_variants_version_check" CHECK (version > 0)
);
-- Create index "campaign_content_campaign_idx" to table: "campaign_content_variants"
CREATE INDEX "campaign_content_campaign_idx" ON "public"."campaign_content_variants" ("campaign_id", "variant_key");
-- Create "campaign_versions" table
CREATE TABLE "public"."campaign_versions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "campaign_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "version" integer NOT NULL,
  "objective" "public"."campaign_objective" NOT NULL,
  "target_audience" text NOT NULL,
  "market" text NOT NULL,
  "country" character(2) NOT NULL,
  "language" character(2) NOT NULL,
  "social_platform_targets" text[] NOT NULL,
  "video_format" text NOT NULL,
  "duration_seconds" integer NOT NULL,
  "aspect_ratio" text NOT NULL DEFAULT '9:16',
  "tone" text NOT NULL,
  "offer" text NOT NULL DEFAULT '',
  "cta" text NOT NULL,
  "planned_ads_budget" numeric(18,2) NULL,
  "budget_currency" character(3) NULL,
  "starts_on" date NULL,
  "ends_on" date NULL,
  "change_summary" text NOT NULL DEFAULT '',
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "campaign_versions_campaign_id_version_key" UNIQUE ("campaign_id", "version"),
  CONSTRAINT "campaign_versions_aspect_ratio_check" CHECK (aspect_ratio = '9:16'::text),
  CONSTRAINT "campaign_versions_budget_currency" CHECK ((planned_ads_budget IS NULL) = (budget_currency IS NULL)),
  CONSTRAINT "campaign_versions_dates" CHECK ((ends_on IS NULL) OR (starts_on IS NULL) OR (ends_on >= starts_on)),
  CONSTRAINT "campaign_versions_duration_seconds_check" CHECK (duration_seconds = ANY (ARRAY[30, 45])),
  CONSTRAINT "campaign_versions_language_check" CHECK (language = ANY (ARRAY['vi'::bpchar, 'en'::bpchar])),
  CONSTRAINT "campaign_versions_planned_ads_budget_check" CHECK ((planned_ads_budget IS NULL) OR (planned_ads_budget >= (0)::numeric)),
  CONSTRAINT "campaign_versions_social_platform_targets_check" CHECK ((cardinality(social_platform_targets) >= 1) AND (cardinality(social_platform_targets) <= 3)),
  CONSTRAINT "campaign_versions_version_check" CHECK (version > 0),
  CONSTRAINT "campaign_versions_video_format_check" CHECK (video_format = ANY (ARRAY['INTERVIEW_REVIEW'::text, 'PROBLEM_SOLUTION'::text]))
);
-- Create index "campaign_versions_scope_idx" to table: "campaign_versions"
CREATE INDEX "campaign_versions_scope_idx" ON "public"."campaign_versions" ("client_id", "workspace_id", "campaign_id", "version" DESC);
-- Create "campaigns" table
CREATE TABLE "public"."campaigns" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "brand_id" uuid NOT NULL,
  "product_id" uuid NOT NULL,
  "name" text NOT NULL,
  "status" "public"."campaign_status" NOT NULL DEFAULT 'DRAFT',
  "current_version" integer NOT NULL DEFAULT 1,
  "selected_concept_id" uuid NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "archived_at" timestamptz NULL,
  "created_by" uuid NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "campaigns_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "campaigns_workspace_id_name_key" UNIQUE ("workspace_id", "name"),
  CONSTRAINT "campaigns_archive_consistency" CHECK ((status = 'ARCHIVED'::public.campaign_status) = (archived_at IS NOT NULL)),
  CONSTRAINT "campaigns_current_version_check" CHECK (current_version > 0),
  CONSTRAINT "campaigns_name_length" CHECK ((length(name) >= 2) AND (length(name) <= 200)),
  CONSTRAINT "campaigns_version_check" CHECK (version > 0)
);
-- Create index "campaigns_scope_status_idx" to table: "campaigns"
CREATE INDEX "campaigns_scope_status_idx" ON "public"."campaigns" ("client_id", "workspace_id", "status", "updated_at" DESC);
-- Create "character_assets" table
CREATE TABLE "public"."character_assets" (
  "character_id" uuid NOT NULL,
  "media_asset_id" uuid NOT NULL,
  "purpose" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("character_id", "media_asset_id", "purpose"),
  CONSTRAINT "character_assets_purpose_check" CHECK (purpose = ANY (ARRAY['PREVIEW'::text, 'REFERENCE_IMAGE'::text, 'REFERENCE_VIDEO'::text, 'VOICE_REFERENCE'::text]))
);
-- Create "character_consents" table
CREATE TABLE "public"."character_consents" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "character_id" uuid NOT NULL,
  "status" "public"."consent_status" NOT NULL,
  "artifact_asset_id" uuid NULL,
  "subject_name" text NOT NULL,
  "granted_at" timestamptz NULL,
  "expires_at" timestamptz NULL,
  "revoked_at" timestamptz NULL,
  "notes" text NOT NULL DEFAULT '',
  "recorded_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "character_consents_dates" CHECK ((expires_at IS NULL) OR (granted_at IS NULL) OR (expires_at > granted_at))
);
-- Create "characters" table
CREATE TABLE "public"."characters" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NULL,
  "workspace_id" uuid NULL,
  "name" text NOT NULL,
  "provider" text NOT NULL,
  "provider_asset_id" text NULL,
  "character_type" "public"."character_type" NOT NULL,
  "gender_presentation" text NOT NULL DEFAULT '',
  "approximate_age_range" text NOT NULL DEFAULT '',
  "appearance_description" text NOT NULL,
  "wardrobe" text NOT NULL DEFAULT '',
  "gesture_style" text NOT NULL DEFAULT '',
  "default_role" text NOT NULL DEFAULT '',
  "supported_languages" text[] NOT NULL DEFAULT '{}',
  "consent_status" "public"."consent_status" NOT NULL,
  "preview_asset_id" uuid NULL,
  "status" "public"."lifecycle_status" NOT NULL DEFAULT 'ACTIVE',
  "version" bigint NOT NULL DEFAULT 1,
  "created_by" uuid NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "characters_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "characters_consent_guard" CHECK (((character_type = 'PRESET'::public.character_type) AND (consent_status = 'NOT_REQUIRED'::public.consent_status)) OR ((character_type = 'TRUSTED_GENERATED'::public.character_type) AND (consent_status = ANY (ARRAY['NOT_REQUIRED'::public.consent_status, 'APPROVED'::public.consent_status]))) OR ((character_type = 'AUTHORIZED_REAL_PERSON'::public.character_type) AND (consent_status = ANY (ARRAY['PENDING'::public.consent_status, 'APPROVED'::public.consent_status, 'REVOKED'::public.consent_status, 'EXPIRED'::public.consent_status])))),
  CONSTRAINT "characters_name_length" CHECK ((length(name) >= 2) AND (length(name) <= 160)),
  CONSTRAINT "characters_scope_pair" CHECK ((client_id IS NULL) = (workspace_id IS NULL)),
  CONSTRAINT "characters_version_check" CHECK (version > 0)
);
-- Create index "characters_scope_status_idx" to table: "characters"
CREATE INDEX "characters_scope_status_idx" ON "public"."characters" ("client_id", "workspace_id", "status", "name");
-- Create "clients" table
CREATE TABLE "public"."clients" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "company_name" text NOT NULL,
  "contact_name" text NOT NULL DEFAULT '',
  "contact_email" text NULL,
  "phone" text NULL,
  "industry" text NOT NULL DEFAULT '',
  "market" text NOT NULL DEFAULT '',
  "internal_notes" text NOT NULL DEFAULT '',
  "status" "public"."lifecycle_status" NOT NULL DEFAULT 'ACTIVE',
  "future_tenant_owner_id" text NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "archived_at" timestamptz NULL,
  "created_by" uuid NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "clients_archive_consistency" CHECK ((status = 'ARCHIVED'::public.lifecycle_status) = (archived_at IS NOT NULL)),
  CONSTRAINT "clients_company_name_length" CHECK ((length(company_name) >= 2) AND (length(company_name) <= 200)),
  CONSTRAINT "clients_contact_email_length" CHECK ((contact_email IS NULL) OR (length(contact_email) <= 320)),
  CONSTRAINT "clients_phone_length" CHECK ((phone IS NULL) OR (length(phone) <= 40)),
  CONSTRAINT "clients_version_check" CHECK (version > 0)
);
-- Create index "clients_status_name_idx" to table: "clients"
CREATE INDEX "clients_status_name_idx" ON "public"."clients" ("status", "company_name");
-- Create "cost_estimates" table
CREATE TABLE "public"."cost_estimates" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "campaign_id" uuid NOT NULL,
  "operation" "public"."generation_operation" NOT NULL,
  "model" text NOT NULL,
  "currency" character(3) NOT NULL DEFAULT 'USD',
  "estimated_input_tokens" bigint NOT NULL DEFAULT 0,
  "estimated_output_tokens" bigint NOT NULL DEFAULT 0,
  "estimated_video_seconds" bigint NOT NULL DEFAULT 0,
  "estimated_cost" numeric(18,6) NOT NULL,
  "assumptions" jsonb NOT NULL DEFAULT '{}',
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "cost_estimates_assumptions_check" CHECK (jsonb_typeof(assumptions) = 'object'::text),
  CONSTRAINT "cost_estimates_estimated_cost_check" CHECK (estimated_cost >= (0)::numeric),
  CONSTRAINT "cost_estimates_estimated_input_tokens_check" CHECK (estimated_input_tokens >= 0),
  CONSTRAINT "cost_estimates_estimated_output_tokens_check" CHECK (estimated_output_tokens >= 0),
  CONSTRAINT "cost_estimates_estimated_video_seconds_check" CHECK (estimated_video_seconds >= 0)
);
-- Create index "cost_estimates_campaign_idx" to table: "cost_estimates"
CREATE INDEX "cost_estimates_campaign_idx" ON "public"."cost_estimates" ("campaign_id", "created_at" DESC);
-- Create "cost_records" table
CREATE TABLE "public"."cost_records" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "usage_ledger_id" uuid NULL,
  "client_id" uuid NULL,
  "workspace_id" uuid NULL,
  "campaign_id" uuid NULL,
  "category" text NOT NULL,
  "provider" text NOT NULL,
  "amount" numeric(18,6) NOT NULL,
  "currency" character(3) NOT NULL,
  "normalized_amount_usd" numeric(18,6) NOT NULL,
  "estimated" boolean NOT NULL DEFAULT false,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "occurred_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "cost_records_usage_ledger_id_category_key" UNIQUE ("usage_ledger_id", "category"),
  CONSTRAINT "cost_records_amount_check" CHECK (amount >= (0)::numeric),
  CONSTRAINT "cost_records_category_check" CHECK (category = ANY (ARRAY['LLM'::text, 'SEEDANCE'::text, 'TRANSCRIPTION'::text, 'RENDER'::text, 'STORAGE'::text, 'META'::text, 'OTHER'::text])),
  CONSTRAINT "cost_records_metadata_check" CHECK (jsonb_typeof(metadata) = 'object'::text),
  CONSTRAINT "cost_records_normalized_amount_usd_check" CHECK (normalized_amount_usd >= (0)::numeric)
);
-- Create index "cost_records_campaign_time_idx" to table: "cost_records"
CREATE INDEX "cost_records_campaign_time_idx" ON "public"."cost_records" ("campaign_id", "occurred_at" DESC);
-- Create index "cost_records_scope_time_idx" to table: "cost_records"
CREATE INDEX "cost_records_scope_time_idx" ON "public"."cost_records" ("client_id", "workspace_id", "occurred_at" DESC);
-- Create "exchange_rate_snapshots" table
CREATE TABLE "public"."exchange_rate_snapshots" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "base_currency" character(3) NOT NULL,
  "quote_currency" character(3) NOT NULL,
  "rate" numeric(24,10) NOT NULL,
  "source" text NOT NULL,
  "captured_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "exchange_rate_snapshots_base_currency_quote_currency_captur_key" UNIQUE ("base_currency", "quote_currency", "captured_at"),
  CONSTRAINT "exchange_rate_snapshots_check" CHECK (base_currency <> quote_currency),
  CONSTRAINT "exchange_rate_snapshots_rate_check" CHECK (rate > (0)::numeric)
);
-- Create "feature_flags" table
CREATE TABLE "public"."feature_flags" (
  "key" text NOT NULL,
  "description" text NOT NULL,
  "enabled" boolean NOT NULL DEFAULT false,
  "safe_config" jsonb NOT NULL DEFAULT '{}',
  "updated_by" uuid NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("key"),
  CONSTRAINT "feature_flags_key_format" CHECK (key ~ '^[a-z][a-z0-9_.-]{1,99}$'::text)
);
-- Create "generation_jobs" table
CREATE TABLE "public"."generation_jobs" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "campaign_id" uuid NOT NULL,
  "operation" "public"."generation_operation" NOT NULL,
  "status" "public"."generation_job_status" NOT NULL DEFAULT 'QUEUED',
  "river_job_id" bigint NULL,
  "idempotency_key_hash" bytea NOT NULL,
  "input_hash" text NOT NULL,
  "estimated_cost_usd" numeric(18,6) NOT NULL DEFAULT 0,
  "actual_cost_usd" numeric(18,6) NULL,
  "provider_request_id" uuid NULL,
  "output_summary" jsonb NOT NULL DEFAULT '{}',
  "error_code" text NULL,
  "error_message" text NULL,
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "started_at" timestamptz NULL,
  "completed_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "generation_jobs_campaign_id_operation_idempotency_key_hash_key" UNIQUE ("campaign_id", "operation", "idempotency_key_hash"),
  CONSTRAINT "generation_jobs_actual_cost_usd_check" CHECK ((actual_cost_usd IS NULL) OR (actual_cost_usd >= (0)::numeric)),
  CONSTRAINT "generation_jobs_estimated_cost_usd_check" CHECK (estimated_cost_usd >= (0)::numeric),
  CONSTRAINT "generation_jobs_output_summary_check" CHECK (jsonb_typeof(output_summary) = 'object'::text)
);
-- Create index "generation_jobs_status_idx" to table: "generation_jobs"
CREATE INDEX "generation_jobs_status_idx" ON "public"."generation_jobs" ("status", "created_at");
-- Create "idempotency_keys" table
CREATE TABLE "public"."idempotency_keys" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "namespace" text NOT NULL,
  "key_hash" bytea NOT NULL,
  "request_hash" bytea NOT NULL,
  "status" "public"."idempotency_status" NOT NULL DEFAULT 'PROCESSING',
  "response_status" integer NULL,
  "response_body" jsonb NULL,
  "locked_until" timestamptz NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "completed_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "idempotency_keys_namespace_key_hash_key" UNIQUE ("namespace", "key_hash"),
  CONSTRAINT "idempotency_keys_namespace_length" CHECK ((length(namespace) >= 1) AND (length(namespace) <= 120)),
  CONSTRAINT "idempotency_keys_response_status" CHECK ((response_status IS NULL) OR ((response_status >= 100) AND (response_status <= 599)))
);
-- Create index "idempotency_keys_expiry_idx" to table: "idempotency_keys"
CREATE INDEX "idempotency_keys_expiry_idx" ON "public"."idempotency_keys" ("expires_at");
-- Create "internal_users" table
CREATE TABLE "public"."internal_users" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "email" text NOT NULL,
  "display_name" text NOT NULL,
  "password_hash" text NOT NULL,
  "role" "public"."internal_user_role" NOT NULL,
  "status" "public"."internal_user_status" NOT NULL DEFAULT 'ACTIVE',
  "requires_password_change" boolean NOT NULL DEFAULT true,
  "failed_login_attempts" integer NOT NULL DEFAULT 0,
  "locked_until" timestamptz NULL,
  "last_login_at" timestamptz NULL,
  "password_changed_at" timestamptz NOT NULL DEFAULT now(),
  "version" bigint NOT NULL DEFAULT 1,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "internal_users_display_name_length" CHECK ((length(display_name) >= 2) AND (length(display_name) <= 120)),
  CONSTRAINT "internal_users_email_length" CHECK ((length(email) >= 3) AND (length(email) <= 320)),
  CONSTRAINT "internal_users_failed_login_attempts_check" CHECK (failed_login_attempts >= 0),
  CONSTRAINT "internal_users_version_check" CHECK (version > 0)
);
-- Create index "internal_users_email_unique" to table: "internal_users"
CREATE UNIQUE INDEX "internal_users_email_unique" ON "public"."internal_users" ((lower(email)));
-- Create index "internal_users_status_role_idx" to table: "internal_users"
CREATE INDEX "internal_users_status_role_idx" ON "public"."internal_users" ("status", "role");
-- Create "media_asset_tags" table
CREATE TABLE "public"."media_asset_tags" (
  "media_asset_id" uuid NOT NULL,
  "tag" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("media_asset_id", "tag"),
  CONSTRAINT "media_asset_tags_format" CHECK (tag ~ '^[[:alnum:]][[:alnum:] _-]{0,79}$'::text)
);
-- Create index "media_asset_tags_tag_idx" to table: "media_asset_tags"
CREATE INDEX "media_asset_tags_tag_idx" ON "public"."media_asset_tags" ("tag", "media_asset_id");
-- Create "media_asset_versions" table
CREATE TABLE "public"."media_asset_versions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "media_asset_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "version" integer NOT NULL,
  "storage_key" text NOT NULL,
  "original_filename" text NOT NULL,
  "mime_type" text NOT NULL,
  "file_extension" text NOT NULL,
  "file_size_bytes" bigint NOT NULL,
  "checksum_sha256" text NULL,
  "width" integer NULL,
  "height" integer NULL,
  "duration_ms" bigint NULL,
  "codec" text NULL,
  "bitrate_bps" bigint NULL,
  "thumbnail_storage_key" text NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "verified_at" timestamptz NULL,
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "media_asset_versions_media_asset_id_version_key" UNIQUE ("media_asset_id", "version"),
  CONSTRAINT "media_asset_versions_storage_key_key" UNIQUE ("storage_key"),
  CONSTRAINT "media_asset_versions_bitrate_bps_check" CHECK ((bitrate_bps IS NULL) OR (bitrate_bps > 0)),
  CONSTRAINT "media_asset_versions_duration_ms_check" CHECK ((duration_ms IS NULL) OR (duration_ms > 0)),
  CONSTRAINT "media_asset_versions_file_size_bytes_check" CHECK (file_size_bytes > 0),
  CONSTRAINT "media_asset_versions_height_check" CHECK ((height IS NULL) OR (height > 0)),
  CONSTRAINT "media_asset_versions_metadata_check" CHECK (jsonb_typeof(metadata) = 'object'::text),
  CONSTRAINT "media_asset_versions_version_check" CHECK (version > 0),
  CONSTRAINT "media_asset_versions_width_check" CHECK ((width IS NULL) OR (width > 0))
);
-- Create "media_assets" table
CREATE TABLE "public"."media_assets" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "brand_id" uuid NULL,
  "product_id" uuid NULL,
  "campaign_id" uuid NULL,
  "asset_type" "public"."media_asset_type" NOT NULL,
  "category" text NOT NULL DEFAULT '',
  "name" text NOT NULL,
  "folder" text NOT NULL DEFAULT '',
  "status" "public"."content_status" NOT NULL DEFAULT 'DRAFT',
  "current_version" integer NOT NULL DEFAULT 1,
  "usage_rights" text NOT NULL,
  "source_metadata" jsonb NOT NULL DEFAULT '{}',
  "expires_at" timestamptz NULL,
  "temporary_until" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "created_by" uuid NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "media_assets_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "media_assets_current_version_check" CHECK (current_version > 0),
  CONSTRAINT "media_assets_name_length" CHECK ((length(name) >= 1) AND (length(name) <= 240)),
  CONSTRAINT "media_assets_source_metadata_check" CHECK (jsonb_typeof(source_metadata) = 'object'::text),
  CONSTRAINT "media_assets_version_check" CHECK (version > 0)
);
-- Create index "media_assets_cleanup_idx" to table: "media_assets"
CREATE INDEX "media_assets_cleanup_idx" ON "public"."media_assets" ("temporary_until") WHERE ((temporary_until IS NOT NULL) AND (deleted_at IS NULL));
-- Create index "media_assets_scope_search_idx" to table: "media_assets"
CREATE INDEX "media_assets_scope_search_idx" ON "public"."media_assets" ("client_id", "workspace_id", "status", "asset_type", "name") WHERE (deleted_at IS NULL);
-- Create "media_uploads" table
CREATE TABLE "public"."media_uploads" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "media_asset_id" uuid NOT NULL,
  "storage_key" text NOT NULL,
  "multipart_upload_id" text NULL,
  "expected_filename" text NOT NULL,
  "expected_mime_type" text NOT NULL,
  "expected_extension" text NOT NULL,
  "expected_size_bytes" bigint NOT NULL,
  "status" "public"."upload_status" NOT NULL DEFAULT 'PENDING',
  "expires_at" timestamptz NOT NULL,
  "completed_at" timestamptz NULL,
  "failure_reason" text NULL,
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "media_uploads_storage_key_key" UNIQUE ("storage_key"),
  CONSTRAINT "media_uploads_expected_size_bytes_check" CHECK (expected_size_bytes > 0),
  CONSTRAINT "media_uploads_expiry" CHECK (expires_at > created_at)
);
-- Create index "media_uploads_cleanup_idx" to table: "media_uploads"
CREATE INDEX "media_uploads_cleanup_idx" ON "public"."media_uploads" ("status", "expires_at");
-- Create "meta_ad_accounts" table
CREATE TABLE "public"."meta_ad_accounts" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "meta_business_id" uuid NULL,
  "connection_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "provider_ad_account_id" text NOT NULL,
  "name" text NOT NULL,
  "currency" character(3) NOT NULL,
  "timezone_name" text NOT NULL,
  "account_status" integer NULL,
  "provider_spend_cap_minor" bigint NULL,
  "amount_spent_minor" bigint NULL,
  "last_synced_at" timestamptz NOT NULL DEFAULT now(),
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "meta_ad_accounts_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "meta_ad_accounts_workspace_id_provider_ad_account_id_key" UNIQUE ("workspace_id", "provider_ad_account_id")
);
-- Create "meta_ad_actions" table
CREATE TABLE "public"."meta_ad_actions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "ad_campaign_id" uuid NOT NULL,
  "action" "public"."meta_action_type" NOT NULL,
  "status" "public"."meta_action_status" NOT NULL DEFAULT 'PENDING_APPROVAL',
  "requested_budget_minor" bigint NULL,
  "previous_budget_minor" bigint NULL,
  "confirmation_text" text NOT NULL DEFAULT '',
  "action_hash" text NOT NULL,
  "idempotency_key" text NOT NULL,
  "river_job_id" bigint NULL,
  "requested_by" uuid NOT NULL,
  "reviewed_by" uuid NULL,
  "review_notes" text NOT NULL DEFAULT '',
  "safe_response" jsonb NOT NULL DEFAULT '{}',
  "error_code" text NULL,
  "error_message" text NULL,
  "requested_at" timestamptz NOT NULL DEFAULT now(),
  "reviewed_at" timestamptz NULL,
  "completed_at" timestamptz NULL,
  "version" bigint NOT NULL DEFAULT 1,
  PRIMARY KEY ("id"),
  CONSTRAINT "meta_ad_actions_ad_campaign_id_action_hash_key" UNIQUE ("ad_campaign_id", "action_hash"),
  CONSTRAINT "meta_ad_actions_idempotency_key_key" UNIQUE ("idempotency_key"),
  CONSTRAINT "meta_ad_actions_safe_response_check" CHECK (jsonb_typeof(safe_response) = 'object'::text),
  CONSTRAINT "meta_ad_actions_version_check" CHECK (version > 0)
);
-- Create "meta_ad_guardrails" table
CREATE TABLE "public"."meta_ad_guardrails" (
  "workspace_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_spend_cap_minor" bigint NOT NULL,
  "default_campaign_spend_cap_minor" bigint NOT NULL,
  "maximum_budget_increase_percent" numeric(6,2) NOT NULL DEFAULT 20,
  "currency" character(3) NOT NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "updated_by" uuid NOT NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("workspace_id"),
  CONSTRAINT "meta_ad_guardrails_check" CHECK (default_campaign_spend_cap_minor <= workspace_spend_cap_minor),
  CONSTRAINT "meta_ad_guardrails_default_campaign_spend_cap_minor_check" CHECK (default_campaign_spend_cap_minor > 0),
  CONSTRAINT "meta_ad_guardrails_maximum_budget_increase_percent_check" CHECK ((maximum_budget_increase_percent >= (0)::numeric) AND (maximum_budget_increase_percent <= (100)::numeric)),
  CONSTRAINT "meta_ad_guardrails_version_check" CHECK (version > 0),
  CONSTRAINT "meta_ad_guardrails_workspace_spend_cap_minor_check" CHECK (workspace_spend_cap_minor > 0)
);
-- Create "meta_audiences" table
CREATE TABLE "public"."meta_audiences" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "meta_ad_account_id" uuid NOT NULL,
  "provider_audience_id" text NOT NULL,
  "name" text NOT NULL,
  "audience_type" text NOT NULL,
  "subtype" text NULL,
  "approximate_count" bigint NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "meta_audiences_meta_ad_account_id_provider_audience_id_key" UNIQUE ("meta_ad_account_id", "provider_audience_id")
);
-- Create "meta_businesses" table
CREATE TABLE "public"."meta_businesses" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "connection_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "provider_business_id" text NOT NULL,
  "name" text NOT NULL,
  "verification_status" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "meta_businesses_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "meta_businesses_workspace_id_provider_business_id_key" UNIQUE ("workspace_id", "provider_business_id")
);
-- Create "meta_connections" table
CREATE TABLE "public"."meta_connections" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "meta_user_id" text NOT NULL,
  "display_name" text NOT NULL DEFAULT '',
  "token_ciphertext" bytea NOT NULL,
  "token_nonce" bytea NOT NULL,
  "token_type" text NOT NULL DEFAULT 'USER',
  "scopes" text[] NOT NULL DEFAULT '{}',
  "token_issued_at" timestamptz NULL,
  "token_expires_at" timestamptz NULL,
  "data_access_expires_at" timestamptz NULL,
  "api_version" text NOT NULL,
  "status" "public"."meta_connection_status" NOT NULL DEFAULT 'CONNECTED',
  "last_validated_at" timestamptz NULL,
  "last_error_code" text NULL,
  "last_error_message" text NULL,
  "disconnected_at" timestamptz NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "created_by" uuid NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "meta_connections_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "meta_connections_version_check" CHECK (version > 0)
);
-- Create index "meta_connections_expiry_idx" to table: "meta_connections"
CREATE INDEX "meta_connections_expiry_idx" ON "public"."meta_connections" ("token_expires_at") WHERE (disconnected_at IS NULL);
-- Create index "meta_connections_workspace_active_idx" to table: "meta_connections"
CREATE UNIQUE INDEX "meta_connections_workspace_active_idx" ON "public"."meta_connections" ("workspace_id") WHERE (disconnected_at IS NULL);
-- Create "meta_oauth_states" table
CREATE TABLE "public"."meta_oauth_states" (
  "state_hash" text NOT NULL,
  "actor_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "consumed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("state_hash")
);
-- Create "meta_pixels" table
CREATE TABLE "public"."meta_pixels" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "meta_ad_account_id" uuid NOT NULL,
  "provider_pixel_id" text NOT NULL,
  "name" text NOT NULL,
  "conversion_event" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "meta_pixels_meta_ad_account_id_provider_pixel_id_key" UNIQUE ("meta_ad_account_id", "provider_pixel_id")
);
-- Create "meta_webhook_events" table
CREATE TABLE "public"."meta_webhook_events" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "delivery_hash" text NOT NULL,
  "signature_valid" boolean NOT NULL,
  "object_type" text NOT NULL,
  "normalized_events" jsonb NOT NULL DEFAULT '[]',
  "processed_at" timestamptz NULL,
  "processing_error" text NULL,
  "received_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "meta_webhook_events_delivery_hash_key" UNIQUE ("delivery_hash"),
  CONSTRAINT "meta_webhook_events_normalized_events_check" CHECK (jsonb_typeof(normalized_events) = 'array'::text)
);
-- Create "notifications" table
CREATE TABLE "public"."notifications" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "internal_user_id" uuid NULL,
  "client_id" uuid NULL,
  "workspace_id" uuid NULL,
  "severity" "public"."notification_severity" NOT NULL DEFAULT 'INFO',
  "notification_type" text NOT NULL,
  "title" text NOT NULL,
  "message" text NOT NULL,
  "entity_type" text NULL,
  "entity_id" uuid NULL,
  "read_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- Create index "notifications_user_unread_idx" to table: "notifications"
CREATE INDEX "notifications_user_unread_idx" ON "public"."notifications" ("internal_user_id", "created_at" DESC) WHERE (read_at IS NULL);
-- Create "product_claim_sources" table
CREATE TABLE "public"."product_claim_sources" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "claim_id" uuid NOT NULL,
  "fact_id" uuid NULL,
  "media_asset_id" uuid NULL,
  "evidence_excerpt" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_claim_sources_claim_id_fact_id_media_asset_id_key" UNIQUE NULLS NOT DISTINCT ("claim_id", "fact_id", "media_asset_id"),
  CONSTRAINT "product_claim_sources_reference" CHECK ((fact_id IS NOT NULL) OR (media_asset_id IS NOT NULL))
);
-- Create "product_claim_versions" table
CREATE TABLE "public"."product_claim_versions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "product_claim_id" uuid NOT NULL,
  "version" bigint NOT NULL,
  "claim_text" text NOT NULL,
  "rationale" text NOT NULL DEFAULT '',
  "change_summary" text NOT NULL DEFAULT '',
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_claim_versions_product_claim_id_version_key" UNIQUE ("product_claim_id", "version"),
  CONSTRAINT "product_claim_versions_version_check" CHECK (version > 0)
);
-- Create "product_claims" table
CREATE TABLE "public"."product_claims" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "product_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "claim_kind" "public"."claim_kind" NOT NULL,
  "claim_text" text NOT NULL,
  "rationale" text NOT NULL DEFAULT '',
  "status" "public"."fact_status" NOT NULL DEFAULT 'DRAFT',
  "effective_from" timestamptz NULL,
  "expires_at" timestamptz NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "approved_by" uuid NULL,
  "approved_at" timestamptz NULL,
  "created_by" uuid NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_claims_approval_consistency" CHECK ((status = 'APPROVED'::public.fact_status) = (approved_at IS NOT NULL)),
  CONSTRAINT "product_claims_text_length" CHECK ((length(claim_text) >= 3) AND (length(claim_text) <= 2000)),
  CONSTRAINT "product_claims_version_check" CHECK (version > 0)
);
-- Create index "product_claims_truth_idx" to table: "product_claims"
CREATE INDEX "product_claims_truth_idx" ON "public"."product_claims" ("product_id", "status", "claim_kind");
-- Create "product_fact_versions" table
CREATE TABLE "public"."product_fact_versions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "product_fact_id" uuid NOT NULL,
  "version" bigint NOT NULL,
  "label" text NOT NULL,
  "exact_value" text NOT NULL,
  "normalized_value" jsonb NULL,
  "unit" text NULL,
  "source_name" text NOT NULL,
  "source_excerpt" text NOT NULL DEFAULT '',
  "source_asset_id" uuid NULL,
  "locked_value" boolean NOT NULL,
  "change_summary" text NOT NULL DEFAULT '',
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_fact_versions_product_fact_id_version_key" UNIQUE ("product_fact_id", "version"),
  CONSTRAINT "product_fact_versions_version_check" CHECK (version > 0)
);
-- Create "product_facts" table
CREATE TABLE "public"."product_facts" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "product_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "fact_key" text NOT NULL,
  "label" text NOT NULL,
  "exact_value" text NOT NULL,
  "normalized_value" jsonb NULL,
  "unit" text NULL,
  "source_name" text NOT NULL,
  "source_excerpt" text NOT NULL DEFAULT '',
  "source_asset_id" uuid NULL,
  "status" "public"."fact_status" NOT NULL DEFAULT 'DRAFT',
  "locked_value" boolean NOT NULL DEFAULT false,
  "effective_from" timestamptz NULL,
  "expires_at" timestamptz NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "approved_by" uuid NULL,
  "approved_at" timestamptz NULL,
  "created_by" uuid NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_facts_product_id_fact_key_key" UNIQUE ("product_id", "fact_key"),
  CONSTRAINT "product_facts_approval_consistency" CHECK ((status = 'APPROVED'::public.fact_status) = (approved_at IS NOT NULL)),
  CONSTRAINT "product_facts_effective_order" CHECK ((expires_at IS NULL) OR (effective_from IS NULL) OR (expires_at > effective_from)),
  CONSTRAINT "product_facts_key_format" CHECK (fact_key ~ '^[a-z][a-z0-9_.-]{1,99}$'::text),
  CONSTRAINT "product_facts_version_check" CHECK (version > 0)
);
-- Create index "product_facts_truth_idx" to table: "product_facts"
CREATE INDEX "product_facts_truth_idx" ON "public"."product_facts" ("product_id", "status", "locked_value", "fact_key");
-- Create "product_versions" table
CREATE TABLE "public"."product_versions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "product_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "version" integer NOT NULL,
  "short_description" text NOT NULL DEFAULT '',
  "long_description" text NOT NULL DEFAULT '',
  "features" text[] NOT NULL DEFAULT '{}',
  "benefits" text[] NOT NULL DEFAULT '{}',
  "differentiators" text[] NOT NULL DEFAULT '{}',
  "intended_audience" text NOT NULL DEFAULT '',
  "currency" character(3) NULL,
  "regular_price" numeric(18,2) NULL,
  "sale_price" numeric(18,2) NULL,
  "discount_code" text NULL,
  "offer_valid_from" timestamptz NULL,
  "offer_valid_until" timestamptz NULL,
  "variants" jsonb NOT NULL DEFAULT '[]',
  "change_summary" text NOT NULL DEFAULT '',
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_versions_product_id_version_key" UNIQUE ("product_id", "version"),
  CONSTRAINT "product_versions_offer_order" CHECK ((offer_valid_until IS NULL) OR (offer_valid_from IS NULL) OR (offer_valid_until > offer_valid_from)),
  CONSTRAINT "product_versions_price_order" CHECK ((sale_price IS NULL) OR (regular_price IS NULL) OR (sale_price <= regular_price)),
  CONSTRAINT "product_versions_regular_price_check" CHECK ((regular_price IS NULL) OR (regular_price >= (0)::numeric)),
  CONSTRAINT "product_versions_sale_price_check" CHECK ((sale_price IS NULL) OR (sale_price >= (0)::numeric)),
  CONSTRAINT "product_versions_variants_check" CHECK (jsonb_typeof(variants) = 'array'::text),
  CONSTRAINT "product_versions_version_check" CHECK (version > 0)
);
-- Create index "product_versions_scope_idx" to table: "product_versions"
CREATE INDEX "product_versions_scope_idx" ON "public"."product_versions" ("client_id", "workspace_id", "product_id", "version" DESC);
-- Create "product_vertical_data" table
CREATE TABLE "public"."product_vertical_data" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "product_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "vertical_key" text NOT NULL,
  "schema_version" integer NOT NULL,
  "data" jsonb NOT NULL,
  "data_hash" text NOT NULL,
  "validated_at" timestamptz NOT NULL,
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_vertical_data_product_id_schema_version_data_hash_key" UNIQUE ("product_id", "schema_version", "data_hash"),
  CONSTRAINT "product_vertical_data_data_check" CHECK (jsonb_typeof(data) = 'object'::text),
  CONSTRAINT "product_vertical_data_schema_version_check" CHECK (schema_version > 0)
);
-- Create "products" table
CREATE TABLE "public"."products" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "brand_id" uuid NULL,
  "name" text NOT NULL,
  "sku" text NOT NULL,
  "model" text NOT NULL DEFAULT '',
  "category" text NOT NULL,
  "vertical_key" text NOT NULL,
  "status" "public"."content_status" NOT NULL DEFAULT 'DRAFT',
  "current_version" integer NOT NULL DEFAULT 1,
  "version" bigint NOT NULL DEFAULT 1,
  "archived_at" timestamptz NULL,
  "created_by" uuid NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "products_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "products_workspace_id_sku_key" UNIQUE ("workspace_id", "sku"),
  CONSTRAINT "products_archive_consistency" CHECK ((status = 'ARCHIVED'::public.content_status) = (archived_at IS NOT NULL)),
  CONSTRAINT "products_current_version_check" CHECK (current_version > 0),
  CONSTRAINT "products_name_length" CHECK ((length(name) >= 2) AND (length(name) <= 200)),
  CONSTRAINT "products_sku_length" CHECK ((length(sku) >= 1) AND (length(sku) <= 100)),
  CONSTRAINT "products_status_check" CHECK (status = ANY (ARRAY['DRAFT'::public.content_status, 'APPROVED'::public.content_status, 'ARCHIVED'::public.content_status])),
  CONSTRAINT "products_version_check" CHECK (version > 0),
  CONSTRAINT "products_vertical_key_format" CHECK (vertical_key ~ '^[a-z][a-z0-9-]{1,79}$'::text)
);
-- Create index "products_scope_status_idx" to table: "products"
CREATE INDEX "products_scope_status_idx" ON "public"."products" ("client_id", "workspace_id", "status", "name");
-- Create "provider_outputs" table
CREATE TABLE "public"."provider_outputs" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "provider_request_id" uuid NOT NULL,
  "output_hash" text NOT NULL,
  "normalized_output" jsonb NOT NULL,
  "validation_errors" jsonb NOT NULL DEFAULT '[]',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "provider_outputs_provider_request_id_output_hash_key" UNIQUE ("provider_request_id", "output_hash"),
  CONSTRAINT "provider_outputs_normalized_output_check" CHECK (jsonb_typeof(normalized_output) = 'object'::text),
  CONSTRAINT "provider_outputs_validation_errors_check" CHECK (jsonb_typeof(validation_errors) = 'array'::text)
);
-- Create "provider_requests" table
CREATE TABLE "public"."provider_requests" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "campaign_id" uuid NULL,
  "provider" text NOT NULL,
  "operation" "public"."generation_operation" NOT NULL,
  "model" text NOT NULL,
  "prompt_version" text NOT NULL,
  "provider_request_id" text NULL,
  "input_hash" text NOT NULL,
  "status" "public"."provider_request_status" NOT NULL DEFAULT 'PENDING',
  "started_at" timestamptz NOT NULL DEFAULT now(),
  "completed_at" timestamptz NULL,
  "latency_ms" bigint NULL,
  "input_tokens" bigint NULL,
  "output_tokens" bigint NULL,
  "estimated_cost_usd" numeric(18,6) NULL,
  "actual_cost_usd" numeric(18,6) NULL,
  "error_code" text NULL,
  "error_message" text NULL,
  "created_by" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "provider_requests_actual_cost_usd_check" CHECK ((actual_cost_usd IS NULL) OR (actual_cost_usd >= (0)::numeric)),
  CONSTRAINT "provider_requests_estimated_cost_usd_check" CHECK ((estimated_cost_usd IS NULL) OR (estimated_cost_usd >= (0)::numeric)),
  CONSTRAINT "provider_requests_input_tokens_check" CHECK ((input_tokens IS NULL) OR (input_tokens >= 0)),
  CONSTRAINT "provider_requests_latency_ms_check" CHECK ((latency_ms IS NULL) OR (latency_ms >= 0)),
  CONSTRAINT "provider_requests_output_tokens_check" CHECK ((output_tokens IS NULL) OR (output_tokens >= 0))
);
-- Create index "provider_requests_campaign_idx" to table: "provider_requests"
CREATE INDEX "provider_requests_campaign_idx" ON "public"."provider_requests" ("campaign_id", "created_at" DESC);
-- Create "provider_webhook_deliveries" table
CREATE TABLE "public"."provider_webhook_deliveries" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "provider" text NOT NULL,
  "provider_task_id" text NOT NULL,
  "payload_hash" text NOT NULL,
  "request_id" text NOT NULL,
  "signature_valid" boolean NOT NULL,
  "status_code" integer NULL,
  "processed_at" timestamptz NULL,
  "processing_error" text NULL,
  "received_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "provider_webhook_deliveries_provider_provider_task_id_paylo_key" UNIQUE ("provider", "provider_task_id", "payload_hash")
);
-- Create "publish_jobs" table
CREATE TABLE "public"."publish_jobs" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "social_post_id" uuid NOT NULL,
  "idempotency_key" text NOT NULL,
  "river_job_id" bigint NULL,
  "attempt_count" integer NOT NULL DEFAULT 0,
  "last_error_retryable" boolean NULL,
  "safe_response" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "publish_jobs_idempotency_key_key" UNIQUE ("idempotency_key"),
  CONSTRAINT "publish_jobs_social_post_id_key" UNIQUE ("social_post_id"),
  CONSTRAINT "publish_jobs_attempt_count_check" CHECK (attempt_count >= 0),
  CONSTRAINT "publish_jobs_safe_response_check" CHECK (jsonb_typeof(safe_response) = 'object'::text)
);
-- Create "render_jobs" table
CREATE TABLE "public"."render_jobs" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "campaign_id" uuid NOT NULL,
  "video_project_id" uuid NOT NULL,
  "video_project_version" integer NOT NULL,
  "render_manifest_id" uuid NULL,
  "status" "public"."render_job_status" NOT NULL DEFAULT 'QUEUED',
  "idempotency_key" text NOT NULL,
  "river_job_id" bigint NULL,
  "output_asset_id" uuid NULL,
  "thumbnail_storage_key" text NULL,
  "srt_storage_key" text NULL,
  "vtt_storage_key" text NULL,
  "output_hash" text NULL,
  "renderer_request_id" text NULL,
  "sanitized_response" jsonb NOT NULL DEFAULT '{}',
  "error_code" text NULL,
  "error_message" text NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "started_at" timestamptz NULL,
  "completed_at" timestamptz NULL,
  "reviewed_at" timestamptz NULL,
  "reviewed_by" uuid NULL,
  "review_notes" text NOT NULL DEFAULT '',
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "render_jobs_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "render_jobs_idempotency_key_key" UNIQUE ("idempotency_key"),
  CONSTRAINT "render_jobs_output_state" CHECK ((status <> ALL (ARRAY['REVIEW_REQUIRED'::public.render_job_status, 'APPROVED'::public.render_job_status, 'REJECTED'::public.render_job_status])) OR (output_asset_id IS NOT NULL)),
  CONSTRAINT "render_jobs_sanitized_response_check" CHECK (jsonb_typeof(sanitized_response) = 'object'::text),
  CONSTRAINT "render_jobs_version_check" CHECK (version > 0)
);
-- Create index "render_jobs_campaign_idx" to table: "render_jobs"
CREATE INDEX "render_jobs_campaign_idx" ON "public"."render_jobs" ("campaign_id", "created_at" DESC);
-- Create "render_manifests" table
CREATE TABLE "public"."render_manifests" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "video_project_id" uuid NOT NULL,
  "video_project_version" integer NOT NULL,
  "manifest_version" integer NOT NULL DEFAULT 1,
  "manifest_hash" text NOT NULL,
  "manifest" jsonb NOT NULL,
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "render_manifests_video_project_id_video_project_version_man_key" UNIQUE ("video_project_id", "video_project_version", "manifest_hash"),
  CONSTRAINT "render_manifests_manifest_check" CHECK (jsonb_typeof(manifest) = 'object'::text),
  CONSTRAINT "render_manifests_manifest_version_check" CHECK (manifest_version > 0)
);
-- Create "scene_assets" table
CREATE TABLE "public"."scene_assets" (
  "scene_version_id" uuid NOT NULL,
  "media_asset_id" uuid NOT NULL,
  "role" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("scene_version_id", "media_asset_id", "role"),
  CONSTRAINT "scene_assets_role_check" CHECK (role = ANY (ARRAY['PRODUCT_REFERENCE'::text, 'CHARACTER_REFERENCE'::text, 'AUDIO_REFERENCE'::text, 'VIDEO_REFERENCE'::text]))
);
-- Create "scene_generation_edits" table
CREATE TABLE "public"."scene_generation_edits" (
  "generation_task_id" uuid NOT NULL,
  "trim_start_ms" bigint NOT NULL DEFAULT 0,
  "trim_end_ms" bigint NULL,
  "mute_audio" boolean NOT NULL DEFAULT false,
  "transition" text NOT NULL DEFAULT 'CUT',
  "replacement_asset_id" uuid NULL,
  "attached_product_asset_ids" uuid[] NOT NULL DEFAULT '{}',
  "subtitle_preview" boolean NOT NULL DEFAULT true,
  "version" bigint NOT NULL DEFAULT 1,
  "updated_by" uuid NULL,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("generation_task_id"),
  CONSTRAINT "scene_generation_edit_trim" CHECK ((trim_end_ms IS NULL) OR (trim_end_ms > trim_start_ms)),
  CONSTRAINT "scene_generation_edits_transition_check" CHECK (transition = ANY (ARRAY['CUT'::text, 'CROSSFADE'::text, 'FADE_TO_BLACK'::text])),
  CONSTRAINT "scene_generation_edits_trim_end_ms_check" CHECK ((trim_end_ms IS NULL) OR (trim_end_ms > 0)),
  CONSTRAINT "scene_generation_edits_trim_start_ms_check" CHECK (trim_start_ms >= 0),
  CONSTRAINT "scene_generation_edits_version_check" CHECK (version > 0)
);
-- Create "scene_generation_events" table
CREATE TABLE "public"."scene_generation_events" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "generation_task_id" uuid NOT NULL,
  "from_status" "public"."scene_generation_status" NULL,
  "to_status" "public"."scene_generation_status" NOT NULL,
  "actor_id" uuid NULL,
  "source" text NOT NULL,
  "provider_request_id" text NULL,
  "safe_detail" text NOT NULL DEFAULT '',
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "occurred_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "scene_generation_events_metadata_check" CHECK (jsonb_typeof(metadata) = 'object'::text),
  CONSTRAINT "scene_generation_events_source_check" CHECK (source = ANY (ARRAY['API'::text, 'WORKER'::text, 'PROVIDER_WEBHOOK'::text, 'PROVIDER_POLL'::text, 'SYSTEM'::text]))
);
-- Create index "scene_generation_events_task_idx" to table: "scene_generation_events"
CREATE INDEX "scene_generation_events_task_idx" ON "public"."scene_generation_events" ("generation_task_id", "occurred_at");
-- Create "scene_generation_tasks" table
CREATE TABLE "public"."scene_generation_tasks" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "campaign_id" uuid NOT NULL,
  "scene_id" uuid NOT NULL,
  "scene_version" integer NOT NULL,
  "provider" text NOT NULL,
  "provider_task_id" text NULL,
  "status" "public"."scene_generation_status" NOT NULL DEFAULT 'QUEUED',
  "idempotency_key" text NOT NULL,
  "attempt_number" integer NOT NULL DEFAULT 1,
  "model" text NOT NULL,
  "api_version" text NOT NULL,
  "resolution" text NOT NULL,
  "aspect_ratio" text NOT NULL,
  "duration_seconds" integer NOT NULL,
  "generate_audio" boolean NOT NULL,
  "scene_hash" text NOT NULL,
  "prompt_hash" text NOT NULL,
  "reference_hash" text NOT NULL,
  "request_hash" text NOT NULL,
  "sanitized_request" jsonb NOT NULL DEFAULT '{}',
  "sanitized_response" jsonb NOT NULL DEFAULT '{}',
  "provider_output_url" text NULL,
  "output_asset_id" uuid NULL,
  "estimated_cost_usd" numeric(18,6) NOT NULL DEFAULT 0,
  "actual_cost_usd" numeric(18,6) NULL,
  "usage_tokens" bigint NULL,
  "provider_seed" bigint NULL,
  "provider_fps" integer NULL,
  "poll_count" integer NOT NULL DEFAULT 0,
  "next_poll_at" timestamptz NULL,
  "timeout_at" timestamptz NOT NULL,
  "error_category" text NULL,
  "error_code" text NULL,
  "error_message" text NULL,
  "cancel_requested_at" timestamptz NULL,
  "cancel_requested_by" uuid NULL,
  "submitted_at" timestamptz NULL,
  "provider_started_at" timestamptz NULL,
  "provider_completed_at" timestamptz NULL,
  "downloaded_at" timestamptz NULL,
  "reviewed_at" timestamptz NULL,
  "reviewed_by" uuid NULL,
  "review_notes" text NOT NULL DEFAULT '',
  "version" bigint NOT NULL DEFAULT 1,
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "scene_generation_tasks_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "scene_generation_tasks_idempotency_key_attempt_number_key" UNIQUE ("idempotency_key", "attempt_number"),
  CONSTRAINT "scene_generation_tasks_provider_provider_task_id_key" UNIQUE ("provider", "provider_task_id"),
  CONSTRAINT "scene_generation_output_state" CHECK ((status <> ALL (ARRAY['VALIDATING'::public.scene_generation_status, 'REVIEW_REQUIRED'::public.scene_generation_status, 'APPROVED'::public.scene_generation_status, 'REJECTED'::public.scene_generation_status])) OR (output_asset_id IS NOT NULL)),
  CONSTRAINT "scene_generation_provider_task_state" CHECK ((status = ANY (ARRAY['DRAFT'::public.scene_generation_status, 'READY'::public.scene_generation_status, 'QUEUED'::public.scene_generation_status, 'SUBMITTING'::public.scene_generation_status, 'FAILED'::public.scene_generation_status, 'CANCELLED'::public.scene_generation_status])) OR (provider_task_id IS NOT NULL)),
  CONSTRAINT "scene_generation_tasks_actual_cost_usd_check" CHECK ((actual_cost_usd IS NULL) OR (actual_cost_usd >= (0)::numeric)),
  CONSTRAINT "scene_generation_tasks_attempt_number_check" CHECK (attempt_number > 0),
  CONSTRAINT "scene_generation_tasks_duration_seconds_check" CHECK ((duration_seconds >= 3) AND (duration_seconds <= 15)),
  CONSTRAINT "scene_generation_tasks_estimated_cost_usd_check" CHECK (estimated_cost_usd >= (0)::numeric),
  CONSTRAINT "scene_generation_tasks_poll_count_check" CHECK (poll_count >= 0),
  CONSTRAINT "scene_generation_tasks_provider_fps_check" CHECK ((provider_fps IS NULL) OR (provider_fps > 0)),
  CONSTRAINT "scene_generation_tasks_sanitized_request_check" CHECK (jsonb_typeof(sanitized_request) = 'object'::text),
  CONSTRAINT "scene_generation_tasks_sanitized_response_check" CHECK (jsonb_typeof(sanitized_response) = 'object'::text),
  CONSTRAINT "scene_generation_tasks_scene_version_check" CHECK (scene_version > 0),
  CONSTRAINT "scene_generation_tasks_usage_tokens_check" CHECK ((usage_tokens IS NULL) OR (usage_tokens >= 0)),
  CONSTRAINT "scene_generation_tasks_version_check" CHECK (version > 0)
);
-- Create index "scene_generation_active_key_idx" to table: "scene_generation_tasks"
CREATE UNIQUE INDEX "scene_generation_active_key_idx" ON "public"."scene_generation_tasks" ("idempotency_key") WHERE (status = ANY (ARRAY['QUEUED'::public.scene_generation_status, 'SUBMITTING'::public.scene_generation_status, 'PROVIDER_QUEUED'::public.scene_generation_status, 'PROVIDER_PROCESSING'::public.scene_generation_status, 'SUCCEEDED'::public.scene_generation_status, 'DOWNLOADING'::public.scene_generation_status, 'VALIDATING'::public.scene_generation_status]));
-- Create index "scene_generation_campaign_idx" to table: "scene_generation_tasks"
CREATE INDEX "scene_generation_campaign_idx" ON "public"."scene_generation_tasks" ("campaign_id", "status", "created_at" DESC);
-- Create index "scene_generation_poll_idx" to table: "scene_generation_tasks"
CREATE INDEX "scene_generation_poll_idx" ON "public"."scene_generation_tasks" ("next_poll_at") WHERE (status = ANY (ARRAY['PROVIDER_QUEUED'::public.scene_generation_status, 'PROVIDER_PROCESSING'::public.scene_generation_status]));
-- Create index "scene_generation_scene_idx" to table: "scene_generation_tasks"
CREATE INDEX "scene_generation_scene_idx" ON "public"."scene_generation_tasks" ("scene_id", "created_at" DESC);
-- Create "scene_quality_checks" table
CREATE TABLE "public"."scene_quality_checks" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "generation_task_id" uuid NOT NULL,
  "status" "public"."quality_check_status" NOT NULL DEFAULT 'QUEUED',
  "deterministic_pass" boolean NULL,
  "transcript_pass" boolean NULL,
  "video_decodes" boolean NULL,
  "duration_pass" boolean NULL,
  "resolution_pass" boolean NULL,
  "audio_stream_present" boolean NULL,
  "silence_warning" boolean NULL,
  "transcript_diff" jsonb NOT NULL DEFAULT '{}',
  "findings" jsonb NOT NULL DEFAULT '[]',
  "character_count_review" integer NULL,
  "duplicate_character_review" boolean NULL,
  "duplicate_product_review" boolean NULL,
  "product_color_mismatch" boolean NULL,
  "blur_or_low_quality_warning" boolean NULL,
  "crop_warning" boolean NULL,
  "subtitle_overflow" boolean NULL,
  "logo_overlap" boolean NULL,
  "cta_safe_zone_violation" boolean NULL,
  "human_notes" text NOT NULL DEFAULT '',
  "reviewed_by" uuid NULL,
  "reviewed_at" timestamptz NULL,
  "completed_at" timestamptz NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "scene_quality_checks_generation_task_id_key" UNIQUE ("generation_task_id"),
  CONSTRAINT "scene_quality_checks_character_count_review_check" CHECK ((character_count_review IS NULL) OR (character_count_review >= 0)),
  CONSTRAINT "scene_quality_checks_findings_check" CHECK (jsonb_typeof(findings) = 'array'::text),
  CONSTRAINT "scene_quality_checks_transcript_diff_check" CHECK (jsonb_typeof(transcript_diff) = 'object'::text),
  CONSTRAINT "scene_quality_checks_version_check" CHECK (version > 0)
);
-- Create "scene_required_facts" table
CREATE TABLE "public"."scene_required_facts" (
  "scene_version_id" uuid NOT NULL,
  "product_fact_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("scene_version_id", "product_fact_id")
);
-- Create "scene_transcriptions" table
CREATE TABLE "public"."scene_transcriptions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "generation_task_id" uuid NOT NULL,
  "status" "public"."transcription_status" NOT NULL DEFAULT 'QUEUED',
  "provider" text NOT NULL,
  "model" text NOT NULL,
  "language" character(2) NULL,
  "transcript" text NOT NULL DEFAULT '',
  "segments" jsonb NOT NULL DEFAULT '[]',
  "transcript_hash" text NULL,
  "provider_request_id" text NULL,
  "input_tokens" bigint NULL,
  "output_tokens" bigint NULL,
  "actual_cost_usd" numeric(18,6) NULL,
  "error_code" text NULL,
  "error_message" text NULL,
  "completed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "scene_transcriptions_generation_task_id_key" UNIQUE ("generation_task_id"),
  CONSTRAINT "scene_transcriptions_actual_cost_usd_check" CHECK ((actual_cost_usd IS NULL) OR (actual_cost_usd >= (0)::numeric)),
  CONSTRAINT "scene_transcriptions_input_tokens_check" CHECK ((input_tokens IS NULL) OR (input_tokens >= 0)),
  CONSTRAINT "scene_transcriptions_language_check" CHECK ((language IS NULL) OR (language = ANY (ARRAY['vi'::bpchar, 'en'::bpchar]))),
  CONSTRAINT "scene_transcriptions_output_tokens_check" CHECK ((output_tokens IS NULL) OR (output_tokens >= 0)),
  CONSTRAINT "scene_transcriptions_segments_check" CHECK (jsonb_typeof(segments) = 'array'::text)
);
-- Create "scene_versions" table
CREATE TABLE "public"."scene_versions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "scene_id" uuid NOT NULL,
  "version" integer NOT NULL,
  "duration_seconds" integer NOT NULL,
  "generation_method" text NOT NULL,
  "speaker_character_id" uuid NULL,
  "listener_character_id" uuid NULL,
  "dialogue" text NOT NULL DEFAULT '',
  "speaker_action" text NOT NULL DEFAULT '',
  "listener_action" text NOT NULL DEFAULT '',
  "camera" text NOT NULL,
  "environment" text NOT NULL,
  "product_placement" text NOT NULL,
  "expected_cost_usd" numeric(18,6) NOT NULL DEFAULT 0,
  "seedance_prompt" text NOT NULL DEFAULT '',
  "scene_hash" text NOT NULL,
  "change_summary" text NOT NULL DEFAULT '',
  "created_by" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "scene_versions_scene_id_version_key" UNIQUE ("scene_id", "version"),
  CONSTRAINT "scene_versions_character_pair" CHECK ((generation_method <> 'seedance'::text) OR ((speaker_character_id IS NOT NULL) AND (listener_character_id IS NOT NULL) AND (speaker_character_id <> listener_character_id))),
  CONSTRAINT "scene_versions_duration_seconds_check" CHECK ((duration_seconds >= 3) AND (duration_seconds <= 15)),
  CONSTRAINT "scene_versions_expected_cost_usd_check" CHECK (expected_cost_usd >= (0)::numeric),
  CONSTRAINT "scene_versions_generation_method_check" CHECK (generation_method = ANY (ARRAY['seedance'::text, 'product_footage'::text, 'still_image'::text])),
  CONSTRAINT "scene_versions_version_check" CHECK (version > 0)
);
-- Create "scenes" table
CREATE TABLE "public"."scenes" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "campaign_id" uuid NOT NULL,
  "script_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "scene_key" text NOT NULL,
  "scene_order" integer NOT NULL,
  "status" "public"."planning_content_status" NOT NULL DEFAULT 'DRAFT',
  "current_version" integer NOT NULL DEFAULT 1,
  "scene_hash" text NOT NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "approved_at" timestamptz NULL,
  "approved_by" uuid NULL,
  "created_by" uuid NULL,
  "updated_by" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "selected_generation_task_id" uuid NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "scenes_campaign_id_scene_key_key" UNIQUE ("campaign_id", "scene_key"),
  CONSTRAINT "scenes_campaign_id_scene_order_key" UNIQUE ("campaign_id", "scene_order"),
  CONSTRAINT "scenes_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "scenes_approval_consistency" CHECK ((status = 'APPROVED'::public.planning_content_status) = (approved_at IS NOT NULL)),
  CONSTRAINT "scenes_current_version_check" CHECK (current_version > 0),
  CONSTRAINT "scenes_key_format" CHECK (scene_key ~ '^scene-[0-9]{2,3}$'::text),
  CONSTRAINT "scenes_scene_order_check" CHECK (scene_order > 0),
  CONSTRAINT "scenes_version_check" CHECK (version > 0)
);
-- Create index "scenes_campaign_order_idx" to table: "scenes"
CREATE INDEX "scenes_campaign_order_idx" ON "public"."scenes" ("campaign_id", "scene_order");
-- Create "script_dialogue_turns" table
CREATE TABLE "public"."script_dialogue_turns" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "script_version_id" uuid NOT NULL,
  "turn_order" integer NOT NULL,
  "character_role" text NOT NULL,
  "dialogue" text NOT NULL,
  "estimated_duration_ms" bigint NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "script_dialogue_turns_script_version_id_turn_order_key" UNIQUE ("script_version_id", "turn_order"),
  CONSTRAINT "script_dialogue_turns_estimated_duration_ms_check" CHECK (estimated_duration_ms > 0),
  CONSTRAINT "script_dialogue_turns_turn_order_check" CHECK (turn_order > 0)
);
-- Create "script_versions" table
CREATE TABLE "public"."script_versions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "script_id" uuid NOT NULL,
  "version" integer NOT NULL,
  "hook" text NOT NULL,
  "introduction" text NOT NULL DEFAULT '',
  "problem" text NOT NULL DEFAULT '',
  "product_solution" text NOT NULL DEFAULT '',
  "product_features" text[] NOT NULL DEFAULT '{}',
  "benefits" text[] NOT NULL DEFAULT '{}',
  "cta" text NOT NULL,
  "closing" text NOT NULL DEFAULT '',
  "approximate_duration_seconds" integer NOT NULL,
  "character_roles" jsonb NOT NULL,
  "spoken_language" character(2) NOT NULL,
  "script_hash" text NOT NULL,
  "prompt_version" text NOT NULL,
  "model" text NOT NULL,
  "change_summary" text NOT NULL DEFAULT '',
  "created_by" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "script_versions_script_id_version_key" UNIQUE ("script_id", "version"),
  CONSTRAINT "script_versions_approximate_duration_seconds_check" CHECK (approximate_duration_seconds = ANY (ARRAY[30, 45])),
  CONSTRAINT "script_versions_character_roles_check" CHECK (jsonb_typeof(character_roles) = 'object'::text),
  CONSTRAINT "script_versions_spoken_language_check" CHECK (spoken_language = ANY (ARRAY['vi'::bpchar, 'en'::bpchar])),
  CONSTRAINT "script_versions_version_check" CHECK (version > 0)
);
-- Create "scripts" table
CREATE TABLE "public"."scripts" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "campaign_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "status" "public"."planning_content_status" NOT NULL DEFAULT 'DRAFT',
  "current_version" integer NOT NULL DEFAULT 1,
  "script_hash" text NOT NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "approved_at" timestamptz NULL,
  "approved_by" uuid NULL,
  "created_by" uuid NULL,
  "updated_by" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "scripts_campaign_id_key" UNIQUE ("campaign_id"),
  CONSTRAINT "scripts_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "scripts_approval_consistency" CHECK ((status = 'APPROVED'::public.planning_content_status) = (approved_at IS NOT NULL)),
  CONSTRAINT "scripts_current_version_check" CHECK (current_version > 0),
  CONSTRAINT "scripts_version_check" CHECK (version > 0)
);
-- Create "sessions" table
CREATE TABLE "public"."sessions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "internal_user_id" uuid NOT NULL,
  "token_hash" bytea NOT NULL,
  "csrf_hash" bytea NOT NULL,
  "ip_address" inet NULL,
  "user_agent" text NOT NULL DEFAULT '',
  "expires_at" timestamptz NOT NULL,
  "last_seen_at" timestamptz NOT NULL DEFAULT now(),
  "revoked_at" timestamptz NULL,
  "revoke_reason" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "sessions_token_hash_key" UNIQUE ("token_hash"),
  CONSTRAINT "sessions_expiry_after_creation" CHECK (expires_at > created_at),
  CONSTRAINT "sessions_user_agent_length" CHECK (length(user_agent) <= 1000)
);
-- Create index "sessions_expiry_idx" to table: "sessions"
CREATE INDEX "sessions_expiry_idx" ON "public"."sessions" ("expires_at") WHERE (revoked_at IS NULL);
-- Create index "sessions_user_active_idx" to table: "sessions"
CREATE INDEX "sessions_user_active_idx" ON "public"."sessions" ("internal_user_id", "expires_at") WHERE (revoked_at IS NULL);
-- Create "social_accounts" table
CREATE TABLE "public"."social_accounts" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "connection_id" uuid NOT NULL,
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "platform" "public"."social_platform" NOT NULL,
  "provider_account_id" text NOT NULL,
  "facebook_page_id" text NULL,
  "instagram_business_id" text NULL,
  "name" text NOT NULL,
  "username" text NULL,
  "picture_url" text NULL,
  "tasks" text[] NOT NULL DEFAULT '{}',
  "token_ciphertext" bytea NOT NULL,
  "token_nonce" bytea NOT NULL,
  "status" "public"."meta_connection_status" NOT NULL DEFAULT 'CONNECTED',
  "last_discovered_at" timestamptz NOT NULL DEFAULT now(),
  "disconnected_at" timestamptz NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "social_accounts_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "social_accounts_workspace_id_platform_provider_account_id_key" UNIQUE ("workspace_id", "platform", "provider_account_id"),
  CONSTRAINT "social_account_provider_shape" CHECK (((platform = 'FACEBOOK'::public.social_platform) AND (facebook_page_id IS NOT NULL)) OR ((platform = 'INSTAGRAM'::public.social_platform) AND (facebook_page_id IS NOT NULL) AND (instagram_business_id IS NOT NULL))),
  CONSTRAINT "social_accounts_version_check" CHECK (version > 0)
);
-- Create index "social_accounts_scope_idx" to table: "social_accounts"
CREATE INDEX "social_accounts_scope_idx" ON "public"."social_accounts" ("client_id", "workspace_id", "platform", "status") WHERE (disconnected_at IS NULL);
-- Create "social_post_metrics_daily" table
CREATE TABLE "public"."social_post_metrics_daily" (
  "social_post_id" uuid NOT NULL,
  "metric_date" date NOT NULL,
  "views" bigint NOT NULL DEFAULT 0,
  "reach" bigint NOT NULL DEFAULT 0,
  "impressions" bigint NOT NULL DEFAULT 0,
  "watch_time_ms" bigint NOT NULL DEFAULT 0,
  "likes" bigint NOT NULL DEFAULT 0,
  "comments" bigint NOT NULL DEFAULT 0,
  "shares" bigint NOT NULL DEFAULT 0,
  "saves" bigint NOT NULL DEFAULT 0,
  "link_clicks" bigint NOT NULL DEFAULT 0,
  "provider_response" jsonb NOT NULL DEFAULT '{}',
  "synced_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("social_post_id", "metric_date"),
  CONSTRAINT "social_post_metrics_daily_provider_response_check" CHECK (jsonb_typeof(provider_response) = 'object'::text)
);
-- Create "social_posts" table
CREATE TABLE "public"."social_posts" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "campaign_id" uuid NOT NULL,
  "social_account_id" uuid NOT NULL,
  "platform" "public"."social_platform" NOT NULL,
  "media_asset_id" uuid NOT NULL,
  "caption" text NOT NULL,
  "scheduled_at" timestamptz NULL,
  "idempotency_key" text NOT NULL,
  "status" "public"."social_post_status" NOT NULL DEFAULT 'DRAFT',
  "content_hash" text NOT NULL,
  "provider_post_id" text NULL,
  "public_url" text NULL,
  "provider_request_id" text NULL,
  "error_category" text NULL,
  "error_code" text NULL,
  "error_message" text NULL,
  "published_at" timestamptz NULL,
  "reviewed_at" timestamptz NULL,
  "reviewed_by" uuid NULL,
  "review_notes" text NOT NULL DEFAULT '',
  "version" bigint NOT NULL DEFAULT 1,
  "created_by" uuid NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "social_posts_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "social_posts_idempotency_key_key" UNIQUE ("idempotency_key"),
  CONSTRAINT "social_post_published_shape" CHECK ((status <> 'PUBLISHED'::public.social_post_status) OR ((provider_post_id IS NOT NULL) AND (published_at IS NOT NULL))),
  CONSTRAINT "social_post_schedule_future" CHECK ((scheduled_at IS NULL) OR (scheduled_at > created_at)),
  CONSTRAINT "social_posts_version_check" CHECK (version > 0)
);
-- Create index "social_posts_campaign_idx" to table: "social_posts"
CREATE INDEX "social_posts_campaign_idx" ON "public"."social_posts" ("campaign_id", "created_at" DESC);
-- Create index "social_posts_schedule_idx" to table: "social_posts"
CREATE INDEX "social_posts_schedule_idx" ON "public"."social_posts" ("scheduled_at") WHERE (status = 'SCHEDULED'::public.social_post_status);
-- Create "subtitle_outputs" table
CREATE TABLE "public"."subtitle_outputs" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "render_job_id" uuid NOT NULL,
  "format" "public"."subtitle_format" NOT NULL,
  "language" character(2) NOT NULL,
  "storage_key" text NOT NULL,
  "checksum_sha256" text NOT NULL,
  "cue_count" integer NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "subtitle_outputs_render_job_id_format_language_key" UNIQUE ("render_job_id", "format", "language"),
  CONSTRAINT "subtitle_outputs_storage_key_key" UNIQUE ("storage_key"),
  CONSTRAINT "subtitle_outputs_cue_count_check" CHECK (cue_count >= 0),
  CONSTRAINT "subtitle_outputs_language_check" CHECK (language = ANY (ARRAY['vi'::bpchar, 'en'::bpchar]))
);
-- Create "usage_ledger" table
CREATE TABLE "public"."usage_ledger" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "provider" text NOT NULL,
  "model" text NOT NULL DEFAULT '',
  "request_reference" text NOT NULL,
  "operation" text NOT NULL,
  "client_id" uuid NULL,
  "workspace_id" uuid NULL,
  "campaign_id" uuid NULL,
  "scene_id" uuid NULL,
  "video_project_id" uuid NULL,
  "input_units" bigint NOT NULL DEFAULT 0,
  "output_units" bigint NOT NULL DEFAULT 0,
  "generated_seconds" numeric(12,3) NOT NULL DEFAULT 0,
  "accepted_seconds" numeric(12,3) NOT NULL DEFAULT 0,
  "provider_reported_cost" numeric(18,6) NULL,
  "estimated_cost" numeric(18,6) NOT NULL DEFAULT 0,
  "currency" character(3) NOT NULL DEFAULT 'USD',
  "exchange_rate_snapshot_id" uuid NULL,
  "outcome" "public"."usage_outcome" NOT NULL,
  "reused" boolean NOT NULL DEFAULT false,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "occurred_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "usage_ledger_provider_request_reference_operation_key" UNIQUE ("provider", "request_reference", "operation"),
  CONSTRAINT "usage_ledger_check" CHECK ((accepted_seconds >= (0)::numeric) AND (accepted_seconds <= generated_seconds)),
  CONSTRAINT "usage_ledger_estimated_cost_check" CHECK (estimated_cost >= (0)::numeric),
  CONSTRAINT "usage_ledger_generated_seconds_check" CHECK (generated_seconds >= (0)::numeric),
  CONSTRAINT "usage_ledger_input_units_check" CHECK (input_units >= 0),
  CONSTRAINT "usage_ledger_metadata_check" CHECK (jsonb_typeof(metadata) = 'object'::text),
  CONSTRAINT "usage_ledger_output_units_check" CHECK (output_units >= 0),
  CONSTRAINT "usage_ledger_provider_reported_cost_check" CHECK ((provider_reported_cost IS NULL) OR (provider_reported_cost >= (0)::numeric))
);
-- Create index "usage_ledger_campaign_time_idx" to table: "usage_ledger"
CREATE INDEX "usage_ledger_campaign_time_idx" ON "public"."usage_ledger" ("campaign_id", "occurred_at" DESC);
-- Create index "usage_ledger_provider_time_idx" to table: "usage_ledger"
CREATE INDEX "usage_ledger_provider_time_idx" ON "public"."usage_ledger" ("provider", "occurred_at" DESC);
-- Create index "usage_ledger_scope_time_idx" to table: "usage_ledger"
CREATE INDEX "usage_ledger_scope_time_idx" ON "public"."usage_ledger" ("client_id", "workspace_id", "occurred_at" DESC);
-- Create "video_outputs" table
CREATE TABLE "public"."video_outputs" (
  "render_job_id" uuid NOT NULL,
  "media_asset_id" uuid NOT NULL,
  "width" integer NOT NULL,
  "height" integer NOT NULL,
  "fps" integer NOT NULL,
  "duration_ms" bigint NOT NULL,
  "codec" text NOT NULL,
  "audio_codec" text NULL,
  "file_size_bytes" bigint NOT NULL,
  "checksum_sha256" text NOT NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("render_job_id"),
  CONSTRAINT "video_outputs_codec_check" CHECK (codec = ANY (ARRAY['h264'::text, 'avc1'::text])),
  CONSTRAINT "video_outputs_duration_ms_check" CHECK (duration_ms > 0),
  CONSTRAINT "video_outputs_file_size_bytes_check" CHECK (file_size_bytes > 0),
  CONSTRAINT "video_outputs_fps_check" CHECK (fps = 30),
  CONSTRAINT "video_outputs_height_check" CHECK (height = 1920),
  CONSTRAINT "video_outputs_metadata_check" CHECK (jsonb_typeof(metadata) = 'object'::text),
  CONSTRAINT "video_outputs_width_check" CHECK (width = 1080)
);
-- Create "video_project_versions" table
CREATE TABLE "public"."video_project_versions" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "video_project_id" uuid NOT NULL,
  "version" integer NOT NULL,
  "headline" text NOT NULL,
  "lower_third" text NOT NULL DEFAULT '',
  "show_price" boolean NOT NULL DEFAULT true,
  "show_discount_code" boolean NOT NULL DEFAULT true,
  "show_cta" boolean NOT NULL DEFAULT true,
  "show_website" boolean NOT NULL DEFAULT true,
  "show_phone" boolean NOT NULL DEFAULT true,
  "show_qr_code" boolean NOT NULL DEFAULT true,
  "show_disclaimer" boolean NOT NULL DEFAULT true,
  "burn_captions" boolean NOT NULL DEFAULT true,
  "project_hash" text NOT NULL,
  "change_summary" text NOT NULL DEFAULT '',
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "video_project_versions_video_project_id_version_key" UNIQUE ("video_project_id", "version"),
  CONSTRAINT "video_project_versions_version_check" CHECK (version > 0)
);
-- Create "video_projects" table
CREATE TABLE "public"."video_projects" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "workspace_id" uuid NOT NULL,
  "campaign_id" uuid NOT NULL,
  "current_version" integer NOT NULL DEFAULT 1,
  "selected_render_job_id" uuid NULL,
  "music_asset_id" uuid NULL,
  "music_gain_db" numeric(6,2) NOT NULL DEFAULT -18,
  "dialogue_ducking_db" numeric(6,2) NOT NULL DEFAULT -9,
  "version" bigint NOT NULL DEFAULT 1,
  "created_by" uuid NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "video_projects_campaign_id_key" UNIQUE ("campaign_id"),
  CONSTRAINT "video_projects_id_client_id_workspace_id_key" UNIQUE ("id", "client_id", "workspace_id"),
  CONSTRAINT "video_projects_current_version_check" CHECK (current_version > 0),
  CONSTRAINT "video_projects_dialogue_ducking_db_check" CHECK ((dialogue_ducking_db >= ('-30'::integer)::numeric) AND (dialogue_ducking_db <= (0)::numeric)),
  CONSTRAINT "video_projects_music_gain_db_check" CHECK ((music_gain_db >= ('-60'::integer)::numeric) AND (music_gain_db <= (0)::numeric)),
  CONSTRAINT "video_projects_version_check" CHECK (version > 0)
);
-- Create "webhook_events" table
CREATE TABLE "public"."webhook_events" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "provider" text NOT NULL,
  "provider_event_id" text NOT NULL,
  "event_type" text NOT NULL,
  "signature_valid" boolean NOT NULL,
  "payload_hash" bytea NOT NULL,
  "sanitized_payload" jsonb NULL,
  "received_at" timestamptz NOT NULL DEFAULT now(),
  "processed_at" timestamptz NULL,
  "processing_error" text NULL,
  "attempt_count" integer NOT NULL DEFAULT 0,
  PRIMARY KEY ("id"),
  CONSTRAINT "webhook_events_provider_provider_event_id_key" UNIQUE ("provider", "provider_event_id"),
  CONSTRAINT "webhook_events_attempt_count_check" CHECK (attempt_count >= 0)
);
-- Create index "webhook_events_unprocessed_idx" to table: "webhook_events"
CREATE INDEX "webhook_events_unprocessed_idx" ON "public"."webhook_events" ("received_at") WHERE (processed_at IS NULL);
-- Create "workspaces" table
CREATE TABLE "public"."workspaces" (
  "id" uuid NOT NULL DEFAULT uuidv7(),
  "client_id" uuid NOT NULL,
  "name" text NOT NULL,
  "slug" text NOT NULL,
  "timezone" text NOT NULL DEFAULT 'Asia/Ho_Chi_Minh',
  "status" "public"."lifecycle_status" NOT NULL DEFAULT 'ACTIVE',
  "future_tenant_owner_id" text NULL,
  "version" bigint NOT NULL DEFAULT 1,
  "archived_at" timestamptz NULL,
  "created_by" uuid NOT NULL,
  "updated_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "workspaces_client_id_slug_key" UNIQUE ("client_id", "slug"),
  CONSTRAINT "workspaces_id_client_id_key" UNIQUE ("id", "client_id"),
  CONSTRAINT "workspaces_archive_consistency" CHECK ((status = 'ARCHIVED'::public.lifecycle_status) = (archived_at IS NOT NULL)),
  CONSTRAINT "workspaces_name_length" CHECK ((length(name) >= 2) AND (length(name) <= 160)),
  CONSTRAINT "workspaces_slug_format" CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'::text),
  CONSTRAINT "workspaces_version_check" CHECK (version > 0)
);
-- Create index "workspaces_client_status_idx" to table: "workspaces"
CREATE INDEX "workspaces_client_status_idx" ON "public"."workspaces" ("client_id", "status", "name");
-- Modify "ad_campaign_metrics_daily" table
ALTER TABLE "public"."ad_campaign_metrics_daily" ADD CONSTRAINT "ad_campaign_metrics_daily_ad_campaign_id_fkey" FOREIGN KEY ("ad_campaign_id") REFERENCES "public"."ad_campaigns" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "ad_campaigns" table
ALTER TABLE "public"."ad_campaigns" ADD CONSTRAINT "ad_campaigns_campaign_id_fkey" FOREIGN KEY ("campaign_id") REFERENCES "public"."campaigns" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "ad_campaigns_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "ad_campaigns_meta_ad_account_id_client_id_workspace_id_fkey" FOREIGN KEY ("meta_ad_account_id", "client_id", "workspace_id") REFERENCES "public"."meta_ad_accounts" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "ad_campaigns_meta_pixel_id_fkey" FOREIGN KEY ("meta_pixel_id") REFERENCES "public"."meta_pixels" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "ad_campaigns_social_account_id_client_id_workspace_id_fkey" FOREIGN KEY ("social_account_id", "client_id", "workspace_id") REFERENCES "public"."social_accounts" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "ad_campaigns_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "ad_campaigns_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "ad_creatives" table
ALTER TABLE "public"."ad_creatives" ADD CONSTRAINT "ad_creatives_ad_campaign_id_fkey" FOREIGN KEY ("ad_campaign_id") REFERENCES "public"."ad_campaigns" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "ad_creatives_media_asset_id_fkey" FOREIGN KEY ("media_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "ad_creatives_thumbnail_asset_id_fkey" FOREIGN KEY ("thumbnail_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "ad_metrics_daily" table
ALTER TABLE "public"."ad_metrics_daily" ADD CONSTRAINT "ad_metrics_daily_ad_id_fkey" FOREIGN KEY ("ad_id") REFERENCES "public"."ads" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "ad_recommendations" table
ALTER TABLE "public"."ad_recommendations" ADD CONSTRAINT "ad_recommendations_ad_campaign_id_fkey" FOREIGN KEY ("ad_campaign_id") REFERENCES "public"."ad_campaigns" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "ad_recommendations_campaign_id_fkey" FOREIGN KEY ("campaign_id") REFERENCES "public"."campaigns" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "ad_recommendations_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "ad_recommendations_reviewer_id_fkey" FOREIGN KEY ("reviewer_id") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "ad_recommendations_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "ad_set_metrics_daily" table
ALTER TABLE "public"."ad_set_metrics_daily" ADD CONSTRAINT "ad_set_metrics_daily_ad_set_id_fkey" FOREIGN KEY ("ad_set_id") REFERENCES "public"."ad_sets" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "ad_sets" table
ALTER TABLE "public"."ad_sets" ADD CONSTRAINT "ad_sets_ad_campaign_id_fkey" FOREIGN KEY ("ad_campaign_id") REFERENCES "public"."ad_campaigns" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "ads" table
ALTER TABLE "public"."ads" ADD CONSTRAINT "ads_ad_campaign_id_fkey" FOREIGN KEY ("ad_campaign_id") REFERENCES "public"."ad_campaigns" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "ads_ad_creative_id_fkey" FOREIGN KEY ("ad_creative_id") REFERENCES "public"."ad_creatives" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "ads_ad_set_id_fkey" FOREIGN KEY ("ad_set_id") REFERENCES "public"."ad_sets" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "approval_events" table
ALTER TABLE "public"."approval_events" ADD CONSTRAINT "approval_events_actor_id_fkey" FOREIGN KEY ("actor_id") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "approval_events_approval_id_fkey" FOREIGN KEY ("approval_id") REFERENCES "public"."approvals" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "approvals" table
ALTER TABLE "public"."approvals" ADD CONSTRAINT "approvals_campaign_id_fkey" FOREIGN KEY ("campaign_id") REFERENCES "public"."campaigns" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "approvals_decided_by_fkey" FOREIGN KEY ("decided_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "approvals_requested_by_fkey" FOREIGN KEY ("requested_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "approvals_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "audit_logs" table
ALTER TABLE "public"."audit_logs" ADD CONSTRAINT "audit_logs_actor_internal_user_id_fkey" FOREIGN KEY ("actor_internal_user_id") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Modify "brand_versions" table
ALTER TABLE "public"."brand_versions" ADD CONSTRAINT "brand_versions_brand_id_client_id_workspace_id_fkey" FOREIGN KEY ("brand_id", "client_id", "workspace_id") REFERENCES "public"."brands" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "brand_versions_brand_id_fkey" FOREIGN KEY ("brand_id") REFERENCES "public"."brands" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "brand_versions_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "brands" table
ALTER TABLE "public"."brands" ADD CONSTRAINT "brands_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "brands_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "brands_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "campaign_characters" table
ALTER TABLE "public"."campaign_characters" ADD CONSTRAINT "campaign_characters_campaign_id_fkey" FOREIGN KEY ("campaign_id") REFERENCES "public"."campaigns" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "campaign_characters_character_id_fkey" FOREIGN KEY ("character_id") REFERENCES "public"."characters" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "campaign_characters_selected_by_fkey" FOREIGN KEY ("selected_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "campaign_concept_versions" table
ALTER TABLE "public"."campaign_concept_versions" ADD CONSTRAINT "campaign_concept_versions_concept_id_fkey" FOREIGN KEY ("concept_id") REFERENCES "public"."campaign_concepts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "campaign_concept_versions_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "campaign_concepts" table
ALTER TABLE "public"."campaign_concepts" ADD CONSTRAINT "campaign_concepts_campaign_id_client_id_workspace_id_fkey" FOREIGN KEY ("campaign_id", "client_id", "workspace_id") REFERENCES "public"."campaigns" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "campaign_concepts_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "campaign_concepts_locked_by_fkey" FOREIGN KEY ("locked_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "campaign_concepts_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "campaign_content_variant_versions" table
ALTER TABLE "public"."campaign_content_variant_versions" ADD CONSTRAINT "campaign_content_variant_versions_content_variant_id_fkey" FOREIGN KEY ("content_variant_id") REFERENCES "public"."campaign_content_variants" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "campaign_content_variant_versions_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "campaign_content_variants" table
ALTER TABLE "public"."campaign_content_variants" ADD CONSTRAINT "campaign_content_variants_approved_by_fkey" FOREIGN KEY ("approved_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "campaign_content_variants_campaign_id_client_id_workspace__fkey" FOREIGN KEY ("campaign_id", "client_id", "workspace_id") REFERENCES "public"."campaigns" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "campaign_content_variants_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "campaign_content_variants_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "campaign_versions" table
ALTER TABLE "public"."campaign_versions" ADD CONSTRAINT "campaign_versions_campaign_id_client_id_workspace_id_fkey" FOREIGN KEY ("campaign_id", "client_id", "workspace_id") REFERENCES "public"."campaigns" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "campaign_versions_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "campaigns" table
ALTER TABLE "public"."campaigns" ADD CONSTRAINT "campaigns_brand_id_client_id_workspace_id_fkey" FOREIGN KEY ("brand_id", "client_id", "workspace_id") REFERENCES "public"."brands" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "campaigns_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "campaigns_product_id_client_id_workspace_id_fkey" FOREIGN KEY ("product_id", "client_id", "workspace_id") REFERENCES "public"."products" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "campaigns_selected_concept_fk" FOREIGN KEY ("selected_concept_id", "client_id", "workspace_id") REFERENCES "public"."campaign_concepts" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "campaigns_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "campaigns_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "character_assets" table
ALTER TABLE "public"."character_assets" ADD CONSTRAINT "character_assets_character_id_fkey" FOREIGN KEY ("character_id") REFERENCES "public"."characters" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "character_assets_media_asset_id_fkey" FOREIGN KEY ("media_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "character_consents" table
ALTER TABLE "public"."character_consents" ADD CONSTRAINT "character_consents_artifact_asset_id_fkey" FOREIGN KEY ("artifact_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "character_consents_character_id_fkey" FOREIGN KEY ("character_id") REFERENCES "public"."characters" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "character_consents_recorded_by_fkey" FOREIGN KEY ("recorded_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "characters" table
ALTER TABLE "public"."characters" ADD CONSTRAINT "characters_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "characters_preview_asset_id_fkey" FOREIGN KEY ("preview_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "characters_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "characters_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "clients" table
ALTER TABLE "public"."clients" ADD CONSTRAINT "clients_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "clients_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "cost_estimates" table
ALTER TABLE "public"."cost_estimates" ADD CONSTRAINT "cost_estimates_campaign_id_fkey" FOREIGN KEY ("campaign_id") REFERENCES "public"."campaigns" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "cost_estimates_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "cost_records" table
ALTER TABLE "public"."cost_records" ADD CONSTRAINT "cost_records_campaign_id_fkey" FOREIGN KEY ("campaign_id") REFERENCES "public"."campaigns" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "cost_records_client_id_fkey" FOREIGN KEY ("client_id") REFERENCES "public"."clients" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "cost_records_usage_ledger_id_fkey" FOREIGN KEY ("usage_ledger_id") REFERENCES "public"."usage_ledger" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "cost_records_workspace_id_fkey" FOREIGN KEY ("workspace_id") REFERENCES "public"."workspaces" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "feature_flags" table
ALTER TABLE "public"."feature_flags" ADD CONSTRAINT "feature_flags_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Modify "generation_jobs" table
ALTER TABLE "public"."generation_jobs" ADD CONSTRAINT "generation_jobs_campaign_id_fkey" FOREIGN KEY ("campaign_id") REFERENCES "public"."campaigns" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "generation_jobs_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "generation_jobs_provider_request_id_fkey" FOREIGN KEY ("provider_request_id") REFERENCES "public"."provider_requests" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "generation_jobs_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "media_asset_tags" table
ALTER TABLE "public"."media_asset_tags" ADD CONSTRAINT "media_asset_tags_media_asset_id_fkey" FOREIGN KEY ("media_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "media_asset_versions" table
ALTER TABLE "public"."media_asset_versions" ADD CONSTRAINT "media_asset_versions_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "media_asset_versions_media_asset_id_client_id_workspace_id_fkey" FOREIGN KEY ("media_asset_id", "client_id", "workspace_id") REFERENCES "public"."media_assets" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "media_assets" table
ALTER TABLE "public"."media_assets" ADD CONSTRAINT "media_assets_brand_id_fkey" FOREIGN KEY ("brand_id") REFERENCES "public"."brands" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "media_assets_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "media_assets_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "public"."products" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "media_assets_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "media_assets_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "media_uploads" table
ALTER TABLE "public"."media_uploads" ADD CONSTRAINT "media_uploads_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "media_uploads_media_asset_id_client_id_workspace_id_fkey" FOREIGN KEY ("media_asset_id", "client_id", "workspace_id") REFERENCES "public"."media_assets" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "meta_ad_accounts" table
ALTER TABLE "public"."meta_ad_accounts" ADD CONSTRAINT "meta_ad_accounts_connection_id_fkey" FOREIGN KEY ("connection_id") REFERENCES "public"."meta_connections" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "meta_ad_accounts_meta_business_id_fkey" FOREIGN KEY ("meta_business_id") REFERENCES "public"."meta_businesses" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "meta_ad_accounts_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "meta_ad_actions" table
ALTER TABLE "public"."meta_ad_actions" ADD CONSTRAINT "meta_ad_actions_ad_campaign_id_fkey" FOREIGN KEY ("ad_campaign_id") REFERENCES "public"."ad_campaigns" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "meta_ad_actions_requested_by_fkey" FOREIGN KEY ("requested_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "meta_ad_actions_reviewed_by_fkey" FOREIGN KEY ("reviewed_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "meta_ad_guardrails" table
ALTER TABLE "public"."meta_ad_guardrails" ADD CONSTRAINT "meta_ad_guardrails_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "meta_ad_guardrails_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "meta_ad_guardrails_workspace_id_fkey" FOREIGN KEY ("workspace_id") REFERENCES "public"."workspaces" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "meta_audiences" table
ALTER TABLE "public"."meta_audiences" ADD CONSTRAINT "meta_audiences_meta_ad_account_id_fkey" FOREIGN KEY ("meta_ad_account_id") REFERENCES "public"."meta_ad_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "meta_businesses" table
ALTER TABLE "public"."meta_businesses" ADD CONSTRAINT "meta_businesses_connection_id_fkey" FOREIGN KEY ("connection_id") REFERENCES "public"."meta_connections" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "meta_businesses_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "meta_connections" table
ALTER TABLE "public"."meta_connections" ADD CONSTRAINT "meta_connections_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "meta_connections_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "meta_connections_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "meta_oauth_states" table
ALTER TABLE "public"."meta_oauth_states" ADD CONSTRAINT "meta_oauth_states_actor_id_fkey" FOREIGN KEY ("actor_id") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "meta_oauth_states_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "meta_pixels" table
ALTER TABLE "public"."meta_pixels" ADD CONSTRAINT "meta_pixels_meta_ad_account_id_fkey" FOREIGN KEY ("meta_ad_account_id") REFERENCES "public"."meta_ad_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "notifications" table
ALTER TABLE "public"."notifications" ADD CONSTRAINT "notifications_client_id_fkey" FOREIGN KEY ("client_id") REFERENCES "public"."clients" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "notifications_internal_user_id_fkey" FOREIGN KEY ("internal_user_id") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "notifications_workspace_id_fkey" FOREIGN KEY ("workspace_id") REFERENCES "public"."workspaces" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "product_claim_sources" table
ALTER TABLE "public"."product_claim_sources" ADD CONSTRAINT "product_claim_sources_claim_id_fkey" FOREIGN KEY ("claim_id") REFERENCES "public"."product_claims" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "product_claim_sources_fact_id_fkey" FOREIGN KEY ("fact_id") REFERENCES "public"."product_facts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "product_claim_sources_media_fk" FOREIGN KEY ("media_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "product_claim_versions" table
ALTER TABLE "public"."product_claim_versions" ADD CONSTRAINT "product_claim_versions_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "product_claim_versions_product_claim_id_fkey" FOREIGN KEY ("product_claim_id") REFERENCES "public"."product_claims" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "product_claims" table
ALTER TABLE "public"."product_claims" ADD CONSTRAINT "product_claims_approved_by_fkey" FOREIGN KEY ("approved_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "product_claims_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "product_claims_product_id_client_id_workspace_id_fkey" FOREIGN KEY ("product_id", "client_id", "workspace_id") REFERENCES "public"."products" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "product_claims_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "product_fact_versions" table
ALTER TABLE "public"."product_fact_versions" ADD CONSTRAINT "product_fact_versions_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "product_fact_versions_product_fact_id_fkey" FOREIGN KEY ("product_fact_id") REFERENCES "public"."product_facts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "product_fact_versions_source_asset_fk" FOREIGN KEY ("source_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "product_facts" table
ALTER TABLE "public"."product_facts" ADD CONSTRAINT "product_facts_approved_by_fkey" FOREIGN KEY ("approved_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "product_facts_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "product_facts_product_id_client_id_workspace_id_fkey" FOREIGN KEY ("product_id", "client_id", "workspace_id") REFERENCES "public"."products" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "product_facts_source_asset_fk" FOREIGN KEY ("source_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "product_facts_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "product_versions" table
ALTER TABLE "public"."product_versions" ADD CONSTRAINT "product_versions_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "product_versions_product_id_client_id_workspace_id_fkey" FOREIGN KEY ("product_id", "client_id", "workspace_id") REFERENCES "public"."products" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "product_vertical_data" table
ALTER TABLE "public"."product_vertical_data" ADD CONSTRAINT "product_vertical_data_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "product_vertical_data_product_id_client_id_workspace_id_fkey" FOREIGN KEY ("product_id", "client_id", "workspace_id") REFERENCES "public"."products" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "products" table
ALTER TABLE "public"."products" ADD CONSTRAINT "products_brand_id_fkey" FOREIGN KEY ("brand_id") REFERENCES "public"."brands" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "products_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "products_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "products_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "provider_outputs" table
ALTER TABLE "public"."provider_outputs" ADD CONSTRAINT "provider_outputs_provider_request_id_fkey" FOREIGN KEY ("provider_request_id") REFERENCES "public"."provider_requests" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "provider_requests" table
ALTER TABLE "public"."provider_requests" ADD CONSTRAINT "provider_requests_campaign_id_fkey" FOREIGN KEY ("campaign_id") REFERENCES "public"."campaigns" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "provider_requests_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "provider_requests_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "publish_jobs" table
ALTER TABLE "public"."publish_jobs" ADD CONSTRAINT "publish_jobs_social_post_id_fkey" FOREIGN KEY ("social_post_id") REFERENCES "public"."social_posts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "render_jobs" table
ALTER TABLE "public"."render_jobs" ADD CONSTRAINT "render_jobs_campaign_id_fkey" FOREIGN KEY ("campaign_id") REFERENCES "public"."campaigns" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "render_jobs_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "render_jobs_output_asset_id_fkey" FOREIGN KEY ("output_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "render_jobs_render_manifest_id_fkey" FOREIGN KEY ("render_manifest_id") REFERENCES "public"."render_manifests" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "render_jobs_reviewed_by_fkey" FOREIGN KEY ("reviewed_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "render_jobs_video_project_id_fkey" FOREIGN KEY ("video_project_id") REFERENCES "public"."video_projects" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "render_jobs_video_project_id_video_project_version_fkey" FOREIGN KEY ("video_project_id", "video_project_version") REFERENCES "public"."video_project_versions" ("video_project_id", "version") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "render_jobs_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "render_manifests" table
ALTER TABLE "public"."render_manifests" ADD CONSTRAINT "render_manifests_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "render_manifests_video_project_id_fkey" FOREIGN KEY ("video_project_id") REFERENCES "public"."video_projects" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "render_manifests_video_project_id_video_project_version_fkey" FOREIGN KEY ("video_project_id", "video_project_version") REFERENCES "public"."video_project_versions" ("video_project_id", "version") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "scene_assets" table
ALTER TABLE "public"."scene_assets" ADD CONSTRAINT "scene_assets_media_asset_id_fkey" FOREIGN KEY ("media_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scene_assets_scene_version_id_fkey" FOREIGN KEY ("scene_version_id") REFERENCES "public"."scene_versions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "scene_generation_edits" table
ALTER TABLE "public"."scene_generation_edits" ADD CONSTRAINT "scene_generation_edits_generation_task_id_fkey" FOREIGN KEY ("generation_task_id") REFERENCES "public"."scene_generation_tasks" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "scene_generation_edits_replacement_asset_id_fkey" FOREIGN KEY ("replacement_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scene_generation_edits_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "scene_generation_events" table
ALTER TABLE "public"."scene_generation_events" ADD CONSTRAINT "scene_generation_events_actor_id_fkey" FOREIGN KEY ("actor_id") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scene_generation_events_generation_task_id_fkey" FOREIGN KEY ("generation_task_id") REFERENCES "public"."scene_generation_tasks" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "scene_generation_tasks" table
ALTER TABLE "public"."scene_generation_tasks" ADD CONSTRAINT "scene_generation_tasks_campaign_id_fkey" FOREIGN KEY ("campaign_id") REFERENCES "public"."campaigns" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scene_generation_tasks_cancel_requested_by_fkey" FOREIGN KEY ("cancel_requested_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scene_generation_tasks_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scene_generation_tasks_output_asset_id_fkey" FOREIGN KEY ("output_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scene_generation_tasks_reviewed_by_fkey" FOREIGN KEY ("reviewed_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scene_generation_tasks_scene_id_client_id_workspace_id_fkey" FOREIGN KEY ("scene_id", "client_id", "workspace_id") REFERENCES "public"."scenes" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scene_generation_tasks_scene_id_scene_version_fkey" FOREIGN KEY ("scene_id", "scene_version") REFERENCES "public"."scene_versions" ("scene_id", "version") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scene_generation_tasks_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "scene_quality_checks" table
ALTER TABLE "public"."scene_quality_checks" ADD CONSTRAINT "scene_quality_checks_generation_task_id_fkey" FOREIGN KEY ("generation_task_id") REFERENCES "public"."scene_generation_tasks" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "scene_quality_checks_reviewed_by_fkey" FOREIGN KEY ("reviewed_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "scene_required_facts" table
ALTER TABLE "public"."scene_required_facts" ADD CONSTRAINT "scene_required_facts_product_fact_id_fkey" FOREIGN KEY ("product_fact_id") REFERENCES "public"."product_facts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scene_required_facts_scene_version_id_fkey" FOREIGN KEY ("scene_version_id") REFERENCES "public"."scene_versions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "scene_transcriptions" table
ALTER TABLE "public"."scene_transcriptions" ADD CONSTRAINT "scene_transcriptions_generation_task_id_fkey" FOREIGN KEY ("generation_task_id") REFERENCES "public"."scene_generation_tasks" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "scene_versions" table
ALTER TABLE "public"."scene_versions" ADD CONSTRAINT "scene_versions_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scene_versions_listener_character_id_fkey" FOREIGN KEY ("listener_character_id") REFERENCES "public"."characters" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scene_versions_scene_id_fkey" FOREIGN KEY ("scene_id") REFERENCES "public"."scenes" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "scene_versions_speaker_character_id_fkey" FOREIGN KEY ("speaker_character_id") REFERENCES "public"."characters" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "scenes" table
ALTER TABLE "public"."scenes" ADD CONSTRAINT "scenes_approved_by_fkey" FOREIGN KEY ("approved_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scenes_campaign_id_client_id_workspace_id_fkey" FOREIGN KEY ("campaign_id", "client_id", "workspace_id") REFERENCES "public"."campaigns" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scenes_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scenes_script_id_client_id_workspace_id_fkey" FOREIGN KEY ("script_id", "client_id", "workspace_id") REFERENCES "public"."scripts" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scenes_selected_generation_task_id_fkey" FOREIGN KEY ("selected_generation_task_id") REFERENCES "public"."scene_generation_tasks" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scenes_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "script_dialogue_turns" table
ALTER TABLE "public"."script_dialogue_turns" ADD CONSTRAINT "script_dialogue_turns_script_version_id_fkey" FOREIGN KEY ("script_version_id") REFERENCES "public"."script_versions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "script_versions" table
ALTER TABLE "public"."script_versions" ADD CONSTRAINT "script_versions_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "script_versions_script_id_fkey" FOREIGN KEY ("script_id") REFERENCES "public"."scripts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "scripts" table
ALTER TABLE "public"."scripts" ADD CONSTRAINT "scripts_approved_by_fkey" FOREIGN KEY ("approved_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scripts_campaign_id_client_id_workspace_id_fkey" FOREIGN KEY ("campaign_id", "client_id", "workspace_id") REFERENCES "public"."campaigns" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scripts_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scripts_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "sessions" table
ALTER TABLE "public"."sessions" ADD CONSTRAINT "sessions_internal_user_id_fkey" FOREIGN KEY ("internal_user_id") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "social_accounts" table
ALTER TABLE "public"."social_accounts" ADD CONSTRAINT "social_accounts_connection_id_fkey" FOREIGN KEY ("connection_id") REFERENCES "public"."meta_connections" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "social_accounts_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "social_post_metrics_daily" table
ALTER TABLE "public"."social_post_metrics_daily" ADD CONSTRAINT "social_post_metrics_daily_social_post_id_fkey" FOREIGN KEY ("social_post_id") REFERENCES "public"."social_posts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "social_posts" table
ALTER TABLE "public"."social_posts" ADD CONSTRAINT "social_posts_campaign_id_fkey" FOREIGN KEY ("campaign_id") REFERENCES "public"."campaigns" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "social_posts_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "social_posts_media_asset_id_fkey" FOREIGN KEY ("media_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "social_posts_reviewed_by_fkey" FOREIGN KEY ("reviewed_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "social_posts_social_account_id_client_id_workspace_id_fkey" FOREIGN KEY ("social_account_id", "client_id", "workspace_id") REFERENCES "public"."social_accounts" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "social_posts_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "social_posts_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "subtitle_outputs" table
ALTER TABLE "public"."subtitle_outputs" ADD CONSTRAINT "subtitle_outputs_render_job_id_fkey" FOREIGN KEY ("render_job_id") REFERENCES "public"."render_jobs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "usage_ledger" table
ALTER TABLE "public"."usage_ledger" ADD CONSTRAINT "usage_ledger_campaign_id_fkey" FOREIGN KEY ("campaign_id") REFERENCES "public"."campaigns" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "usage_ledger_client_id_fkey" FOREIGN KEY ("client_id") REFERENCES "public"."clients" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "usage_ledger_exchange_rate_snapshot_id_fkey" FOREIGN KEY ("exchange_rate_snapshot_id") REFERENCES "public"."exchange_rate_snapshots" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "usage_ledger_scene_id_fkey" FOREIGN KEY ("scene_id") REFERENCES "public"."scenes" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "usage_ledger_video_project_id_fkey" FOREIGN KEY ("video_project_id") REFERENCES "public"."video_projects" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "usage_ledger_workspace_id_fkey" FOREIGN KEY ("workspace_id") REFERENCES "public"."workspaces" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "video_outputs" table
ALTER TABLE "public"."video_outputs" ADD CONSTRAINT "video_outputs_media_asset_id_fkey" FOREIGN KEY ("media_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "video_outputs_render_job_id_fkey" FOREIGN KEY ("render_job_id") REFERENCES "public"."render_jobs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "video_project_versions" table
ALTER TABLE "public"."video_project_versions" ADD CONSTRAINT "video_project_versions_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "video_project_versions_video_project_id_fkey" FOREIGN KEY ("video_project_id") REFERENCES "public"."video_projects" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "video_projects" table
ALTER TABLE "public"."video_projects" ADD CONSTRAINT "video_projects_campaign_id_client_id_workspace_id_fkey" FOREIGN KEY ("campaign_id", "client_id", "workspace_id") REFERENCES "public"."campaigns" ("id", "client_id", "workspace_id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "video_projects_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "video_projects_music_asset_id_fkey" FOREIGN KEY ("music_asset_id") REFERENCES "public"."media_assets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "video_projects_selected_render_fk" FOREIGN KEY ("selected_render_job_id") REFERENCES "public"."render_jobs" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "video_projects_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "video_projects_workspace_id_client_id_fkey" FOREIGN KEY ("workspace_id", "client_id") REFERENCES "public"."workspaces" ("id", "client_id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "workspaces" table
ALTER TABLE "public"."workspaces" ADD CONSTRAINT "workspaces_client_id_fkey" FOREIGN KEY ("client_id") REFERENCES "public"."clients" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "workspaces_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "workspaces_updated_by_fkey" FOREIGN KEY ("updated_by") REFERENCES "public"."internal_users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Atlas Community schema inspection omits materialized views; keep the authoritative M6 view in this canonical snapshot.
CREATE MATERIALIZED VIEW "public"."analytics_workspace_daily" AS
WITH costs AS (
    SELECT workspace_id,campaign_id,occurred_at::date metric_date,sum(normalized_amount_usd) provider_cost_usd,0::bigint social_views,0::bigint social_impressions,0::bigint social_clicks,0::bigint ad_spend_minor,0::bigint ad_impressions,0::bigint ad_clicks,0::numeric ad_conversions,0::bigint ad_revenue_minor
    FROM cost_records WHERE workspace_id IS NOT NULL AND campaign_id IS NOT NULL GROUP BY workspace_id,campaign_id,occurred_at::date
), social AS (
    SELECT p.workspace_id,p.campaign_id,m.metric_date,0::numeric provider_cost_usd,sum(m.views)::bigint social_views,sum(m.impressions)::bigint social_impressions,sum(m.link_clicks)::bigint social_clicks,0::bigint ad_spend_minor,0::bigint ad_impressions,0::bigint ad_clicks,0::numeric ad_conversions,0::bigint ad_revenue_minor
    FROM social_post_metrics_daily m JOIN social_posts p ON p.id=m.social_post_id GROUP BY p.workspace_id,p.campaign_id,m.metric_date
), ads AS (
    SELECT a.workspace_id,a.campaign_id,m.metric_date,0::numeric provider_cost_usd,0::bigint social_views,0::bigint social_impressions,0::bigint social_clicks,sum(m.spend_minor)::bigint ad_spend_minor,sum(m.impressions)::bigint ad_impressions,sum(m.clicks)::bigint ad_clicks,sum(m.conversions) ad_conversions,sum(m.revenue_minor)::bigint ad_revenue_minor
    FROM ad_campaign_metrics_daily m JOIN ad_campaigns a ON a.id=m.ad_campaign_id GROUP BY a.workspace_id,a.campaign_id,m.metric_date
), combined AS (SELECT * FROM costs UNION ALL SELECT * FROM social UNION ALL SELECT * FROM ads)
SELECT workspace_id,campaign_id,metric_date,sum(provider_cost_usd) provider_cost_usd,sum(social_views)::bigint social_views,sum(social_impressions)::bigint social_impressions,sum(social_clicks)::bigint social_clicks,sum(ad_spend_minor)::bigint ad_spend_minor,sum(ad_impressions)::bigint ad_impressions,sum(ad_clicks)::bigint ad_clicks,sum(ad_conversions) ad_conversions,sum(ad_revenue_minor)::bigint ad_revenue_minor
FROM combined GROUP BY workspace_id,campaign_id,metric_date;
CREATE UNIQUE INDEX "analytics_workspace_daily_unique_idx" ON "public"."analytics_workspace_daily" ("workspace_id", "campaign_id", "metric_date");
