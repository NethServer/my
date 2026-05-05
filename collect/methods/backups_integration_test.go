//go:build integration
// +build integration

/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackupStorageRoundTrip runs the same sequence of S3 operations the
// UploadBackup, ListBackups, DownloadBackup and DeleteSystemBackup
// handlers perform — PutObject, CopyObject (for the sha256 metadata
// reconciliation), HeadObject, ListObjectsV2, GetObject, DeleteObject
// — against a live S3-compatible endpoint. It does not exercise auth,
// DB lookups, or the Gin layer; its purpose is to catch any drift
// between what the SDK emits and what the target server (MinIO,
// Garage, Spaces, AWS S3) accepts.
//
// Runs only under `go test -tags=integration` and only when the
// `BACKUP_S3_*` environment variables are set — point it at any
// bucket you own (DO Spaces, AWS S3, R2, MinIO, …). It is not part
// of the default CI pipeline; run it manually when touching the S3
// integration layer.
func TestBackupStorageRoundTrip(t *testing.T) {
	endpoint := os.Getenv("S3_ENDPOINT")
	bucket := os.Getenv("BACKUP_S3_BUCKET")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		t.Skip("BACKUP_S3_* env vars not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := buildIntegrationClient(t, endpoint, accessKey, secretKey)

	testKey := fmt.Sprintf("integration-test/%d.bin", time.Now().UnixNano())
	body := []byte("hello backup integration test\n")

	// Clean up on any exit path — best effort.
	defer func() {
		_, _ = client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(testKey),
		})
	}()

	t.Run("put_object_with_metadata_placeholder", func(t *testing.T) {
		_, err := client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:               aws.String(bucket),
			Key:                  aws.String(testKey),
			Body:                 bytes.NewReader(body),
			ContentLength:        aws.Int64(int64(len(body))),
			ContentType:          aws.String("application/octet-stream"),
			Metadata:             map[string]string{"sha256": "pending"},
			ServerSideEncryption: s3types.ServerSideEncryptionAes256,
		})
		require.NoError(t, err)
	})

	t.Run("copy_object_replaces_metadata", func(t *testing.T) {
		_, err := client.CopyObject(ctx, &s3.CopyObjectInput{
			Bucket:               aws.String(bucket),
			Key:                  aws.String(testKey),
			CopySource:           aws.String(bucket + "/" + testKey),
			Metadata:             map[string]string{"sha256": "final-hash", "filename": "test.bin"},
			MetadataDirective:    s3types.MetadataDirectiveReplace,
			ContentType:          aws.String("application/octet-stream"),
			ServerSideEncryption: s3types.ServerSideEncryptionAes256,
		})
		require.NoError(t, err)
	})

	t.Run("head_object_returns_final_metadata", func(t *testing.T) {
		head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(testKey),
		})
		require.NoError(t, err)
		assert.Equal(t, "final-hash", head.Metadata["sha256"])
		assert.Equal(t, "test.bin", head.Metadata["filename"])
		assert.Equal(t, int64(len(body)), aws.ToInt64(head.ContentLength))
	})

	t.Run("list_objects_includes_test_key", func(t *testing.T) {
		paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
			Bucket: aws.String(bucket),
			Prefix: aws.String("integration-test/"),
		})
		found := false
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			require.NoError(t, err)
			for _, obj := range page.Contents {
				if aws.ToString(obj.Key) == testKey {
					found = true
					assert.Equal(t, int64(len(body)), aws.ToInt64(obj.Size))
				}
			}
		}
		assert.True(t, found, "test key must be present in the listing")
	})

	t.Run("get_object_returns_exact_body", func(t *testing.T) {
		get, err := client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(testKey),
		})
		require.NoError(t, err)
		defer func() {
			_ = get.Body.Close()
		}()
		gotBody, err := io.ReadAll(get.Body)
		require.NoError(t, err)
		assert.Equal(t, body, gotBody)
	})

	t.Run("delete_object_removes_the_key", func(t *testing.T) {
		_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(testKey),
		})
		require.NoError(t, err)

		_, err = client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(testKey),
		})
		assert.Error(t, err, "the key must be gone after delete")
	})
}

func buildIntegrationClient(t *testing.T, endpoint, accessKey, secretKey string) *s3.Client {
	t.Helper()
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	require.NoError(t, err)
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		// Local emulators and many S3-compatible gateways reach objects
		// via path-style URLs. Safe to force on for the integration
		// test regardless of provider.
		o.UsePathStyle = true
	})
}
