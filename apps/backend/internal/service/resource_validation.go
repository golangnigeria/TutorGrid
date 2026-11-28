package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"github.com/tutorgrid/tutorgrid/internal/lib/storage"
)

const (
	MaxResourceSizeBytes int64 = 25 * 1024 * 1024 // 25 MiB MVP policy
	maxPPTXZipEntries          = 2000
	maxFilenameRunes           = 120

	MimePNG  = "image/png"
	MimeJPEG = "image/jpeg"
	MimeWebP = "image/webp"
	MimePDF  = "application/pdf"
	MimePPT  = "application/vnd.ms-powerpoint"
	MimePPTX = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
)

var (
	ErrInvalidFilename     = errors.New("resource: valid filename is required")
	ErrUnsupportedMimeType = errors.New("resource: unsupported content type (allowed: PNG, JPEG, WebP, PDF, PPT, PPTX)")
	ErrFileTooLarge        = errors.New("resource: file size must be between 1 byte and 25 MB")
	ErrInvalidFileContent  = errors.New("resource: uploaded file content does not match declared content type")
)

var allowedMimeTypes = map[string]bool{
	MimePNG:  true,
	MimeJPEG: true,
	MimeWebP: true,
	MimePDF:  true,
	MimePPT:  true,
	MimePPTX: true,
}

func NormalizeAndValidateMimeType(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if idx := strings.Index(normalized, ";"); idx != -1 {
		normalized = strings.TrimSpace(normalized[:idx])
	}
	if !allowedMimeTypes[normalized] {
		return "", ErrUnsupportedMimeType
	}
	return normalized, nil
}

func ValidateDeclaredSize(sizeBytes int64) error {
	if sizeBytes <= 0 || sizeBytes > MaxResourceSizeBytes {
		return ErrFileTooLarge
	}
	return nil
}

// SanitizeFilename removes directory traversal segments, path separators, control chars,
// and bounds length so filenames are safe for display, headers, and S3/B2 keys.
func SanitizeFilename(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrInvalidFilename
	}

	// Replace backslashes before taking filepath.Base so Windows-style paths are stripped on Linux too.
	normalized := strings.ReplaceAll(trimmed, "\\", "/")
	base := filepath.Base(normalized)
	if base == "." || base == "/" || base == ".." {
		return "", ErrInvalidFilename
	}

	var b strings.Builder
	for _, r := range base {
		if unicode.IsControl(r) || r == '/' || r == '\\' || r == '"' || r == '\'' || r == '\x00' {
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' || r == ' ' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}

	cleaned := strings.TrimSpace(b.String())
	cleaned = strings.TrimLeft(cleaned, ".")
	for strings.Contains(cleaned, "..") {
		cleaned = strings.ReplaceAll(cleaned, "..", ".")
	}
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if cleaned == "" {
		return "", ErrInvalidFilename
	}

	runes := []rune(cleaned)
	if len(runes) > maxFilenameRunes {
		ext := filepath.Ext(cleaned)
		extRunes := []rune(ext)
		if len(extRunes) > 0 && len(extRunes) < 20 {
			stemMax := maxFilenameRunes - len(extRunes)
			cleaned = string(runes[:stemMax]) + ext
		} else {
			cleaned = string(runes[:maxFilenameRunes])
		}
	}

	return cleaned, nil
}

// BuildObjectKey generates a server-owned, tenant-prefixed Backblaze B2 object key.
// Never accepts or trusts client-supplied object keys.
func BuildObjectKey(tenantID, resourceID uuid.UUID, safeFilename string) string {
	keyName := strings.ReplaceAll(safeFilename, " ", "_")
	return fmt.Sprintf("tenants/%s/resources/%s/%s", tenantID.String(), resourceID.String(), keyName)
}

// ValidateMagicBytes verifies the leading bytes of the uploaded object match the declared MIME type.
func ValidateMagicBytes(contentType string, header []byte) error {
	switch contentType {
	case MimePNG:
		if len(header) >= 8 && bytes.Equal(header[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
			return nil
		}
	case MimeJPEG:
		if len(header) >= 3 && header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF {
			return nil
		}
	case MimeWebP:
		if len(header) >= 12 && bytes.Equal(header[:4], []byte("RIFF")) && bytes.Equal(header[8:12], []byte("WEBP")) {
			return nil
		}
	case MimePDF:
		if len(header) >= 5 && bytes.Equal(header[:5], []byte("%PDF-")) {
			return nil
		}
	case MimePPT:
		// Legacy PowerPoint binary format (OLE2 Compound File Binary)
		if len(header) >= 8 && bytes.Equal(header[:8], []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}) {
			return nil
		}
	case MimePPTX:
		// OOXML ZIP local file header; full package verification runs in ValidatePPTXPackage.
		if len(header) >= 4 && header[0] == 'P' && header[1] == 'K' &&
			((header[2] == 0x03 && header[3] == 0x04) || (header[2] == 0x05 && header[3] == 0x06) || (header[2] == 0x07 && header[3] == 0x08)) {
			return nil
		}
	default:
		return ErrUnsupportedMimeType
	}
	return ErrInvalidFileContent
}

// storeRangeReaderAt adapts ObjectStore.ReadRange to io.ReaderAt so archive/zip can
// inspect the ZIP Central Directory via small tail range reads without downloading 25 MB.
type storeRangeReaderAt struct {
	ctx   context.Context
	store storage.ObjectStore
	key   string
	size  int64
}

func (r *storeRangeReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= r.size {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	end := off + int64(len(p)) - 1
	if end >= r.size {
		end = r.size - 1
	}
	data, err := r.store.ReadRange(r.ctx, r.key, off, end)
	if err != nil {
		return 0, err
	}
	n := copy(p, data)
	if off+int64(n) >= r.size && n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// ValidatePPTXPackage verifies that an uploaded PPTX is not merely an arbitrary ZIP archive:
// it parses the ZIP Central Directory with a bounded entry cap and requires both
// [Content_Types].xml and ppt/presentation.xml per the OOXML PresentationML specification.
func ValidatePPTXPackage(ctx context.Context, store storage.ObjectStore, key string, actualSize int64) error {
	if actualSize <= 0 || actualSize > MaxResourceSizeBytes {
		return ErrFileTooLarge
	}
	ra := &storeRangeReaderAt{
		ctx:   ctx,
		store: store,
		key:   key,
		size:  actualSize,
	}
	zr, err := zip.NewReader(ra, actualSize)
	if err != nil {
		return ErrInvalidFileContent
	}
	if len(zr.File) == 0 || len(zr.File) > maxPPTXZipEntries {
		return ErrInvalidFileContent
	}

	var hasContentTypes, hasPresentation bool
	for _, f := range zr.File {
		switch f.Name {
		case "[Content_Types].xml":
			hasContentTypes = true
		case "ppt/presentation.xml":
			hasPresentation = true
		}
		if hasContentTypes && hasPresentation {
			return nil
		}
	}
	return ErrInvalidFileContent
}
