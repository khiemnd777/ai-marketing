package producttruth

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/audit"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
)

var (
	ErrInvalid  = errors.New("invalid product truth")
	ErrNotFound = errors.New("product truth not found")
	ErrConflict = errors.New("product truth conflict")
	ErrLocked   = errors.New("locked product truth cannot be edited")
)

type Fact struct {
	ID              uuid.UUID  `json:"id"`
	ProductID       uuid.UUID  `json:"productId"`
	ClientID        uuid.UUID  `json:"clientId"`
	WorkspaceID     uuid.UUID  `json:"workspaceId"`
	FactKey         string     `json:"factKey"`
	Label           string     `json:"label"`
	ExactValue      string     `json:"exactValue"`
	NormalizedValue any        `json:"normalizedValue"`
	Unit            *string    `json:"unit"`
	SourceName      string     `json:"sourceName"`
	SourceExcerpt   string     `json:"sourceExcerpt"`
	SourceAssetID   *uuid.UUID `json:"sourceAssetId"`
	Status          string     `json:"status"`
	LockedValue     bool       `json:"lockedValue"`
	EffectiveFrom   *time.Time `json:"effectiveFrom"`
	ExpiresAt       *time.Time `json:"expiresAt"`
	Version         int64      `json:"version"`
	ApprovedBy      *uuid.UUID `json:"approvedBy"`
	ApprovedAt      *time.Time `json:"approvedAt"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}
type FactInput struct {
	FactKey         string     `json:"factKey"`
	Label           string     `json:"label"`
	ExactValue      string     `json:"exactValue"`
	NormalizedValue any        `json:"normalizedValue"`
	Unit            *string    `json:"unit"`
	SourceName      string     `json:"sourceName"`
	SourceExcerpt   string     `json:"sourceExcerpt"`
	SourceAssetID   *uuid.UUID `json:"sourceAssetId"`
	EffectiveFrom   *time.Time `json:"effectiveFrom"`
	ExpiresAt       *time.Time `json:"expiresAt"`
	ChangeSummary   string     `json:"changeSummary"`
	Version         int64      `json:"version"`
}
type Claim struct {
	ID         uuid.UUID  `json:"id"`
	ProductID  uuid.UUID  `json:"productId"`
	ClaimKind  string     `json:"claimKind"`
	ClaimText  string     `json:"claimText"`
	Rationale  string     `json:"rationale"`
	Status     string     `json:"status"`
	Version    int64      `json:"version"`
	ApprovedAt *time.Time `json:"approvedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}
type ClaimInput struct {
	ClaimKind     string      `json:"claimKind"`
	ClaimText     string      `json:"claimText"`
	Rationale     string      `json:"rationale"`
	FactIDs       []uuid.UUID `json:"factIds"`
	MediaAssetIDs []uuid.UUID `json:"mediaAssetIds"`
	ChangeSummary string      `json:"changeSummary"`
	Version       int64       `json:"version"`
}
type Service struct{ pool *pgxpool.Pool }

func NewService(p *pgxpool.Pool) *Service { return &Service{pool: p} }

const factCols = `id,product_id,client_id,workspace_id,fact_key,label,exact_value,normalized_value,unit,source_name,source_excerpt,source_asset_id,status::text,locked_value,effective_from,expires_at,version,approved_by,approved_at,created_at,updated_at`

func (s *Service) ListFacts(ctx context.Context, clientID, workspaceID, productID uuid.UUID) ([]Fact, error) {
	rows, e := s.pool.Query(ctx, `SELECT `+factCols+` FROM product_facts WHERE client_id=$1 AND workspace_id=$2 AND product_id=$3 ORDER BY fact_key`, clientID, workspaceID, productID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []Fact{}
	for rows.Next() {
		i, e := scanFact(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
func (s *Service) CreateFact(ctx context.Context, clientID, workspaceID, productID, actorID uuid.UUID, i FactInput) (Fact, error) {
	raw, e := validateFactInput(&i, false)
	if e != nil {
		return Fact{}, e
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return Fact{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	item, e := scanFact(tx.QueryRow(ctx, `INSERT INTO product_facts(product_id,client_id,workspace_id,fact_key,label,exact_value,normalized_value,unit,source_name,source_excerpt,source_asset_id,effective_from,expires_at,created_by,updated_by) SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14 WHERE EXISTS(SELECT 1 FROM products WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND status<>'ARCHIVED') AND ($11::uuid IS NULL OR EXISTS(SELECT 1 FROM media_assets WHERE id=$11 AND client_id=$2 AND workspace_id=$3 AND product_id=$1 AND deleted_at IS NULL)) RETURNING `+factCols, productID, clientID, workspaceID, i.FactKey, i.Label, i.ExactValue, raw, i.Unit, i.SourceName, i.SourceExcerpt, i.SourceAssetID, i.EffectiveFrom, i.ExpiresAt, actorID))
	if errors.Is(e, pgx.ErrNoRows) {
		return Fact{}, ErrNotFound
	}
	if e != nil {
		return Fact{}, mapConflict(e)
	}
	if e = insertFactVersion(ctx, tx, item, raw, i.ChangeSummary, actorID); e != nil {
		return Fact{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Fact{}, e
	}
	return item, nil
}
func (s *Service) UpdateFact(ctx context.Context, clientID, workspaceID, productID, id, actorID uuid.UUID, i FactInput) (Fact, error) {
	raw, e := validateFactInput(&i, true)
	if e != nil {
		return Fact{}, e
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return Fact{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var locked bool
	var currentVersion int64
	e = tx.QueryRow(ctx, `SELECT locked_value,version FROM product_facts WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND product_id=$4 FOR UPDATE`, id, clientID, workspaceID, productID).Scan(&locked, &currentVersion)
	if errors.Is(e, pgx.ErrNoRows) {
		return Fact{}, ErrNotFound
	}
	if e != nil {
		return Fact{}, e
	}
	if locked {
		return Fact{}, ErrLocked
	}
	if currentVersion != i.Version {
		return Fact{}, ErrConflict
	}
	item, e := scanFact(tx.QueryRow(ctx, `UPDATE product_facts SET fact_key=$5,label=$6,exact_value=$7,normalized_value=$8,unit=$9,source_name=$10,source_excerpt=$11,source_asset_id=$12,effective_from=$13,expires_at=$14,status='DRAFT',approved_by=NULL,approved_at=NULL,version=version+1,updated_by=$15,updated_at=now() WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND product_id=$4 AND ($12::uuid IS NULL OR EXISTS(SELECT 1 FROM media_assets WHERE id=$12 AND client_id=$2 AND workspace_id=$3 AND product_id=$4 AND deleted_at IS NULL)) RETURNING `+factCols, id, clientID, workspaceID, productID, i.FactKey, i.Label, i.ExactValue, raw, i.Unit, i.SourceName, i.SourceExcerpt, i.SourceAssetID, i.EffectiveFrom, i.ExpiresAt, actorID))
	if e != nil {
		return Fact{}, mapConflict(e)
	}
	if e = insertFactVersion(ctx, tx, item, raw, i.ChangeSummary, actorID); e != nil {
		return Fact{}, e
	}
	if e = invalidateProductApprovals(ctx, tx, clientID, workspaceID, productID, actorID, "Product Truth fact changed"); e != nil {
		return Fact{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Fact{}, e
	}
	return item, nil
}
func (s *Service) ApproveFact(ctx context.Context, clientID, workspaceID, productID, id uuid.UUID, actor auth.Principal, metadata auth.ClientMetadata, lock bool, version int64) (Fact, error) {
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return Fact{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, e := scanFact(tx.QueryRow(ctx, `SELECT `+factCols+` FROM product_facts WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND product_id=$4 FOR UPDATE`, id, clientID, workspaceID, productID))
	if e != nil {
		return Fact{}, e
	}
	if before.Version != version {
		return Fact{}, ErrConflict
	}
	if before.LockedValue {
		return Fact{}, ErrLocked
	}
	item, e := scanFact(tx.QueryRow(ctx, `UPDATE product_facts SET status='APPROVED',locked_value=$5,approved_by=$6,approved_at=now(),version=version+1,updated_by=$6,updated_at=now() WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND product_id=$4 RETURNING `+factCols, id, clientID, workspaceID, productID, lock, actor.UserID))
	if e != nil {
		return Fact{}, e
	}
	raw, e := json.Marshal(item.NormalizedValue)
	if e != nil {
		return Fact{}, e
	}
	if e = insertFactVersion(ctx, tx, item, raw, "Approved Product Truth fact", actor.UserID); e != nil {
		return Fact{}, e
	}
	if e = audit.Record(ctx, db.New(tx), audit.Event{ActorID: uuid.NullUUID{UUID: actor.UserID, Valid: true}, Action: "product_fact.approved", EntityType: "product_fact", EntityID: uuid.NullUUID{UUID: id, Valid: true}, ClientID: uuid.NullUUID{UUID: clientID, Valid: true}, WorkspaceID: uuid.NullUUID{UUID: workspaceID, Valid: true}, RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS", Before: before, After: item, Metadata: map[string]any{"locked": lock, "productId": productID}}); e != nil {
		return Fact{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Fact{}, e
	}
	return item, nil
}
func insertFactVersion(ctx context.Context, tx pgx.Tx, i Fact, raw []byte, summary string, actorID uuid.UUID) error {
	_, e := tx.Exec(ctx, `INSERT INTO product_fact_versions(product_fact_id,version,label,exact_value,normalized_value,unit,source_name,source_excerpt,source_asset_id,locked_value,change_summary,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, i.ID, i.Version, i.Label, i.ExactValue, raw, i.Unit, i.SourceName, i.SourceExcerpt, i.SourceAssetID, i.LockedValue, summary, actorID)
	return e
}
func validateFactInput(i *FactInput, version bool) ([]byte, error) {
	i.FactKey = strings.ToLower(strings.TrimSpace(i.FactKey))
	i.Label = strings.TrimSpace(i.Label)
	i.ExactValue = strings.TrimSpace(i.ExactValue)
	i.SourceName = strings.TrimSpace(i.SourceName)
	if i.FactKey == "" || i.Label == "" || i.SourceName == "" || (version && i.Version < 1) || ValidateFact(i.FactKey, i.ExactValue) != nil || i.ExpiresAt != nil && i.EffectiveFrom != nil && !i.ExpiresAt.After(*i.EffectiveFrom) {
		return nil, ErrInvalid
	}
	raw, e := json.Marshal(i.NormalizedValue)
	if e != nil {
		return nil, ErrInvalid
	}
	return raw, nil
}
func (s *Service) ListClaims(ctx context.Context, clientID, workspaceID, productID uuid.UUID) ([]Claim, error) {
	rows, e := s.pool.Query(ctx, `SELECT id,product_id,claim_kind::text,claim_text,rationale,status::text,version,approved_at,created_at,updated_at FROM product_claims WHERE client_id=$1 AND workspace_id=$2 AND product_id=$3 ORDER BY claim_kind,created_at`, clientID, workspaceID, productID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []Claim{}
	for rows.Next() {
		var i Claim
		if e = rows.Scan(&i.ID, &i.ProductID, &i.ClaimKind, &i.ClaimText, &i.Rationale, &i.Status, &i.Version, &i.ApprovedAt, &i.CreatedAt, &i.UpdatedAt); e != nil {
			return nil, e
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
func (s *Service) CreateClaim(ctx context.Context, clientID, workspaceID, productID, actorID uuid.UUID, i ClaimInput) (Claim, error) {
	if e := validateClaim(&i, false); e != nil {
		return Claim{}, e
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return Claim{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var item Claim
	e = tx.QueryRow(ctx, `INSERT INTO product_claims(product_id,client_id,workspace_id,claim_kind,claim_text,rationale,created_by,updated_by) SELECT $1,$2,$3,$4::claim_kind,$5,$6,$7,$7 WHERE EXISTS(SELECT 1 FROM products WHERE id=$1 AND client_id=$2 AND workspace_id=$3) RETURNING id,product_id,claim_kind::text,claim_text,rationale,status::text,version,approved_at,created_at,updated_at`, productID, clientID, workspaceID, i.ClaimKind, i.ClaimText, i.Rationale, actorID).Scan(&item.ID, &item.ProductID, &item.ClaimKind, &item.ClaimText, &item.Rationale, &item.Status, &item.Version, &item.ApprovedAt, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return Claim{}, ErrNotFound
	}
	if e != nil {
		return Claim{}, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO product_claim_versions(product_claim_id,version,claim_text,rationale,change_summary,created_by)VALUES($1,1,$2,$3,$4,$5)`, item.ID, item.ClaimText, item.Rationale, i.ChangeSummary, actorID)
	if e != nil {
		return Claim{}, e
	}
	for _, id := range i.FactIDs {
		if _, e = tx.Exec(ctx, `INSERT INTO product_claim_sources(claim_id,fact_id,evidence_excerpt) SELECT $1,id,source_excerpt FROM product_facts WHERE id=$2 AND product_id=$3 AND status='APPROVED'`, item.ID, id, productID); e != nil {
			return Claim{}, e
		}
	}
	for _, id := range i.MediaAssetIDs {
		if _, e = tx.Exec(ctx, `INSERT INTO product_claim_sources(claim_id,media_asset_id) SELECT $1,id FROM media_assets WHERE id=$2 AND product_id=$3 AND deleted_at IS NULL`, item.ID, id, productID); e != nil {
			return Claim{}, e
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return Claim{}, e
	}
	return item, nil
}

func (s *Service) UpdateClaim(ctx context.Context, clientID, workspaceID, productID, id, actorID uuid.UUID, i ClaimInput) (Claim, error) {
	if e := validateClaim(&i, true); e != nil {
		return Claim{}, e
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return Claim{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var currentVersion int64
	e = tx.QueryRow(ctx, `SELECT version FROM product_claims WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND product_id=$4 FOR UPDATE`, id, clientID, workspaceID, productID).Scan(&currentVersion)
	if errors.Is(e, pgx.ErrNoRows) {
		return Claim{}, ErrNotFound
	}
	if e != nil {
		return Claim{}, e
	}
	if currentVersion != i.Version {
		return Claim{}, ErrConflict
	}
	var item Claim
	e = tx.QueryRow(ctx, `UPDATE product_claims SET claim_kind=$5::claim_kind,claim_text=$6,rationale=$7,status='DRAFT',approved_by=NULL,approved_at=NULL,version=version+1,updated_by=$8,updated_at=now() WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND product_id=$4 RETURNING id,product_id,claim_kind::text,claim_text,rationale,status::text,version,approved_at,created_at,updated_at`, id, clientID, workspaceID, productID, i.ClaimKind, i.ClaimText, i.Rationale, actorID).Scan(&item.ID, &item.ProductID, &item.ClaimKind, &item.ClaimText, &item.Rationale, &item.Status, &item.Version, &item.ApprovedAt, &item.CreatedAt, &item.UpdatedAt)
	if e != nil {
		return Claim{}, e
	}
	if _, e = tx.Exec(ctx, `DELETE FROM product_claim_sources WHERE claim_id=$1`, id); e != nil {
		return Claim{}, e
	}
	for _, factID := range i.FactIDs {
		if _, e = tx.Exec(ctx, `INSERT INTO product_claim_sources(claim_id,fact_id,evidence_excerpt) SELECT $1,id,source_excerpt FROM product_facts WHERE id=$2 AND product_id=$3 AND client_id=$4 AND workspace_id=$5 AND status='APPROVED'`, item.ID, factID, productID, clientID, workspaceID); e != nil {
			return Claim{}, e
		}
	}
	for _, assetID := range i.MediaAssetIDs {
		if _, e = tx.Exec(ctx, `INSERT INTO product_claim_sources(claim_id,media_asset_id) SELECT $1,id FROM media_assets WHERE id=$2 AND product_id=$3 AND client_id=$4 AND workspace_id=$5 AND deleted_at IS NULL`, item.ID, assetID, productID, clientID, workspaceID); e != nil {
			return Claim{}, e
		}
	}
	if _, e = tx.Exec(ctx, `INSERT INTO product_claim_versions(product_claim_id,version,claim_text,rationale,change_summary,created_by)VALUES($1,$2,$3,$4,$5,$6)`, item.ID, item.Version, item.ClaimText, item.Rationale, i.ChangeSummary, actorID); e != nil {
		return Claim{}, e
	}
	if e = invalidateProductApprovals(ctx, tx, clientID, workspaceID, productID, actorID, "Product Truth claim changed"); e != nil {
		return Claim{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Claim{}, e
	}
	return item, nil
}

func (s *Service) ApproveClaim(ctx context.Context, clientID, workspaceID, productID, id uuid.UUID, actor auth.Principal, metadata auth.ClientMetadata, version int64) (Claim, error) {
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return Claim{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var before Claim
	e = tx.QueryRow(ctx, `SELECT id,product_id,claim_kind::text,claim_text,rationale,status::text,version,approved_at,created_at,updated_at FROM product_claims WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND product_id=$4 FOR UPDATE`, id, clientID, workspaceID, productID).Scan(&before.ID, &before.ProductID, &before.ClaimKind, &before.ClaimText, &before.Rationale, &before.Status, &before.Version, &before.ApprovedAt, &before.CreatedAt, &before.UpdatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return Claim{}, ErrNotFound
	}
	if e != nil {
		return Claim{}, e
	}
	if before.Version != version {
		return Claim{}, ErrConflict
	}
	var item Claim
	e = tx.QueryRow(ctx, `UPDATE product_claims SET status='APPROVED',approved_by=$5,approved_at=now(),version=version+1,updated_by=$5,updated_at=now() WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND product_id=$4 RETURNING id,product_id,claim_kind::text,claim_text,rationale,status::text,version,approved_at,created_at,updated_at`, id, clientID, workspaceID, productID, actor.UserID).Scan(&item.ID, &item.ProductID, &item.ClaimKind, &item.ClaimText, &item.Rationale, &item.Status, &item.Version, &item.ApprovedAt, &item.CreatedAt, &item.UpdatedAt)
	if e != nil {
		return Claim{}, e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO product_claim_versions(product_claim_id,version,claim_text,rationale,change_summary,created_by) VALUES($1,$2,$3,$4,$5,$6)`, item.ID, item.Version, item.ClaimText, item.Rationale, "Approved Product Truth claim", actor.UserID); e != nil {
		return Claim{}, e
	}
	if e = audit.Record(ctx, db.New(tx), audit.Event{ActorID: uuid.NullUUID{UUID: actor.UserID, Valid: true}, Action: "product_claim.approved", EntityType: "product_claim", EntityID: uuid.NullUUID{UUID: id, Valid: true}, ClientID: uuid.NullUUID{UUID: clientID, Valid: true}, WorkspaceID: uuid.NullUUID{UUID: workspaceID, Valid: true}, RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS", Before: before, After: item, Metadata: map[string]any{"productId": productID}}); e != nil {
		return Claim{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Claim{}, e
	}
	return item, nil
}
func validateClaim(i *ClaimInput, version bool) error {
	i.ClaimKind = strings.ToUpper(strings.TrimSpace(i.ClaimKind))
	i.ClaimText = strings.TrimSpace(i.ClaimText)
	if (i.ClaimKind != "APPROVED" && i.ClaimKind != "PROHIBITED") || len(i.ClaimText) < 3 || len(i.ClaimText) > 2000 || (version && i.Version < 1) {
		return ErrInvalid
	}
	if ValidateNoUniversalAirlineClaim(i.ClaimText, false) != nil && i.ClaimKind == "APPROVED" {
		return ErrInvalid
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scanFact(r scanner) (Fact, error) {
	var i Fact
	var raw []byte
	e := r.Scan(&i.ID, &i.ProductID, &i.ClientID, &i.WorkspaceID, &i.FactKey, &i.Label, &i.ExactValue, &raw, &i.Unit, &i.SourceName, &i.SourceExcerpt, &i.SourceAssetID, &i.Status, &i.LockedValue, &i.EffectiveFrom, &i.ExpiresAt, &i.Version, &i.ApprovedBy, &i.ApprovedAt, &i.CreatedAt, &i.UpdatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return Fact{}, ErrNotFound
	}
	if e != nil {
		return Fact{}, e
	}
	_ = json.Unmarshal(raw, &i.NormalizedValue)
	return i, nil
}
func mapConflict(e error) error {
	if strings.Contains(e.Error(), "23505") {
		return ErrConflict
	}
	return e
}

func invalidateProductApprovals(ctx context.Context, tx pgx.Tx, clientID, workspaceID, productID, actorID uuid.UUID, reason string) error {
	if _, err := tx.Exec(ctx, `WITH affected AS (
		SELECT id FROM campaigns WHERE client_id=$1 AND workspace_id=$2 AND product_id=$3
	), changed AS (
		UPDATE approvals a SET invalidated_at=now(),invalidation_reason=$5
		FROM affected x WHERE a.campaign_id=x.id AND a.invalidated_at IS NULL
		RETURNING a.id,a.entity_version,a.entity_hash
	) INSERT INTO approval_events(approval_id,event_type,actor_id,entity_version,entity_hash,notes)
	SELECT id,'INVALIDATED',$4,entity_version,entity_hash,$5 FROM changed`, clientID, workspaceID, productID, actorID, reason); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE campaigns SET status='DRAFT',selected_concept_id=NULL,version=version+1,updated_by=$4,updated_at=now() WHERE client_id=$1 AND workspace_id=$2 AND product_id=$3 AND status<>'ARCHIVED'`, clientID, workspaceID, productID, actorID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE campaign_concepts x SET status='DRAFT',locked_at=NULL,locked_by=NULL,version=x.version+1,updated_by=$4,updated_at=now() FROM campaigns c WHERE x.campaign_id=c.id AND c.client_id=$1 AND c.workspace_id=$2 AND c.product_id=$3 AND x.status IN('APPROVED','LOCKED')`, clientID, workspaceID, productID, actorID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE campaign_content_variants x SET status='DRAFT',approved_at=NULL,approved_by=NULL,version=x.version+1,updated_by=$4,updated_at=now() FROM campaigns c WHERE x.campaign_id=c.id AND c.client_id=$1 AND c.workspace_id=$2 AND c.product_id=$3 AND x.status='APPROVED'`, clientID, workspaceID, productID, actorID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE scripts x SET status='DRAFT',approved_at=NULL,approved_by=NULL,version=x.version+1,updated_by=$4,updated_at=now() FROM campaigns c WHERE x.campaign_id=c.id AND c.client_id=$1 AND c.workspace_id=$2 AND c.product_id=$3 AND x.status='APPROVED'`, clientID, workspaceID, productID, actorID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE scenes x SET status='DRAFT',approved_at=NULL,approved_by=NULL,version=x.version+1,updated_by=$4,updated_at=now() FROM campaigns c WHERE x.campaign_id=c.id AND c.client_id=$1 AND c.workspace_id=$2 AND c.product_id=$3 AND x.status='APPROVED'`, clientID, workspaceID, productID, actorID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE meta_ad_actions x SET status='REJECTED',reviewed_by=$4,review_notes=$5,reviewed_at=now(),version=x.version+1 FROM ad_campaigns a JOIN campaigns c ON c.id=a.campaign_id WHERE x.ad_campaign_id=a.id AND c.client_id=$1 AND c.workspace_id=$2 AND c.product_id=$3 AND x.action='CREATE_PAUSED' AND x.status IN('PENDING_APPROVAL','APPROVED','QUEUED')`, clientID, workspaceID, productID, actorID, reason)
	return err
}
