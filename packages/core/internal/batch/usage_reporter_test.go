package batch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ============================================================
// mockSelectDB: extends mockDB with Select support
// ============================================================

type mockSelectDB struct {
	mockDB
	selectResult interface{}
	selectError  error
}

func (m *mockSelectDB) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	if m.selectError != nil {
		return m.selectError
	}
	if m.selectResult != nil {
		if results, ok := m.selectResult.([]ClusterNodeCount); ok {
			if destPtr, ok := dest.(*[]ClusterNodeCount); ok {
				*destPtr = results
			}
		}
	}
	return nil
}

// ============================================================
// NewUsageReporter Tests
// ============================================================

func TestNewUsageReporter_Success(t *testing.T) {
	t.Run("should create reporter with valid parameters", func(t *testing.T) {
		reporter, err := NewUsageReporter(
			&mockSelectDB{},
			&config.UsageReporterConfig{
				MgmtBaseURL: "https://api.mgmt.test",
				LicenseKey:  "test-key-123456",
				TenantID:    "0000000001",
				HTTPTimeout: 10 * time.Second,
			},
			zap.NewNop(),
		)

		assert.NoError(t, err)
		assert.NotNil(t, reporter)
		assert.Empty(t, reporter.pending)
	})
}

func TestNewUsageReporter_ValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		db        interface{ Select(context.Context, interface{}, string, ...interface{}) error }
		cfg       *config.UsageReporterConfig
		logger    *zap.Logger
		errorMsg  string
	}{
		{
			name:   "nil DB",
			db:     nil,
			cfg:    &config.UsageReporterConfig{MgmtBaseURL: "https://x", LicenseKey: "k", TenantID: "t"},
			logger: zap.NewNop(),
			errorMsg: "database connection cannot be nil",
		},
		{
			name:   "nil config",
			cfg:    nil,
			logger: zap.NewNop(),
			errorMsg: "config cannot be nil",
		},
		{
			name:   "empty MgmtBaseURL",
			cfg:    &config.UsageReporterConfig{MgmtBaseURL: "", LicenseKey: "k", TenantID: "t"},
			logger: zap.NewNop(),
			errorMsg: "MgmtBaseURL cannot be empty",
		},
		{
			name:   "empty LicenseKey",
			cfg:    &config.UsageReporterConfig{MgmtBaseURL: "https://x", LicenseKey: "", TenantID: "t"},
			logger: zap.NewNop(),
			errorMsg: "LicenseKey cannot be empty",
		},
		{
			name:   "empty TenantID",
			cfg:    &config.UsageReporterConfig{MgmtBaseURL: "https://x", LicenseKey: "k", TenantID: ""},
			logger: zap.NewNop(),
			errorMsg: "TenantID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var db interface{} = tt.db
			if db == nil && tt.name == "nil DB" {
				_, err := NewUsageReporter(nil, tt.cfg, tt.logger)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}
			if tt.cfg == nil {
				_, err := NewUsageReporter(&mockSelectDB{}, nil, tt.logger)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}
			_, err := NewUsageReporter(&mockSelectDB{}, tt.cfg, tt.logger)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

// ============================================================
// queryNodeCounts Tests
// ============================================================

func TestQueryNodeCounts_Success(t *testing.T) {
	t.Run("should return node counts from ClickHouse", func(t *testing.T) {
		reporter := &UsageReporter{
			db: &mockSelectDB{
				selectResult: []ClusterNodeCount{
					{ClusterName: "<YOUR_CLUSTER>", NodeCount: 4},
					{ClusterName: "<YOUR_CLUSTER>", NodeCount: 6},
				},
			},
			logger: zap.NewNop(),
		}

		results, err := reporter.queryNodeCounts(context.Background())

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "<YOUR_CLUSTER>", results[0].ClusterName)
		assert.Equal(t, uint64(4), results[0].NodeCount)
	})
}

func TestQueryNodeCounts_Empty(t *testing.T) {
	t.Run("should return empty slice when no nodes exist", func(t *testing.T) {
		reporter := &UsageReporter{
			db:     &mockSelectDB{selectResult: []ClusterNodeCount{}},
			logger: zap.NewNop(),
		}

		results, err := reporter.queryNodeCounts(context.Background())

		assert.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestQueryNodeCounts_DBError(t *testing.T) {
	t.Run("should return error when ClickHouse query fails", func(t *testing.T) {
		reporter := &UsageReporter{
			db:     &mockSelectDB{selectError: fmt.Errorf("connection refused")},
			logger: zap.NewNop(),
		}

		results, err := reporter.queryNodeCounts(context.Background())

		assert.Error(t, err)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "ClickHouse query failed")
	})
}

// ============================================================
// postUsage Tests (httptest)
// ============================================================

func TestPostUsage_Success(t *testing.T) {
	t.Run("should send payload and receive 200 OK", func(t *testing.T) {
		var receivedBody UsagePayload
		var receivedAuth string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAuth = r.Header.Get("Authorization")
			json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		reporter := &UsageReporter{
			cfg: &config.UsageReporterConfig{
				MgmtBaseURL: server.URL,
				LicenseKey:  "sk-live-test1234",
				MaxRetries:  3,
			},
			httpClient: &http.Client{Timeout: 5 * time.Second},
			logger:     zap.NewNop(),
		}

		payload := &UsagePayload{
			Meta: UsagePayloadMeta{TenantID: "0000000001", Timestamp: "2026-03-04T15:00:00Z"},
			Data: UsagePayloadData{
				Clusters:       []ClusterUsage{{ClusterName: "<YOUR_CLUSTER>", NodeCount: 4}},
				TotalNodeCount: 4,
			},
		}

		err := reporter.postUsage(context.Background(), payload)

		assert.NoError(t, err)
		assert.Equal(t, "Bearer sk-live-test1234", receivedAuth)
		assert.Equal(t, "0000000001", receivedBody.Meta.TenantID)
		assert.Equal(t, 4, receivedBody.Data.TotalNodeCount)
	})
}

func TestPostUsage_401_NoRetry(t *testing.T) {
	t.Run("should not retry on 401 Unauthorized", func(t *testing.T) {
		var callCount int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&callCount, 1)
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		reporter := &UsageReporter{
			cfg: &config.UsageReporterConfig{
				MgmtBaseURL: server.URL,
				LicenseKey:  "bad-key-12345678",
				MaxRetries:  3,
			},
			httpClient: &http.Client{Timeout: 5 * time.Second},
			logger:     zap.NewNop(),
		}

		err := reporter.postUsage(context.Background(), &UsagePayload{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed (401)")
		assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
	})
}

func TestPostUsage_503_RetryThenSuccess(t *testing.T) {
	t.Run("should retry on 503 and succeed on second attempt", func(t *testing.T) {
		var callCount int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if atomic.AddInt32(&callCount, 1) == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		reporter := &UsageReporter{
			cfg: &config.UsageReporterConfig{
				MgmtBaseURL: server.URL,
				LicenseKey:  "test-key",
				MaxRetries:  3,
			},
			httpClient: &http.Client{Timeout: 5 * time.Second},
			logger:     zap.NewNop(),
		}

		err := reporter.postUsage(context.Background(), &UsagePayload{})

		assert.NoError(t, err)
		assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
	})
}

func TestPostUsage_503_AllRetriesExhausted(t *testing.T) {
	t.Run("should fail after all retries exhausted", func(t *testing.T) {
		var callCount int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&callCount, 1)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		reporter := &UsageReporter{
			cfg: &config.UsageReporterConfig{
				MgmtBaseURL: server.URL,
				LicenseKey:  "test-key",
				MaxRetries:  2,
			},
			httpClient: &http.Client{Timeout: 5 * time.Second},
			logger:     zap.NewNop(),
		}

		err := reporter.postUsage(context.Background(), &UsagePayload{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "all 2 retries exhausted")
		assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
	})
}

// ============================================================
// collectAndSend Tests (mock DB + httptest)
// ============================================================

func TestCollectAndSend_FullFlow(t *testing.T) {
	t.Run("should query CH and send to mgmt portal successfully", func(t *testing.T) {
		var receivedBody UsagePayload

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		reporter := &UsageReporter{
			db: &mockSelectDB{
				selectResult: []ClusterNodeCount{
					{ClusterName: "<YOUR_CLUSTER>", NodeCount: 4},
					{ClusterName: "<YOUR_CLUSTER>", NodeCount: 6},
				},
			},
			cfg: &config.UsageReporterConfig{
				MgmtBaseURL: server.URL,
				LicenseKey:  "test-key",
				TenantID:    "0000000001",
				MaxRetries:  3,
			},
			httpClient: &http.Client{Timeout: 5 * time.Second},
			logger:     zap.NewNop(),
			pending:    make([]UsagePayload, 0),
		}

		err := reporter.collectAndSend(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, "0000000001", receivedBody.Meta.TenantID)
		assert.Len(t, receivedBody.Data.Clusters, 2)
		assert.Equal(t, 10, receivedBody.Data.TotalNodeCount)
		assert.Empty(t, reporter.pending)
	})
}

func TestCollectAndSend_EmptyNodes(t *testing.T) {
	t.Run("should skip send when no nodes found", func(t *testing.T) {
		reporter := &UsageReporter{
			db:         &mockSelectDB{selectResult: []ClusterNodeCount{}},
			cfg:        &config.UsageReporterConfig{TenantID: "t"},
			httpClient: &http.Client{Timeout: 5 * time.Second},
			logger:     zap.NewNop(),
			pending:    make([]UsagePayload, 0),
		}

		err := reporter.collectAndSend(context.Background())

		assert.NoError(t, err)
	})
}

func TestCollectAndSend_HTTPFailure_QueuesPending(t *testing.T) {
	t.Run("should queue payload when HTTP fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		reporter := &UsageReporter{
			db: &mockSelectDB{
				selectResult: []ClusterNodeCount{
					{ClusterName: "test-cluster", NodeCount: 3},
				},
			},
			cfg: &config.UsageReporterConfig{
				MgmtBaseURL: server.URL,
				LicenseKey:  "test-key",
				TenantID:    "0000000001",
				MaxRetries:  1, // fast fail
			},
			httpClient: &http.Client{Timeout: 5 * time.Second},
			logger:     zap.NewNop(),
			pending:    make([]UsagePayload, 0),
		}

		err := reporter.collectAndSend(context.Background())

		assert.Error(t, err)
		assert.Len(t, reporter.pending, 1, "should have 1 pending payload")
		assert.Equal(t, 3, reporter.pending[0].Data.TotalNodeCount)
	})
}

// ============================================================
// Pending Queue Tests
// ============================================================

func TestEnqueuePending_BasicAppend(t *testing.T) {
	t.Run("should append payload to pending queue", func(t *testing.T) {
		reporter := &UsageReporter{
			logger:  zap.NewNop(),
			pending: make([]UsagePayload, 0),
		}

		reporter.enqueuePending(UsagePayload{Meta: UsagePayloadMeta{Timestamp: "t1"}})
		reporter.enqueuePending(UsagePayload{Meta: UsagePayloadMeta{Timestamp: "t2"}})

		assert.Len(t, reporter.pending, 2)
		assert.Equal(t, "t1", reporter.pending[0].Meta.Timestamp)
		assert.Equal(t, "t2", reporter.pending[1].Meta.Timestamp)
	})
}

func TestEnqueuePending_Unbounded(t *testing.T) {
	t.Run("should keep all payloads without dropping", func(t *testing.T) {
		reporter := &UsageReporter{
			logger:  zap.NewNop(),
			pending: make([]UsagePayload, 0),
		}

		for i := 0; i < 100; i++ {
			reporter.enqueuePending(UsagePayload{
				Meta: UsagePayloadMeta{Timestamp: fmt.Sprintf("t%d", i)},
			})
		}

		assert.Len(t, reporter.pending, 100)
		assert.Equal(t, "t0", reporter.pending[0].Meta.Timestamp)
		assert.Equal(t, "t99", reporter.pending[99].Meta.Timestamp)
	})
}

func TestDrainPending_SendsAllAndClears(t *testing.T) {
	t.Run("should send all pending and clear queue", func(t *testing.T) {
		var receivedCount int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&receivedCount, 1)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		reporter := &UsageReporter{
			cfg: &config.UsageReporterConfig{
				MgmtBaseURL: server.URL,
				LicenseKey:  "test-key",
				MaxRetries:  1,
			},
			httpClient: &http.Client{Timeout: 5 * time.Second},
			logger:     zap.NewNop(),
			pending: []UsagePayload{
				{Meta: UsagePayloadMeta{Timestamp: "t1"}},
				{Meta: UsagePayloadMeta{Timestamp: "t2"}},
				{Meta: UsagePayloadMeta{Timestamp: "t3"}},
			},
		}

		reporter.drainPending(context.Background())

		assert.Equal(t, int32(3), atomic.LoadInt32(&receivedCount))
		assert.Empty(t, reporter.pending)
	})
}

func TestDrainPending_StopsOnFailure(t *testing.T) {
	t.Run("should stop and keep remaining on failure", func(t *testing.T) {
		var callCount int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&callCount, 1)
			if count == 2 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		reporter := &UsageReporter{
			cfg: &config.UsageReporterConfig{
				MgmtBaseURL: server.URL,
				LicenseKey:  "test-key",
				MaxRetries:  1,
			},
			httpClient: &http.Client{Timeout: 5 * time.Second},
			logger:     zap.NewNop(),
			pending: []UsagePayload{
				{Meta: UsagePayloadMeta{Timestamp: "t1"}},
				{Meta: UsagePayloadMeta{Timestamp: "t2"}}, // this will fail
				{Meta: UsagePayloadMeta{Timestamp: "t3"}},
			},
		}

		reporter.drainPending(context.Background())

		// t1 sent OK, t2 failed, t2+t3 should remain
		assert.Len(t, reporter.pending, 2)
		assert.Equal(t, "t2", reporter.pending[0].Meta.Timestamp)
		assert.Equal(t, "t3", reporter.pending[1].Meta.Timestamp)
	})
}

