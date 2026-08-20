package media

import "testing"

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
