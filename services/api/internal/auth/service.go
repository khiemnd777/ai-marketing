package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/mail"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/audit"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountDisabled    = errors.New("account disabled")
	ErrAccountLocked      = errors.New("account temporarily locked")
	ErrUnauthenticated    = errors.New("authentication required")
	ErrInvalidCSRF        = errors.New("invalid CSRF token")
	ErrInvalidPassword    = errors.New("current password is invalid")
	ErrPasswordPolicy     = errors.New("new password does not meet policy")
	ErrConflict           = errors.New("authentication state conflict")
	ErrSessionNotFound    = errors.New("session not found")
	ErrBootstrapClosed    = errors.New("admin bootstrap is no longer available")
	ErrBootstrapInput     = errors.New("invalid admin bootstrap input")
	ErrEmailInUse         = errors.New("email is already in use")
)

type ClientMetadata struct {
	RequestID string
	IPAddress *netip.Addr
	UserAgent string
}

type Principal struct {
	SessionID              uuid.UUID
	UserID                 uuid.UUID
	Email                  string
	DisplayName            string
	Role                   db.InternalUserRole
	RequiresPasswordChange bool
	Version                int64
	CSRFHash               []byte
	CreatedAt              time.Time
	UpdatedAt              time.Time
	LastLoginAt            *time.Time
}

type LoginResult struct {
	Principal    Principal
	SessionToken string
	CSRFToken    string
	ExpiresAt    time.Time
}

type BootstrapInput struct {
	Email       string
	DisplayName string
	Password    string
}

type SessionInfo struct {
	ID         uuid.UUID
	IPAddress  string
	UserAgent  string
	ExpiresAt  time.Time
	LastSeenAt time.Time
	CreatedAt  time.Time
	Current    bool
}

type Service struct {
	pool       *pgxpool.Pool
	queries    *db.Queries
	secret     []byte
	sessionTTL time.Duration
	dummyHash  string
	now        func() time.Time
}

func NewService(pool *pgxpool.Pool, secret []byte, sessionTTL time.Duration) (*Service, error) {
	dummyHash, err := HashPassword("invalid-credential-probe", DefaultArgon2Params)
	if err != nil {
		return nil, fmt.Errorf("create password timing sentinel: %w", err)
	}
	return &Service{
		pool: pool, queries: db.New(pool), secret: secret, sessionTTL: sessionTTL,
		dummyHash: dummyHash, now: func() time.Time { return time.Now().UTC() },
	}, nil
}

func (s *Service) BootstrapRequired(ctx context.Context) (bool, error) {
	count, err := s.queries.CountAdminUsers(ctx)
	if err != nil {
		return false, fmt.Errorf("count admin users: %w", err)
	}
	return count == 0, nil
}

