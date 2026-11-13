/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

// maxNoteLength mirrors the support_session_notes body constraint
const maxNoteLength = 10000

// systemInScope answers 400/404/500 itself and reports whether to go on
func systemInScope(c *gin.Context, repo *entities.SupportRepository, systemID string) bool {
	if systemID == "" || len(systemID) > 255 {
		c.JSON(http.StatusBadRequest, response.BadRequest("system_id is required", nil))
		return false
	}
	_, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	found, err := repo.SystemInScope(systemID, userOrgRole, userOrgID)
	if err != nil {
		logger.Error().Err(err).Str("system_id", systemID).Msg("failed to check system scope")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to check system", nil))
		return false
	}
	if !found {
		c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
		return false
	}
	return true
}

// noteBody trims a note and checks its length, answering 400 when invalid
func noteBody(c *gin.Context, body string) (string, bool) {
	body = strings.TrimSpace(body)
	if body == "" || utf8.RuneCountInString(body) > maxNoteLength || !utf8.ValidString(body) {
		c.JSON(http.StatusBadRequest, response.BadRequest("note must be between 1 and 10000 characters", nil))
		return "", false
	}
	return body, true
}

// GetSupportActivity handles GET /api/support-sessions/activity?system_id=
func GetSupportActivity(c *gin.Context) {
	systemID := c.Query("system_id")
	repo := entities.NewSupportRepository()
	if !systemInScope(c, repo, systemID) {
		return
	}

	activity, err := repo.GetSystemActivity(systemID)
	if err != nil {
		logger.Error().Err(err).Str("system_id", systemID).Msg("failed to get support activity")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get support activity", nil))
		return
	}

	liveSessions := []string{}
	for _, s := range activity.Sessions {
		if s.Status == "active" || s.Status == "pending" {
			liveSessions = append(liveSessions, s.ID)
		}
	}
	activity.WebPresence = []models.SupportWebPresence{}
	for _, p := range cache.GetSupportPresence(liveSessions) {
		activity.WebPresence = append(activity.WebPresence, models.SupportWebPresence(p))
	}

	c.JSON(http.StatusOK, response.OK("support activity retrieved successfully", activity))
}

// GetSupportNotes handles GET /api/support-notes?system_id=
func GetSupportNotes(c *gin.Context) {
	systemID := c.Query("system_id")
	repo := entities.NewSupportRepository()
	if !systemInScope(c, repo, systemID) {
		return
	}

	notes, err := repo.GetSystemNotes(systemID)
	if err != nil {
		logger.Error().Err(err).Str("system_id", systemID).Msg("failed to get support notes")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get support notes", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("support notes retrieved successfully", gin.H{"notes": notes}))
}

// CreateSupportNote handles POST /api/support-notes
func CreateSupportNote(c *gin.Context) {
	var request models.SupportSessionNoteRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}
	body, ok := noteBody(c, request.Body)
	if !ok {
		return
	}
	repo := entities.NewSupportRepository()
	if !systemInScope(c, repo, request.SystemID) {
		return
	}

	userID, _, _, _ := helpers.GetUserContextExtended(c)
	userName := ""
	if name, exists := c.Get("name"); exists {
		userName, _ = name.(string)
	}
	note, err := repo.CreateNote(request.SystemID, userID, userName, body)
	if err != nil {
		logger.Error().Err(err).Str("system_id", request.SystemID).Msg("failed to create support note")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create support note", nil))
		return
	}
	if note == nil {
		c.JSON(http.StatusNotFound, response.NotFound("the system has no support session", nil))
		return
	}

	logger.LogBusinessOperation(c, "support", "create", "note", note.ID, true, nil)
	c.JSON(http.StatusCreated, response.Created("support note created successfully", note))
}

