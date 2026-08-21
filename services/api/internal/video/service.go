package video

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/jobs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/providerconfigs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/usage"
)

var (
	ErrInvalid      = errors.New("invalid video generation operation")
	ErrNotFound     = errors.New("video generation resource not found")
	ErrConflict     = errors.New("video generation conflict")
	ErrPrerequisite = errors.New("video generation prerequisite not met")
	ErrUnavailable  = errors.New("video provider unavailable")
)

type Service struct {
	pool     *pgxpool.Pool
	enqueuer *jobs.Enqueuer
	provider Provider
	config   config.Config
	resolver interface {
		Load(context.Context, uuid.UUID) (providerconfigs.Bundle, error)
	}
	now func() time.Time
}

func NewService(pool *pgxpool.Pool, enqueuer *jobs.Enqueuer, provider Provider, cfg config.Config) *Service {
	return &Service{pool: pool, enqueuer: enqueuer, provider: provider, config: cfg, now: func() time.Time { return time.Now().UTC() }}
}

func NewTenantService(pool *pgxpool.Pool, enqueuer *jobs.Enqueuer, resolver interface {
	Load(context.Context, uuid.UUID) (providerconfigs.Bundle, error)
}, cfg config.Config) *Service {
	return &Service{pool: pool, enqueuer: enqueuer, resolver: resolver, config: cfg, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) effectiveConfig(ctx context.Context, clientID uuid.UUID) (config.Config, error) {
	if s.resolver == nil {
		return s.config, nil
	}
	bundle, err := s.resolver.Load(ctx, clientID)
	if err != nil || bundle.Seedance.Model == "" {
		return config.Config{}, ErrUnavailable
	}
	return config.Config{DemoMode: bundle.DemoMode, OpenAI: bundle.OpenAI, Seedance: bundle.Seedance, R2: bundle.R2, Meta: bundle.Meta, Renderer: bundle.Renderer, WorkerTempDir: s.config.WorkerTempDir}, nil
}

type sceneGenerationInput struct {
	SceneVersion    int32
	DurationSeconds int32
	SceneHash       string
	Prompt          string
	SceneVersionID  uuid.UUID
	ProductID       uuid.UUID
}

func (s *Service) Start(ctx context.Context, clientID, workspaceID, campaignID, sceneID, actorID uuid.UUID, idempotencyKey string, input StartInput) (Generation, error) {
	cfg, configErr := s.effectiveConfig(ctx, clientID)
	if configErr != nil {
		return Generation{}, configErr
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return Generation{}, ErrInvalid
	}
	if input.Resolution == "" {
		input.Resolution = cfg.Seedance.Resolution
	}
	if input.AspectRatio == "" {
		input.AspectRatio = cfg.Seedance.AspectRatio
	}
	if !supportedFormat(input.Resolution, input.AspectRatio) {
		return Generation{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Generation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var scene sceneGenerationInput
	err = tx.QueryRow(ctx, `
		SELECT s.current_version,sv.duration_seconds,s.scene_hash,sv.seedance_prompt,sv.id,c.product_id
		FROM scenes s JOIN campaigns c ON c.id=s.campaign_id
		JOIN scene_versions sv ON sv.scene_id=s.id AND sv.version=s.current_version
		WHERE s.id=$1 AND s.campaign_id=$2 AND s.client_id=$3 AND s.workspace_id=$4
		  AND s.status='APPROVED' AND sv.generation_method='seedance'
		FOR UPDATE OF s`, sceneID, campaignID, clientID, workspaceID).Scan(&scene.SceneVersion, &scene.DurationSeconds, &scene.SceneHash, &scene.Prompt, &scene.SceneVersionID, &scene.ProductID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Generation{}, ErrPrerequisite
	}
	if err != nil {
		return Generation{}, err
	}
	if strings.TrimSpace(scene.Prompt) == "" {
		return Generation{}, ErrPrerequisite
	}
	if err = validateSceneProductReferences(ctx, tx, scene.SceneVersionID, scene.ProductID); err != nil {
		return Generation{}, err
	}
	referenceHash, err := referenceFingerprint(ctx, tx, scene.SceneVersionID)
	if err != nil {
		return Generation{}, err
	}
	promptHash := digest(scene.Prompt)
	requestHash := digest(strings.Join([]string{workspaceID.String(), campaignID.String(), sceneID.String(), fmt.Sprint(scene.SceneVersion), scene.SceneHash, promptHash, referenceHash, cfg.Seedance.Model, input.Resolution, input.AspectRatio, fmt.Sprint(scene.DurationSeconds), fmt.Sprint(input.GenerateAudio)}, "\x1f"))
	keyHash := digest(idempotencyKey + "\x1f" + requestHash)

	if existing, findErr := getReusable(ctx, tx, clientID, workspaceID, campaignID, sceneID, keyHash); findErr == nil {
		existing.Reused = true
		if err = tx.Commit(ctx); err != nil {
			return Generation{}, err
		}
		return existing, nil
	} else if !errors.Is(findErr, pgx.ErrNoRows) {
		return Generation{}, findErr
	}

	attempt := int32(1)
	if err = tx.QueryRow(ctx, `SELECT COALESCE(max(attempt_number),0)+1 FROM scene_generation_tasks WHERE idempotency_key=$1`, keyHash).Scan(&attempt); err != nil {
		return Generation{}, err
	}
	id := uuid.New()
	providerName := "byteplus-modelark"
	if cfg.DemoMode {
		providerName = "demo"
	}
	estimated := float64(scene.DurationSeconds) * cfg.Seedance.USDPerSecond
	sanitized, _ := json.Marshal(map[string]any{"sceneId": sceneID, "sceneVersion": scene.SceneVersion, "promptHash": promptHash, "referenceHash": referenceHash, "model": cfg.Seedance.Model, "resolution": input.Resolution, "ratio": input.AspectRatio, "duration": scene.DurationSeconds, "generateAudio": input.GenerateAudio})
	_, err = tx.Exec(ctx, `
		INSERT INTO scene_generation_tasks(id,client_id,workspace_id,campaign_id,scene_id,scene_version,provider,status,idempotency_key,attempt_number,model,api_version,resolution,aspect_ratio,duration_seconds,generate_audio,scene_hash,prompt_hash,reference_hash,request_hash,sanitized_request,estimated_cost_usd,timeout_at,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,'QUEUED',$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)`,
		id, clientID, workspaceID, campaignID, sceneID, scene.SceneVersion, providerName, keyHash, attempt, cfg.Seedance.Model, cfg.Seedance.APIVersion, input.Resolution, input.AspectRatio, scene.DurationSeconds, input.GenerateAudio, scene.SceneHash, promptHash, referenceHash, requestHash, sanitized, estimated, s.now().Add(cfg.Seedance.TaskTimeout), actorID)
	if err != nil {
		return Generation{}, classifyDatabaseConflict(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO scene_generation_events(generation_task_id,to_status,actor_id,source,safe_detail)VALUES($1,'QUEUED',$2,'API','Generation requested')`, id, actorID); err != nil {
		return Generation{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO scene_generation_edits(generation_task_id,updated_by)VALUES($1,$2)`, id, actorID); err != nil {
		return Generation{}, err
	}
	if _, err = s.enqueuer.EnqueueSeedanceSubmit(ctx, tx, id); err != nil {
		return Generation{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE campaigns SET status='SCENES_GENERATING',version=version+1,updated_by=$2,updated_at=now() WHERE id=$1 AND status IN ('SCRIPT_APPROVED','SCENE_REVIEW','SCENES_GENERATING')`, campaignID, actorID); err != nil {
		return Generation{}, err
	}
	item, err := getGeneration(ctx, tx, clientID, workspaceID, campaignID, sceneID, id)
	if err != nil {
		return Generation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Generation{}, err
	}
	return item, nil
}

func (s *Service) List(ctx context.Context, clientID, workspaceID, campaignID, sceneID uuid.UUID) ([]Generation, error) {
	rows, err := s.pool.Query(ctx, generationSelect+` WHERE g.client_id=$1 AND g.workspace_id=$2 AND g.campaign_id=$3 AND g.scene_id=$4 ORDER BY g.created_at DESC`, clientID, workspaceID, campaignID, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Generation{}
	for rows.Next() {
		item, scanErr := scanGeneration(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		if scanErr = s.loadDetails(ctx, &item); scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) Get(ctx context.Context, clientID, workspaceID, campaignID, sceneID, generationID uuid.UUID) (Generation, error) {
	item, err := getGeneration(ctx, s.pool, clientID, workspaceID, campaignID, sceneID, generationID)
	if err != nil {
		return Generation{}, err
	}
	if err = s.loadDetails(ctx, &item); err != nil {
		return Generation{}, err
	}
	return item, nil
}

func (s *Service) Cancel(ctx context.Context, clientID, workspaceID, campaignID, sceneID, generationID, actorID uuid.UUID) (Generation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Generation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	var providerTaskID *string
	err = tx.QueryRow(ctx, `SELECT status::text,provider_task_id FROM scene_generation_tasks WHERE id=$1 AND scene_id=$2 AND campaign_id=$3 AND client_id=$4 AND workspace_id=$5 FOR UPDATE`, generationID, sceneID, campaignID, clientID, workspaceID).Scan(&status, &providerTaskID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Generation{}, ErrNotFound
	}
	if err != nil {
		return Generation{}, err
	}
	if status == "CANCELLED" {
		item, getErr := getGeneration(ctx, tx, clientID, workspaceID, campaignID, sceneID, generationID)
		if getErr != nil {
			return Generation{}, getErr
		}
		_ = tx.Commit(ctx)
		return item, nil
	}
	if status != "QUEUED" && status != "PROVIDER_QUEUED" {
		return Generation{}, ErrConflict
	}
	if status == "PROVIDER_QUEUED" {
		if providerTaskID == nil || s.provider == nil {
			return Generation{}, ErrUnavailable
		}
		if err = s.provider.Cancel(ctx, *providerTaskID); err != nil {
			return Generation{}, ErrConflict
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE scene_generation_tasks SET status='CANCELLED',cancel_requested_at=now(),cancel_requested_by=$2,version=version+1,updated_at=now() WHERE id=$1`, generationID, actorID); err != nil {
		return Generation{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO scene_generation_events(generation_task_id,from_status,to_status,actor_id,source,safe_detail)VALUES($1,$2,'CANCELLED',$3,'API','Generation cancelled')`, generationID, status, actorID); err != nil {
		return Generation{}, err
	}
	item, err := getGeneration(ctx, tx, clientID, workspaceID, campaignID, sceneID, generationID)
	if err != nil {
		return Generation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Generation{}, err
	}
	return item, nil
}

func (s *Service) Review(ctx context.Context, clientID, workspaceID, campaignID, sceneID, generationID, actorID uuid.UUID, input ReviewInput) (Generation, error) {
	action := strings.ToUpper(strings.TrimSpace(input.Action))
	if (action != "APPROVE" && action != "REJECT") || input.Version < 1 || (action == "REJECT" && strings.TrimSpace(input.Notes) == "") {
		return Generation{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Generation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status, hash, provider, model string
	var version int64
	var duration int32
	var usageUnits int64
	var estimatedCost, actualCost float64
	err = tx.QueryRow(ctx, `SELECT status::text,request_hash,version,provider,model,duration_seconds,COALESCE(usage_tokens,0),estimated_cost_usd::float8,COALESCE(actual_cost_usd,estimated_cost_usd)::float8 FROM scene_generation_tasks WHERE id=$1 AND scene_id=$2 AND campaign_id=$3 AND client_id=$4 AND workspace_id=$5 FOR UPDATE`, generationID, sceneID, campaignID, clientID, workspaceID).Scan(&status, &hash, &version, &provider, &model, &duration, &usageUnits, &estimatedCost, &actualCost)
	if errors.Is(err, pgx.ErrNoRows) {
		return Generation{}, ErrNotFound
	}
	if err != nil {
		return Generation{}, err
	}
	if version != input.Version || status != "REVIEW_REQUIRED" {
		return Generation{}, ErrConflict
	}
	if input.CharacterCount == nil || input.DuplicateCharacter == nil || input.DuplicateProduct == nil || input.ProductColorMismatch == nil || input.BlurOrLowQualityWarning == nil || input.CropWarning == nil || input.SubtitleOverflow == nil || input.LogoOverlap == nil || input.CTASafeZoneViolation == nil {
		return Generation{}, ErrInvalid
	}
	passed := action == "APPROVE" && *input.CharacterCount == 2 && !*input.DuplicateCharacter && !*input.DuplicateProduct && !*input.ProductColorMismatch && !*input.BlurOrLowQualityWarning && !*input.CropWarning && !*input.SubtitleOverflow && !*input.LogoOverlap && !*input.CTASafeZoneViolation
	if action == "APPROVE" && !passed {
		return Generation{}, ErrPrerequisite
	}
	qcStatus := "FAILED"
	newStatus := "REJECTED"
	if passed {
		qcStatus, newStatus = "PASSED", "APPROVED"
	}
	_, err = tx.Exec(ctx, `UPDATE scene_quality_checks SET status=$2,character_count_review=$3,duplicate_character_review=$4,duplicate_product_review=$5,product_color_mismatch=$6,blur_or_low_quality_warning=$7,crop_warning=$8,subtitle_overflow=$9,logo_overlap=$10,cta_safe_zone_violation=$11,human_notes=$12,reviewed_by=$13,reviewed_at=now(),version=version+1,updated_at=now() WHERE generation_task_id=$1`, generationID, qcStatus, input.CharacterCount, input.DuplicateCharacter, input.DuplicateProduct, input.ProductColorMismatch, input.BlurOrLowQualityWarning, input.CropWarning, input.SubtitleOverflow, input.LogoOverlap, input.CTASafeZoneViolation, strings.TrimSpace(input.Notes), actorID)
	if err != nil {
		return Generation{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE scene_generation_tasks SET status=$2,review_notes=$3,reviewed_by=$4,reviewed_at=now(),version=version+1,updated_at=now() WHERE id=$1`, generationID, newStatus, strings.TrimSpace(input.Notes), actorID); err != nil {
		return Generation{}, err
	}
	approvalStatus := "REJECTED"
	if passed {
		approvalStatus = "APPROVED"
	}
	var approvalID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO approvals(client_id,workspace_id,campaign_id,entity_type,entity_id,entity_version,entity_hash,status,requested_by,decided_by,decided_at,notes)VALUES($1,$2,$3,'SCENE_GENERATION',$4,$5,$6,$7,$8,$8,now(),$9)RETURNING id`, clientID, workspaceID, campaignID, generationID, version, hash, approvalStatus, actorID, strings.TrimSpace(input.Notes)).Scan(&approvalID)
	if err != nil {
		return Generation{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO approval_events(approval_id,event_type,actor_id,entity_version,entity_hash,notes)VALUES($1,$2,$3,$4,$5,$6)`, approvalID, approvalStatus, actorID, version, hash, strings.TrimSpace(input.Notes)); err != nil {
		return Generation{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO scene_generation_events(generation_task_id,from_status,to_status,actor_id,source,safe_detail)VALUES($1,'REVIEW_REQUIRED',$2,$3,'API',$4)`, generationID, newStatus, actorID, "Human review: "+strings.ToLower(action)); err != nil {
		return Generation{}, err
	}
	acceptedSeconds := 0.0
	if passed {
		acceptedSeconds = float64(duration)
	}
	if err = usage.Record(ctx, tx, usage.Entry{Provider: provider, Model: model, RequestReference: generationID.String(), Operation: "SEEDANCE_VIDEO", ClientID: &clientID, WorkspaceID: &workspaceID, CampaignID: &campaignID, SceneID: &sceneID, InputUnits: usageUnits, GeneratedSeconds: float64(duration), AcceptedSeconds: acceptedSeconds, ProviderReportedCost: &actualCost, EstimatedCost: estimatedCost, Currency: "USD", Outcome: "SUCCESS", Category: "SEEDANCE", Metadata: map[string]any{"reviewStatus": newStatus}}); err != nil {
		return Generation{}, err
	}
	item, err := getGeneration(ctx, tx, clientID, workspaceID, campaignID, sceneID, generationID)
	if err != nil {
		return Generation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Generation{}, err
	}
	return item, nil
}

func (s *Service) Select(ctx context.Context, clientID, workspaceID, campaignID, sceneID, generationID, actorID uuid.UUID) (Generation, error) {
	result, err := s.pool.Exec(ctx, `UPDATE scenes s SET selected_generation_task_id=$1,version=s.version+1,updated_by=$2,updated_at=now() FROM scene_generation_tasks g WHERE s.id=$3 AND s.campaign_id=$4 AND s.client_id=$5 AND s.workspace_id=$6 AND g.id=$1 AND g.scene_id=s.id AND g.status='APPROVED'`, generationID, actorID, sceneID, campaignID, clientID, workspaceID)
	if err != nil {
		return Generation{}, err
	}
	if result.RowsAffected() != 1 {
		return Generation{}, ErrPrerequisite
	}
	return s.Get(ctx, clientID, workspaceID, campaignID, sceneID, generationID)
}

func (s *Service) UpdateEdit(ctx context.Context, clientID, workspaceID, campaignID, sceneID, generationID, actorID uuid.UUID, input GenerationEdit) (Generation, error) {
	if input.Version < 1 || input.TrimStartMS < 0 || (input.TrimEndMS != nil && *input.TrimEndMS <= input.TrimStartMS) || !map[string]bool{"CUT": true, "CROSSFADE": true, "FADE_TO_BLACK": true}[input.Transition] {
		return Generation{}, ErrInvalid
	}
	if !uniqueUUIDs(input.AttachedProductAssetIDs) {
		return Generation{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Generation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if input.ReplacementAssetID != nil {
		var valid bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM media_assets WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND asset_type='VIDEO' AND deleted_at IS NULL)`, *input.ReplacementAssetID, clientID, workspaceID).Scan(&valid); err != nil {
			return Generation{}, err
		}
		if !valid {
			return Generation{}, ErrInvalid
		}
	}
	if len(input.AttachedProductAssetIDs) > 0 {
		var validCount int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM media_assets a JOIN campaigns c ON c.id=$1 JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version AND v.verified_at IS NOT NULL WHERE a.id=ANY($2) AND a.client_id=$3 AND a.workspace_id=$4 AND a.product_id=c.product_id AND a.asset_type IN ('IMAGE','VIDEO','LOGO','SCREENSHOT','SCREEN_RECORDING') AND a.status='APPROVED' AND (a.expires_at IS NULL OR a.expires_at>now()) AND a.deleted_at IS NULL`, campaignID, input.AttachedProductAssetIDs, clientID, workspaceID).Scan(&validCount); err != nil {
			return Generation{}, err
		}
		if validCount != len(input.AttachedProductAssetIDs) {
			return Generation{}, ErrInvalid
		}
	}
	result, err := tx.Exec(ctx, `UPDATE scene_generation_edits e SET trim_start_ms=$2,trim_end_ms=$3,mute_audio=$4,transition=$5,replacement_asset_id=$6,attached_product_asset_ids=$7,subtitle_preview=$8,version=e.version+1,updated_by=$9,updated_at=now() FROM scene_generation_tasks g WHERE e.generation_task_id=$1 AND e.version=$10 AND g.id=e.generation_task_id AND g.scene_id=$11 AND g.campaign_id=$12 AND g.client_id=$13 AND g.workspace_id=$14 AND g.status IN ('REVIEW_REQUIRED','APPROVED')`, generationID, input.TrimStartMS, input.TrimEndMS, input.MuteAudio, input.Transition, input.ReplacementAssetID, input.AttachedProductAssetIDs, input.SubtitlePreview, actorID, input.Version, sceneID, campaignID, clientID, workspaceID)
	if err != nil {
		return Generation{}, err
	}
	if result.RowsAffected() != 1 {
		return Generation{}, ErrConflict
	}
	if err = tx.Commit(ctx); err != nil {
		return Generation{}, err
	}
	return s.Get(ctx, clientID, workspaceID, campaignID, sceneID, generationID)
}

func validateSceneProductReferences(ctx context.Context, tx pgx.Tx, sceneVersionID, productID uuid.UUID) error {
	var valid bool
	err := tx.QueryRow(ctx, `SELECT NOT EXISTS(
		SELECT 1 FROM scene_assets sa
		LEFT JOIN media_assets a ON a.id=sa.media_asset_id
		LEFT JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version
		WHERE sa.scene_version_id=$1 AND sa.role='PRODUCT_REFERENCE' AND (
			a.id IS NULL OR a.product_id IS DISTINCT FROM $2 OR a.deleted_at IS NOT NULL OR a.status<>'APPROVED'
			OR (a.expires_at IS NOT NULL AND a.expires_at<=now()) OR v.verified_at IS NULL
			OR a.asset_type NOT IN ('IMAGE','VIDEO','LOGO','SCREENSHOT','SCREEN_RECORDING')
		)
	)`, sceneVersionID, productID).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return ErrPrerequisite
	}
	return nil
}

const generationSelect = `SELECT g.id,g.scene_id,g.scene_version,g.provider,g.provider_task_id,g.status::text,g.attempt_number,g.model,g.api_version,g.resolution,g.aspect_ratio,g.duration_seconds,g.generate_audio,g.scene_hash,g.output_asset_id,g.estimated_cost_usd::float8,g.actual_cost_usd::float8,g.usage_tokens,g.error_category,g.error_code,g.error_message,g.review_notes,g.version,g.created_at,g.updated_at,COALESCE(s.selected_generation_task_id=g.id,false) FROM scene_generation_tasks g JOIN scenes s ON s.id=g.scene_id`

type rowScanner interface{ Scan(...any) error }
type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func scanGeneration(row rowScanner) (Generation, error) {
	var item Generation
	err := row.Scan(&item.ID, &item.SceneID, &item.SceneVersion, &item.Provider, &item.ProviderTaskID, &item.Status, &item.AttemptNumber, &item.Model, &item.APIVersion, &item.Resolution, &item.AspectRatio, &item.DurationSeconds, &item.GenerateAudio, &item.SceneHash, &item.OutputAssetID, &item.EstimatedCostUSD, &item.ActualCostUSD, &item.UsageTokens, &item.ErrorCategory, &item.ErrorCode, &item.ErrorMessage, &item.ReviewNotes, &item.Version, &item.CreatedAt, &item.UpdatedAt, &item.Selected)
	return item, err
}

func getGeneration(ctx context.Context, q queryer, clientID, workspaceID, campaignID, sceneID, id uuid.UUID) (Generation, error) {
	item, err := scanGeneration(q.QueryRow(ctx, generationSelect+` WHERE g.id=$1 AND g.scene_id=$2 AND g.campaign_id=$3 AND g.client_id=$4 AND g.workspace_id=$5`, id, sceneID, campaignID, clientID, workspaceID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Generation{}, ErrNotFound
	}
	return item, err
}

func getReusable(ctx context.Context, q queryer, clientID, workspaceID, campaignID, sceneID uuid.UUID, key string) (Generation, error) {
	return scanGeneration(q.QueryRow(ctx, generationSelect+` WHERE g.client_id=$1 AND g.workspace_id=$2 AND g.campaign_id=$3 AND g.scene_id=$4 AND g.idempotency_key=$5 AND g.status NOT IN ('FAILED','CANCELLED','REJECTED') ORDER BY g.attempt_number DESC LIMIT 1`, clientID, workspaceID, campaignID, sceneID, key))
}

func (s *Service) loadDetails(ctx context.Context, item *Generation) error {
	var transcript Transcription
	err := s.pool.QueryRow(ctx, `SELECT status::text,provider,model,language,transcript,segments,error_code FROM scene_transcriptions WHERE generation_task_id=$1`, item.ID).Scan(&transcript.Status, &transcript.Provider, &transcript.Model, &transcript.Language, &transcript.Transcript, &transcript.Segments, &transcript.ErrorCode)
	if err == nil {
		item.Transcription = &transcript
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	var qc QualityCheck
	err = s.pool.QueryRow(ctx, `SELECT status::text,deterministic_pass,transcript_pass,video_decodes,duration_pass,resolution_pass,audio_stream_present,silence_warning,transcript_diff,findings,character_count_review,duplicate_character_review,duplicate_product_review,product_color_mismatch,blur_or_low_quality_warning,crop_warning,subtitle_overflow,logo_overlap,cta_safe_zone_violation,human_notes,version FROM scene_quality_checks WHERE generation_task_id=$1`, item.ID).Scan(&qc.Status, &qc.DeterministicPass, &qc.TranscriptPass, &qc.VideoDecodes, &qc.DurationPass, &qc.ResolutionPass, &qc.AudioStreamPresent, &qc.SilenceWarning, &qc.TranscriptDiff, &qc.Findings, &qc.CharacterCountReview, &qc.DuplicateCharacterReview, &qc.DuplicateProductReview, &qc.ProductColorMismatch, &qc.BlurOrLowQualityWarning, &qc.CropWarning, &qc.SubtitleOverflow, &qc.LogoOverlap, &qc.CTASafeZoneViolation, &qc.HumanNotes, &qc.Version)
	if err == nil {
		item.QualityCheck = &qc
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	var edit GenerationEdit
	err = s.pool.QueryRow(ctx, `SELECT trim_start_ms,trim_end_ms,mute_audio,transition,replacement_asset_id,attached_product_asset_ids,subtitle_preview,version FROM scene_generation_edits WHERE generation_task_id=$1`, item.ID).Scan(&edit.TrimStartMS, &edit.TrimEndMS, &edit.MuteAudio, &edit.Transition, &edit.ReplacementAssetID, &edit.AttachedProductAssetIDs, &edit.SubtitlePreview, &edit.Version)
	if err == nil {
		item.Edit = &edit
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	return nil
}

func referenceFingerprint(ctx context.Context, tx pgx.Tx, sceneVersionID uuid.UUID) (string, error) {
	rows, err := tx.Query(ctx, `
		SELECT role,asset_id,checksum,version FROM (
		  SELECT sa.role,sa.media_asset_id AS asset_id,COALESCE(v.checksum_sha256,'') AS checksum,v.version
		  FROM scene_assets sa JOIN media_assets a ON a.id=sa.media_asset_id AND a.deleted_at IS NULL AND a.status='APPROVED' AND (a.expires_at IS NULL OR a.expires_at>now())
		  JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version AND v.verified_at IS NOT NULL WHERE sa.scene_version_id=$1
		  UNION ALL
		  SELECT 'CHARACTER_'||ca.purpose,ca.media_asset_id,COALESCE(v.checksum_sha256,''),v.version
		  FROM scene_versions sv JOIN character_assets ca ON ca.character_id IN (sv.speaker_character_id,sv.listener_character_id)
		  JOIN media_assets a ON a.id=ca.media_asset_id AND a.deleted_at IS NULL
		  JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version WHERE sv.id=$1
		) refs`, sceneVersionID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	values := []string{}
	for rows.Next() {
		var role, checksum string
		var assetID uuid.UUID
		var version int32
		if err = rows.Scan(&role, &assetID, &checksum, &version); err != nil {
			return "", err
		}
		values = append(values, role+":"+assetID.String()+":"+fmt.Sprint(version)+":"+checksum)
	}
	if err = rows.Err(); err != nil {
		return "", err
	}
	var speakerProviderAsset, listenerProviderAsset *string
	if err = tx.QueryRow(ctx, `SELECT speaker.provider_asset_id,listener.provider_asset_id FROM scene_versions sv JOIN characters speaker ON speaker.id=sv.speaker_character_id JOIN characters listener ON listener.id=sv.listener_character_id WHERE sv.id=$1`, sceneVersionID).Scan(&speakerProviderAsset, &listenerProviderAsset); err != nil {
		return "", err
	}
	if speakerProviderAsset != nil {
		values = append(values, "SPEAKER_PROVIDER:"+*speakerProviderAsset)
	}
	if listenerProviderAsset != nil {
		values = append(values, "LISTENER_PROVIDER:"+*listenerProviderAsset)
	}
	sort.Strings(values)
	return digest(strings.Join(values, "|")), nil
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func supportedFormat(resolution, ratio string) bool {
	return map[string]bool{"480p": true, "720p": true, "1080p": true, "4k": true}[resolution] && map[string]bool{"16:9": true, "4:3": true, "1:1": true, "3:4": true, "9:16": true, "21:9": true, "adaptive": true}[ratio]
}

func classifyDatabaseConflict(err error) error {
	if strings.Contains(err.Error(), "scene_generation_active_key_idx") || strings.Contains(err.Error(), "scene_generation_tasks_idempotency_key") {
		return ErrConflict
	}
	return err
}

func uniqueUUIDs(values []uuid.UUID) bool {
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			return false
		}
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}
