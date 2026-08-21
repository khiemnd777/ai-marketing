package brands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalid  = errors.New("invalid brand")
	ErrNotFound = errors.New("brand not found")
	ErrConflict = errors.New("brand version conflict")
)

type Profile struct {
	ID                    uuid.UUID   `json:"id"`
	ClientID              uuid.UUID   `json:"clientId"`
	WorkspaceID           uuid.UUID   `json:"workspaceId"`
	Name                  string      `json:"name"`
	Status                string      `json:"status"`
	CurrentVersion        int32       `json:"currentVersion"`
	Version               int64       `json:"version"`
	LogoAssetIDs          []uuid.UUID `json:"logoAssetIds"`
	PrimaryColor          *string     `json:"primaryColor"`
	SecondaryColor        *string     `json:"secondaryColor"`
	BackgroundColor       *string     `json:"backgroundColor"`
	HeadingFont           *string     `json:"headingFont"`
	BodyFont              *string     `json:"bodyFont"`
	ToneOfVoice           string      `json:"toneOfVoice"`
	PrimaryLanguage       string      `json:"primaryLanguage"`
	TargetAudience        string      `json:"targetAudience"`
	MainMessage           string      `json:"mainMessage"`
	DefaultCTA            string      `json:"defaultCta"`
	Website               *string     `json:"website"`
	PhoneNumber           *string     `json:"phoneNumber"`
	PreferredTerminology  []string    `json:"preferredTerminology"`
	ProhibitedTerminology []string    `json:"prohibitedTerminology"`
	DefaultDisclaimer     string      `json:"defaultDisclaimer"`
	DefaultVideoStyle     string      `json:"defaultVideoStyle"`
	DefaultMusicStyle     string      `json:"defaultMusicStyle"`
	ChangeSummary         string      `json:"changeSummary"`
	CreatedAt             time.Time   `json:"createdAt"`
	UpdatedAt             time.Time   `json:"updatedAt"`
}
type Input struct {
	Name                  string      `json:"name"`
	LogoAssetIDs          []uuid.UUID `json:"logoAssetIds"`
	PrimaryColor          *string     `json:"primaryColor"`
	SecondaryColor        *string     `json:"secondaryColor"`
	BackgroundColor       *string     `json:"backgroundColor"`
	HeadingFont           *string     `json:"headingFont"`
	BodyFont              *string     `json:"bodyFont"`
	ToneOfVoice           string      `json:"toneOfVoice"`
	PrimaryLanguage       string      `json:"primaryLanguage"`
	TargetAudience        string      `json:"targetAudience"`
	MainMessage           string      `json:"mainMessage"`
	DefaultCTA            string      `json:"defaultCta"`
	Website               *string     `json:"website"`
	PhoneNumber           *string     `json:"phoneNumber"`
	PreferredTerminology  []string    `json:"preferredTerminology"`
	ProhibitedTerminology []string    `json:"prohibitedTerminology"`
	DefaultDisclaimer     string      `json:"defaultDisclaimer"`
	DefaultVideoStyle     string      `json:"defaultVideoStyle"`
	DefaultMusicStyle     string      `json:"defaultMusicStyle"`
	ChangeSummary         string      `json:"changeSummary"`
	Version               int64       `json:"version"`
}
type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

const columns = `b.id,b.client_id,b.workspace_id,b.name,b.status::text,b.current_version,b.version,v.logo_asset_ids,v.primary_color,v.secondary_color,v.background_color,v.heading_font,v.body_font,v.tone_of_voice,v.primary_language,v.target_audience,v.main_message,v.default_cta,v.website,v.phone_number,v.preferred_terminology,v.prohibited_terminology,v.default_disclaimer,v.default_video_style,v.default_music_style,v.change_summary,b.created_at,b.updated_at`

