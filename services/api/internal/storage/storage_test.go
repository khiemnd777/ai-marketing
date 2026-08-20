package storage

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidateObjectKey(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{name: "scoped", key: "workspaces/a/assets/b/original.mp4", valid: true},
		{name: "parent traversal", key: "workspaces/a/../secret", valid: false},
		{name: "absolute", key: "/workspaces/a", valid: false},
		{name: "backslash", key: "workspaces\\a", valid: false},
		{name: "empty segment", key: "workspaces//a", valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateObjectKey(test.key)
			if test.valid && err != nil {
				t.Fatalf("validateObjectKey() error = %v", err)
			}
			if !test.valid && !errors.Is(err, ErrInvalidObjectKey) {
				t.Fatalf("validateObjectKey() error = %v, want ErrInvalidObjectKey", err)
			}
		})
	}
}

func TestScopedObjectKeyDoesNotKeepFilename(t *testing.T) {
	key, err := ScopedObjectKey(uuid.New(), uuid.New(), "customer passport scan.JPG")
	if err != nil {
		t.Fatalf("ScopedObjectKey() error = %v", err)
	}
	if strings.Contains(key, "passport") || !strings.HasSuffix(key, ".jpg") {
		t.Fatalf("ScopedObjectKey() = %q", key)
	}
}
