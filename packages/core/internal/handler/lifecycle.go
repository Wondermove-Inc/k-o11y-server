package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/infrastructure"
	"github.com/gin-gonic/gin"
)

// DataLifecycleConfig represents the data_lifecycle_config table row.
type DataLifecycleConfig struct {
	SignalType           string `json:"signal_type" ch:"signal_type"`
	HotDays              int32  `json:"hot_days" ch:"hot_days"`
	WarmDays             int32  `json:"warm_days" ch:"warm_days"`
	GlacierEnabled       uint8  `json:"glacier_enabled" ch:"glacier_enabled"`
	GlacierRetentionDays int32  `json:"glacier_retention_days" ch:"glacier_retention_days"`
	BackupFrequencyHours int32  `json:"backup_frequency_hours" ch:"backup_frequency_hours"`
	LastBackupStatus     string `json:"last_backup_status" ch:"last_backup_status"`
	LastBackupAt         string `json:"last_backup_at" ch:"last_backup_at"`
	LastBackupError      string `json:"last_backup_error" ch:"last_backup_error"`
	UpdatedBy            string `json:"updated_by" ch:"updated_by"`
	UpdatedAt            string `json:"updated_at" ch:"updated_at"`
	Version              uint64 `json:"version" ch:"version"`
}

// UpdateLifecycleConfigRequest is the request body for PUT.
type UpdateLifecycleConfigRequest struct {
	HotDays              *int32 `json:"hot_days,omitempty"`
	WarmDays             *int32 `json:"warm_days,omitempty"`
	GlacierEnabled       *uint8 `json:"glacier_enabled,omitempty"`
	GlacierRetentionDays *int32 `json:"glacier_retention_days,omitempty"`
}

// GetLifecycleConfig handles GET /api/v1/settings/lifecycle.
func GetLifecycleConfig(c *gin.Context) {
	conn := infrastructure.GetClickHouseConn()
	if conn == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ClickHouse connection not available"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	row := conn.QueryRow(ctx, `
		SELECT signal_type, hot_days, warm_days,
			glacier_enabled, glacier_retention_days, backup_frequency_hours,
			last_backup_status, toString(last_backup_at) as last_backup_at,
			last_backup_error, updated_by, toString(updated_at) as updated_at, version
		FROM ko11y.data_lifecycle_config FINAL
		WHERE signal_type = 'global' LIMIT 1
	`)

	var config DataLifecycleConfig
	if err := row.ScanStruct(&config); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lifecycle config not found", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateLifecycleConfig handles PUT /api/v1/settings/lifecycle.
func UpdateLifecycleConfig(c *gin.Context) {
	conn := infrastructure.GetClickHouseConn()
	if conn == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ClickHouse connection not available"})
		return
	}

	var req UpdateLifecycleConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "detail": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Read current config
	row := conn.QueryRow(ctx, `
		SELECT signal_type, hot_days, warm_days,
			glacier_enabled, glacier_retention_days, backup_frequency_hours,
			last_backup_status, toString(last_backup_at) as last_backup_at,
			last_backup_error, updated_by, toString(updated_at) as updated_at, version
		FROM ko11y.data_lifecycle_config FINAL
		WHERE signal_type = 'global' LIMIT 1
	`)

	var current DataLifecycleConfig
	if err := row.ScanStruct(&current); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lifecycle config not found"})
		return
	}

	// Merge non-nil fields
	hotDays := current.HotDays
	if req.HotDays != nil {
		hotDays = *req.HotDays
	}
	warmDays := current.WarmDays
	if req.WarmDays != nil {
		warmDays = *req.WarmDays
	}
	glacierEnabled := current.GlacierEnabled
	if req.GlacierEnabled != nil {
		glacierEnabled = *req.GlacierEnabled
	}
	glacierRetentionDays := current.GlacierRetentionDays
	if req.GlacierRetentionDays != nil {
		glacierRetentionDays = *req.GlacierRetentionDays
	}

	// Validation
	if hotDays < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hot_days must be >= 1"})
		return
	}
	if warmDays < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "warm_days must be >= 1"})
		return
	}
	if glacierRetentionDays < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "glacier_retention_days must be >= 0 (0 = unlimited)"})
		return
	}

	// INSERT new version (ReplacingMergeTree deduplicates by signal_type)
	newVersion := uint64(time.Now().Unix())
	query := fmt.Sprintf(
		`INSERT INTO ko11y.data_lifecycle_config
		(signal_type, hot_days, warm_days, glacier_enabled, glacier_retention_days,
		 backup_frequency_hours, updated_by, version)
		VALUES ('global', %d, %d, %d, %d, %d, 'o11y-core', %d)`,
		hotDays, warmDays, glacierEnabled, glacierRetentionDays,
		current.BackupFrequencyHours, newVersion,
	)

	if err := conn.Exec(ctx, query); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update config", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "lifecycle config updated"})
}

// lifecycleHandler registers lifecycle routes.
func lifecycleHandler() {
	settingsRouter := routeGroup.Group("/settings")
	settingsRouter.GET("/lifecycle", GetLifecycleConfig)
	settingsRouter.PUT("/lifecycle", UpdateLifecycleConfig)

	// Legacy: cold-storage API 하위 호환 (deprecated)
	settingsRouter.GET("/cold-storage", GetColdStorageConfig)
	settingsRouter.PUT("/cold-storage", UpdateColdStorageConfig)
}
