package planning

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	studioai "github.com/internal/ai-product-marketing-studio/services/api/internal/ai"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/jobs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/providerconfigs"
)

var (
	ErrInvalid      = errors.New("invalid planning operation")
	ErrNotFound     = errors.New("planning resource not found")
	ErrConflict     = errors.New("planning conflict")
	ErrPrerequisite = errors.New("planning prerequisite not satisfied")
	ErrLocked       = errors.New("planning resource is locked")
)

type GenerationJob struct {
	ID               uuid.UUID      `json:"id"`
	CampaignID       uuid.UUID      `json:"campaignId"`
	Operation        string         `json:"operation"`
	Status           string         `json:"status"`
	RiverJobID       *int64         `json:"riverJobId"`
	EstimatedCostUSD float64        `json:"estimatedCostUsd"`
	ActualCostUSD    *float64       `json:"actualCostUsd"`
	OutputSummary    map[string]any `json:"outputSummary"`
	ErrorCode        *string        `json:"errorCode"`
	ErrorMessage     *string        `json:"errorMessage"`
	CreatedAt        time.Time      `json:"createdAt"`
	StartedAt        *time.Time     `json:"startedAt"`
	CompletedAt      *time.Time     `json:"completedAt"`
}

type CostEstimate struct {
	ID                    uuid.UUID      `json:"id"`
	CampaignID            uuid.UUID      `json:"campaignId"`
	Operation             string         `json:"operation"`
	Model                 string         `json:"model"`
	Currency              string         `json:"currency"`
	EstimatedInputTokens  int64          `json:"estimatedInputTokens"`
	EstimatedOutputTokens int64          `json:"estimatedOutputTokens"`
	EstimatedVideoSeconds int64          `json:"estimatedVideoSeconds"`
	EstimatedCost         float64        `json:"estimatedCost"`
	Assumptions           map[string]any `json:"assumptions"`
	ExpiresAt             time.Time      `json:"expiresAt"`
	CreatedAt             time.Time      `json:"createdAt"`
}

type Enqueuer interface {
	EnqueueAIPlanning(context.Context, pgx.Tx, jobs.AIPlanningArgs) (int64, error)
}

type Service struct {
	pool     *pgxpool.Pool
	enqueuer Enqueuer
	config   config.Config
	resolver interface {
		Load(context.Context, uuid.UUID) (providerconfigs.Bundle, error)
	}
}

func NewService(pool *pgxpool.Pool, enqueuer Enqueuer, cfg config.Config) *Service {
	return &Service{pool: pool, enqueuer: enqueuer, config: cfg}
}

func NewTenantService(pool *pgxpool.Pool, enqueuer Enqueuer, resolver interface {
	Load(context.Context, uuid.UUID) (providerconfigs.Bundle, error)
}) *Service {
	return &Service{pool: pool, enqueuer: enqueuer, resolver: resolver}
}

func (s *Service) effectiveConfig(ctx context.Context, clientID uuid.UUID) (config.Config, error) {
	if s.resolver == nil {
		return s.config, nil
	}
	bundle, err := s.resolver.Load(ctx, clientID)
	if err != nil || strings.TrimSpace(bundle.OpenAI.Model) == "" || (!bundle.DemoMode && bundle.OpenAI.Validate() != nil) {
		return config.Config{}, ErrPrerequisite
	}
	return config.Config{DemoMode: bundle.DemoMode, OpenAI: bundle.OpenAI, Seedance: bundle.Seedance, R2: bundle.R2, Meta: bundle.Meta, Renderer: bundle.Renderer}, nil
}

