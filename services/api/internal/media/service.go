package media

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrInvalid     = errors.New("invalid media")
	ErrNotFound    = errors.New("media not found")
	ErrUnavailable = errors.New("object storage unavailable")
	ErrConflict    = errors.New("media conflict")
)

const multipartThreshold int64 = 100 * 1024 * 1024

type Asset struct {
	ID            uuid.UUID  `json:"id"`
	ClientID      uuid.UUID  `json:"clientId"`
	WorkspaceID   uuid.UUID  `json:"workspaceId"`
	BrandID       *uuid.UUID `json:"brandId"`
	ProductID     *uuid.UUID `json:"productId"`
	CampaignID    *uuid.UUID `json:"campaignId"`
	AssetType     string     `json:"assetType"`
	Category      string     `json:"category"`
	Name          string     `json:"name"`
	Folder        string     `json:"folder"`
	Status        string     `json:"status"`
	UsageRights   string     `json:"usageRights"`
	Tags          []string   `json:"tags"`
	MimeType      *string    `json:"mimeType"`
	FileSizeBytes *int64     `json:"fileSizeBytes"`
	Width         *int32     `json:"width"`
	Height        *int32     `json:"height"`
	DurationMs    *int64     `json:"durationMs"`
	Version       int64      `json:"version"`
	ExpiresAt     *time.Time `json:"expiresAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}
type UploadInput struct {
	BrandID        *uuid.UUID `json:"brandId"`
	ProductID      *uuid.UUID `json:"productId"`
	CampaignID     *uuid.UUID `json:"campaignId"`
	AssetType      string     `json:"assetType"`
	Category       string     `json:"category"`
	Name           string     `json:"name"`
	Folder         string     `json:"folder"`
	Filename       string     `json:"filename"`
	MimeType       string     `json:"mimeType"`
	SizeBytes      int64      `json:"sizeBytes"`
	UsageRights    string     `json:"usageRights"`
	SourceMetadata any        `json:"sourceMetadata"`
	Tags           []string   `json:"tags"`
	ExpiresAt      *time.Time `json:"expiresAt"`
	TemporaryUntil *time.Time `json:"temporaryUntil"`
}
type UpdateInput struct {
	Category    string     `json:"category"`
	Name        string     `json:"name"`
	Folder      string     `json:"folder"`
	UsageRights string     `json:"usageRights"`
	Tags        []string   `json:"tags"`
	ExpiresAt   *time.Time `json:"expiresAt"`
	Version     int64      `json:"version"`
}
type UploadSession struct {
	ID                uuid.UUID                 `json:"id"`
	AssetID           uuid.UUID                 `json:"assetId"`
	StorageKey        string                    `json:"storageKey"`
	Multipart         bool                      `json:"multipart"`
	MultipartUploadID *string                   `json:"multipartUploadId"`
	Request           *storage.PresignedRequest `json:"request"`
	ExpiresAt         time.Time                 `json:"expiresAt"`
}
type Service struct {
	pool     *pgxpool.Pool
	store    storage.ObjectStore
	enqueuer MetadataEnqueuer
}

type MetadataEnqueuer interface {
	EnqueueMediaMetadata(context.Context, pgx.Tx, uuid.UUID, uuid.UUID, uuid.UUID) error
}

func NewService(p *pgxpool.Pool, s storage.ObjectStore, enqueuer MetadataEnqueuer) *Service {
	return &Service{pool: p, store: s, enqueuer: enqueuer}
}

const assetCols = `a.id,a.client_id,a.workspace_id,a.brand_id,a.product_id,a.campaign_id,a.asset_type::text,a.category,a.name,a.folder,a.status::text,a.usage_rights,COALESCE(array_agg(DISTINCT t.tag) FILTER(WHERE t.tag IS NOT NULL),'{}'),v.mime_type,v.file_size_bytes,v.width,v.height,v.duration_ms,a.version,a.expires_at,a.created_at,a.updated_at`

func (s *Service) List(ctx context.Context, clientID, workspaceID uuid.UUID, search, assetType, status string) ([]Asset, error) {
	rows, e := s.pool.Query(ctx, `SELECT `+assetCols+` FROM media_assets a LEFT JOIN media_asset_tags t ON t.media_asset_id=a.id LEFT JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version WHERE a.client_id=$1 AND a.workspace_id=$2 AND a.deleted_at IS NULL AND ($3='' OR a.asset_type::text=$3) AND ($4='' OR a.status::text=$4) AND ($5='' OR a.name ILIKE '%'||$5||'%' OR t.tag ILIKE '%'||$5||'%') GROUP BY a.id,v.id ORDER BY a.created_at DESC`, clientID, workspaceID, strings.ToUpper(assetType), strings.ToUpper(status), strings.TrimSpace(search))
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []Asset{}
	for rows.Next() {
		i, e := scanAsset(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
func (s *Service) StartUpload(ctx context.Context, clientID, workspaceID, actorID uuid.UUID, i UploadInput) (UploadSession, error) {
	if s.store == nil {
		return UploadSession{}, ErrUnavailable
	}
	ext, e := validateUpload(&i)
	if e != nil {
		return UploadSession{}, e
	}
	var scopeValid bool
	if e = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces w WHERE w.id=$2 AND w.client_id=$1 AND w.status='ACTIVE' AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM brands b WHERE b.id=$3 AND b.client_id=$1 AND b.workspace_id=$2)) AND ($4::uuid IS NULL OR EXISTS(SELECT 1 FROM products p WHERE p.id=$4 AND p.client_id=$1 AND p.workspace_id=$2)) AND ($5::uuid IS NULL OR EXISTS(SELECT 1 FROM campaigns c WHERE c.id=$5 AND c.client_id=$1 AND c.workspace_id=$2)))`, clientID, workspaceID, i.BrandID, i.ProductID, i.CampaignID).Scan(&scopeValid); e != nil {
		return UploadSession{}, e
	}
	if !scopeValid {
		return UploadSession{}, ErrNotFound
	}
	assetID, uploadID := uuid.New(), uuid.New()
	key, e := storage.ScopedObjectKey(workspaceID, assetID, i.Filename)
	if e != nil {
		return UploadSession{}, ErrInvalid
	}
	expires := time.Now().UTC().Add(30 * time.Minute)
	session := UploadSession{ID: uploadID, AssetID: assetID, StorageKey: key, ExpiresAt: expires}
	multipartCreated := false
	persisted := false
	defer func() {
		if multipartCreated && !persisted && session.MultipartUploadID != nil {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_ = s.store.AbortMultipartUpload(cleanupCtx, key, *session.MultipartUploadID)
		}
	}()
	if i.SizeBytes > multipartThreshold {
		providerID, e := s.store.CreateMultipartUpload(ctx, key, i.MimeType, map[string]string{"workspace-id": workspaceID.String(), "asset-id": assetID.String()})
		if e != nil {
			return UploadSession{}, e
		}
		session.Multipart = true
		session.MultipartUploadID = &providerID
		multipartCreated = true
	} else {
		request, e := s.store.PresignPut(ctx, key, i.MimeType, i.SizeBytes, 30*time.Minute)
		if e != nil {
			return UploadSession{}, e
		}
		session.Request = &request
	}
	metadata, _ := json.Marshal(i.SourceMetadata)
	if string(metadata) == "null" {
		metadata = []byte("{}")
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return UploadSession{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, e := tx.Exec(ctx, `INSERT INTO media_assets(id,client_id,workspace_id,brand_id,product_id,campaign_id,asset_type,category,name,folder,usage_rights,source_metadata,expires_at,temporary_until,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7::media_asset_type,$8,$9,$10,$11,$12,$13,$14,$15,$15)`, assetID, clientID, workspaceID, i.BrandID, i.ProductID, i.CampaignID, i.AssetType, i.Category, i.Name, i.Folder, i.UsageRights, metadata, i.ExpiresAt, i.TemporaryUntil, actorID)
	if e != nil {
		return UploadSession{}, e
	}
	if tag.RowsAffected() != 1 {
		return UploadSession{}, ErrNotFound
	}
	_, e = tx.Exec(ctx, `INSERT INTO media_asset_tags(media_asset_id,tag) SELECT $1,unnest($2::text[])`, assetID, i.Tags)
	if e != nil {
		return UploadSession{}, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO media_uploads(id,client_id,workspace_id,media_asset_id,storage_key,multipart_upload_id,expected_filename,expected_mime_type,expected_extension,expected_size_bytes,status,expires_at,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::upload_status,$12,$13)`, uploadID, clientID, workspaceID, assetID, key, session.MultipartUploadID, i.Filename, i.MimeType, ext, i.SizeBytes, map[bool]string{true: "UPLOADING", false: "PENDING"}[session.Multipart], expires, actorID)
	if e != nil {
		return UploadSession{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return UploadSession{}, e
	}
	persisted = true
	return session, nil
}
func (s *Service) PresignPart(ctx context.Context, clientID, workspaceID, uploadID uuid.UUID, part int32) (storage.PresignedRequest, error) {
	if s.store == nil {
		return storage.PresignedRequest{}, ErrUnavailable
	}
	var key, providerID string
	var expires time.Time
	e := s.pool.QueryRow(ctx, `SELECT storage_key,multipart_upload_id,expires_at FROM media_uploads WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND status='UPLOADING'`, uploadID, clientID, workspaceID).Scan(&key, &providerID, &expires)
	if errors.Is(e, pgx.ErrNoRows) {
		return storage.PresignedRequest{}, ErrNotFound
	}
	if e != nil {
		return storage.PresignedRequest{}, e
	}
	if time.Now().After(expires) {
		return storage.PresignedRequest{}, ErrConflict
	}
	return s.store.PresignUploadPart(ctx, key, providerID, part, 15*time.Minute)
}
func (s *Service) Complete(ctx context.Context, clientID, workspaceID, uploadID, actorID uuid.UUID, parts []storage.UploadedPart) (Asset, error) {
	if s.store == nil {
		return Asset{}, ErrUnavailable
	}
	var assetID uuid.UUID
	var key, mime, ext string
	var size int64
	var multipartID *string
	var status string
	e := s.pool.QueryRow(ctx, `SELECT media_asset_id,storage_key,multipart_upload_id,expected_mime_type,expected_extension,expected_size_bytes,status::text FROM media_uploads WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND status IN('PENDING','UPLOADING','VERIFIED') AND (status='VERIFIED' OR expires_at>now())`, uploadID, clientID, workspaceID).Scan(&assetID, &key, &multipartID, &mime, &ext, &size, &status)
	if errors.Is(e, pgx.ErrNoRows) {
		return Asset{}, ErrNotFound
	}
	if e != nil {
		return Asset{}, e
	}
	if status == "VERIFIED" {
		return s.Get(ctx, clientID, workspaceID, assetID)
	}
	var meta storage.ObjectMetadata
	metadataLoaded := false
	if multipartID != nil {
		if e = s.store.CompleteMultipartUpload(ctx, key, *multipartID, parts); e != nil {
			// The provider can commit the object while the response is lost. A HEAD
			// check makes the server completion endpoint safe to retry in that case.
			meta, e = s.store.Head(ctx, key)
			if e != nil {
				return Asset{}, e
			}
			metadataLoaded = true
		}
	} else if len(parts) > 0 {
		return Asset{}, ErrInvalid
	}
	if !metadataLoaded {
		meta, e = s.store.Head(ctx, key)
		if e != nil {
			return Asset{}, e
		}
	}
	if meta.ContentLength != size || !sameMime(meta.ContentType, mime) {
		_, _ = s.pool.Exec(ctx, `UPDATE media_uploads SET status='FAILED',failure_reason='server metadata mismatch' WHERE id=$1`, uploadID)
		return Asset{}, ErrConflict
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return Asset{}, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, e = tx.Exec(ctx, `INSERT INTO media_asset_versions(media_asset_id,client_id,workspace_id,version,storage_key,original_filename,mime_type,file_extension,file_size_bytes,metadata,verified_at,created_by) SELECT media_asset_id,client_id,workspace_id,1,storage_key,expected_filename,expected_mime_type,expected_extension,expected_size_bytes,$2,now(),$3 FROM media_uploads WHERE id=$1`, uploadID, mustJSON(meta.Metadata), actorID)
	if e != nil {
		return Asset{}, e
	}
	_, e = tx.Exec(ctx, `UPDATE media_uploads SET status='VERIFIED',completed_at=now() WHERE id=$1`, uploadID)
	if e != nil {
		return Asset{}, e
	}
	if s.enqueuer != nil {
		if e = s.enqueuer.EnqueueMediaMetadata(ctx, tx, assetID, clientID, workspaceID); e != nil {
			return Asset{}, e
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return Asset{}, e
	}
	return s.Get(ctx, clientID, workspaceID, assetID)
}
func (s *Service) Get(ctx context.Context, clientID, workspaceID, id uuid.UUID) (Asset, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+assetCols+` FROM media_assets a LEFT JOIN media_asset_tags t ON t.media_asset_id=a.id LEFT JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version WHERE a.id=$1 AND a.client_id=$2 AND a.workspace_id=$3 AND a.deleted_at IS NULL GROUP BY a.id,v.id`, id, clientID, workspaceID)
	return scanAsset(row)
}
func (s *Service) Download(ctx context.Context, clientID, workspaceID, id uuid.UUID) (storage.PresignedRequest, error) {
	if s.store == nil {
		return storage.PresignedRequest{}, ErrUnavailable
	}
	var key string
	e := s.pool.QueryRow(ctx, `SELECT v.storage_key FROM media_assets a JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version WHERE a.id=$1 AND a.client_id=$2 AND a.workspace_id=$3 AND a.deleted_at IS NULL AND a.status<>'REJECTED'`, id, clientID, workspaceID).Scan(&key)
	if errors.Is(e, pgx.ErrNoRows) {
		return storage.PresignedRequest{}, ErrNotFound
	}
	if e != nil {
		return storage.PresignedRequest{}, e
	}
	return s.store.PresignGet(ctx, key, 10*time.Minute)
}
func (s *Service) Update(ctx context.Context, clientID, workspaceID, id, actorID uuid.UUID, input UpdateInput) (Asset, error) {
	if err := validateUpdate(&input); err != nil {
		return Asset{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Asset{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE media_assets SET category=$4,name=$5,folder=$6,usage_rights=$7,expires_at=$8,version=version+1,updated_by=$9,updated_at=now() WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND version=$10 AND deleted_at IS NULL`, id, clientID, workspaceID, input.Category, input.Name, input.Folder, input.UsageRights, input.ExpiresAt, actorID, input.Version)
	if err != nil {
		return Asset{}, err
	}
	if tag.RowsAffected() != 1 {
		return Asset{}, ErrConflict
	}
	if _, err = tx.Exec(ctx, `DELETE FROM media_asset_tags WHERE media_asset_id=$1`, id); err != nil {
		return Asset{}, err
	}
	if len(input.Tags) > 0 {
		if _, err = tx.Exec(ctx, `INSERT INTO media_asset_tags(media_asset_id,tag) SELECT $1,unnest($2::text[])`, id, input.Tags); err != nil {
			return Asset{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Asset{}, err
	}
	return s.Get(ctx, clientID, workspaceID, id)
}
func (s *Service) SetStatus(ctx context.Context, clientID, workspaceID, id, actorID uuid.UUID, status string, version int64) (Asset, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if !map[string]bool{"DRAFT": true, "APPROVED": true, "REJECTED": true, "ARCHIVED": true}[status] || version < 1 {
		return Asset{}, ErrInvalid
	}
	tag, err := s.pool.Exec(ctx, `UPDATE media_assets SET status=$4::content_status,version=version+1,updated_by=$5,updated_at=now() WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND version=$6 AND deleted_at IS NULL`, id, clientID, workspaceID, status, actorID, version)
	if err != nil {
		return Asset{}, err
	}
	if tag.RowsAffected() != 1 {
		return Asset{}, ErrConflict
	}
	return s.Get(ctx, clientID, workspaceID, id)
}
func (s *Service) SoftDelete(ctx context.Context, clientID, workspaceID, id, actorID uuid.UUID, version int64) error {
	tag, e := s.pool.Exec(ctx, `UPDATE media_assets SET deleted_at=now(),status='ARCHIVED',version=version+1,updated_by=$4,updated_at=now() WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND version=$5 AND deleted_at IS NULL`, id, clientID, workspaceID, actorID, version)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return nil
}

func validateUpdate(input *UpdateInput) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Category = strings.TrimSpace(input.Category)
	input.Folder = strings.Trim(strings.TrimSpace(input.Folder), "/")
	input.UsageRights = strings.TrimSpace(input.UsageRights)
	if input.Name == "" || len(input.Name) > 240 || len(input.Category) > 120 || len(input.Folder) > 500 || input.UsageRights == "" || len(input.UsageRights) > 4000 || input.Version < 1 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for index, tag := range input.Tags {
		input.Tags[index] = strings.TrimSpace(tag)
		if input.Tags[index] == "" || len(input.Tags[index]) > 80 || seen[input.Tags[index]] {
			return ErrInvalid
		}
		seen[input.Tags[index]] = true
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scanAsset(r scanner) (Asset, error) {
	var i Asset
	e := r.Scan(&i.ID, &i.ClientID, &i.WorkspaceID, &i.BrandID, &i.ProductID, &i.CampaignID, &i.AssetType, &i.Category, &i.Name, &i.Folder, &i.Status, &i.UsageRights, &i.Tags, &i.MimeType, &i.FileSizeBytes, &i.Width, &i.Height, &i.DurationMs, &i.Version, &i.ExpiresAt, &i.CreatedAt, &i.UpdatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return Asset{}, ErrNotFound
	}
	return i, e
}

var allowed = map[string]struct {
	types []string
	max   int64
}{".jpg": {[]string{"image/jpeg"}, 25 << 20}, ".jpeg": {[]string{"image/jpeg"}, 25 << 20}, ".png": {[]string{"image/png"}, 25 << 20}, ".webp": {[]string{"image/webp"}, 25 << 20}, ".mp4": {[]string{"video/mp4"}, 2 << 30}, ".mov": {[]string{"video/quicktime"}, 2 << 30}, ".mp3": {[]string{"audio/mpeg"}, 250 << 20}, ".wav": {[]string{"audio/wav", "audio/x-wav"}, 250 << 20}, ".pdf": {[]string{"application/pdf"}, 50 << 20}}

func validateUpload(i *UploadInput) (string, error) {
	i.AssetType = strings.ToUpper(strings.TrimSpace(i.AssetType))
	i.Name = strings.TrimSpace(i.Name)
	i.Filename = filepath.Base(strings.TrimSpace(i.Filename))
	i.MimeType = strings.ToLower(strings.TrimSpace(strings.Split(i.MimeType, ";")[0]))
	i.UsageRights = strings.TrimSpace(i.UsageRights)
	if i.Name == "" || i.UsageRights == "" || i.SizeBytes <= 0 {
		return "", ErrInvalid
	}
	validTypes := map[string]bool{"IMAGE": true, "VIDEO": true, "AUDIO": true, "LOGO": true, "BROCHURE": true, "SCREENSHOT": true, "SCREEN_RECORDING": true}
	if !validTypes[i.AssetType] {
		return "", ErrInvalid
	}
	ext := strings.ToLower(filepath.Ext(i.Filename))
	rule, ok := allowed[ext]
	if !ok || i.SizeBytes > rule.max {
		return "", ErrInvalid
	}
	ok = false
	for _, t := range rule.types {
		if t == i.MimeType {
			ok = true
		}
	}
	if !ok {
		return "", ErrInvalid
	}
	seen := map[string]bool{}
	for n, tag := range i.Tags {
		i.Tags[n] = strings.TrimSpace(tag)
		if i.Tags[n] == "" || seen[i.Tags[n]] {
			return "", ErrInvalid
		}
		seen[i.Tags[n]] = true
	}
	return ext, nil
}
func sameMime(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(strings.Split(a, ";")[0]), strings.TrimSpace(strings.Split(b, ";")[0]))
}
func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
