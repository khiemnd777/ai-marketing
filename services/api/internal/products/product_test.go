package products

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuildMediaReadinessRequiresApprovedAssetInEveryMinimumCategory(t *testing.T) {
	productID := uuid.New()
	minimum := []string{"HERO_IMAGE", "FRONT_VIEW", "PACKSHOT"}
	readiness := buildMediaReadiness(productID, "travel-luggage", minimum, map[string]mediaCounts{
		"HERO_IMAGE": {total: 2, approved: 1},
		"FRONT_VIEW": {total: 1, approved: 0},
		"PACKSHOT":   {total: 1, approved: 1},
	})
	if readiness.Ready {
		t.Fatal("expected readiness to fail when a minimum category has no approved asset")
	}
	if readiness.ProductID != productID || len(readiness.Requirements) != 3 {
		t.Fatalf("unexpected readiness result: %#v", readiness)
	}
	if readiness.Requirements[1].Ready || readiness.Requirements[1].TotalAssets != 1 {
		t.Fatalf("expected FRONT_VIEW to report one draft asset: %#v", readiness.Requirements[1])
	}

	readiness = buildMediaReadiness(productID, "travel-luggage", minimum, map[string]mediaCounts{
		"HERO_IMAGE": {total: 2, approved: 1},
		"FRONT_VIEW": {total: 1, approved: 1},
		"PACKSHOT":   {total: 1, approved: 1},
	})
	if !readiness.Ready {
		t.Fatal("expected all minimum categories to be ready")
	}
}