func (s *Service) BootstrapAdmin(ctx context.Context, input BootstrapInput, metadata ClientMetadata) (LoginResult, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if _, err := mail.ParseAddress(input.Email); err != nil || len(input.Email) > 320 || len(input.DisplayName) < 2 || len(input.DisplayName) > 120 || len(input.Password) < 14 || len(input.Password) > 200 {
		return LoginResult{}, ErrBootstrapInput
	}
	passwordHash, err := HashPassword(input.Password, DefaultArgon2Params)
	if err != nil {
		return LoginResult{}, fmt.Errorf("hash bootstrap password: %w", err)
	}
	sessionToken, err := NewOpaqueToken()
	if err != nil {
		return LoginResult{}, err
	}
	csrfToken, err := NewOpaqueToken()
	if err != nil {
		return LoginResult{}, err
	}
	expiresAt := s.now().Add(s.sessionTTL)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return LoginResult{}, fmt.Errorf("begin admin bootstrap transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	if err = queries.LockInternalUsersForAdminBootstrap(ctx); err != nil {
		return LoginResult{}, fmt.Errorf("lock admin bootstrap: %w", err)
	}
	adminCount, err := queries.CountAdminUsers(ctx)
	if err != nil {
		return LoginResult{}, fmt.Errorf("count admin users for bootstrap: %w", err)
	}
	if adminCount != 0 {
		return LoginResult{}, ErrBootstrapClosed
	}
	created, err := queries.CreateInternalUser(ctx, db.CreateInternalUserParams{
		Email: input.Email, DisplayName: input.DisplayName, PasswordHash: passwordHash,
		Role: db.InternalUserRoleADMIN, RequiresPasswordChange: false,
	})
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == "23505" {
			return LoginResult{}, ErrEmailInUse
		}
		return LoginResult{}, fmt.Errorf("create bootstrap admin: %w", err)
	}
	session, err := queries.CreateSession(ctx, db.CreateSessionParams{
		InternalUserID: created.ID,
		TokenHash:      TokenDigest(s.secret, sessionToken),
		CsrfHash:       TokenDigest(s.secret, csrfToken),
		IpAddress:      metadata.IPAddress,
		UserAgent:      truncate(metadata.UserAgent, 1000),
		ExpiresAt:      pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return LoginResult{}, fmt.Errorf("create bootstrap session: %w", err)
	}
	actor := uuid.NullUUID{UUID: created.ID, Valid: true}
	if err = audit.Record(ctx, queries, audit.Event{
		ActorID: actor, Action: "auth.bootstrap.succeeded", EntityType: "internal_user", EntityID: actor,
		RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: truncate(metadata.UserAgent, 1000),
		Outcome: "SUCCESS", After: map[string]any{"email": created.Email, "role": created.Role, "status": created.Status},
	}); err != nil {
		return LoginResult{}, fmt.Errorf("write admin bootstrap audit: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return LoginResult{}, fmt.Errorf("commit admin bootstrap: %w", err)
	}
	return LoginResult{
		Principal: Principal{
			SessionID: session.ID, UserID: created.ID, Email: created.Email, DisplayName: created.DisplayName,
			Role: created.Role, RequiresPasswordChange: created.RequiresPasswordChange, Version: created.Version,
			CSRFHash: session.CsrfHash, CreatedAt: created.CreatedAt.Time, UpdatedAt: created.UpdatedAt.Time,
		},
		SessionToken: sessionToken, CSRFToken: csrfToken, ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) Login(ctx context.Context, email, password string, metadata ClientMetadata) (LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.queries.GetInternalUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		_, _ = VerifyPassword(password, s.dummyHash)
		_ = s.auditFailure(ctx, uuid.NullUUID{}, "auth.login.failed", "invalid_credentials", metadata)
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, fmt.Errorf("load internal user: %w", err)
	}
	actor := uuid.NullUUID{UUID: user.ID, Valid: true}
	if user.Status != db.InternalUserStatusACTIVE {
		_ = s.auditFailure(ctx, actor, "auth.login.denied", "account_disabled", metadata)
		return LoginResult{}, ErrAccountDisabled
	}
	if user.LockedUntil.Valid && user.LockedUntil.Time.After(s.now()) {
		_ = s.auditFailure(ctx, actor, "auth.login.denied", "account_locked", metadata)
		return LoginResult{}, ErrAccountLocked
	}

	valid, verifyErr := VerifyPassword(password, user.PasswordHash)
	if verifyErr != nil || !valid {
		interval := pgtype.Interval{Microseconds: int64((15 * time.Minute) / time.Microsecond), Valid: true}
		_ = s.queries.RecordFailedLogin(ctx, db.RecordFailedLoginParams{LockThreshold: 5, LockDuration: interval, ID: user.ID})
		_ = s.auditFailure(ctx, actor, "auth.login.failed", "invalid_credentials", metadata)
		return LoginResult{}, ErrInvalidCredentials
	}

	sessionToken, err := NewOpaqueToken()
	if err != nil {
		return LoginResult{}, err
	}
	csrfToken, err := NewOpaqueToken()
	if err != nil {
		return LoginResult{}, err
	}
	expiresAt := s.now().Add(s.sessionTTL)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return LoginResult{}, fmt.Errorf("begin login transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	if err := queries.RecordSuccessfulLogin(ctx, user.ID); err != nil {
		return LoginResult{}, fmt.Errorf("record successful login: %w", err)
	}
	session, err := queries.CreateSession(ctx, db.CreateSessionParams{
		InternalUserID: user.ID,
		TokenHash:      TokenDigest(s.secret, sessionToken),
		CsrfHash:       TokenDigest(s.secret, csrfToken),
		IpAddress:      metadata.IPAddress,
		UserAgent:      truncate(metadata.UserAgent, 1000),
		ExpiresAt:      pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return LoginResult{}, fmt.Errorf("create session: %w", err)
	}
	if err := audit.Record(ctx, queries, audit.Event{
		ActorID: actor, Action: "auth.login.succeeded", EntityType: "internal_user",
		EntityID: actor, RequestID: metadata.RequestID, IPAddress: metadata.IPAddress,
		UserAgent: truncate(metadata.UserAgent, 1000), Outcome: "SUCCESS",
	}); err != nil {
		return LoginResult{}, fmt.Errorf("write login audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return LoginResult{}, fmt.Errorf("commit login transaction: %w", err)
	}

	lastLogin := s.now()
	return LoginResult{
		Principal: Principal{
			SessionID: session.ID, UserID: user.ID, Email: user.Email, DisplayName: user.DisplayName,
			Role: user.Role, RequiresPasswordChange: user.RequiresPasswordChange, Version: user.Version,
			CSRFHash: session.CsrfHash, CreatedAt: user.CreatedAt.Time, UpdatedAt: user.UpdatedAt.Time, LastLoginAt: &lastLogin,
		},
		SessionToken: sessionToken, CSRFToken: csrfToken, ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (Principal, error) {
	if token == "" {
		return Principal{}, ErrUnauthenticated
	}
	row, err := s.queries.GetActiveSessionByTokenHash(ctx, TokenDigest(s.secret, token))
	if errors.Is(err, pgx.ErrNoRows) {
		return Principal{}, ErrUnauthenticated
	}
	if err != nil {
		return Principal{}, fmt.Errorf("load session: %w", err)
	}
	_ = s.queries.TouchSession(ctx, row.SessionID)
	var lastLogin *time.Time
	if row.LastLoginAt.Valid {
		value := row.LastLoginAt.Time
		lastLogin = &value
	}
	return Principal{
		SessionID: row.SessionID, UserID: row.InternalUserID, Email: row.Email, DisplayName: row.DisplayName,
		Role: row.Role, RequiresPasswordChange: row.RequiresPasswordChange, Version: row.Version,
		CSRFHash: row.CsrfHash, CreatedAt: row.UserCreatedAt.Time, UpdatedAt: row.UserUpdatedAt.Time, LastLoginAt: lastLogin,
	}, nil
}

func (s *Service) ValidateCSRF(principal Principal, supplied string) error {
	digest := TokenDigest(s.secret, supplied)
	if len(principal.CSRFHash) != len(digest) || subtle.ConstantTimeCompare(principal.CSRFHash, digest) != 1 {
		return ErrInvalidCSRF
	}
	return nil
}

func (s *Service) Logout(ctx context.Context, principal Principal, metadata ClientMetadata) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin logout transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	reason := "user_logout"
	if _, err := queries.RevokeSession(ctx, db.RevokeSessionParams{ID: principal.SessionID, Reason: &reason}); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	actor := uuid.NullUUID{UUID: principal.UserID, Valid: true}
	if err := audit.Record(ctx, queries, audit.Event{
		ActorID: actor, Action: "auth.logout", EntityType: "session",
		EntityID: uuid.NullUUID{UUID: principal.SessionID, Valid: true}, RequestID: metadata.RequestID,
		IPAddress: metadata.IPAddress, UserAgent: truncate(metadata.UserAgent, 1000), Outcome: "SUCCESS",
	}); err != nil {
		return fmt.Errorf("write logout audit: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Service) ChangePassword(ctx context.Context, principal Principal, currentPassword, newPassword string, metadata ClientMetadata) error {
	if len(currentPassword) < 10 || len(currentPassword) > 200 || len(newPassword) < 14 || len(newPassword) > 200 || currentPassword == newPassword {
		return ErrPasswordPolicy
	}
	passwordHash, err := HashPassword(newPassword, DefaultArgon2Params)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin password transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	user, err := queries.GetInternalUserByIDForUpdate(ctx, principal.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUnauthenticated
	}
	if err != nil {
		return fmt.Errorf("load password owner: %w", err)
	}
	valid, verifyErr := VerifyPassword(currentPassword, user.PasswordHash)
	if verifyErr != nil || !valid {
		return ErrInvalidPassword
	}
	if user.Version != principal.Version {
		return ErrConflict
	}
	if _, err = queries.UpdateInternalUserPasswordVersioned(ctx, db.UpdateInternalUserPasswordVersionedParams{
		PasswordHash: passwordHash, RequiresPasswordChange: false, ID: user.ID, Version: user.Version,
	}); errors.Is(err, pgx.ErrNoRows) {
		return ErrConflict
	} else if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	reason := "password_changed"
	if _, err = queries.RevokeOtherUserSessions(ctx, db.RevokeOtherUserSessionsParams{
		Reason: &reason, InternalUserID: user.ID, CurrentSessionID: principal.SessionID,
	}); err != nil {
		return fmt.Errorf("revoke other sessions: %w", err)
	}
	actor := uuid.NullUUID{UUID: principal.UserID, Valid: true}
	if err = audit.Record(ctx, queries, audit.Event{
		ActorID: actor, Action: "auth.password.changed", EntityType: "internal_user", EntityID: actor,
		RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: truncate(metadata.UserAgent, 1000),
		Outcome: "SUCCESS", After: map[string]any{"requiresPasswordChange": false, "otherSessionsRevoked": true},
	}); err != nil {
		return fmt.Errorf("write password audit: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Service) ListSessions(ctx context.Context, principal Principal) ([]SessionInfo, error) {
	rows, err := s.queries.ListActiveUserSessions(ctx, principal.UserID)
	if err != nil {
		return nil, fmt.Errorf("list active sessions: %w", err)
	}
	items := make([]SessionInfo, 0, len(rows))
	for _, row := range rows {
		address := ""
		if row.IpAddress != nil {
			address = row.IpAddress.String()
		}
		items = append(items, SessionInfo{
			ID: row.ID, IPAddress: address, UserAgent: row.UserAgent, ExpiresAt: row.ExpiresAt.Time,
			LastSeenAt: row.LastSeenAt.Time, CreatedAt: row.CreatedAt.Time, Current: row.ID == principal.SessionID,
		})
	}
	return items, nil
}

func (s *Service) RevokeOwnSession(ctx context.Context, principal Principal, sessionID uuid.UUID, metadata ClientMetadata) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin session revocation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	reason := "user_revoked"
	affected, err := queries.RevokeUserSession(ctx, db.RevokeUserSessionParams{Reason: &reason, ID: sessionID, InternalUserID: principal.UserID})
	if err != nil {
		return false, fmt.Errorf("revoke user session: %w", err)
	}
	if affected != 1 {
		return false, ErrSessionNotFound
	}
	actor := uuid.NullUUID{UUID: principal.UserID, Valid: true}
	if err = audit.Record(ctx, queries, audit.Event{
		ActorID: actor, Action: "auth.session.revoked", EntityType: "session",
		EntityID: uuid.NullUUID{UUID: sessionID, Valid: true}, RequestID: metadata.RequestID,
		IPAddress: metadata.IPAddress, UserAgent: truncate(metadata.UserAgent, 1000), Outcome: "SUCCESS",
		After: map[string]any{"currentSession": sessionID == principal.SessionID},
	}); err != nil {
		return false, fmt.Errorf("write session revocation audit: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit session revocation: %w", err)
	}
	return sessionID == principal.SessionID, nil
}

func (s *Service) auditFailure(ctx context.Context, actor uuid.NullUUID, action, reason string, metadata ClientMetadata) error {
	return audit.Record(ctx, s.queries, audit.Event{
		ActorID: actor, Action: action, EntityType: "internal_user", RequestID: metadata.RequestID,
		IPAddress: metadata.IPAddress, UserAgent: truncate(metadata.UserAgent, 1000), Outcome: "FAILURE", Reason: reason,
	})
}

func truncate(value string, maximum int) string {
	if len(value) <= maximum {
		return value
	}
	return value[:maximum]
}