func TestDrainPending_EmptyQueue(t *testing.T) {
	t.Run("should do nothing when queue is empty", func(t *testing.T) {
		reporter := &UsageReporter{
			logger:  zap.NewNop(),
			pending: make([]UsagePayload, 0),
		}

		reporter.drainPending(context.Background()) // should not panic
		assert.Empty(t, reporter.pending)
	})
}

// ============================================================
// timeUntilNextRun Tests
// ============================================================

func TestTimeUntilNextRun(t *testing.T) {
	t.Run("should calculate correct wait time", func(t *testing.T) {
		reporter := &UsageReporter{logger: zap.NewNop()}

		wait := reporter.timeUntilNextRun()

		// Should be > 0 and <= 1 hour
		assert.Greater(t, wait, time.Duration(0))
		assert.LessOrEqual(t, wait, 1*time.Hour)
	})
}

// ============================================================
// withRetry Tests
// ============================================================

func TestWithRetry_SuccessOnFirstAttempt(t *testing.T) {
	callCount := 0
	err := withRetry(context.Background(), 3, func() error {
		callCount++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestWithRetry_SuccessAfterFailures(t *testing.T) {
	callCount := 0
	err := withRetry(context.Background(), 3, func() error {
		callCount++
		if callCount < 3 {
			return fmt.Errorf("temporary error")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestWithRetry_AllFailed(t *testing.T) {
	callCount := 0
	err := withRetry(context.Background(), 2, func() error {
		callCount++
		return fmt.Errorf("persistent error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all 2 retries exhausted")
	assert.Equal(t, 2, callCount)
}

func TestWithRetry_NonRetryableError(t *testing.T) {
	callCount := 0
	err := withRetry(context.Background(), 3, func() error {
		callCount++
		return &nonRetryableError{statusCode: 401, message: "unauthorized"}
	})

	assert.Error(t, err)
	assert.Equal(t, 1, callCount)
}

func TestWithRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := withRetry(ctx, 3, func() error {
		return fmt.Errorf("error")
	})

	assert.Error(t, err)
}

// ============================================================
// maskLicenseKey Tests
// ============================================================

func TestMaskLicenseKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"normal key", "sk-live-abcdefgh1234", "sk-l****1234"},
		{"short key", "abc", "***"},
		{"exactly 8 chars", "12345678", "***"},
		{"9 chars", "123456789", "1234****6789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, maskLicenseKey(tt.key))
		})
	}
}

// ============================================================
// Run Tests
// ============================================================

func TestUsageReporter_Run_ContextCancellation(t *testing.T) {
	t.Run("should stop gracefully when context is cancelled", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		reporter, err := NewUsageReporter(
			&mockSelectDB{
				selectResult: []ClusterNodeCount{{ClusterName: "test", NodeCount: 2}},
			},
			&config.UsageReporterConfig{
				Enabled:     true,
				MgmtBaseURL: server.URL,
				LicenseKey:  "test-key-12345678",
				TenantID:    "0000000001",
				Interval:    50 * time.Millisecond,
				HTTPTimeout: 5 * time.Second,
				MaxRetries:  1,
			},
			zap.NewNop(),
		)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			reporter.Run(ctx)
			close(done)
		}()

		// Cancel quickly — Run should exit
		time.Sleep(50 * time.Millisecond)
		cancel()

		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Error("reporter did not stop within timeout")
		}
	})
}
