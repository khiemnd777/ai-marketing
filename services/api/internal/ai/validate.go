package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/producttruth"
)

var variantKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,79}$`)

func ValidateConcepts(output ConceptOutput) error {
	if len(output.Concepts) < 2 || len(output.Concepts) > 6 {
		return errors.New("concept output requires 2-6 concepts")
	}
	formats := map[string]bool{}
	for _, concept := range output.Concepts {
		if strings.TrimSpace(concept.Title) == "" || strings.TrimSpace(concept.Hook) == "" || len(concept.CharacterRoles) != 2 || concept.CharacterRoles[0] == concept.CharacterRoles[1] || concept.ExpectedSceneCount < 2 || concept.ExpectedSceneCount > 8 || concept.ExpectedSeedanceSeconds < 6 || concept.EstimatedCostUSD < 0 {
			return errors.New("concept violates planning constraints")
		}
		if concept.VideoFormat != "INTERVIEW_REVIEW" && concept.VideoFormat != "PROBLEM_SOLUTION" {
			return errors.New("unsupported concept format")
		}
		formats[concept.VideoFormat] = true
	}
	if !formats["INTERVIEW_REVIEW"] || !formats["PROBLEM_SOLUTION"] {
		return errors.New("concept candidates must cover both locked formats")
	}
	return nil
}

func ValidateContent(output ContentOutput, context PlanningContext) error {
	required := []string{"facebook_caption", "instagram_caption", "reels_caption", "primary_ads_text", "headline", "description", "cta_variants", "hook_a", "hook_b", "hook_c", "hashtags", "first_comment", "thumbnail_text", "retargeting_message"}
	seen := map[string]bool{}
	all := strings.Builder{}
	for _, variant := range output.Variants {
		if !variantKeyPattern.MatchString(variant.Key) || seen[variant.Key] || strings.TrimSpace(variant.Content) == "" {
			return errors.New("content variant is invalid or duplicated")
		}
		seen[variant.Key] = true
		all.WriteString(variant.Content)
		all.WriteByte('\n')
	}
	for _, key := range required {
		if !seen[key] {
			return fmt.Errorf("missing required content variant %s", key)
		}
	}
	return validateTruth(all.String(), context)
}

func ValidateScript(output ScriptOutput, context PlanningContext) error {
	if output.ApproximateDurationSeconds != context.DurationSeconds || output.SpokenLanguage != context.Language || len(output.DialogueTurns) < 2 || output.CharacterRoles["primary"] == "" || output.CharacterRoles["listener"] == "" || output.CharacterRoles["primary"] == output.CharacterRoles["listener"] {
		return errors.New("script violates duration, language, or two-character rules")
	}
	all, lastOrder := output.Hook+"\n"+output.Introduction+"\n"+output.Problem+"\n"+output.ProductSolution+"\n"+output.CTA+"\n"+output.Closing, int32(0)
	var duration int64
	for _, turn := range output.DialogueTurns {
		if turn.Order <= lastOrder || turn.EstimatedDurationMS < 200 || strings.TrimSpace(turn.Dialogue) == "" {
			return errors.New("script dialogue order or duration is invalid")
		}
		lastOrder, duration = turn.Order, duration+turn.EstimatedDurationMS
		all += "\n" + turn.Dialogue
	}
	if duration > int64(context.DurationSeconds)*1200 {
		return errors.New("dialogue does not fit selected duration")
	}
	return validateTruth(all, context)
}

func ValidateScenes(output SceneOutput, input SceneInput) error {
	if len(output.Scenes) < 2 || len(output.Scenes) > 8 || input.SpeakerCharacterID == input.ListenerCharacterID {
		return errors.New("scene output or character selection is invalid")
	}
	var duration int32
	all := strings.Builder{}
	for index, scene := range output.Scenes {
		if scene.Order != int32(index+1) || scene.SceneID != fmt.Sprintf("scene-%02d", index+1) || scene.DurationSeconds < 3 || scene.DurationSeconds > 15 || scene.ExpectedCostUSD < 0 {
			return errors.New("scene order, duration, or cost is invalid")
		}
		if scene.GenerationMethod == "seedance" && (scene.SpeakerCharacterID != input.SpeakerCharacterID || scene.ListenerCharacterID != input.ListenerCharacterID || scene.SpeakerCharacterID == scene.ListenerCharacterID || !strings.Contains(strings.ToLower(scene.ListenerAction), "mouth closed")) {
			return errors.New("Seedance scene violates two-character speaker/listener guardrails")
		}
		duration += scene.DurationSeconds
		all.WriteString(scene.Dialogue)
		all.WriteByte('\n')
	}
	if duration != input.Context.DurationSeconds {
		return errors.New("scene durations must exactly match campaign duration")
	}
	if err := producttruth.ValidateNoUniversalAirlineClaim(all.String(), false); err != nil {
		return err
	}
	return nil
}

func validateTruth(text string, context PlanningContext) error {
	locked := []producttruth.LockedFact{}
	for _, fact := range context.ProductTruth {
		if fact.Locked {
			locked = append(locked, producttruth.LockedFact{Key: fact.Key, ExactValue: fact.ExactValue})
		}
	}
	if err := producttruth.ValidateLockedFacts(text, locked); err != nil {
		return err
	}
	return producttruth.ValidateNoUniversalAirlineClaim(text, false)
}

func Hash(value any) (string, []byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", nil, err
	}
	var canonical any
	if err := json.Unmarshal(encoded, &canonical); err != nil {
		return "", nil, err
	}
	canonical = normalize(canonical)
	encoded, err = json.Marshal(canonical)
	if err != nil {
		return "", nil, err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), encoded, nil
}

func normalize(value any) any {
	switch current := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(current))
		for key := range current {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		ordered := make(map[string]any, len(current))
		for _, key := range keys {
			ordered[key] = normalize(current[key])
		}
		return ordered
	case []any:
		for index := range current {
			current[index] = normalize(current[index])
		}
	}
	return value
}
