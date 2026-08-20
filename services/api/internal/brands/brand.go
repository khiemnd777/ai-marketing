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
	if err := validateLogoAssets(ctx, tx, clientID, workspaceID, input.LogoAssetIDs); err != nil {
		return Profile{}, err
	}
	var id uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO brands(client_id,workspace_id,name,created_by,updated_by) SELECT $1,$2,$3,$4,$4 WHERE EXISTS(SELECT 1 FROM workspaces WHERE id=$2 AND client_id=$1 AND status='ACTIVE') RETURNING id`, clientID, workspaceID, input.Name, actorID).Scan(&id)
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
	if err := validateLogoAssets(ctx, tx, clientID, workspaceID, input.LogoAssetIDs); err != nil {
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
	if err := tx.Commit(ctx); err != nil {
		return Profile{}, err
	}
	return s.Get(ctx, clientID, workspaceID, id)
}
func insertVersion(ctx context.Context, tx pgx.Tx, id, clientID, workspaceID uuid.UUID, version int32, actorID uuid.UUID, i Input) error {
	_, err := tx.Exec(ctx, `INSERT INTO brand_versions(brand_id,client_id,workspace_id,version,logo_asset_ids,primary_color,secondary_color,background_color,heading_font,body_font,tone_of_voice,primary_language,target_audience,main_message,default_cta,website,phone_number,preferred_terminology,prohibited_terminology,default_disclaimer,default_video_style,default_music_style,change_summary,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)`, id, clientID, workspaceID, version, i.LogoAssetIDs, i.PrimaryColor, i.SecondaryColor, i.BackgroundColor, i.HeadingFont, i.BodyFont, i.ToneOfVoice, i.PrimaryLanguage, i.TargetAudience, i.MainMessage, i.DefaultCTA, i.Website, i.PhoneNumber, i.PreferredTerminology, i.ProhibitedTerminology, i.DefaultDisclaimer, i.DefaultVideoStyle, i.DefaultMusicStyle, i.ChangeSummary, actorID)
	return err
}
func validateLogoAssets(ctx context.Context, tx pgx.Tx, clientID, workspaceID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	var valid bool
	if err := tx.QueryRow(ctx, `SELECT count(*) = cardinality($3::uuid[]) FROM media_assets WHERE id=ANY($3::uuid[]) AND client_id=$1 AND workspace_id=$2 AND asset_type IN('IMAGE','LOGO') AND deleted_at IS NULL`, clientID, workspaceID, ids).Scan(&valid); err != nil {
		return err
	}
	if !valid {
		return ErrInvalid
	}
	return nil
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
	if len(i.Name) < 2 || len(i.Name) > 160 || (i.PrimaryLanguage != "vi" && i.PrimaryLanguage != "en") || (version && i.Version < 1) || len(i.PreferredTerminology) > 200 || len(i.ProhibitedTerminology) > 200 {
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
