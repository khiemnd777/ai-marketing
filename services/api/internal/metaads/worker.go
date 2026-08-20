package metaads

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/jobs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/meta"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/cryptox"
)

type ActionWorker struct {
	river.WorkerDefaults[jobs.MetaAdActionArgs]
	Pool     *pgxpool.Pool
	Cipher   *cryptox.Cipher
	Provider meta.Provider
	Enqueuer *jobs.Enqueuer
}

func (w *ActionWorker) Work(ctx context.Context, job *river.Job[jobs.MetaAdActionArgs]) error {
	var actionID, campaignID, connectionID uuid.UUID
	var action, providerAccountID, name, objective, buyingType string
	var providerCampaignID *string
	var requested, daily, lifetime *int64
	var cipherText, nonce []byte
	err := w.Pool.QueryRow(ctx, `UPDATE meta_ad_actions x SET status='PROCESSING',version=x.version+1 FROM ad_campaigns a JOIN meta_ad_accounts aa ON aa.id=a.meta_ad_account_id JOIN meta_connections c ON c.id=aa.connection_id WHERE x.id=$1 AND x.ad_campaign_id=a.id AND x.status='QUEUED' AND c.status IN ('CONNECTED','EXPIRING') AND (c.token_expires_at IS NULL OR c.token_expires_at>now()) AND (c.data_access_expires_at IS NULL OR c.data_access_expires_at>now()) AND (x.action IN ('CREATE_PAUSED','PAUSE','ARCHIVE') OR EXISTS(SELECT 1 FROM approvals ap WHERE ap.entity_id=x.id AND ap.entity_hash=x.action_hash AND ap.status='APPROVED' AND ap.invalidated_at IS NULL)) RETURNING x.id,a.id,c.id,x.action::text,aa.provider_ad_account_id,a.name,a.objective,a.buying_type,a.provider_campaign_id,x.requested_budget_minor,a.daily_budget_minor,a.lifetime_budget_minor,c.token_ciphertext,c.token_nonce`, job.Args.ActionID).Scan(&actionID, &campaignID, &connectionID, &action, &providerAccountID, &name, &objective, &buyingType, &providerCampaignID, &requested, &daily, &lifetime, &cipherText, &nonce)
	if errors.Is(err, pgx.ErrNoRows) {
		return river.JobCancel(ErrPrerequisite)
	} else if err != nil {
		return err
	}
	if w.Cipher == nil || w.Provider == nil {
		return errors.New("Meta Ads worker is not configured")
	}
	token, err := w.Cipher.Decrypt(cipherText, nonce, "meta-connection:"+connectionID.String())
	if err != nil {
		return w.fail(ctx, actionID, campaignID, action, "token_decrypt", "Stored Meta token is unavailable")
	}
	safe := map[string]any{"action": action}
	newStatus := ""
	switch action {
	case "CREATE_PAUSED":
		result, e := w.Provider.CreatePausedCampaign(ctx, meta.CampaignRequest{AdAccountID: providerAccountID, Name: name, Objective: objective, BuyingType: buyingType, AccessToken: string(token)})
		if e != nil {
			return w.providerFail(ctx, actionID, campaignID, action, e)
		}
		providerCampaignID = &result.ProviderCampaignID
		newStatus = "PAUSED"
		safe["requestId"] = result.RequestID
	case "ACTIVATE", "RESUME":
		if providerCampaignID == nil {
			return w.fail(ctx, actionID, campaignID, action, "missing_provider_campaign", "Meta campaign is missing")
		}
		_, e := w.Provider.SetCampaignStatus(ctx, *providerCampaignID, "ACTIVE", string(token))
		if e != nil {
			return w.providerFail(ctx, actionID, campaignID, action, e)
		}
		newStatus = "ACTIVE"
	case "PAUSE":
		if providerCampaignID == nil {
			return w.fail(ctx, actionID, campaignID, action, "missing_provider_campaign", "Meta campaign is missing")
		}
		_, e := w.Provider.SetCampaignStatus(ctx, *providerCampaignID, "PAUSED", string(token))
		if e != nil {
			return w.providerFail(ctx, actionID, campaignID, action, e)
		}
		newStatus = "PAUSED"
	case "ARCHIVE":
		if providerCampaignID == nil {
			return w.fail(ctx, actionID, campaignID, action, "missing_provider_campaign", "Meta campaign is missing")
		}
		_, e := w.Provider.SetCampaignStatus(ctx, *providerCampaignID, "ARCHIVED", string(token))
		if e != nil {
			return w.providerFail(ctx, actionID, campaignID, action, e)
		}
		newStatus = "ARCHIVED"
	case "BUDGET_CHANGE":
		if providerCampaignID == nil || requested == nil {
			return w.fail(ctx, actionID, campaignID, action, "invalid_budget_action", "Budget action is incomplete")
		}
		_, e := w.Provider.SetCampaignBudget(ctx, *providerCampaignID, *requested, lifetime != nil, string(token))
		if e != nil {
			return w.providerFail(ctx, actionID, campaignID, action, e)
		}
	default:
		return river.JobCancel(ErrInvalid)
	}
	raw, _ := json.Marshal(safe)
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if action == "CREATE_PAUSED" {
		_, err = tx.Exec(ctx, `UPDATE ad_campaigns SET provider_campaign_id=$2,status='PAUSED',last_error_code=NULL,last_error_message=NULL,version=version+1,updated_at=now() WHERE id=$1`, campaignID, *providerCampaignID)
	} else if action == "BUDGET_CHANGE" {
		if lifetime != nil {
			_, err = tx.Exec(ctx, `UPDATE ad_campaigns SET lifetime_budget_minor=$2,last_error_code=NULL,last_error_message=NULL,version=version+1,updated_at=now() WHERE id=$1`, campaignID, *requested)
		} else {
			_, err = tx.Exec(ctx, `UPDATE ad_campaigns SET daily_budget_minor=$2,last_error_code=NULL,last_error_message=NULL,version=version+1,updated_at=now() WHERE id=$1`, campaignID, *requested)
		}
	} else {
		_, err = tx.Exec(ctx, `UPDATE ad_campaigns SET status=$2::meta_ad_campaign_status,last_error_code=NULL,last_error_message=NULL,version=version+1,updated_at=now() WHERE id=$1`, campaignID, newStatus)
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE meta_ad_actions SET status='SUCCEEDED',safe_response=$2,completed_at=now(),version=version+1 WHERE id=$1`, actionID, raw)
	if err != nil {
		return err
	}
	if w.Enqueuer != nil && action == "CREATE_PAUSED" {
		_, err = w.Enqueuer.EnqueueMetaMetricsSync(ctx, tx, campaignID)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (w *ActionWorker) providerFail(ctx context.Context, actionID, campaignID uuid.UUID, action string, cause error) error {
	code, message := "meta_action_failed", "Meta could not apply the Ads action"
	var pe *meta.ProviderError
	if errors.As(cause, &pe) {
		code, message = pe.Code, pe.SafeMessage
	}
	return w.fail(ctx, actionID, campaignID, action, code, message)
}
func (w *ActionWorker) fail(ctx context.Context, actionID, campaignID uuid.UUID, action, code, message string) error {
	_, _ = w.Pool.Exec(ctx, `UPDATE meta_ad_actions SET status='FAILED',error_code=$2,error_message=$3,completed_at=now(),version=version+1 WHERE id=$1`, actionID, code, message)
	if action == "CREATE_PAUSED" {
		_, _ = w.Pool.Exec(ctx, `UPDATE ad_campaigns SET status='FAILED',last_error_code=$2,last_error_message=$3,version=version+1,updated_at=now() WHERE id=$1`, campaignID, code, message)
	}
	return errors.New(message)
}

type MetricsWorker struct {
	river.WorkerDefaults[jobs.MetaMetricsSyncArgs]
	Pool     *pgxpool.Pool
	Cipher   *cryptox.Cipher
	Provider meta.Provider
}

func (w *MetricsWorker) Work(ctx context.Context, job *river.Job[jobs.MetaMetricsSyncArgs]) error {
	var providerID string
	var connectionID uuid.UUID
	var cipherText, nonce []byte
	err := w.Pool.QueryRow(ctx, `SELECT a.provider_campaign_id,c.id,c.token_ciphertext,c.token_nonce FROM ad_campaigns a JOIN meta_ad_accounts aa ON aa.id=a.meta_ad_account_id JOIN meta_connections c ON c.id=aa.connection_id WHERE a.id=$1 AND a.provider_campaign_id IS NOT NULL AND c.status IN ('CONNECTED','EXPIRING') AND (c.token_expires_at IS NULL OR c.token_expires_at>now()) AND (c.data_access_expires_at IS NULL OR c.data_access_expires_at>now())`, job.Args.AdCampaignID).Scan(&providerID, &connectionID, &cipherText, &nonce)
	if errors.Is(err, pgx.ErrNoRows) {
		return river.JobCancel(ErrPrerequisite)
	} else if err != nil {
		return err
	}
	token, err := w.Cipher.Decrypt(cipherText, nonce, "meta-connection:"+connectionID.String())
	if err != nil {
		return err
	}
	days, err := w.Provider.CampaignInsights(ctx, providerID, string(token))
	if err != nil {
		return err
	}
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, day := range days {
		raw, _ := json.Marshal(day.SafeResponse)
		_, err = tx.Exec(ctx, `INSERT INTO ad_campaign_metrics_daily(ad_campaign_id,metric_date,spend_minor,impressions,reach,clicks,conversions,leads,purchases,revenue_minor,frequency,provider_response,synced_at)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,now())ON CONFLICT(ad_campaign_id,metric_date)DO UPDATE SET spend_minor=EXCLUDED.spend_minor,impressions=EXCLUDED.impressions,reach=EXCLUDED.reach,clicks=EXCLUDED.clicks,conversions=EXCLUDED.conversions,leads=EXCLUDED.leads,purchases=EXCLUDED.purchases,revenue_minor=EXCLUDED.revenue_minor,frequency=EXCLUDED.frequency,provider_response=EXCLUDED.provider_response,synced_at=now()`, job.Args.AdCampaignID, day.Date, day.SpendMinor, day.Impressions, day.Reach, day.Clicks, day.Conversions, day.Leads, day.Purchases, day.RevenueMinor, day.Frequency, raw)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
