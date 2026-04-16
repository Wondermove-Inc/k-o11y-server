package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/infrastructure"
)

type Config struct {
	Server        ServerConfig
	Logging       LoggingConfig
	ClickHouse    infrastructure.ClickHouseConfig
	Batch         BatchConfig
	UsageReporter UsageReporterConfig
	ENV           string
}

// UsageReporterConfig contains configuration for the Usage Reporter module.
// Usage Reporter periodically collects cluster node counts from ClickHouse
// and sends them to mgmt portal for billing.
type UsageReporterConfig struct {
	// Enabled controls whether Usage Reporter is active.
	// Environment variable: USAGE_REPORTER_ENABLED (default: false)
	Enabled bool

	// MgmtBaseURL is the mgmt portal base URL (must be HTTPS in production).
	// Environment variable: MGMT_BASE_URL
	MgmtBaseURL string

	// LicenseKey is the Bearer token for mgmt portal authentication.
	// MUST NEVER be logged in plaintext.
	// Environment variable: MGMT_LICENSE_KEY
	LicenseKey string

	// TenantID is included in meta.tenantId of the usage payload.
	// Auto-extracted from LicenseKey JWT if not set.
	// Environment variable: MGMT_TENANT_ID (optional, overrides JWT)
	TenantID string

	// Interval is the collection/send interval.
	// Environment variable: USAGE_REPORTER_INTERVAL (default: 1h)
	Interval time.Duration

	// HTTPTimeout is the HTTP request timeout for mgmt portal API calls.
	// Environment variable: USAGE_REPORTER_HTTP_TIMEOUT (default: 10s)
	HTTPTimeout time.Duration

	// MaxRetries is the maximum number of retry attempts for failed HTTP requests.
	// Environment variable: USAGE_REPORTER_MAX_RETRIES (default: 3)
	MaxRetries int
}

type ServerConfig struct {
	Port int
}

type LoggingConfig struct {
	Level string
	File  string
}

// BatchConfig contains configuration for batch processing operations.
// Batch processing is used for aggregating and processing servicemap data periodically.
type BatchConfig struct {
	// ServiceMapEnabled controls whether batch servicemap processing is enabled.
	// Environment variable: BATCH_SERVICEMAP_ENABLED
	// Default: true
	ServiceMapEnabled bool

	// ServiceMapInterval specifies how often batch servicemap processing runs.
	// Must be greater than 0 if ServiceMapEnabled is true.
	// Environment variable: BATCH_SERVICEMAP_INTERVAL
	// Default: 20s
	// Examples: "20s", "1m", "30s"
	ServiceMapInterval time.Duration

	// InsertTimeout specifies the maximum duration for INSERT query execution.
	// The INSERT query is complex with multiple Dictionary lookups and quantile calculations.
	// Environment variable: BATCH_INSERT_TIMEOUT
	// Default: 120s
	// Examples: "120s", "5m", "300s"
	InsertTimeout time.Duration

	// SafetyBuffer is the time buffer subtracted from now() to wait for data stabilization.
	// Ensures late-arriving data is included before processing.
	// Environment variable: BATCH_SAFETY_BUFFER
	// Default: 20s
	SafetyBuffer time.Duration

	// MaxWindow limits the maximum time range processed in a single batch.
	// Prevents snowball effect where accumulated data causes cascading delays.
	// When gap > MaxWindow, catch-up mode processes MaxWindow-sized chunks sequentially.
	// Environment variable: BATCH_MAX_WINDOW
	// Default: 30s
	MaxWindow time.Duration
}

const (
	ENV_LOCAL = "local"
	ENV_DEV   = "dev"
	ENV_STG   = "stg"
	ENV_PROD  = "prod"
)

var globalConfig *Config = nil

