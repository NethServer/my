/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
	"github.com/nethesis/my/backend/storage"
)

// backupIDPattern pins the shape of a valid backup identifier: a UUIDv7
// plus one of the extensions produced by the ingest side. Anything else —
// path components, traversal tokens, URL-encoded slashes, unexpected
// suffixes — is refused before it reaches the storage layer so the S3
// key cannot be diverted outside the system's prefix.
var backupIDPattern = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}(?:\.(?:tar\.gz|tar\.xz|tar\.bz2|tar\.zst|gpg|bin))?$`,
)

func isValidBackupID(id string) bool {
	return backupIDPattern.MatchString(strings.ToLower(id))
}

// BackupMetadata is the JSON payload returned in list responses for a single
// backup object. The uploader IP is intentionally not exposed: on traffic
// that transits the translation proxy the stored value is the proxy's IP
// (not the appliance's), and where it is accurate it would still be a
// reconnaissance aid for any admin above the customer tier.
type BackupMetadata struct {
	ID         string    `json:"id"`
	Filename   string    `json:"filename"`
	Size       int64     `json:"size"`
	SHA256     string    `json:"sha256"`
	MimeType   string    `json:"mimetype"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// BackupListResponse wraps the list of backups with aggregate usage counters
// (useful for rendering quota/slot indicators in the UI).
type BackupListResponse struct {
	Backups        []BackupMetadata `json:"backups"`
	QuotaUsedBytes int64            `json:"quota_used_bytes"`
	SlotsUsed      int              `json:"slots_used"`
}

// GetSystemBackups handles GET /api/systems/:id/backups — returns the list of
// backups stored for the given system, enriched with size, checksum, and
// uploader metadata. Access is gated by the same RBAC rules as GetSystem.
func GetSystemBackups(c *gin.Context) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	systemsService := local.NewSystemsService()
	system, err := systemsService.GetSystem(systemID, user.OrgRole, user.OrganizationID)
	if helpers.HandleAccessError(c, err, "system", systemID) {
		return
	}

	ctx := c.Request.Context()
	client, _, err := storage.BackupClient(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("backup storage client unavailable")
		c.JSON(http.StatusServiceUnavailable, response.Error(http.StatusServiceUnavailable, "backup storage unavailable", nil))
		return
	}

	items, used, err := listSystemBackups(ctx, client, system.Organization.LogtoID, system.SystemKey)
	if err != nil {
		logger.Error().Err(err).Str("system_id", systemID).Msg("list backups failed")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "failed to list backups", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("backups retrieved successfully", BackupListResponse{
		Backups:        items,
		QuotaUsedBytes: used,
		SlotsUsed:      len(items),
	}))
}

// DownloadSystemBackup handles GET /api/systems/:id/backups/:backup_id/download.
// It returns 200 with a JSON body containing a short-lived presigned URL so
// the caller can fetch the object directly from Spaces.
func DownloadSystemBackup(c *gin.Context) {
	systemID := c.Param("id")
	backupID := c.Param("backup_id")
	if systemID == "" || backupID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID and backup ID required", nil))
		return
	}
	if !isValidBackupID(backupID) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid backup id", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	systemsService := local.NewSystemsService()
	system, err := systemsService.GetSystem(systemID, user.OrgRole, user.OrganizationID)
	if helpers.HandleAccessError(c, err, "system", systemID) {
		return
	}

	ctx := c.Request.Context()
	client, presigner, err := storage.BackupClient(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, response.Error(http.StatusServiceUnavailable, "backup storage unavailable", nil))
		return
	}

	key := fmt.Sprintf("%s/%s/%s", system.Organization.LogtoID, system.SystemKey, backupID)

	// Verify the object exists before issuing a presigned URL. Otherwise
	// the client gets a 200 with a URL that 404s on GET, which is
	// confusing UX and leaks that the ID didn't exist only after an
	// extra round-trip.
	_, err = client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(configuration.Config.BackupS3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nf *s3types.NotFound
		if errors.As(err, &nf) {
			c.JSON(http.StatusNotFound, response.NotFound("backup not found", nil))
			return
		}
		logger.Error().Err(err).Str("key", key).Msg("head backup failed")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "failed to generate download URL", nil))
		return
	}

	presigned, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(configuration.Config.BackupS3Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(configuration.Config.BackupPresignTTL))
	if err != nil {
		logger.Error().Err(err).Str("key", key).Msg("presign backup URL failed")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "failed to generate download URL", nil))
		return
	}

	logger.LogBusinessOperation(c, "systems", "download_backup", "backup", backupID, true, nil)
	logger.RequestLogger(c, "systems").Info().
		Str("operation", "download_backup").
		Str("system_id", systemID).
		Str("backup_id", backupID).
		Int("expires_in_seconds", int(configuration.Config.BackupPresignTTL.Seconds())).
		Msg("backup download URL issued")

	c.JSON(http.StatusOK, response.OK("download URL issued", gin.H{
		"download_url":       presigned.URL,
		"expires_in_seconds": int(configuration.Config.BackupPresignTTL.Seconds()),
	}))
}

