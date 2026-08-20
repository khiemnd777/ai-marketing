package meta

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

type LiveProvider struct {
	cfg    config.MetaConfig
	client *http.Client
}

func NewLiveProvider(cfg config.MetaConfig) (*LiveProvider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfiguration, err)
	}
	return &LiveProvider{cfg: cfg, client: &http.Client{Timeout: 45 * time.Second}}, nil
}

func (p *LiveProvider) AuthorizationURL(state string) (string, error) {
	if len(state) < 32 {
		return "", errors.New("OAuth state is too short")
	}
	endpoint, _ := url.Parse(strings.TrimRight(p.cfg.DialogBaseURL, "/") + "/" + p.cfg.APIVersion + "/dialog/oauth")
	query := endpoint.Query()
	query.Set("client_id", p.cfg.AppID)
	query.Set("redirect_uri", p.cfg.RedirectURL)
	query.Set("state", state)
	query.Set("response_type", "code")
	query.Set("scope", strings.Join([]string{"pages_show_list", "pages_read_engagement", "pages_manage_posts", "instagram_basic", "instagram_content_publish", "business_management", "ads_management", "ads_read", "read_insights"}, ","))
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}

func (p *LiveProvider) ExchangeCode(ctx context.Context, code string) (Token, error) {
	var exchange struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	err := p.call(ctx, http.MethodGet, "oauth/access_token", url.Values{"client_id": {p.cfg.AppID}, "client_secret": {p.cfg.AppSecret}, "redirect_uri": {p.cfg.RedirectURL}, "code": {code}}, "", &exchange)
	if err != nil {
		return Token{}, err
	}
	if exchange.AccessToken == "" {
		return Token{}, &ProviderError{Category: "AUTH", Code: "missing_token", SafeMessage: "Meta did not return an access token"}
	}
	var debug struct {
		Data struct {
			Type        string   `json:"type"`
			UserID      string   `json:"user_id"`
			IsValid     bool     `json:"is_valid"`
			Scopes      []string `json:"scopes"`
			IssuedAt    int64    `json:"issued_at"`
			ExpiresAt   int64    `json:"expires_at"`
			DataExpires int64    `json:"data_access_expires_at"`
		} `json:"data"`
	}
	appToken := p.cfg.AppID + "|" + p.cfg.AppSecret
	if err = p.call(ctx, http.MethodGet, "debug_token", url.Values{"input_token": {exchange.AccessToken}, "access_token": {appToken}}, "", &debug); err != nil {
		return Token{}, err
	}
	if !debug.Data.IsValid {
		return Token{}, ErrUnauthorized
	}
	var profile struct{ ID, Name string }
	if err = p.call(ctx, http.MethodGet, "me", url.Values{"fields": {"id,name"}}, exchange.AccessToken, &profile); err != nil {
		return Token{}, err
	}
	token := Token{AccessToken: exchange.AccessToken, TokenType: exchange.TokenType, UserID: profile.ID, UserName: profile.Name, Scopes: debug.Data.Scopes}
	token.IssuedAt, token.ExpiresAt, token.DataAccessExpiresAt = unixTime(debug.Data.IssuedAt), unixTime(debug.Data.ExpiresAt), unixTime(debug.Data.DataExpires)
	if token.ExpiresAt == nil && exchange.ExpiresIn > 0 {
		expires := time.Now().UTC().Add(time.Duration(exchange.ExpiresIn) * time.Second)
		token.ExpiresAt = &expires
	}
	return token, nil
}

