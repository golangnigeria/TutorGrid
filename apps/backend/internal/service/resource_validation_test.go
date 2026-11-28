package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tutorgrid/tutorgrid/internal/lib/storage"
)

type bytesObjectStore struct {
	data map[string][]byte
}

func (b *bytesObjectStore) PresignPut(_ context.Context, _, _ string, _ int64, _ time.Duration) (string, error) {
	return "", nil
}
func (b *bytesObjectStore) PresignGet(_ context.Context, _, _ string, _ time.Duration) (string, error) {
	return "", nil
}
func (b *bytesObjectStore) HeadObject(_ context.Context, key string) (storage.ObjectInfo, error) {
	d, ok := b.data[key]
	if !ok {
		return storage.ObjectInfo{Exists: false}, nil
	}
	return storage.ObjectInfo{Exists: true, SizeBytes: int64(len(d))}, nil
}
func (b *bytesObjectStore) ReadRange(_ context.Context, key string, start, endInclusive int64) ([]byte, error) {
	d, ok := b.data[key]
	if !ok {
		return nil, errors.New("not found")
	}
	if endInclusive >= int64(len(d)) {
		endInclusive = int64(len(d)) - 1
	}
	return d[start : endInclusive+1], nil
}
func (b *bytesObjectStore) DeleteObject(_ context.Context, key string) error {
	delete(b.data, key)
	return nil
}
func (b *bytesObjectStore) ObjectURL(key string) string { return "https://example.test/" + key }

func buildZipBytes(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func TestResourceValidationMimeAndSize(t *testing.T) {
	for _, allowed := range []string{MimePNG, MimeJPEG, MimeWebP, MimePDF, MimePPT, MimePPTX} {
		if _, err := NormalizeAndValidateMimeType(allowed); err != nil {
			t.Fatalf("expected %s to be allowed, got %v", allowed, err)
		}
	}
	for _, blocked := range []string{"image/svg+xml", "text/html", "application/javascript", "application/x-msdownload", "application/zip"} {
		if _, err := NormalizeAndValidateMimeType(blocked); !errors.Is(err, ErrUnsupportedMimeType) {
			t.Fatalf("expected %s to be rejected with ErrUnsupportedMimeType, got %v", blocked, err)
		}
	}

	if err := ValidateDeclaredSize(0); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("expected 0 bytes rejected, got %v", err)
	}
	if err := ValidateDeclaredSize(MaxResourceSizeBytes + 1); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("expected >25MB rejected, got %v", err)
	}
	if err := ValidateDeclaredSize(MaxResourceSizeBytes); err != nil {
		t.Fatalf("expected exact 25MB allowed, got %v", err)
	}
}

func TestSanitizeFilenameAndObjectKey(t *testing.T) {
	safe, err := SanitizeFilename("../../etc/passwd\\..\\Lesson 1 <script>.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(safe, "..") || strings.Contains(safe, "/") || strings.Contains(safe, "\\") {
		t.Fatalf("unsafe filename produced: %q", safe)
	}

	tid := uuid.New()
	rid := uuid.New()
	key := BuildObjectKey(tid, rid, safe)
	expectedPrefix := "tenants/" + tid.String() + "/resources/" + rid.String() + "/"
	if !strings.HasPrefix(key, expectedPrefix) {
		t.Fatalf("expected tenant-prefixed key %q, got %q", expectedPrefix, key)
	}
}

func TestMagicBytesAndOOXMLPPTXValidation(t *testing.T) {
	cases := []struct {
		name  string
		mime  string
		bytes []byte
		valid bool
	}{
		{"valid png", MimePNG, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0x00}, true},
		{"valid jpeg", MimeJPEG, []byte{0xFF, 0xD8, 0xFF, 0xE0}, true},
		{"valid webp", MimeWebP, []byte("RIFF1234WEBPVP8 "), true},
		{"valid pdf", MimePDF, []byte("%PDF-1.7\n"), true},
		{"valid ppt", MimePPT, []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, true},
		{"spoofed html as pdf", MimePDF, []byte("<html><script>alert(1)</script></html>"), false},
		{"spoofed exe as png", MimePNG, []byte("MZ\x90\x00\x03\x00\x00\x00"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateMagicBytes(tc.mime, tc.bytes)
			if tc.valid && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
			if !tc.valid && err == nil {
				t.Fatalf("expected invalid magic bytes to fail")
			}
		})
	}

	// PPTX deep OOXML validation: valid PPTX passes; plain ZIP pretending to be PPTX fails!
	validPPTX := buildZipBytes(t, map[string]string{
		"[Content_Types].xml":  `<?xml version="1.0"?><Types/>`,
		"ppt/presentation.xml": `<?xml version="1.0"?><p:presentation/>`,
	})
	plainZip := buildZipBytes(t, map[string]string{
		"notes.txt": "hello not a presentation",
	})

	store := &bytesObjectStore{
		data: map[string][]byte{
			"valid.pptx": validPPTX,
			"fake.pptx":  plainZip,
		},
	}

	if err := ValidateMagicBytes(MimePPTX, validPPTX); err != nil {
		t.Fatalf("valid pptx magic bytes failed: %v", err)
	}
	if err := ValidatePPTXPackage(context.Background(), store, "valid.pptx", int64(len(validPPTX))); err != nil {
		t.Fatalf("expected valid OOXML PPTX to pass, got %v", err)
	}
	if err := ValidateMagicBytes(MimePPTX, plainZip); err != nil {
		t.Fatalf("plain zip has PK magic bytes, expected deep OOXML check to catch it")
	}
	if err := ValidatePPTXPackage(context.Background(), store, "fake.pptx", int64(len(plainZip))); !errors.Is(err, ErrInvalidFileContent) {
		t.Fatalf("expected plain ZIP disguised as PPTX to fail with ErrInvalidFileContent, got %v", err)
	}
}
