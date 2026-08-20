package ai

import "encoding/json"

func object(properties map[string]any, required ...string) map[string]any {
	return map[string]any{"type": "object", "additionalProperties": false, "properties": properties, "required": required}
}

func array(items any, minimum int, maximum int) map[string]any {
	return map[string]any{"type": "array", "items": items, "minItems": minimum, "maxItems": maximum}
}

var stringSchema = map[string]any{"type": "string"}

var conceptSchema = object(map[string]any{
	"concepts": array(object(map[string]any{
		"title": stringSchema, "hook": stringSchema, "coreMessage": stringSchema, "audienceFit": stringSchema,
		"videoFormat":    map[string]any{"type": "string", "enum": []string{"INTERVIEW_REVIEW", "PROBLEM_SOLUTION"}},
		"characterRoles": array(stringSchema, 2, 2), "environment": stringSchema, "productPlacement": stringSchema,
		"expectedSceneCount":         map[string]any{"type": "integer", "minimum": 2, "maximum": 8},
		"expectedSeedanceSeconds":    map[string]any{"type": "integer", "minimum": 6, "maximum": 90},
		"requiredRealProductFootage": array(stringSchema, 0, 12), "estimatedCostUsd": map[string]any{"type": "number", "minimum": 0},
		"generationRisks": array(stringSchema, 0, 12), "campaignFitReason": stringSchema,
	}, "title", "hook", "coreMessage", "audienceFit", "videoFormat", "characterRoles", "environment", "productPlacement", "expectedSceneCount", "expectedSeedanceSeconds", "requiredRealProductFootage", "estimatedCostUsd", "generationRisks", "campaignFitReason"), 2, 6),
}, "concepts")

var contentSchema = object(map[string]any{
	"variants": array(object(map[string]any{"key": stringSchema, "platform": stringSchema, "content": stringSchema}, "key", "platform", "content"), 14, 14),
}, "variants")

var scriptSchema = object(map[string]any{
	"hook": stringSchema, "introduction": stringSchema, "problem": stringSchema, "productSolution": stringSchema,
	"productFeatures": array(stringSchema, 0, 12), "benefits": array(stringSchema, 0, 12), "cta": stringSchema, "closing": stringSchema,
	"approximateDurationSeconds": map[string]any{"type": "integer", "enum": []int{30, 45}},
	"characterRoles":             object(map[string]any{"primary": stringSchema, "listener": stringSchema}, "primary", "listener"),
	"spokenLanguage":             map[string]any{"type": "string", "enum": []string{"vi", "en"}},
	"dialogueTurns":              array(object(map[string]any{"order": map[string]any{"type": "integer", "minimum": 1}, "characterRole": stringSchema, "dialogue": stringSchema, "estimatedDurationMs": map[string]any{"type": "integer", "minimum": 200}}, "order", "characterRole", "dialogue", "estimatedDurationMs"), 2, 30),
}, "hook", "introduction", "problem", "productSolution", "productFeatures", "benefits", "cta", "closing", "approximateDurationSeconds", "characterRoles", "spokenLanguage", "dialogueTurns")

var sceneSchema = object(map[string]any{
	"scenes": array(object(map[string]any{
		"sceneId": stringSchema, "order": map[string]any{"type": "integer", "minimum": 1}, "durationSeconds": map[string]any{"type": "integer", "minimum": 3, "maximum": 15},
		"generationMethod":   map[string]any{"type": "string", "enum": []string{"seedance", "product_footage", "still_image"}},
		"speakerCharacterId": map[string]any{"type": "string", "format": "uuid"}, "listenerCharacterId": map[string]any{"type": "string", "format": "uuid"},
		"dialogue": stringSchema, "speakerAction": stringSchema, "listenerAction": stringSchema, "camera": stringSchema, "environment": stringSchema, "productPlacement": stringSchema,
		"referenceAssetIds": array(map[string]any{"type": "string", "format": "uuid"}, 0, 12), "requiredProductFactIds": array(map[string]any{"type": "string", "format": "uuid"}, 0, 50),
		"expectedCostUsd": map[string]any{"type": "number", "minimum": 0}, "seedancePrompt": stringSchema,
	}, "sceneId", "order", "durationSeconds", "generationMethod", "speakerCharacterId", "listenerCharacterId", "dialogue", "speakerAction", "listenerAction", "camera", "environment", "productPlacement", "referenceAssetIds", "requiredProductFactIds", "expectedCostUsd", "seedancePrompt"), 2, 8),
}, "scenes")

var auditSchema = object(map[string]any{
	"pass": map[string]any{"type": "boolean"}, "duplicateProductReview": map[string]any{"type": "boolean"}, "productColorMismatch": map[string]any{"type": "boolean"},
	"blurOrLowQualityWarning": map[string]any{"type": "boolean"}, "cropWarning": map[string]any{"type": "boolean"}, "subtitleOverflow": map[string]any{"type": "boolean"},
	"logoOverlap": map[string]any{"type": "boolean"}, "ctaSafeZoneViolation": map[string]any{"type": "boolean"}, "findings": array(stringSchema, 0, 30),
}, "pass", "duplicateProductReview", "productColorMismatch", "blurOrLowQualityWarning", "cropWarning", "subtitleOverflow", "logoOverlap", "ctaSafeZoneViolation", "findings")

func schemaJSON(schema map[string]any) json.RawMessage {
	value, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return value
}
