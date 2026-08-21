package brands

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateBrandLogoLimit(t *testing.T) {
	input := Input{Name: "Northstar", PrimaryLanguage: "vi", LogoAssetIDs: make([]uuid.UUID, 21)}
	for index := range input.LogoAssetIDs {
		input.LogoAssetIDs[index] = uuid.New()
	}
	if err := validate(&input, false); err == nil {
		t.Fatal("expected more than 20 logo assets to fail")
	}
}

func TestSameUUIDsPreservesPrimaryLogoOrder(t *testing.T) {
	primary, alternate := uuid.New(), uuid.New()
	if !sameUUIDs([]uuid.UUID{primary, alternate}, []uuid.UUID{primary, alternate}) {
		t.Fatal("expected identical ordered logo selection to match")
	}
	if sameUUIDs([]uuid.UUID{primary, alternate}, []uuid.UUID{alternate, primary}) {
		t.Fatal("expected changing the primary logo order to be meaningful")
	}
}
