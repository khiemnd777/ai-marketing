package ai

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

const PromptVersion = "planning-v1.0.0"

type ProductFact struct {
	ID         uuid.UUID `json:"id"`
	Key        string    `json:"key"`
	ExactValue string    `json:"exactValue"`
	Locked     bool      `json:"locked"`
}

type PlanningContext struct {
	CampaignID       uuid.UUID     `json:"campaignId"`
	CampaignName     string        `json:"campaignName"`
	Objective        string        `json:"objective"`
	Audience         string        `json:"audience"`
	Market           string        `json:"market"`
	Language         string        `json:"language"`
	VideoFormat      string        `json:"videoFormat"`
	DurationSeconds  int32         `json:"durationSeconds"`
	Tone             string        `json:"tone"`
	Offer            string        `json:"offer"`
	CTA              string        `json:"cta"`
	BrandName        string        `json:"brandName"`
	ProductName      string        `json:"productName"`
	ProductTruth     []ProductFact `json:"productTruth"`
	ProhibitedClaims []string      `json:"prohibitedClaims"`
}

type ConceptInput struct {
	Context PlanningContext `json:"context"`
}
type ConceptCandidate struct {
	Title                      string   `json:"title"`
	Hook                       string   `json:"hook"`
	CoreMessage                string   `json:"coreMessage"`
	AudienceFit                string   `json:"audienceFit"`
	VideoFormat                string   `json:"videoFormat"`
	CharacterRoles             []string `json:"characterRoles"`
	Environment                string   `json:"environment"`
	ProductPlacement           string   `json:"productPlacement"`
	ExpectedSceneCount         int32    `json:"expectedSceneCount"`
	ExpectedSeedanceSeconds    int32    `json:"expectedSeedanceSeconds"`
	RequiredRealProductFootage []string `json:"requiredRealProductFootage"`
	EstimatedCostUSD           float64  `json:"estimatedCostUsd"`
	GenerationRisks            []string `json:"generationRisks"`
	CampaignFitReason          string   `json:"campaignFitReason"`
}
type ConceptOutput struct {
	Concepts []ConceptCandidate `json:"concepts"`
}

type ContentInput struct {
	Context PlanningContext `json:"context"`
}
type ContentVariant struct {
	Key      string `json:"key"`
	Platform string `json:"platform"`
	Content  string `json:"content"`
}
type ContentOutput struct {
	Variants []ContentVariant `json:"variants"`
}

type DialogueTurn struct {
	Order               int32  `json:"order"`
	CharacterRole       string `json:"characterRole"`
	Dialogue            string `json:"dialogue"`
	EstimatedDurationMS int64  `json:"estimatedDurationMs"`
}
type ScriptInput struct {
	Context PlanningContext `json:"context"`
}
type ScriptOutput struct {
	Hook                       string            `json:"hook"`
	Introduction               string            `json:"introduction"`
	Problem                    string            `json:"problem"`
	ProductSolution            string            `json:"productSolution"`
	ProductFeatures            []string          `json:"productFeatures"`
	Benefits                   []string          `json:"benefits"`
	CTA                        string            `json:"cta"`
	Closing                    string            `json:"closing"`
	ApproximateDurationSeconds int32             `json:"approximateDurationSeconds"`
	CharacterRoles             map[string]string `json:"characterRoles"`
	SpokenLanguage             string            `json:"spokenLanguage"`
	DialogueTurns              []DialogueTurn    `json:"dialogueTurns"`
}

type SceneInput struct {
	Context             PlanningContext `json:"context"`
	Script              ScriptOutput    `json:"script"`
	SpeakerCharacterID  uuid.UUID       `json:"speakerCharacterId"`
	ListenerCharacterID uuid.UUID       `json:"listenerCharacterId"`
}
type SceneDirection struct {
	SceneID                string      `json:"sceneId"`
	Order                  int32       `json:"order"`
	DurationSeconds        int32       `json:"durationSeconds"`
	GenerationMethod       string      `json:"generationMethod"`
	SpeakerCharacterID     uuid.UUID   `json:"speakerCharacterId"`
	ListenerCharacterID    uuid.UUID   `json:"listenerCharacterId"`
	Dialogue               string      `json:"dialogue"`
	SpeakerAction          string      `json:"speakerAction"`
	ListenerAction         string      `json:"listenerAction"`
	Camera                 string      `json:"camera"`
	Environment            string      `json:"environment"`
	ProductPlacement       string      `json:"productPlacement"`
	ReferenceAssetIDs      []uuid.UUID `json:"referenceAssetIds"`
	RequiredProductFactIDs []uuid.UUID `json:"requiredProductFactIds"`
	ExpectedCostUSD        float64     `json:"expectedCostUsd"`
	SeedancePrompt         string      `json:"seedancePrompt"`
}
type SceneOutput struct {
	Scenes []SceneDirection `json:"scenes"`
}

type AuditInput struct {
	Context PlanningContext `json:"context"`
	Content string          `json:"content"`
}
type AuditOutput struct {
	Pass                    bool     `json:"pass"`
	DuplicateProductReview  bool     `json:"duplicateProductReview"`
	ProductColorMismatch    bool     `json:"productColorMismatch"`
	BlurOrLowQualityWarning bool     `json:"blurOrLowQualityWarning"`
	CropWarning             bool     `json:"cropWarning"`
	SubtitleOverflow        bool     `json:"subtitleOverflow"`
	LogoOverlap             bool     `json:"logoOverlap"`
	CTASafeZoneViolation    bool     `json:"ctaSafeZoneViolation"`
	Findings                []string `json:"findings"`
}

type Metadata struct {
	Provider         string
	Model            string
	RequestID        string
	PromptVersion    string
	InputTokens      int64
	OutputTokens     int64
	EstimatedCostUSD float64
	ActualCostUSD    float64
	Latency          time.Duration
}

type LLMProvider interface {
	GenerateConcepts(context.Context, ConceptInput) (ConceptOutput, Metadata, error)
	GenerateContent(context.Context, ContentInput) (ContentOutput, Metadata, error)
	GenerateScript(context.Context, ScriptInput) (ScriptOutput, Metadata, error)
	GenerateScenes(context.Context, SceneInput) (SceneOutput, Metadata, error)
	AuditContent(context.Context, AuditInput) (AuditOutput, Metadata, error)
}

func NewProvider(cfg config.Config) (LLMProvider, error) {
	if cfg.DemoMode {
		return NewDemoProvider(cfg.OpenAI.Model), nil
	}
	return NewOpenAIProvider(cfg.OpenAI)
}
