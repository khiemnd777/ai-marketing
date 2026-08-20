package metaconnections

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/meta"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/cryptox"
)

var (
	ErrNotFound = errors.New("Meta connection not found")
	ErrInvalid  = errors.New("invalid Meta connection operation")
	ErrConflict = errors.New("Meta connection conflict")
)

type SocialAccount struct {
	ID                  string   `json:"id"`
	Platform            string   `json:"platform"`
	ProviderAccountID   string   `json:"providerAccountId"`
	Name                string   `json:"name"`
	Status              string   `json:"status"`
	FacebookPageID      *string  `json:"facebookPageId"`
	InstagramBusinessID *string  `json:"instagramBusinessId"`
	Username            *string  `json:"username"`
	Tasks               []string `json:"tasks"`
}
type Business struct {
	ID                 string `json:"id"`
	ProviderBusinessID string `json:"providerBusinessId"`
	Name               string `json:"name"`
}
type AdAccount struct {
	ID                  string `json:"id"`
	ProviderAdAccountID string `json:"providerAdAccountId"`
	Name                string `json:"name"`
	Currency            string `json:"currency"`
	TimezoneName        string `json:"timezoneName"`
	AccountStatus       *int   `json:"accountStatus"`
}
type Pixel struct {
	ID              string `json:"id"`
	MetaAdAccountID string `json:"metaAdAccountId"`
	ProviderPixelID string `json:"providerPixelId"`
	Name            string `json:"name"`
}
type Audience struct {
	ID                 string `json:"id"`
	MetaAdAccountID    string `json:"metaAdAccountId"`
	ProviderAudienceID string `json:"providerAudienceId"`
	Name               string `json:"name"`
	AudienceType       string `json:"audienceType"`
	Subtype            string `json:"subtype"`
	ApproximateCount   *int64 `json:"approximateCount"`
}
type Connection struct {
	ID                  string          `json:"id"`
	MetaUserID          string          `json:"metaUserId"`
	DisplayName         string          `json:"displayName"`
	Status              string          `json:"status"`
	APIVersion          string          `json:"apiVersion"`
	Scopes              []string        `json:"scopes"`
	TokenIssuedAt       *time.Time      `json:"tokenIssuedAt"`
	TokenExpiresAt      *time.Time      `json:"tokenExpiresAt"`
	DataAccessExpiresAt *time.Time      `json:"dataAccessExpiresAt"`
	LastValidatedAt     *time.Time      `json:"lastValidatedAt"`
	Version             int64           `json:"version"`
	Accounts            []SocialAccount `json:"accounts"`
	Businesses          []Business      `json:"businesses"`
	AdAccounts          []AdAccount     `json:"adAccounts"`
	Pixels              []Pixel         `json:"pixels"`
	Audiences           []Audience      `json:"audiences"`
}
type OAuthStart struct {
	AuthorizationURL string `json:"authorizationUrl"`
}
type CallbackResult struct {
	ClientID    uuid.UUID
	WorkspaceID uuid.UUID
}

type Service struct {
	pool       *pgxpool.Pool
	cipher     *cryptox.Cipher
	provider   meta.Provider
	apiVersion string
}

func NewService(pool *pgxpool.Pool, cipher *cryptox.Cipher, provider meta.Provider, apiVersion string) *Service {
	return &Service{pool: pool, cipher: cipher, provider: provider, apiVersion: apiVersion}
}