func (s *Service) Estimate(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID, operation string) (CostEstimate, error) {
	cfg, err := s.effectiveConfig(ctx, clientID)
	if err != nil {
		return CostEstimate{}, err
	}
	operation = strings.ToUpper(strings.TrimSpace(operation))
	inputTokens, outputTokens, err := tokenEstimate(operation)
	if err != nil {
		return CostEstimate{}, err
	}
	var duration int64
	if err = s.pool.QueryRow(ctx, `SELECT v.duration_seconds FROM campaigns c JOIN campaign_versions v ON v.campaign_id=c.id AND v.version=c.current_version WHERE c.id=$1 AND c.client_id=$2 AND c.workspace_id=$3 AND c.status<>'ARCHIVED'`, campaignID, clientID, workspaceID).Scan(&duration); errors.Is(err, pgx.ErrNoRows) {
		return CostEstimate{}, ErrNotFound
	} else if err != nil {
		return CostEstimate{}, err
	}
	videoSeconds := int64(0)
	if operation == "SCENES" {
		videoSeconds = duration
	}
	cost := float64(inputTokens)/1_000_000*cfg.OpenAI.InputUSDPer1M + float64(outputTokens)/1_000_000*cfg.OpenAI.OutputUSDPer1M + float64(videoSeconds)*cfg.Seedance.USDPerSecond
	assumptions := map[string]any{"openaiInputUsdPer1M": cfg.OpenAI.InputUSDPer1M, "openaiOutputUsdPer1M": cfg.OpenAI.OutputUSDPer1M, "seedanceUsdPerSecond": cfg.Seedance.USDPerSecond, "providerPricingConfigured": cfg.OpenAI.InputUSDPer1M > 0 || cfg.OpenAI.OutputUSDPer1M > 0 || cfg.Seedance.USDPerSecond > 0}
	expires := time.Now().UTC().Add(30 * time.Minute)
	encoded, _ := json.Marshal(assumptions)
	var item CostEstimate
	var raw []byte
	err = s.pool.QueryRow(ctx, `INSERT INTO cost_estimates(client_id,workspace_id,campaign_id,operation,model,estimated_input_tokens,estimated_output_tokens,estimated_video_seconds,estimated_cost,assumptions,expires_at) VALUES($1,$2,$3,$4::generation_operation,$5,$6,$7,$8,$9,$10,$11) RETURNING id,campaign_id,operation::text,model,currency,estimated_input_tokens,estimated_output_tokens,estimated_video_seconds,estimated_cost::float8,assumptions,expires_at,created_at`, clientID, workspaceID, campaignID, operation, cfg.OpenAI.Model, inputTokens, outputTokens, videoSeconds, cost, encoded, expires).Scan(&item.ID, &item.CampaignID, &item.Operation, &item.Model, &item.Currency, &item.EstimatedInputTokens, &item.EstimatedOutputTokens, &item.EstimatedVideoSeconds, &item.EstimatedCost, &raw, &item.ExpiresAt, &item.CreatedAt)
	if err != nil {
		return CostEstimate{}, err
	}
	_ = json.Unmarshal(raw, &item.Assumptions)
	return item, nil
}

func (s *Service) StartGeneration(ctx context.Context, clientID, workspaceID, campaignID, actorID uuid.UUID, operation, idempotencyKey string) (GenerationJob, error) {
	cfg, configErr := s.effectiveConfig(ctx, clientID)
	if configErr != nil {
		return GenerationJob{}, configErr
	}
	operation = strings.ToUpper(strings.TrimSpace(operation))
	if _, _, err := tokenEstimate(operation); err != nil || strings.TrimSpace(idempotencyKey) == "" || s.enqueuer == nil {
		return GenerationJob{}, ErrInvalid
	}
	if err := s.checkPrerequisites(ctx, clientID, workspaceID, campaignID, operation); err != nil {
		return GenerationJob{}, err
	}
	estimate, err := s.Estimate(ctx, clientID, workspaceID, campaignID, operation)
	if err != nil {
		return GenerationJob{}, err
	}
	keyHash := sha256.Sum256([]byte(idempotencyKey))
	var campaignVersion int32
	var conceptHash, scriptHash, characters string
	if err = s.pool.QueryRow(ctx, `SELECT c.current_version,coalesce(cc.output_hash,''),coalesce(s.script_hash,''),
		coalesce((SELECT string_agg(character_id::text || ':' || role,',' ORDER BY role) FROM campaign_characters WHERE campaign_id=c.id),'')
		FROM campaigns c LEFT JOIN campaign_concepts cc ON cc.id=c.selected_concept_id LEFT JOIN scripts s ON s.campaign_id=c.id
		WHERE c.id=$1 AND c.client_id=$2 AND c.workspace_id=$3`, campaignID, clientID, workspaceID).Scan(&campaignVersion, &conceptHash, &scriptHash, &characters); err != nil {
		return GenerationJob{}, err
	}
	inputDigest := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s:%s:%s:%s:%s:%s", campaignID, campaignVersion, operation, cfg.OpenAI.Model, studioai.PromptVersion, conceptHash, scriptHash, characters)))
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return GenerationJob{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var active bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM generation_jobs WHERE campaign_id=$1 AND operation=$2::generation_operation AND status IN('QUEUED','RUNNING'))`, campaignID, operation).Scan(&active); err != nil {
		return GenerationJob{}, err
	}
	if active {
		return GenerationJob{}, ErrConflict
	}
	id := uuid.New()
	_, err = tx.Exec(ctx, `INSERT INTO generation_jobs(id,client_id,workspace_id,campaign_id,operation,idempotency_key_hash,input_hash,estimated_cost_usd,created_by) VALUES($1,$2,$3,$4,$5::generation_operation,$6,$7,$8,$9)`, id, clientID, workspaceID, campaignID, operation, keyHash[:], hex.EncodeToString(inputDigest[:]), estimate.EstimatedCost, actorID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return s.getJobByKey(ctx, campaignID, operation, keyHash[:])
		}
		return GenerationJob{}, err
	}
	riverID, err := s.enqueuer.EnqueueAIPlanning(ctx, tx, jobs.AIPlanningArgs{GenerationJobID: id, ClientID: clientID, WorkspaceID: workspaceID, CampaignID: campaignID, Operation: operation})
	if err != nil {
		return GenerationJob{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE generation_jobs SET river_job_id=$2 WHERE id=$1`, id, riverID); err != nil {
		return GenerationJob{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return GenerationJob{}, err
	}
	return s.GetJob(ctx, clientID, workspaceID, campaignID, id)
}

