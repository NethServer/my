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

package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/backend/configuration"
)

// TestCopyAndDeleteBackupPrefix exercises the helpers used by the
// org-reassignment flow against a live S3-compatible endpoint. It seeds three
// objects under {fromOrg}/{systemKey}/, copies them to {toOrg}/{systemKey}/,
// asserts the destination matches, runs the copy a second time to confirm
// idempotency, then deletes the source prefix.
//
// Skipped unless BACKUP_S3_* env vars are set; not part of the default CI
// pipeline. Run with `go test -tags=integration ./storage/...` after pointing
// the BACKUP_S3_* env vars at any reachable bucket (MinIO, DO Spaces, R2, AWS).
func TestCopyAndDeleteBackupPrefix(t *testing.T) {
	endpoint := os.Getenv("S3_ENDPOINT")
	bucket := os.Getenv("BACKUP_S3_BUCKET")
	if endpoint == "" || bucket == "" {
		t.Skip("BACKUP_S3_* env vars not set; skipping integration test")
	}

	configuration.Config.S3Endpoint = endpoint
	configuration.Config.BackupS3Bucket = bucket
	configuration.Config.S3AccessKey = os.Getenv("S3_ACCESS_KEY")
	configuration.Config.S3SecretKey = os.Getenv("S3_SECRET_KEY")
	configuration.Config.BackupS3Region = os.Getenv("BACKUP_S3_REGION")
	if configuration.Config.BackupS3Region == "" {
		configuration.Config.BackupS3Region = "us-east-1"
	}
	if v := os.Getenv("BACKUP_S3_USE_PATH_STYLE"); v == "true" {
		configuration.Config.BackupS3UsePathStyle = true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, _, err := BackupClient(ctx)
	require.NoError(t, err, "backup client init should succeed once env vars are set")

	systemKey := fmt.Sprintf("integration-test-NETH-%d", time.Now().UnixNano())
	fromOrg := "org-source-" + systemKey
	toOrg := "org-target-" + systemKey

	srcPrefix := fmt.Sprintf("%s/%s/", fromOrg, systemKey)
	dstPrefix := fmt.Sprintf("%s/%s/", toOrg, systemKey)

	// Seed: three objects under the source prefix with realistic metadata.
	seedKeys := []string{
		srcPrefix + "01934fab-bc33-7890-a1b2-c3d4e5f6a701.tar.gz",
		srcPrefix + "01934fab-bc33-7890-a1b2-c3d4e5f6a702.tar.xz",
		srcPrefix + "01934fab-bc33-7890-a1b2-c3d4e5f6a703.gpg",
	}
	for i, k := range seedKeys {
		body := []byte(fmt.Sprintf("test backup payload %d", i))
		_, err := client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:               aws.String(bucket),
			Key:                  aws.String(k),
			Body:                 bytes.NewReader(body),
			ContentLength:        aws.Int64(int64(len(body))),
			ContentType:          aws.String("application/octet-stream"),
			Metadata:             map[string]string{"sha256": fmt.Sprintf("seed-%d", i), "filename": "seeded.bin"},
			ServerSideEncryption: s3types.ServerSideEncryptionAes256,
		})
		require.NoError(t, err, "seed put for %s", k)
	}

	// Best-effort cleanup on any exit path.
	defer func() {
		_, _ = DeleteBackupPrefix(context.Background(), fromOrg, systemKey)
		_, _ = DeleteBackupPrefix(context.Background(), toOrg, systemKey)
	}()

	t.Run("copy", func(t *testing.T) {
		copied, err := CopyBackupPrefix(ctx, fromOrg, toOrg, systemKey)
		require.NoError(t, err)
		assert.Equal(t, 3, copied)

		dstObjs := listKeys(t, ctx, client, bucket, dstPrefix)
		assert.Len(t, dstObjs, 3)

		// Each destination key must mirror the source name byte-for-byte
		// after the prefix swap, so backup IDs survive the migration.
		expected := map[string]bool{
			dstPrefix + "01934fab-bc33-7890-a1b2-c3d4e5f6a701.tar.gz": true,
			dstPrefix + "01934fab-bc33-7890-a1b2-c3d4e5f6a702.tar.xz": true,
			dstPrefix + "01934fab-bc33-7890-a1b2-c3d4e5f6a703.gpg":    true,
		}
		for _, k := range dstObjs {
			assert.True(t, expected[k], "unexpected destination key %s", k)
		}

		// Spot-check that the payload survived intact.
		got, err := client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(dstPrefix + "01934fab-bc33-7890-a1b2-c3d4e5f6a701.tar.gz"),
		})
		require.NoError(t, err)
		body, err := io.ReadAll(got.Body)
		require.NoError(t, err)
		_ = got.Body.Close()
		assert.Equal(t, "test backup payload 0", string(body))
	})

	t.Run("copy_is_idempotent", func(t *testing.T) {
		copied, err := CopyBackupPrefix(ctx, fromOrg, toOrg, systemKey)
		require.NoError(t, err)
		// Second run still copies all source objects (CopyObject overwrites).
		// What matters is the destination object count is unchanged.
		assert.Equal(t, 3, copied)
		assert.Len(t, listKeys(t, ctx, client, bucket, dstPrefix), 3)
	})

	t.Run("delete_source", func(t *testing.T) {
		deleted, err := DeleteBackupPrefix(ctx, fromOrg, systemKey)
		require.NoError(t, err)
		assert.Equal(t, 3, deleted)
		assert.Empty(t, listKeys(t, ctx, client, bucket, srcPrefix))
		// Destination is still intact after the source-side cleanup.
		assert.Len(t, listKeys(t, ctx, client, bucket, dstPrefix), 3)
	})

	t.Run("delete_is_idempotent", func(t *testing.T) {
		deleted, err := DeleteBackupPrefix(ctx, fromOrg, systemKey)
		require.NoError(t, err)
		assert.Equal(t, 0, deleted)
	})

	t.Run("same_org_is_no_op", func(t *testing.T) {
		copied, err := CopyBackupPrefix(ctx, toOrg, toOrg, systemKey)
		require.NoError(t, err)
		assert.Equal(t, 0, copied)
	})
}

func listKeys(t *testing.T, ctx context.Context, client *s3.Client, bucket, prefix string) []string {
	t.Helper()
	out := []string{}
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		require.NoError(t, err)
		for _, o := range page.Contents {
			out = append(out, aws.ToString(o.Key))
		}
	}
	return out
}
