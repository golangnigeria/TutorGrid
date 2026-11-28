package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/tutorgrid/tutorgrid/internal/config"
)

// ObjectInfo describes object metadata returned by HeadObject.
type ObjectInfo struct {
	Exists      bool
	SizeBytes   int64
	ETag        string
	ContentType string
}

// ObjectStore abstracts Backblaze B2's S3-compatible API behind a narrow, testable boundary.
type ObjectStore interface {
	PresignPut(ctx context.Context, key, contentType string, contentLength int64, ttl time.Duration) (string, error)
	PresignGet(ctx context.Context, key, downloadFilename string, ttl time.Duration) (string, error)
	HeadObject(ctx context.Context, key string) (ObjectInfo, error)
	ReadRange(ctx context.Context, key string, start, endInclusive int64) ([]byte, error)
	DeleteObject(ctx context.Context, key string) error
	ObjectURL(key string) string
}

// B2Store implements ObjectStore using aws-sdk-go-v2/service/s3 targeting Backblaze B2 exclusively (ADR 0003).
type B2Store struct {
	client       *s3.Client
	presigner    *s3.PresignClient
	bucket       string
	endpoint     string
	usePathStyle bool
}

func NewB2Store(ctx context.Context, cfg config.StorageConfig) (*B2Store, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("storage: invalid config: %w", err)
	}

	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(strings.TrimSpace(cfg.Region)),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				strings.TrimSpace(cfg.AccessKeyID),
				strings.TrimSpace(cfg.SecretAccessKey),
				"",
			),
		),
		// Critical for Backblaze B2 browser presigned PUT compatibility:
		// prevents aws-sdk-go-v2 from injecting flexible CRC32 checksum headers rejected by B2.
		awsconfig.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
		awsconfig.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired),
	)
	if err != nil {
		return nil, fmt.Errorf("storage: build aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = cfg.UsePathStyle
	})

	return &B2Store{
		client:       client,
		presigner:    s3.NewPresignClient(client),
		bucket:       strings.TrimSpace(cfg.Bucket),
		endpoint:     endpoint,
		usePathStyle: cfg.UsePathStyle,
	}, nil
}

func (s *B2Store) PresignPut(
	ctx context.Context,
	key, contentType string,
	contentLength int64,
	ttl time.Duration,
) (string, error) {
	req, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(contentLength),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("storage: presign put: %w", err)
	}
	return req.URL, nil
}

// PresignGet returns a short-lived signed URL enforcing Content-Disposition: attachment
// so unscanned files are never rendered inline in the browser (ADR 0003).
func (s *B2Store) PresignGet(
	ctx context.Context,
	key, downloadFilename string,
	ttl time.Duration,
) (string, error) {
	safeDispName := sanitizeHeaderFilename(downloadFilename)
	disposition := fmt.Sprintf(`attachment; filename="%s"`, safeDispName)

	req, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(s.bucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String(disposition),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("storage: presign get: %w", err)
	}
	return req.URL, nil
}

func (s *B2Store) HeadObject(ctx context.Context, key string) (ObjectInfo, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundError(err) {
			return ObjectInfo{Exists: false}, nil
		}
		return ObjectInfo{}, fmt.Errorf("storage: head object: %w", err)
	}

	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return ObjectInfo{
		Exists:      true,
		SizeBytes:   size,
		ETag:        strings.Trim(aws.ToString(out.ETag), `"`),
		ContentType: aws.ToString(out.ContentType),
	}, nil
}

func (s *B2Store) ReadRange(ctx context.Context, key string, start, endInclusive int64) ([]byte, error) {
	if start < 0 || endInclusive < start {
		return nil, fmt.Errorf("storage: invalid byte range %d-%d", start, endInclusive)
	}
	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, endInclusive)
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Range:  aws.String(rangeHeader),
	})
	if err != nil {
		return nil, fmt.Errorf("storage: get range %s: %w", rangeHeader, err)
	}
	defer out.Body.Close()

	maxRead := (endInclusive - start) + 1
	data, err := io.ReadAll(io.LimitReader(out.Body, maxRead))
	if err != nil {
		return nil, fmt.Errorf("storage: read range body: %w", err)
	}
	return data, nil
}

// DeleteObject is idempotent: missing objects (NotFound / NoSuchKey) return nil.
func (s *B2Store) DeleteObject(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("storage: delete object: %w", err)
	}
	return nil
}

func (s *B2Store) ObjectURL(key string) string {
	trimmedKey := strings.TrimLeft(key, "/")
	if s.usePathStyle {
		return fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucket, trimmedKey)
	}
	return fmt.Sprintf("%s/%s", s.endpoint, trimmedKey)
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return true
	}
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}
	var respErr *awshttp.ResponseError
	if errors.As(err, &respErr) && respErr.HTTPStatusCode() == http.StatusNotFound {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		if code == "NotFound" || code == "NoSuchKey" || code == "404" {
			return true
		}
	}
	return false
}

func sanitizeHeaderFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r == '"' || r == '\\' || r == '\r' || r == '\n' || r < 32 || r == 127 {
			continue
		}
		b.WriteRune(r)
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "download"
	}
	return out
}
