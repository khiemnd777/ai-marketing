package publishing

import (
	"testing"

	"github.com/google/uuid"
)

func TestPostHashIgnoresOptimisticLockVersion(t *testing.T) {
	input := Input{
		SocialAccountID: uuid.New(),
		MediaAssetID:    uuid.New(),
		Caption:         "Nội dung đã được duyệt",
		Version:         1,
	}
	want := postHash("FACEBOOK", input)

	input.Version = 9
	if got := postHash("FACEBOOK", input); got != want {
		t.Fatalf("post hash changed with row version: got %q, want %q", got, want)
	}

	input.Caption = "Nội dung đã thay đổi"
	if got := postHash("FACEBOOK", input); got == want {
		t.Fatal("post hash did not change with publishable content")
	}
}