// DeleteSystemBackup handles DELETE /api/systems/:id/backups/:backup_id —
// removes a backup object from storage. RBAC-gated like the read endpoints.
func DeleteSystemBackup(c *gin.Context) {
	systemID := c.Param("id")
	backupID := c.Param("backup_id")
	if systemID == "" || backupID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID and backup ID required", nil))
		return
	}
	if !isValidBackupID(backupID) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid backup id", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	systemsService := local.NewSystemsService()
	system, err := systemsService.GetSystem(systemID, user.OrgRole, user.OrganizationID)
	if helpers.HandleAccessError(c, err, "system", systemID) {
		return
	}

	ctx := c.Request.Context()
	client, _, err := storage.BackupClient(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, response.Error(http.StatusServiceUnavailable, "backup storage unavailable", nil))
		return
	}

	key := fmt.Sprintf("%s/%s/%s", system.Organization.LogtoID, system.SystemKey, backupID)

	// S3's DeleteObject is idempotent — it returns success whether the
	// key existed or not — so a NoSuchKey error is never surfaced. Probe
	// with HeadObject first so the caller sees a 404 for phantom IDs
	// instead of a misleading 200.
	_, err = client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(configuration.Config.BackupS3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nf *s3types.NotFound
		if errors.As(err, &nf) {
			c.JSON(http.StatusNotFound, response.NotFound("backup not found", nil))
			return
		}
		logger.Error().Err(err).Str("key", key).Msg("head backup failed")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "failed to delete backup", nil))
		return
	}

	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(configuration.Config.BackupS3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		logger.Error().Err(err).Str("key", key).Msg("delete backup failed")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "failed to delete backup", nil))
		return
	}

	// Invalidate the per-org usage counter on the shared Redis so collect's
	// next quota check recomputes from S3 instead of trusting a value that
	// no longer reflects what's in the bucket.
	cache.InvalidateOrgBackupUsage(ctx, system.Organization.LogtoID)

	logger.LogBusinessOperation(c, "systems", "delete_backup", "backup", backupID, true, nil)

	c.JSON(http.StatusOK, response.OK("backup deleted", gin.H{
		"system_id": systemID,
		"backup_id": backupID,
	}))
}

// purgeSystemBackups removes every object stored under a system's S3
// prefix. It is invoked by DestroySystem before the DB row goes away
// to honour the GDPR Article 17 "right to erasure" — backups that
// belong to a deleted system must not survive in storage. A non-nil
// error means the destroy must be refused so the operator can retry.
//
// The function is a no-op when the backup storage client is not
// configured (e.g. dev environments without S3_ENDPOINT set);
// in that case there is nothing to purge.
func purgeSystemBackups(ctx context.Context, orgID, systemKey string) error {
	client, _, err := storage.BackupClient(ctx)
	if err != nil {
		// No backup storage configured for this environment.
		logger.Warn().Err(err).Str("system_key", systemKey).Msg("backup storage unavailable; skipping purge")
		return nil
	}

	prefix := fmt.Sprintf("%s/%s/", orgID, systemKey)
	purged := 0
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(configuration.Config.BackupS3Bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("list backups for purge: %w", err)
		}
		for _, o := range page.Contents {
			_, delErr := client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(configuration.Config.BackupS3Bucket),
				Key:    o.Key,
			})
			if delErr != nil {
				return fmt.Errorf("delete %s: %w", aws.ToString(o.Key), delErr)
			}
			purged++
		}
	}

	// Invalidate the per-org usage counter so collect's next upload's quota
	// check sees the freed space.
	if purged > 0 {
		cache.InvalidateOrgBackupUsage(ctx, orgID)
	}

	logger.Info().
		Str("system_key", systemKey).
		Str("org_id", orgID).
		Int("objects_purged", purged).
		Msg("system backups purged")
	return nil
}

// listSystemBackups returns the backups for a system along with the
// total bytes stored under the prefix. Paginates explicitly so the
// per-system list is never silently truncated at the S3 1000-item
// response cap.
func listSystemBackups(ctx context.Context, client *s3.Client, orgID, systemKey string) ([]BackupMetadata, int64, error) {
	prefix := fmt.Sprintf("%s/%s/", orgID, systemKey)

	items := make([]BackupMetadata, 0)
	var total int64

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(configuration.Config.BackupS3Bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, 0, err
		}
		for _, o := range page.Contents {
			key := aws.ToString(o.Key)
			id := strings.TrimPrefix(key, prefix)

			head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
				Bucket: aws.String(configuration.Config.BackupS3Bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				logger.Warn().Err(err).Str("key", key).Msg("head failed during backup list")
				continue
			}

			md := head.Metadata
			size := aws.ToInt64(head.ContentLength)
			total += size
			items = append(items, BackupMetadata{
				ID:         id,
				Filename:   md["filename"],
				Size:       size,
				SHA256:     md["sha256"],
				MimeType:   aws.ToString(head.ContentType),
				UploadedAt: aws.ToTime(o.LastModified),
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].UploadedAt.After(items[j].UploadedAt)
	})

	return items, total, nil
}
