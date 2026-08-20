package rendering

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

type ObjectReference struct {
	ObjectKey   string `json:"objectKey"`
	SHA256      string `json:"sha256"`
	ContentType string `json:"contentType"`
}

type ManifestScene struct {
	SceneID      string            `json:"sceneId"`
	SceneVersion int32             `json:"sceneVersion"`
	Source       ObjectReference   `json:"source"`
	DurationMS   int64             `json:"durationMs"`
	TrimStartMS  int64             `json:"trimStartMs"`
	TrimEndMS    int64             `json:"trimEndMs"`
	Muted        bool              `json:"muted"`
	Transition   string            `json:"transition"`
	ProductMedia []ObjectReference `json:"productMedia"`
}

type Overlay struct {
	Type         string  `json:"type"`
	Value        string  `json:"value"`
	StartFrame   int32   `json:"startFrame"`
	EndFrame     int32   `json:"endFrame"`
	SafeZone     string  `json:"safeZone"`
	SourceFactID *string `json:"sourceFactId"`
}

type CaptionCue struct {
	StartMS int64   `json:"startMs"`
	EndMS   int64   `json:"endMs"`
	Text    string  `json:"text"`
	Speaker *string `json:"speaker"`
}

type SoundEffect struct {
	Source  ObjectReference `json:"source"`
	StartMS int64           `json:"startMs"`
	GainDB  float64         `json:"gainDb"`
}

type RenderManifest struct {
	RenderID            string `json:"renderId"`
	ManifestVersion     int32  `json:"manifestVersion"`
	WorkspaceID         string `json:"workspaceId"`
	CampaignID          string `json:"campaignId"`
	VideoProjectID      string `json:"videoProjectId"`
	VideoProjectVersion int32  `json:"videoProjectVersion"`
	VideoProjectHash    string `json:"videoProjectHash"`
	Language            string `json:"language"`
	Output              struct {
		Width           int32  `json:"width"`
		Height          int32  `json:"height"`
		FPS             int32  `json:"fps"`
		DurationSeconds int32  `json:"durationSeconds"`
		Codec           string `json:"codec"`
	} `json:"output"`
	Scenes             []ManifestScene  `json:"scenes"`
	Overlays           []Overlay        `json:"overlays"`
	Captions           []CaptionCue     `json:"captions"`
	BurnCaptions       bool             `json:"burnCaptions"`
	Logo               *ObjectReference `json:"logo"`
	Music              *ObjectReference `json:"music"`
	SoundEffects       []SoundEffect    `json:"soundEffects"`
	MusicGainDB        float64          `json:"musicGainDb"`
	DialogueDuckingDB  float64          `json:"dialogueDuckingDb"`
	OutputObjectKey    string           `json:"outputObjectKey"`
	ThumbnailObjectKey string           `json:"thumbnailObjectKey"`
	CreatedAt          string           `json:"createdAt"`
}

type RendererResult struct {
	RequestID          string  `json:"requestId"`
	Reused             bool    `json:"reused"`
	OutputObjectKey    string  `json:"outputObjectKey"`
	ThumbnailObjectKey string  `json:"thumbnailObjectKey"`
	ChecksumSHA256     string  `json:"checksumSha256"`
	FileSizeBytes      int64   `json:"fileSizeBytes"`
	Width              int32   `json:"width"`
	Height             int32   `json:"height"`
	FPS                float64 `json:"fps"`
	DurationMS         int64   `json:"durationMs"`
	Codec              string  `json:"codec"`
	AudioCodec         *string `json:"audioCodec"`
}

type RendererClient struct {
	baseURL, secret string
	client          *http.Client
}

const rendererRequestTimeout = 20 * time.Minute

func NewRendererClient(cfg config.RendererConfig) (*RendererClient, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" || len(cfg.SharedSecret) < 32 {
		return nil, errors.New("renderer URL and a 32-character shared secret are required")
	}
	return &RendererClient{baseURL: strings.TrimRight(cfg.BaseURL, "/"), secret: cfg.SharedSecret, client: &http.Client{Timeout: rendererRequestTimeout}}, nil
}

func (c *RendererClient) Render(ctx context.Context, raw []byte) (RendererResult, error) {
	ctx, span := otel.Tracer("studio-worker/renderer").Start(ctx, "renderer.render", trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(attribute.String("server.address", c.baseURL)))
	defer span.End()
	mac := hmac.New(sha256.New, []byte(c.secret))
	_, _ = mac.Write(raw)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/renders", bytes.NewReader(raw))
	if err != nil {
		return RendererResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Render-Signature", hex.EncodeToString(mac.Sum(nil)))
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	response, err := c.client.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "renderer request failed")
		return RendererResult{}, err
	}
	defer response.Body.Close()
	span.SetAttributes(attribute.Int("http.response.status_code", response.StatusCode))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		span.SetStatus(codes.Error, "renderer returned an error")
		return RendererResult{}, fmt.Errorf("renderer returned HTTP %d", response.StatusCode)
	}
	var result RendererResult
	if err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return RendererResult{}, err
	}
	if result.OutputObjectKey == "" || result.ChecksumSHA256 == "" || result.Width != 1080 || result.Height != 1920 || result.FileSizeBytes <= 0 {
		span.SetStatus(codes.Error, "renderer metadata invalid")
		return RendererResult{}, errors.New("renderer returned invalid output metadata")
	}
	span.SetAttributes(attribute.String("renderer.request.id", result.RequestID), attribute.Bool("renderer.reused", result.Reused))
	return result, nil
}
