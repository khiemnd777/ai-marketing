package meta

import (
	"context"
	"errors"
	"time"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

var (
	ErrConfiguration = errors.New("Meta provider is not configured")
	ErrUnauthorized  = errors.New("Meta token is invalid or expired")
)

type ProviderError struct {
	Category, Code, SafeMessage string
	Retryable                   bool
}

func (e *ProviderError) Error() string { return e.SafeMessage }

type Token struct {
	AccessToken, TokenType, UserID, UserName string
	Scopes                                   []string
	IssuedAt, ExpiresAt, DataAccessExpiresAt *time.Time
}

type InstagramAccount struct{ ID, Username, Name, PictureURL string }
type Page struct {
	ID, Name, AccessToken string
	Tasks                 []string
	Instagram             *InstagramAccount
}
type Business struct{ ID, Name, VerificationStatus string }
type AdAccount struct {
	ID, Name, Currency, TimezoneName string
	AccountStatus                    int
	SpendCapMinor, AmountSpentMinor  int64
}
type Pixel struct{ ID, AdAccountID, Name string }
type Audience struct {
	ID, AdAccountID, Name, Type, Subtype string
	ApproximateCount                     int64
}
type Discovery struct {
	Pages      []Page
	Businesses []Business
	AdAccounts []AdAccount
	Pixels     []Pixel
	Audiences  []Audience
}

type PublishRequest struct {
	Platform, PageID, InstagramID, MediaURL, MediaType, Caption, AccessToken string
}
type PublishResult struct{ ProviderPostID, PublicURL, RequestID string }

type CampaignRequest struct {
	AdAccountID, Name, Objective, BuyingType, AccessToken string
}
type CampaignResult struct{ ProviderCampaignID, RequestID string }

type InsightDay struct {
	Date                                     time.Time
	SpendMinor, Impressions, Reach, Clicks   int64
	Conversions, Leads, Purchases, Frequency float64
	RevenueMinor                             int64
	SafeResponse                             map[string]any
}

type Provider interface {
	AuthorizationURL(string) (string, error)
	ExchangeCode(context.Context, string) (Token, error)
	Discover(context.Context, string) (Discovery, error)
	Publish(context.Context, PublishRequest) (PublishResult, error)
	CreatePausedCampaign(context.Context, CampaignRequest) (CampaignResult, error)
	SetCampaignStatus(context.Context, string, string, string) (string, error)
	SetCampaignBudget(context.Context, string, int64, bool, string) (string, error)
	CampaignInsights(context.Context, string, string) ([]InsightDay, error)
}

func NewProvider(demo bool, cfg config.MetaConfig) (Provider, error) {
	if demo {
		return NewDemoProvider(), nil
	}
	return NewLiveProvider(cfg)
}

type UnavailableProvider struct{}

func NewUnavailableProvider() Provider                               { return &UnavailableProvider{} }
func (*UnavailableProvider) AuthorizationURL(string) (string, error) { return "", ErrConfiguration }
func (*UnavailableProvider) ExchangeCode(context.Context, string) (Token, error) {
	return Token{}, ErrConfiguration
}
func (*UnavailableProvider) Discover(context.Context, string) (Discovery, error) {
	return Discovery{}, ErrConfiguration
}
func (*UnavailableProvider) Publish(context.Context, PublishRequest) (PublishResult, error) {
	return PublishResult{}, ErrConfiguration
}
func (*UnavailableProvider) CreatePausedCampaign(context.Context, CampaignRequest) (CampaignResult, error) {
	return CampaignResult{}, ErrConfiguration
}
func (*UnavailableProvider) SetCampaignStatus(context.Context, string, string, string) (string, error) {
	return "", ErrConfiguration
}
func (*UnavailableProvider) SetCampaignBudget(context.Context, string, int64, bool, string) (string, error) {
	return "", ErrConfiguration
}
func (*UnavailableProvider) CampaignInsights(context.Context, string, string) ([]InsightDay, error) {
	return nil, ErrConfiguration
}
