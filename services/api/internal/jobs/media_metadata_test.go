package jobs

import (
	"bytes"
	"slices"
	"testing"
)

func TestCopyAndChecksum(t *testing.T) {
	t.Parallel()
	var destination bytes.Buffer
	checksum, err := copyAndChecksum(&destination, bytes.NewBufferString("studio-media"))
	if err != nil {
		t.Fatalf("copyAndChecksum() error = %v", err)
	}
	if destination.String() != "studio-media" {
		t.Fatalf("copied value = %q", destination.String())
	}
	if checksum != "6ddbbff30267ae7d2abbcf34e64c0a0e8bcad0152827625191e46d0bb20fae03" {
		t.Fatalf("checksum = %q", checksum)
	}
}

func TestThumbnailArgsOnlySeekForVideo(t *testing.T) {
	t.Parallel()
	imageArgs := thumbnailArgs("input.png", "output.jpg", false)
	if slices.Contains(imageArgs, "-ss") {
		t.Fatalf("image thumbnail args unexpectedly seek: %v", imageArgs)
	}
	videoArgs := thumbnailArgs("input.mp4", "output.jpg", true)
	seek := slices.Index(videoArgs, "-ss")
	input := slices.Index(videoArgs, "-i")
	if seek < 0 || input < 0 || seek > input {
		t.Fatalf("video thumbnail args must seek before input: %v", videoArgs)
	}
}
