/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
)

// FactoryResetSystem handles POST /api/systems/:id/factory-reset - dangerous operation
func FactoryResetSystem(c *gin.Context) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	system, exists := systemsStorage[systemID]
	if !exists {
		c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
		return
	}

	// Parse confirmation request
	var request struct {
		Confirmation string `json:"confirmation" binding:"required"`
	}
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("confirmation required", err.Error()))
		return
	}

	if request.Confirmation != "FACTORY_RESET" {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid confirmation", nil))
		return
	}

	// Simulate factory reset
	system.Status = "factory_resetting"
	system.UpdatedAt = time.Now()

	logger.LogSystemOperation(c, "factory_reset", systemID, true, nil)

	c.JSON(http.StatusOK, response.OK("factory reset initiated", gin.H{
		"system_id": systemID,
		"status":    "factory_resetting",
		"warning":   "All data will be permanently lost",
	}))
}

// DestroySystem handles DELETE /api/systems/:id/destroy - permanent deletion
func DestroySystem(c *gin.Context) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	system, exists := systemsStorage[systemID]
	if !exists {
		c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
		return
	}

	// Parse confirmation request
	var request struct {
		Confirmation string `json:"confirmation" binding:"required"`
	}
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("confirmation required", err.Error()))
		return
	}

	if request.Confirmation != system.Name {
		c.JSON(http.StatusBadRequest, response.BadRequest("system name confirmation required", gin.H{"expected": system.Name}))
		return
	}

	// Permanently destroy system
	delete(systemsStorage, systemID)
	delete(subscriptionsStorage, systemID)

	logger.LogSystemOperation(c, "destroy", systemID, true, nil)

	c.JSON(http.StatusOK, response.OK("system permanently destroyed", gin.H{
		"system_id":    systemID,
		"system_name":  system.Name,
		"destroyed_at": time.Now(),
	}))
}

// GetSystemLogs handles GET /api/systems/:id/logs - audit operation
func GetSystemLogs(c *gin.Context) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	_, exists := systemsStorage[systemID]
	if !exists {
		c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
		return
	}

	// Simulate audit logs
	auditLogs := []gin.H{
		{
			"timestamp": time.Now().Add(-2 * time.Hour),
			"level":     "INFO",
			"message":   "System started successfully",
			"user":      "system",
		},
		{
			"timestamp": time.Now().Add(-1 * time.Hour),
			"level":     "WARN",
			"message":   "High CPU usage detected",
			"user":      "monitoring",
		},
		{
			"timestamp": time.Now().Add(-30 * time.Minute),
			"level":     "INFO",
			"message":   "Backup completed successfully",
			"user":      "backup-service",
		},
	}

	logger.RequestLogger(c, "systems").Info().
		Str("operation", "access_logs").
		Str("system_id", systemID).
		Msg("System logs accessed")

	c.JSON(http.StatusOK, response.OK("system logs retrieved", gin.H{
		"system_id": systemID,
		"logs":      auditLogs,
		"count":     len(auditLogs),
	}))
}

// GetSystemsAudit handles GET /api/systems/audit - global audit
func GetSystemsAudit(c *gin.Context) {
	// Simulate global audit data
	auditData := gin.H{
		"total_systems":       len(systemsStorage),
		"online_systems":      1,
		"offline_systems":     1,
		"maintenance_systems": 0,
		"last_audit":          time.Now(),
		"compliance_score":    85.5,
		"critical_alerts":     2,
		"warnings":            7,
	}

	logger.RequestLogger(c, "systems").Info().
		Str("operation", "access_audit").
		Msg("Global systems audit accessed")

	c.JSON(http.StatusOK, response.OK("systems audit data retrieved", auditData))
}

// BackupSystem handles POST /api/systems/:id/backup - backup operation
func BackupSystem(c *gin.Context) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	_, exists := systemsStorage[systemID]
	if !exists {
		c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
		return
	}

	// Parse backup options
	var request struct {
		Type        string `json:"type" binding:"required"` // full, incremental, differential
		Compression bool   `json:"compression"`
		Encryption  bool   `json:"encryption"`
	}
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("backup options required", err.Error()))
		return
	}

	// Simulate backup process
	backupID := "backup-" + systemID + "-" + time.Now().Format("20060102-150405")

	logger.LogSystemOperation(c, "backup", systemID, true, nil)

	c.JSON(http.StatusOK, response.OK("backup initiated", gin.H{
		"backup_id":          backupID,
		"system_id":          systemID,
		"type":               request.Type,
		"compression":        request.Compression,
		"encryption":         request.Encryption,
		"estimated_duration": "15-30 minutes",
	}))
}

// RestoreSystem handles POST /api/systems/:id/restore - restore operation
func RestoreSystem(c *gin.Context) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	system, exists := systemsStorage[systemID]
	if !exists {
		c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
		return
	}

	// Parse restore options
	var request struct {
		BackupID string `json:"backup_id" binding:"required"`
		Force    bool   `json:"force"`
	}
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("restore options required", err.Error()))
		return
	}

	// Simulate restore process
	system.Status = "restoring"
	system.UpdatedAt = time.Now()

	logger.LogSystemOperation(c, "restore", systemID, true, nil)

	c.JSON(http.StatusOK, response.OK("restore initiated", gin.H{
		"system_id":          systemID,
		"backup_id":          request.BackupID,
		"status":             "restoring",
		"estimated_duration": "30-60 minutes",
		"warning":            "Current data will be overwritten",
	}))
}
