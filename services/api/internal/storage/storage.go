package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

var ErrInvalidObjectKey = errors.New("invalid object key")

type PresignedRequest struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expiresAt"`
}

type UploadedPart struct {
	PartNumber int32  `json:"partNumber"`
	ETag       string `json:"etag"`
}

type ObjectMetadata struct {
	ContentType   string
	ContentLength int64
	ETag          string
	LastModified  time.Time
	Metadata      map[string]string
}

type ObjectStore interface {
	PresignPut(context.Context, string, string, int64, time.Duration) (PresignedRequest, error)
	PresignGet(context.Context, string, time.Duration) (PresignedRequest, error)
	CreateMultipartUpload(context.Context, string, string, map[string]string) (string, error)
	PresignUploadPart(context.Context, string, string, int32, time.Duration) (PresignedRequest, error)
	CompleteMultipartUpload(context.Context, string, string, []UploadedPart) error
	AbortMultipartUpload(context.Context, string, string) error
	Head(context.Context, string) (ObjectMetadata, error)
	Get(context.Context, string) (io.ReadCloser, error)
	Put(context.Context, string, string, int64, io.Reader, map[string]string) error
}

type S3Store struct {
	bucket    string
	client    *s3.Client
	presigner *s3.PresignClient
}

func NewS3Store(ctx context.Context, cfg config.R2Config) (*S3Store, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	endpoint, err := parseEndpoint(cfg.Endpoint)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return nil, errors.New("R2_ENDPOINT must be an absolute URL")
	}
	browserEndpoint := endpoint
	if cfg.BrowserEndpoint != "" {
		browserEndpoint, err = parseEndpoint(cfg.BrowserEndpoint)
		if err != nil || browserEndpoint.Scheme == "" || browserEndpoint.Host == "" {
			return nil, errors.New("R2_BROWSER_ENDPOINT must be an absolute URL")
		}
	}
	awsConfig, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load R2 S3 configuration: %w", err)
	}
	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(cfg.Endpoint)
		options.UsePathStyle = usePathStyle(endpoint)
	})
	presignClient := client
	if cfg.BrowserEndpoint != "" {
		presignClient = s3.NewFromConfig(awsConfig, func(options *s3.Options) {
			options.BaseEndpoint = aws.String(cfg.BrowserEndpoint)
			options.UsePathStyle = usePathStyle(browserEndpoint)
		})
	}
	return &S3Store{bucket: cfg.Bucket, client: client, presigner: s3.NewPresignClient(presignClient)}, nil
}

func parseEndpoint(raw string) (*url.URL, error) {
	endpoint, err := url.Parse(raw)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, errors.New("object storage endpoint must be an absolute URL without credentials, query, or fragment")
	}
	return endpoint, nil
}

func usePathStyle(endpoint *url.URL) bool {
	host := endpoint.Hostname()
	return host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasPrefix(host, "127.") || host == "::1" || host == "minio"
}

func (s *S3Store) PresignPut(ctx context.Context, key, contentType string, contentLength int64, expires time.Duration) (PresignedRequest, error) {
	if err := validateObjectKey(key); err != nil {
		return PresignedRequest{}, err
	}
	if contentLength <= 0 {
		return PresignedRequest{}, errors.New("content length must be positive")
	}
	result, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), ContentType: aws.String(contentType), ContentLength: aws.Int64(contentLength),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return PresignedRequest{}, fmt.Errorf("presign R2 put: %w", err)
	}
	return presigned(result.URL, result.Method, result.SignedHeader, expires), nil
}

func (s *S3Store) PresignGet(ctx context.Context, key string, expires time.Duration) (PresignedRequest, error) {
	if err := validateObjectKey(key); err != nil {
		return PresignedRequest{}, err
	}
	result, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)}, s3.WithPresignExpires(expires))
	if err != nil {
		return PresignedRequest{}, fmt.Errorf("presign R2 get: %w", err)
	}
	return presigned(result.URL, result.Method, result.SignedHeader, expires), nil
}

func (s *S3Store) CreateMultipartUpload(ctx context.Context, key, contentType string, metadata map[string]string) (string, error) {
	if err := validateObjectKey(key); err != nil {
		return "", err
	}
	result, err := s.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{Bucket: aws.String(s.bucket), Key: aws.String(key), ContentType: aws.String(contentType), Metadata: metadata})
	if err != nil {
		return "", fmt.Errorf("create R2 multipart upload: %w", err)
	}
	if result.UploadId == nil || *result.UploadId == "" {
		return "", errors.New("R2 returned an empty multipart upload ID")
	}
	return *result.UploadId, nil
}

