package batch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/config"
	"go.uber.org/zap"
)

// UsageReporter periodically collects cluster node counts from ClickHouse
// and sends them to mgmt portal for billing.
// Failed payloads are queued in memory (unbounded) and retried on the next cycle.
type UsageReporter struct {
	db         driver.Conn
	cfg        *config.UsageReporterConfig
	httpClient *http.Client
	logger     *zap.Logger

	// pending holds payloads that failed to send, retried on next cycle.
	pending []UsagePayload
	mu      sync.Mutex
}

// ClusterNodeCount holds a single ClickHouse query result row.
type ClusterNodeCount struct {
	ClusterName string `ch:"cluster_name"`
	NodeCount   uint64 `ch:"node_count"`
}

// UsagePayload is the request body sent to mgmt portal (meta/data format).
type UsagePayload struct {
	Meta UsagePayloadMeta `json:"meta"`
	Data UsagePayloadData `json:"data"`
}

// UsagePayloadMeta contains metadata for the usage report.
type UsagePayloadMeta struct {
	TenantID  string `json:"tenantId"`
	Timestamp string `json:"timestamp"`
}

// UsagePayloadData contains the actual usage data.
type UsagePayloadData struct {
	Clusters       []ClusterUsage `json:"clusters"`
	TotalNodeCount int            `json:"totalNodeCount"`
}

// ClusterUsage holds node count for a single cluster.
type ClusterUsage struct {
	ClusterName string `json:"clusterName"`
	NodeCount   int    `json:"nodeCount"`
}

// NewUsageReporter creates a new UsageReporter with dependency injection and validation.
func NewUsageReporter(
	db driver.Conn,
	cfg *config.UsageReporterConfig,
	logger *zap.Logger,
) (*UsageReporter, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	if cfg.MgmtBaseURL == "" {
		return nil, fmt.Errorf("MgmtBaseURL cannot be empty")
	}
	if cfg.LicenseKey == "" {
		return nil, fmt.Errorf("LicenseKey cannot be empty")
	}
	if cfg.TenantID == "" {
		return nil, fmt.Errorf("TenantID cannot be empty")
	}

	httpTimeout := cfg.HTTPTimeout
	if httpTimeout <= 0 {
		httpTimeout = 10 * time.Second
	}

	return &UsageReporter{
		db:  db,
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		logger:  logger,
		pending: make([]UsagePayload, 0),
	}, nil
}

// Run starts the usage reporter on a fixed schedule: every hour at :30.
// Blocks until ctx is cancelled (graceful shutdown).
func (r *UsageReporter) Run(ctx context.Context) {
	r.logger.Info("UsageReporter started (schedule: every hour at :30)",
		zap.String("mgmtBaseURL", r.cfg.MgmtBaseURL),
		zap.String("tenantID", r.cfg.TenantID))

	for {
		// Wait until the next :30
		waitDuration := r.timeUntilNextRun()
		r.logger.Info("UsageReporter waiting for next run",
			zap.Duration("wait", waitDuration),
			zap.Time("nextRun", time.Now().Add(waitDuration)))

		select {
		case <-ctx.Done():
			r.logger.Info("UsageReporter shutting down")
			return
		case <-time.After(waitDuration):
			r.runCycle(ctx)
		}
	}
}

// timeUntilNextRun calculates the duration until the next :30 mark.
func (r *UsageReporter) timeUntilNextRun() time.Duration {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 30, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(1 * time.Hour)
	}
	return next.Sub(now)
}

// runCycle executes one full cycle: send pending payloads, then collect and send current.
func (r *UsageReporter) runCycle(ctx context.Context) {
	// Step 1: Drain pending queue (oldest first)
	r.drainPending(ctx)

	// Step 2: Collect current data and send
	if err := r.collectAndSend(ctx); err != nil {
		r.logger.Error("UsageReporter send failed", zap.Error(err))
	}
}

// drainPending attempts to send all pending (previously failed) payloads.
// Stops on first failure and keeps remaining payloads for next cycle.
func (r *UsageReporter) drainPending(ctx context.Context) {
	r.mu.Lock()
	pending := r.pending
	r.pending = make([]UsagePayload, 0)
	r.mu.Unlock()

	if len(pending) == 0 {
		return
	}

	r.logger.Info("retrying pending payloads", zap.Int("count", len(pending)))

	var failedFrom int
	for i := range pending {
		if err := r.postUsage(ctx, &pending[i]); err != nil {
			r.logger.Warn("pending payload retry failed, keeping remaining",
				zap.Int("sent", i),
				zap.Int("remaining", len(pending)-i),
				zap.Error(err))
			failedFrom = i
			// Put unsent payloads back
			r.mu.Lock()
			r.pending = append(pending[failedFrom:], r.pending...)
			r.mu.Unlock()
			return
		}
		r.logger.Info("pending payload sent successfully",
			zap.String("timestamp", pending[i].Meta.Timestamp))
	}

	r.logger.Info("all pending payloads sent", zap.Int("count", len(pending)))
}

