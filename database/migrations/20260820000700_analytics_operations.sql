CREATE TYPE usage_outcome AS ENUM ('SUCCESS','FAILURE');
CREATE TYPE recommendation_status AS ENUM ('DRAFT','APPROVED','REJECTED','APPLIED','DISMISSED');
CREATE TYPE notification_severity AS ENUM ('INFO','WARNING','CRITICAL');

CREATE TABLE exchange_rate_snapshots (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    base_currency char(3) NOT NULL,
    quote_currency char(3) NOT NULL,
    rate numeric(24,10) NOT NULL CHECK (rate > 0),
    source text NOT NULL,
    captured_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (base_currency,quote_currency,captured_at),
    CHECK (base_currency <> quote_currency)
);

CREATE TABLE usage_ledger (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    provider text NOT NULL,
    model text NOT NULL DEFAULT '',
    request_reference text NOT NULL,
    operation text NOT NULL,
    client_id uuid REFERENCES clients(id),
    workspace_id uuid REFERENCES workspaces(id),
    campaign_id uuid REFERENCES campaigns(id),
    scene_id uuid REFERENCES scenes(id),
    video_project_id uuid REFERENCES video_projects(id),
    input_units bigint NOT NULL DEFAULT 0 CHECK (input_units >= 0),
    output_units bigint NOT NULL DEFAULT 0 CHECK (output_units >= 0),
    generated_seconds numeric(12,3) NOT NULL DEFAULT 0 CHECK (generated_seconds >= 0),
    accepted_seconds numeric(12,3) NOT NULL DEFAULT 0 CHECK (accepted_seconds >= 0 AND accepted_seconds <= generated_seconds),
    provider_reported_cost numeric(18,6) CHECK (provider_reported_cost IS NULL OR provider_reported_cost >= 0),
    estimated_cost numeric(18,6) NOT NULL DEFAULT 0 CHECK (estimated_cost >= 0),
    currency char(3) NOT NULL DEFAULT 'USD',
    exchange_rate_snapshot_id uuid REFERENCES exchange_rate_snapshots(id),
    outcome usage_outcome NOT NULL,
    reused boolean NOT NULL DEFAULT false,
    metadata jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(metadata)='object'),
    occurred_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider,request_reference,operation)
);
CREATE INDEX usage_ledger_scope_time_idx ON usage_ledger(client_id,workspace_id,occurred_at DESC);
CREATE INDEX usage_ledger_campaign_time_idx ON usage_ledger(campaign_id,occurred_at DESC);
CREATE INDEX usage_ledger_provider_time_idx ON usage_ledger(provider,occurred_at DESC);

CREATE TABLE cost_records (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    usage_ledger_id uuid REFERENCES usage_ledger(id),
    client_id uuid REFERENCES clients(id),
    workspace_id uuid REFERENCES workspaces(id),
    campaign_id uuid REFERENCES campaigns(id),
    category text NOT NULL CHECK (category IN ('LLM','SEEDANCE','TRANSCRIPTION','RENDER','STORAGE','META','OTHER')),
    provider text NOT NULL,
    amount numeric(18,6) NOT NULL CHECK (amount >= 0),
    currency char(3) NOT NULL,
    normalized_amount_usd numeric(18,6) NOT NULL CHECK (normalized_amount_usd >= 0),
    estimated boolean NOT NULL DEFAULT false,
    metadata jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(metadata)='object'),
    occurred_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (usage_ledger_id,category)
);
CREATE INDEX cost_records_scope_time_idx ON cost_records(client_id,workspace_id,occurred_at DESC);
CREATE INDEX cost_records_campaign_time_idx ON cost_records(campaign_id,occurred_at DESC);

CREATE TABLE ad_recommendations (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    client_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    campaign_id uuid REFERENCES campaigns(id),
    ad_campaign_id uuid REFERENCES ad_campaigns(id) ON DELETE CASCADE,
    recommendation_type text NOT NULL CHECK (recommendation_type IN ('PAUSE_WEAK_CREATIVE','SCALE_WINNER','INVESTIGATE_LOW_CTR','INVESTIGATE_RISING_CPC','INVESTIGATE_HIGH_FREQUENCY','INVESTIGATE_CREATIVE_FATIGUE','INVESTIGATE_CPA','SUGGEST_HOOK','SUGGEST_CTA','SHORTEN_SCENE','LOWEST_COST_TEMPLATE','NEXT_CAMPAIGN_DIRECTION')),
    recommendation_hash text NOT NULL UNIQUE,
    input_snapshot jsonb NOT NULL CHECK (jsonb_typeof(input_snapshot)='object'),
    model text NOT NULL,
    output text NOT NULL,
    rationale text NOT NULL,
    status recommendation_status NOT NULL DEFAULT 'DRAFT',
    reviewer_id uuid REFERENCES internal_users(id),
    review_notes text NOT NULL DEFAULT '',
    reviewed_at timestamptz,
    action_taken text NOT NULL DEFAULT '',
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by uuid NOT NULL REFERENCES internal_users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (workspace_id,client_id) REFERENCES workspaces(id,client_id),
    CONSTRAINT recommendation_review_shape CHECK ((status='DRAFT')=(reviewed_at IS NULL))
);
CREATE INDEX ad_recommendations_scope_status_idx ON ad_recommendations(client_id,workspace_id,status,created_at DESC);

