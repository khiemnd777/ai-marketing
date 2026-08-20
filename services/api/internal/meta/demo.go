package meta

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

type DemoProvider struct{}

func NewDemoProvider() *DemoProvider { return &DemoProvider{} }
func (d *DemoProvider) AuthorizationURL(state string) (string, error) {
	return "http://localhost:8080/v1/meta/oauth/callback?code=demo-code&state=" + url.QueryEscape(state), nil
}
func (d *DemoProvider) ExchangeCode(_ context.Context, _ string) (Token, error) {
	now, expires := time.Now().UTC(), time.Now().UTC().Add(60*24*time.Hour)
	return Token{AccessToken: "demo-user-token", TokenType: "USER", UserID: "demo-user", UserName: "Demo Meta Operator", Scopes: []string{"pages_manage_posts", "instagram_content_publish", "ads_management", "read_insights"}, IssuedAt: &now, ExpiresAt: &expires, DataAccessExpiresAt: &expires}, nil
}
func (d *DemoProvider) Discover(_ context.Context, _ string) (Discovery, error) {
	return Discovery{
		Pages:      []Page{{ID: "demo-page", Name: "Northstar Travel", AccessToken: "demo-page-token", Tasks: []string{"CREATE_CONTENT", "MANAGE"}, Instagram: &InstagramAccount{ID: "demo-instagram", Username: "northstar.travel", Name: "Northstar Travel"}}},
		Businesses: []Business{{ID: "demo-business", Name: "Northstar Demo Business", VerificationStatus: "verified"}},
		AdAccounts: []AdAccount{{ID: "act_demo", Name: "Northstar Demo Ads", Currency: "VND", TimezoneName: "Asia/Ho_Chi_Minh", AccountStatus: 1, SpendCapMinor: 10_000_000_00}},
		Pixels:     []Pixel{{ID: "demo-pixel", AdAccountID: "act_demo", Name: "Northstar Website"}},
		Audiences:  []Audience{{ID: "demo-audience", AdAccountID: "act_demo", Name: "Website visitors 30d", Type: "CUSTOM", Subtype: "WEBSITE", ApproximateCount: 12500}},
	}, nil
}
func (d *DemoProvider) Publish(_ context.Context, request PublishRequest) (PublishResult, error) {
	return PublishResult{ProviderPostID: fmt.Sprintf("demo-%s-post", request.Platform), PublicURL: "https://example.test/meta/demo-post", RequestID: "demo-publish-request"}, nil
}
func (d *DemoProvider) CreatePausedCampaign(_ context.Context, _ CampaignRequest) (CampaignResult, error) {
	return CampaignResult{ProviderCampaignID: "demo-paused-campaign", RequestID: "demo-ad-create-request"}, nil
}
func (d *DemoProvider) SetCampaignStatus(_ context.Context, id, status, _ string) (string, error) {
	return "demo-" + id + "-" + status, nil
}
func (d *DemoProvider) SetCampaignBudget(_ context.Context, id string, amount int64, _ bool, _ string) (string, error) {
	return fmt.Sprintf("demo-%s-budget-%d", id, amount), nil
}
func (d *DemoProvider) CampaignInsights(_ context.Context, _ string, _ string) ([]InsightDay, error) {
	return []InsightDay{{Date: time.Now().UTC().AddDate(0, 0, -1), SpendMinor: 250000, Impressions: 12000, Reach: 9000, Clicks: 240, Conversions: 18, Leads: 18, Purchases: 3, RevenueMinor: 1200000, Frequency: 1.33, SafeResponse: map[string]any{"provider": "demo"}}}, nil
}
