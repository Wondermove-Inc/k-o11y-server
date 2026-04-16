package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/infrastructure"
	"github.com/gin-gonic/gin"
)

// ColdStorageConfig represents the cold_storage_config table row.
type ColdStorageConfig struct {
	SignalType             string `json:"signal_type" ch:"signal_type"`
	GlacierEnabled        uint8  `json:"glacier_enabled" ch:"glacier_enabled"`
	GlacierRetentionDays  int32  `json:"glacier_retention_days" ch:"glacier_retention_days"`
	BackupFrequencyHours  int32  `json:"backup_frequency_hours" ch:"backup_frequency_hours"`
	MinDeleteRetentionDays int32 `json:"min_delete_retention_days" ch:"min_delete_retention_days"`
	UpdatedBy             string `json:"updated_by" ch:"updated_by"`
	UpdatedAt             string `json:"updated_at" ch:"updated_at"`
	Version               uint64 `json:"version" ch:"version"`
}

// UpdateColdStorageConfigRequest is the request body for PUT.
type UpdateColdStorageConfigRequest struct {
	GlacierEnabled        *uint8 `json:"glacier_enabled,omitempty"`
	GlacierRetentionDays  *int32 `json:"glacier_retention_days,omitempty"`
	BackupFrequencyHours  *int32 `json:"backup_frequency_hours,omitempty"`
	MinDeleteRetentionDays *int32 `json:"min_delete_retention_days,omitempty"`
}

// GetColdStorageConfig handles GET /api/v1/settings/cold-storage.
func GetColdStorageConfig(c *gin.Context) {
	conn := infrastructure.GetClickHouseConn()
	if conn == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ClickHouse connection not available"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	row := conn.QueryRow(ctx, `
		SELECT signal_type, glacier_enabled, glacier_retention_days,
			backup_frequency_hours, min_delete_retention_days, updated_by,
			toString(updated_at) as updated_at, version
		FROM ko11y.cold_storage_config FINAL
		WHERE signal_type = 'global' LIMIT 1
	`)

	var config ColdStorageConfig
	if err := row.ScanStruct(&config); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cold storage config not found", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateColdStorageConfig handles PUT /api/v1/settings/cold-storage.
func UpdateColdStorageConfig(c *gin.Context) {
	conn := infrastructure.GetClickHouseConn()
	if conn == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ClickHouse connection not available"})
		return
	}

	var req UpdateColdStorageConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "detail": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Read current config
	row := conn.QueryRow(ctx, `
		SELECT signal_type, glacier_enabled, glacier_retention_days,
			backup_frequency_hours, min_delete_retention_days, updated_by,
			toString(updated_at) as updated_at, version
		FROM ko11y.cold_storage_config FINAL
		WHERE signal_type = 'global' LIMIT 1
	`)

	var current ColdStorageConfig
	if err := row.ScanStruct(&current); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cold storage config not found"})
		return
	}

	// Merge non-nil fields
	glacierEnabled := current.GlacierEnabled
	if req.GlacierEnabled != nil {
		glacierEnabled = *req.GlacierEnabled
	}
	glacierRetentionDays := current.GlacierRetentionDays
	if req.GlacierRetentionDays != nil {
		glacierRetentionDays = *req.GlacierRetentionDays
	}
	backupFrequencyHours := current.BackupFrequencyHours
	if req.BackupFrequencyHours != nil {
		backupFrequencyHours = *req.BackupFrequencyHours
	}
	minDeleteRetentionDays := current.MinDeleteRetentionDays
	if req.MinDeleteRetentionDays != nil {
		minDeleteRetentionDays = *req.MinDeleteRetentionDays
	}

	// Validation
	if glacierRetentionDays < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "glacier_retention_days must be >= 0 (0 = unlimited)"})
		return
	}
	if backupFrequencyHours < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "backup_frequency_hours must be >= 1"})
		return
	}
	if minDeleteRetentionDays < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "min_delete_retention_days must be >= 1"})
		return
	}

	// INSERT new version (ReplacingMergeTree deduplicates by signal_type)
	newVersion := uint64(time.Now().Unix())
	query := fmt.Sprintf(
		`INSERT INTO ko11y.cold_storage_config
		(signal_type, glacier_enabled, glacier_retention_days, backup_frequency_hours, min_delete_retention_days, updated_by, version)
		VALUES ('global', %d, %d, %d, %d, 'signoz-ui', %d)`,
		glacierEnabled, glacierRetentionDays, backupFrequencyHours, minDeleteRetentionDays, newVersion,
	)

	if err := conn.Exec(ctx, query); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update config", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cold storage config updated"})
}

// coldStorageHandler registers cold storage routes.
func coldStorageHandler() {
	settingsRouter := routeGroup.Group("/settings")
	settingsRouter.GET("/cold-storage", GetColdStorageConfig)
	settingsRouter.PUT("/cold-storage", UpdateColdStorageConfig)
}
