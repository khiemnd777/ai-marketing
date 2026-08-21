package verticals

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const TravelLuggage = "travel-luggage"

type Pack struct {
	Key               string
	SchemaVersion     int
	Schema            *jsonschema.Schema
	SchemaHash        string
	AssetRequirements AssetRequirements
}

type AssetRequirements struct {
	Version            int      `json:"version"`
	Categories         []string `json:"categories"`
	MinimumForApproval []string `json:"minimumForApproval"`
}

type Registry struct {
	packs map[string]Pack
}

func Load(root string) (*Registry, error) {
	if root == "" {
		return nil, errors.New("verticals directory is required")
	}
	pack, err := loadPack(root, TravelLuggage, 1)
	if err != nil {
		return nil, err
	}
	return &Registry{packs: map[string]Pack{pack.Key: pack}}, nil
}

func loadPack(root, key string, version int) (Pack, error) {
	path := filepath.Join(root, key, "product-schema.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return Pack{}, fmt.Errorf("read vertical %s schema: %w", key, err)
	}
	var document any
	if err := json.Unmarshal(raw, &document); err != nil {
		return Pack{}, fmt.Errorf("decode vertical %s schema: %w", key, err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	const schemaURL = "https://studio.internal/verticals/travel-luggage/product-schema.json"
	if err := compiler.AddResource(schemaURL, document); err != nil {
		return Pack{}, fmt.Errorf("register vertical %s schema: %w", key, err)
	}
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return Pack{}, fmt.Errorf("compile vertical %s schema: %w", key, err)
	}
	digest := sha256.Sum256(raw)
	requirementsRaw, err := os.ReadFile(filepath.Join(root, key, "asset-requirements.json"))
	if err != nil {
		return Pack{}, fmt.Errorf("read vertical %s asset requirements: %w", key, err)
	}
	var requirements AssetRequirements
	if err := json.Unmarshal(requirementsRaw, &requirements); err != nil {
		return Pack{}, fmt.Errorf("decode vertical %s asset requirements: %w", key, err)
	}
	if err := validateAssetRequirements(requirements); err != nil {
		return Pack{}, fmt.Errorf("validate vertical %s asset requirements: %w", key, err)
	}
	return Pack{Key: key, SchemaVersion: version, Schema: schema, SchemaHash: hex.EncodeToString(digest[:]), AssetRequirements: requirements}, nil
}

func validateAssetRequirements(requirements AssetRequirements) error {
	if requirements.Version < 1 || len(requirements.Categories) == 0 || len(requirements.MinimumForApproval) == 0 {
		return errors.New("version, categories, and minimumForApproval are required")
	}
	categories := make(map[string]struct{}, len(requirements.Categories))
	for _, category := range requirements.Categories {
		if category == "" {
			return errors.New("asset category cannot be empty")
		}
		if _, exists := categories[category]; exists {
			return fmt.Errorf("duplicate asset category %q", category)
		}
		categories[category] = struct{}{}
	}
	minimum := make(map[string]struct{}, len(requirements.MinimumForApproval))
	for _, category := range requirements.MinimumForApproval {
		if _, exists := categories[category]; !exists {
			return fmt.Errorf("minimum asset category %q is not declared", category)
		}
		if _, exists := minimum[category]; exists {
			return fmt.Errorf("duplicate minimum asset category %q", category)
		}
		minimum[category] = struct{}{}
	}
	return nil
}

func (r *Registry) Validate(key string, value any) (Pack, error) {
	pack, ok := r.packs[key]
	if !ok {
		return Pack{}, fmt.Errorf("unsupported vertical %q", key)
	}
	if err := pack.Schema.Validate(value); err != nil {
		return Pack{}, fmt.Errorf("vertical data does not match %s schema: %w", key, err)
	}
	return pack, nil
}

func (r *Registry) Get(key string) (Pack, bool) {
	pack, ok := r.packs[key]
	return pack, ok
}

func (r *Registry) Keys() []string {
	keys := make([]string, 0, len(r.packs))
	for key := range r.packs {
		keys = append(keys, key)
	}
	return keys
}