func (s *S3Store) PresignUploadPart(ctx context.Context, key, uploadID string, partNumber int32, expires time.Duration) (PresignedRequest, error) {
	if err := validateObjectKey(key); err != nil {
		return PresignedRequest{}, err
	}
	if uploadID == "" || partNumber < 1 || partNumber > 10_000 {
		return PresignedRequest{}, errors.New("invalid multipart upload ID or part number")
	}
	result, err := s.presigner.PresignUploadPart(ctx, &s3.UploadPartInput{Bucket: aws.String(s.bucket), Key: aws.String(key), UploadId: aws.String(uploadID), PartNumber: aws.Int32(partNumber)}, s3.WithPresignExpires(expires))
	if err != nil {
		return PresignedRequest{}, fmt.Errorf("presign R2 upload part: %w", err)
	}
	return presigned(result.URL, result.Method, result.SignedHeader, expires), nil
}

func (s *S3Store) CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []UploadedPart) error {
	if err := validateObjectKey(key); err != nil {
		return err
	}
	if len(parts) == 0 || len(parts) > 10_000 {
		return errors.New("multipart completion requires between 1 and 10000 parts")
	}
	completed := make([]types.CompletedPart, 0, len(parts))
	for _, part := range parts {
		if part.PartNumber < 1 || part.ETag == "" {
			return errors.New("multipart part number and ETag are required")
		}
		completed = append(completed, types.CompletedPart{PartNumber: aws.Int32(part.PartNumber), ETag: aws.String(part.ETag)})
	}
	_, err := s.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{Parts: completed},
	})
	if err != nil {
		return fmt.Errorf("complete R2 multipart upload: %w", err)
	}
	return nil
}

func (s *S3Store) AbortMultipartUpload(ctx context.Context, key, uploadID string) error {
	if err := validateObjectKey(key); err != nil {
		return err
	}
	_, err := s.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{Bucket: aws.String(s.bucket), Key: aws.String(key), UploadId: aws.String(uploadID)})
	if err != nil {
		return fmt.Errorf("abort R2 multipart upload: %w", err)
	}
	return nil
}

func (s *S3Store) Head(ctx context.Context, key string) (ObjectMetadata, error) {
	if err := validateObjectKey(key); err != nil {
		return ObjectMetadata{}, err
	}
	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return ObjectMetadata{}, fmt.Errorf("inspect R2 object: %w", err)
	}
	metadata := ObjectMetadata{Metadata: result.Metadata}
	if metadata.Metadata == nil {
		metadata.Metadata = map[string]string{}
	}
	if result.ContentType != nil {
		metadata.ContentType = *result.ContentType
	}
	if result.ContentLength != nil {
		metadata.ContentLength = *result.ContentLength
	}
	if result.ETag != nil {
		metadata.ETag = *result.ETag
	}
	if result.LastModified != nil {
		metadata.LastModified = *result.LastModified
	}
	return metadata, nil
}

func (s *S3Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := validateObjectKey(key); err != nil {
		return nil, err
	}
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, fmt.Errorf("download R2 object: %w", err)
	}
	return result.Body, nil
}

func (s *S3Store) Put(ctx context.Context, key, contentType string, contentLength int64, body io.Reader, metadata map[string]string) error {
	if err := validateObjectKey(key); err != nil {
		return err
	}
	if contentLength <= 0 || body == nil {
		return errors.New("object body and positive content length are required")
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), ContentType: aws.String(contentType),
		ContentLength: aws.Int64(contentLength), Body: body, Metadata: metadata,
	})
	if err != nil {
		return fmt.Errorf("upload R2 object: %w", err)
	}
	return nil
}

func ScopedObjectKey(workspaceID, assetID uuid.UUID, filename string) (string, error) {
	extension := path.Ext(strings.ToLower(filename))
	if len(extension) > 12 || strings.ContainsAny(extension, "\\/\x00") {
		return "", ErrInvalidObjectKey
	}
	return fmt.Sprintf("workspaces/%s/assets/%s/original%s", workspaceID, assetID, extension), nil
}

func ThumbnailObjectKey(workspaceID, assetID uuid.UUID) string {
	return fmt.Sprintf("workspaces/%s/assets/%s/thumbnail.jpg", workspaceID, assetID)
}

func validateObjectKey(key string) error {
	if key == "" || len(key) > 1024 || strings.HasPrefix(key, "/") || strings.Contains(key, "\\") || strings.ContainsRune(key, '\x00') {
		return ErrInvalidObjectKey
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return ErrInvalidObjectKey
		}
	}
	if path.Clean(key) != key {
		return ErrInvalidObjectKey
	}
	return nil
}

func presigned(rawURL, method string, signedHeaders map[string][]string, expires time.Duration) PresignedRequest {
	headers := make(map[string]string, len(signedHeaders))
	for key, values := range signedHeaders {
		headers[key] = strings.Join(values, ",")
	}
	return PresignedRequest{URL: rawURL, Method: method, Headers: headers, ExpiresAt: time.Now().UTC().Add(expires)}
}
