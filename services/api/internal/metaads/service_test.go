package metaads

import (
	"testing"

	"github.com/google/uuid"
)

func TestCampaignHashIgnoresOptimisticLockVersion(t *testing.T) {
	dailyBudget := int64(100_000)
	input := CampaignInput{
		MetaAdAccountID:       uuid.New(),
		SocialAccountID:       uuid.New(),
		Name:                  "Launch",
		Objective:             "OUTCOME_TRAFFIC",
		DailyBudgetMinor:      &dailyBudget,
		CampaignSpendCapMinor: 1_000_000,
		Currency:              "VND",
		DestinationURL:        "https://example.com/products/carry-on",
		Creative: CreativeInput{
			MediaAssetID:        uuid.New(),
			PrimaryTextVariants: []string{"Bản nháp đã duyệt"},
			HeadlineVariants:    []string{"Carry-on mới"},
			CTAVariants:         []string{"LEARN_MORE"},
		},
		Version: 1,
	}
	want := campaignHash(input)

	input.Version = 4
	if got := campaignHash(input); got != want {
		t.Fatalf("campaign hash changed with row version: got %q, want %q", got, want)
	}

	input.Creative.PrimaryTextVariants = []string{"Nội dung đã thay đổi"}
	if got := campaignHash(input); got == want {
		t.Fatal("campaign hash did not change with provider-facing content")
	}
}
