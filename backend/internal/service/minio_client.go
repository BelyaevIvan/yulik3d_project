package service

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

// MinioClient — тонкая обёртка для нужд сервиса.
type MinioClient struct {
	cl        *minio.Client
	bucket    string
	publicURL string
}

func NewMinioClient(cl *minio.Client, bucket, publicURL string) *MinioClient {
	return &MinioClient{cl: cl, bucket: bucket, publicURL: publicURL}
}

// Put загружает объект. Возвращает nil при успехе.
func (m *MinioClient) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := m.cl.PutObject(ctx, m.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

// Delete удаляет объект. Ошибка best-effort.
func (m *MinioClient) Delete(ctx context.Context, key string) error {
	return m.cl.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
}

// URL собирает абсолютный публичный URL.
func (m *MinioClient) URL(objectKey string) string {
	return fmt.Sprintf("%s/%s/%s", m.publicURL, m.bucket, objectKey)
}