package publishing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	ErrInvalid      = errors.New("invalid social post")
	ErrNotFound     = errors.New("social post not found")
	ErrConflict     = errors.New("social post conflict")
	ErrPrerequisite = errors.New("publishing prerequisite not met")
)

type Input struct {
	SocialAccountID uuid.UUID  `json:"socialAccountId"`
	MediaAssetID    uuid.UUID  `json:"mediaAssetId"`
	Caption         string     `json:"caption"`
	ScheduledAt     *time.Time `json:"scheduledAt"`
	Version         int64      `json:"version"`
}
type ReviewInput struct {
	Action  string `json:"action"`
	Version int64  `json:"version"`
	Notes   string `json:"notes"`
}
type Post struct {
	ID              uuid.UUID  `json:"id"`
	CampaignID      uuid.UUID  `json:"campaignId"`
	SocialAccountID uuid.UUID  `json:"socialAccountId"`
	Platform        string     `json:"platform"`
	MediaAssetID    uuid.UUID  `json:"mediaAssetId"`
	Caption         string     `json:"caption"`
	ScheduledAt     *time.Time `json:"scheduledAt"`
	Status          string     `json:"status"`
	ContentHash     string     `json:"contentHash"`
	ProviderPostID  *string    `json:"providerPostId"`
	PublicURL       *string    `json:"publicUrl"`
	ErrorCategory   *string    `json:"errorCategory"`
	ErrorCode       *string    `json:"errorCode"`
	ErrorMessage    *string    `json:"errorMessage"`
	PublishedAt     *time.Time `json:"publishedAt"`
	ReviewNotes     string     `json:"reviewNotes"`
	Version         int64      `json:"version"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	Reused          bool       `json:"reused,omitempty"`
}

type Service struct {
	pool     *pgxpool.Pool
	enqueuer *jobs.Enqueuer
}

func NewService(pool *pgxpool.Pool, enqueuer *jobs.Enqueuer) *Service {
	return &Service{pool: pool, enqueuer: enqueuer}
}

func (s *Service) Create(ctx context.Context, clientID, workspaceID, campaignID, actorID uuid.UUID, key string, input Input) (Post, error) {
	input.Caption = strings.TrimSpace(input.Caption)
	key = strings.TrimSpace(key)
	if len(key) < 16 || len(input.Caption) < 1 || len(input.Caption) > 2200 || input.SocialAccountID == uuid.Nil || input.MediaAssetID == uuid.Nil {
		return Post{}, ErrInvalid
	}
	if input.ScheduledAt != nil && input.ScheduledAt.Before(time.Now().Add(time.Minute)) {
		return Post{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if existing, findErr := getByKey(ctx, tx, clientID, workspaceID, campaignID, key); findErr == nil {
		existing.Reused = true
		_ = tx.Commit(ctx)
		return existing, nil
	} else if !errors.Is(findErr, pgx.ErrNoRows) {
		return Post{}, findErr
	}
	var platform, mime string
	var campaignApproved, accountConnected bool
	err = tx.QueryRow(ctx, `SELECT a.platform::text,a.status IN ('CONNECTED','EXPIRING'),v.mime_type,EXISTS(SELECT 1 FROM campaigns c JOIN video_projects p ON p.campaign_id=c.id JOIN render_jobs r ON r.id=p.selected_render_job_id AND r.status='APPROVED' WHERE c.id=$1 AND c.client_id=$2 AND c.workspace_id=$3 AND c.status='APPROVED') FROM social_accounts a JOIN media_assets m ON m.id=$5 AND m.client_id=$2 AND m.workspace_id=$3 AND m.deleted_at IS NULL JOIN media_asset_versions v ON v.media_asset_id=m.id AND v.version=m.current_version WHERE a.id=$4 AND a.client_id=$2 AND a.workspace_id=$3 AND a.disconnected_at IS NULL`, campaignID, clientID, workspaceID, input.SocialAccountID, input.MediaAssetID).Scan(&platform, &accountConnected, &mime, &campaignApproved)
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, ErrNotFound
	} else if err != nil {
		return Post{}, err
	}
	if !accountConnected || !campaignApproved || (!strings.HasPrefix(mime, "image/") && !strings.HasPrefix(mime, "video/")) {
		return Post{}, ErrPrerequisite
	}
	hash := postHash(platform, input)
	id := uuid.New()
	_, err = tx.Exec(ctx, `INSERT INTO social_posts(id,client_id,workspace_id,campaign_id,social_account_id,platform,media_asset_id,caption,scheduled_at,idempotency_key,status,content_hash,created_by,updated_by)VALUES($1,$2,$3,$4,$5,$6::social_platform,$7,$8,$9,$10,'APPROVAL_REQUIRED',$11,$12,$12)`, id, clientID, workspaceID, campaignID, input.SocialAccountID, platform, input.MediaAssetID, input.Caption, input.ScheduledAt, key, hash, actorID)
	if err != nil {
		return Post{}, err
	}
	item, err := get(ctx, tx, clientID, workspaceID, campaignID, id)
	if err != nil {
		return Post{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Post{}, err
	}
	return item, nil
}
func (s *Service) Update(ctx context.Context, clientID, workspaceID, campaignID, postID uuid.UUID, actor auth.Principal, metadata auth.ClientMetadata, input Input) (Post, error) {
	input.Caption = strings.TrimSpace(input.Caption)
	if input.Version < 1 || len(input.Caption) < 1 || len(input.Caption) > 2200 || input.SocialAccountID == uuid.Nil || input.MediaAssetID == uuid.Nil {
		return Post{}, ErrInvalid
	}
	if input.ScheduledAt != nil && input.ScheduledAt.Before(time.Now().Add(time.Minute)) {
		return Post{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := scanPost(tx.QueryRow(ctx, postSelect+` WHERE p.id=$1 AND p.client_id=$2 AND p.workspace_id=$3 AND p.campaign_id=$4 FOR UPDATE`, postID, clientID, workspaceID, campaignID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, ErrNotFound
	}
	if err != nil {
		return Post{}, err
	}
	if before.Version != input.Version || !map[string]bool{"DRAFT": true, "APPROVAL_REQUIRED": true, "CANCELLED": true}[before.Status] {
		return Post{}, ErrConflict
	}
	var platform, mime string
	var campaignApproved, accountConnected bool
	err = tx.QueryRow(ctx, `SELECT a.platform::text,a.status IN ('CONNECTED','EXPIRING'),v.mime_type,EXISTS(SELECT 1 FROM campaigns c JOIN video_projects p ON p.campaign_id=c.id JOIN render_jobs r ON r.id=p.selected_render_job_id AND r.status='APPROVED' WHERE c.id=$1 AND c.client_id=$2 AND c.workspace_id=$3 AND c.status='APPROVED') FROM social_accounts a JOIN media_assets m ON m.id=$5 AND m.client_id=$2 AND m.workspace_id=$3 AND m.deleted_at IS NULL JOIN media_asset_versions v ON v.media_asset_id=m.id AND v.version=m.current_version WHERE a.id=$4 AND a.client_id=$2 AND a.workspace_id=$3 AND a.disconnected_at IS NULL`, campaignID, clientID, workspaceID, input.SocialAccountID, input.MediaAssetID).Scan(&platform, &accountConnected, &mime, &campaignApproved)
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, ErrNotFound
	}
	if err != nil {
		return Post{}, err
	}
	if !accountConnected || !campaignApproved || (!strings.HasPrefix(mime, "image/") && !strings.HasPrefix(mime, "video/")) {
		return Post{}, ErrPrerequisite
	}
	hash := postHash(platform, input)
	_, err = tx.Exec(ctx, `UPDATE social_posts SET social_account_id=$5,platform=$6::social_platform,media_asset_id=$7,caption=$8,scheduled_at=$9,status='APPROVAL_REQUIRED',content_hash=$10,reviewed_at=NULL,reviewed_by=NULL,review_notes='',version=version+1,updated_by=$11,updated_at=now() WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND campaign_id=$4 AND version=$12`, postID, clientID, workspaceID, campaignID, input.SocialAccountID, platform, input.MediaAssetID, input.Caption, input.ScheduledAt, hash, actor.UserID, input.Version)
	if err != nil {
		return Post{}, err
	}
	if _, err = tx.Exec(ctx, `WITH changed AS (
		UPDATE approvals SET invalidated_at=now(),invalidation_reason='Social post content changed'
		WHERE entity_type='SOCIAL_POST' AND entity_id=$1 AND invalidated_at IS NULL
		RETURNING id,entity_version,entity_hash
	) INSERT INTO approval_events(approval_id,event_type,actor_id,entity_version,entity_hash,notes)
	SELECT id,'INVALIDATED',$2,entity_version,entity_hash,'Social post content changed' FROM changed`, postID, actor.UserID); err != nil {
		return Post{}, err
	}
	item, err := get(ctx, tx, clientID, workspaceID, campaignID, postID)
	if err != nil {
		return Post{}, err
	}
	if err = audit.Record(ctx, db.New(tx), audit.Event{ActorID: uuid.NullUUID{UUID: actor.UserID, Valid: true}, Action: "social_post.updated", EntityType: "social_post", EntityID: uuid.NullUUID{UUID: postID, Valid: true}, ClientID: uuid.NullUUID{UUID: clientID, Valid: true}, WorkspaceID: uuid.NullUUID{UUID: workspaceID, Valid: true}, RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS", Before: before, After: item}); err != nil {
		return Post{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Post{}, err
	}
	return item, nil
}
func (s *Service) Review(ctx context.Context, clientID, workspaceID, campaignID, postID, actorID uuid.UUID, input ReviewInput) (Post, error) {
	action := strings.ToUpper(strings.TrimSpace(input.Action))
	if (action != "APPROVE" && action != "REJECT") || input.Version < 1 || (action == "REJECT" && strings.TrimSpace(input.Notes) == "") {
		return Post{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status, hash string
	var version int64
	var scheduled *time.Time
	err = tx.QueryRow(ctx, `SELECT status::text,content_hash,version,scheduled_at FROM social_posts WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND campaign_id=$4 FOR UPDATE`, postID, clientID, workspaceID, campaignID).Scan(&status, &hash, &version, &scheduled)
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, ErrNotFound
	} else if err != nil {
		return Post{}, err
	}
	if status != "APPROVAL_REQUIRED" || version != input.Version {
		return Post{}, ErrConflict
	}
	approvalStatus, newStatus := "REJECTED", "CANCELLED"
	if action == "APPROVE" {
		approvalStatus = "APPROVED"
		newStatus = "APPROVED"
		if scheduled != nil {
			newStatus = "SCHEDULED"
		}
	}
	_, err = tx.Exec(ctx, `UPDATE social_posts SET status=$2::social_post_status,reviewed_at=now(),reviewed_by=$3,review_notes=$4,version=version+1,updated_at=now() WHERE id=$1`, postID, newStatus, actorID, strings.TrimSpace(input.Notes))
	if err != nil {
		return Post{}, err
	}
	var approvalID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO approvals(client_id,workspace_id,campaign_id,entity_type,entity_id,entity_version,entity_hash,status,requested_by,decided_by,decided_at,notes)VALUES($1,$2,$3,'SOCIAL_POST',$4,$5,$6,$7,$8,$8,now(),$9)RETURNING id`, clientID, workspaceID, campaignID, postID, version, hash, approvalStatus, actorID, input.Notes).Scan(&approvalID)
	if err != nil {
		return Post{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO approval_events(approval_id,event_type,actor_id,entity_version,entity_hash,notes)VALUES($1,$2::approval_event_type,$3,$4,$5,$6)`, approvalID, approvalStatus, actorID, version, hash, input.Notes)
	if err != nil {
		return Post{}, err
	}
	if action == "APPROVE" {
		jobID, enqueueErr := s.enqueuer.EnqueueSocialPublish(ctx, tx, postID, scheduled)
		if enqueueErr != nil {
			return Post{}, enqueueErr
		}
		_, err = tx.Exec(ctx, `INSERT INTO publish_jobs(social_post_id,idempotency_key,river_job_id)VALUES($1,$2,$3)`, postID, "publish:"+postID.String()+":"+hash, jobID)
		if err != nil {
			return Post{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Post{}, err
	}
	return get(ctx, s.pool, clientID, workspaceID, campaignID, postID)
}
func (s *Service) List(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID) ([]Post, error) {
	rows, err := s.pool.Query(ctx, postSelect+` WHERE p.client_id=$1 AND p.workspace_id=$2 AND p.campaign_id=$3 ORDER BY p.created_at DESC`, clientID, workspaceID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Post{}
	for rows.Next() {
		item, scanErr := scanPost(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

const postSelect = `SELECT p.id,p.campaign_id,p.social_account_id,p.platform::text,p.media_asset_id,p.caption,p.scheduled_at,p.status::text,p.content_hash,p.provider_post_id,p.public_url,p.error_category,p.error_code,p.error_message,p.published_at,p.review_notes,p.version,p.created_at,p.updated_at FROM social_posts p`

type scanner interface{ Scan(...any) error }
type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func scanPost(row scanner) (Post, error) {
	var p Post
	err := row.Scan(&p.ID, &p.CampaignID, &p.SocialAccountID, &p.Platform, &p.MediaAssetID, &p.Caption, &p.ScheduledAt, &p.Status, &p.ContentHash, &p.ProviderPostID, &p.PublicURL, &p.ErrorCategory, &p.ErrorCode, &p.ErrorMessage, &p.PublishedAt, &p.ReviewNotes, &p.Version, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}
func get(ctx context.Context, q queryer, clientID, workspaceID, campaignID, id uuid.UUID) (Post, error) {
	item, err := scanPost(q.QueryRow(ctx, postSelect+` WHERE p.id=$1 AND p.client_id=$2 AND p.workspace_id=$3 AND p.campaign_id=$4`, id, clientID, workspaceID, campaignID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, ErrNotFound
	}
	return item, err
}
func getByKey(ctx context.Context, q queryer, clientID, workspaceID, campaignID uuid.UUID, key string) (Post, error) {
	return scanPost(q.QueryRow(ctx, postSelect+` WHERE p.client_id=$1 AND p.workspace_id=$2 AND p.campaign_id=$3 AND p.idempotency_key=$4`, clientID, workspaceID, campaignID, key))
}
func postHash(platform string, input Input) string {
	input.Version = 0
	raw, _ := json.Marshal(struct {
		Platform string
		Input    Input
	}{platform, input})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
