package campaigns

import (
	"testing"

	"github.com/google/uuid"
)

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

func TestNewProgressMapsOnlyPersistedCompletion(t *testing.T) {
	campaignID := uuid.New()
	progress := newProgress(campaignID, []bool{true, false, true})

	if progress.CampaignID != campaignID || len(progress.Steps) != 9 {
		t.Fatalf("unexpected progress shape: %#v", progress)
	}
	if !progress.Steps[0].Completed || progress.Steps[1].Completed || !progress.Steps[2].Completed {
		t.Fatalf("persisted completion was not preserved: %#v", progress.Steps[:3])
	}
	if progress.Steps[3].Completed || progress.Steps[8].Completed || !progress.Steps[8].Optional {
		t.Fatalf("missing and optional steps must fail closed: %#v", progress.Steps[3:])
	}
}
