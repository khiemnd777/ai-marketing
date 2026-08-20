package providerconfigs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/audit"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/cryptox"
)

type Service struct {
	pool   *pgxpool.Pool
	cipher *cryptox.Cipher
}

func NewService(pool *pgxpool.Pool, cipher *cryptox.Cipher) *Service {
	return &Service{pool: pool, cipher: cipher}
}

func (s *Service) Get(ctx context.Context, clientID uuid.UUID) (Profile, error) {
	if clientID == uuid.Nil || s.cipher == nil {
		return Profile{}, ErrInvalid
	}
	queries := db.New(s.pool)
	if _, err := s.clientExists(ctx, s.pool, clientID); err != nil {
		return Profile{}, err
	}
	profile := Profile{ClientID: clientID, DemoMode: true, Version: 0}
	storedProfile, err := queries.GetClientProviderProfile(ctx, clientID)
	if err == nil {
		profile.DemoMode, profile.Version = storedProfile.DemoMode, storedProfile.Version
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, err
	}
	rows, err := queries.ListProviderConfigurationsByClient(ctx, clientID)
	if err != nil {
		return Profile{}, err
	}
	byKind := make(map[string]db.ProviderConfiguration, len(rows))
	for _, row := range rows {
		byKind[string(row.Provider)] = row
	}
	for _, kind := range kinds {
		view := ProviderView{Provider: kind, Enabled: true, Settings: defaultSettings(kind), ConfiguredSecretFields: []string{}, Version: 0}
		if row, ok := byKind[kind]; ok {
			view.Enabled, view.Version = row.Enabled, row.Version
			if err = json.Unmarshal(row.SafeConfig, &view.Settings); err != nil {
				return Profile{}, fmt.Errorf("decode %s provider settings: %w", kind, err)
			}
			view.ConfiguredSecretFields = append([]string(nil), row.ConfiguredSecretFields...)
			sort.Strings(view.ConfiguredSecretFields)
			bundle := Bundle{ClientID: clientID, DemoMode: profile.DemoMode}
			secrets, decryptErr := s.decrypt(row)
			if decryptErr == nil && bundleConfiguration(kind, view.Settings, secrets, &bundle) == nil && row.Enabled && validateConfigured(kind, bundle) == nil {
				view.Configured = true
			}
		}
		profile.Providers = append(profile.Providers, view)
	}
	return profile, nil
}

func (s *Service) Load(ctx context.Context, clientID uuid.UUID) (Bundle, error) {
	if clientID == uuid.Nil || s.cipher == nil {
		return Bundle{}, ErrInvalid
	}
	queries := db.New(s.pool)
	bundle := Bundle{ClientID: clientID, DemoMode: true}
	profile, err := queries.GetClientProviderProfile(ctx, clientID)
	if err == nil {
		bundle.DemoMode = profile.DemoMode
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Bundle{}, err
	}
	rows, err := queries.ListProviderConfigurationsByClient(ctx, clientID)
	if err != nil {
		return Bundle{}, err
	}
	for _, row := range rows {
		if !row.Enabled {
			continue
		}
		settings := map[string]any{}
		if err = json.Unmarshal(row.SafeConfig, &settings); err != nil {
			return Bundle{}, fmt.Errorf("decode provider settings: %w", err)
		}
		secrets, decryptErr := s.decrypt(row)
		if decryptErr != nil {
			return Bundle{}, fmt.Errorf("decrypt provider secrets: %w", decryptErr)
		}
		if err = bundleConfiguration(string(row.Provider), settings, secrets, &bundle); err != nil {
			return Bundle{}, err
		}
	}
	return bundle, nil
}