func (s *Service) StartOAuth(ctx context.Context, clientID, workspaceID, actorID uuid.UUID) (OAuthStart, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE id=$1 AND client_id=$2 AND status='ACTIVE')`, workspaceID, clientID).Scan(&exists); err != nil {
		return OAuthStart{}, err
	}
	if !exists {
		return OAuthStart{}, ErrNotFound
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return OAuthStart{}, err
	}
	state := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(state))
	if _, err := s.pool.Exec(ctx, `INSERT INTO meta_oauth_states(state_hash,actor_id,client_id,workspace_id,expires_at)VALUES($1,$2,$3,$4,now()+interval '10 minutes')`, hex.EncodeToString(hash[:]), actorID, clientID, workspaceID); err != nil {
		return OAuthStart{}, err
	}
	url, err := s.provider.AuthorizationURL(state)
	if err != nil {
		return OAuthStart{}, err
	}
	return OAuthStart{AuthorizationURL: url}, nil
}

func (s *Service) Callback(ctx context.Context, state, code string) (CallbackResult, error) {
	if len(state) < 32 || code == "" {
		return CallbackResult{}, ErrInvalid
	}
	hash := sha256.Sum256([]byte(state))
	stateHash := hex.EncodeToString(hash[:])
	var result CallbackResult
	var actorID uuid.UUID
	if err := s.pool.QueryRow(ctx, `SELECT client_id,workspace_id,actor_id FROM meta_oauth_states WHERE state_hash=$1 AND consumed_at IS NULL AND expires_at>now()`, stateHash).Scan(&result.ClientID, &result.WorkspaceID, &actorID); errors.Is(err, pgx.ErrNoRows) {
		return CallbackResult{}, ErrInvalid
	} else if err != nil {
		return CallbackResult{}, err
	}
	token, err := s.provider.ExchangeCode(ctx, code)
	if err != nil {
		return CallbackResult{}, err
	}
	discovery, err := s.provider.Discover(ctx, token.AccessToken)
	if err != nil {
		return CallbackResult{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CallbackResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	update, err := tx.Exec(ctx, `UPDATE meta_oauth_states SET consumed_at=now() WHERE state_hash=$1 AND consumed_at IS NULL`, stateHash)
	if err != nil || update.RowsAffected() != 1 {
		return CallbackResult{}, ErrConflict
	}
	_, err = tx.Exec(ctx, `UPDATE meta_connections SET status='DISCONNECTED',disconnected_at=now(),token_ciphertext=''::bytea,token_nonce=''::bytea,version=version+1,updated_at=now() WHERE workspace_id=$1 AND disconnected_at IS NULL`, result.WorkspaceID)
	if err != nil {
		return CallbackResult{}, err
	}
	connectionID := uuid.New()
	ciphertext, nonce, err := s.cipher.Encrypt([]byte(token.AccessToken), "meta-connection:"+connectionID.String())
	if err != nil {
		return CallbackResult{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO meta_connections(id,client_id,workspace_id,meta_user_id,display_name,token_ciphertext,token_nonce,token_type,scopes,token_issued_at,token_expires_at,data_access_expires_at,api_version,status,last_validated_at,created_by,updated_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,'CONNECTED',now(),$14,$14)`, connectionID, result.ClientID, result.WorkspaceID, token.UserID, token.UserName, ciphertext, nonce, token.TokenType, token.Scopes, token.IssuedAt, token.ExpiresAt, token.DataAccessExpiresAt, s.apiVersion, actorID)
	if err != nil {
		return CallbackResult{}, err
	}
	if err = s.persistDiscovery(ctx, tx, connectionID, result.ClientID, result.WorkspaceID, discovery); err != nil {
		return CallbackResult{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_logs(actor_internal_user_id,action,entity_type,entity_id,client_id,workspace_id,request_id,outcome,metadata)VALUES($1,'meta.connection.oauth_callback','META_CONNECTION',$2,$3,$4,$5,'SUCCESS',$6)`, actorID, connectionID, result.ClientID, result.WorkspaceID, "meta-oauth:"+stateHash[:16], map[string]any{"apiVersion": s.apiVersion, "pages": len(discovery.Pages), "adAccounts": len(discovery.AdAccounts)})
	if err != nil {
		return CallbackResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return CallbackResult{}, err
	}
	return result, nil
}

func (s *Service) Sync(ctx context.Context, clientID, workspaceID uuid.UUID) (Connection, error) {
	id, token, err := s.connectionToken(ctx, clientID, workspaceID)
	if err != nil {
		return Connection{}, err
	}
	discovery, err := s.provider.Discover(ctx, token)
	if err != nil {
		return Connection{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Connection{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = s.persistDiscovery(ctx, tx, id, clientID, workspaceID, discovery); err != nil {
		return Connection{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE meta_connections SET last_validated_at=now(),last_error_code=NULL,last_error_message=NULL,version=version+1,updated_at=now() WHERE id=$1`, id); err != nil {
		return Connection{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Connection{}, err
	}
	return s.Get(ctx, clientID, workspaceID)
}

func (s *Service) Disconnect(ctx context.Context, clientID, workspaceID, actorID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	if err = tx.QueryRow(ctx, `UPDATE meta_connections SET status='DISCONNECTED',disconnected_at=now(),token_ciphertext=''::bytea,token_nonce=''::bytea,updated_by=$3,version=version+1,updated_at=now() WHERE client_id=$1 AND workspace_id=$2 AND disconnected_at IS NULL RETURNING id`, clientID, workspaceID, actorID).Scan(&id); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE social_accounts SET status='DISCONNECTED',disconnected_at=now(),token_ciphertext=''::bytea,token_nonce=''::bytea,version=version+1,updated_at=now() WHERE connection_id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) connectionToken(ctx context.Context, clientID, workspaceID uuid.UUID) (uuid.UUID, string, error) {
	var id uuid.UUID
	var ciphertext, nonce []byte
	err := s.pool.QueryRow(ctx, `SELECT id,token_ciphertext,token_nonce FROM meta_connections WHERE client_id=$1 AND workspace_id=$2 AND disconnected_at IS NULL AND status IN ('CONNECTED','EXPIRING') AND (token_expires_at IS NULL OR token_expires_at>now()) AND (data_access_expires_at IS NULL OR data_access_expires_at>now())`, clientID, workspaceID).Scan(&id, &ciphertext, &nonce)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", ErrNotFound
	} else if err != nil {
		return uuid.Nil, "", err
	}
	plaintext, err := s.cipher.Decrypt(ciphertext, nonce, "meta-connection:"+id.String())
	if err != nil {
		return uuid.Nil, "", ErrInvalid
	}
	return id, string(plaintext), nil
}

