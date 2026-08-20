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
	Key           string
	SchemaVersion int
	Schema        *jsonschema.Schema
	SchemaHash    string
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
	return Pack{Key: key, SchemaVersion: version, Schema: schema, SchemaHash: hex.EncodeToString(digest[:])}, nil
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

func (r *Registry) Keys() []string {
	keys := make([]string, 0, len(r.packs))
	for key := range r.packs {
		keys = append(keys, key)
	}
	return keys
}
