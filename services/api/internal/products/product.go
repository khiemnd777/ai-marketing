package products

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/verticals"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
)

var (
	ErrInvalid       = errors.New("invalid product")
	ErrNotFound      = errors.New("product not found")
	ErrConflict      = errors.New("product conflict")
	ErrMediaNotReady = errors.New("product media is not ready")
)

type Product struct {
	ID               uuid.UUID  `json:"id"`
	ClientID         uuid.UUID  `json:"clientId"`
	WorkspaceID      uuid.UUID  `json:"workspaceId"`
	BrandID          *uuid.UUID `json:"brandId"`
	Name             string     `json:"name"`
	SKU              string     `json:"sku"`
	Model            string     `json:"model"`
	Category         string     `json:"category"`
	VerticalKey      string     `json:"verticalKey"`
	Status           string     `json:"status"`
	CurrentVersion   int32      `json:"currentVersion"`
	Version          int64      `json:"version"`
	ShortDescription string     `json:"shortDescription"`
	LongDescription  string     `json:"longDescription"`
	Features         []string   `json:"features"`
	Benefits         []string   `json:"benefits"`
	Differentiators  []string   `json:"differentiators"`
	IntendedAudience string     `json:"intendedAudience"`
	Currency         *string    `json:"currency"`
	RegularPrice     *float64   `json:"regularPrice"`
	SalePrice        *float64   `json:"salePrice"`
	DiscountCode     *string    `json:"discountCode"`
	OfferValidFrom   *time.Time `json:"offerValidFrom"`
	OfferValidUntil  *time.Time `json:"offerValidUntil"`
	Variants         any        `json:"variants"`
	VerticalData     any        `json:"verticalData"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}
type Input struct {
	BrandID          *uuid.UUID `json:"brandId"`
	Name             string     `json:"name"`
	SKU              string     `json:"sku"`
	Model            string     `json:"model"`
	Category         string     `json:"category"`
	VerticalKey      string     `json:"verticalKey"`
	ShortDescription string     `json:"shortDescription"`
	LongDescription  string     `json:"longDescription"`
	Features         []string   `json:"features"`
	Benefits         []string   `json:"benefits"`
	Differentiators  []string   `json:"differentiators"`
	IntendedAudience string     `json:"intendedAudience"`
	Currency         *string    `json:"currency"`
	RegularPrice     *float64   `json:"regularPrice"`
	SalePrice        *float64   `json:"salePrice"`
	DiscountCode     *string    `json:"discountCode"`
	OfferValidFrom   *time.Time `json:"offerValidFrom"`
	OfferValidUntil  *time.Time `json:"offerValidUntil"`
	Variants         any        `json:"variants"`
	VerticalData     any        `json:"verticalData"`
	ChangeSummary    string     `json:"changeSummary"`
	Version          int64      `json:"version"`
}

type MediaRequirement struct {
	Category       string `json:"category"`
	TotalAssets    int32  `json:"totalAssets"`
	ApprovedAssets int32  `json:"approvedAssets"`
	Ready          bool   `json:"ready"`
}

type MediaReadiness struct {
	ProductID    uuid.UUID          `json:"productId"`
	VerticalKey  string             `json:"verticalKey"`
	Ready        bool               `json:"ready"`
	Requirements []MediaRequirement `json:"requirements"`
}

type mediaCounts struct{ total, approved int32 }
type Service struct {
	pool      *pgxpool.Pool
	verticals *verticals.Registry
}

func NewService(p *pgxpool.Pool, v *verticals.Registry) *Service {
	return &Service{pool: p, verticals: v}
}

const cols = `p.id,p.client_id,p.workspace_id,p.brand_id,p.name,p.sku,p.model,p.category,p.vertical_key,p.status::text,p.current_version,p.version,v.short_description,v.long_description,v.features,v.benefits,v.differentiators,v.intended_audience,v.currency,v.regular_price::float8,v.sale_price::float8,v.discount_code,v.offer_valid_from,v.offer_valid_until,v.variants,d.data,p.created_at,p.updated_at`

func (s *Service) List(ctx context.Context, clientID, workspaceID uuid.UUID, search, status string) ([]Product, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	rows, e := s.pool.Query(ctx, `SELECT `+cols+` FROM products p JOIN product_versions v ON v.product_id=p.id AND v.version=p.current_version LEFT JOIN LATERAL(SELECT data FROM product_vertical_data WHERE product_id=p.id ORDER BY created_at DESC LIMIT 1)d ON true WHERE p.client_id=$1 AND p.workspace_id=$2 AND ($3='' OR p.status::text=$3) AND ($4='' OR p.name ILIKE '%'||$4||'%' OR p.sku ILIKE '%'||$4||'%') ORDER BY p.status,p.name`, clientID, workspaceID, status, strings.TrimSpace(search))
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []Product{}
	for rows.Next() {
		i, e := scan(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
func (s *Service) Get(ctx context.Context, clientID, workspaceID, id uuid.UUID) (Product, error) {
	return scan(s.pool.QueryRow(ctx, `SELECT `+cols+` FROM products p JOIN product_versions v ON v.product_id=p.id AND v.version=p.current_version LEFT JOIN LATERAL(SELECT data FROM product_vertical_data WHERE product_id=p.id ORDER BY created_at DESC LIMIT 1)d ON true WHERE p.id=$1 AND p.client_id=$2 AND p.workspace_id=$3`, id, clientID, workspaceID))
}

func (s *Service) MediaReadiness(ctx context.Context, clientID, workspaceID, id uuid.UUID) (MediaReadiness, error) {
	var verticalKey string
	err := s.pool.QueryRow(ctx, `SELECT vertical_key FROM products WHERE id=$1 AND client_id=$2 AND workspace_id=$3`, id, clientID, workspaceID).Scan(&verticalKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return MediaReadiness{}, ErrNotFound
	}
	if err != nil {
		return MediaReadiness{}, err
	}
	return s.mediaReadiness(ctx, s.pool, clientID, workspaceID, id, verticalKey)
}
func (s *Service) Create(ctx context.Context, clientID, workspaceID, actorID uuid.UUID, i Input) (Product, error) {
	pack, raw, e := s.validate(&i, false)
	if e != nil {
		return Product{}, e
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return Product{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	e = tx.QueryRow(ctx, `INSERT INTO products(client_id,workspace_id,brand_id,name,sku,model,category,vertical_key,created_by,updated_by) SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$9 WHERE EXISTS(SELECT 1 FROM workspaces WHERE id=$2 AND client_id=$1 AND status='ACTIVE') AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM brands WHERE id=$3 AND client_id=$1 AND workspace_id=$2 AND status='ACTIVE')) RETURNING id`, clientID, workspaceID, i.BrandID, i.Name, i.SKU, i.Model, i.Category, i.VerticalKey, actorID).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	if e != nil {
		return Product{}, mapConflict(e)
	}
	if e = insertVersion(ctx, tx, id, clientID, workspaceID, 1, actorID, i); e != nil {
		return Product{}, e
	}
	digest := sha256.Sum256(raw)
	_, e = tx.Exec(ctx, `INSERT INTO product_vertical_data(product_id,client_id,workspace_id,vertical_key,schema_version,data,data_hash,validated_at,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,now(),$8)`, id, clientID, workspaceID, i.VerticalKey, pack.SchemaVersion, raw, hex.EncodeToString(digest[:]), actorID)
	if e != nil {
		return Product{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Product{}, e
	}
	return s.Get(ctx, clientID, workspaceID, id)
}
func (s *Service) Update(ctx context.Context, clientID, workspaceID, id, actorID uuid.UUID, i Input) (Product, error) {
	pack, raw, e := s.validate(&i, true)
	if e != nil {
		return Product{}, e
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return Product{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var next int32
	e = tx.QueryRow(ctx, `UPDATE products SET brand_id=$4,name=$5,sku=$6,model=$7,category=$8,vertical_key=$9,status='DRAFT',archived_at=NULL,current_version=current_version+1,version=version+1,updated_by=$10,updated_at=now() WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND version=$11 AND status<>'ARCHIVED' AND ($4::uuid IS NULL OR EXISTS(SELECT 1 FROM brands WHERE id=$4 AND client_id=$2 AND workspace_id=$3 AND status='ACTIVE')) RETURNING current_version`, id, clientID, workspaceID, i.BrandID, i.Name, i.SKU, i.Model, i.Category, i.VerticalKey, actorID, i.Version).Scan(&next)
	if errors.Is(e, pgx.ErrNoRows) {
		return Product{}, ErrConflict
	}
	if e != nil {
		return Product{}, mapConflict(e)
	}
	if e = insertVersion(ctx, tx, id, clientID, workspaceID, next, actorID, i); e != nil {
		return Product{}, e
	}
	digest := sha256.Sum256(raw)
	_, e = tx.Exec(ctx, `INSERT INTO product_vertical_data(product_id,client_id,workspace_id,vertical_key,schema_version,data,data_hash,validated_at,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,now(),$8)`, id, clientID, workspaceID, i.VerticalKey, pack.SchemaVersion, raw, hex.EncodeToString(digest[:]), actorID)
	if e != nil {
		return Product{}, e
	}
	if _, e = tx.Exec(ctx, `WITH affected AS (
		SELECT id FROM campaigns WHERE client_id=$1 AND workspace_id=$2 AND product_id=$3
	), changed AS (
		UPDATE approvals a SET invalidated_at=now(),invalidation_reason='Product profile changed'
		FROM affected x WHERE a.campaign_id=x.id AND a.invalidated_at IS NULL
		RETURNING a.id,a.entity_version,a.entity_hash
	) INSERT INTO approval_events(approval_id,event_type,actor_id,entity_version,entity_hash,notes)
	SELECT id,'INVALIDATED',$4,entity_version,entity_hash,'Product profile changed' FROM changed`, clientID, workspaceID, id, actorID); e != nil {
		return Product{}, e
	}
	if _, e = tx.Exec(ctx, `UPDATE campaigns SET status='DRAFT',selected_concept_id=NULL,version=version+1,updated_by=$4,updated_at=now() WHERE client_id=$1 AND workspace_id=$2 AND product_id=$3 AND status<>'ARCHIVED'`, clientID, workspaceID, id, actorID); e != nil {
		return Product{}, e
	}
	if _, e = tx.Exec(ctx, `UPDATE campaign_concepts x SET status='DRAFT',locked_at=NULL,locked_by=NULL,version=x.version+1,updated_by=$4,updated_at=now() FROM campaigns c WHERE x.campaign_id=c.id AND c.client_id=$1 AND c.workspace_id=$2 AND c.product_id=$3 AND x.status IN('APPROVED','LOCKED')`, clientID, workspaceID, id, actorID); e != nil {
		return Product{}, e
	}
	if _, e = tx.Exec(ctx, `UPDATE campaign_content_variants x SET status='DRAFT',approved_at=NULL,approved_by=NULL,version=x.version+1,updated_by=$4,updated_at=now() FROM campaigns c WHERE x.campaign_id=c.id AND c.client_id=$1 AND c.workspace_id=$2 AND c.product_id=$3 AND x.status='APPROVED'`, clientID, workspaceID, id, actorID); e != nil {
		return Product{}, e
	}
	if _, e = tx.Exec(ctx, `UPDATE scripts x SET status='DRAFT',approved_at=NULL,approved_by=NULL,version=x.version+1,updated_by=$4,updated_at=now() FROM campaigns c WHERE x.campaign_id=c.id AND c.client_id=$1 AND c.workspace_id=$2 AND c.product_id=$3 AND x.status='APPROVED'`, clientID, workspaceID, id, actorID); e != nil {
		return Product{}, e
	}
	if _, e = tx.Exec(ctx, `UPDATE scenes x SET status='DRAFT',approved_at=NULL,approved_by=NULL,version=x.version+1,updated_by=$4,updated_at=now() FROM campaigns c WHERE x.campaign_id=c.id AND c.client_id=$1 AND c.workspace_id=$2 AND c.product_id=$3 AND x.status='APPROVED'`, clientID, workspaceID, id, actorID); e != nil {
		return Product{}, e
	}
	if _, e = tx.Exec(ctx, `UPDATE meta_ad_actions x SET status='REJECTED',reviewed_by=$4,review_notes='Product profile changed',reviewed_at=now(),version=x.version+1 FROM ad_campaigns a JOIN campaigns c ON c.id=a.campaign_id WHERE x.ad_campaign_id=a.id AND c.client_id=$1 AND c.workspace_id=$2 AND c.product_id=$3 AND x.action='CREATE_PAUSED' AND x.status IN('PENDING_APPROVAL','APPROVED','QUEUED')`, clientID, workspaceID, id, actorID); e != nil {
		return Product{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Product{}, e
	}
	return s.Get(ctx, clientID, workspaceID, id)
}
func insertVersion(ctx context.Context, tx pgx.Tx, id, clientID, workspaceID uuid.UUID, version int32, actorID uuid.UUID, i Input) error {
	variants, _ := json.Marshal(i.Variants)
	if string(variants) == "null" {
		variants = []byte("[]")
	}
	_, e := tx.Exec(ctx, `INSERT INTO product_versions(product_id,client_id,workspace_id,version,short_description,long_description,features,benefits,differentiators,intended_audience,currency,regular_price,sale_price,discount_code,offer_valid_from,offer_valid_until,variants,change_summary,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`, id, clientID, workspaceID, version, i.ShortDescription, i.LongDescription, i.Features, i.Benefits, i.Differentiators, i.IntendedAudience, i.Currency, i.RegularPrice, i.SalePrice, i.DiscountCode, i.OfferValidFrom, i.OfferValidUntil, variants, i.ChangeSummary, actorID)
	return e
}
func (s *Service) SetStatus(ctx context.Context, clientID, workspaceID, id uuid.UUID, status string, version int64, actorID uuid.UUID) (Product, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if (status != "DRAFT" && status != "APPROVED" && status != "ARCHIVED") || version < 1 {
		return Product{}, ErrInvalid
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return Product{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var verticalKey string
	var currentVersion int64
	e = tx.QueryRow(ctx, `SELECT vertical_key,version FROM products WHERE id=$1 AND client_id=$2 AND workspace_id=$3 FOR UPDATE`, id, clientID, workspaceID).Scan(&verticalKey, &currentVersion)
	if errors.Is(e, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	if e != nil {
		return Product{}, e
	}
	if currentVersion != version {
		return Product{}, ErrConflict
	}
	if status == "APPROVED" {
		readiness, readinessErr := s.mediaReadiness(ctx, tx, clientID, workspaceID, id, verticalKey)
		if readinessErr != nil {
			return Product{}, readinessErr
		}
		if !readiness.Ready {
			return Product{}, ErrMediaNotReady
		}
	}
	tag, e := tx.Exec(ctx, `UPDATE products SET status=$4::content_status,archived_at=CASE WHEN $4='ARCHIVED' THEN now() ELSE NULL END,version=version+1,updated_by=$5,updated_at=now() WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND version=$6`, id, clientID, workspaceID, status, actorID, version)
	if e != nil {
		return Product{}, e
	}
	if tag.RowsAffected() != 1 {
		return Product{}, ErrConflict
	}
	if e = tx.Commit(ctx); e != nil {
		return Product{}, e
	}
	return s.Get(ctx, clientID, workspaceID, id)
}

type readinessQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (s *Service) mediaReadiness(ctx context.Context, q readinessQueryer, clientID, workspaceID, productID uuid.UUID, verticalKey string) (MediaReadiness, error) {
	pack, ok := s.verticals.Get(verticalKey)
	if !ok {
		return MediaReadiness{}, ErrInvalid
	}
	rows, err := q.Query(ctx, `SELECT upper(a.category),count(*)::int,count(*) FILTER (
		WHERE a.status='APPROVED' AND (a.expires_at IS NULL OR a.expires_at>now()) AND v.verified_at IS NOT NULL
	)::int FROM media_assets a
	LEFT JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version
	WHERE a.client_id=$1 AND a.workspace_id=$2 AND a.product_id=$3 AND a.deleted_at IS NULL
		AND a.asset_type IN ('IMAGE','VIDEO','LOGO','SCREENSHOT','SCREEN_RECORDING')
	GROUP BY upper(a.category)`, clientID, workspaceID, productID)
	if err != nil {
		return MediaReadiness{}, err
	}
	defer rows.Close()
	byCategory := make(map[string]mediaCounts)
	for rows.Next() {
		var category string
		var value mediaCounts
		if err = rows.Scan(&category, &value.total, &value.approved); err != nil {
			return MediaReadiness{}, err
		}
		byCategory[category] = value
	}
	if err = rows.Err(); err != nil {
		return MediaReadiness{}, err
	}
	return buildMediaReadiness(productID, verticalKey, pack.AssetRequirements.MinimumForApproval, byCategory), nil
}

func buildMediaReadiness(productID uuid.UUID, verticalKey string, minimum []string, byCategory map[string]mediaCounts) MediaReadiness {
	result := MediaReadiness{ProductID: productID, VerticalKey: verticalKey, Ready: true, Requirements: make([]MediaRequirement, 0, len(minimum))}
	for _, category := range minimum {
		value := byCategory[category]
		requirement := MediaRequirement{Category: category, TotalAssets: value.total, ApprovedAssets: value.approved, Ready: value.approved > 0}
		result.Requirements = append(result.Requirements, requirement)
		result.Ready = result.Ready && requirement.Ready
	}
	return result
}
func (s *Service) validate(i *Input, version bool) (verticals.Pack, []byte, error) {
	i.Name = strings.TrimSpace(i.Name)
	i.SKU = strings.TrimSpace(i.SKU)
	i.Model = strings.TrimSpace(i.Model)
	i.Category = strings.TrimSpace(i.Category)
	i.VerticalKey = strings.TrimSpace(i.VerticalKey)
	if len(i.Name) < 2 || len(i.Name) > 200 || len(i.SKU) < 1 || len(i.SKU) > 100 || i.Category == "" || (version && i.Version < 1) {
		return verticals.Pack{}, nil, ErrInvalid
	}
	if i.Features == nil {
		i.Features = []string{}
	}
	if i.Benefits == nil {
		i.Benefits = []string{}
	}
	if i.Differentiators == nil {
		i.Differentiators = []string{}
	}
	if i.Variants == nil {
		i.Variants = []any{}
	}
	if i.Currency != nil {
		v := strings.ToUpper(strings.TrimSpace(*i.Currency))
		if len(v) != 3 {
			return verticals.Pack{}, nil, ErrInvalid
		}
		i.Currency = &v
	}
	if i.RegularPrice != nil && *i.RegularPrice < 0 || i.SalePrice != nil && *i.SalePrice < 0 || i.RegularPrice != nil && i.SalePrice != nil && *i.SalePrice > *i.RegularPrice {
		return verticals.Pack{}, nil, ErrInvalid
	}
	raw, e := json.Marshal(i.VerticalData)
	if e != nil {
		return verticals.Pack{}, nil, ErrInvalid
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return verticals.Pack{}, nil, ErrInvalid
	}
	pack, e := s.verticals.Validate(i.VerticalKey, value)
	if e != nil {
		return verticals.Pack{}, nil, ErrInvalid
	}
	return pack, raw, nil
}

type scanner interface{ Scan(...any) error }

func scan(r scanner) (Product, error) {
	var i Product
	var variantsRaw, verticalRaw []byte
	e := r.Scan(&i.ID, &i.ClientID, &i.WorkspaceID, &i.BrandID, &i.Name, &i.SKU, &i.Model, &i.Category, &i.VerticalKey, &i.Status, &i.CurrentVersion, &i.Version, &i.ShortDescription, &i.LongDescription, &i.Features, &i.Benefits, &i.Differentiators, &i.IntendedAudience, &i.Currency, &i.RegularPrice, &i.SalePrice, &i.DiscountCode, &i.OfferValidFrom, &i.OfferValidUntil, &variantsRaw, &verticalRaw, &i.CreatedAt, &i.UpdatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	if e != nil {
		return Product{}, e
	}
	_ = json.Unmarshal(variantsRaw, &i.Variants)
	_ = json.Unmarshal(verticalRaw, &i.VerticalData)
	return i, nil
}
func mapConflict(e error) error {
	if strings.Contains(e.Error(), "23505") {
		return ErrConflict
	}
	return e
}