func (p *LiveProvider) Discover(ctx context.Context, token string) (Discovery, error) {
	var result Discovery
	var pages struct {
		Data []struct {
			ID          string   `json:"id"`
			Name        string   `json:"name"`
			AccessToken string   `json:"access_token"`
			Tasks       []string `json:"tasks"`
			Instagram   *struct {
				ID, Username, Name string
				PictureURL         string `json:"profile_picture_url"`
			} `json:"instagram_business_account"`
		} `json:"data"`
	}
	if err := p.call(ctx, http.MethodGet, "me/accounts", url.Values{"fields": {"id,name,access_token,tasks,instagram_business_account{id,username,name,profile_picture_url}"}, "limit": {"100"}}, token, &pages); err != nil {
		return result, err
	}
	for _, item := range pages.Data {
		page := Page{ID: item.ID, Name: item.Name, AccessToken: item.AccessToken, Tasks: item.Tasks}
		if item.Instagram != nil {
			page.Instagram = &InstagramAccount{ID: item.Instagram.ID, Username: item.Instagram.Username, Name: item.Instagram.Name, PictureURL: item.Instagram.PictureURL}
		}
		result.Pages = append(result.Pages, page)
	}
	var businesses struct {
		Data []struct {
			ID                 string `json:"id"`
			Name               string `json:"name"`
			VerificationStatus string `json:"verification_status"`
		} `json:"data"`
	}
	if err := p.call(ctx, http.MethodGet, "me/businesses", url.Values{"fields": {"id,name,verification_status"}, "limit": {"100"}}, token, &businesses); err == nil {
		for _, item := range businesses.Data {
			result.Businesses = append(result.Businesses, Business{ID: item.ID, Name: item.Name, VerificationStatus: item.VerificationStatus})
		}
	}
	var accounts struct {
		Data []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Currency      string `json:"currency"`
			TimezoneName  string `json:"timezone_name"`
			AccountStatus int    `json:"account_status"`
			SpendCap      string `json:"spend_cap"`
			AmountSpent   string `json:"amount_spent"`
		} `json:"data"`
	}
	if err := p.call(ctx, http.MethodGet, "me/adaccounts", url.Values{"fields": {"id,name,currency,timezone_name,account_status,spend_cap,amount_spent"}, "limit": {"100"}}, token, &accounts); err == nil {
		for _, item := range accounts.Data {
			result.AdAccounts = append(result.AdAccounts, AdAccount{ID: item.ID, Name: item.Name, Currency: item.Currency, TimezoneName: item.TimezoneName, AccountStatus: item.AccountStatus, SpendCapMinor: int64Value(item.SpendCap), AmountSpentMinor: int64Value(item.AmountSpent)})
		}
	}
	for _, account := range result.AdAccounts {
		var pixels struct {
			Data []struct{ ID, Name string } `json:"data"`
		}
		if err := p.call(ctx, http.MethodGet, account.ID+"/adspixels", url.Values{"fields": {"id,name"}, "limit": {"100"}}, token, &pixels); err == nil {
			for _, item := range pixels.Data {
				result.Pixels = append(result.Pixels, Pixel{ID: item.ID, AdAccountID: account.ID, Name: item.Name})
			}
		}
		var audiences struct {
			Data []struct {
				ID, Name, Subtype string
				ApproximateCount  int64 `json:"approximate_count_lower_bound"`
			} `json:"data"`
		}
		if err := p.call(ctx, http.MethodGet, account.ID+"/customaudiences", url.Values{"fields": {"id,name,subtype,approximate_count_lower_bound"}, "limit": {"100"}}, token, &audiences); err == nil {
			for _, item := range audiences.Data {
				result.Audiences = append(result.Audiences, Audience{ID: item.ID, AdAccountID: account.ID, Name: item.Name, Type: "CUSTOM", Subtype: item.Subtype, ApproximateCount: item.ApproximateCount})
			}
		}
	}
	return result, nil
}

