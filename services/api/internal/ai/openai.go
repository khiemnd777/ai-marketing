package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

type OpenAIProvider struct {
	config config.OpenAIConfig
	client *http.Client
}

func NewOpenAIProvider(cfg config.OpenAIConfig) (*OpenAIProvider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &OpenAIProvider{config: cfg, client: &http.Client{Timeout: cfg.Timeout}}, nil
}

func (p *OpenAIProvider) GenerateConcepts(ctx context.Context, input ConceptInput) (ConceptOutput, Metadata, error) {
	var output ConceptOutput
	metadata, err := p.call(ctx, "campaign_concepts", conceptSchema, "Generate two to four distinct campaign concepts. Include both supported formats. Product Truth is authoritative. Input:\n", input, &output)
	return output, metadata, err
}
func (p *OpenAIProvider) GenerateContent(ctx context.Context, input ContentInput) (ContentOutput, Metadata, error) {
	var output ContentOutput
	metadata, err := p.call(ctx, "campaign_content", contentSchema, "Generate exactly the 14 required marketing variants: facebook_caption, instagram_caption, reels_caption, primary_ads_text, headline, description, cta_variants, hook_a, hook_b, hook_c, hashtags, first_comment, thumbnail_text, retargeting_message. Never fabricate experience or facts. Input:\n", input, &output)
	return output, metadata, err
}
func (p *OpenAIProvider) GenerateScript(ctx context.Context, input ScriptInput) (ScriptOutput, Metadata, error) {
	var output ScriptOutput
	metadata, err := p.call(ctx, "campaign_script", scriptSchema, "Generate a natural two-character spoken script for the exact selected duration. One primary speaker per turn; keep URLs and exact technical text visual. Input:\n", input, &output)
	return output, metadata, err
}
func (p *OpenAIProvider) GenerateScenes(ctx context.Context, input SceneInput) (SceneOutput, Metadata, error) {
	var output SceneOutput
	metadata, err := p.call(ctx, "campaign_scenes", sceneSchema, "Direct the approved script into short independent scenes. Each Seedance scene has exactly two different supplied characters, one speaker, and a closed-mouth listener. Do not ask video generation to render exact text. Input:\n", input, &output)
	return output, metadata, err
}
func (p *OpenAIProvider) AuditContent(ctx context.Context, input AuditInput) (AuditOutput, Metadata, error) {
	var output AuditOutput
	metadata, err := p.call(ctx, "content_audit", auditSchema, "Audit the supplied content against Product Truth and flag review risks. Automated checks do not guarantee correctness. Input:\n", input, &output)
	return output, metadata, err
}

func (p *OpenAIProvider) call(ctx context.Context, name string, schema map[string]any, instruction string, input any, output any) (Metadata, error) {
	started := time.Now()
	encodedInput, err := json.Marshal(input)
	if err != nil {
		return Metadata{}, err
	}
	body := map[string]any{
		"model":     p.config.Model,
		"input":     []map[string]any{{"role": "developer", "content": []map[string]string{{"type": "input_text", "text": "You are the server-side planning engine for an internal product marketing studio. Follow the strict output schema and never introduce facts not present in Product Truth."}}}, {"role": "user", "content": []map[string]string{{"type": "input_text", "text": instruction + string(encodedInput)}}}},
		"reasoning": map[string]any{"effort": p.config.ReasoningEffort},
		"text":      map[string]any{"format": map[string]any{"type": "json_schema", "name": name, "strict": true, "schema": json.RawMessage(schemaJSON(schema))}},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Metadata{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.config.BaseURL, "/")+"/responses", bytes.NewReader(payload))
	if err != nil {
		return Metadata{}, err
	}
	request.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		category, code, message, retryable := "NETWORK", "request_failed", "OpenAI request failed", true
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			category, code, message = "TIMEOUT", "timeout", "OpenAI request timed out"
		} else if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			category, code, message, retryable = "CANCELLED", "cancelled", "OpenAI request was cancelled", false
		}
		return Metadata{}, &ProviderError{Category: category, Code: code, SafeMessage: message, Retryable: retryable, Cause: err}
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return Metadata{}, err
	}
	metadata := Metadata{Provider: "openai", Model: p.config.Model, RequestID: response.Header.Get("x-request-id"), PromptVersion: PromptVersion, Latency: time.Since(started)}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		providerType, providerCode := sanitizedProviderError(responseBody)
		category, retryable, message := "INVALID_REQUEST", false, "OpenAI rejected the request"
		combined := strings.ToLower(providerType + " " + providerCode)
		switch {
		case response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden:
			category, message = "AUTHENTICATION", "OpenAI authentication failed"
		case response.StatusCode == http.StatusTooManyRequests:
			category, retryable, message = "RATE_LIMIT", true, "OpenAI rate limit was reached"
		case response.StatusCode >= 500:
			category, retryable, message = "OUTAGE", true, "OpenAI is temporarily unavailable"
		case strings.Contains(combined, "content_policy") || strings.Contains(combined, "moderation") || strings.Contains(combined, "safety"):
			category, message = "MODERATION", "OpenAI safety review rejected the request"
		}
		if providerCode == "" {
			providerCode = fmt.Sprintf("http_%d", response.StatusCode)
		}
		return metadata, &ProviderError{Category: category, Code: providerCode, SafeMessage: message, Retryable: retryable}
	}
	var envelope struct {
		ID     string `json:"id"`
		Output []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return metadata, &ProviderError{Category: "PROTOCOL", Code: "invalid_json", SafeMessage: "OpenAI returned an invalid response", Cause: err}
	}
	if metadata.RequestID == "" {
		metadata.RequestID = envelope.ID
	}
	metadata.InputTokens, metadata.OutputTokens = envelope.Usage.InputTokens, envelope.Usage.OutputTokens
	for _, item := range envelope.Output {
		for _, content := range item.Content {
			if content.Type == "output_text" && content.Text != "" {
				decoder := json.NewDecoder(strings.NewReader(content.Text))
				decoder.DisallowUnknownFields()
				if err := decoder.Decode(output); err != nil {
					return metadata, &ProviderError{Category: "VALIDATION", Code: "invalid_structured_output", SafeMessage: "OpenAI structured output failed validation", Cause: err}
				}
				return metadata, nil
			}
		}
	}
	return metadata, &ProviderError{Category: "PROTOCOL", Code: "missing_output_text", SafeMessage: "OpenAI response did not contain structured output"}
}

func sanitizedProviderError(value []byte) (string, string) {
	var problem struct {
		Error struct{ Type, Code, Message string } `json:"error"`
	}
	if json.Unmarshal(value, &problem) == nil {
		return problem.Error.Type, problem.Error.Code
	}
	return "", ""
}
