package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type DemoProvider struct{ model string }

func NewDemoProvider(model string) *DemoProvider { return &DemoProvider{model: model} }
func (p *DemoProvider) metadata(operation string) Metadata {
	return Metadata{Provider: "demo", Model: p.model, RequestID: "demo-" + operation, PromptVersion: PromptVersion}
}
func (p *DemoProvider) GenerateConcepts(_ context.Context, input ConceptInput) (ConceptOutput, Metadata, error) {
	name := input.Context.ProductName
	return ConceptOutput{Concepts: []ConceptCandidate{
		{Title: "Góc nhìn người dùng", Hook: "Một chi tiết nhỏ thay đổi cả chuyến đi", CoreMessage: name + " giải quyết những bất tiện thường gặp bằng thông tin đã được xác minh.", AudienceFit: input.Context.Audience, VideoFormat: "INTERVIEW_REVIEW", CharacterRoles: []string{"host", "traveler"}, Environment: "modern airport lounge", ProductPlacement: "product beside the traveler", ExpectedSceneCount: 4, ExpectedSeedanceSeconds: input.Context.DurationSeconds, RequiredRealProductFootage: []string{"packshot", "wheel close-up"}, EstimatedCostUSD: float64(input.Context.DurationSeconds) * 0.04, GenerationRisks: []string{"Keep listener mouth closed", "Use real product inserts for exact appearance"}, CampaignFitReason: "A conversational review matches the audience and keeps claims attributable."},
		{Title: "Từ rắc rối đến nhẹ nhàng", Hook: "Bắt đầu bằng một khoảnh khắc du lịch quen thuộc", CoreMessage: "Show the problem, then demonstrate how " + name + " helps without unsupported guarantees.", AudienceFit: input.Context.Audience, VideoFormat: "PROBLEM_SOLUTION", CharacterRoles: []string{"guide", "traveler"}, Environment: "airport check-in area", ProductPlacement: "product remains visible and is demonstrated with real inserts", ExpectedSceneCount: 4, ExpectedSeedanceSeconds: input.Context.DurationSeconds, RequiredRealProductFootage: []string{"pulling demo", "interior view"}, EstimatedCostUSD: float64(input.Context.DurationSeconds) * 0.04, GenerationRisks: []string{"Avoid universal airline claims", "Color must match reference"}, CampaignFitReason: "The structure makes the product benefit clear within a short vertical video."},
	}}, p.metadata("concepts"), nil
}
func (p *DemoProvider) GenerateContent(_ context.Context, input ContentInput) (ContentOutput, Metadata, error) {
	base := fmt.Sprintf("%s — %s. %s %s", input.Context.ProductName, input.Context.Tone, lockedFactText(input.Context.ProductTruth), input.Context.CTA)
	keys := []string{"facebook_caption", "instagram_caption", "reels_caption", "primary_ads_text", "headline", "description", "cta_variants", "hook_a", "hook_b", "hook_c", "hashtags", "first_comment", "thumbnail_text", "retargeting_message"}
	variants := make([]ContentVariant, 0, len(keys))
	for _, key := range keys {
		variants = append(variants, ContentVariant{Key: key, Platform: platformFor(key), Content: base + " [" + key + "]"})
	}
	return ContentOutput{Variants: variants}, p.metadata("content"), nil
}
func (p *DemoProvider) GenerateScript(_ context.Context, input ScriptInput) (ScriptOutput, Metadata, error) {
	turns := []DialogueTurn{{Order: 1, CharacterRole: "primary", Dialogue: "Bạn thường gặp khó khăn gì khi chuẩn bị hành lý?", EstimatedDurationMS: 4000}, {Order: 2, CharacterRole: "listener", Dialogue: "Mình cần mọi thứ gọn và dễ di chuyển.", EstimatedDurationMS: 3500}, {Order: 3, CharacterRole: "primary", Dialogue: "Hãy xem " + input.Context.ProductName + " và các thông tin đã được xác minh.", EstimatedDurationMS: 5000}, {Order: 4, CharacterRole: "listener", Dialogue: "Chi tiết thực tế giúp mình lựa chọn rõ ràng hơn.", EstimatedDurationMS: 4500}, {Order: 5, CharacterRole: "primary", Dialogue: input.Context.CTA, EstimatedDurationMS: 3000}}
	return ScriptOutput{Hook: "Một câu hỏi du lịch quen thuộc", Introduction: "Hai nhân vật trao đổi tự nhiên.", Problem: "Sắp xếp và di chuyển hành lý thiếu thuận tiện.", ProductSolution: input.Context.ProductName + " được giới thiệu dựa trên Product Truth: " + lockedFactText(input.Context.ProductTruth), ProductFeatures: []string{"Verified product details"}, Benefits: []string{"Clear decision support"}, CTA: input.Context.CTA, Closing: "Hiển thị CTA và sản phẩm thật.", ApproximateDurationSeconds: input.Context.DurationSeconds, CharacterRoles: map[string]string{"primary": "host", "listener": "traveler"}, SpokenLanguage: input.Context.Language, DialogueTurns: turns}, p.metadata("script"), nil
}
func (p *DemoProvider) GenerateScenes(_ context.Context, input SceneInput) (SceneOutput, Metadata, error) {
	count := 4
	if input.Context.DurationSeconds == 45 {
		count = 6
	}
	base, remainder := int(input.Context.DurationSeconds)/count, int(input.Context.DurationSeconds)%count
	scenes := make([]SceneDirection, 0, count)
	for index := range count {
		duration := base
		if index < remainder {
			duration++
		}
		turn := input.Script.DialogueTurns[index%len(input.Script.DialogueTurns)]
		scenes = append(scenes, SceneDirection{SceneID: fmt.Sprintf("scene-%02d", index+1), Order: int32(index + 1), DurationSeconds: int32(duration), GenerationMethod: "seedance", SpeakerCharacterID: input.SpeakerCharacterID, ListenerCharacterID: input.ListenerCharacterID, Dialogue: turn.Dialogue, SpeakerAction: "speaks naturally toward the listener", ListenerAction: "maintains eye contact, mouth closed, one subtle nod", Camera: "medium two-shot", Environment: "modern airport lounge", ProductPlacement: "product visible beside the listener", ReferenceAssetIDs: []uuid.UUID{}, RequiredProductFactIDs: []uuid.UUID{}, ExpectedCostUSD: float64(duration) * 0.04, SeedancePrompt: "Two-character conversational scene; one speaker; listener mouth closed; product appearance comes from supplied references."})
	}
	return SceneOutput{Scenes: scenes}, p.metadata("scenes"), nil
}
func (p *DemoProvider) AuditContent(_ context.Context, _ AuditInput) (AuditOutput, Metadata, error) {
	return AuditOutput{Pass: true, Findings: []string{}}, p.metadata("audit"), nil
}
func platformFor(key string) string {
	switch key {
	case "instagram_caption", "reels_caption", "hashtags":
		return "INSTAGRAM"
	case "primary_ads_text", "headline", "description", "retargeting_message":
		return "META_ADS"
	default:
		return "FACEBOOK"
	}
}

func lockedFactText(facts []ProductFact) string {
	values := []string{}
	for _, fact := range facts {
		if fact.Locked {
			values = append(values, fact.ExactValue)
		}
	}
	return strings.Join(values, "; ")
}