func (p *LiveProvider) Publish(ctx context.Context, request PublishRequest) (PublishResult, error) {
	var created struct {
		ID        string `json:"id"`
		RequestID string `json:"fbtrace_id"`
	}
	if request.Platform == "FACEBOOK" {
		path, values := request.PageID+"/photos", url.Values{"url": {request.MediaURL}, "caption": {request.Caption}, "published": {"true"}}
		if strings.HasPrefix(request.MediaType, "video/") {
			path, values = request.PageID+"/videos", url.Values{"file_url": {request.MediaURL}, "description": {request.Caption}, "published": {"true"}}
		}
		if err := p.call(ctx, http.MethodPost, path, values, request.AccessToken, &created); err != nil {
			return PublishResult{}, err
		}
	} else {
		values := url.Values{"caption": {request.Caption}}
		if strings.HasPrefix(request.MediaType, "video/") {
			values.Set("media_type", "REELS")
			values.Set("video_url", request.MediaURL)
		} else {
			values.Set("image_url", request.MediaURL)
		}
		var container struct {
			ID string `json:"id"`
		}
		if err := p.call(ctx, http.MethodPost, request.InstagramID+"/media", values, request.AccessToken, &container); err != nil {
			return PublishResult{}, err
		}
		var status struct {
			StatusCode string `json:"status_code"`
		}
		if err := p.call(ctx, http.MethodGet, container.ID, url.Values{"fields": {"status_code"}}, request.AccessToken, &status); err != nil {
			return PublishResult{}, err
		}
		if status.StatusCode != "FINISHED" {
			return PublishResult{}, &ProviderError{Category: "PROCESSING", Code: "container_not_ready", SafeMessage: "Instagram media is still processing", Retryable: true}
		}
		if err := p.call(ctx, http.MethodPost, request.InstagramID+"/media_publish", url.Values{"creation_id": {container.ID}}, request.AccessToken, &created); err != nil {
			return PublishResult{}, err
		}
	}
	var permalink struct {
		Permalink string `json:"permalink"`
	}
	_ = p.call(ctx, http.MethodGet, created.ID, url.Values{"fields": {"permalink"}}, request.AccessToken, &permalink)
	return PublishResult{ProviderPostID: created.ID, PublicURL: permalink.Permalink, RequestID: created.RequestID}, nil
}

func (p *LiveProvider) CreatePausedCampaign(ctx context.Context, request CampaignRequest) (CampaignResult, error) {
	var created struct {
		ID string `json:"id"`
	}
	values := url.Values{"name": {request.Name}, "objective": {request.Objective}, "buying_type": {request.BuyingType}, "status": {"PAUSED"}, "special_ad_categories": {"[]"}}
	accountID := request.AdAccountID
	if !strings.HasPrefix(accountID, "act_") {
		accountID = "act_" + accountID
	}
	if err := p.call(ctx, http.MethodPost, accountID+"/campaigns", values, request.AccessToken, &created); err != nil {
		return CampaignResult{}, err
	}
	return CampaignResult{ProviderCampaignID: created.ID}, nil
}

func (p *LiveProvider) SetCampaignStatus(ctx context.Context, id, status, token string) (string, error) {
	var result struct {
		Success bool `json:"success"`
	}
	if err := p.call(ctx, http.MethodPost, id, url.Values{"status": {status}}, token, &result); err != nil {
		return "", err
	}
	if !result.Success {
		return "", &ProviderError{Category: "PROVIDER", Code: "status_not_applied", SafeMessage: "Meta did not apply the campaign status"}
	}
	return id, nil
}
func (p *LiveProvider) SetCampaignBudget(ctx context.Context, id string, amount int64, lifetime bool, token string) (string, error) {
	field := "daily_budget"
	if lifetime {
		field = "lifetime_budget"
	}
	var result struct {
		Success bool `json:"success"`
	}
	if err := p.call(ctx, http.MethodPost, id, url.Values{field: {strconv.FormatInt(amount, 10)}}, token, &result); err != nil {
		return "", err
	}
	if !result.Success {
		return "", &ProviderError{Category: "PROVIDER", Code: "budget_not_applied", SafeMessage: "Meta did not apply the campaign budget"}
	}
	return id, nil
}

