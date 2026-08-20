package metaads

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/audit"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/jobs"
)

var (
	ErrInvalid      = errors.New("invalid Meta Ads operation")
	ErrNotFound     = errors.New("Meta Ads resource not found")
	ErrConflict     = errors.New("Meta Ads conflict")
	ErrGuardrail    = errors.New("Meta Ads guardrail blocked the operation")
	ErrPrerequisite = errors.New("Meta Ads prerequisite not met")
)

type Guardrails struct {
	WorkspaceID                  uuid.UUID `json:"workspaceId"`
	WorkspaceSpendCapMinor       int64     `json:"workspaceSpendCapMinor"`
	DefaultCampaignSpendCapMinor int64     `json:"defaultCampaignSpendCapMinor"`
	MaximumBudgetIncreasePercent float64   `json:"maximumBudgetIncreasePercent"`
	Currency                     string    `json:"currency"`
	Version                      int64     `json:"version"`
}
type Audience struct {
	Countries              []string `json:"countries"`
	AgeMin                 int      `json:"ageMin"`
	AgeMax                 int      `json:"ageMax"`
	Genders                []string `json:"genders"`
	Interests              []string `json:"interests"`
	CustomAudienceIDs      []string `json:"customAudienceIds"`
	RetargetingAudienceIDs []string `json:"retargetingAudienceIds"`
}
type CreativeInput struct {
	MediaAssetID        uuid.UUID  `json:"mediaAssetId"`
	ThumbnailAssetID    *uuid.UUID `json:"thumbnailAssetId"`
	PrimaryTextVariants []string   `json:"primaryTextVariants"`
	HeadlineVariants    []string   `json:"headlineVariants"`
	CTAVariants         []string   `json:"ctaVariants"`
}
type CampaignInput struct {
	MetaAdAccountID       uuid.UUID         `json:"metaAdAccountId"`
	SocialAccountID       uuid.UUID         `json:"socialAccountId"`
	MetaPixelID           *uuid.UUID        `json:"metaPixelId"`
	Name                  string            `json:"name"`
	Objective             string            `json:"objective"`
	DailyBudgetMinor      *int64            `json:"dailyBudgetMinor"`
	LifetimeBudgetMinor   *int64            `json:"lifetimeBudgetMinor"`
	CampaignSpendCapMinor int64             `json:"campaignSpendCapMinor"`
	Currency              string            `json:"currency"`
	StartsAt              *time.Time        `json:"startsAt"`
	EndsAt                *time.Time        `json:"endsAt"`
	Audience              Audience          `json:"audience"`
	Placements            []string          `json:"placements"`
	DestinationURL        string            `json:"destinationUrl"`
	UTMParameters         map[string]string `json:"utmParameters"`
	ConversionEvent       *string           `json:"conversionEvent"`
	Creative              CreativeInput     `json:"creative"`
	Version               int64             `json:"version"`
}
type Campaign struct {
	ID                    uuid.UUID         `json:"id"`
	CampaignID            uuid.UUID         `json:"campaignId"`
	MetaAdAccountID       uuid.UUID         `json:"metaAdAccountId"`
	SocialAccountID       uuid.UUID         `json:"socialAccountId"`
	MetaPixelID           *uuid.UUID        `json:"metaPixelId"`
	Name                  string            `json:"name"`
	Objective             string            `json:"objective"`
	Currency              string            `json:"currency"`
	DestinationURL        string            `json:"destinationUrl"`
	Status                string            `json:"status"`
	CampaignHash          string            `json:"campaignHash"`
	DailyBudgetMinor      *int64            `json:"dailyBudgetMinor"`
	LifetimeBudgetMinor   *int64            `json:"lifetimeBudgetMinor"`
	CampaignSpendCapMinor int64             `json:"campaignSpendCapMinor"`
	StartsAt              *time.Time        `json:"startsAt"`
	EndsAt                *time.Time        `json:"endsAt"`
	Audience              Audience          `json:"audience"`
	Placements            []string          `json:"placements"`
	UTMParameters         map[string]string `json:"utmParameters"`
	ConversionEvent       *string           `json:"conversionEvent"`
	ProviderCampaignID    *string           `json:"providerCampaignId"`
	LastErrorCode         *string           `json:"lastErrorCode"`
	LastErrorMessage      *string           `json:"lastErrorMessage"`
	Version               int64             `json:"version"`
	CreatedAt             time.Time         `json:"createdAt"`
	UpdatedAt             time.Time         `json:"updatedAt"`
	Creative              CreativeInput     `json:"creative"`
}
type ReviewInput struct {
	Action               string `json:"action"`
	Version              int64  `json:"version"`
	Notes                string `json:"notes"`
	ConfirmedBudgetMinor int64  `json:"confirmedBudgetMinor"`
	ConfirmationText     string `json:"confirmationText"`
}
type ActionInput struct {
	Action               string `json:"action"`
	RequestedBudgetMinor *int64 `json:"requestedBudgetMinor"`
	ConfirmationText     string `json:"confirmationText"`
	IdempotencyKey       string `json:"-"`
}
type Action struct {
	ID                   uuid.UUID `json:"id"`
	AdCampaignID         uuid.UUID `json:"adCampaignId"`
	Action               string    `json:"action"`
	Status               string    `json:"status"`
	RequestedBudgetMinor *int64    `json:"requestedBudgetMinor"`
	PreviousBudgetMinor  *int64    `json:"previousBudgetMinor"`
	ConfirmationText     string    `json:"confirmationText"`
	ErrorCode            *string   `json:"errorCode"`
	ErrorMessage         *string   `json:"errorMessage"`
	Version              int64     `json:"version"`
	RequestedAt          time.Time `json:"requestedAt"`
}
type Service struct {
	pool     *pgxpool.Pool
	enqueuer *jobs.Enqueuer
}

