package rendering

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/jobs"
)

var (
	ErrInvalid      = errors.New("invalid rendering operation")
	ErrNotFound     = errors.New("rendering resource not found")
	ErrConflict     = errors.New("rendering conflict")
	ErrPrerequisite = errors.New("rendering prerequisite not met")
)

type Service struct {
	pool     *pgxpool.Pool
	enqueuer *jobs.Enqueuer
}

func NewService(pool *pgxpool.Pool, enqueuer *jobs.Enqueuer) *Service {
	return &Service{pool: pool, enqueuer: enqueuer}
}

func (s *Service) GetProject(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID) (Project, error) {
	return getProject(ctx, s.pool, clientID, workspaceID, campaignID)
}

func (s *Service) SaveProject(ctx context.Context, clientID, workspaceID, campaignID, actorID uuid.UUID, input ProjectInput) (Project, error) {
	input.Headline, input.LowerThird, input.ChangeSummary = strings.TrimSpace(input.Headline), strings.TrimSpace(input.LowerThird), strings.TrimSpace(input.ChangeSummary)
	if input.Headline == "" || len(input.Headline) > 300 || input.MusicGainDB < -60 || input.MusicGainDB > 0 || input.DialogueDuckingDB < -30 || input.DialogueDuckingDB > 0 {
		return Project{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Project{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if input.MusicAssetID != nil {
		var valid bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM media_assets WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND asset_type='AUDIO' AND deleted_at IS NULL)`, *input.MusicAssetID, clientID, workspaceID).Scan(&valid); err != nil {
			return Project{}, err
		}
		if !valid {
			return Project{}, ErrInvalid
		}
	}
	project, err := getProject(ctx, tx, clientID, workspaceID, campaignID)
	if errors.Is(err, ErrNotFound) {
		project.ID = uuid.New()
		project.CurrentVersion = 1
		project.Version = 1
		if _, err = tx.Exec(ctx, `INSERT INTO video_projects(id,client_id,workspace_id,campaign_id,music_asset_id,music_gain_db,dialogue_ducking_db,created_by,updated_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8)`, project.ID, clientID, workspaceID, campaignID, input.MusicAssetID, input.MusicGainDB, input.DialogueDuckingDB, actorID); err != nil {
			return Project{}, err
		}
	} else if err != nil {
		return Project{}, err
	} else {
		if input.Version != project.Version {
			return Project{}, ErrConflict
		}
		project.CurrentVersion++
		if _, err = tx.Exec(ctx, `UPDATE video_projects SET current_version=$2,music_asset_id=$3,music_gain_db=$4,dialogue_ducking_db=$5,selected_render_job_id=NULL,version=version+1,updated_by=$6,updated_at=now() WHERE id=$1 AND version=$7`, project.ID, project.CurrentVersion, input.MusicAssetID, input.MusicGainDB, input.DialogueDuckingDB, actorID, input.Version); err != nil {
			return Project{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE approvals SET invalidated_at=now(),invalidation_reason='Final composition settings changed' WHERE campaign_id=$1 AND entity_type='FINAL_RENDER' AND invalidated_at IS NULL`, campaignID); err != nil {
			return Project{}, err
		}
	}
	hash, raw := hashProject(input)
	_, err = tx.Exec(ctx, `INSERT INTO video_project_versions(video_project_id,version,headline,lower_third,show_price,show_discount_code,show_cta,show_website,show_phone,show_qr_code,show_disclaimer,burn_captions,project_hash,change_summary,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`, project.ID, project.CurrentVersion, input.Headline, input.LowerThird, input.ShowPrice, input.ShowDiscountCode, input.ShowCTA, input.ShowWebsite, input.ShowPhone, input.ShowQRCode, input.ShowDisclaimer, input.BurnCaptions, hash, input.ChangeSummary, actorID)
	if err != nil {
		return Project{}, err
	}
	_ = raw
	if err = tx.Commit(ctx); err != nil {
		return Project{}, err
	}
	return s.GetProject(ctx, clientID, workspaceID, campaignID)
}

func (s *Service) Start(ctx context.Context, clientID, workspaceID, campaignID, actorID uuid.UUID, idempotencyKey string) (RenderJob, error) {
	if len(strings.TrimSpace(idempotencyKey)) < 16 {
		return RenderJob{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RenderJob{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	project, err := getProject(ctx, tx, clientID, workspaceID, campaignID)
	if errors.Is(err, ErrNotFound) {
		var name string
		if err = tx.QueryRow(ctx, `SELECT p.name FROM campaigns c JOIN products p ON p.id=c.product_id WHERE c.id=$1 AND c.client_id=$2 AND c.workspace_id=$3`, campaignID, clientID, workspaceID).Scan(&name); err != nil {
			return RenderJob{}, ErrNotFound
		}
		input := ProjectInput{Headline: name, ShowPrice: true, ShowDiscountCode: true, ShowCTA: true, ShowWebsite: true, ShowPhone: true, ShowQRCode: true, ShowDisclaimer: true, BurnCaptions: true, MusicGainDB: -18, DialogueDuckingDB: -9, ChangeSummary: "Default final composition"}
		project.ID, project.CurrentVersion, project.Version = uuid.New(), 1, 1
		if _, err = tx.Exec(ctx, `INSERT INTO video_projects(id,client_id,workspace_id,campaign_id,music_gain_db,dialogue_ducking_db,created_by,updated_by)VALUES($1,$2,$3,$4,$5,$6,$7,$7)`, project.ID, clientID, workspaceID, campaignID, input.MusicGainDB, input.DialogueDuckingDB, actorID); err != nil {
			return RenderJob{}, err
		}
		hash, _ := hashProject(input)
		if _, err = tx.Exec(ctx, `INSERT INTO video_project_versions(video_project_id,version,headline,show_price,show_discount_code,show_cta,show_website,show_phone,show_qr_code,show_disclaimer,burn_captions,project_hash,change_summary,created_by)VALUES($1,1,$2,true,true,true,true,true,true,true,true,$3,$4,$5)`, project.ID, input.Headline, hash, input.ChangeSummary, actorID); err != nil {
			return RenderJob{}, err
		}
		project.ProjectHash = hash
	} else if err != nil {
		return RenderJob{}, err
	}
	var sceneCount, selectedCount, totalDuration, campaignDuration, brandVersion int
	var selectedHash string
	var brandID uuid.UUID
	var logoIDs []uuid.UUID
	err = tx.QueryRow(ctx, `SELECT count(s.id),count(g.id),COALESCE(sum(sv.duration_seconds),0),cv.duration_seconds,COALESCE(string_agg(g.id::text||':'||g.version::text||':'||COALESCE(g.output_asset_id::text,'')||':'||e.version::text,'|' ORDER BY s.scene_order),''),c.brand_id,bv.version,bv.logo_asset_ids FROM campaigns c JOIN campaign_versions cv ON cv.campaign_id=c.id AND cv.version=c.current_version JOIN brands b ON b.id=c.brand_id AND b.client_id=c.client_id AND b.workspace_id=c.workspace_id JOIN brand_versions bv ON bv.brand_id=b.id AND bv.version=b.current_version LEFT JOIN scenes s ON s.campaign_id=c.id LEFT JOIN scene_versions sv ON sv.scene_id=s.id AND sv.version=s.current_version LEFT JOIN scene_generation_tasks g ON g.id=s.selected_generation_task_id AND g.status='APPROVED' AND g.scene_version=s.current_version LEFT JOIN scene_generation_edits e ON e.generation_task_id=g.id WHERE c.id=$1 AND c.client_id=$2 AND c.workspace_id=$3 GROUP BY cv.duration_seconds,c.brand_id,bv.version,bv.logo_asset_ids`, campaignID, clientID, workspaceID).Scan(&sceneCount, &selectedCount, &totalDuration, &campaignDuration, &selectedHash, &brandID, &brandVersion, &logoIDs)
	if errors.Is(err, pgx.ErrNoRows) {
		return RenderJob{}, ErrNotFound
	}
	if err != nil {
		return RenderJob{}, err
	}
	if sceneCount == 0 || sceneCount != selectedCount || totalDuration != campaignDuration {
		return RenderJob{}, ErrPrerequisite
	}
	logoFingerprint := fmt.Sprintf("brand:%d", brandVersion)
	if len(logoIDs) > 0 {
		var assetVersion int64
		var checksum string
		err = tx.QueryRow(ctx, `SELECT a.version,v.checksum_sha256 FROM media_assets a JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version WHERE a.id=$1 AND a.client_id=$2 AND a.workspace_id=$3 AND (a.brand_id IS NULL OR a.brand_id=$4) AND a.product_id IS NULL AND a.campaign_id IS NULL AND a.asset_type IN('IMAGE','LOGO') AND a.status='APPROVED' AND (a.expires_at IS NULL OR a.expires_at>now()) AND a.deleted_at IS NULL AND v.mime_type IN('image/jpeg','image/png','image/webp') AND v.verified_at IS NOT NULL AND COALESCE(v.checksum_sha256,'')<>''`, logoIDs[0], clientID, workspaceID, brandID).Scan(&assetVersion, &checksum)
		if errors.Is(err, pgx.ErrNoRows) {
			return RenderJob{}, ErrPrerequisite
		}
		if err != nil {
			return RenderJob{}, err
		}
		logoFingerprint += fmt.Sprintf("|%s:%d:%s", logoIDs[0], assetVersion, checksum)
	}
	inputHash := digest(project.ProjectHash + "|" + selectedHash + "|" + logoFingerprint)
	keyHash := digest(idempotencyKey + "|" + inputHash)
	if existing, findErr := getJobByKey(ctx, tx, clientID, workspaceID, campaignID, keyHash); findErr == nil {
		existing.Reused = true
		_ = tx.Commit(ctx)
		return existing, nil
	} else if !errors.Is(findErr, pgx.ErrNoRows) {
		return RenderJob{}, findErr
	}
	id := uuid.New()
	_, err = tx.Exec(ctx, `INSERT INTO render_jobs(id,client_id,workspace_id,campaign_id,video_project_id,video_project_version,status,idempotency_key,created_by)VALUES($1,$2,$3,$4,$5,$6,'QUEUED',$7,$8)`, id, clientID, workspaceID, campaignID, project.ID, project.CurrentVersion, keyHash, actorID)
	if err != nil {
		return RenderJob{}, err
	}
	riverID, err := s.enqueuer.EnqueueFinalRender(ctx, tx, id)
	if err != nil {
		return RenderJob{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE render_jobs SET river_job_id=$2 WHERE id=$1`, id, riverID); err != nil {
		return RenderJob{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE campaigns SET status='FINAL_RENDERING',version=version+1,updated_by=$2,updated_at=now() WHERE id=$1`, campaignID, actorID); err != nil {
		return RenderJob{}, err
	}
	item, err := getJob(ctx, tx, clientID, workspaceID, campaignID, id)
	if err != nil {
		return RenderJob{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return RenderJob{}, err
	}
	return item, nil
}

func (s *Service) List(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID) ([]RenderJob, error) {
	rows, err := s.pool.Query(ctx, jobSelect+` WHERE r.client_id=$1 AND r.workspace_id=$2 AND r.campaign_id=$3 ORDER BY r.created_at DESC`, clientID, workspaceID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []RenderJob{}
	for rows.Next() {
		item, scanErr := scanJob(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) Review(ctx context.Context, clientID, workspaceID, campaignID, jobID, actorID uuid.UUID, input ReviewInput) (RenderJob, error) {
	action := strings.ToUpper(strings.TrimSpace(input.Action))
	if (action != "APPROVE" && action != "REJECT") || input.Version < 1 || (action == "REJECT" && strings.TrimSpace(input.Notes) == "") {
		return RenderJob{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RenderJob{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status, hash string
	var version int64
	err = tx.QueryRow(ctx, `SELECT status::text,output_hash,version FROM render_jobs WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND campaign_id=$4 FOR UPDATE`, jobID, clientID, workspaceID, campaignID).Scan(&status, &hash, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return RenderJob{}, ErrNotFound
	}
	if err != nil {
		return RenderJob{}, err
	}
	if status != "REVIEW_REQUIRED" || version != input.Version {
		return RenderJob{}, ErrConflict
	}
	newStatus := "REJECTED"
	if action == "APPROVE" {
		newStatus = "APPROVED"
	}
	_, err = tx.Exec(ctx, `UPDATE render_jobs SET status=$2::render_job_status,reviewed_at=now(),reviewed_by=$3,review_notes=$4,version=version+1,updated_at=now() WHERE id=$1`, jobID, newStatus, actorID, strings.TrimSpace(input.Notes))
	if err != nil {
		return RenderJob{}, err
	}
	var approvalID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO approvals(client_id,workspace_id,campaign_id,entity_type,entity_id,entity_version,entity_hash,status,requested_by,decided_by,decided_at,notes)VALUES($1,$2,$3,'FINAL_RENDER',$4,$5,$6,$7,$8,$8,now(),$9)RETURNING id`, clientID, workspaceID, campaignID, jobID, version, hash, newStatus, actorID, input.Notes).Scan(&approvalID)
	if err != nil {
		return RenderJob{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO approval_events(approval_id,event_type,actor_id,entity_version,entity_hash,notes)VALUES($1,$2::approval_event_type,$3,$4,$5,$6)`, approvalID, newStatus, actorID, version, hash, input.Notes)
	if err != nil {
		return RenderJob{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return RenderJob{}, err
	}
	return getJob(ctx, s.pool, clientID, workspaceID, campaignID, jobID)
}

func (s *Service) Select(ctx context.Context, clientID, workspaceID, campaignID, jobID, actorID uuid.UUID) (RenderJob, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RenderJob{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := tx.Exec(ctx, `UPDATE video_projects p SET selected_render_job_id=$1,version=p.version+1,updated_by=$2,updated_at=now() FROM render_jobs r WHERE r.id=$1 AND r.video_project_id=p.id AND r.status='APPROVED' AND r.client_id=$3 AND r.workspace_id=$4 AND r.campaign_id=$5`, jobID, actorID, clientID, workspaceID, campaignID)
	if err != nil {
		return RenderJob{}, err
	}
	if result.RowsAffected() != 1 {
		return RenderJob{}, ErrPrerequisite
	}
	if _, err = tx.Exec(ctx, `UPDATE campaigns SET status='APPROVED',version=version+1,updated_by=$2,updated_at=now() WHERE id=$1`, campaignID, actorID); err != nil {
		return RenderJob{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return RenderJob{}, err
	}
	return getJob(ctx, s.pool, clientID, workspaceID, campaignID, jobID)
}

const projectSelect = `SELECT p.id,p.campaign_id,p.current_version,v.project_hash,p.selected_render_job_id,v.headline,v.lower_third,v.show_price,v.show_discount_code,v.show_cta,v.show_website,v.show_phone,v.show_qr_code,v.show_disclaimer,v.burn_captions,p.music_asset_id,p.music_gain_db::float8,p.dialogue_ducking_db::float8,v.change_summary,p.version,p.updated_at FROM video_projects p JOIN video_project_versions v ON v.video_project_id=p.id AND v.version=p.current_version`
const jobSelect = `SELECT r.id,r.campaign_id,r.video_project_id,r.video_project_version,r.render_manifest_id,r.status::text,r.output_asset_id,r.output_hash,r.thumbnail_storage_key,r.srt_storage_key,r.vtt_storage_key,r.renderer_request_id,r.error_code,r.error_message,r.sanitized_response,r.review_notes,r.version,r.created_at,r.updated_at,COALESCE(p.selected_render_job_id=r.id,false) FROM render_jobs r JOIN video_projects p ON p.id=r.video_project_id`

type rowScanner interface{ Scan(...any) error }
type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func getProject(ctx context.Context, q queryer, clientID, workspaceID, campaignID uuid.UUID) (Project, error) {
	var p Project
	err := q.QueryRow(ctx, projectSelect+` WHERE p.client_id=$1 AND p.workspace_id=$2 AND p.campaign_id=$3`, clientID, workspaceID, campaignID).Scan(&p.ID, &p.CampaignID, &p.CurrentVersion, &p.ProjectHash, &p.SelectedJobID, &p.Headline, &p.LowerThird, &p.ShowPrice, &p.ShowDiscountCode, &p.ShowCTA, &p.ShowWebsite, &p.ShowPhone, &p.ShowQRCode, &p.ShowDisclaimer, &p.BurnCaptions, &p.MusicAssetID, &p.MusicGainDB, &p.DialogueDuckingDB, &p.ChangeSummary, &p.Version, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	return p, err
}
func scanJob(r rowScanner) (RenderJob, error) {
	var j RenderJob
	err := r.Scan(&j.ID, &j.CampaignID, &j.VideoProjectID, &j.VideoProjectVersion, &j.RenderManifestID, &j.Status, &j.OutputAssetID, &j.OutputHash, &j.ThumbnailStorageKey, &j.SRTStorageKey, &j.VTTStorageKey, &j.RendererRequestID, &j.ErrorCode, &j.ErrorMessage, &j.SanitizedResponse, &j.ReviewNotes, &j.Version, &j.CreatedAt, &j.UpdatedAt, &j.Selected)
	return j, err
}
func getJob(ctx context.Context, q queryer, clientID, workspaceID, campaignID, id uuid.UUID) (RenderJob, error) {
	j, err := scanJob(q.QueryRow(ctx, jobSelect+` WHERE r.id=$1 AND r.client_id=$2 AND r.workspace_id=$3 AND r.campaign_id=$4`, id, clientID, workspaceID, campaignID))
	if errors.Is(err, pgx.ErrNoRows) {
		return RenderJob{}, ErrNotFound
	}
	return j, err
}
func getJobByKey(ctx context.Context, q queryer, clientID, workspaceID, campaignID uuid.UUID, key string) (RenderJob, error) {
	return scanJob(q.QueryRow(ctx, jobSelect+` WHERE r.client_id=$1 AND r.workspace_id=$2 AND r.campaign_id=$3 AND r.idempotency_key=$4`, clientID, workspaceID, campaignID, key))
}
func hashProject(input ProjectInput) (string, []byte) {
	raw, _ := json.Marshal(input)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), raw
}
func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
