package workspaces

import (
	"context"
	"errors"
	"fmt"
	"regexp"
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
	ErrInvalid  = errors.New("invalid workspace")
	ErrNotFound = errors.New("workspace not found")
	ErrConflict = errors.New("workspace conflict")
)
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Workspace struct {
	ID        uuid.UUID `json:"id"`
	ClientID  uuid.UUID `json:"clientId"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Timezone  string    `json:"timezone"`
	Status    string    `json:"status"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type Input struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Timezone string `json:"timezone"`
	Version  int64  `json:"version"`
}
type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }
func (s *Service) List(ctx context.Context, clientID uuid.UUID) ([]Workspace, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,client_id,name,slug,timezone,status::text,version,created_at,updated_at FROM workspaces WHERE client_id=$1 ORDER BY status,name,id`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Workspace{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (s *Service) Get(ctx context.Context, clientID, id uuid.UUID) (Workspace, error) {
	return scan(s.pool.QueryRow(ctx, `SELECT id,client_id,name,slug,timezone,status::text,version,created_at,updated_at FROM workspaces WHERE id=$1 AND client_id=$2`, id, clientID))
}
func (s *Service) Create(ctx context.Context, clientID uuid.UUID, input Input, actor auth.Principal, metadata auth.ClientMetadata) (Workspace, error) {
	if err := validate(&input, false); err != nil {
		return Workspace{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Workspace{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	item, err := scan(tx.QueryRow(ctx, `INSERT INTO workspaces(client_id,name,slug,timezone,created_by,updated_by) SELECT $1,$2,$3,$4,$5,$5 WHERE EXISTS(SELECT 1 FROM clients WHERE id=$1 AND status='ACTIVE') RETURNING id,client_id,name,slug,timezone,status::text,version,created_at,updated_at`, clientID, input.Name, input.Slug, input.Timezone, actor.UserID))
	if err != nil {
		return Workspace{}, mapDBError(err)
	}
	if err := audit.Record(ctx, db.New(tx), audit.Event{ActorID: uuid.NullUUID{UUID: actor.UserID, Valid: true}, Action: "workspace.created", EntityType: "workspace", EntityID: uuid.NullUUID{UUID: item.ID, Valid: true}, ClientID: uuid.NullUUID{UUID: clientID, Valid: true}, WorkspaceID: uuid.NullUUID{UUID: item.ID, Valid: true}, RequestID: metadata.RequestID, UserAgent: metadata.UserAgent, Outcome: "SUCCESS", After: item}); err != nil {
		return Workspace{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Workspace{}, err
	}
	return item, nil
}
func (s *Service) Update(ctx context.Context, clientID, id uuid.UUID, input Input, actor auth.Principal) (Workspace, error) {
	if err := validate(&input, true); err != nil {
		return Workspace{}, err
	}
	item, err := scan(s.pool.QueryRow(ctx, `UPDATE workspaces SET name=$3,slug=$4,timezone=$5,version=version+1,updated_by=$6,updated_at=now() WHERE id=$1 AND client_id=$2 AND version=$7 RETURNING id,client_id,name,slug,timezone,status::text,version,created_at,updated_at`, id, clientID, input.Name, input.Slug, input.Timezone, actor.UserID, input.Version))
	if err != nil {
		return Workspace{}, mapDBError(err)
	}
	return item, nil
}
func (s *Service) SetStatus(ctx context.Context, clientID, id uuid.UUID, status string, version int64, actor auth.Principal) (Workspace, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if (status != "ACTIVE" && status != "ARCHIVED") || version < 1 {
		return Workspace{}, ErrInvalid
	}
	item, err := scan(s.pool.QueryRow(ctx, `UPDATE workspaces SET status=$3::lifecycle_status,archived_at=CASE WHEN $3='ARCHIVED' THEN now() ELSE NULL END,version=version+1,updated_by=$4,updated_at=now() WHERE id=$1 AND client_id=$2 AND version=$5 RETURNING id,client_id,name,slug,timezone,status::text,version,created_at,updated_at`, id, clientID, status, actor.UserID, version))
	if err != nil {
		return Workspace{}, mapDBError(err)
	}
	return item, nil
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (Workspace, error) {
	var i Workspace
	err := row.Scan(&i.ID, &i.ClientID, &i.Name, &i.Slug, &i.Timezone, &i.Status, &i.Version, &i.CreatedAt, &i.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Workspace{}, ErrNotFound
	}
	return i, err
}
func validate(i *Input, requireVersion bool) error {
	i.Name = strings.TrimSpace(i.Name)
	i.Slug = strings.ToLower(strings.TrimSpace(i.Slug))
	i.Timezone = strings.TrimSpace(i.Timezone)
	if i.Timezone == "" {
		i.Timezone = "Asia/Ho_Chi_Minh"
	}
	if len(i.Name) < 2 || len(i.Name) > 160 || !slugPattern.MatchString(i.Slug) || len(i.Timezone) > 100 || (requireVersion && i.Version < 1) {
		return ErrInvalid
	}
	return nil
}
func mapDBError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return fmt.Errorf("workspace database operation: %w", err)
}
