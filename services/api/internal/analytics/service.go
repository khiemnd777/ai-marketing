package analytics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalid  = errors.New("invalid analytics operation")
	ErrNotFound = errors.New("analytics resource not found")
	ErrConflict = errors.New("analytics resource conflict")
)

type Filter struct {
	From, To   time.Time
	CampaignID *uuid.UUID
}

type CostBreakdown struct {
	Category string  `json:"category"`
	USD      float64 `json:"usd"`
}
type VideoMetrics struct {
	VideoCount               int64    `json:"videoCount"`
	SceneCount               int64    `json:"sceneCount"`
	RejectedScenes           int64    `json:"rejectedScenes"`
	RegeneratedScenes        int64    `json:"regeneratedScenes"`
	AttemptFactor            float64  `json:"attemptFactor"`
	AverageGenerationSeconds float64  `json:"averageGenerationSeconds"`
	AverageReviewSeconds     float64  `json:"averageReviewSeconds"`
	TemplateSuccessRate      *float64 `json:"templateSuccessRate"`
	LLMCostUSD               float64  `json:"llmCostUsd"`
	SeedanceCostUSD          float64  `json:"seedanceCostUsd"`
	TranscriptionCostUSD     float64  `json:"transcriptionCostUsd"`
	RenderCostUSD            float64  `json:"renderCostUsd"`
	StorageCostUSD           float64  `json:"storageCostUsd"`
	FullCostUSD              float64  `json:"fullCostUsd"`
}
type SocialMetrics struct {
	Views          int64   `json:"views"`
	Reach          int64   `json:"reach"`
	Impressions    int64   `json:"impressions"`
	WatchTimeMS    int64   `json:"watchTimeMs"`
	Likes          int64   `json:"likes"`
	Comments       int64   `json:"comments"`
	Shares         int64   `json:"shares"`
	Saves          int64   `json:"saves"`
	LinkClicks     int64   `json:"linkClicks"`
	CTR            float64 `json:"ctr"`
	CompletionRate float64 `json:"completionRate"`
}
type AdMetrics struct {
	SpendMinor     int64   `json:"spendMinor"`
	Impressions    int64   `json:"impressions"`
	Reach          int64   `json:"reach"`
	Clicks         int64   `json:"clicks"`
	RevenueMinor   int64   `json:"revenueMinor"`
	Conversions    float64 `json:"conversions"`
	Leads          float64 `json:"leads"`
	Purchases      float64 `json:"purchases"`
	CPM            float64 `json:"cpm"`
	CPC            float64 `json:"cpc"`
	CTR            float64 `json:"ctr"`
	CPA            float64 `json:"cpa"`
	ROAS           float64 `json:"roas"`
	ROI            float64 `json:"roi"`
	Frequency      float64 `json:"frequency"`
	ConversionRate float64 `json:"conversionRate"`
}
type DailyPoint struct {
	Date              string  `json:"date"`
	CostUSD           float64 `json:"costUsd"`
	SocialViews       int64   `json:"socialViews"`
	SocialImpressions int64   `json:"socialImpressions"`
	SocialClicks      int64   `json:"socialClicks"`
	AdSpendMinor      int64   `json:"adSpendMinor"`
	AdImpressions     int64   `json:"adImpressions"`
	AdClicks          int64   `json:"adClicks"`
	AdConversions     float64 `json:"adConversions"`
	AdRevenueMinor    int64   `json:"adRevenueMinor"`
}
type CreativeComparison struct {
	VideoFormat     string  `json:"videoFormat"`
	CTA             string  `json:"cta"`
	DurationSeconds int32   `json:"durationSeconds"`
	Campaigns       int64   `json:"campaigns"`
	Impressions     int64   `json:"impressions"`
	Clicks          int64   `json:"clicks"`
	SpendMinor      int64   `json:"spendMinor"`
	RevenueMinor    int64   `json:"revenueMinor"`
	CTR             float64 `json:"ctr"`
	ROAS            float64 `json:"roas"`
}
type Summary struct {
	From                time.Time            `json:"from"`
	To                  time.Time            `json:"to"`
	TotalCostUSD        float64              `json:"totalCostUsd"`
	Costs               []CostBreakdown      `json:"costs"`
	Video               VideoMetrics         `json:"video"`
	Social              SocialMetrics        `json:"social"`
	Ads                 AdMetrics            `json:"ads"`
	Daily               []DailyPoint         `json:"daily"`
	CreativeComparisons []CreativeComparison `json:"creativeComparisons"`
}