func LoadConfig() (*Config, error) {
	if globalConfig != nil {
		return globalConfig, nil
	}

	env := getEnv("APP_ENV", ENV_LOCAL)
	fmt.Printf("\033[1;41;37mENV: %s\033[0m\n", env)

	// Load batch configuration
	batchConfig, err := loadBatchConfig()
	if err != nil {
		return nil, err
	}

	// Load usage reporter configuration
	usageReporterConfig, err := loadUsageReporterConfig()
	if err != nil {
		return nil, err
	}

	config := Config{
		ENV: env,
		Server: ServerConfig{
			Port: getEnvInt("APP_PORT", 3001),
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
			File:  getEnv("LOG_FILE", "./logs/local-ko11y.log"),
		},
		ClickHouse: infrastructure.ClickHouseConfig{
			Host:       getEnv("CLICKHOUSE_HOST", ""),
			Port:       getEnvInt("CLICKHOUSE_PORT", 0),
			Database:   getEnv("CLICKHOUSE_DATABASE", ""),
			Username:   getEnv("CLICKHOUSE_USER", ""),
			Password:   getEnv("CLICKHOUSE_PASSWORD", ""),
			Timeout:    getEnv("CLICKHOUSE_TIMEOUT", "10s"),
			MaxRetries: getEnvInt("CLICKHOUSE_MAX_RETRIES", 3),
		},
		Batch:         batchConfig,
		UsageReporter: usageReporterConfig,
	}

	if missing := ValidateRequiredConfigs(&config); len(missing) > 0 {
		return nil, fmt.Errorf("missing required configs: %v", missing)
	}

	globalConfig = &config
	return &config, nil
}

func (cfg *Config) GetEnv() string {
	return cfg.ENV
}

func (cfg *Config) GetClickHouseConfig() *infrastructure.ClickHouseConfig {
	return &cfg.ClickHouse
}

// GetBatchConfig returns a pointer to the batch configuration.
func (cfg *Config) GetBatchConfig() *BatchConfig {
	return &cfg.Batch
}

// GetUsageReporterConfig returns a pointer to the usage reporter configuration.
func (cfg *Config) GetUsageReporterConfig() *UsageReporterConfig {
	return &cfg.UsageReporter
}

// config 설정 검증 - startup probe
func ValidateRequiredConfigs(cfg *Config) []string {
	missingConfigs := []string{}

	// 서버 설정 검증
	if cfg.Server.Port == 0 {
		missingConfigs = append(missingConfigs, "server.port")
	}

	// ClickHouse 설정 검증
	if cfg.ClickHouse.Host == "" {
		missingConfigs = append(missingConfigs, "clickhouse.host")
	}
	if cfg.ClickHouse.Port == 0 {
		missingConfigs = append(missingConfigs, "clickhouse.port")
	}
	if cfg.ClickHouse.Database == "" {
		missingConfigs = append(missingConfigs, "clickhouse.database")
	}

	// Batch 설정 검증 (enabled인 경우에만)
	if cfg.Batch.ServiceMapEnabled {
		if cfg.Batch.ServiceMapInterval <= 0 {
			missingConfigs = append(missingConfigs, "batch.servicemap_interval")
		}
	}

	return missingConfigs
}

// loadBatchConfig loads batch configuration from environment variables.
// It validates the interval value and ensures it's greater than 0 if batch is enabled.
//
// Environment variables:
//   - BATCH_SERVICEMAP_ENABLED: Enable/disable batch servicemap processing (default: false)
//   - BATCH_SERVICEMAP_INTERVAL: Interval for batch processing (default: 20s)
//
// Returns an error if:
//   - BATCH_SERVICEMAP_INTERVAL is set but has an invalid format
//   - ServiceMapEnabled is true but ServiceMapInterval <= 0
func loadBatchConfig() (BatchConfig, error) {
	cfg := BatchConfig{
		ServiceMapEnabled: getEnvBool("BATCH_SERVICEMAP_ENABLED", true),
	}

	// Parse interval with explicit error handling to catch malformed values
	intervalStr := os.Getenv("BATCH_SERVICEMAP_INTERVAL")
	if intervalStr != "" {
		interval, err := time.ParseDuration(intervalStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid batch servicemap interval: %w", err)
		}
		cfg.ServiceMapInterval = interval
	} else {
		// Use default interval if not specified
		cfg.ServiceMapInterval = 20 * time.Second
	}

	// Parse INSERT timeout
	timeoutStr := os.Getenv("BATCH_INSERT_TIMEOUT")
	if timeoutStr != "" {
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid batch insert timeout: %w", err)
		}
		cfg.InsertTimeout = timeout
	} else {
		cfg.InsertTimeout = 120 * time.Second
	}

	// Parse safety buffer
	safetyStr := os.Getenv("BATCH_SAFETY_BUFFER")
	if safetyStr != "" {
		safety, err := time.ParseDuration(safetyStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid batch safety buffer: %w", err)
		}
		cfg.SafetyBuffer = safety
	} else {
		cfg.SafetyBuffer = 20 * time.Second
	}

	// Parse max window (bounded window for snowball prevention)
	maxWindowStr := os.Getenv("BATCH_MAX_WINDOW")
	if maxWindowStr != "" {
		maxWindow, err := time.ParseDuration(maxWindowStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid batch max window: %w", err)
		}
		cfg.MaxWindow = maxWindow
	} else {
		cfg.MaxWindow = 30 * time.Second
	}

	// Validate interval if batch is enabled
	if cfg.ServiceMapEnabled && cfg.ServiceMapInterval <= 0 {
		return cfg, fmt.Errorf("batch servicemap interval must be greater than 0")
	}

	return cfg, nil
}