func (s *Service) SaveMode(ctx context.Context, clientID uuid.UUID, input ModeInput, actor auth.Principal, metadata auth.ClientMetadata) (Profile, error) {
	if clientID == uuid.Nil || actor.UserID == uuid.Nil || input.Version < 0 {
		return Profile{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Profile{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = s.clientExists(ctx, tx, clientID); err != nil {
		return Profile{}, err
	}
	queries := db.New(tx)
	current, loadErr := queries.GetClientProviderProfile(ctx, clientID)
	if errors.Is(loadErr, pgx.ErrNoRows) {
		if input.Version != 0 {
			return Profile{}, ErrConflict
		}
	} else if loadErr != nil {
		return Profile{}, loadErr
	} else if current.Version != input.Version {
		return Profile{}, ErrConflict
	}
	stored, err := queries.UpsertClientProviderProfile(ctx, db.UpsertClientProviderProfileParams{ClientID: clientID, DemoMode: input.DemoMode, ActorID: actor.UserID, Version: input.Version})
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrConflict
	}
	if err != nil {
		return Profile{}, err
	}
	if err = audit.Record(ctx, queries, audit.Event{ActorID: valid(actor.UserID), Action: "provider.mode.updated", EntityType: "client", EntityID: valid(clientID), ClientID: valid(clientID), RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS", After: map[string]any{"demoMode": stored.DemoMode, "version": stored.Version}}); err != nil {
		return Profile{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Profile{}, err
	}
	return s.Get(ctx, clientID)
}

func (s *Service) Save(ctx context.Context, clientID uuid.UUID, provider string, input SaveInput, actor auth.Principal, metadata auth.ClientMetadata) (Profile, error) {
	kind, ok := normalizeKind(provider)
	if !ok || clientID == uuid.Nil || actor.UserID == uuid.Nil || input.Version < 0 || s.cipher == nil || validateSecrets(kind, input.Secrets) != nil {
		return Profile{}, ErrInvalid
	}
	allowedSecrets := map[string]bool{}
	for _, name := range secretNames(kind) {
		allowedSecrets[name] = true
	}
	for _, name := range input.ClearSecrets {
		if !allowedSecrets[name] {
			return Profile{}, ErrInvalid
		}
	}
	settings := input.Settings
	if settings == nil {
		settings = defaultSettings(kind)
	}
	testBundle := Bundle{ClientID: clientID, DemoMode: true}
	if err := bundleConfiguration(kind, settings, map[string]string{}, &testBundle); err != nil {
		return Profile{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Profile{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = s.clientExists(ctx, tx, clientID); err != nil {
		return Profile{}, err
	}
	queries := db.New(tx)
	secrets := map[string]string{}
	var before any
	row, loadErr := queries.GetProviderConfiguration(ctx, db.GetProviderConfigurationParams{ClientID: clientID, Provider: db.ProviderKind(kind)})
	if loadErr == nil {
		if input.Version == 0 || row.Version != input.Version {
			return Profile{}, ErrConflict
		}
		secrets, err = s.decrypt(row)
		if err != nil {
			return Profile{}, err
		}
		before = safeAuditView(row)
	} else if !errors.Is(loadErr, pgx.ErrNoRows) {
		return Profile{}, loadErr
	} else if input.Version != 0 {
		return Profile{}, ErrConflict
	}
	for name, value := range input.Secrets {
		value = strings.TrimSpace(value)
		if value != "" {
			secrets[name] = value
		}
	}
	for _, name := range input.ClearSecrets {
		delete(secrets, name)
	}
	secretRaw, err := json.Marshal(secrets)
	if err != nil {
		return Profile{}, err
	}
	ciphertext, nonce, err := s.cipher.Encrypt(secretRaw, associatedData(clientID, kind))
	if err != nil {
		return Profile{}, err
	}
	safeRaw, err := json.Marshal(settings)
	if err != nil {
		return Profile{}, ErrInvalid
	}
	stored, err := queries.UpsertProviderConfiguration(ctx, db.UpsertProviderConfigurationParams{ClientID: clientID, Provider: db.ProviderKind(kind), Enabled: input.Enabled, SafeConfig: safeRaw, SecretCiphertext: ciphertext, SecretNonce: nonce, ConfiguredSecretFields: configuredSecretFields(secrets), ActorID: actor.UserID, Version: input.Version})
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrConflict
	}
	if err != nil {
		return Profile{}, err
	}
	if err = audit.Record(ctx, queries, audit.Event{ActorID: valid(actor.UserID), Action: "provider.configuration.updated", EntityType: "provider_configuration", EntityID: valid(stored.ID), ClientID: valid(clientID), RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS", Before: before, After: safeAuditView(stored), Metadata: map[string]any{"provider": kind}}); err != nil {
		return Profile{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Profile{}, err
	}
	return s.Get(ctx, clientID)
}

func (s *Service) decrypt(row db.ProviderConfiguration) (map[string]string, error) {
	plaintext, err := s.cipher.Decrypt(row.SecretCiphertext, row.SecretNonce, associatedData(row.ClientID, string(row.Provider)))
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	if err = json.Unmarshal(plaintext, &values); err != nil {
		return nil, err
	}
	return values, nil
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *Service) clientExists(ctx context.Context, q queryRower, clientID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := q.QueryRow(ctx, `SELECT id FROM clients WHERE id=$1`, clientID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return id, err
}

func safeAuditView(row db.ProviderConfiguration) map[string]any {
	settings := map[string]any{}
	_ = json.Unmarshal(row.SafeConfig, &settings)
	return map[string]any{"provider": row.Provider, "enabled": row.Enabled, "settings": settings, "configuredSecretFields": row.ConfiguredSecretFields, "version": row.Version}
}

func valid(id uuid.UUID) uuid.NullUUID { return uuid.NullUUID{UUID: id, Valid: id != uuid.Nil} }
