package internalusers

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/audit"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
)

var (
	ErrInvalidInput = errors.New("invalid internal user input")
	ErrEmailExists  = errors.New("internal user email already exists")
)

type CreateInput struct {
	Email             string
	DisplayName       string
	Role              db.InternalUserRole
	TemporaryPassword string
}

type User struct {
	ID                     uuid.UUID
	Email                  string
	DisplayName            string
	Role                   db.InternalUserRole
	Status                 db.InternalUserStatus
	RequiresPasswordChange bool
	LastLoginAt            *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type Page struct {
	Items      []User
	Number     int
	Size       int
	TotalItems int64
	TotalPages int64
}

type Service struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, queries: db.New(pool)}
}

func (s *Service) Create(ctx context.Context, input CreateInput, actor auth.Principal, metadata auth.ClientMetadata) (User, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if _, err := mail.ParseAddress(input.Email); err != nil || len(input.Email) > 320 || len(input.DisplayName) < 2 || len(input.DisplayName) > 120 || len(input.TemporaryPassword) < 14 || len(input.TemporaryPassword) > 200 || !validRole(input.Role) {
		return User{}, ErrInvalidInput
	}
	passwordHash, err := auth.HashPassword(input.TemporaryPassword, auth.DefaultArgon2Params)
	if err != nil {
		return User{}, fmt.Errorf("hash temporary password: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin create internal user transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	created, err := queries.CreateInternalUser(ctx, db.CreateInternalUserParams{
		Email: input.Email, DisplayName: input.DisplayName, PasswordHash: passwordHash,
		Role: input.Role, RequiresPasswordChange: true,
	})
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == "23505" {
			return User{}, ErrEmailExists
		}
		return User{}, fmt.Errorf("insert internal user: %w", err)
	}
	actorID := uuid.NullUUID{UUID: actor.UserID, Valid: true}
	createdID := uuid.NullUUID{UUID: created.ID, Valid: true}
	if err := audit.Record(ctx, queries, audit.Event{
		ActorID: actorID, Action: "internal_user.created", EntityType: "internal_user", EntityID: createdID,
		RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent,
		Outcome: "SUCCESS", After: map[string]any{"email": created.Email, "role": created.Role, "status": created.Status},
	}); err != nil {
		return User{}, fmt.Errorf("write internal user audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit create internal user: %w", err)
	}
	return mapUser(created), nil
}

func (s *Service) List(ctx context.Context, page, pageSize int) (Page, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 25
	}
	if pageSize > 100 {
		pageSize = 100
	}
	total, err := s.queries.CountInternalUsers(ctx)
	if err != nil {
		return Page{}, fmt.Errorf("count internal users: %w", err)
	}
	rows, err := s.queries.ListInternalUsers(ctx, db.ListInternalUsersParams{PageSize: int32(pageSize), PageOffset: int32((page - 1) * pageSize)})
	if err != nil {
		return Page{}, fmt.Errorf("list internal users: %w", err)
	}
	items := make([]User, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapUser(row))
	}
	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}
	return Page{Items: items, Number: page, Size: pageSize, TotalItems: total, TotalPages: totalPages}, nil
}

func mapUser(user db.InternalUser) User {
	var lastLogin *time.Time
	if user.LastLoginAt.Valid {
		value := user.LastLoginAt.Time
		lastLogin = &value
	}
	return User{ID: user.ID, Email: user.Email, DisplayName: user.DisplayName, Role: user.Role, Status: user.Status, RequiresPasswordChange: user.RequiresPasswordChange, LastLoginAt: lastLogin, CreatedAt: user.CreatedAt.Time, UpdatedAt: user.UpdatedAt.Time}
}

func validRole(role db.InternalUserRole) bool {
	return role == db.InternalUserRoleADMIN || role == db.InternalUserRoleOPERATOR || role == db.InternalUserRoleREVIEWER
}