func NewService(pool *pgxpool.Pool, enqueuer *jobs.Enqueuer) *Service {
	return &Service{pool: pool, enqueuer: enqueuer}
}

func (s *Service) SaveGuardrails(ctx context.Context, clientID, workspaceID, actorID uuid.UUID, input Guardrails) (Guardrails, error) {
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if input.WorkspaceSpendCapMinor <= 0 || input.DefaultCampaignSpendCapMinor <= 0 || input.DefaultCampaignSpendCapMinor > input.WorkspaceSpendCapMinor || input.MaximumBudgetIncreasePercent < 0 || input.MaximumBudgetIncreasePercent > 100 || len(input.Currency) != 3 {
		return Guardrails{}, ErrInvalid
	}
	result, err := s.pool.Exec(ctx, `INSERT INTO meta_ad_guardrails(workspace_id,client_id,workspace_spend_cap_minor,default_campaign_spend_cap_minor,maximum_budget_increase_percent,currency,updated_by)VALUES($1,$2,$3,$4,$5,$6,$7)ON CONFLICT(workspace_id)DO UPDATE SET workspace_spend_cap_minor=EXCLUDED.workspace_spend_cap_minor,default_campaign_spend_cap_minor=EXCLUDED.default_campaign_spend_cap_minor,maximum_budget_increase_percent=EXCLUDED.maximum_budget_increase_percent,currency=EXCLUDED.currency,version=meta_ad_guardrails.version+1,updated_by=EXCLUDED.updated_by,updated_at=now() WHERE meta_ad_guardrails.client_id=EXCLUDED.client_id AND meta_ad_guardrails.version=$8`, workspaceID, clientID, input.WorkspaceSpendCapMinor, input.DefaultCampaignSpendCapMinor, input.MaximumBudgetIncreasePercent, input.Currency, actorID, input.Version)
	if err != nil {
		return Guardrails{}, err
	}
	if result.RowsAffected() != 1 {
		return Guardrails{}, ErrConflict
	}
	return s.GetGuardrails(ctx, clientID, workspaceID)
}
func (s *Service) GetGuardrails(ctx context.Context, clientID, workspaceID uuid.UUID) (Guardrails, error) {
	var g Guardrails
	err := s.pool.QueryRow(ctx, `SELECT workspace_id,workspace_spend_cap_minor,default_campaign_spend_cap_minor,maximum_budget_increase_percent::float8,currency,version FROM meta_ad_guardrails WHERE workspace_id=$1 AND client_id=$2`, workspaceID, clientID).Scan(&g.WorkspaceID, &g.WorkspaceSpendCapMinor, &g.DefaultCampaignSpendCapMinor, &g.MaximumBudgetIncreasePercent, &g.Currency, &g.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return g, ErrNotFound
	}
	return g, err
}

