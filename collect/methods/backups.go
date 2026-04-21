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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/queue"
	"github.com/nethesis/my/collect/response"
	"github.com/nethesis/my/collect/storage"
)

// backupIDPattern pins the shape of a valid backup identifier: a UUIDv7
// (32 hex digits with the version nibble and variant enforced) plus one of
// the extensions derived by extractExtension. Anything else — path
// components, traversal tokens, URL-encoded slashes, unexpected suffixes —
// is refused before it reaches the storage layer so the S3 key can never
// reach outside the authenticated system's prefix.
var backupIDPattern = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}(?:\.(?:tar\.gz|tar\.xz|tar\.bz2|tar\.zst|gpg|bin))?$`,
)

// isValidBackupID is the strict allowlist for backup IDs that arrive via
// user-controlled path parameters.
func isValidBackupID(id string) bool {
	return backupIDPattern.MatchString(strings.ToLower(id))
}

// safeFilenameChar keeps filenames to a conservative subset so that no
// header or UI reflection can escape its context via CR/LF, quote,
// wildcard, or non-printable bytes. Appliance-provided filenames that
// violate the set are replaced char-by-char with '_' rather than
// rejected outright — a valid backup should not be dropped just because
// the caller labelled it with Unicode or spaces.
func safeFilenameChar(r rune) rune {
	switch {
	case r >= 'A' && r <= 'Z':
	case r >= 'a' && r <= 'z':
	case r >= '0' && r <= '9':
	case r == '.' || r == '-' || r == '_':
	default:
		return '_'
	}
	return r
}

// sanitizeFilename strips path components, maps any byte outside the
// safe allowlist to '_', and truncates to 255 chars so the final value
// is safe to ship as an S3 metadata header and as an inline filename in
// Content-Disposition.
func sanitizeFilename(raw string) string {
	s := strings.Map(safeFilenameChar, raw)
	if len(s) > 255 {
		s = s[:255]
	}
	return s
}

// enforceIngestRateLimit keeps a Redis-backed sliding counter per
// system and rejects requests past the per-minute and per-hour caps.
// Returns (allowed, retryAfterSeconds). A Redis outage fails open —
// a DoS mitigation should never itself become the cause of a service
// outage, and the inline retention + auth gates still apply.
//
// Setting BackupRateLimitPerMinute or BackupRateLimitPerHour to 0 or
// below disables the corresponding check; setting both to 0 turns the
// limiter off entirely (useful in tests and for emergencies).
func enforceIngestRateLimit(ctx context.Context, systemKey string) (bool, int) {
	perMin := configuration.Config.BackupRateLimitPerMinute
	perHour := configuration.Config.BackupRateLimitPerHour
	if perMin <= 0 && perHour <= 0 {
		return true, 0
	}
	rdb := queue.GetClient()
	if rdb == nil {
		// Redis not initialised — only happens in tests or during a
		// brief boot window. Fail open.
		return true, 0
	}
	now := time.Now().Unix()

	minuteBucket := now / 60
	hourBucket := now / 3600
	minuteKey := fmt.Sprintf("backup:rate:min:%s:%d", systemKey, minuteBucket)
	hourKey := fmt.Sprintf("backup:rate:hour:%s:%d", systemKey, hourBucket)

	minCount, err := rdb.Incr(ctx, minuteKey).Result()
	if err != nil {
		logger.Warn().Err(err).Str("system_key", systemKey).Msg("rate-limit counter unreachable, failing open")
		return true, 0
	}
	if minCount == 1 {
		_ = rdb.Expire(ctx, minuteKey, 90*time.Second).Err()
	}
	if perMin > 0 && minCount > int64(perMin) {
		retry := 60 - int(now%60)
		return false, retry
	}

	hourCount, err := rdb.Incr(ctx, hourKey).Result()
	if err != nil {
		logger.Warn().Err(err).Str("system_key", systemKey).Msg("rate-limit counter unreachable, failing open")
		return true, 0
	}
	if hourCount == 1 {
		_ = rdb.Expire(ctx, hourKey, 2*time.Hour).Err()
	}
	if perHour > 0 && hourCount > int64(perHour) {
		retry := 3600 - int(now%3600)
		return false, retry
	}
	return true, 0
}

// remoteAddrIP returns the true peer IP observed by the server. Unlike
// c.ClientIP() it never honours X-Forwarded-For, so a malicious
// appliance cannot forge the uploader IP in audit metadata.
func remoteAddrIP(c *gin.Context) string {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}

// countingReader tracks how many bytes the downstream consumer has read
// so the handler can reject clients that under-declare Content-Length
// (which would otherwise leave the SHA-256 tee hashing only a prefix of
// the uploaded body).
type countingReader struct {
	r io.Reader
	n atomic.Int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 {
		c.n.Add(int64(n))
	}
	return n, err
}

// BackupMetadata is the JSON payload returned for list/upload operations.
type BackupMetadata struct {
	ID         string    `json:"id"`
	Filename   string    `json:"filename"`
	Size       int64     `json:"size"`
	SHA256     string    `json:"sha256"`
	MimeType   string    `json:"mimetype"`
	UploadedAt time.Time `json:"uploaded_at"`
	UploaderIP string    `json:"uploader_ip,omitempty"`
}

// UploadBackup streams a configuration backup uploaded by an authenticated
// appliance into the Garage-backed object store. The body is read as a single
// stream, hashed with SHA-256 on the fly, and multipart-uploaded to S3. After
// the object is committed the final SHA-256 is attached via CopyObject so it
// appears in subsequent HEAD/GET metadata. Retention caps (count + total size)
// are then enforced by pruning the oldest objects under the system's prefix.
func UploadBackup(c *gin.Context) {
	systemID, ok := getAuthenticatedSystemID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}
	systemKey, _ := getAuthenticatedSystemKey(c)

	// Per-system ingest rate limit — blocks flood-style abuse ahead of
	// the expensive storage path.
	if ok, retry := enforceIngestRateLimit(c.Request.Context(), systemKey); !ok {
		c.Header("Retry-After", fmt.Sprintf("%d", retry))
		c.JSON(http.StatusTooManyRequests, response.Error(http.StatusTooManyRequests, "upload rate limit exceeded", gin.H{
			"retry_after_seconds": retry,
		}))
		return
	}

	// Require an explicit Content-Length. A missing or -1 value would
	// leave the handler dependent on MaxBytesReader alone and make the
	// post-upload byte reconciliation impossible.
	if c.Request.ContentLength <= 0 {
		c.JSON(http.StatusLengthRequired, response.Error(http.StatusLengthRequired, "Content-Length required", nil))
		return
	}
	if c.Request.ContentLength > configuration.Config.BackupMaxUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, response.Error(http.StatusRequestEntityTooLarge, "backup exceeds max upload size", gin.H{
			"max_bytes": configuration.Config.BackupMaxUploadSize,
		}))
		return
	}

	orgID, err := lookupSystemOrgID(systemID)
	if err != nil {
		logger.Error().Err(err).Str("system_id", systemID).Msg("failed to resolve system organization")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve system organization", nil))
		return
	}

	ctx := c.Request.Context()
	client, _, err := storage.BackupClient(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("backup storage client unavailable")
		c.JSON(http.StatusServiceUnavailable, response.Error(http.StatusServiceUnavailable, "backup storage unavailable", nil))
		return
	}

	// Per-org aggregate quota — reject before streaming the body if the
	// incoming upload would push the org over its ceiling.
	if configuration.Config.BackupMaxSizePerOrg > 0 {
		used, err := computeOrgBackupUsage(ctx, client, orgID)
		if err != nil {
			logger.Warn().Err(err).Str("org_id", orgID).Msg("failed to compute org quota, allowing upload")
		} else if used+c.Request.ContentLength > configuration.Config.BackupMaxSizePerOrg {
			c.JSON(http.StatusRequestEntityTooLarge, response.Error(http.StatusRequestEntityTooLarge, "organization backup quota exceeded", gin.H{
				"used_bytes": used,
				"max_bytes":  configuration.Config.BackupMaxSizePerOrg,
			}))
			return
		}
	}

	backupID := uuid.Must(uuid.NewV7())
	filename := extractFilename(c.GetHeader("X-Filename"), backupID.String())
	ext := extractExtension(filename)
	// Object key uses system_key (the stable NETH-…-shaped identifier)
	// instead of the internal UUID system_id so operators browsing a raw
	// bucket listing can recognise each system at a glance.
	key := fmt.Sprintf("%s/%s/%s%s", orgID, systemKey, backupID.String(), ext)
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	hasher := sha256.New()
	limitedBody := http.MaxBytesReader(c.Writer, c.Request.Body, configuration.Config.BackupMaxUploadSize)
	counter := &countingReader{r: limitedBody}
	teeReader := io.TeeReader(counter, hasher)

	metadata := map[string]string{
		"filename":    sanitizeFilename(filename),
		"uploader-ip": remoteAddrIP(c),
		"sha256":      "pending",
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(configuration.Config.BackupS3Bucket),
		Key:                  aws.String(key),
		Body:                 teeReader,
		ContentLength:        aws.Int64(c.Request.ContentLength),
		ContentType:          aws.String(contentType),
		Metadata:             metadata,
		ServerSideEncryption: s3types.ServerSideEncryptionAes256,
	})
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, response.Error(http.StatusRequestEntityTooLarge, "backup exceeds max upload size", nil))
			return
		}
		logger.Error().Err(err).Str("system_id", systemID).Str("key", key).Msg("backup upload failed")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "backup upload failed", nil))
		return
	}

	// Reconcile actual body length against the declared Content-Length.
	// A mismatch means either the client lied in the header (truncating
	// the SHA-256 tee) or the S3 client silently padded the request.
	// In either case the stored object cannot be trusted — drop it.
	observed := counter.n.Load()
	if observed != c.Request.ContentLength {
		deleteOrphanObject(ctx, client, key)
		logger.Error().
			Str("system_id", systemID).
			Str("key", key).
			Int64("declared_bytes", c.Request.ContentLength).
			Int64("observed_bytes", observed).
			Msg("upload length mismatch — object dropped")
		c.JSON(http.StatusBadRequest, response.BadRequest("upload length mismatch", nil))
		return
	}

	sha := hex.EncodeToString(hasher.Sum(nil))
	metadata["sha256"] = sha

	// Rewrite the object's metadata with the real checksum. CopyObject
	// against the same key is a server-side operation, so it is fast and
	// does not transfer the body again. Any failure here leaves the
	// object with sha256=pending, which would silently lie to later
	// readers — drop the object and surface a 502 instead.
	_, err = client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:               aws.String(configuration.Config.BackupS3Bucket),
		Key:                  aws.String(key),
		CopySource:           aws.String(configuration.Config.BackupS3Bucket + "/" + key),
		Metadata:             metadata,
		MetadataDirective:    s3types.MetadataDirectiveReplace,
		ContentType:          aws.String(contentType),
		ServerSideEncryption: s3types.ServerSideEncryptionAes256,
	})
	if err != nil {
		deleteOrphanObject(ctx, client, key)
		logger.Error().Err(err).Str("key", key).Msg("failed to attach final sha256 metadata — object dropped")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "failed to persist backup metadata", nil))
		return
	}

	head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(configuration.Config.BackupS3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		logger.Warn().Err(err).Str("key", key).Msg("failed to HEAD uploaded backup")
	}

	var size int64
	if head != nil {
		size = aws.ToInt64(head.ContentLength)
	}

	if err := enforceBackupRetention(ctx, client, orgID, systemKey); err != nil {
		logger.Warn().Err(err).Str("system_id", systemID).Str("system_key", systemKey).Msg("retention enforcement failed")
	}

	logger.Info().
		Str("component", "backup").
		Str("operation", "upload").
		Str("system_id", systemID).
		Str("org_id", orgID).
		Str("key", key).
		Str("sha256", sha).
		Int64("size", size).
		Msg("backup stored")

	c.JSON(http.StatusCreated, response.Created("backup stored", BackupMetadata{
		ID:         backupID.String() + ext,
		Filename:   metadata["filename"],
		Size:       size,
		SHA256:     sha,
		MimeType:   contentType,
		UploadedAt: time.Now().UTC(),
		UploaderIP: metadata["uploader-ip"],
	}))
}

// ListBackups returns the list of backups stored for the authenticated system.
func ListBackups(c *gin.Context) {
	systemID, ok := getAuthenticatedSystemID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}
	systemKey, _ := getAuthenticatedSystemKey(c)

	orgID, err := lookupSystemOrgID(systemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve system organization", nil))
		return
	}

	ctx := c.Request.Context()
	client, _, err := storage.BackupClient(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, response.Error(http.StatusServiceUnavailable, "backup storage unavailable", nil))
		return
	}

	items, err := listBackupsForSystem(ctx, client, orgID, systemKey)
	if err != nil {
		logger.Error().Err(err).Str("system_id", systemID).Str("system_key", systemKey).Msg("list backups failed")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "failed to list backups", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("backups listed", gin.H{"backups": items}))
}

// DownloadBackup streams a stored backup back to the authenticated appliance
// for restore operations.
func DownloadBackup(c *gin.Context) {
	systemID, ok := getAuthenticatedSystemID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}
	systemKey, _ := getAuthenticatedSystemKey(c)

	backupID := c.Param("id")
	if backupID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("backup id required", nil))
		return
	}
	if !isValidBackupID(backupID) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid backup id", nil))
		return
	}

	orgID, err := lookupSystemOrgID(systemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve system organization", nil))
		return
	}

	ctx := c.Request.Context()
	client, _, err := storage.BackupClient(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, response.Error(http.StatusServiceUnavailable, "backup storage unavailable", nil))
		return
	}

	key := fmt.Sprintf("%s/%s/%s", orgID, systemKey, backupID)
	obj, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(configuration.Config.BackupS3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *s3types.NoSuchKey
		if errors.As(err, &nsk) {
			c.JSON(http.StatusNotFound, response.NotFound("backup not found", nil))
			return
		}
		logger.Error().Err(err).Str("key", key).Msg("get backup failed")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "failed to fetch backup", nil))
		return
	}
	defer func() {
		if cerr := obj.Body.Close(); cerr != nil {
			logger.Warn().Err(cerr).Str("key", key).Msg("backup body close failed")
		}
	}()

	contentType := aws.ToString(obj.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	if obj.ContentLength != nil {
		c.Header("Content-Length", fmt.Sprintf("%d", aws.ToInt64(obj.ContentLength)))
	}
	if filename, ok := obj.Metadata["filename"]; ok && filename != "" {
		// Filename has already been sanitized at ingest, but apply the
		// same filter again on the read path in case an object was
		// written by an older build or directly via the S3 API.
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, sanitizeFilename(filename)))
	}
	c.Status(http.StatusOK)

	if _, err := io.Copy(c.Writer, obj.Body); err != nil {
		logger.Warn().Err(err).Str("key", key).Msg("backup stream interrupted")
	}
}

// listBackupsForSystem lists every backup object under the
// {org}/{system_key} prefix and maps it to BackupMetadata. Paginates
// explicitly so the per-system list is never silently truncated at the
// S3 1000-item response cap.
func listBackupsForSystem(ctx context.Context, client *s3.Client, orgID, systemKey string) ([]BackupMetadata, error) {
	prefix := fmt.Sprintf("%s/%s/", orgID, systemKey)

	items := make([]BackupMetadata, 0)
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(configuration.Config.BackupS3Bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, o := range page.Contents {
			key := aws.ToString(o.Key)
			id := strings.TrimPrefix(key, prefix)

			head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
				Bucket: aws.String(configuration.Config.BackupS3Bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				logger.Warn().Err(err).Str("key", key).Msg("head failed during list")
				continue
			}

			md := head.Metadata
			items = append(items, BackupMetadata{
				ID:         id,
				Filename:   md["filename"],
				Size:       aws.ToInt64(head.ContentLength),
				SHA256:     md["sha256"],
				MimeType:   aws.ToString(head.ContentType),
				UploadedAt: aws.ToTime(o.LastModified),
				UploaderIP: md["uploader-ip"],
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].UploadedAt.After(items[j].UploadedAt)
	})

	return items, nil
}

// enforceBackupRetention deletes the oldest backups under a system's
// prefix until both BackupMaxPerSystem and BackupMaxSizePerSystem
// limits are met. It is serialised by a Redis lock keyed on the system
// (so two concurrent uploads cannot both delete the same victim and
// race over the survivor count) and orders objects by their key —
// because the keys embed a UUIDv7, lexicographic order is monotonic in
// upload time and immune to S3 LastModified clock skew.
func enforceBackupRetention(ctx context.Context, client *s3.Client, orgID, systemKey string) error {
	lockKey := fmt.Sprintf("backup:retention:%s", systemKey)
	rdb := queue.GetClient()
	locked, err := rdb.SetNX(ctx, lockKey, "1", 60*time.Second).Result()
	if err != nil {
		// Redis unreachable — skip retention this round; the next
		// upload will retry. Better to leave one extra backup than
		// to delete the wrong one.
		logger.Warn().Err(err).Str("system_key", systemKey).Msg("retention: lock acquire failed, skipping")
		return nil
	}
	if !locked {
		// Another upload is already pruning this system; do nothing.
		return nil
	}
	defer func() {
		if _, err := rdb.Del(ctx, lockKey).Result(); err != nil {
			logger.Warn().Err(err).Str("system_key", systemKey).Msg("retention: lock release failed")
		}
	}()

	prefix := fmt.Sprintf("%s/%s/", orgID, systemKey)

	objs := make([]s3types.Object, 0)
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(configuration.Config.BackupS3Bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}
		objs = append(objs, page.Contents...)
	}

	// Sort by key (ascending). The {uuid}.{ext} suffix begins with a
	// UUIDv7 whose first 48 bits are the upload's millisecond
	// timestamp, so lexicographic order matches chronological order
	// without depending on the storage backend's clock.
	sort.Slice(objs, func(i, j int) bool {
		return aws.ToString(objs[i].Key) < aws.ToString(objs[j].Key)
	})

	var totalSize int64
	for _, o := range objs {
		totalSize += aws.ToInt64(o.Size)
	}

	maxCount := configuration.Config.BackupMaxPerSystem
	maxSize := configuration.Config.BackupMaxSizePerSystem

	for len(objs) > 0 && (len(objs) > maxCount || totalSize > maxSize) {
		victim := objs[0]
		objs = objs[1:]
		_, delErr := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(configuration.Config.BackupS3Bucket),
			Key:    victim.Key,
		})
		if delErr != nil {
			logger.Warn().Err(delErr).Str("key", aws.ToString(victim.Key)).Msg("retention: delete failed")
			continue
		}
		totalSize -= aws.ToInt64(victim.Size)
		logger.Info().Str("key", aws.ToString(victim.Key)).Msg("retention: oldest backup pruned")
	}

	return nil
}

// computeOrgBackupUsage returns the sum of every backup object size
// stored under the given organization prefix. Paginates to handle orgs
// with many systems × many retained backups.
func computeOrgBackupUsage(ctx context.Context, client *s3.Client, orgID string) (int64, error) {
	prefix := fmt.Sprintf("%s/", orgID)
	var total int64
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(configuration.Config.BackupS3Bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return 0, err
		}
		for _, o := range page.Contents {
			total += aws.ToInt64(o.Size)
		}
	}
	return total, nil
}

// lookupSystemOrgID returns the organization_id for the given system_id.
func lookupSystemOrgID(systemID string) (string, error) {
	var orgID string
	err := database.DB.QueryRow(
		`SELECT organization_id FROM systems WHERE id = $1 AND deleted_at IS NULL`,
		systemID,
	).Scan(&orgID)
	if err != nil {
		return "", err
	}
	return orgID, nil
}

// extractFilename returns a safe user-facing filename, falling back to the
// backup UUID if the appliance did not provide one.
func extractFilename(headerValue, fallbackID string) string {
	h := strings.TrimSpace(headerValue)
	if h == "" {
		return "backup-" + fallbackID
	}
	// Strip any path components so headers cannot escape the bucket prefix.
	if i := strings.LastIndexAny(h, "/\\"); i >= 0 {
		h = h[i+1:]
	}
	return h
}

// deleteOrphanObject removes an S3 object that was written but whose
// metadata could not be committed; it is always best-effort (the caller
// already owns the surfaced error) and only logs the outcome.
func deleteOrphanObject(ctx context.Context, client *s3.Client, key string) {
	_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(configuration.Config.BackupS3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		logger.Warn().Err(err).Str("key", key).Msg("orphan backup cleanup failed")
	}
}

// extractExtension returns the file extension from a filename, including the
// leading dot. Falls back to ".bin" when no extension is present.
func extractExtension(filename string) string {
	// Preserve compound extensions like ".tar.gz" and ".tar.xz".
	lower := strings.ToLower(filename)
	for _, compound := range []string{".tar.gz", ".tar.xz", ".tar.bz2", ".tar.zst"} {
		if strings.HasSuffix(lower, compound) {
			return compound
		}
	}
	if i := strings.LastIndex(filename, "."); i >= 0 && i < len(filename)-1 {
		return strings.ToLower(filename[i:])
	}
	return ".bin"
}
