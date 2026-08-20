package operations

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
)

type QueueMetric struct {
	Queue            string  `json:"queue"`
	State            string  `json:"state"`
	Count            int64   `json:"count"`
	OldestAgeSeconds float64 `json:"oldestAgeSeconds"`
}
type StateCount struct {
	State string `json:"state"`
	Count int64  `json:"count"`
}
type JobView struct {
	ID          int64            `json:"id"`
	Kind        string           `json:"kind"`
	Queue       string           `json:"queue"`
	State       string           `json:"state"`
	Attempt     int              `json:"attempt"`
	MaxAttempts int              `json:"maxAttempts"`
	ScheduledAt time.Time        `json:"scheduledAt"`
	AttemptedAt *time.Time       `json:"attemptedAt"`
	AgeSeconds  float64          `json:"ageSeconds"`
	Errors      []map[string]any `json:"errors"`
}
type CostDimension struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	USD  float64 `json:"usd"`
}
type FailureCounts struct {
	Webhooks   int64 `json:"webhooks"`
	Publishing int64 `json:"publishing"`
	MetaAds    int64 `json:"metaAds"`
	Uploads    int64 `json:"uploads"`
}
type AuditView struct {
	ID         string    `json:"id"`
	Action     string    `json:"action"`
	Outcome    string    `json:"outcome"`
	EntityType *string   `json:"entityType"`
	OccurredAt time.Time `json:"occurredAt"`
}
type FeatureFlag struct {
	Key         string         `json:"key"`
	Description string         `json:"description"`
	Enabled     bool           `json:"enabled"`
	SafeConfig  map[string]any `json:"safeConfig"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}
type VersionView struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	PromptVersion string `json:"promptVersion"`
	APIVersion    string `json:"apiVersion"`
}
type WebhookView struct {
	Provider     string    `json:"provider"`
	EventType    string    `json:"eventType"`
	Status       string    `json:"status"`
	AttemptCount int       `json:"attemptCount"`
	ReceivedAt   time.Time `json:"receivedAt"`
}
type ProviderFailureView struct {
	System     string    `json:"system"`
	EntityID   string    `json:"entityId"`
	Code       string    `json:"code"`
	OccurredAt time.Time `json:"occurredAt"`
}
type NotificationView struct {
	ID        string    `json:"id"`
	Severity  string    `json:"severity"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}
type ConsoleOverview struct {
	GeneratedAt       time.Time             `json:"generatedAt"`
	APIHealth         string                `json:"apiHealth"`
	MaintenanceMode   bool                  `json:"maintenanceMode"`
	LastMaintenanceAt *time.Time            `json:"lastMaintenanceAt"`
	Queues            []QueueMetric         `json:"queues"`
	Jobs              []JobView             `json:"jobs"`
	SeedanceTasks     []StateCount          `json:"seedanceTasks"`
	RenderJobs        []StateCount          `json:"renderJobs"`
	Webhooks          []WebhookView         `json:"webhooks"`
	ProviderErrors    []ProviderFailureView `json:"providerErrors"`
	CostAnomalies     []NotificationView    `json:"costAnomalies"`
	DailyCostUSD      float64               `json:"dailyCostUsd"`
	MonthlyCostUSD    float64               `json:"monthlyCostUsd"`
	AttemptFactor     float64               `json:"attemptFactor"`
	StorageBytes      float64               `json:"storageBytes"`
	CostByClient      []CostDimension       `json:"costByClient"`
	CostByCampaign    []CostDimension       `json:"costByCampaign"`
	Failures          FailureCounts         `json:"failures"`
	Versions          []VersionView         `json:"versions"`
	AuditLogs         []AuditView           `json:"auditLogs"`
	FeatureFlags      []FeatureFlag         `json:"featureFlags"`
	SafeConfiguration map[string]any        `json:"safeConfiguration"`
}