// loadUsageReporterConfig loads usage reporter configuration from environment variables.
func loadUsageReporterConfig() (UsageReporterConfig, error) {
	cfg := UsageReporterConfig{
		Enabled:     getEnvBool("USAGE_REPORTER_ENABLED", false),
		MgmtBaseURL: getEnv("MGMT_BASE_URL", "https://<YOUR_MGMT_PORTAL_URL>"),
		LicenseKey:  getEnv("MGMT_LICENSE_KEY", ""),
		TenantID:    getEnv("MGMT_TENANT_ID", ""),
		MaxRetries:  getEnvInt("USAGE_REPORTER_MAX_RETRIES", 3),
	}

	// Parse interval
	intervalStr := os.Getenv("USAGE_REPORTER_INTERVAL")
	if intervalStr != "" {
		interval, err := time.ParseDuration(intervalStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid usage reporter interval: %w", err)
		}
		cfg.Interval = interval
	} else {
		cfg.Interval = 1 * time.Hour
	}

	// Parse HTTP timeout
	timeoutStr := os.Getenv("USAGE_REPORTER_HTTP_TIMEOUT")
	if timeoutStr != "" {
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid usage reporter http timeout: %w", err)
		}
		cfg.HTTPTimeout = timeout
	} else {
		cfg.HTTPTimeout = 10 * time.Second
	}

	// Validate required fields when enabled
	if cfg.Enabled {
		if cfg.LicenseKey == "" {
			return cfg, fmt.Errorf("MGMT_LICENSE_KEY is required when USAGE_REPORTER_ENABLED=true")
		}
		// Auto-extract tenant_id from license key JWT if not explicitly set
		if cfg.TenantID == "" {
			tenantID, err := extractTenantFromJWT(cfg.LicenseKey)
			if err != nil {
				return cfg, fmt.Errorf("MGMT_TENANT_ID not set and failed to extract from license key JWT: %w", err)
			}
			cfg.TenantID = tenantID
		}
		if cfg.Interval <= 0 {
			return cfg, fmt.Errorf("usage reporter interval must be greater than 0")
		}
	}

	return cfg, nil
}

// extractTenantFromJWT extracts tenant_id from a JWT token's payload without verifying the signature.
// The JWT is expected to have the format: header.payload.signature
// The payload must contain a "tenant_id" field.
func extractTenantFromJWT(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	// Add padding if needed for base64 decoding
	payload := parts[1]
	if m := len(payload) % 4; m != 0 {
		payload += strings.Repeat("=", 4-m)
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return "", fmt.Errorf("failed to parse JWT payload: %w", err)
	}

	if claims.TenantID == "" {
		return "", fmt.Errorf("tenant_id not found in JWT payload")
	}

	return claims.TenantID, nil
}

// 간단한 env helper들 (optimization 콘솔 패턴과 정렬)
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool retrieves a boolean environment variable with a fallback default.
// Accepts: "1", "t", "T", "true", "TRUE", "True", "0", "f", "F", "false", "FALSE", "False"
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