type Recommendation struct {
	ID            uuid.UUID      `json:"id"`
	CampaignID    *uuid.UUID     `json:"campaignId"`
	AdCampaignID  *uuid.UUID     `json:"adCampaignId"`
	Type          string         `json:"type"`
	Model         string         `json:"model"`
	Output        string         `json:"output"`
	Rationale     string         `json:"rationale"`
	Status        string         `json:"status"`
	ReviewNotes   string         `json:"reviewNotes"`
	ActionTaken   string         `json:"actionTaken"`
	InputSnapshot map[string]any `json:"inputSnapshot"`
	ReviewerID    *uuid.UUID     `json:"reviewerId"`
	ReviewedAt    *time.Time     `json:"reviewedAt"`
	Version       int64          `json:"version"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}
type ReviewInput struct {
	Action  string `json:"action"`
	Version int64  `json:"version"`
	Notes   string `json:"notes"`
}

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

func (s *Service) Summary(ctx context.Context, clientID, workspaceID uuid.UUID, filter Filter) (Summary, error) {
	if filter.From.IsZero() || filter.To.IsZero() || filter.To.Before(filter.From) || filter.To.Sub(filter.From) > 366*24*time.Hour {
		return Summary{}, ErrInvalid
	}
	result := Summary{From: filter.From, To: filter.To, Costs: []CostBreakdown{}, Daily: []DailyPoint{}, CreativeComparisons: []CreativeComparison{}}
	rows, err := s.pool.Query(ctx, `SELECT category,sum(normalized_amount_usd)::float8 FROM cost_records WHERE client_id=$1 AND workspace_id=$2 AND occurred_at>=$3 AND occurred_at<$4 AND ($5::uuid IS NULL OR campaign_id=$5) GROUP BY category ORDER BY category`, clientID, workspaceID, filter.From, filter.To, filter.CampaignID)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var item CostBreakdown
		if err = rows.Scan(&item.Category, &item.USD); err != nil {
			rows.Close()
			return result, err
		}
		result.Costs = append(result.Costs, item)
		result.TotalCostUSD += item.USD
		switch item.Category {
		case "LLM":
			result.Video.LLMCostUSD = item.USD
		case "SEEDANCE":
			result.Video.SeedanceCostUSD = item.USD
		case "TRANSCRIPTION":
			result.Video.TranscriptionCostUSD = item.USD
		case "RENDER":
			result.Video.RenderCostUSD = item.USD
		case "STORAGE":
			result.Video.StorageCostUSD = item.USD
		}
	}
	rows.Close()
	err = s.pool.QueryRow(ctx, `SELECT
		count(DISTINCT r.id) FILTER(WHERE r.status IN ('REVIEW_REQUIRED','APPROVED','REJECTED')),
		count(DISTINCT s.id),
		count(g.id) FILTER(WHERE g.status='REJECTED'),
		COALESCE(sum(GREATEST(g.attempt_number-1,0)),0),
		COALESCE(count(g.id)::float8/NULLIF(count(DISTINCT s.id),0),0),
		COALESCE(avg(EXTRACT(EPOCH FROM(g.provider_completed_at-g.submitted_at))) FILTER(WHERE g.provider_completed_at IS NOT NULL),0),
		COALESCE(avg(EXTRACT(EPOCH FROM(g.reviewed_at-g.provider_completed_at))) FILTER(WHERE g.reviewed_at IS NOT NULL AND g.provider_completed_at IS NOT NULL),0)
	FROM campaigns c LEFT JOIN scenes s ON s.campaign_id=c.id LEFT JOIN scene_generation_tasks g ON g.scene_id=s.id LEFT JOIN render_jobs r ON r.campaign_id=c.id
	WHERE c.client_id=$1 AND c.workspace_id=$2 AND c.created_at<$4 AND c.updated_at>=$3 AND ($5::uuid IS NULL OR c.id=$5)`, clientID, workspaceID, filter.From, filter.To, filter.CampaignID).Scan(&result.Video.VideoCount, &result.Video.SceneCount, &result.Video.RejectedScenes, &result.Video.RegeneratedScenes, &result.Video.AttemptFactor, &result.Video.AverageGenerationSeconds, &result.Video.AverageReviewSeconds)
	if err != nil {
		return result, err
	}
	result.Video.FullCostUSD = result.TotalCostUSD
	var completionDenominator float64
	err = s.pool.QueryRow(ctx, `SELECT COALESCE(sum(m.views),0),COALESCE(sum(m.reach),0),COALESCE(sum(m.impressions),0),COALESCE(sum(m.watch_time_ms),0),COALESCE(sum(m.likes),0),COALESCE(sum(m.comments),0),COALESCE(sum(m.shares),0),COALESCE(sum(m.saves),0),COALESCE(sum(m.link_clicks),0),COALESCE(sum(m.views*v.duration_ms),0)::float8 FROM social_post_metrics_daily m JOIN social_posts p ON p.id=m.social_post_id JOIN media_assets a ON a.id=p.media_asset_id JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version WHERE p.client_id=$1 AND p.workspace_id=$2 AND m.metric_date>=$3::date AND m.metric_date<$4::date AND ($5::uuid IS NULL OR p.campaign_id=$5)`, clientID, workspaceID, filter.From, filter.To, filter.CampaignID).Scan(&result.Social.Views, &result.Social.Reach, &result.Social.Impressions, &result.Social.WatchTimeMS, &result.Social.Likes, &result.Social.Comments, &result.Social.Shares, &result.Social.Saves, &result.Social.LinkClicks, &completionDenominator)
	if err != nil {
		return result, err
	}
	result.Social.CTR = percentage(float64(result.Social.LinkClicks), float64(result.Social.Impressions))
	result.Social.CompletionRate = percentage(float64(result.Social.WatchTimeMS), completionDenominator)
	err = s.pool.QueryRow(ctx, `SELECT COALESCE(sum(m.spend_minor),0),COALESCE(sum(m.impressions),0),COALESCE(sum(m.reach),0),COALESCE(sum(m.clicks),0),COALESCE(sum(m.conversions),0)::float8,COALESCE(sum(m.leads),0)::float8,COALESCE(sum(m.purchases),0)::float8,COALESCE(sum(m.revenue_minor),0) FROM ad_campaign_metrics_daily m JOIN ad_campaigns a ON a.id=m.ad_campaign_id WHERE a.client_id=$1 AND a.workspace_id=$2 AND m.metric_date>=$3::date AND m.metric_date<$4::date AND ($5::uuid IS NULL OR a.campaign_id=$5)`, clientID, workspaceID, filter.From, filter.To, filter.CampaignID).Scan(&result.Ads.SpendMinor, &result.Ads.Impressions, &result.Ads.Reach, &result.Ads.Clicks, &result.Ads.Conversions, &result.Ads.Leads, &result.Ads.Purchases, &result.Ads.RevenueMinor)
	if err != nil {
		return result, err
	}
	result.Ads.CPM = ratio(float64(result.Ads.SpendMinor)*1000, float64(result.Ads.Impressions))
	result.Ads.CPC = ratio(float64(result.Ads.SpendMinor), float64(result.Ads.Clicks))
	result.Ads.CTR = percentage(float64(result.Ads.Clicks), float64(result.Ads.Impressions))
	result.Ads.CPA = ratio(float64(result.Ads.SpendMinor), result.Ads.Conversions)
	result.Ads.ROAS = ratio(float64(result.Ads.RevenueMinor), float64(result.Ads.SpendMinor))
	result.Ads.ROI = ratio(float64(result.Ads.RevenueMinor-result.Ads.SpendMinor), float64(result.Ads.SpendMinor))
	result.Ads.Frequency = ratio(float64(result.Ads.Impressions), float64(result.Ads.Reach))
	result.Ads.ConversionRate = percentage(result.Ads.Conversions, float64(result.Ads.Clicks))
	rows, err = s.pool.Query(ctx, `WITH dates AS (SELECT generate_series($3::date,($4::date-1),interval '1 day')::date d), c AS (SELECT occurred_at::date d,sum(normalized_amount_usd)::float8 cost FROM cost_records WHERE client_id=$1 AND workspace_id=$2 AND occurred_at>=$3 AND occurred_at<$4 AND ($5::uuid IS NULL OR campaign_id=$5) GROUP BY 1), sm AS (SELECT m.metric_date d,sum(m.views)::bigint views,sum(m.impressions)::bigint impressions,sum(m.link_clicks)::bigint clicks FROM social_post_metrics_daily m JOIN social_posts p ON p.id=m.social_post_id WHERE p.client_id=$1 AND p.workspace_id=$2 AND m.metric_date>=$3::date AND m.metric_date<$4::date AND ($5::uuid IS NULL OR p.campaign_id=$5) GROUP BY 1), am AS (SELECT m.metric_date d,sum(m.spend_minor)::bigint spend,sum(m.impressions)::bigint impressions,sum(m.clicks)::bigint clicks,sum(m.conversions)::float8 conversions,sum(m.revenue_minor)::bigint revenue FROM ad_campaign_metrics_daily m JOIN ad_campaigns a ON a.id=m.ad_campaign_id WHERE a.client_id=$1 AND a.workspace_id=$2 AND m.metric_date>=$3::date AND m.metric_date<$4::date AND ($5::uuid IS NULL OR a.campaign_id=$5) GROUP BY 1) SELECT dates.d::text,COALESCE(c.cost,0),COALESCE(sm.views,0),COALESCE(sm.impressions,0),COALESCE(sm.clicks,0),COALESCE(am.spend,0),COALESCE(am.impressions,0),COALESCE(am.clicks,0),COALESCE(am.conversions,0),COALESCE(am.revenue,0) FROM dates LEFT JOIN c USING(d) LEFT JOIN sm USING(d) LEFT JOIN am USING(d) ORDER BY dates.d`, clientID, workspaceID, filter.From, filter.To, filter.CampaignID)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var d DailyPoint
		if err = rows.Scan(&d.Date, &d.CostUSD, &d.SocialViews, &d.SocialImpressions, &d.SocialClicks, &d.AdSpendMinor, &d.AdImpressions, &d.AdClicks, &d.AdConversions, &d.AdRevenueMinor); err != nil {
			rows.Close()
			return result, err
		}
		result.Daily = append(result.Daily, d)
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `SELECT cv.video_format,cv.duration_seconds,cv.cta,count(DISTINCT c.id),COALESCE(sum(m.impressions),0),COALESCE(sum(m.clicks),0),COALESCE(sum(m.spend_minor),0),COALESCE(sum(m.revenue_minor),0) FROM campaigns c JOIN campaign_versions cv ON cv.campaign_id=c.id AND cv.version=c.current_version LEFT JOIN ad_campaigns a ON a.campaign_id=c.id LEFT JOIN ad_campaign_metrics_daily m ON m.ad_campaign_id=a.id AND m.metric_date>=$3::date AND m.metric_date<$4::date WHERE c.client_id=$1 AND c.workspace_id=$2 AND ($5::uuid IS NULL OR c.id=$5) GROUP BY cv.video_format,cv.duration_seconds,cv.cta ORDER BY sum(m.revenue_minor) DESC NULLS LAST`, clientID, workspaceID, filter.From, filter.To, filter.CampaignID)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var x CreativeComparison
		if err = rows.Scan(&x.VideoFormat, &x.DurationSeconds, &x.CTA, &x.Campaigns, &x.Impressions, &x.Clicks, &x.SpendMinor, &x.RevenueMinor); err != nil {
			rows.Close()
			return result, err
		}
		x.CTR = percentage(float64(x.Clicks), float64(x.Impressions))
		x.ROAS = ratio(float64(x.RevenueMinor), float64(x.SpendMinor))
		result.CreativeComparisons = append(result.CreativeComparisons, x)
	}
	rows.Close()
	return result, rows.Err()
}

func (s *Service) GenerateRecommendations(ctx context.Context, clientID, workspaceID, actorID uuid.UUID, campaignID *uuid.UUID) ([]Recommendation, error) {
	rows, err := s.pool.Query(ctx, `SELECT a.id,a.campaign_id,a.status::text,COALESCE(sum(m.spend_minor),0),COALESCE(sum(m.impressions),0),COALESCE(sum(m.reach),0),COALESCE(sum(m.clicks),0),COALESCE(sum(m.conversions),0)::float8,COALESCE(sum(m.revenue_minor),0) FROM ad_campaigns a LEFT JOIN ad_campaign_metrics_daily m ON m.ad_campaign_id=a.id WHERE a.client_id=$1 AND a.workspace_id=$2 AND ($3::uuid IS NULL OR a.campaign_id=$3) GROUP BY a.id,a.campaign_id,a.status`, clientID, workspaceID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var adID, campID uuid.UUID
		var status string
		var spend, impressions, reach, clicks, revenue int64
		var conversions float64
		if err = rows.Scan(&adID, &campID, &status, &spend, &impressions, &reach, &clicks, &conversions, &revenue); err != nil {
			return nil, err
		}
		ctr := percentage(float64(clicks), float64(impressions))
		frequency := ratio(float64(impressions), float64(reach))
		kind, output, rationale := "", "", ""
		switch {
		case impressions >= 1000 && ctr < 1:
			kind, output, rationale = "INVESTIGATE_LOW_CTR", "Kiểm tra hook, thumbnail và primary text trước khi tăng chi.", fmt.Sprintf("CTR %.2f%% dưới ngưỡng quan sát 1%% sau %d impressions.", ctr, impressions)
		case frequency > 3.5:
			kind, output, rationale = "INVESTIGATE_HIGH_FREQUENCY", "Kiểm tra creative fatigue và mở rộng audience nếu phù hợp.", fmt.Sprintf("Frequency %.2f vượt 3.5.", frequency)
		case impressions >= 1000 && ctr >= 2 && conversions > 0:
			kind, output, rationale = "SCALE_WINNER", "Creative có tín hiệu tốt; operator có thể đánh giá một mức tăng trong guardrail.", fmt.Sprintf("CTR %.2f%% với %.0f conversions; không có budget nào được tự động thay đổi.", ctr, conversions)
		}
		if kind == "" {
			continue
		}
		snapshot := map[string]any{"status": status, "spendMinor": spend, "impressions": impressions, "reach": reach, "clicks": clicks, "conversions": conversions, "revenueMinor": revenue, "ctr": ctr, "frequency": frequency}
		raw, _ := json.Marshal(snapshot)
		sum := sha256.Sum256(append([]byte(kind+"|"+adID.String()+"|"), raw...))
		_, err = s.pool.Exec(ctx, `INSERT INTO ad_recommendations(client_id,workspace_id,campaign_id,ad_campaign_id,recommendation_type,recommendation_hash,input_snapshot,model,output,rationale,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,'deterministic-analytics-v1',$8,$9,$10)ON CONFLICT(recommendation_hash)DO NOTHING`, clientID, workspaceID, campID, adID, kind, hex.EncodeToString(sum[:]), raw, output, rationale, actorID)
		if err != nil {
			return nil, err
		}
	}
	return s.ListRecommendations(ctx, clientID, workspaceID, campaignID)
}
func (s *Service) ListRecommendations(ctx context.Context, clientID, workspaceID uuid.UUID, campaignID *uuid.UUID) ([]Recommendation, error) {
	rows, err := s.pool.Query(ctx, recommendationSelect+` WHERE client_id=$1 AND workspace_id=$2 AND ($3::uuid IS NULL OR campaign_id=$3) ORDER BY created_at DESC`, clientID, workspaceID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Recommendation{}
	for rows.Next() {
		x, e := scanRecommendation(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (s *Service) ReviewRecommendation(ctx context.Context, clientID, workspaceID, id, actorID uuid.UUID, input ReviewInput) (Recommendation, error) {
	action := strings.ToUpper(strings.TrimSpace(input.Action))
	if (action != "APPROVE" && action != "REJECT") || input.Version < 1 || (action == "REJECT" && strings.TrimSpace(input.Notes) == "") {
		return Recommendation{}, ErrInvalid
	}
	status := "REJECTED"
	if action == "APPROVE" {
		status = "APPROVED"
	}
	tag, err := s.pool.Exec(ctx, `UPDATE ad_recommendations SET status=$2::recommendation_status,reviewer_id=$3,review_notes=$4,reviewed_at=now(),version=version+1,updated_at=now() WHERE id=$1 AND client_id=$5 AND workspace_id=$6 AND status='DRAFT' AND version=$7`, id, status, actorID, strings.TrimSpace(input.Notes), clientID, workspaceID, input.Version)
	if err != nil {
		return Recommendation{}, err
	}
	if tag.RowsAffected() != 1 {
		return Recommendation{}, ErrConflict
	}
	x, err := scanRecommendation(s.pool.QueryRow(ctx, recommendationSelect+` WHERE id=$1 AND client_id=$2 AND workspace_id=$3`, id, clientID, workspaceID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Recommendation{}, ErrNotFound
	}
	return x, err
}

const recommendationSelect = `SELECT id,campaign_id,ad_campaign_id,recommendation_type,input_snapshot,model,output,rationale,status::text,reviewer_id,review_notes,reviewed_at,action_taken,version,created_at,updated_at FROM ad_recommendations`

type scanner interface{ Scan(...any) error }

func scanRecommendation(row scanner) (Recommendation, error) {
	var x Recommendation
	var raw []byte
	err := row.Scan(&x.ID, &x.CampaignID, &x.AdCampaignID, &x.Type, &raw, &x.Model, &x.Output, &x.Rationale, &x.Status, &x.ReviewerID, &x.ReviewNotes, &x.ReviewedAt, &x.ActionTaken, &x.Version, &x.CreatedAt, &x.UpdatedAt)
	_ = json.Unmarshal(raw, &x.InputSnapshot)
	return x, err
}
func ratio(n, d float64) float64 {
	if d <= 0 {
		return 0
	}
	return n / d
}
func percentage(n, d float64) float64 { return ratio(n, d) * 100 }
