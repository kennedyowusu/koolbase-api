package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Client struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

// NewR2Client accepts a full S3-compatible endpoint URL.
// For Cloudflare R2: https://<accountID>.r2.cloudflarestorage.com
// For MinIO:         http://minio:9000
// For AWS S3:        https://s3.<region>.amazonaws.com
func NewR2Client(endpoint, accessKeyID, secretAccessKey, bucket, publicURL string) *R2Client {
	client := s3.New(s3.Options{
		BaseEndpoint:       aws.String(endpoint),
		Region:             "auto",
		Credentials:        credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		UsePathStyle:       true, // required for MinIO
	})
	return &R2Client{client: client, bucket: bucket, publicURL: publicURL}
}

func (r *R2Client) GenerateUploadURL(ctx context.Context, key, contentType string) (string, error) {
	presigner := s3.NewPresignClient(r.client)
	req, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (r *R2Client) GenerateDownloadURL(ctx context.Context, key string) (string, error) {
	presigner := s3.NewPresignClient(r.client)
	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(1*time.Hour))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (r *R2Client) DeleteObject(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (r *R2Client) ObjectExists(ctx context.Context, key string) bool {
	_, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	return err == nil
}

func (r *R2Client) StorageKey(projectID, bucketName, path string) string {
	return fmt.Sprintf("%s/%s/%s", projectID, bucketName, path)
}

func (r *R2Client) PutObject(ctx context.Context, key, contentType string, body io.Reader, size int64) error {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(r.bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(contentType),
		Body:          body,
		ContentLength: aws.Int64(size),
	})
	return err
}