func (p *LiveProvider) CampaignInsights(ctx context.Context, id, token string) ([]InsightDay, error) {
	var response struct {
		Data []map[string]any `json:"data"`
	}
	fields := "date_start,spend,impressions,reach,clicks,frequency,actions,action_values"
	if err := p.call(ctx, http.MethodGet, id+"/insights", url.Values{"fields": {fields}, "time_increment": {"1"}, "date_preset": {"last_7d"}}, token, &response); err != nil {
		return nil, err
	}
	items := make([]InsightDay, 0, len(response.Data))
	for _, raw := range response.Data {
		date, _ := time.Parse("2006-01-02", stringValue(raw["date_start"]))
		spend, _ := strconv.ParseFloat(stringValue(raw["spend"]), 64)
		item := InsightDay{Date: date, SpendMinor: int64(spend * 100), Impressions: int64Value(stringValue(raw["impressions"])), Reach: int64Value(stringValue(raw["reach"])), Clicks: int64Value(stringValue(raw["clicks"])), Frequency: floatValue(raw["frequency"]), SafeResponse: raw}
		item.Leads = actionValue(raw["actions"], "lead")
		item.Purchases = actionValue(raw["actions"], "purchase")
		item.Conversions = item.Leads + item.Purchases
		item.RevenueMinor = int64(actionValue(raw["action_values"], "purchase") * 100)
		items = append(items, item)
	}
	return items, nil
}

func (p *LiveProvider) call(ctx context.Context, method, path string, values url.Values, token string, output any) error {
	endpoint, _ := url.Parse(strings.TrimRight(p.cfg.GraphBaseURL, "/") + "/" + p.cfg.APIVersion + "/" + strings.TrimLeft(path, "/"))
	if token != "" {
		values.Set("access_token", token)
		values.Set("appsecret_proof", proof(token, p.cfg.AppSecret))
	}
	var body io.Reader
	if method == http.MethodGet {
		endpoint.RawQuery = values.Encode()
	} else {
		body = strings.NewReader(values.Encode())
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return err
	}
	if method != http.MethodGet {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	response, err := p.client.Do(request)
	if err != nil {
		return &ProviderError{Category: "NETWORK", Code: "request_failed", SafeMessage: "Meta request failed", Retryable: true}
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, 2<<20)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var envelope struct {
			Error struct {
				Code, ErrorSubcode int
				IsTransient        bool `json:"is_transient"`
			} `json:"error"`
		}
		_ = json.NewDecoder(limited).Decode(&envelope)
		code := strconv.Itoa(envelope.Error.Code)
		retryable := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 || envelope.Error.IsTransient || map[int]bool{1: true, 2: true, 4: true, 17: true, 32: true, 613: true}[envelope.Error.Code]
		category := "PERMISSION"
		if retryable {
			category = "TRANSIENT"
		}
		if response.StatusCode == http.StatusUnauthorized {
			category = "AUTH"
		}
		return &ProviderError{Category: category, Code: code, SafeMessage: fmt.Sprintf("Meta returned HTTP %d (code %s)", response.StatusCode, code), Retryable: retryable}
	}
	if err = json.NewDecoder(limited).Decode(output); err != nil {
		return &ProviderError{Category: "PROTOCOL", Code: "invalid_json", SafeMessage: "Meta returned an invalid response"}
	}
	return nil
}

func proof(token, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}
func unixTime(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	result := time.Unix(value, 0).UTC()
	return &result
}
func int64Value(value string) int64 { result, _ := strconv.ParseInt(value, 10, 64); return result }
func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}
func floatValue(value any) float64 {
	result, _ := strconv.ParseFloat(stringValue(value), 64)
	return result
}
func actionValue(value any, name string) float64 {
	items, _ := value.([]any)
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if stringValue(item["action_type"]) == name {
			return floatValue(item["value"])
		}
	}
	return 0
}
