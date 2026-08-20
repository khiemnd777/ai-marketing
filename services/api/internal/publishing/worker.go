package publishing

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/jobs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/meta"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/cryptox"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/storage"
)

type mediaSigner interface {
	PresignGet(context.Context, string, time.Duration) (storage.PresignedRequest, error)
}
type Worker struct {
	river.WorkerDefaults[jobs.SocialPublishArgs]
	Pool     *pgxpool.Pool
	Store    mediaSigner
	Cipher   *cryptox.Cipher
	Provider meta.Provider
}

func (w *Worker) Work(ctx context.Context, job *river.Job[jobs.SocialPublishArgs]) error {
	var accountID, clientID, workspaceID uuid.UUID
	var platform, pageID string
	var instagramID *string
	var tokenCipher, nonce []byte
	var key, mime, caption, hash string
	err := w.Pool.QueryRow(ctx, `UPDATE social_posts p SET status='PUBLISHING',version=p.version+1,updated_at=now() FROM social_accounts a JOIN meta_connections c ON c.id=a.connection_id,media_assets m JOIN media_asset_versions v ON v.media_asset_id=m.id AND v.version=m.current_version WHERE p.id=$1 AND p.social_account_id=a.id AND p.media_asset_id=m.id AND p.status IN ('APPROVED','SCHEDULED') AND a.status IN ('CONNECTED','EXPIRING') AND c.status IN ('CONNECTED','EXPIRING') AND (c.token_expires_at IS NULL OR c.token_expires_at>now()) AND (c.data_access_expires_at IS NULL OR c.data_access_expires_at>now()) AND EXISTS(SELECT 1 FROM approvals ap WHERE ap.entity_type='SOCIAL_POST' AND ap.entity_id=p.id AND ap.entity_hash=p.content_hash AND ap.status='APPROVED' AND ap.invalidated_at IS NULL) RETURNING a.id,p.client_id,p.workspace_id,p.platform::text,a.facebook_page_id,a.instagram_business_id,a.token_ciphertext,a.token_nonce,v.storage_key,v.mime_type,p.caption,p.content_hash`, job.Args.SocialPostID).Scan(&accountID, &clientID, &workspaceID, &platform, &pageID, &instagramID, &tokenCipher, &nonce, &key, &mime, &caption, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return river.JobCancel(ErrPrerequisite)
	} else if err != nil {
		return err
	}
	if w.Store == nil || w.Cipher == nil || w.Provider == nil {
		return errors.New("publishing worker is not configured")
	}
	token, err := w.Cipher.Decrypt(tokenCipher, nonce, "social-account:"+accountID.String())
	if err != nil {
		return w.fail(ctx, job, &meta.ProviderError{Category: "AUTH", Code: "token_decrypt", SafeMessage: "Stored Meta token is unavailable"})
	}
	media, err := w.Store.PresignGet(ctx, key, 24*time.Hour)
	if err != nil {
		return w.fail(ctx, job, err)
	}
	result, err := w.Provider.Publish(ctx, meta.PublishRequest{Platform: platform, PageID: pageID, InstagramID: value(instagramID), MediaURL: media.URL, MediaType: mime, Caption: caption, AccessToken: string(token)})
	if err != nil {
		return w.fail(ctx, job, err)
	}
	safe, _ := json.Marshal(map[string]any{"requestId": result.RequestID, "publicUrlAvailable": result.PublicURL != ""})
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `UPDATE social_posts SET status='PUBLISHED',provider_post_id=$2,public_url=NULLIF($3,''),provider_request_id=NULLIF($4,''),published_at=now(),error_category=NULL,error_code=NULL,error_message=NULL,version=version+1,updated_at=now() WHERE id=$1 AND status='PUBLISHING'`, job.Args.SocialPostID, result.ProviderPostID, result.PublicURL, result.RequestID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE publish_jobs SET attempt_count=$2,safe_response=$3,updated_at=now() WHERE social_post_id=$1`, job.Args.SocialPostID, job.Attempt, safe)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (w *Worker) fail(ctx context.Context, job *river.Job[jobs.SocialPublishArgs], cause error) error {
	category, code, message, retryable := "INTERNAL", "publish_failed", "Publishing failed", false
	var providerErr *meta.ProviderError
	if errors.As(cause, &providerErr) {
		category, code, message, retryable = providerErr.Category, providerErr.Code, providerErr.SafeMessage, providerErr.Retryable
	}
	status := "FAILED"
	if retryable && job.Attempt < job.MaxAttempts {
		status = "APPROVED"
	}
	if !retryable {
		status = "PERMANENT_FAILURE"
	}
	_, _ = w.Pool.Exec(ctx, `UPDATE social_posts SET status=$2::social_post_status,error_category=$3,error_code=$4,error_message=$5,version=version+1,updated_at=now() WHERE id=$1`, job.Args.SocialPostID, status, category, code, message)
	_, _ = w.Pool.Exec(ctx, `UPDATE publish_jobs SET attempt_count=$2,last_error_retryable=$3,updated_at=now() WHERE social_post_id=$1`, job.Args.SocialPostID, job.Attempt, retryable)
	return cause
}
func value(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
