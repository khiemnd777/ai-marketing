package providerconfigs

import (
	"errors"
	"testing"
)

func TestBundleConfigurationKeepsR2InternalAndBrowserEndpointsSeparate(t *testing.T) {
	t.Parallel()
	bundle := Bundle{}
	err := bundleConfiguration(R2, map[string]any{
		"accountId":       "local-minio",
		"bucket":          "studio-media",
		"endpoint":        "http://minio:9000/",
		"browserEndpoint": "http://localhost:9100/",
		"publicBaseUrl":   "",
	}, map[string]string{
		"accessKeyId":     "test-access",
		"secretAccessKey": "test-secret",
	}, &bundle)
	if err != nil {
		t.Fatalf("bundleConfiguration() error = %v", err)
	}
	if bundle.R2.Endpoint != "http://minio:9000" || bundle.R2.BrowserEndpoint != "http://localhost:9100" {
		t.Fatalf("R2 endpoints = internal %q, browser %q", bundle.R2.Endpoint, bundle.R2.BrowserEndpoint)
	}
}

func TestBundleConfigurationRejectsInvalidR2BrowserEndpoint(t *testing.T) {
	t.Parallel()
	bundle := Bundle{}
	err := bundleConfiguration(R2, map[string]any{
		"bucket":          "studio-media",
		"endpoint":        "http://minio:9000",
		"browserEndpoint": "http://localhost:9100/upload?secret=value",
	}, map[string]string{}, &bundle)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("bundleConfiguration() error = %v, want ErrInvalid", err)
	}
}