// ownNote loads a note the caller may change: in scope and written by them
func ownNote(c *gin.Context, repo *entities.SupportRepository) (*models.SupportSessionNote, string, bool) {
	noteID := c.Param("note_id")
	if _, err := uuid.Parse(noteID); err != nil {
		c.JSON(http.StatusNotFound, response.NotFound("note not found", nil))
		return nil, "", false
	}
	note, err := repo.GetNote(noteID)
	if err != nil {
		logger.Error().Err(err).Str("note_id", noteID).Msg("failed to get support note")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get support note", nil))
		return nil, "", false
	}
	if note == nil {
		c.JSON(http.StatusNotFound, response.NotFound("note not found", nil))
		return nil, "", false
	}
	if !systemInScope(c, repo, note.SystemID) {
		return nil, "", false
	}
	userID, _, _, _ := helpers.GetUserContextExtended(c)
	if note.AuthorID != userID {
		c.JSON(http.StatusForbidden, response.Forbidden("only the author can change a note", nil))
		return nil, "", false
	}
	return note, userID, true
}

// UpdateSupportNote handles PUT /api/support-notes/:note_id
func UpdateSupportNote(c *gin.Context) {
	var request models.SupportSessionNoteRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}
	body, ok := noteBody(c, request.Body)
	if !ok {
		return
	}
	repo := entities.NewSupportRepository()
	note, userID, ok := ownNote(c, repo)
	if !ok {
		return
	}

	updated, err := repo.UpdateNote(note.ID, userID, body)
	if err != nil {
		logger.Error().Err(err).Str("note_id", note.ID).Msg("failed to update support note")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update support note", nil))
		return
	}
	if updated == nil {
		c.JSON(http.StatusNotFound, response.NotFound("note not found", nil))
		return
	}

	logger.LogBusinessOperation(c, "support", "update", "note", note.ID, true, nil)
	c.JSON(http.StatusOK, response.OK("support note updated successfully", updated))
}

// DeleteSupportNote handles DELETE /api/support-notes/:note_id
func DeleteSupportNote(c *gin.Context) {
	repo := entities.NewSupportRepository()
	note, userID, ok := ownNote(c, repo)
	if !ok {
		return
	}

	deleted, err := repo.DeleteNote(note.ID, userID)
	if err != nil {
		logger.Error().Err(err).Str("note_id", note.ID).Msg("failed to delete support note")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete support note", nil))
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, response.NotFound("note not found", nil))
		return
	}

	logger.LogBusinessOperation(c, "support", "delete", "note", note.ID, true, nil)
	c.JSON(http.StatusOK, response.OK("support note deleted successfully", gin.H{"id": note.ID}))
}

// workspaceEvents are the access log entries only the browser can see: a
// login form filled with the session credentials (they never leave the
// browser) and a web interface tab closed
var workspaceEvents = map[string]bool{"autofill": true, "ui_close": true}

// LogSupportWorkspaceEvent handles POST /api/support-sessions/:id/events
func LogSupportWorkspaceEvent(c *gin.Context) {
	sessionID := c.Param("id")
	if getActiveSession(c, sessionID) == nil {
		return
	}
	var request struct {
		Type    string `json:"type" binding:"required"`
		Service string `json:"service" binding:"required"`
	}
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}
	if !workspaceEvents[request.Type] {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid event type", nil))
		return
	}
	if len(request.Service) > 100 || !validServiceName.MatchString(request.Service) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid service name", nil))
		return
	}
	logAccessWithMetadata(c, sessionID, request.Type, map[string]string{"service": request.Service})
	c.JSON(http.StatusOK, response.OK("event recorded", nil))
}

// GetSupportSessionSavedServices handles GET /api/support-sessions/:id/saved-services:
// the services operators added to the session's node, who added them and
// until when they are remembered
func GetSupportSessionSavedServices(c *gin.Context) {
	sessionID := c.Param("id")
	_, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	repo := entities.NewSupportRepository()
	sess, err := repo.GetSessionByID(sessionID, userOrgRole, userOrgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get support session", nil))
		return
	}
	if sess == nil {
		c.JSON(http.StatusNotFound, response.NotFound("support session not found", nil))
		return
	}
	services, err := repo.GetSessionSavedServices(sessionID)
	if err != nil {
		logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to list saved services")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list saved services", nil))
		return
	}
	c.JSON(http.StatusOK, response.OK("saved services", gin.H{"services": services}))
}
