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
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
)

// CopyBackupPrefix copies every backup object stored under
// {fromOrgID}/{systemKey}/ to {toOrgID}/{systemKey}/, preserving the per-object
// metadata. CopyObject is idempotent (it overwrites the destination), so a
// retry after a partial failure safely resumes the copy. The function does
// not delete source objects — callers run DeleteBackupPrefix only after the
// authoritative organization_id flip has been committed in the database.
//
// A no-op when the backup storage client is not configured (dev environments
// without S3_ENDPOINT): the system has no backups to migrate anywhere.
func CopyBackupPrefix(ctx context.Context, fromOrgID, toOrgID, systemKey string) (int, error) {
	if fromOrgID == toOrgID {
		return 0, nil
	}

	client, _, err := BackupClient(ctx)
	if err != nil {
		logger.Warn().Err(err).Str("system_key", systemKey).Msg("backup storage unavailable; skipping prefix copy")
		return 0, nil
	}

	bucket := configuration.Config.BackupS3Bucket
	srcPrefix := fmt.Sprintf("%s/%s/", fromOrgID, systemKey)
	dstPrefix := fmt.Sprintf("%s/%s/", toOrgID, systemKey)

	copied := 0
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(srcPrefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return copied, fmt.Errorf("list source backups: %w", err)
		}
		for _, o := range page.Contents {
			srcKey := aws.ToString(o.Key)
			// Trim only the source prefix and append it to the destination
			// prefix so the trailing UUIDv7 + extension is preserved verbatim.
			dstKey := dstPrefix + srcKey[len(srcPrefix):]

			// CopySource expects "{bucket}/{url-escaped-key}". The key may
			// contain characters AWS treats as path separators if not escaped.
			copySource := bucket + "/" + url.PathEscape(srcKey)

			_, err := client.CopyObject(ctx, &s3.CopyObjectInput{
				Bucket:            aws.String(bucket),
				Key:               aws.String(dstKey),
				CopySource:        aws.String(copySource),
				MetadataDirective: "COPY",
				// Re-assert SSE on the destination so the copy doesn't
				// silently land unencrypted if the bucket-default policy
				// drifts. The ingest path on collect always sets AES256;
				// matching it here keeps every object on the bucket
				// uniformly encrypted.
				ServerSideEncryption: s3types.ServerSideEncryptionAes256,
			})
			if err != nil {
				return copied, fmt.Errorf("copy %s -> %s: %w", srcKey, dstKey, err)
			}
			copied++
		}
	}

	logger.Info().
		Str("system_key", systemKey).
		Str("from_org_id", fromOrgID).
		Str("to_org_id", toOrgID).
		Int("objects_copied", copied).
		Msg("backup prefix copied across organizations")
	return copied, nil
}

// DeleteBackupPrefix removes every object stored under {orgID}/{systemKey}/.
// Used as a best-effort cleanup after a successful CopyBackupPrefix +
// organization_id flip: the destination prefix is now authoritative and the
// source bytes are safe to drop. A failure is recoverable (the orphan objects
// still occupy the source organization's quota until a sweep removes them)
// and must not surface as a request failure to the admin who triggered the
// reassignment.
//
// A no-op when the backup storage client is not configured.
func DeleteBackupPrefix(ctx context.Context, orgID, systemKey string) (int, error) {
	client, _, err := BackupClient(ctx)
	if err != nil {
		logger.Warn().Err(err).Str("system_key", systemKey).Msg("backup storage unavailable; skipping prefix delete")
		return 0, nil
	}

	bucket := configuration.Config.BackupS3Bucket
	prefix := fmt.Sprintf("%s/%s/", orgID, systemKey)

	deleted := 0
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return deleted, fmt.Errorf("list backups for delete: %w", err)
		}
		for _, o := range page.Contents {
			_, delErr := client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    o.Key,
			})
			if delErr != nil {
				return deleted, fmt.Errorf("delete %s: %w", aws.ToString(o.Key), delErr)
			}
			deleted++
		}
	}

	logger.Info().
		Str("system_key", systemKey).
		Str("org_id", orgID).
		Int("objects_deleted", deleted).
		Msg("backup prefix deleted")
	return deleted, nil
}
