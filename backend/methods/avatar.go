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
	"fmt"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strings"

	"golang.org/x/image/draw"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
	logto "github.com/nethesis/my/backend/services/logto"
)

const (
	maxAvatarFileSize = 500 * 1024 // 500KB (resized to 256x256 PNG on server)
	avatarMaxDim      = 256        // 256x256 pixels
)

var allowedAvatarMimes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
}

// GetPublicAvatar serves a user's avatar image without authentication.
// GET /api/public/avatars/:id
func GetPublicAvatar(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("user id required", nil))
		return
	}

	repo := entities.NewLocalUserRepository()
	data, mime, err := repo.GetAvatar(userID)
	if err != nil || data == nil {
		c.JSON(http.StatusNotFound, response.NotFound("avatar not found", nil))
		return
	}

	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, mime, data)
}

// UploadMyAvatar handles avatar upload for the current user.
// PUT /api/me/avatar
func UploadMyAvatar(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	if user.LogtoID == nil || *user.LogtoID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("avatar upload not available for this user", nil))
		return
	}

	processAvatarUpload(c, user.ID, *user.LogtoID)
}

// DeleteMyAvatar removes the avatar for the current user.
// DELETE /api/me/avatar
func DeleteMyAvatar(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	if user.LogtoID == nil || *user.LogtoID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("avatar removal not available for this user", nil))
		return
	}

	processAvatarDelete(c, user.ID, *user.LogtoID)
}

// UploadUserAvatar handles avatar upload for another user (admin).
// PUT /api/users/:id/avatar
func UploadUserAvatar(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("user id required", nil))
		return
	}

	currentUser, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get target user
	repo := entities.NewLocalUserRepository()
	targetUser, err := repo.GetByID(userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("user not found", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get user", nil))
		return
	}

	// Check hierarchical access
	if !canManageUserAvatar(currentUser, targetUser) {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied", nil))
		return
	}

	if targetUser.LogtoID == nil || *targetUser.LogtoID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("user not synced to logto", nil))
		return
	}

	processAvatarUpload(c, targetUser.ID, *targetUser.LogtoID)
}

// DeleteUserAvatar removes the avatar for another user (admin).
// DELETE /api/users/:id/avatar
func DeleteUserAvatar(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("user id required", nil))
		return
	}

	currentUser, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get target user
	repo := entities.NewLocalUserRepository()
	targetUser, err := repo.GetByID(userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, response.NotFound("user not found", nil))
			return
		}
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get user", nil))
		return
	}

	// Check hierarchical access
	if !canManageUserAvatar(currentUser, targetUser) {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied", nil))
		return
	}

	if targetUser.LogtoID == nil || *targetUser.LogtoID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("user not synced to logto", nil))
		return
	}

	processAvatarDelete(c, targetUser.ID, *targetUser.LogtoID)
}

// processAvatarUpload handles the common avatar upload logic.
func processAvatarUpload(c *gin.Context, localUserID, logtoID string) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(maxAvatarFileSize); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid multipart form", nil))
		return
	}

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("avatar file is required", nil))
		return
	}
	defer func() { _ = file.Close() }()

	// Validate file size
	if header.Size > maxAvatarFileSize {
		c.JSON(http.StatusBadRequest, response.BadRequest("avatar exceeds maximum size of 500KB", gin.H{
			"max_size":    maxAvatarFileSize,
			"actual_size": header.Size,
		}))
		return
	}

	// Validate MIME type
	contentType := header.Header.Get("Content-Type")
	if !allowedAvatarMimes[contentType] {
		c.JSON(http.StatusBadRequest, response.BadRequest("avatar must be png, jpeg, or webp", gin.H{
			"content_type": contentType,
		}))
		return
	}

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to read avatar file", nil))
		return
	}

	// Decode image to validate it and get dimensions
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid image file", nil))
		return
	}

	// Resize to 64x64 if larger
	bounds := img.Bounds()
	if bounds.Dx() > avatarMaxDim || bounds.Dy() > avatarMaxDim {
		img = resizeImage(img, avatarMaxDim, avatarMaxDim)
	}

	// Re-encode as PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to process avatar", nil))
		return
	}

	pngData := buf.Bytes()
	pngMime := "image/png"

	// Save to database
	repo := entities.NewLocalUserRepository()
	if err := repo.SetAvatar(localUserID, pngData, pngMime); err != nil {
		logger.Error().Err(err).Str("user_id", localUserID).Msg("failed to save avatar to database")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to save avatar", nil))
		return
	}

	// Build public URL and update Logto
	avatarURL := buildAvatarURL(logtoID)
	logtoClient := logto.NewManagementClient()
	updateReq := models.UpdateUserRequest{Avatar: &avatarURL}
	if _, err := logtoClient.UpdateUser(logtoID, updateReq); err != nil {
		logger.Error().Err(err).Str("user_id", localUserID).Str("logto_id", logtoID).Msg("failed to update avatar URL in logto")
		// Avatar is saved locally, just log the Logto sync failure
	}

	// Invalidate cache
	invalidateUserProfileCache(&logtoID)

	logger.LogBusinessOperation(c, "avatar", "upload", "user", localUserID, true, nil)

	c.JSON(http.StatusOK, response.OK("avatar uploaded successfully", gin.H{
		"avatar_url": avatarURL,
	}))
}

// processAvatarDelete handles the common avatar delete logic.
func processAvatarDelete(c *gin.Context, localUserID, logtoID string) {
	repo := entities.NewLocalUserRepository()
	if err := repo.DeleteAvatar(localUserID); err != nil {
		logger.Error().Err(err).Str("user_id", localUserID).Msg("failed to delete avatar from database")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete avatar", nil))
		return
	}

	// Clear avatar URL in Logto
	emptyURL := ""
	logtoClient := logto.NewManagementClient()
	updateReq := models.UpdateUserRequest{Avatar: &emptyURL}
	if _, err := logtoClient.UpdateUser(logtoID, updateReq); err != nil {
		logger.Error().Err(err).Str("user_id", localUserID).Str("logto_id", logtoID).Msg("failed to clear avatar URL in logto")
	}

	// Invalidate cache
	invalidateUserProfileCache(&logtoID)

	logger.LogBusinessOperation(c, "avatar", "delete", "user", localUserID, true, nil)

	c.JSON(http.StatusOK, response.OK("avatar deleted successfully", nil))
}

// canManageUserAvatar checks if the current user can manage another user's avatar.
func canManageUserAvatar(currentUser *models.User, targetUser *models.LocalUser) bool {
	userOrgRole := strings.ToLower(currentUser.OrgRole)

	targetOrgID := ""
	if targetUser.OrganizationID != nil {
		targetOrgID = *targetUser.OrganizationID
	}

	service := local.NewUserService()
	canUpdate, _ := service.CanUpdateUser(userOrgRole, currentUser.OrganizationID, targetOrgID)
	return canUpdate
}

// resizeImage resizes an image to fit within maxWidth x maxHeight, maintaining aspect ratio.
func resizeImage(img image.Image, maxWidth, maxHeight int) image.Image {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	// Calculate scale factor maintaining aspect ratio
	scaleW := float64(maxWidth) / float64(srcW)
	scaleH := float64(maxHeight) / float64(srcH)
	scale := scaleW
	if scaleH < scaleW {
		scale = scaleH
	}

	newW := int(float64(srcW) * scale)
	newH := int(float64(srcH) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}

// buildAvatarURL constructs the public URL for a user's avatar.
func buildAvatarURL(logtoID string) string {
	return fmt.Sprintf("%s/api/public/avatars/%s", configuration.Config.AppURL, logtoID)
}
