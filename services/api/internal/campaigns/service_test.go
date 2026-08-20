package campaigns

import "testing"

func TestValidateCampaign(t *testing.T) {
	input := Input{Name: "Launch", Objective: "awareness", TargetAudience: "Frequent travelers", Market: "Vietnam", Country: "vn", Language: "vi", SocialPlatformTargets: []string{"facebook", "reels"}, VideoFormat: "interview_review", DurationSeconds: 30, AspectRatio: "9:16", Tone: "Practical", CTA: "Tìm hiểu thêm"}
	input.BrandID[0], input.ProductID[0] = 1, 2
	if err := validate(&input, false); err != nil {
		t.Fatalf("valid campaign rejected: %v", err)
	}
	input.DurationSeconds = 31
	if err := validate(&input, false); err == nil {
		t.Fatal("unsupported duration accepted")
	}
}
