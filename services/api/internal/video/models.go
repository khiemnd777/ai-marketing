package video

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Generation struct {
	ID               uuid.UUID       `json:"id"`
	SceneID          uuid.UUID       `json:"sceneId"`
	SceneVersion     int32           `json:"sceneVersion"`
	Provider         string          `json:"provider"`
	ProviderTaskID   *string         `json:"providerTaskId"`
	Status           string          `json:"status"`
	AttemptNumber    int32           `json:"attemptNumber"`
	Model            string          `json:"model"`
	APIVersion       string          `json:"apiVersion"`
	Resolution       string          `json:"resolution"`
	AspectRatio      string          `json:"aspectRatio"`
	DurationSeconds  int32           `json:"durationSeconds"`
	GenerateAudio    bool            `json:"generateAudio"`
	SceneHash        string          `json:"sceneHash"`
	OutputAssetID    *uuid.UUID      `json:"outputAssetId"`
	EstimatedCostUSD float64         `json:"estimatedCostUsd"`
	ActualCostUSD    *float64        `json:"actualCostUsd"`
	UsageTokens      *int64          `json:"usageTokens"`
	ErrorCategory    *string         `json:"errorCategory"`
	ErrorCode        *string         `json:"errorCode"`
	ErrorMessage     *string         `json:"errorMessage"`
	ReviewNotes      string          `json:"reviewNotes"`
	Version          int64           `json:"version"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
	Reused           bool            `json:"reused,omitempty"`
	Selected         bool            `json:"selected"`
	Transcription    *Transcription  `json:"transcription,omitempty"`
	QualityCheck     *QualityCheck   `json:"qualityCheck,omitempty"`
	Edit             *GenerationEdit `json:"edit,omitempty"`
}

type Transcription struct {
	Status     string          `json:"status"`
	Provider   string          `json:"provider"`
	Model      string          `json:"model"`
	Language   *string         `json:"language"`
	Transcript string          `json:"transcript"`
	Segments   json.RawMessage `json:"segments"`
	ErrorCode  *string         `json:"errorCode"`
}

type QualityCheck struct {
	Status                   string          `json:"status"`
	DeterministicPass        *bool           `json:"deterministicPass"`
	TranscriptPass           *bool           `json:"transcriptPass"`
	VideoDecodes             *bool           `json:"videoDecodes"`
	DurationPass             *bool           `json:"durationPass"`
	ResolutionPass           *bool           `json:"resolutionPass"`
	AudioStreamPresent       *bool           `json:"audioStreamPresent"`
	SilenceWarning           *bool           `json:"silenceWarning"`
	TranscriptDiff           json.RawMessage `json:"transcriptDiff"`
	Findings                 json.RawMessage `json:"findings"`
	CharacterCountReview     *int32          `json:"characterCountReview"`
	DuplicateCharacterReview *bool           `json:"duplicateCharacterReview"`
	DuplicateProductReview   *bool           `json:"duplicateProductReview"`
	ProductColorMismatch     *bool           `json:"productColorMismatch"`
	BlurOrLowQualityWarning  *bool           `json:"blurOrLowQualityWarning"`
	CropWarning              *bool           `json:"cropWarning"`
	SubtitleOverflow         *bool           `json:"subtitleOverflow"`
	LogoOverlap              *bool           `json:"logoOverlap"`
	CTASafeZoneViolation     *bool           `json:"ctaSafeZoneViolation"`
	HumanNotes               string          `json:"humanNotes"`
	Version                  int64           `json:"version"`
}

type GenerationEdit struct {
	TrimStartMS             int64       `json:"trimStartMs"`
	TrimEndMS               *int64      `json:"trimEndMs"`
	MuteAudio               bool        `json:"muteAudio"`
	Transition              string      `json:"transition"`
	ReplacementAssetID      *uuid.UUID  `json:"replacementAssetId"`
	AttachedProductAssetIDs []uuid.UUID `json:"attachedProductAssetIds"`
	SubtitlePreview         bool        `json:"subtitlePreview"`
	Version                 int64       `json:"version"`
}

type StartInput struct {
	Resolution    string `json:"resolution"`
	AspectRatio   string `json:"aspectRatio"`
	GenerateAudio bool   `json:"generateAudio"`
}

type ReviewInput struct {
	Action                  string `json:"action"`
	Version                 int64  `json:"version"`
	Notes                   string `json:"notes"`
	CharacterCount          *int32 `json:"characterCount"`
	DuplicateCharacter      *bool  `json:"duplicateCharacter"`
	DuplicateProduct        *bool  `json:"duplicateProduct"`
	ProductColorMismatch    *bool  `json:"productColorMismatch"`
	BlurOrLowQualityWarning *bool  `json:"blurOrLowQualityWarning"`
	CropWarning             *bool  `json:"cropWarning"`
	SubtitleOverflow        *bool  `json:"subtitleOverflow"`
	LogoOverlap             *bool  `json:"logoOverlap"`
	CTASafeZoneViolation    *bool  `json:"ctaSafeZoneViolation"`
}
