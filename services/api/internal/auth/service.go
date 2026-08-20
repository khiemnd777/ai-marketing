package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