func (s *Service) List(ctx context.Context, clientID, workspaceID uuid.UUID) ([]Profile, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+columns+` FROM brands b JOIN brand_versions v ON v.brand_id=b.id AND v.version=b.current_version WHERE b.client_id=$1 AND b.workspace_id=$2 ORDER BY b.status,b.name`, clientID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Profile{}
	for rows.Next() {
		i, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
func (s *Service) Get(ctx context.Context, clientID, workspaceID, id uuid.UUID) (Profile, error) {
	return scan(s.pool.QueryRow(ctx, `SELECT `+columns+` FROM brands b JOIN brand_versions v ON v.brand_id=b.id AND v.version=b.current_version WHERE b.client_id=$1 AND b.workspace_id=$2 AND b.id=$3`, clientID, workspaceID, id))
}
func (s *Service) Create(ctx context.Context, clientID, workspaceID, actorID uuid.UUID, input Input) (Profile, error) {
	if err := validate(&input, false); err != nil {
		return Profile{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Profile{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id := uuid.New()
	if err := validateLogoAssets(ctx, tx, clientID, workspaceID, id, input.LogoAssetIDs); err != nil {
		return Profile{}, err
	}
	err = tx.QueryRow(ctx, `INSERT INTO brands(id,client_id,workspace_id,name,created_by,updated_by) SELECT $1,$2,$3,$4,$5,$5 WHERE EXISTS(SELECT 1 FROM workspaces WHERE id=$3 AND client_id=$2 AND status='ACTIVE') RETURNING id`, id, clientID, workspaceID, input.Name, actorID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	if err := insertVersion(ctx, tx, id, clientID, workspaceID, 1, actorID, input); err != nil {
		return Profile{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Profile{}, err
	}
	return s.Get(ctx, clientID, workspaceID, id)
}
func (s *Service) Update(ctx context.Context, clientID, workspaceID, id, actorID uuid.UUID, input Input) (Profile, error) {
	if err := validate(&input, true); err != nil {
		return Profile{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Profile{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var previousLogoAssetIDs []uuid.UUID
	err = tx.QueryRow(ctx, `SELECT v.logo_asset_ids FROM brands b JOIN brand_versions v ON v.brand_id=b.id AND v.version=b.current_version WHERE b.id=$1 AND b.client_id=$2 AND b.workspace_id=$3 AND b.version=$4 FOR UPDATE OF b`, id, clientID, workspaceID, input.Version).Scan(&previousLogoAssetIDs)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrConflict
	}
	if err != nil {
		return Profile{}, err
	}
	if err := validateLogoAssets(ctx, tx, clientID, workspaceID, id, input.LogoAssetIDs); err != nil {
		return Profile{}, err
	}
	var next int32
	err = tx.QueryRow(ctx, `UPDATE brands SET name=$4,current_version=current_version+1,version=version+1,updated_by=$5,updated_at=now() WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND version=$6 RETURNING current_version`, id, clientID, workspaceID, input.Name, actorID, input.Version).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrConflict
	}
	if err != nil {
		return Profile{}, err
	}
	if err := insertVersion(ctx, tx, id, clientID, workspaceID, next, actorID, input); err != nil {
		return Profile{}, err
	}
	if !sameUUIDs(previousLogoAssetIDs, input.LogoAssetIDs) {
		if err := invalidateLogoApprovals(ctx, tx, id, actorID); err != nil {
			return Profile{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Profile{}, err
	}
	return s.Get(ctx, clientID, workspaceID, id)
}
func insertVersion(ctx context.Context, tx pgx.Tx, id, clientID, workspaceID uuid.UUID, version int32, actorID uuid.UUID, i Input) error {
	_, err := tx.Exec(ctx, `INSERT INTO brand_versions(brand_id,client_id,workspace_id,version,logo_asset_ids,primary_color,secondary_color,background_color,heading_font,body_font,tone_of_voice,primary_language,target_audience,main_message,default_cta,website,phone_number,preferred_terminology,prohibited_terminology,default_disclaimer,default_video_style,default_music_style,change_summary,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)`, id, clientID, workspaceID, version, i.LogoAssetIDs, i.PrimaryColor, i.SecondaryColor, i.BackgroundColor, i.HeadingFont, i.BodyFont, i.ToneOfVoice, i.PrimaryLanguage, i.TargetAudience, i.MainMessage, i.DefaultCTA, i.Website, i.PhoneNumber, i.PreferredTerminology, i.ProhibitedTerminology, i.DefaultDisclaimer, i.DefaultVideoStyle, i.DefaultMusicStyle, i.ChangeSummary, actorID)
	return err
}
func validateLogoAssets(ctx context.Context, tx pgx.Tx, clientID, workspaceID, brandID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	var valid bool
	if err := tx.QueryRow(ctx, `SELECT count(*) = cardinality($3::uuid[])
		FROM media_assets a
		JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version
		WHERE a.id=ANY($3::uuid[]) AND a.client_id=$1 AND a.workspace_id=$2
		AND (a.brand_id IS NULL OR a.brand_id=$4)
		AND a.product_id IS NULL AND a.campaign_id IS NULL
		AND a.asset_type IN('IMAGE','LOGO') AND a.status='APPROVED'
		AND (a.expires_at IS NULL OR a.expires_at>now()) AND a.deleted_at IS NULL
		AND v.mime_type IN('image/jpeg','image/png','image/webp')
		AND v.verified_at IS NOT NULL AND COALESCE(v.checksum_sha256,'')<>''`, clientID, workspaceID, ids, brandID).Scan(&valid); err != nil {
		return err
	}
	if !valid {
		return ErrInvalid
	}
	return nil
}

func sameUUIDs(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func invalidateLogoApprovals(ctx context.Context, tx pgx.Tx, brandID, actorID uuid.UUID) error {
	const reason = "Brand logo selection changed"
	if _, err := tx.Exec(ctx, `WITH affected_campaigns AS (
		SELECT id FROM campaigns WHERE brand_id=$1
	), changed AS (
		UPDATE approvals a SET invalidated_at=now(),invalidation_reason=$3
		FROM affected_campaigns c
		WHERE a.campaign_id=c.id AND a.entity_type='FINAL_RENDER' AND a.invalidated_at IS NULL
		RETURNING a.id,a.entity_version,a.entity_hash
	) INSERT INTO approval_events(approval_id,event_type,actor_id,entity_version,entity_hash,notes)
	SELECT id,'INVALIDATED',$2,entity_version,entity_hash,$3 FROM changed`, brandID, actorID, reason); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE video_projects p SET selected_render_job_id=NULL,version=p.version+1,updated_by=$2,updated_at=now()
		FROM campaigns c WHERE p.campaign_id=c.id AND c.brand_id=$1 AND p.selected_render_job_id IS NOT NULL`, brandID, actorID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE campaigns SET status='SCENE_REVIEW',version=version+1,updated_by=$2,updated_at=now()
		WHERE brand_id=$1 AND status IN('FINAL_RENDERING','FINAL_REVIEW','APPROVED','READY_TO_PUBLISH')`, brandID, actorID)
	return err
}
func (s *Service) SetStatus(ctx context.Context, clientID, workspaceID, id uuid.UUID, status string, version int64, actorID uuid.UUID) (Profile, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if (status != "ACTIVE" && status != "ARCHIVED") || version < 1 {
		return Profile{}, ErrInvalid
	}
	tag, err := s.pool.Exec(ctx, `UPDATE brands SET status=$4::lifecycle_status,archived_at=CASE WHEN $4='ARCHIVED' THEN now() ELSE NULL END,version=version+1,updated_by=$5,updated_at=now() WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND version=$6`, id, clientID, workspaceID, status, actorID, version)
	if err != nil {
		return Profile{}, err
	}
	if tag.RowsAffected() != 1 {
		return Profile{}, ErrConflict
	}
	return s.Get(ctx, clientID, workspaceID, id)
}

type scanner interface{ Scan(...any) error }

func scan(r scanner) (Profile, error) {
	var i Profile
	err := r.Scan(&i.ID, &i.ClientID, &i.WorkspaceID, &i.Name, &i.Status, &i.CurrentVersion, &i.Version, &i.LogoAssetIDs, &i.PrimaryColor, &i.SecondaryColor, &i.BackgroundColor, &i.HeadingFont, &i.BodyFont, &i.ToneOfVoice, &i.PrimaryLanguage, &i.TargetAudience, &i.MainMessage, &i.DefaultCTA, &i.Website, &i.PhoneNumber, &i.PreferredTerminology, &i.ProhibitedTerminology, &i.DefaultDisclaimer, &i.DefaultVideoStyle, &i.DefaultMusicStyle, &i.ChangeSummary, &i.CreatedAt, &i.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	return i, err
}
func validate(i *Input, version bool) error {
	i.Name = strings.TrimSpace(i.Name)
	i.PrimaryLanguage = strings.ToLower(strings.TrimSpace(i.PrimaryLanguage))
	if len(i.Name) < 2 || len(i.Name) > 160 || (i.PrimaryLanguage != "vi" && i.PrimaryLanguage != "en") || (version && i.Version < 1) || len(i.LogoAssetIDs) > 20 || len(i.PreferredTerminology) > 200 || len(i.ProhibitedTerminology) > 200 {
		return ErrInvalid
	}
	if i.LogoAssetIDs == nil {
		i.LogoAssetIDs = []uuid.UUID{}
	}
	if i.PreferredTerminology == nil {
		i.PreferredTerminology = []string{}
	}
	if i.ProhibitedTerminology == nil {
		i.ProhibitedTerminology = []string{}
	}
	for _, c := range []*string{i.PrimaryColor, i.SecondaryColor, i.BackgroundColor} {
		if c != nil && !isHex(*c) {
			return ErrInvalid
		}
	}
	return nil
}
func isHex(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, r := range s[1:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

var _ = fmt.Sprintf