func (s *Service) Create(ctx context.Context, clientID, workspaceID, campaignID, actorID uuid.UUID, key string, input CampaignInput) (Campaign, error) {
	key = strings.TrimSpace(key)
	input.Name = strings.TrimSpace(input.Name)
	input.Objective = strings.TrimSpace(input.Objective)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if len(key) < 16 || input.Name == "" || input.Objective == "" || input.MetaAdAccountID == uuid.Nil || input.SocialAccountID == uuid.Nil || input.CampaignSpendCapMinor <= 0 || ((input.DailyBudgetMinor == nil) == (input.LifetimeBudgetMinor == nil)) || len(input.Creative.PrimaryTextVariants) == 0 || len(input.Creative.HeadlineVariants) == 0 || len(input.Creative.CTAVariants) == 0 {
		return Campaign{}, ErrInvalid
	}
	destination, err := url.Parse(input.DestinationURL)
	if err != nil || destination.Scheme != "https" || destination.Host == "" {
		return Campaign{}, ErrInvalid
	}
	budget := value64(input.DailyBudgetMinor)
	if budget == 0 {
		budget = value64(input.LifetimeBudgetMinor)
	}
	if budget <= 0 || budget > input.CampaignSpendCapMinor {
		return Campaign{}, ErrGuardrail
	}
	if input.EndsAt != nil && input.StartsAt != nil && !input.EndsAt.After(*input.StartsAt) {
		return Campaign{}, ErrInvalid
	}
	if input.Audience.AgeMin < 18 || input.Audience.AgeMax > 65 || input.Audience.AgeMax < input.Audience.AgeMin || len(input.Audience.Countries) == 0 {
		return Campaign{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Campaign{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if existing, findErr := getByKey(ctx, tx, clientID, workspaceID, campaignID, key); findErr == nil {
		_ = tx.Commit(ctx)
		return existing, nil
	} else if !errors.Is(findErr, pgx.ErrNoRows) {
		return Campaign{}, findErr
	}
	var workspaceCap, defaultCap, allocatedCap int64
	var currency string
	var accountValid, creativeValid, campaignApproved, pixelValid bool
	err = tx.QueryRow(ctx, `SELECT g.workspace_spend_cap_minor,g.default_campaign_spend_cap_minor,g.currency,COALESCE((SELECT sum(a.campaign_spend_cap_minor) FROM ad_campaigns a WHERE a.workspace_id=$2 AND a.status NOT IN ('ARCHIVED','FAILED')),0),EXISTS(SELECT 1 FROM meta_ad_accounts a JOIN meta_connections c ON c.id=a.connection_id WHERE a.id=$4 AND a.client_id=$1 AND a.workspace_id=$2 AND a.currency=g.currency AND c.status IN ('CONNECTED','EXPIRING') AND (c.token_expires_at IS NULL OR c.token_expires_at>now()) AND (c.data_access_expires_at IS NULL OR c.data_access_expires_at>now())),EXISTS(SELECT 1 FROM media_assets m WHERE m.id=$6 AND m.client_id=$1 AND m.workspace_id=$2 AND m.deleted_at IS NULL),EXISTS(SELECT 1 FROM campaigns c JOIN video_projects p ON p.campaign_id=c.id JOIN render_jobs r ON r.id=p.selected_render_job_id AND r.status='APPROVED' WHERE c.id=$3 AND c.client_id=$1 AND c.workspace_id=$2 AND c.status='APPROVED' AND EXISTS(SELECT 1 FROM social_accounts s WHERE s.id=$5 AND s.client_id=$1 AND s.workspace_id=$2 AND s.status IN ('CONNECTED','EXPIRING'))),($7::uuid IS NULL OR EXISTS(SELECT 1 FROM meta_pixels px WHERE px.id=$7 AND px.meta_ad_account_id=$4)) FROM meta_ad_guardrails g WHERE g.client_id=$1 AND g.workspace_id=$2`, clientID, workspaceID, campaignID, input.MetaAdAccountID, input.SocialAccountID, input.Creative.MediaAssetID, input.MetaPixelID).Scan(&workspaceCap, &defaultCap, &currency, &allocatedCap, &accountValid, &creativeValid, &campaignApproved, &pixelValid)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, ErrPrerequisite
	} else if err != nil {
		return Campaign{}, err
	}
	if input.Currency != currency || allocatedCap+input.CampaignSpendCapMinor > workspaceCap || input.CampaignSpendCapMinor > defaultCap || !accountValid || !creativeValid || !campaignApproved || !pixelValid {
		return Campaign{}, ErrGuardrail
	}
	audienceRaw, _ := json.Marshal(input.Audience)
	utmRaw, _ := json.Marshal(input.UTMParameters)
	hash := campaignHash(input)
	id := uuid.New()
	_, err = tx.Exec(ctx, `INSERT INTO ad_campaigns(id,client_id,workspace_id,campaign_id,meta_ad_account_id,social_account_id,meta_pixel_id,name,objective,daily_budget_minor,lifetime_budget_minor,campaign_spend_cap_minor,currency,starts_at,ends_at,audience,placements,destination_url,utm_parameters,conversion_event,status,campaign_hash,created_by,updated_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,'APPROVAL_REQUIRED',$21,$22,$22)`, id, clientID, workspaceID, campaignID, input.MetaAdAccountID, input.SocialAccountID, input.MetaPixelID, input.Name, input.Objective, input.DailyBudgetMinor, input.LifetimeBudgetMinor, input.CampaignSpendCapMinor, input.Currency, input.StartsAt, input.EndsAt, audienceRaw, input.Placements, input.DestinationURL, utmRaw, input.ConversionEvent, hash, actorID)
	if err != nil {
		return Campaign{}, err
	}
	texts, _ := json.Marshal(input.Creative.PrimaryTextVariants)
	headlines, _ := json.Marshal(input.Creative.HeadlineVariants)
	ctas, _ := json.Marshal(input.Creative.CTAVariants)
	preview, _ := json.Marshal(map[string]any{"destinationUrl": input.DestinationURL, "platform": "Meta", "status": "PAUSED"})
	_, err = tx.Exec(ctx, `INSERT INTO ad_creatives(ad_campaign_id,media_asset_id,thumbnail_asset_id,primary_text_variants,headline_variants,cta_variants,preview_spec)VALUES($1,$2,$3,$4,$5,$6,$7)`, id, input.Creative.MediaAssetID, input.Creative.ThumbnailAssetID, texts, headlines, ctas, preview)
	if err != nil {
		return Campaign{}, err
	}
	actionID := uuid.New()
	actionHash := digest(id.String() + "|CREATE_PAUSED|" + hash)
	_, err = tx.Exec(ctx, `INSERT INTO meta_ad_actions(id,ad_campaign_id,action,status,requested_budget_minor,confirmation_text,action_hash,idempotency_key,requested_by)VALUES($1,$2,'CREATE_PAUSED','PENDING_APPROVAL',$3,'',$4,$5,$6)`, actionID, id, budget, actionHash, key, actorID)
	if err != nil {
		return Campaign{}, err
	}
	item, err := get(ctx, tx, clientID, workspaceID, campaignID, id)
	if err != nil {
		return Campaign{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Campaign{}, err
	}
	return item, nil
}

func (s *Service) Update(ctx context.Context, clientID, workspaceID, campaignID, id uuid.UUID, actor auth.Principal, metadata auth.ClientMetadata, input CampaignInput) (Campaign, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Objective = strings.TrimSpace(input.Objective)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if input.Version < 1 || input.Name == "" || input.Objective == "" || input.MetaAdAccountID == uuid.Nil || input.SocialAccountID == uuid.Nil || input.CampaignSpendCapMinor <= 0 || ((input.DailyBudgetMinor == nil) == (input.LifetimeBudgetMinor == nil)) || len(input.Creative.PrimaryTextVariants) == 0 || len(input.Creative.HeadlineVariants) == 0 || len(input.Creative.CTAVariants) == 0 {
		return Campaign{}, ErrInvalid
	}
	destination, err := url.Parse(input.DestinationURL)
	if err != nil || destination.Scheme != "https" || destination.Host == "" {
		return Campaign{}, ErrInvalid
	}
	budget := value64(input.DailyBudgetMinor)
	if budget == 0 {
		budget = value64(input.LifetimeBudgetMinor)
	}
	if budget <= 0 || budget > input.CampaignSpendCapMinor {
		return Campaign{}, ErrGuardrail
	}
	if input.EndsAt != nil && input.StartsAt != nil && !input.EndsAt.After(*input.StartsAt) {
		return Campaign{}, ErrInvalid
	}
	if input.Audience.AgeMin < 18 || input.Audience.AgeMax > 65 || input.Audience.AgeMax < input.Audience.AgeMin || len(input.Audience.Countries) == 0 {
		return Campaign{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Campaign{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := scanCampaign(tx.QueryRow(ctx, campaignSelect+` WHERE a.id=$1 AND a.client_id=$2 AND a.workspace_id=$3 AND a.campaign_id=$4 FOR UPDATE`, id, clientID, workspaceID, campaignID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, ErrNotFound
	}
	if err != nil {
		return Campaign{}, err
	}
	if before.Version != input.Version || before.ProviderCampaignID != nil || !map[string]bool{"DRAFT": true, "APPROVAL_REQUIRED": true}[before.Status] {
		return Campaign{}, ErrConflict
	}
	var workspaceCap, defaultCap, allocatedCap int64
	var currency string
	var accountValid, creativeValid, campaignApproved, pixelValid bool
	err = tx.QueryRow(ctx, `SELECT g.workspace_spend_cap_minor,g.default_campaign_spend_cap_minor,g.currency,COALESCE((SELECT sum(a.campaign_spend_cap_minor) FROM ad_campaigns a WHERE a.workspace_id=$2 AND a.id<>$8 AND a.status NOT IN ('ARCHIVED','FAILED')),0),EXISTS(SELECT 1 FROM meta_ad_accounts a JOIN meta_connections c ON c.id=a.connection_id WHERE a.id=$4 AND a.client_id=$1 AND a.workspace_id=$2 AND a.currency=g.currency AND c.status IN ('CONNECTED','EXPIRING') AND (c.token_expires_at IS NULL OR c.token_expires_at>now()) AND (c.data_access_expires_at IS NULL OR c.data_access_expires_at>now())),EXISTS(SELECT 1 FROM media_assets m WHERE m.id=$6 AND m.client_id=$1 AND m.workspace_id=$2 AND m.deleted_at IS NULL),EXISTS(SELECT 1 FROM campaigns c JOIN video_projects p ON p.campaign_id=c.id JOIN render_jobs r ON r.id=p.selected_render_job_id AND r.status='APPROVED' WHERE c.id=$3 AND c.client_id=$1 AND c.workspace_id=$2 AND c.status='APPROVED' AND EXISTS(SELECT 1 FROM social_accounts s WHERE s.id=$5 AND s.client_id=$1 AND s.workspace_id=$2 AND s.status IN ('CONNECTED','EXPIRING'))),($7::uuid IS NULL OR EXISTS(SELECT 1 FROM meta_pixels px WHERE px.id=$7 AND px.meta_ad_account_id=$4)) FROM meta_ad_guardrails g WHERE g.client_id=$1 AND g.workspace_id=$2`, clientID, workspaceID, campaignID, input.MetaAdAccountID, input.SocialAccountID, input.Creative.MediaAssetID, input.MetaPixelID, id).Scan(&workspaceCap, &defaultCap, &currency, &allocatedCap, &accountValid, &creativeValid, &campaignApproved, &pixelValid)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, ErrPrerequisite
	}
	if err != nil {
		return Campaign{}, err
	}
	if input.Currency != currency || allocatedCap+input.CampaignSpendCapMinor > workspaceCap || input.CampaignSpendCapMinor > defaultCap || !accountValid || !creativeValid || !campaignApproved || !pixelValid {
		return Campaign{}, ErrGuardrail
	}
	audienceRaw, _ := json.Marshal(input.Audience)
	utmRaw, _ := json.Marshal(input.UTMParameters)
	hash := campaignHash(input)
	_, err = tx.Exec(ctx, `UPDATE ad_campaigns SET meta_ad_account_id=$5,social_account_id=$6,meta_pixel_id=$7,name=$8,objective=$9,daily_budget_minor=$10,lifetime_budget_minor=$11,campaign_spend_cap_minor=$12,currency=$13,starts_at=$14,ends_at=$15,audience=$16,placements=$17,destination_url=$18,utm_parameters=$19,conversion_event=$20,status='APPROVAL_REQUIRED',campaign_hash=$21,last_error_code=NULL,last_error_message=NULL,version=version+1,updated_by=$22,updated_at=now() WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND campaign_id=$4 AND version=$23`, id, clientID, workspaceID, campaignID, input.MetaAdAccountID, input.SocialAccountID, input.MetaPixelID, input.Name, input.Objective, input.DailyBudgetMinor, input.LifetimeBudgetMinor, input.CampaignSpendCapMinor, input.Currency, input.StartsAt, input.EndsAt, audienceRaw, input.Placements, input.DestinationURL, utmRaw, input.ConversionEvent, hash, actor.UserID, input.Version)
	if err != nil {
		return Campaign{}, err
	}
	texts, _ := json.Marshal(input.Creative.PrimaryTextVariants)
	headlines, _ := json.Marshal(input.Creative.HeadlineVariants)
	ctas, _ := json.Marshal(input.Creative.CTAVariants)
	preview, _ := json.Marshal(map[string]any{"destinationUrl": input.DestinationURL, "platform": "Meta", "status": "PAUSED"})
	if _, err = tx.Exec(ctx, `UPDATE ad_creatives SET media_asset_id=$2,thumbnail_asset_id=$3,primary_text_variants=$4,headline_variants=$5,cta_variants=$6,preview_spec=$7,updated_at=now() WHERE ad_campaign_id=$1`, id, input.Creative.MediaAssetID, input.Creative.ThumbnailAssetID, texts, headlines, ctas, preview); err != nil {
		return Campaign{}, err
	}
	if _, err = tx.Exec(ctx, `WITH changed AS (
		UPDATE approvals SET invalidated_at=now(),invalidation_reason='Meta Ads draft changed'
		WHERE entity_type='AD_CAMPAIGN' AND entity_id=$1 AND invalidated_at IS NULL
		RETURNING id,entity_version,entity_hash
	) INSERT INTO approval_events(approval_id,event_type,actor_id,entity_version,entity_hash,notes)
	SELECT id,'INVALIDATED',$2,entity_version,entity_hash,'Meta Ads draft changed' FROM changed`, id, actor.UserID); err != nil {
		return Campaign{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE meta_ad_actions SET status='REJECTED',reviewed_by=$2,review_notes='Superseded by draft edit',reviewed_at=now(),version=version+1 WHERE ad_campaign_id=$1 AND action='CREATE_PAUSED' AND status='PENDING_APPROVAL'`, id, actor.UserID); err != nil {
		return Campaign{}, err
	}
	actionID := uuid.New()
	actionHash := digest(id.String() + "|CREATE_PAUSED|" + hash)
	editKey := fmt.Sprintf("edit:%s:%d", id, before.Version+1)
	if _, err = tx.Exec(ctx, `INSERT INTO meta_ad_actions(id,ad_campaign_id,action,status,requested_budget_minor,confirmation_text,action_hash,idempotency_key,requested_by)VALUES($1,$2,'CREATE_PAUSED','PENDING_APPROVAL',$3,'',$4,$5,$6)`, actionID, id, budget, actionHash, editKey, actor.UserID); err != nil {
		return Campaign{}, err
	}
	item, err := get(ctx, tx, clientID, workspaceID, campaignID, id)
	if err != nil {
		return Campaign{}, err
	}
	if err = audit.Record(ctx, db.New(tx), audit.Event{ActorID: uuid.NullUUID{UUID: actor.UserID, Valid: true}, Action: "meta_ad_campaign.updated", EntityType: "ad_campaign", EntityID: uuid.NullUUID{UUID: id, Valid: true}, ClientID: uuid.NullUUID{UUID: clientID, Valid: true}, WorkspaceID: uuid.NullUUID{UUID: workspaceID, Valid: true}, RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS", Before: before, After: item}); err != nil {
		return Campaign{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Campaign{}, err
	}
	return item, nil
}

func (s *Service) ReviewCreate(ctx context.Context, clientID, workspaceID, campaignID, id, actorID uuid.UUID, input ReviewInput) (Campaign, error) {
	action := strings.ToUpper(strings.TrimSpace(input.Action))
	if (action != "APPROVE" && action != "REJECT") || input.Version < 1 || (action == "REJECT" && strings.TrimSpace(input.Notes) == "") {
		return Campaign{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Campaign{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status, hash, currency string
	var version int64
	var daily, lifetime *int64
	err = tx.QueryRow(ctx, `SELECT status::text,campaign_hash,version,currency,daily_budget_minor,lifetime_budget_minor FROM ad_campaigns WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND campaign_id=$4 FOR UPDATE`, id, clientID, workspaceID, campaignID).Scan(&status, &hash, &version, &currency, &daily, &lifetime)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, ErrNotFound
	} else if err != nil {
		return Campaign{}, err
	}
	if status != "APPROVAL_REQUIRED" || version != input.Version {
		return Campaign{}, ErrConflict
	}
	budget := value64(daily)
	if budget == 0 {
		budget = value64(lifetime)
	}
	if action == "APPROVE" && (input.ConfirmedBudgetMinor != budget || strings.TrimSpace(input.ConfirmationText) != "CREATE PAUSED "+currency+" "+fmt.Sprint(budget)) {
		return Campaign{}, ErrGuardrail
	}
	newStatus, approvalStatus, actionStatus := "DRAFT", "REJECTED", "REJECTED"
	if action == "APPROVE" {
		newStatus, approvalStatus, actionStatus = "APPROVED", "APPROVED", "QUEUED"
	}
	_, err = tx.Exec(ctx, `UPDATE ad_campaigns SET status=$2::meta_ad_campaign_status,version=version+1,updated_by=$3,updated_at=now() WHERE id=$1`, id, newStatus, actorID)
	if err != nil {
		return Campaign{}, err
	}
	var approvalID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO approvals(client_id,workspace_id,campaign_id,entity_type,entity_id,entity_version,entity_hash,status,requested_by,decided_by,decided_at,notes)VALUES($1,$2,$3,'AD_CAMPAIGN',$4,$5,$6,$7,$8,$8,now(),$9)RETURNING id`, clientID, workspaceID, campaignID, id, version, hash, approvalStatus, actorID, input.Notes).Scan(&approvalID)
	if err != nil {
		return Campaign{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO approval_events(approval_id,event_type,actor_id,entity_version,entity_hash,notes)VALUES($1,$2::approval_event_type,$3,$4,$5,$6)`, approvalID, approvalStatus, actorID, version, hash, input.Notes)
	if err != nil {
		return Campaign{}, err
	}
	var actionID uuid.UUID
	err = tx.QueryRow(ctx, `UPDATE meta_ad_actions SET status=$2::meta_action_status,reviewed_by=$3,review_notes=$4,reviewed_at=now(),version=version+1 WHERE ad_campaign_id=$1 AND action='CREATE_PAUSED' AND status='PENDING_APPROVAL' RETURNING id`, id, actionStatus, actorID, input.Notes).Scan(&actionID)
	if err != nil {
		return Campaign{}, err
	}
	if action == "APPROVE" {
		riverID, enqueueErr := s.enqueuer.EnqueueMetaAdAction(ctx, tx, actionID)
		if enqueueErr != nil {
			return Campaign{}, enqueueErr
		}
		if _, err = tx.Exec(ctx, `UPDATE meta_ad_actions SET river_job_id=$2 WHERE id=$1`, actionID, riverID); err != nil {
			return Campaign{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Campaign{}, err
	}
	return get(ctx, s.pool, clientID, workspaceID, campaignID, id)
}

func (s *Service) RequestAction(ctx context.Context, clientID, workspaceID, campaignID, id, actorID uuid.UUID, input ActionInput) (Action, error) {
	input.Action = strings.ToUpper(strings.TrimSpace(input.Action))
	input.ConfirmationText = strings.TrimSpace(input.ConfirmationText)
	if len(strings.TrimSpace(input.IdempotencyKey)) < 16 || !map[string]bool{"ACTIVATE": true, "RESUME": true, "PAUSE": true, "ARCHIVE": true, "BUDGET_CHANGE": true}[input.Action] {
		return Action{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Action{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status, currency string
	var providerID *string
	var version int64
	var daily, lifetime *int64
	var campaignCap, workspaceCap int64
	var maxIncrease float64
	err = tx.QueryRow(ctx, `SELECT a.status::text,a.provider_campaign_id,a.version,a.currency,a.daily_budget_minor,a.lifetime_budget_minor,a.campaign_spend_cap_minor,g.workspace_spend_cap_minor,g.maximum_budget_increase_percent::float8 FROM ad_campaigns a JOIN meta_ad_guardrails g ON g.workspace_id=a.workspace_id AND g.client_id=a.client_id WHERE a.id=$1 AND a.client_id=$2 AND a.workspace_id=$3 AND a.campaign_id=$4 FOR UPDATE`, id, clientID, workspaceID, campaignID).Scan(&status, &providerID, &version, &currency, &daily, &lifetime, &campaignCap, &workspaceCap, &maxIncrease)
	if errors.Is(err, pgx.ErrNoRows) {
		return Action{}, ErrNotFound
	} else if err != nil {
		return Action{}, err
	}
	if providerID == nil {
		return Action{}, ErrPrerequisite
	}
	previous := value64(daily)
	if previous == 0 {
		previous = value64(lifetime)
	}
	requiresReview := input.Action == "ACTIVATE" || input.Action == "RESUME" || input.Action == "BUDGET_CHANGE"
	if (input.Action == "ACTIVATE" || input.Action == "RESUME") && input.ConfirmationText != "ACTIVATE "+currency+" "+fmt.Sprint(previous) {
		return Action{}, ErrGuardrail
	}
	if input.Action == "BUDGET_CHANGE" {
		if input.RequestedBudgetMinor == nil || *input.RequestedBudgetMinor <= 0 || *input.RequestedBudgetMinor > campaignCap || *input.RequestedBudgetMinor > workspaceCap {
			return Action{}, ErrGuardrail
		}
		if *input.RequestedBudgetMinor > previous {
			increase := float64(*input.RequestedBudgetMinor-previous) / float64(previous) * 100
			if increase > maxIncrease {
				return Action{}, ErrGuardrail
			}
		}
		if input.ConfirmationText != "BUDGET "+currency+" "+fmt.Sprint(*input.RequestedBudgetMinor) {
			return Action{}, ErrGuardrail
		}
	}
	actionHash := digest(fmt.Sprintf("%s|%s|%d|%d", id, input.Action, version, value64(input.RequestedBudgetMinor)))
	actionID := uuid.New()
	actionStatus := "APPROVED"
	if requiresReview {
		actionStatus = "PENDING_APPROVAL"
	}
	err = tx.QueryRow(ctx, `INSERT INTO meta_ad_actions(id,ad_campaign_id,action,status,requested_budget_minor,previous_budget_minor,confirmation_text,action_hash,idempotency_key,requested_by)VALUES($1,$2,$3::meta_action_type,$4::meta_action_status,$5,$6,$7,$8,$9,$10)ON CONFLICT(idempotency_key)DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key RETURNING id`, actionID, id, input.Action, actionStatus, input.RequestedBudgetMinor, previous, input.ConfirmationText, actionHash, input.IdempotencyKey, actorID).Scan(&actionID)
	if err != nil {
		return Action{}, err
	}
	if !requiresReview {
		riverID, enqueueErr := s.enqueuer.EnqueueMetaAdAction(ctx, tx, actionID)
		if enqueueErr != nil {
			return Action{}, enqueueErr
		}
		_, err = tx.Exec(ctx, `UPDATE meta_ad_actions SET status='QUEUED',river_job_id=$2 WHERE id=$1`, actionID, riverID)
		if err != nil {
			return Action{}, err
		}
	}
	item, err := getAction(ctx, tx, id, actionID)
	if err != nil {
		return Action{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Action{}, err
	}
	return item, nil
}

func (s *Service) ReviewAction(ctx context.Context, clientID, workspaceID, campaignID, campaignID2, actionID, actorID uuid.UUID, input ReviewInput) (Action, error) {
	action := strings.ToUpper(strings.TrimSpace(input.Action))
	if (action != "APPROVE" && action != "REJECT") || input.Version < 1 {
		return Action{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Action{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var kind, status, hash string
	var version int64
	err = tx.QueryRow(ctx, `SELECT x.action::text,x.status::text,x.action_hash,x.version FROM meta_ad_actions x JOIN ad_campaigns a ON a.id=x.ad_campaign_id WHERE x.id=$1 AND a.id=$2 AND a.client_id=$3 AND a.workspace_id=$4 AND a.campaign_id=$5 FOR UPDATE`, actionID, campaignID2, clientID, workspaceID, campaignID).Scan(&kind, &status, &hash, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Action{}, ErrNotFound
	} else if err != nil {
		return Action{}, err
	}
	if status != "PENDING_APPROVAL" || version != input.Version {
		return Action{}, ErrConflict
	}
	newStatus, approvalStatus := "REJECTED", "REJECTED"
	if action == "APPROVE" {
		newStatus, approvalStatus = "QUEUED", "APPROVED"
	}
	_, err = tx.Exec(ctx, `UPDATE meta_ad_actions SET status=$2::meta_action_status,reviewed_by=$3,review_notes=$4,reviewed_at=now(),version=version+1 WHERE id=$1`, actionID, newStatus, actorID, input.Notes)
	if err != nil {
		return Action{}, err
	}
	entityType := "AD_CAMPAIGN"
	if kind == "BUDGET_CHANGE" {
		entityType = "BUDGET_CHANGE"
	}
	var approvalID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO approvals(client_id,workspace_id,campaign_id,entity_type,entity_id,entity_version,entity_hash,status,requested_by,decided_by,decided_at,notes)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9,now(),$10)RETURNING id`, clientID, workspaceID, campaignID, entityType, actionID, version, hash, approvalStatus, actorID, input.Notes).Scan(&approvalID)
	if err != nil {
		return Action{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO approval_events(approval_id,event_type,actor_id,entity_version,entity_hash,notes)VALUES($1,$2::approval_event_type,$3,$4,$5,$6)`, approvalID, approvalStatus, actorID, version, hash, input.Notes)
	if err != nil {
		return Action{}, err
	}
	if action == "APPROVE" {
		riverID, enqueueErr := s.enqueuer.EnqueueMetaAdAction(ctx, tx, actionID)
		if enqueueErr != nil {
			return Action{}, enqueueErr
		}
		_, err = tx.Exec(ctx, `UPDATE meta_ad_actions SET river_job_id=$2 WHERE id=$1`, actionID, riverID)
		if err != nil {
			return Action{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Action{}, err
	}
	return getAction(ctx, s.pool, campaignID2, actionID)
}

func (s *Service) List(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID) ([]Campaign, error) {
	rows, err := s.pool.Query(ctx, campaignSelect+` WHERE a.client_id=$1 AND a.workspace_id=$2 AND a.campaign_id=$3 ORDER BY a.created_at DESC`, clientID, workspaceID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Campaign{}
	for rows.Next() {
		item, e := scanCampaign(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) ListActions(ctx context.Context, clientID, workspaceID, campaignID, adCampaignID uuid.UUID) ([]Action, error) {
	rows, err := s.pool.Query(ctx, actionSelect+` x JOIN ad_campaigns a ON a.id=x.ad_campaign_id WHERE a.client_id=$1 AND a.workspace_id=$2 AND a.campaign_id=$3 AND a.id=$4 ORDER BY x.requested_at DESC`, clientID, workspaceID, campaignID, adCampaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Action{}
	for rows.Next() {
		item, scanErr := scanAction(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

const campaignSelect = `SELECT a.id,a.campaign_id,a.meta_ad_account_id,a.social_account_id,a.meta_pixel_id,a.name,a.objective,a.currency,a.destination_url,a.status::text,a.campaign_hash,a.daily_budget_minor,a.lifetime_budget_minor,a.campaign_spend_cap_minor,a.starts_at,a.ends_at,a.audience,a.placements,a.utm_parameters,a.conversion_event,a.provider_campaign_id,a.last_error_code,a.last_error_message,a.version,a.created_at,a.updated_at,c.media_asset_id,c.thumbnail_asset_id,c.primary_text_variants,c.headline_variants,c.cta_variants FROM ad_campaigns a JOIN ad_creatives c ON c.ad_campaign_id=a.id`

type scanner interface{ Scan(...any) error }
type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func scanCampaign(row scanner) (Campaign, error) {
	var c Campaign
	var audience, utm, texts, headlines, ctas []byte
	err := row.Scan(&c.ID, &c.CampaignID, &c.MetaAdAccountID, &c.SocialAccountID, &c.MetaPixelID, &c.Name, &c.Objective, &c.Currency, &c.DestinationURL, &c.Status, &c.CampaignHash, &c.DailyBudgetMinor, &c.LifetimeBudgetMinor, &c.CampaignSpendCapMinor, &c.StartsAt, &c.EndsAt, &audience, &c.Placements, &utm, &c.ConversionEvent, &c.ProviderCampaignID, &c.LastErrorCode, &c.LastErrorMessage, &c.Version, &c.CreatedAt, &c.UpdatedAt, &c.Creative.MediaAssetID, &c.Creative.ThumbnailAssetID, &texts, &headlines, &ctas)
	if err != nil {
		return c, err
	}
	_ = json.Unmarshal(audience, &c.Audience)
	_ = json.Unmarshal(utm, &c.UTMParameters)
	_ = json.Unmarshal(texts, &c.Creative.PrimaryTextVariants)
	_ = json.Unmarshal(headlines, &c.Creative.HeadlineVariants)
	_ = json.Unmarshal(ctas, &c.Creative.CTAVariants)
	return c, nil
}
func get(ctx context.Context, q queryer, clientID, workspaceID, campaignID, id uuid.UUID) (Campaign, error) {
	item, err := scanCampaign(q.QueryRow(ctx, campaignSelect+` WHERE a.id=$1 AND a.client_id=$2 AND a.workspace_id=$3 AND a.campaign_id=$4`, id, clientID, workspaceID, campaignID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, ErrNotFound
	}
	return item, err
}
func getByKey(ctx context.Context, q queryer, clientID, workspaceID, campaignID uuid.UUID, key string) (Campaign, error) {
	return scanCampaign(q.QueryRow(ctx, campaignSelect+` JOIN meta_ad_actions x ON x.ad_campaign_id=a.id AND x.action='CREATE_PAUSED' WHERE a.client_id=$1 AND a.workspace_id=$2 AND a.campaign_id=$3 AND x.idempotency_key=$4`, clientID, workspaceID, campaignID, key))
}

const actionSelect = `SELECT x.id,x.ad_campaign_id,x.action::text,x.status::text,x.requested_budget_minor,x.previous_budget_minor,x.confirmation_text,x.error_code,x.error_message,x.version,x.requested_at FROM meta_ad_actions`

func scanAction(row scanner) (Action, error) {
	var a Action
	err := row.Scan(&a.ID, &a.AdCampaignID, &a.Action, &a.Status, &a.RequestedBudgetMinor, &a.PreviousBudgetMinor, &a.ConfirmationText, &a.ErrorCode, &a.ErrorMessage, &a.Version, &a.RequestedAt)
	return a, err
}
func getAction(ctx context.Context, q queryer, campaignID, actionID uuid.UUID) (Action, error) {
	a, err := scanAction(q.QueryRow(ctx, actionSelect+` x WHERE x.id=$1 AND x.ad_campaign_id=$2`, actionID, campaignID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Action{}, ErrNotFound
	}
	return a, err
}
func campaignHash(input CampaignInput) string {
	input.Version = 0
	raw, _ := json.Marshal(input)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func digest(v string) string { sum := sha256.Sum256([]byte(v)); return hex.EncodeToString(sum[:]) }
func value64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
