package main

import (
	"testing"

	"github.com/google/uuid"
)

func TestLoadOptions(t *testing.T) {
	t.Parallel()
	clientID := uuid.New()
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		wantErr bool
	}{
		{
			name: "accepts complete development configuration",
			args: []string{"configure-local-storage", clientID.String()},
			env: map[string]string{
				"APP_ENV":                         "development",
				"LOCAL_STORAGE_ENDPOINT":          "http://minio:9000/",
				"LOCAL_STORAGE_BROWSER_ENDPOINT":  "http://localhost:9100/",
				"LOCAL_STORAGE_BUCKET":            "media-test",
				"LOCAL_STORAGE_ACCESS_KEY_ID":     "test-access",
				"LOCAL_STORAGE_SECRET_ACCESS_KEY": "test-secret",
			},
		},
		{
			name:    "refuses production",
			args:    []string{"configure-local-storage", clientID.String()},
			env:     map[string]string{"APP_ENV": "production"},
			wantErr: true,
		},
		{
			name:    "requires all local storage values",
			args:    []string{"configure-local-storage", clientID.String()},
			env:     map[string]string{"APP_ENV": "development"},
			wantErr: true,
		},
		{
			name:    "requires a client UUID",
			args:    []string{"configure-local-storage", "not-a-uuid"},
			env:     map[string]string{"APP_ENV": "development"},
			wantErr: true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			getenv := func(name string) string { return test.env[name] }
			got, err := loadOptions(test.args, getenv)
			if (err != nil) != test.wantErr {
				t.Fatalf("loadOptions() error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && (got.clientID != clientID || got.endpoint != "http://minio:9000" || got.browserEndpoint != "http://localhost:9100") {
				t.Fatalf("loadOptions() = %#v", got)
			}
		})
	}
}