func (s *Service) persistDiscovery(ctx context.Context, tx pgx.Tx, connectionID, clientID, workspaceID uuid.UUID, d meta.Discovery) error {
	for _, page := range d.Pages {
		if err := s.upsertAccount(ctx, tx, connectionID, clientID, workspaceID, "FACEBOOK", page.ID, page.ID, nil, page.Name, nil, "", page.Tasks, page.AccessToken); err != nil {
			return err
		}
		if page.Instagram != nil {
			ig := page.Instagram
			if err := s.upsertAccount(ctx, tx, connectionID, clientID, workspaceID, "INSTAGRAM", ig.ID, page.ID, &ig.ID, ig.Name, &ig.Username, ig.PictureURL, page.Tasks, page.AccessToken); err != nil {
				return err
			}
		}
	}
	for _, item := range d.Businesses {
		_, err := tx.Exec(ctx, `INSERT INTO meta_businesses(connection_id,client_id,workspace_id,provider_business_id,name,verification_status)VALUES($1,$2,$3,$4,$5,$6)ON CONFLICT(workspace_id,provider_business_id)DO UPDATE SET connection_id=EXCLUDED.connection_id,name=EXCLUDED.name,verification_status=EXCLUDED.verification_status,updated_at=now()`, connectionID, clientID, workspaceID, item.ID, item.Name, item.VerificationStatus)
		if err != nil {
			return err
		}
	}
	accountIDs := map[string]uuid.UUID{}
	for _, item := range d.AdAccounts {
		id := uuid.New()
		err := tx.QueryRow(ctx, `INSERT INTO meta_ad_accounts(id,connection_id,client_id,workspace_id,provider_ad_account_id,name,currency,timezone_name,account_status,provider_spend_cap_minor,amount_spent_minor)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)ON CONFLICT(workspace_id,provider_ad_account_id)DO UPDATE SET connection_id=EXCLUDED.connection_id,name=EXCLUDED.name,currency=EXCLUDED.currency,timezone_name=EXCLUDED.timezone_name,account_status=EXCLUDED.account_status,provider_spend_cap_minor=EXCLUDED.provider_spend_cap_minor,amount_spent_minor=EXCLUDED.amount_spent_minor,last_synced_at=now(),updated_at=now() RETURNING id`, id, connectionID, clientID, workspaceID, item.ID, item.Name, item.Currency, item.TimezoneName, item.AccountStatus, item.SpendCapMinor, item.AmountSpentMinor).Scan(&id)
		if err != nil {
			return err
		}
		accountIDs[item.ID] = id
	}
	for _, item := range d.Pixels {
		accountID, ok := accountIDs[item.AdAccountID]
		if !ok {
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO meta_pixels(meta_ad_account_id,provider_pixel_id,name)VALUES($1,$2,$3)ON CONFLICT(meta_ad_account_id,provider_pixel_id)DO UPDATE SET name=EXCLUDED.name,updated_at=now()`, accountID, item.ID, item.Name)
		if err != nil {
			return err
		}
	}
	for _, item := range d.Audiences {
		accountID, ok := accountIDs[item.AdAccountID]
		if !ok {
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO meta_audiences(meta_ad_account_id,provider_audience_id,name,audience_type,subtype,approximate_count)VALUES($1,$2,$3,$4,$5,$6)ON CONFLICT(meta_ad_account_id,provider_audience_id)DO UPDATE SET name=EXCLUDED.name,audience_type=EXCLUDED.audience_type,subtype=EXCLUDED.subtype,approximate_count=EXCLUDED.approximate_count,updated_at=now()`, accountID, item.ID, item.Name, item.Type, item.Subtype, item.ApproximateCount)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) upsertAccount(ctx context.Context, tx pgx.Tx, connectionID, clientID, workspaceID uuid.UUID, platform, providerID, pageID string, instagramID *string, name string, username *string, picture string, tasks []string, token string) error {
	id := uuid.New()
	if scanErr := tx.QueryRow(ctx, `SELECT id FROM social_accounts WHERE workspace_id=$1 AND platform=$2::social_platform AND provider_account_id=$3`, workspaceID, platform, providerID).Scan(&id); scanErr != nil && !errors.Is(scanErr, pgx.ErrNoRows) {
		return scanErr
	}
	ciphertext, nonce, err := s.cipher.Encrypt([]byte(token), "social-account:"+id.String())
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO social_accounts(id,connection_id,client_id,workspace_id,platform,provider_account_id,facebook_page_id,instagram_business_id,name,username,picture_url,tasks,token_ciphertext,token_nonce,status,last_discovered_at)VALUES($1,$2,$3,$4,$5::social_platform,$6,$7,$8,$9,$10,$11,$12,$13,$14,'CONNECTED',now())ON CONFLICT(workspace_id,platform,provider_account_id)DO UPDATE SET connection_id=EXCLUDED.connection_id,facebook_page_id=EXCLUDED.facebook_page_id,instagram_business_id=EXCLUDED.instagram_business_id,name=EXCLUDED.name,username=EXCLUDED.username,picture_url=EXCLUDED.picture_url,tasks=EXCLUDED.tasks,token_ciphertext=EXCLUDED.token_ciphertext,token_nonce=EXCLUDED.token_nonce,status='CONNECTED',disconnected_at=NULL,last_discovered_at=now(),version=social_accounts.version+1,updated_at=now()`, id, connectionID, clientID, workspaceID, platform, providerID, pageID, instagramID, name, username, picture, tasks, ciphertext, nonce)
	return err
}

func (s *Service) Get(ctx context.Context, clientID, workspaceID uuid.UUID) (Connection, error) {
	var c Connection
	err := s.pool.QueryRow(ctx, `SELECT id,meta_user_id,display_name,CASE WHEN (token_expires_at IS NOT NULL AND token_expires_at<=now()) OR (data_access_expires_at IS NOT NULL AND data_access_expires_at<=now()) THEN 'EXPIRED' WHEN (token_expires_at IS NOT NULL AND token_expires_at<=now()+interval '14 days') OR (data_access_expires_at IS NOT NULL AND data_access_expires_at<=now()+interval '14 days') THEN 'EXPIRING' ELSE status::text END,api_version,scopes,token_issued_at,token_expires_at,data_access_expires_at,last_validated_at,version FROM meta_connections WHERE client_id=$1 AND workspace_id=$2 AND disconnected_at IS NULL`, clientID, workspaceID).Scan(&c.ID, &c.MetaUserID, &c.DisplayName, &c.Status, &c.APIVersion, &c.Scopes, &c.TokenIssuedAt, &c.TokenExpiresAt, &c.DataAccessExpiresAt, &c.LastValidatedAt, &c.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	} else if err != nil {
		return c, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id,platform::text,provider_account_id,facebook_page_id,instagram_business_id,name,username,tasks,status::text FROM social_accounts WHERE connection_id=$1 AND disconnected_at IS NULL ORDER BY platform,name`, c.ID)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	for rows.Next() {
		var a SocialAccount
		if err = rows.Scan(&a.ID, &a.Platform, &a.ProviderAccountID, &a.FacebookPageID, &a.InstagramBusinessID, &a.Name, &a.Username, &a.Tasks, &a.Status); err != nil {
			return c, err
		}
		c.Accounts = append(c.Accounts, a)
	}
	return c, s.loadBusinessAssets(ctx, &c, clientID, workspaceID)
}
func (s *Service) loadBusinessAssets(ctx context.Context, c *Connection, clientID, workspaceID uuid.UUID) error {
	rows, err := s.pool.Query(ctx, `SELECT id,provider_business_id,name FROM meta_businesses WHERE connection_id=$1 ORDER BY name`, c.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var x Business
		if err = rows.Scan(&x.ID, &x.ProviderBusinessID, &x.Name); err != nil {
			rows.Close()
			return err
		}
		c.Businesses = append(c.Businesses, x)
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `SELECT id,provider_ad_account_id,name,currency,timezone_name,account_status FROM meta_ad_accounts WHERE client_id=$1 AND workspace_id=$2 ORDER BY name`, clientID, workspaceID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var x AdAccount
		if err = rows.Scan(&x.ID, &x.ProviderAdAccountID, &x.Name, &x.Currency, &x.TimezoneName, &x.AccountStatus); err != nil {
			rows.Close()
			return err
		}
		c.AdAccounts = append(c.AdAccounts, x)
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `SELECT p.id,p.meta_ad_account_id,p.provider_pixel_id,p.name FROM meta_pixels p JOIN meta_ad_accounts a ON a.id=p.meta_ad_account_id WHERE a.client_id=$1 AND a.workspace_id=$2 ORDER BY p.name`, clientID, workspaceID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var x Pixel
		if err = rows.Scan(&x.ID, &x.MetaAdAccountID, &x.ProviderPixelID, &x.Name); err != nil {
			rows.Close()
			return err
		}
		c.Pixels = append(c.Pixels, x)
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `SELECT a.id,a.meta_ad_account_id,a.provider_audience_id,a.name,a.audience_type,a.subtype,a.approximate_count FROM meta_audiences a JOIN meta_ad_accounts m ON m.id=a.meta_ad_account_id WHERE m.client_id=$1 AND m.workspace_id=$2 ORDER BY a.name`, clientID, workspaceID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var x Audience
		if err = rows.Scan(&x.ID, &x.MetaAdAccountID, &x.ProviderAudienceID, &x.Name, &x.AudienceType, &x.Subtype, &x.ApproximateCount); err != nil {
			rows.Close()
			return err
		}
		c.Audiences = append(c.Audiences, x)
	}
	rows.Close()
	return nil
}
