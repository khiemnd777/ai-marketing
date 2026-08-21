package campaigns

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/audit"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
)

var (
	ErrInvalid  = errors.New("invalid campaign")
	ErrNotFound = errors.New("campaign not found")
	ErrConflict = errors.New("campaign conflict")
)

type Campaign struct {
	ID                    uuid.UUID  `json:"id"`
	ClientID              uuid.UUID  `json:"clientId"`
	WorkspaceID           uuid.UUID  `json:"workspaceId"`
	BrandID               uuid.UUID  `json:"brandId"`
	ProductID             uuid.UUID  `json:"productId"`
	Name                  string     `json:"name"`
	Status                string     `json:"status"`
	CurrentVersion        int32      `json:"currentVersion"`
	SelectedConceptID     *uuid.UUID `json:"selectedConceptId"`
	Version               int64      `json:"version"`
	Objective             string     `json:"objective"`
	TargetAudience        string     `json:"targetAudience"`
	Market                string     `json:"market"`
	Country               string     `json:"country"`
	Language              string     `json:"language"`
	SocialPlatformTargets []string   `json:"socialPlatformTargets"`
	VideoFormat           string     `json:"videoFormat"`
	DurationSeconds       int32      `json:"durationSeconds"`
	AspectRatio           string     `json:"aspectRatio"`
	Tone                  string     `json:"tone"`
	Offer                 string     `json:"offer"`
	CTA                   string     `json:"cta"`
	PlannedAdsBudget      *float64   `json:"plannedAdsBudget"`
	BudgetCurrency        *string    `json:"budgetCurrency"`
	StartsOn              *time.Time `json:"startsOn"`
	EndsOn                *time.Time `json:"endsOn"`
	ChangeSummary         string     `json:"changeSummary"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

type Input struct {
	BrandID               uuid.UUID  `json:"brandId"`
	ProductID             uuid.UUID  `json:"productId"`
	Name                  string     `json:"name"`
	Objective             string     `json:"objective"`
	TargetAudience        string     `json:"targetAudience"`
	Market                string     `json:"market"`
	Country               string     `json:"country"`
	Language              string     `json:"language"`
	SocialPlatformTargets []string   `json:"socialPlatformTargets"`
	VideoFormat           string     `json:"videoFormat"`
	DurationSeconds       int32      `json:"durationSeconds"`
	AspectRatio           string     `json:"aspectRatio"`
	Tone                  string     `json:"tone"`
	Offer                 string     `json:"offer"`
	CTA                   string     `json:"cta"`
	PlannedAdsBudget      *float64   `json:"plannedAdsBudget"`
	BudgetCurrency        *string    `json:"budgetCurrency"`
	StartsOn              *time.Time `json:"startsOn"`
	EndsOn                *time.Time `json:"endsOn"`
	ChangeSummary         string     `json:"changeSummary"`
	Version               int64      `json:"version"`
}

type ProgressStep struct {
	Key       string `json:"key"`
	Completed bool   `json:"completed"`
	Optional  bool   `json:"optional"`
}

type Progress struct {
	CampaignID uuid.UUID      `json:"campaignId"`
	Steps      []ProgressStep `json:"steps"`
}

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

const columns = `c.id,c.client_id,c.workspace_id,c.brand_id,c.product_id,c.name,c.status::text,c.current_version,c.selected_concept_id,c.version,v.objective::text,v.target_audience,v.market,v.country,v.language,v.social_platform_targets,v.video_format,v.duration_seconds,v.aspect_ratio,v.tone,v.offer,v.cta,v.planned_ads_budget::float8,v.budget_currency,v.starts_on,v.ends_on,v.change_summary,c.created_at,c.updated_at`

func (s *Service) List(ctx context.Context, clientID, workspaceID uuid.UUID, search, status string) ([]Campaign, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+columns+` FROM campaigns c JOIN campaign_versions v ON v.campaign_id=c.id AND v.version=c.current_version WHERE c.client_id=$1 AND c.workspace_id=$2 AND ($3='' OR c.status::text=$3) AND ($4='' OR c.name ILIKE '%'||$4||'%') ORDER BY c.updated_at DESC,c.id`, clientID, workspaceID, strings.ToUpper(strings.TrimSpace(status)), strings.TrimSpace(search))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Campaign{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) Get(ctx context.Context, clientID, workspaceID, id uuid.UUID) (Campaign, error) {
	return scan(s.pool.QueryRow(ctx, `SELECT `+columns+` FROM campaigns c JOIN campaign_versions v ON v.campaign_id=c.id AND v.version=c.current_version WHERE c.id=$1 AND c.client_id=$2 AND c.workspace_id=$3`, id, clientID, workspaceID))
}

func (s *Service) GetProgress(ctx context.Context, clientID, workspaceID, id uuid.UUID) (Progress, error) {
	return getProgress(ctx, s.pool, clientID, workspaceID, id)
}

type progressQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func getProgress(ctx context.Context, queryer progressQueryer, clientID, workspaceID, id uuid.UUID) (Progress, error) {
	completed := make([]bool, 9)
	err := queryer.QueryRow(ctx, `SELECT
		(SELECT count(*)=2
		 FROM campaign_characters selection
		 JOIN characters character ON character.id=selection.character_id
		 WHERE selection.campaign_id=c.id
		   AND character.status='ACTIVE'
		   AND character.consent_status IN ('NOT_REQUIRED','APPROVED')
		   AND (character.client_id IS NULL OR (character.client_id=c.client_id AND character.workspace_id=c.workspace_id))),
		EXISTS(
			SELECT 1 FROM campaign_concepts concept
			WHERE concept.id=c.selected_concept_id AND concept.campaign_id=c.id AND concept.status='LOCKED'
			  AND EXISTS(SELECT 1 FROM approvals approval WHERE approval.client_id=c.client_id AND approval.workspace_id=c.workspace_id AND approval.campaign_id=c.id AND approval.entity_type='CONCEPT' AND approval.entity_id=concept.id AND approval.entity_hash=concept.output_hash AND approval.status='APPROVED' AND approval.invalidated_at IS NULL)
		),
		(SELECT count(*)=14 FROM campaign_content_variants content WHERE content.campaign_id=c.id AND content.client_id=c.client_id AND content.workspace_id=c.workspace_id)
		AND NOT EXISTS(
			SELECT 1 FROM campaign_content_variants content
			WHERE content.campaign_id=c.id AND content.client_id=c.client_id AND content.workspace_id=c.workspace_id
			  AND (content.status<>'APPROVED' OR NOT EXISTS(SELECT 1 FROM approvals approval WHERE approval.client_id=c.client_id AND approval.workspace_id=c.workspace_id AND approval.campaign_id=c.id AND approval.entity_type='CONTENT_VARIANT' AND approval.entity_id=content.id AND approval.entity_version=content.version AND approval.entity_hash=content.content_hash AND approval.status='APPROVED' AND approval.invalidated_at IS NULL))
		),
		EXISTS(
			SELECT 1 FROM scripts script
			WHERE script.campaign_id=c.id AND script.client_id=c.client_id AND script.workspace_id=c.workspace_id AND script.status='APPROVED'
			  AND EXISTS(SELECT 1 FROM approvals approval WHERE approval.client_id=c.client_id AND approval.workspace_id=c.workspace_id AND approval.campaign_id=c.id AND approval.entity_type='SCRIPT' AND approval.entity_id=script.id AND approval.entity_version=script.version AND approval.entity_hash=script.script_hash AND approval.status='APPROVED' AND approval.invalidated_at IS NULL)
		),
		EXISTS(SELECT 1 FROM scenes scene WHERE scene.campaign_id=c.id AND scene.client_id=c.client_id AND scene.workspace_id=c.workspace_id)
		AND NOT EXISTS(
			SELECT 1 FROM scenes scene
			WHERE scene.campaign_id=c.id AND scene.client_id=c.client_id AND scene.workspace_id=c.workspace_id
			  AND (scene.status<>'APPROVED' OR NOT EXISTS(SELECT 1 FROM approvals approval WHERE approval.client_id=c.client_id AND approval.workspace_id=c.workspace_id AND approval.campaign_id=c.id AND approval.entity_type='SCENE' AND approval.entity_id=scene.id AND approval.entity_version=scene.version AND approval.entity_hash=scene.scene_hash AND approval.status='APPROVED' AND approval.invalidated_at IS NULL))
		),
		EXISTS(SELECT 1 FROM scenes scene WHERE scene.campaign_id=c.id AND scene.client_id=c.client_id AND scene.workspace_id=c.workspace_id)
		AND NOT EXISTS(
			SELECT 1 FROM scenes scene
			LEFT JOIN scene_generation_tasks generation ON generation.id=scene.selected_generation_task_id AND generation.scene_id=scene.id AND generation.campaign_id=c.id AND generation.client_id=c.client_id AND generation.workspace_id=c.workspace_id
			WHERE scene.campaign_id=c.id AND scene.client_id=c.client_id AND scene.workspace_id=c.workspace_id
			  AND (generation.id IS NULL OR generation.scene_version<>scene.current_version OR generation.status<>'APPROVED' OR NOT EXISTS(SELECT 1 FROM approvals approval WHERE approval.client_id=c.client_id AND approval.workspace_id=c.workspace_id AND approval.campaign_id=c.id AND approval.entity_type='SCENE_GENERATION' AND approval.entity_id=generation.id AND approval.entity_hash=generation.request_hash AND approval.status='APPROVED' AND approval.invalidated_at IS NULL))
		),
		EXISTS(
			SELECT 1 FROM video_projects project
			JOIN render_jobs render ON render.id=project.selected_render_job_id AND render.video_project_id=project.id
			WHERE project.campaign_id=c.id AND project.client_id=c.client_id AND project.workspace_id=c.workspace_id
			  AND render.campaign_id=c.id AND render.client_id=c.client_id AND render.workspace_id=c.workspace_id
			  AND render.video_project_version=project.current_version AND render.status='APPROVED'
			  AND EXISTS(SELECT 1 FROM approvals approval WHERE approval.client_id=c.client_id AND approval.workspace_id=c.workspace_id AND approval.campaign_id=c.id AND approval.entity_type='FINAL_RENDER' AND approval.entity_id=render.id AND approval.entity_hash=render.output_hash AND approval.status='APPROVED' AND approval.invalidated_at IS NULL)
		),
		EXISTS(SELECT 1 FROM social_posts post WHERE post.campaign_id=c.id AND post.client_id=c.client_id AND post.workspace_id=c.workspace_id AND post.status='PUBLISHED' AND post.provider_post_id IS NOT NULL),
		EXISTS(SELECT 1 FROM ad_campaigns ad WHERE ad.campaign_id=c.id AND ad.client_id=c.client_id AND ad.workspace_id=c.workspace_id AND ad.status IN ('PAUSED','ACTIVE','ARCHIVED') AND ad.provider_campaign_id IS NOT NULL)
	FROM campaigns c
	WHERE c.id=$1 AND c.client_id=$2 AND c.workspace_id=$3`, id, clientID, workspaceID).Scan(
		&completed[0], &completed[1], &completed[2], &completed[3], &completed[4], &completed[5], &completed[6], &completed[7], &completed[8],
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Progress{}, ErrNotFound
	}
	if err != nil {
		return Progress{}, err
	}
	return newProgress(id, completed), nil
}

func newProgress(campaignID uuid.UUID, completed []bool) Progress {
	keys := [...]string{"BRIEF", "CONCEPT", "CONTENT", "SCRIPT", "SCENES", "QUALITY", "COMPOSER", "PUBLISHING", "ADS"}
	steps := make([]ProgressStep, len(keys))
	for index, key := range keys {
		steps[index] = ProgressStep{Key: key, Completed: index < len(completed) && completed[index], Optional: key == "ADS"}
	}
	return Progress{CampaignID: campaignID, Steps: steps}
}

func (s *Service) Create(ctx context.Context, clientID, workspaceID uuid.UUID, input Input, actor auth.Principal, metadata auth.ClientMetadata) (Campaign, error) {
	if err := validate(&input, false); err != nil {
		return Campaign{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Campaign{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id := uuid.New()
	tag, err := tx.Exec(ctx, `INSERT INTO campaigns(id,client_id,workspace_id,brand_id,product_id,name,created_by,updated_by) SELECT $1,$2,$3,$4,$5,$6,$7,$7 WHERE EXISTS(SELECT 1 FROM brands WHERE id=$4 AND client_id=$2 AND workspace_id=$3 AND status='ACTIVE') AND EXISTS(SELECT 1 FROM products WHERE id=$5 AND client_id=$2 AND workspace_id=$3 AND status<>'ARCHIVED')`, id, clientID, workspaceID, input.BrandID, input.ProductID, input.Name, actor.UserID)
	if err != nil {
		return Campaign{}, mapDB(err)
	}
	if tag.RowsAffected() != 1 {
		return Campaign{}, ErrNotFound
	}
	if err = insertVersion(ctx, tx, id, clientID, workspaceID, 1, input, actor.UserID); err != nil {
		return Campaign{}, err
	}
	item, err := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM campaigns c JOIN campaign_versions v ON v.campaign_id=c.id AND v.version=c.current_version WHERE c.id=$1`, id))
	if err != nil {
		return Campaign{}, err
	}
	if err = audit.Record(ctx, db.New(tx), audit.Event{ActorID: validUUID(actor.UserID), Action: "campaign.created", EntityType: "campaign", EntityID: validUUID(id), ClientID: validUUID(clientID), WorkspaceID: validUUID(workspaceID), RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS", After: item}); err != nil {
		return Campaign{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Campaign{}, err
	}
	return item, nil
}

func (s *Service) Update(ctx context.Context, clientID, workspaceID, id uuid.UUID, input Input, actor auth.Principal, metadata auth.ClientMetadata) (Campaign, error) {
	if err := validate(&input, true); err != nil {
		return Campaign{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Campaign{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM campaigns c JOIN campaign_versions v ON v.campaign_id=c.id AND v.version=c.current_version WHERE c.id=$1 AND c.client_id=$2 AND c.workspace_id=$3 FOR UPDATE OF c`, id, clientID, workspaceID))
	if err != nil {
		return Campaign{}, err
	}
	if before.Version != input.Version || before.Status == "ARCHIVED" {
		return Campaign{}, ErrConflict
	}
	var next int32
	err = tx.QueryRow(ctx, `UPDATE campaigns SET brand_id=$4,product_id=$5,name=$6,status='DRAFT',selected_concept_id=NULL,current_version=current_version+1,version=version+1,updated_by=$7,updated_at=now() WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND EXISTS(SELECT 1 FROM brands WHERE id=$4 AND client_id=$2 AND workspace_id=$3 AND status='ACTIVE') AND EXISTS(SELECT 1 FROM products WHERE id=$5 AND client_id=$2 AND workspace_id=$3 AND status<>'ARCHIVED') RETURNING current_version`, id, clientID, workspaceID, input.BrandID, input.ProductID, input.Name, actor.UserID).Scan(&next)
	if err != nil {
		return Campaign{}, mapDB(err)
	}
	if err = insertVersion(ctx, tx, id, clientID, workspaceID, next, input, actor.UserID); err != nil {
		return Campaign{}, err
	}
	if err = invalidatePlanning(ctx, tx, id, actor.UserID, "Campaign brief changed"); err != nil {
		return Campaign{}, err
	}
	after, err := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM campaigns c JOIN campaign_versions v ON v.campaign_id=c.id AND v.version=c.current_version WHERE c.id=$1`, id))
	if err != nil {
		return Campaign{}, err
	}
	if err = audit.Record(ctx, db.New(tx), audit.Event{ActorID: validUUID(actor.UserID), Action: "campaign.updated", EntityType: "campaign", EntityID: validUUID(id), ClientID: validUUID(clientID), WorkspaceID: validUUID(workspaceID), RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS", Before: before, After: after}); err != nil {
		return Campaign{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Campaign{}, err
	}
	return after, nil
}

func (s *Service) Duplicate(ctx context.Context, clientID, workspaceID, id uuid.UUID, name string, actor auth.Principal, metadata auth.ClientMetadata) (Campaign, error) {
	source, err := s.Get(ctx, clientID, workspaceID, id)
	if err != nil {
		return Campaign{}, err
	}
	return s.Create(ctx, clientID, workspaceID, Input{BrandID: source.BrandID, ProductID: source.ProductID, Name: strings.TrimSpace(name), Objective: source.Objective, TargetAudience: source.TargetAudience, Market: source.Market, Country: source.Country, Language: source.Language, SocialPlatformTargets: source.SocialPlatformTargets, VideoFormat: source.VideoFormat, DurationSeconds: source.DurationSeconds, AspectRatio: source.AspectRatio, Tone: source.Tone, Offer: source.Offer, CTA: source.CTA, PlannedAdsBudget: source.PlannedAdsBudget, BudgetCurrency: source.BudgetCurrency, StartsOn: source.StartsOn, EndsOn: source.EndsOn, ChangeSummary: "Duplicated from " + source.Name}, actor, metadata)
}

func insertVersion(ctx context.Context, tx pgx.Tx, id, clientID, workspaceID uuid.UUID, version int32, input Input, actorID uuid.UUID) error {
	_, err := tx.Exec(ctx, `INSERT INTO campaign_versions(campaign_id,client_id,workspace_id,version,objective,target_audience,market,country,language,social_platform_targets,video_format,duration_seconds,aspect_ratio,tone,offer,cta,planned_ads_budget,budget_currency,starts_on,ends_on,change_summary,created_by) VALUES($1,$2,$3,$4,$5::campaign_objective,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`, id, clientID, workspaceID, version, input.Objective, input.TargetAudience, input.Market, input.Country, input.Language, input.SocialPlatformTargets, input.VideoFormat, input.DurationSeconds, input.AspectRatio, input.Tone, input.Offer, input.CTA, input.PlannedAdsBudget, input.BudgetCurrency, input.StartsOn, input.EndsOn, input.ChangeSummary, actorID)
	return err
}

func invalidatePlanning(ctx context.Context, tx pgx.Tx, campaignID, actorID uuid.UUID, reason string) error {
	if _, err := tx.Exec(ctx, `WITH changed AS (
		UPDATE approvals SET invalidated_at=now(),invalidation_reason=$2
		WHERE campaign_id=$1 AND invalidated_at IS NULL RETURNING id,entity_version,entity_hash
	) INSERT INTO approval_events(approval_id,event_type,entity_version,entity_hash,notes)
	SELECT id,'INVALIDATED',entity_version,entity_hash,$2 FROM changed`, campaignID, reason); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE campaign_concepts SET status='DRAFT',locked_at=NULL,locked_by=NULL,version=version+1,updated_by=$2,updated_at=now() WHERE campaign_id=$1 AND status IN('APPROVED','LOCKED')`, campaignID, actorID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE campaign_content_variants SET status='DRAFT',approved_at=NULL,approved_by=NULL,version=version+1,updated_by=$2,updated_at=now() WHERE campaign_id=$1 AND status='APPROVED'`, campaignID, actorID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE scripts SET status='DRAFT',approved_at=NULL,approved_by=NULL,version=version+1,updated_by=$2,updated_at=now() WHERE campaign_id=$1 AND status='APPROVED'`, campaignID, actorID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE scenes SET status='DRAFT',approved_at=NULL,approved_by=NULL,version=version+1,updated_by=$2,updated_at=now() WHERE campaign_id=$1 AND status='APPROVED'`, campaignID, actorID)
	return err
}

func validate(input *Input, requireVersion bool) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Objective = strings.ToUpper(strings.TrimSpace(input.Objective))
	input.TargetAudience = strings.TrimSpace(input.TargetAudience)
	input.Market = strings.TrimSpace(input.Market)
	input.Country = strings.ToUpper(strings.TrimSpace(input.Country))
	input.Language = strings.ToLower(strings.TrimSpace(input.Language))
	input.VideoFormat = strings.ToUpper(strings.TrimSpace(input.VideoFormat))
	input.AspectRatio = strings.TrimSpace(input.AspectRatio)
	input.Tone = strings.TrimSpace(input.Tone)
	input.CTA = strings.TrimSpace(input.CTA)
	if len(input.Name) < 2 || len(input.Name) > 200 || input.BrandID == uuid.Nil || input.ProductID == uuid.Nil || input.TargetAudience == "" || input.Market == "" || len(input.Country) != 2 || (input.Language != "vi" && input.Language != "en") || (input.VideoFormat != "INTERVIEW_REVIEW" && input.VideoFormat != "PROBLEM_SOLUTION") || (input.DurationSeconds != 30 && input.DurationSeconds != 45) || input.AspectRatio != "9:16" || input.Tone == "" || input.CTA == "" || (requireVersion && input.Version < 1) {
		return ErrInvalid
	}
	objectives := map[string]bool{"PRODUCT_INTRODUCTION": true, "AWARENESS": true, "ENGAGEMENT": true, "WEBSITE_TRAFFIC": true, "LEAD_GENERATION": true, "SALES": true, "PROMOTION": true}
	if !objectives[input.Objective] || len(input.SocialPlatformTargets) == 0 || len(input.SocialPlatformTargets) > 3 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for index, platform := range input.SocialPlatformTargets {
		platform = strings.ToUpper(strings.TrimSpace(platform))
		if !map[string]bool{"FACEBOOK": true, "INSTAGRAM": true, "REELS": true}[platform] || seen[platform] {
			return ErrInvalid
		}
		seen[platform] = true
		input.SocialPlatformTargets[index] = platform
	}
	if input.PlannedAdsBudget != nil {
		if *input.PlannedAdsBudget < 0 || input.BudgetCurrency == nil || len(strings.TrimSpace(*input.BudgetCurrency)) != 3 {
			return ErrInvalid
		}
		currency := strings.ToUpper(strings.TrimSpace(*input.BudgetCurrency))
		input.BudgetCurrency = &currency
	} else if input.BudgetCurrency != nil {
		return ErrInvalid
	}
	if input.StartsOn != nil && input.EndsOn != nil && input.EndsOn.Before(*input.StartsOn) {
		return ErrInvalid
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (Campaign, error) {
	var item Campaign
	err := row.Scan(&item.ID, &item.ClientID, &item.WorkspaceID, &item.BrandID, &item.ProductID, &item.Name, &item.Status, &item.CurrentVersion, &item.SelectedConceptID, &item.Version, &item.Objective, &item.TargetAudience, &item.Market, &item.Country, &item.Language, &item.SocialPlatformTargets, &item.VideoFormat, &item.DurationSeconds, &item.AspectRatio, &item.Tone, &item.Offer, &item.CTA, &item.PlannedAdsBudget, &item.BudgetCurrency, &item.StartsOn, &item.EndsOn, &item.ChangeSummary, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, ErrNotFound
	}
	return item, err
}

func mapDB(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return fmt.Errorf("campaign database operation: %w", err)
}

func validUUID(value uuid.UUID) uuid.NullUUID { return uuid.NullUUID{UUID: value, Valid: true} }