func (h *Handler) Overview(c fiber.Ctx) error {
	ctx := c.Context()
	out := ConsoleOverview{GeneratedAt: time.Now().UTC(), APIHealth: "ok", Queues: []QueueMetric{}, Jobs: []JobView{}, SeedanceTasks: []StateCount{}, RenderJobs: []StateCount{}, Webhooks: []WebhookView{}, ProviderErrors: []ProviderFailureView{}, CostAnomalies: []NotificationView{}, CostByClient: []CostDimension{}, CostByCampaign: []CostDimension{}, Versions: []VersionView{}, AuditLogs: []AuditView{}, FeatureFlags: []FeatureFlag{}, SafeConfiguration: safeConfiguration(h.config)}
	rows, err := h.pool.Query(ctx, `SELECT queue,state::text,count(*),COALESCE(EXTRACT(EPOCH FROM(now()-min(CASE WHEN state IN ('available','scheduled','retryable') THEN scheduled_at ELSE created_at END))),0) FROM river_job GROUP BY queue,state ORDER BY queue,state`)
	if err != nil {
		return consoleError(c, err)
	}
	for rows.Next() {
		var x QueueMetric
		if err = rows.Scan(&x.Queue, &x.State, &x.Count, &x.OldestAgeSeconds); err != nil {
			rows.Close()
			return consoleError(c, err)
		}
		out.Queues = append(out.Queues, x)
	}
	rows.Close()
	rows, err = h.pool.Query(ctx, `SELECT id,kind,queue,state::text,attempt,max_attempts,scheduled_at,attempted_at,EXTRACT(EPOCH FROM(now()-COALESCE(attempted_at,scheduled_at,created_at))),COALESCE(array_to_json(errors),'[]'::json)::jsonb FROM river_job WHERE state IN ('running','retryable','discarded','cancelled') OR (state IN ('available','scheduled') AND scheduled_at<now()-interval '15 minutes') ORDER BY CASE WHEN state='running' THEN 0 WHEN state='discarded' THEN 1 ELSE 2 END,created_at DESC LIMIT 100`)
	if err != nil {
		return consoleError(c, err)
	}
	for rows.Next() {
		var x JobView
		var raw []byte
		if err = rows.Scan(&x.ID, &x.Kind, &x.Queue, &x.State, &x.Attempt, &x.MaxAttempts, &x.ScheduledAt, &x.AttemptedAt, &x.AgeSeconds, &raw); err != nil {
			rows.Close()
			return consoleError(c, err)
		}
		_ = json.Unmarshal(raw, &x.Errors)
		out.Jobs = append(out.Jobs, x)
	}
	rows.Close()
	for _, stateQuery := range []struct {
		query string
		items *[]StateCount
	}{
		{`SELECT status::text,count(*) FROM scene_generation_tasks GROUP BY status ORDER BY status`, &out.SeedanceTasks},
		{`SELECT status::text,count(*) FROM render_jobs GROUP BY status ORDER BY status`, &out.RenderJobs},
	} {
		rows, err = h.pool.Query(ctx, stateQuery.query)
		if err != nil {
			return consoleError(c, err)
		}
		for rows.Next() {
			var item StateCount
			if err = rows.Scan(&item.State, &item.Count); err != nil {
				rows.Close()
				return consoleError(c, err)
			}
			*stateQuery.items = append(*stateQuery.items, item)
		}
		rows.Close()
	}
	rows, err = h.pool.Query(ctx, `SELECT provider,event_type,CASE WHEN processing_error IS NOT NULL THEN 'FAILED' WHEN processed_at IS NOT NULL THEN 'PROCESSED' ELSE 'PENDING' END,attempt_count,received_at FROM webhook_events UNION ALL SELECT 'meta',object_type,CASE WHEN processing_error IS NOT NULL THEN 'FAILED' WHEN processed_at IS NOT NULL THEN 'PROCESSED' ELSE 'PENDING' END,0,received_at FROM meta_webhook_events ORDER BY received_at DESC LIMIT 50`)
	if err != nil {
		return consoleError(c, err)
	}
	for rows.Next() {
		var item WebhookView
		if err = rows.Scan(&item.Provider, &item.EventType, &item.Status, &item.AttemptCount, &item.ReceivedAt); err != nil {
			rows.Close()
			return consoleError(c, err)
		}
		out.Webhooks = append(out.Webhooks, item)
	}
	rows.Close()
	rows, err = h.pool.Query(ctx, `SELECT system,entity_id,code,occurred_at FROM (SELECT 'seedance' system,id::text entity_id,COALESCE(error_code,'failed') code,updated_at occurred_at FROM scene_generation_tasks WHERE status='FAILED' UNION ALL SELECT 'renderer',id::text,COALESCE(error_code,'failed'),updated_at FROM render_jobs WHERE status='FAILED' UNION ALL SELECT 'meta-publishing',id::text,COALESCE(error_code,'failed'),updated_at FROM social_posts WHERE status IN ('FAILED','PERMANENT_FAILURE') UNION ALL SELECT 'meta-ads',id::text,COALESCE(error_code,'failed'),COALESCE(completed_at,reviewed_at,requested_at) FROM meta_ad_actions WHERE status='FAILED' UNION ALL SELECT 'r2-upload',id::text,'upload_failed',created_at FROM media_uploads WHERE status='FAILED') failures ORDER BY occurred_at DESC LIMIT 50`)
	if err != nil {
		return consoleError(c, err)
	}
	for rows.Next() {
		var item ProviderFailureView
		if err = rows.Scan(&item.System, &item.EntityID, &item.Code, &item.OccurredAt); err != nil {
			rows.Close()
			return consoleError(c, err)
		}
		out.ProviderErrors = append(out.ProviderErrors, item)
	}
	rows.Close()
	rows, err = h.pool.Query(ctx, `SELECT id::text,severity::text,notification_type,title,message,created_at FROM notifications WHERE notification_type='DAILY_COST_ANOMALY' ORDER BY created_at DESC LIMIT 20`)
	if err != nil {
		return consoleError(c, err)
	}
	for rows.Next() {
		var item NotificationView
		if err = rows.Scan(&item.ID, &item.Severity, &item.Type, &item.Title, &item.Message, &item.CreatedAt); err != nil {
			rows.Close()
			return consoleError(c, err)
		}
		out.CostAnomalies = append(out.CostAnomalies, item)
	}
	rows.Close()
	if err = h.pool.QueryRow(ctx, `SELECT max(finalized_at) FROM river_job WHERE kind='maintenance.cleanup' AND state='completed'`).Scan(&out.LastMaintenanceAt); err != nil {
		return consoleError(c, err)
	}
	if err = h.pool.QueryRow(ctx, `SELECT COALESCE(sum(normalized_amount_usd) FILTER(WHERE occurred_at>=date_trunc('day',now())),0)::float8,COALESCE(sum(normalized_amount_usd) FILTER(WHERE occurred_at>=date_trunc('month',now())),0)::float8 FROM cost_records`).Scan(&out.DailyCostUSD, &out.MonthlyCostUSD); err != nil {
		return consoleError(c, err)
	}
	if err = h.pool.QueryRow(ctx, `SELECT COALESCE(count(*)::float8/NULLIF(count(DISTINCT scene_id),0),0) FROM scene_generation_tasks`).Scan(&out.AttemptFactor); err != nil {
		return consoleError(c, err)
	}
	if err = h.pool.QueryRow(ctx, `SELECT COALESCE(sum(file_size_bytes),0)::float8 FROM media_asset_versions`).Scan(&out.StorageBytes); err != nil {
		return consoleError(c, err)
	}
	rows, err = h.pool.Query(ctx, `SELECT c.id::text,c.company_name,sum(r.normalized_amount_usd)::float8 FROM cost_records r JOIN clients c ON c.id=r.client_id WHERE r.occurred_at>=date_trunc('month',now()) GROUP BY c.id,c.company_name ORDER BY sum(r.normalized_amount_usd) DESC LIMIT 20`)
	if err != nil {
		return consoleError(c, err)
	}
	for rows.Next() {
		var x CostDimension
		if err = rows.Scan(&x.ID, &x.Name, &x.USD); err != nil {
			rows.Close()
			return consoleError(c, err)
		}
		out.CostByClient = append(out.CostByClient, x)
	}
	rows.Close()
	rows, err = h.pool.Query(ctx, `SELECT c.id::text,c.name,sum(r.normalized_amount_usd)::float8 FROM cost_records r JOIN campaigns c ON c.id=r.campaign_id WHERE r.occurred_at>=date_trunc('month',now()) GROUP BY c.id,c.name ORDER BY sum(r.normalized_amount_usd) DESC LIMIT 20`)
	if err != nil {
		return consoleError(c, err)
	}
	for rows.Next() {
		var x CostDimension
		if err = rows.Scan(&x.ID, &x.Name, &x.USD); err != nil {
			rows.Close()
			return consoleError(c, err)
		}
		out.CostByCampaign = append(out.CostByCampaign, x)
	}
	rows.Close()
	if err = h.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM webhook_events WHERE processing_error IS NOT NULL)+(SELECT count(*) FROM meta_webhook_events WHERE processing_error IS NOT NULL),(SELECT count(*) FROM social_posts WHERE status IN ('FAILED','PERMANENT_FAILURE')),(SELECT count(*) FROM meta_ad_actions WHERE status='FAILED'),(SELECT count(*) FROM media_uploads WHERE status='FAILED')`).Scan(&out.Failures.Webhooks, &out.Failures.Publishing, &out.Failures.MetaAds, &out.Failures.Uploads); err != nil {
		return consoleError(c, err)
	}
	rows, err = h.pool.Query(ctx, `SELECT DISTINCT provider,model,prompt_version,'' FROM provider_requests UNION SELECT DISTINCT provider,model,'',api_version FROM scene_generation_tasks UNION SELECT 'meta','','',api_version FROM meta_connections ORDER BY 1,2,3,4`)
	if err != nil {
		return consoleError(c, err)
	}
	for rows.Next() {
		var x VersionView
		if err = rows.Scan(&x.Provider, &x.Model, &x.PromptVersion, &x.APIVersion); err != nil {
			rows.Close()
			return consoleError(c, err)
		}
		out.Versions = append(out.Versions, x)
	}
	rows.Close()
	rows, err = h.pool.Query(ctx, `SELECT id::text,action,outcome,entity_type,occurred_at FROM audit_logs ORDER BY occurred_at DESC LIMIT 50`)
	if err != nil {
		return consoleError(c, err)
	}
	for rows.Next() {
		var x AuditView
		if err = rows.Scan(&x.ID, &x.Action, &x.Outcome, &x.EntityType, &x.OccurredAt); err != nil {
			rows.Close()
			return consoleError(c, err)
		}
		out.AuditLogs = append(out.AuditLogs, x)
	}
	rows.Close()
	rows, err = h.pool.Query(ctx, `SELECT key,description,enabled,safe_config,updated_at FROM feature_flags ORDER BY key`)
	if err != nil {
		return consoleError(c, err)
	}
	for rows.Next() {
		var x FeatureFlag
		var raw []byte
		if err = rows.Scan(&x.Key, &x.Description, &x.Enabled, &raw, &x.UpdatedAt); err != nil {
			rows.Close()
			return consoleError(c, err)
		}
		_ = json.Unmarshal(raw, &x.SafeConfig)
		if x.Key == "maintenance_mode" {
			out.MaintenanceMode = x.Enabled
		}
		out.FeatureFlags = append(out.FeatureFlags, x)
	}
	rows.Close()
	return c.JSON(out)
}

func (h *Handler) RetryJob(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("jobId"), 10, 64)
	if err != nil || id < 1 {
		return problem.Write(c, 422, "job-invalid", "Job ID không hợp lệ", "Job ID phải là số dương.")
	}
	client, e := river.NewClient(riverpgxv5.New(h.pool), &river.Config{})
	if e != nil {
		return consoleError(c, e)
	}
	job, e := client.JobRetry(c.Context(), id)
	if errors.Is(e, river.ErrNotFound) {
		return problem.Write(c, 404, "job-not-found", "Không tìm thấy job", "Job đã bị xóa hoặc không tồn tại.")
	}
	if e != nil {
		return consoleError(c, e)
	}
	return c.JSON(jobView(job))
}
func (h *Handler) CancelJob(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("jobId"), 10, 64)
	if err != nil || id < 1 {
		return problem.Write(c, 422, "job-invalid", "Job ID không hợp lệ", "Job ID phải là số dương.")
	}
	client, e := river.NewClient(riverpgxv5.New(h.pool), &river.Config{})
	if e != nil {
		return consoleError(c, e)
	}
	job, e := client.JobCancel(c.Context(), id)
	if errors.Is(e, river.ErrNotFound) {
		return problem.Write(c, 404, "job-not-found", "Không tìm thấy job", "Job đã bị xóa hoặc không tồn tại.")
	}
	if e != nil {
		return consoleError(c, e)
	}
	return c.JSON(jobView(job))
}
func (h *Handler) SetMaintenance(c fiber.Ctx) error {
	var input struct {
		Enabled bool   `json:"enabled"`
		Reason  string `json:"reason"`
	}
	if c.Bind().Body(&input) != nil || len(input.Reason) > 500 {
		return problem.Write(c, 422, "maintenance-invalid", "Maintenance config không hợp lệ", "Reason tối đa 500 ký tự.")
	}
	_, err := h.pool.Exec(c.Context(), `INSERT INTO feature_flags(key,description,enabled,safe_config,updated_at)VALUES('maintenance_mode','Block non-operational mutations during maintenance',$1,jsonb_build_object('reason',$2),now())ON CONFLICT(key)DO UPDATE SET enabled=EXCLUDED.enabled,safe_config=EXCLUDED.safe_config,updated_at=now()`, input.Enabled, strings.TrimSpace(input.Reason))
	if err != nil {
		return consoleError(c, err)
	}
	return c.JSON(fiber.Map{"enabled": input.Enabled, "reason": strings.TrimSpace(input.Reason)})
}

func jobView(job *rivertype.JobRow) JobView {
	return JobView{ID: job.ID, Kind: job.Kind, Queue: job.Queue, State: string(job.State), Attempt: job.Attempt, MaxAttempts: job.MaxAttempts, ScheduledAt: job.ScheduledAt, AttemptedAt: job.AttemptedAt, AgeSeconds: time.Since(job.CreatedAt).Seconds(), Errors: []map[string]any{}}
}
func safeConfiguration(cfg config.Config) map[string]any {
	return map[string]any{"environment": cfg.Environment, "demoMode": cfg.DemoMode, "openai": map[string]any{"configured": cfg.OpenAI.Validate() == nil, "model": cfg.OpenAI.Model, "baseUrl": safeHost(cfg.OpenAI.BaseURL)}, "seedance": map[string]any{"configured": cfg.Seedance.Validate() == nil, "model": cfg.Seedance.Model, "apiVersion": cfg.Seedance.APIVersion}, "meta": map[string]any{"configured": cfg.Meta.Validate() == nil, "apiVersion": cfg.Meta.APIVersion}, "renderer": map[string]any{"configured": rendererConfigured(cfg.Renderer), "baseUrl": safeHost(cfg.Renderer.BaseURL)}, "storage": map[string]any{"configured": cfg.R2.Validate() == nil, "bucket": cfg.R2.Bucket}}
}
func consoleError(c fiber.Ctx, _ error) error {
	return problem.Write(c, 500, "operations-error", "Không thể tải Operations Console", "Kiểm tra database, River và provider health.")
}

func MaintenanceGuard(pool *pgxpool.Pool) fiber.Handler {
	var enabled atomic.Bool
	var expires atomic.Int64
	var mu sync.Mutex
	return func(c fiber.Ctx) error {
		if c.Method() == fiber.MethodGet || strings.HasSuffix(c.Path(), "/operations/maintenance") {
			return c.Next()
		}
		now := time.Now().UnixNano()
		if now >= expires.Load() {
			mu.Lock()
			if now >= expires.Load() {
				var value bool
				_ = pool.QueryRow(context.Background(), `SELECT COALESCE((SELECT enabled FROM feature_flags WHERE key='maintenance_mode'),false)`).Scan(&value)
				enabled.Store(value)
				expires.Store(time.Now().Add(5 * time.Second).UnixNano())
			}
			mu.Unlock()
		}
		if enabled.Load() {
			return problem.Write(c, 503, "maintenance-mode", "Hệ thống đang bảo trì", "Mutation tạm thời bị khóa; thao tác đọc và Operations Console vẫn khả dụng.")
		}
		return c.Next()
	}
}
