package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type Service struct {
	repo *Repository
	r2   *R2Client
}

func NewService(repo *Repository, r2 *R2Client) *Service {
	return &Service{repo: repo, r2: r2}
}

func (s *Service) GetUploadURL(ctx context.Context, projectID, userID, bucketName, path, contentType string) (*UploadURLResponse, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}

	bucket, err := s.repo.GetBucket(ctx, projectID, bucketName)
	if err != nil {
		return nil, ErrBucketNotFound
	}

	key := s.r2.StorageKey(projectID, bucket.Name, path)
	uploadURL, err := s.r2.GenerateUploadURL(ctx, key, contentType)
	if err != nil {
		return nil, fmt.Errorf("generate upload url: %w", err)
	}

	return &UploadURLResponse{
		UploadURL: uploadURL,
		FileKey:   key,
	}, nil
}

func (s *Service) ConfirmUpload(ctx context.Context, projectID, userID, bucketName, path, contentType, etag string, size int64) (*Object, error) {
	bucket, err := s.repo.GetBucket(ctx, projectID, bucketName)
	if err != nil {
		return nil, ErrBucketNotFound
	}

	key := s.r2.StorageKey(projectID, bucket.Name, path)

	// Verify file actually exists in R2
	if !s.r2.ObjectExists(ctx, key) {
		return nil, errors.New("file not found in storage — upload may have failed")
	}

	var uid *string
	if userID != "" {
		uid = &userID
	}

	obj, err := s.repo.InsertObject(ctx, projectID, bucket.ID, uid, path, size, contentType, etag)
	if err != nil {
		return nil, fmt.Errorf("insert object metadata: %w", err)
	}

	return obj, nil
}

func (s *Service) GetDownloadURL(ctx context.Context, projectID, bucketName, path string) (*DownloadURLResponse, error) {
	bucket, err := s.repo.GetBucket(ctx, projectID, bucketName)
	if err != nil {
		return nil, ErrBucketNotFound
	}

	key := s.r2.StorageKey(projectID, bucket.Name, path)
	url, err := s.r2.GenerateDownloadURL(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("generate download url: %w", err)
	}

	return &DownloadURLResponse{URL: url}, nil
}

func (s *Service) DeleteObject(ctx context.Context, projectID, bucketName, path string) error {
	bucket, err := s.repo.GetBucket(ctx, projectID, bucketName)
	if err != nil {
		return ErrBucketNotFound
	}

	key := s.r2.StorageKey(projectID, bucket.Name, path)

	// Delete from R2
	if err := s.r2.DeleteObject(ctx, key); err != nil {
		return fmt.Errorf("delete from storage: %w", err)
	}

	// Delete metadata
	if err := s.repo.DeleteObject(ctx, bucket.ID, path); err != nil && !errors.Is(err, ErrObjectNotFound) {
		return err
	}

	return nil
}

func (s *Service) ListObjects(ctx context.Context, projectID, bucketName, prefix string, limit, offset int) ([]Object, int, error) {
	bucket, err := s.repo.GetBucket(ctx, projectID, bucketName)
	if err != nil {
		return nil, 0, ErrBucketNotFound
	}
	return s.repo.ListObjects(ctx, bucket.ID, prefix, limit, offset)
}

func (s *Service) CreateBucket(ctx context.Context, projectID, name string, public bool) (*Bucket, error) {
	if err := validateBucketName(name); err != nil {
		return nil, err
	}
	return s.repo.CreateBucket(ctx, projectID, name, public)
}

func (s *Service) ListBuckets(ctx context.Context, projectID string) ([]Bucket, error) {
	return s.repo.ListBuckets(ctx, projectID)
}

func (s *Service) DeleteBucket(ctx context.Context, projectID, name string) error {
	return s.repo.DeleteBucket(ctx, projectID, name)
}

func validatePath(path string) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}
	if strings.Contains(path, "..") {
		return errors.New("path cannot contain ..")
	}
	if strings.HasPrefix(path, "/") {
		return errors.New("path cannot start with /")
	}
	return nil
}

func validateBucketName(name string) error {
	if name == "" {
		return errors.New("bucket name cannot be empty")
	}
	if len(name) > 63 {
		return errors.New("bucket name too long")
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return errors.New("bucket name must be lowercase alphanumeric or hyphens only")
		}
	}
	return nil
}
