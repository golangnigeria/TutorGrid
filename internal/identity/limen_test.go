package identity

import (
	"strings"
	"testing"
)

func TestParseSecretBytes(t *testing.T) {
	t.Run("accepts 64-char hex secret as 32 raw bytes", func(t *testing.T) {
		hexSecret := strings.Repeat("ab", 32)
		got, err := parseSecretBytes(hexSecret)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 32 {
			t.Fatalf("expected 32 bytes, got %d", len(got))
		}
	})

	t.Run("accepts raw 32-byte secret", func(t *testing.T) {
		rawSecret := strings.Repeat("x", 32)
		got, err := parseSecretBytes(rawSecret)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 32 {
			t.Fatalf("expected 32 bytes, got %d", len(got))
		}
	})

	t.Run("rejects invalid secret lengths", func(t *testing.T) {
		if _, err := parseSecretBytes("too-short"); err == nil {
			t.Fatal("expected error for short secret")
		}
		if _, err := parseSecretBytes(strings.Repeat("z", 64)); err == nil {
			t.Fatal("expected error for non-hex 64-char secret")
		}
	})
}
