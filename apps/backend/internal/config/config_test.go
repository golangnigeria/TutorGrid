package config

import "testing"

func TestStorageConfigValidate(t *testing.T) {
	validBase := StorageConfig{
		Endpoint:        "https://s3.us-west-004.backblazeb2.com",
		Region:          "us-west-004",
		Bucket:          "tutorgrid-dev",
		AccessKeyID:     "004testkeyid",
		SecretAccessKey: "K004testsecret",
		UsePathStyle:    true,
	}

	if err := validBase.Validate(); err != nil {
		t.Fatalf("expected valid Backblaze B2 config to pass, got error: %v", err)
	}

	disallowed := []struct {
		name     string
		endpoint string
	}{
		{"http scheme", "http://s3.us-west-004.backblazeb2.com"},
		{"localhost", "https://localhost:9000"},
		{"loopback ipv4", "https://127.0.0.1:9000"},
		{"loopback ipv6", "https://[::1]:9000"},
		{"minio host", "https://minio:9000"},
		{"minio subdomain", "https://minio.internal.example.com"},
		{"dot local", "https://storage.local"},
		{"empty", ""},
	}

	for _, tc := range disallowed {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validBase
			cfg.Endpoint = tc.endpoint
			if err := cfg.Validate(); err == nil {
				t.Fatalf("expected endpoint %q to be rejected", tc.endpoint)
			}
		})
	}
}