// collectAndSend queries CH for current node counts and sends to mgmt portal.
// On failure, the payload is added to the pending queue.
func (r *UsageReporter) collectAndSend(ctx context.Context) error {
	// Query ClickHouse for cluster node counts
	nodeCounts, err := r.queryNodeCounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to query node counts: %w", err)
	}

	if len(nodeCounts) == 0 {
		r.logger.Warn("no cluster nodes found in ClickHouse, skipping send")
		return nil
	}

	// Build meta/data payload
	clusters := make([]ClusterUsage, 0, len(nodeCounts))
	totalNodeCount := 0
	for _, nc := range nodeCounts {
		count := int(nc.NodeCount)
		clusters = append(clusters, ClusterUsage{
			ClusterName: nc.ClusterName,
			NodeCount:   count,
		})
		totalNodeCount += count
	}

	payload := UsagePayload{
		Meta: UsagePayloadMeta{
			TenantID:  r.cfg.TenantID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
		Data: UsagePayloadData{
			Clusters:       clusters,
			TotalNodeCount: totalNodeCount,
		},
	}

	// POST to mgmt portal
	if err := r.postUsage(ctx, &payload); err != nil {
		r.logger.Warn("send failed, queuing for next cycle",
			zap.Int("totalNodeCount", totalNodeCount),
			zap.Error(err))
		r.enqueuePending(payload)
		return err
	}

	r.logger.Info("usage report sent successfully",
		zap.Int("totalNodeCount", totalNodeCount),
		zap.Int("clusterCount", len(clusters)))

	return nil
}

// enqueuePending adds a failed payload to the pending queue (unbounded).
func (r *UsageReporter) enqueuePending(payload UsagePayload) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.pending = append(r.pending, payload)
	r.logger.Warn("payload queued for retry",
		zap.Int("pendingCount", len(r.pending)))
}

// queryNodeCounts queries ClickHouse for active node counts per cluster.
func (r *UsageReporter) queryNodeCounts(ctx context.Context) ([]ClusterNodeCount, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT
			cluster_name,
			count(DISTINCT node_name) AS node_count
		FROM signoz_traces.cluster_nodes
		FINAL
		WHERE is_active = 1
		  AND last_seen >= now() - INTERVAL 1 HOUR
		GROUP BY cluster_name
	`

	var results []ClusterNodeCount
	if err := r.db.Select(queryCtx, &results, query); err != nil {
		return nil, fmt.Errorf("ClickHouse query failed: %w", err)
	}

	return results, nil
}

// postUsage sends the usage payload to mgmt portal with exponential backoff retry.
func (r *UsageReporter) postUsage(ctx context.Context, payload *UsagePayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := r.cfg.MgmtBaseURL + "/api/v1/usage/observability"
	maxRetries := r.cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	return withRetry(ctx, maxRetries, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+r.cfg.LicenseKey)

		resp, err := r.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body)

		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			return nil
		case resp.StatusCode == 401:
			r.logger.Error("License Key authentication failed (401) - check MGMT_LICENSE_KEY",
				zap.String("licenseKey", maskLicenseKey(r.cfg.LicenseKey)))
			return &nonRetryableError{statusCode: resp.StatusCode, message: "authentication failed (401)"}
		case resp.StatusCode == 400 || resp.StatusCode == 403:
			return &nonRetryableError{statusCode: resp.StatusCode, message: fmt.Sprintf("client error (%d)", resp.StatusCode)}
		case resp.StatusCode >= 500:
			return fmt.Errorf("server error (%d)", resp.StatusCode)
		default:
			return &nonRetryableError{statusCode: resp.StatusCode, message: fmt.Sprintf("unexpected status (%d)", resp.StatusCode)}
		}
	})
}

// nonRetryableError wraps errors that should not be retried (4xx responses).
type nonRetryableError struct {
	statusCode int
	message    string
}

func (e *nonRetryableError) Error() string {
	return e.message
}

// withRetry executes fn with exponential backoff. Stops on non-retryable errors.
func withRetry(ctx context.Context, maxRetries int, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := fn(); err != nil {
			// Don't retry non-retryable errors (4xx)
			if _, ok := err.(*nonRetryableError); ok {
				return err
			}

			lastErr = err

			// Don't wait after the last attempt
			if attempt < maxRetries-1 {
				waitDuration := time.Duration(1<<uint(attempt)) * time.Second // 1s, 2s, 4s
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(waitDuration):
				}
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("all %d retries exhausted: %w", maxRetries, lastErr)
}

// maskLicenseKey masks the license key for safe logging.
func maskLicenseKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
