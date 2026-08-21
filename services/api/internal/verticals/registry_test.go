package verticals

import (
	"path/filepath"
	"testing"
)

func TestTravelLuggageSchema(t *testing.T) {
	registry, err := Load(filepath.Join("..", "..", "..", "..", "verticals"))
	if err != nil {
		t.Fatal(err)
	}
	valid := map[string]any{
		"luggageType": "CARRY_ON", "sizeInches": 20.0,
		"externalDimensions": map[string]any{"heightCm": 55.0, "widthCm": 36.0, "depthCm": 23.0},
		"emptyWeightKg":      2.9, "capacityLiters": 38.0, "shellMaterial": "Polycarbonate",
		"wheelType": "Spinner", "wheelCount": 4, "lockType": "TSA", "handleType": "Telescopic",
		"interiorCompartments": []any{"Divider"}, "expandable": true, "waterResistance": "Splash resistant",
		"warranty": "5 years", "availableColors": []any{"Black"},
	}
	if _, err := registry.Validate(TravelLuggage, valid); err != nil {
		t.Fatalf("expected valid luggage data: %v", err)
	}
	delete(valid, "externalDimensions")
	if _, err := registry.Validate(TravelLuggage, valid); err == nil {
		t.Fatal("expected missing dimensions to fail")
	}
	pack, ok := registry.Get(TravelLuggage)
	if !ok {
		t.Fatal("expected travel-luggage pack")
	}
	if len(pack.AssetRequirements.Categories) != 14 {
		t.Fatalf("expected 14 media categories, got %d", len(pack.AssetRequirements.Categories))
	}
	wantMinimum := []string{"HERO_IMAGE", "FRONT_VIEW", "SIDE_VIEW", "INTERIOR_VIEW", "PACKSHOT"}
	if len(pack.AssetRequirements.MinimumForApproval) != len(wantMinimum) {
		t.Fatalf("expected %d minimum categories, got %d", len(wantMinimum), len(pack.AssetRequirements.MinimumForApproval))
	}
	for index, category := range wantMinimum {
		if pack.AssetRequirements.MinimumForApproval[index] != category {
			t.Fatalf("minimum category %d: want %q, got %q", index, category, pack.AssetRequirements.MinimumForApproval[index])
		}
	}
}
