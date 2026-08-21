package media

import (
	"encoding/json"
	"testing"
)

func TestUploadMetadataJSONAlwaysProducesObject(t *testing.T) {
	t.Parallel()
	for _, metadata := range []map[string]string{nil, {}, {"source": "upload"}} {
		encoded := uploadMetadataJSON(metadata)
		var decoded any
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("json.Unmarshal(%s) error = %v", encoded, err)
		}
		if _, ok := decoded.(map[string]any); !ok {
			t.Fatalf("uploadMetadataJSON(%#v) = %s, want JSON object", metadata, encoded)
		}
	}
}

func TestValidateUpload(t *testing.T) {
	valid := UploadInput{AssetType: "VIDEO", Name: "Wheel demo", Filename: "wheel.mp4", MimeType: "video/mp4", SizeBytes: 20 << 20, UsageRights: "Owned by client", Tags: []string{"wheel-demo"}}
	if _, err := validateUpload(&valid); err != nil {
		t.Fatalf("expected valid video: %v", err)
	}
	spoofed := valid
	spoofed.Filename = "wheel.jpg"
	if _, err := validateUpload(&spoofed); err == nil {
		t.Fatal("expected extension/MIME mismatch to fail")
	}
	oversized := valid
	oversized.SizeBytes = (2 << 30) + 1
	if _, err := validateUpload(&oversized); err == nil {
		t.Fatal("expected oversized video to fail")
	}
}

func TestValidateUpdateNormalizesAndRejectsDuplicateTags(t *testing.T) {
	valid := UpdateInput{Name: "  Hero image  ", Category: " HERO_IMAGE ", Folder: " /campaign/summer/ ", UsageRights: " Client owned ", Tags: []string{"hero", "summer"}, Version: 2}
	if err := validateUpdate(&valid); err != nil {
		t.Fatalf("expected valid update: %v", err)
	}
	if valid.Name != "Hero image" || valid.Category != "HERO_IMAGE" || valid.Folder != "campaign/summer" || valid.UsageRights != "Client owned" {
		t.Fatalf("expected normalized metadata, got %#v", valid)
	}
	duplicate := valid
	duplicate.Tags = []string{"hero", "hero"}
	if err := validateUpdate(&duplicate); err == nil {
		t.Fatal("expected duplicate tags to fail")
	}
}
