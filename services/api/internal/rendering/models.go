package rendering

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ProjectInput struct {
	Headline          string     `json:"headline"`
	LowerThird        string     `json:"lowerThird"`
	ShowPrice         bool       `json:"showPrice"`
	ShowDiscountCode  bool       `json:"showDiscountCode"`
	ShowCTA           bool       `json:"showCta"`
	ShowWebsite       bool       `json:"showWebsite"`
	ShowPhone         bool       `json:"showPhone"`
	ShowQRCode        bool       `json:"showQrCode"`
	ShowDisclaimer    bool       `json:"showDisclaimer"`
	BurnCaptions      bool       `json:"burnCaptions"`
	MusicAssetID      *uuid.UUID `json:"musicAssetId"`
	MusicGainDB       float64    `json:"musicGainDb"`
	DialogueDuckingDB float64    `json:"dialogueDuckingDb"`
	ChangeSummary     string     `json:"changeSummary"`
	Version           int64      `json:"version"`
}

type Project struct {
	ID             uuid.UUID  `json:"id"`
	CampaignID     uuid.UUID  `json:"campaignId"`
	CurrentVersion int32      `json:"currentVersion"`
	ProjectHash    string     `json:"projectHash"`
	SelectedJobID  *uuid.UUID `json:"selectedRenderJobId"`
	ProjectInput
	UpdatedAt time.Time `json:"updatedAt"`
}

type RenderJob struct {
	ID                  uuid.UUID       `json:"id"`
	CampaignID          uuid.UUID       `json:"campaignId"`
	VideoProjectID      uuid.UUID       `json:"videoProjectId"`
	VideoProjectVersion int32           `json:"videoProjectVersion"`
	RenderManifestID    *uuid.UUID      `json:"renderManifestId"`
	Status              string          `json:"status"`
	OutputAssetID       *uuid.UUID      `json:"outputAssetId"`
	OutputHash          *string         `json:"outputHash"`
	ThumbnailStorageKey *string         `json:"thumbnailStorageKey"`
	SRTStorageKey       *string         `json:"srtStorageKey"`
	VTTStorageKey       *string         `json:"vttStorageKey"`
	RendererRequestID   *string         `json:"rendererRequestId"`
	ErrorCode           *string         `json:"errorCode"`
	ErrorMessage        *string         `json:"errorMessage"`
	SanitizedResponse   json.RawMessage `json:"sanitizedResponse"`
	ReviewNotes         string          `json:"reviewNotes"`
	Version             int64           `json:"version"`
	CreatedAt           time.Time       `json:"createdAt"`
	UpdatedAt           time.Time       `json:"updatedAt"`
	Reused              bool            `json:"reused,omitempty"`
	Selected            bool            `json:"selected"`
}

type ReviewInput struct {
	Action  string `json:"action"`
	Version int64  `json:"version"`
	Notes   string `json:"notes"`
}
