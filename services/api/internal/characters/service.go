package characters

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalid  = errors.New("invalid character")
	ErrNotFound = errors.New("character not found")
	ErrConflict = errors.New("character conflict")
)

type Character struct {
	ID                    uuid.UUID  `json:"id"`
	ClientID              *uuid.UUID `json:"clientId"`
	WorkspaceID           *uuid.UUID `json:"workspaceId"`
	Name                  string     `json:"name"`
	Provider              string     `json:"provider"`
	ProviderAssetID       *string    `json:"providerAssetId"`
	CharacterType         string     `json:"characterType"`
	GenderPresentation    string     `json:"genderPresentation"`
	ApproximateAgeRange   string     `json:"approximateAgeRange"`
	AppearanceDescription string     `json:"appearanceDescription"`
	Wardrobe              string     `json:"wardrobe"`
	GestureStyle          string     `json:"gestureStyle"`
	DefaultRole           string     `json:"defaultRole"`
	SupportedLanguages    []string   `json:"supportedLanguages"`
	ConsentStatus         string     `json:"consentStatus"`
	PreviewAssetID        *uuid.UUID `json:"previewAssetId"`
	Status                string     `json:"status"`
	Version               int64      `json:"version"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

type Input struct {
	Name                  string     `json:"name"`
	Provider              string     `json:"provider"`
	ProviderAssetID       *string    `json:"providerAssetId"`
	CharacterType         string     `json:"characterType"`
	GenderPresentation    string     `json:"genderPresentation"`
	ApproximateAgeRange   string     `json:"approximateAgeRange"`
	AppearanceDescription string     `json:"appearanceDescription"`
	Wardrobe              string     `json:"wardrobe"`
	GestureStyle          string     `json:"gestureStyle"`
	DefaultRole           string     `json:"defaultRole"`
	SupportedLanguages    []string   `json:"supportedLanguages"`
	ConsentStatus         string     `json:"consentStatus"`
	PreviewAssetID        *uuid.UUID `json:"previewAssetId"`
}

type Selection struct {
	Primary  Character `json:"primary"`
	Listener Character `json:"listener"`
}

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

const characterColumns = `id,client_id,workspace_id,name,provider,provider_asset_id,character_type::text,gender_presentation,approximate_age_range,appearance_description,wardrobe,gesture_style,default_role,supported_languages,consent_status::text,preview_asset_id,status::text,version,created_at,updated_at`

func (s *Service) List(ctx context.Context, clientID, workspaceID uuid.UUID) ([]Character, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+characterColumns+` FROM characters WHERE status='ACTIVE' AND ((client_id IS NULL AND workspace_id IS NULL) OR (client_id=$1 AND workspace_id=$2)) ORDER BY character_type,name`, clientID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Character{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) Create(ctx context.Context, clientID, workspaceID, actorID uuid.UUID, input Input) (Character, error) {
	if err := validate(&input); err != nil {
		return Character{}, err
	}
	item, err := scan(s.pool.QueryRow(ctx, `INSERT INTO characters(client_id,workspace_id,name,provider,provider_asset_id,character_type,gender_presentation,approximate_age_range,appearance_description,wardrobe,gesture_style,default_role,supported_languages,consent_status,preview_asset_id,created_by,updated_by) SELECT $1,$2,$3,$4,$5,$6::character_type,$7,$8,$9,$10,$11,$12,$13,$14::consent_status,$15,$16,$16 WHERE EXISTS(SELECT 1 FROM workspaces WHERE id=$2 AND client_id=$1 AND status='ACTIVE') AND ($15::uuid IS NULL OR EXISTS(SELECT 1 FROM media_assets WHERE id=$15 AND client_id=$1 AND workspace_id=$2 AND asset_type IN('IMAGE','LOGO') AND deleted_at IS NULL)) RETURNING `+characterColumns, clientID, workspaceID, input.Name, input.Provider, input.ProviderAssetID, input.CharacterType, input.GenderPresentation, input.ApproximateAgeRange, input.AppearanceDescription, input.Wardrobe, input.GestureStyle, input.DefaultRole, input.SupportedLanguages, input.ConsentStatus, input.PreviewAssetID, actorID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Character{}, ErrNotFound
	}
	return item, err
}

func (s *Service) Select(ctx context.Context, clientID, workspaceID, campaignID, primaryID, listenerID, actorID uuid.UUID) (Selection, error) {
	if primaryID == uuid.Nil || listenerID == uuid.Nil || primaryID == listenerID {
		return Selection{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Selection{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var count int
	err = tx.QueryRow(ctx, `SELECT count(*) FROM characters WHERE id=ANY($1::uuid[]) AND status='ACTIVE' AND ((client_id IS NULL AND workspace_id IS NULL) OR (client_id=$2 AND workspace_id=$3)) AND ((character_type='AUTHORIZED_REAL_PERSON' AND consent_status='APPROVED') OR (character_type<>'AUTHORIZED_REAL_PERSON' AND consent_status IN('NOT_REQUIRED','APPROVED')))`, []uuid.UUID{primaryID, listenerID}, clientID, workspaceID).Scan(&count)
	if err != nil {
		return Selection{}, err
	}
	if count != 2 {
		return Selection{}, ErrInvalid
	}
	var campaignExists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM campaigns WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND status<>'ARCHIVED')`, campaignID, clientID, workspaceID).Scan(&campaignExists); err != nil || !campaignExists {
		return Selection{}, ErrNotFound
	}
	if _, err = tx.Exec(ctx, `DELETE FROM campaign_characters WHERE campaign_id=$1`, campaignID); err != nil {
		return Selection{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO campaign_characters(campaign_id,character_id,role,selected_by) VALUES($1,$2,'PRIMARY',$4),($1,$3,'LISTENER',$4)`, campaignID, primaryID, listenerID, actorID); err != nil {
		return Selection{}, err
	}
	if _, err = tx.Exec(ctx, `WITH changed AS (
		UPDATE approvals SET invalidated_at=now(),invalidation_reason='Character selection changed'
		WHERE campaign_id=$1 AND entity_type IN('SCRIPT','SCENE') AND invalidated_at IS NULL
		RETURNING id,entity_version,entity_hash
	) INSERT INTO approval_events(approval_id,event_type,actor_id,entity_version,entity_hash,notes)
	SELECT id,'INVALIDATED',$2,entity_version,entity_hash,'Character selection changed' FROM changed`, campaignID, actorID); err != nil {
		return Selection{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE scripts SET status='DRAFT',approved_at=NULL,approved_by=NULL,version=version+1,updated_by=$2,updated_at=now() WHERE campaign_id=$1 AND status='APPROVED'`, campaignID, actorID); err != nil {
		return Selection{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE scenes SET status='DRAFT',approved_at=NULL,approved_by=NULL,version=version+1,updated_by=$2,updated_at=now() WHERE campaign_id=$1 AND status='APPROVED'`, campaignID, actorID); err != nil {
		return Selection{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Selection{}, err
	}
	return s.GetSelection(ctx, clientID, workspaceID, campaignID)
}

func (s *Service) GetSelection(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID) (Selection, error) {
	rows, err := s.pool.Query(ctx, `SELECT cc.role,characters.id,characters.client_id,characters.workspace_id,characters.name,characters.provider,characters.provider_asset_id,characters.character_type::text,characters.gender_presentation,characters.approximate_age_range,characters.appearance_description,characters.wardrobe,characters.gesture_style,characters.default_role,characters.supported_languages,characters.consent_status::text,characters.preview_asset_id,characters.status::text,characters.version,characters.created_at,characters.updated_at FROM campaign_characters cc JOIN characters ON characters.id=cc.character_id JOIN campaigns c ON c.id=cc.campaign_id WHERE cc.campaign_id=$1 AND c.client_id=$2 AND c.workspace_id=$3 ORDER BY cc.role`, campaignID, clientID, workspaceID)
	if err != nil {
		return Selection{}, err
	}
	defer rows.Close()
	selection, found := Selection{}, 0
	for rows.Next() {
		var role string
		var item Character
		if err = rows.Scan(&role, &item.ID, &item.ClientID, &item.WorkspaceID, &item.Name, &item.Provider, &item.ProviderAssetID, &item.CharacterType, &item.GenderPresentation, &item.ApproximateAgeRange, &item.AppearanceDescription, &item.Wardrobe, &item.GestureStyle, &item.DefaultRole, &item.SupportedLanguages, &item.ConsentStatus, &item.PreviewAssetID, &item.Status, &item.Version, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return Selection{}, err
		}
		if role == "PRIMARY" {
			selection.Primary = item
		} else {
			selection.Listener = item
		}
		found++
	}
	if found != 2 {
		return Selection{}, ErrNotFound
	}
	return selection, rows.Err()
}

func validate(input *Input) error {
	input.Name, input.Provider = strings.TrimSpace(input.Name), strings.TrimSpace(input.Provider)
	input.CharacterType, input.ConsentStatus = strings.ToUpper(strings.TrimSpace(input.CharacterType)), strings.ToUpper(strings.TrimSpace(input.ConsentStatus))
	input.AppearanceDescription = strings.TrimSpace(input.AppearanceDescription)
	if len(input.Name) < 2 || input.Provider == "" || input.AppearanceDescription == "" || !map[string]bool{"PRESET": true, "TRUSTED_GENERATED": true, "AUTHORIZED_REAL_PERSON": true}[input.CharacterType] || !map[string]bool{"NOT_REQUIRED": true, "PENDING": true, "APPROVED": true, "REVOKED": true, "EXPIRED": true}[input.ConsentStatus] {
		return ErrInvalid
	}
	if input.CharacterType == "PRESET" && input.ConsentStatus != "NOT_REQUIRED" || input.CharacterType == "AUTHORIZED_REAL_PERSON" && input.ConsentStatus == "NOT_REQUIRED" {
		return ErrInvalid
	}
	if len(input.SupportedLanguages) == 0 {
		return ErrInvalid
	}
	for index, language := range input.SupportedLanguages {
		language = strings.ToLower(strings.TrimSpace(language))
		if language != "vi" && language != "en" {
			return ErrInvalid
		}
		input.SupportedLanguages[index] = language
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (Character, error) {
	var item Character
	err := row.Scan(&item.ID, &item.ClientID, &item.WorkspaceID, &item.Name, &item.Provider, &item.ProviderAssetID, &item.CharacterType, &item.GenderPresentation, &item.ApproximateAgeRange, &item.AppearanceDescription, &item.Wardrobe, &item.GestureStyle, &item.DefaultRole, &item.SupportedLanguages, &item.ConsentStatus, &item.PreviewAssetID, &item.Status, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Character{}, ErrNotFound
	}
	return item, err
}