CREATE TABLE notifications (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    internal_user_id uuid REFERENCES internal_users(id) ON DELETE CASCADE,
    client_id uuid REFERENCES clients(id),
    workspace_id uuid REFERENCES workspaces(id),
    severity notification_severity NOT NULL DEFAULT 'INFO',
    notification_type text NOT NULL,
    title text NOT NULL,
    message text NOT NULL,
    entity_type text,
    entity_id uuid,
    read_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notifications_user_unread_idx ON notifications(internal_user_id,created_at DESC) WHERE read_at IS NULL;

INSERT INTO usage_ledger(provider,model,request_reference,operation,client_id,workspace_id,campaign_id,input_units,output_units,provider_reported_cost,estimated_cost,currency,outcome,reused,occurred_at,metadata)
SELECT provider,model,id::text,operation::text,client_id,workspace_id,campaign_id,COALESCE(input_tokens,0),COALESCE(output_tokens,0),actual_cost_usd,estimated_cost_usd,'USD',CASE WHEN status='SUCCEEDED' THEN 'SUCCESS'::usage_outcome ELSE 'FAILURE'::usage_outcome END,false,started_at,jsonb_build_object('source','provider_requests','promptVersion',prompt_version)
FROM provider_requests
ON CONFLICT DO NOTHING;

INSERT INTO usage_ledger(provider,model,request_reference,operation,client_id,workspace_id,campaign_id,scene_id,input_units,generated_seconds,accepted_seconds,provider_reported_cost,estimated_cost,currency,outcome,reused,occurred_at,metadata)
SELECT provider,model,id::text,'SEEDANCE_VIDEO',client_id,workspace_id,campaign_id,scene_id,COALESCE(usage_tokens,0),duration_seconds,CASE WHEN status='APPROVED' THEN duration_seconds ELSE 0 END,actual_cost_usd,estimated_cost_usd,'USD',CASE WHEN status IN ('SUCCEEDED','DOWNLOADING','VALIDATING','REVIEW_REQUIRED','APPROVED','REJECTED') THEN 'SUCCESS'::usage_outcome ELSE 'FAILURE'::usage_outcome END,false,created_at,jsonb_build_object('source','scene_generation_tasks','attemptNumber',attempt_number,'status',status)
FROM scene_generation_tasks
WHERE status NOT IN ('DRAFT','READY','QUEUED')
ON CONFLICT DO NOTHING;

INSERT INTO usage_ledger(provider,model,request_reference,operation,client_id,workspace_id,campaign_id,video_project_id,generated_seconds,accepted_seconds,estimated_cost,currency,outcome,reused,occurred_at,metadata)
SELECT 'renderer','remotion-4.0.513',r.id::text,'FINAL_RENDER',r.client_id,r.workspace_id,r.campaign_id,r.video_project_id,
       COALESCE((SELECT sum(sv.duration_seconds) FROM scenes s JOIN scene_versions sv ON sv.scene_id=s.id AND sv.version=s.current_version WHERE s.campaign_id=r.campaign_id),0),
       CASE WHEN r.status='APPROVED' THEN COALESCE((SELECT sum(sv.duration_seconds) FROM scenes s JOIN scene_versions sv ON sv.scene_id=s.id AND sv.version=s.current_version WHERE s.campaign_id=r.campaign_id),0) ELSE 0 END,
       0,'USD',CASE WHEN r.status IN ('REVIEW_REQUIRED','APPROVED','REJECTED') THEN 'SUCCESS'::usage_outcome ELSE 'FAILURE'::usage_outcome END,false,r.created_at,jsonb_build_object('source','render_jobs','status',r.status)
FROM render_jobs r
WHERE r.status NOT IN ('QUEUED','BUILDING_MANIFEST','RENDERING','VALIDATING','UPLOADING')
ON CONFLICT DO NOTHING;

INSERT INTO cost_records(usage_ledger_id,client_id,workspace_id,campaign_id,category,provider,amount,currency,normalized_amount_usd,estimated,occurred_at)
SELECT id,client_id,workspace_id,campaign_id,
       CASE WHEN operation IN ('CONCEPTS','CONTENT','SCRIPT','SCENES') THEN 'LLM' WHEN operation='SEEDANCE_VIDEO' THEN 'SEEDANCE' WHEN operation='FINAL_RENDER' THEN 'RENDER' ELSE 'OTHER' END,
       provider,COALESCE(provider_reported_cost,estimated_cost),currency,CASE WHEN currency='USD' THEN COALESCE(provider_reported_cost,estimated_cost) ELSE 0 END,provider_reported_cost IS NULL,occurred_at
FROM usage_ledger
ON CONFLICT DO NOTHING;

CREATE MATERIALIZED VIEW analytics_workspace_daily AS
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
CREATE UNIQUE INDEX analytics_workspace_daily_unique_idx ON analytics_workspace_daily(workspace_id,campaign_id,metric_date);