func (s *Service) checkPrerequisites(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID, operation string) error {
	if operation == "CONCEPTS" {
		return nil
	}
	if operation == "CONTENT" || operation == "SCRIPT" {
		var exists bool
		err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM campaigns c JOIN campaign_concepts cc ON cc.id=c.selected_concept_id WHERE c.id=$1 AND c.client_id=$2 AND c.workspace_id=$3 AND cc.status='LOCKED')`, campaignID, clientID, workspaceID).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return ErrPrerequisite
		}
		return nil
	}
	if operation == "SCENES" {
		var scriptApproved bool
		var characterCount int
		err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM scripts s JOIN campaigns c ON c.id=s.campaign_id WHERE s.campaign_id=$1 AND s.status='APPROVED' AND c.client_id=$2 AND c.workspace_id=$3),(SELECT count(*) FROM campaign_characters WHERE campaign_id=$1)`, campaignID, clientID, workspaceID).Scan(&scriptApproved, &characterCount)
		if err != nil {
			return err
		}
		if !scriptApproved || characterCount != 2 {
			return ErrPrerequisite
		}
		return nil
	}
	return ErrInvalid
}

func (s *Service) GetJob(ctx context.Context, clientID, workspaceID, campaignID, id uuid.UUID) (GenerationJob, error) {
	return scanJob(s.pool.QueryRow(ctx, `SELECT id,campaign_id,operation::text,status::text,river_job_id,estimated_cost_usd::float8,actual_cost_usd::float8,output_summary,error_code,error_message,created_at,started_at,completed_at FROM generation_jobs WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND campaign_id=$4`, id, clientID, workspaceID, campaignID))
}

func (s *Service) ListJobs(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID) ([]GenerationJob, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,campaign_id,operation::text,status::text,river_job_id,estimated_cost_usd::float8,actual_cost_usd::float8,output_summary,error_code,error_message,created_at,started_at,completed_at FROM generation_jobs WHERE client_id=$1 AND workspace_id=$2 AND campaign_id=$3 ORDER BY created_at DESC`, clientID, workspaceID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GenerationJob{}
	for rows.Next() {
		item, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) getJobByKey(ctx context.Context, campaignID uuid.UUID, operation string, hash []byte) (GenerationJob, error) {
	return scanJob(s.pool.QueryRow(ctx, `SELECT id,campaign_id,operation::text,status::text,river_job_id,estimated_cost_usd::float8,actual_cost_usd::float8,output_summary,error_code,error_message,created_at,started_at,completed_at FROM generation_jobs WHERE campaign_id=$1 AND operation=$2::generation_operation AND idempotency_key_hash=$3`, campaignID, operation, hash))
}

type scanner interface{ Scan(...any) error }

func scanJob(row scanner) (GenerationJob, error) {
	var item GenerationJob
	var raw []byte
	err := row.Scan(&item.ID, &item.CampaignID, &item.Operation, &item.Status, &item.RiverJobID, &item.EstimatedCostUSD, &item.ActualCostUSD, &raw, &item.ErrorCode, &item.ErrorMessage, &item.CreatedAt, &item.StartedAt, &item.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return GenerationJob{}, ErrNotFound
	}
	if err == nil {
		_ = json.Unmarshal(raw, &item.OutputSummary)
	}
	return item, err
}

func tokenEstimate(operation string) (int64, int64, error) {
	switch operation {
	case "CONCEPTS":
		return 2500, 1800, nil
	case "CONTENT":
		return 4000, 2500, nil
	case "SCRIPT":
		return 3500, 2200, nil
	case "SCENES":
		return 3000, 2800, nil
	default:
		return 0, 0, ErrInvalid
	}
}
