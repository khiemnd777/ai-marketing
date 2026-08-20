package internalusers

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
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
	ErrInvalidInput = errors.New("invalid internal user input")
	ErrEmailExists  = errors.New("internal user email already exists")
	ErrNotFound     = errors.New("internal user not found")
	ErrConflict     = errors.New("internal user version conflict")
	ErrSelfDisable  = errors.New("administrator cannot disable own account")
	ErrLastAdmin    = errors.New("cannot disable the last active administrator")
)

type CreateInput struct {
	Email             string
	DisplayName       string
	Role              db.InternalUserRole
	TemporaryPassword string
}

type UpdateInput struct {
	Email       string
	DisplayName string
	Role        db.InternalUserRole
	Version     int64
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
	Version                int64
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

func (s *Service) Update(ctx context.Context, userID uuid.UUID, input UpdateInput, actor auth.Principal, metadata auth.ClientMetadata) (User, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if _, err := mail.ParseAddress(input.Email); err != nil || len(input.Email) > 320 || len(input.DisplayName) < 2 || len(input.DisplayName) > 120 || input.Version < 1 || !validRole(input.Role) {
		return User{}, ErrInvalidInput
	}
	preview, err := s.queries.GetInternalUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("load internal user update target: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin internal user update transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	if preview.Role == db.InternalUserRoleADMIN && preview.Status == db.InternalUserStatusACTIVE && input.Role != db.InternalUserRoleADMIN {
		if _, err = queries.LockInternalAdminUsers(ctx); err != nil {
			return User{}, fmt.Errorf("lock administrator set: %w", err)
		}
	}
	current, err := queries.GetInternalUserByIDForUpdate(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("lock internal user update target: %w", err)
	}
	if current.Version != input.Version {
		return User{}, ErrConflict
	}
	if current.Role == db.InternalUserRoleADMIN && current.Status == db.InternalUserStatusACTIVE && input.Role != db.InternalUserRoleADMIN {
		remaining, countErr := queries.CountOtherActiveAdmins(ctx, userID)
		if countErr != nil {
			return User{}, fmt.Errorf("count active administrators: %w", countErr)
		}
		if remaining == 0 {
			return User{}, ErrLastAdmin
		}
	}
	updated, err := queries.UpdateInternalUserProfileVersioned(ctx, db.UpdateInternalUserProfileVersionedParams{
		Email: input.Email, DisplayName: input.DisplayName, Role: input.Role, ID: userID, Version: input.Version,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrConflict
	}
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == "23505" {
			return User{}, ErrEmailExists
		}
		return User{}, fmt.Errorf("update internal user: %w", err)
	}
	actorID := uuid.NullUUID{UUID: actor.UserID, Valid: true}
	targetID := uuid.NullUUID{UUID: userID, Valid: true}
	if err = audit.Record(ctx, queries, audit.Event{
		ActorID: actorID, Action: "internal_user.updated", EntityType: "internal_user", EntityID: targetID,
		RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS",
		Before: map[string]any{"email": current.Email, "displayName": current.DisplayName, "role": current.Role, "version": current.Version},
		After:  map[string]any{"email": updated.Email, "displayName": updated.DisplayName, "role": updated.Role, "version": updated.Version},
	}); err != nil {
		return User{}, fmt.Errorf("write internal user update audit: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit internal user update: %w", err)
	}
	return mapUser(updated), nil
}

func (s *Service) ResetPassword(ctx context.Context, userID uuid.UUID, version int64, temporaryPassword string, actor auth.Principal, metadata auth.ClientMetadata) (User, error) {
	if version < 1 || len(temporaryPassword) < 14 || len(temporaryPassword) > 200 {
		return User{}, ErrInvalidInput
	}
	passwordHash, err := auth.HashPassword(temporaryPassword, auth.DefaultArgon2Params)
	if err != nil {
		return User{}, fmt.Errorf("hash reset password: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin password reset transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	current, err := queries.GetInternalUserByIDForUpdate(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("load reset target: %w", err)
	}
	if current.Version != version {
		return User{}, ErrConflict
	}
	updated, err := queries.UpdateInternalUserPasswordVersioned(ctx, db.UpdateInternalUserPasswordVersionedParams{
		PasswordHash: passwordHash, RequiresPasswordChange: true, ID: userID, Version: version,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrConflict
	}
	if err != nil {
		return User{}, fmt.Errorf("reset internal user password: %w", err)
	}
	reason := "admin_password_reset"
	revoked, err := queries.RevokeAllUserSessions(ctx, db.RevokeAllUserSessionsParams{Reason: &reason, InternalUserID: userID})
	if err != nil {
		return User{}, fmt.Errorf("revoke reset user sessions: %w", err)
	}
	actorID := uuid.NullUUID{UUID: actor.UserID, Valid: true}
	targetID := uuid.NullUUID{UUID: userID, Valid: true}
	if err = audit.Record(ctx, queries, audit.Event{
		ActorID: actorID, Action: "internal_user.password.reset", EntityType: "internal_user", EntityID: targetID,
		RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS",
		After: map[string]any{"requiresPasswordChange": true, "sessionsRevoked": revoked},
	}); err != nil {
		return User{}, fmt.Errorf("write password reset audit: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit password reset: %w", err)
	}
	return mapUser(updated), nil
}

func (s *Service) SetStatus(ctx context.Context, userID uuid.UUID, status db.InternalUserStatus, version int64, actor auth.Principal, metadata auth.ClientMetadata) (User, error) {
	if version < 1 || (status != db.InternalUserStatusACTIVE && status != db.InternalUserStatusDISABLED) {
		return User{}, ErrInvalidInput
	}
	if userID == actor.UserID && status == db.InternalUserStatusDISABLED {
		return User{}, ErrSelfDisable
	}
	preview, err := s.queries.GetInternalUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("load status target: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin status transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	if preview.Role == db.InternalUserRoleADMIN && status == db.InternalUserStatusDISABLED {
		if _, err = queries.LockInternalAdminUsers(ctx); err != nil {
			return User{}, fmt.Errorf("lock administrator set: %w", err)
		}
	}
	current, err := queries.GetInternalUserByIDForUpdate(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("lock status target: %w", err)
	}
	if current.Version != version {
		return User{}, ErrConflict
	}
	if current.Status == status {
		return mapUser(current), nil
	}
	if current.Role == db.InternalUserRoleADMIN && status == db.InternalUserStatusDISABLED {
		remaining, countErr := queries.CountOtherActiveAdmins(ctx, userID)
		if countErr != nil {
			return User{}, fmt.Errorf("count active administrators: %w", countErr)
		}
		if remaining == 0 {
			return User{}, ErrLastAdmin
		}
	}
	updated, err := queries.SetInternalUserStatusVersioned(ctx, db.SetInternalUserStatusVersionedParams{Status: status, ID: userID, Version: version})
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrConflict
	}
	if err != nil {
		return User{}, fmt.Errorf("set internal user status: %w", err)
	}
	revoked := int64(0)
	if status == db.InternalUserStatusDISABLED {
		reason := "account_disabled"
		revoked, err = queries.RevokeAllUserSessions(ctx, db.RevokeAllUserSessionsParams{Reason: &reason, InternalUserID: userID})
		if err != nil {
			return User{}, fmt.Errorf("revoke disabled user sessions: %w", err)
		}
	}
	actorID := uuid.NullUUID{UUID: actor.UserID, Valid: true}
	targetID := uuid.NullUUID{UUID: userID, Valid: true}
	if err = audit.Record(ctx, queries, audit.Event{
		ActorID: actorID, Action: "internal_user.status.changed", EntityType: "internal_user", EntityID: targetID,
		RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS",
		Before: map[string]any{"status": current.Status, "version": current.Version},
		After:  map[string]any{"status": updated.Status, "version": updated.Version, "sessionsRevoked": revoked},
	}); err != nil {
		return User{}, fmt.Errorf("write status audit: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit status change: %w", err)
	}
	return mapUser(updated), nil
}

func mapUser(user db.InternalUser) User {
	var lastLogin *time.Time
	if user.LastLoginAt.Valid {
		value := user.LastLoginAt.Time
		lastLogin = &value
	}
	return User{ID: user.ID, Email: user.Email, DisplayName: user.DisplayName, Role: user.Role, Status: user.Status, RequiresPasswordChange: user.RequiresPasswordChange, LastLoginAt: lastLogin, CreatedAt: user.CreatedAt.Time, UpdatedAt: user.UpdatedAt.Time, Version: user.Version}
}

func validRole(role db.InternalUserRole) bool {
	return role == db.InternalUserRoleADMIN || role == db.InternalUserRoleOPERATOR || role == db.InternalUserRoleREVIEWER
}
