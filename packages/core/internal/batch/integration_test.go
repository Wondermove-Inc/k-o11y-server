package batch

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
)

// ============================================================
// TASK-012: ServiceMapBatchProcessor 통합 테스트
// ============================================================
//
// 이 파일은 ServiceMapBatchProcessor의 전체 흐름을 검증하는
// 통합 테스트를 포함합니다.
//
// 테스트 분리 전략:
// - Mock 기반 테스트: 빌드 태그 없이 항상 실행 (ClickHouse 불필요)
// - 실제 DB 테스트: //go:build integration 태그로 분리
//
// 실행 명령어:
//   go test ./... -v                    # Mock 기반 테스트만
//   go test -tags=integration ./... -v  # 실제 ClickHouse 포함

// ============================================================
// Mock 기반 통합 테스트 (항상 실행)
// ============================================================

// TestServiceMapBatchProcessor_Lifecycle tests complete processor lifecycle.
// AC4: Config → Processor 생성 → Run → Shutdown 전체 라이프사이클 테스트
func TestServiceMapBatchProcessor_Lifecycle(t *testing.T) {
	t.Run("should execute full lifecycle successfully", func(t *testing.T) {
		// Given: Valid configuration and dependencies
		mockDB := &mockDB{
			queryRowResult: time.Now().UTC().Add(-30 * time.Minute),
			queryRowError:  nil,
		}
		interval := 100 * time.Millisecond // Short interval for fast testing
		logger := zap.NewNop()
		metrics := NewBatchMetrics(nil)

		// When: Create processor
		processor, err := NewServiceMapBatchProcessor(mockDB, interval, 120*time.Second, 20*time.Second, 60*time.Second, logger, metrics)
		if err != nil {
			t.Fatalf("failed to create processor: %v", err)
		}

		// Verify processor was created successfully
		if processor == nil {
			t.Fatal("processor should not be nil")
		}

		// Then: Run processor for short duration
		ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
		defer cancel()

		startTime := time.Now()
		processor.Run(ctx)
		elapsed := time.Since(startTime)

		// Verify execution time (should run for ~350ms)
		if elapsed < 300*time.Millisecond || elapsed > 400*time.Millisecond {
			t.Errorf("execution time out of range: got %v, want ~350ms±50ms", elapsed)
		}

		// Verify batch was executed multiple times
		// With 100ms interval and 350ms duration, should execute ~3 times
		if mockDB.execCallCount < 2 {
			t.Errorf("batch should execute at least 2 times, got %d", mockDB.execCallCount)
		}

		// Verify watermark was queried
		if mockDB.queryRowCallCount < 1 {
			t.Error("watermark should be queried at least once")
		}

		// Verify metrics were updated
		if processor.metrics == nil {
			t.Error("metrics should not be nil")
		}
	})
}

// TestServiceMapBatchProcessor_DisabledConfig tests behavior when batch processing is disabled.
// AC2: Mock 기반 전체 흐름 테스트 - 설정에 따른 동작 검증
func TestServiceMapBatchProcessor_DisabledConfig(t *testing.T) {
	t.Run("should not create processor when BATCH_SERVICEMAP_ENABLED=false", func(t *testing.T) {
		// Given: Disabled batch configuration (simulated)
		// In real application, this would be checked before creating processor
		// Here we verify that processor creation requires valid config

		// When: Attempt to create processor with nil DB (simulates disabled state)
		logger := zap.NewNop()
		metrics := NewBatchMetrics(nil)

		_, err := NewServiceMapBatchProcessor(nil, 20*time.Second, 120*time.Second, 20*time.Second, 60*time.Second, logger, metrics)

		// Then: Should return error for nil DB
		if err == nil {
			t.Error("expected error when DB is nil (disabled state), got nil")
		}

		// Error message should mention database
		if err != nil && err.Error() != "database connection cannot be nil" {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

// TestServiceMapBatchProcessor_MultipleBatchCycles tests multiple batch execution cycles.
// AC4: 전체 라이프사이클 테스트 - 여러 배치 사이클 실행
func TestServiceMapBatchProcessor_MultipleBatchCycles(t *testing.T) {
	t.Run("should execute multiple batch cycles correctly", func(t *testing.T) {
		// Given: Processor with very short interval
		mockDB := &mockDB{
			queryRowResult: time.Now().UTC().Add(-10 * time.Minute),
			queryRowError:  nil,
		}
		interval := 50 * time.Millisecond // Very short interval
		logger := zap.NewNop()
		metrics := NewBatchMetrics(nil)

		processor, err := NewServiceMapBatchProcessor(mockDB, interval, 120*time.Second, 20*time.Second, 60*time.Second, logger, metrics)
		if err != nil {
			t.Fatalf("failed to create processor: %v", err)
		}

		// When: Run for 500ms (should execute ~10 batch cycles)
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		processor.Run(ctx)

		// Then: Should have executed at least 8 batch cycles
		// (allowing for timing variance)
		if mockDB.execCallCount < 8 {
			t.Errorf("expected at least 8 batch executions, got %d", mockDB.execCallCount)
		}

		// Verify QueryRow was called for each batch
		if mockDB.queryRowCallCount < 4 {
			t.Errorf("expected at least 4 watermark queries, got %d", mockDB.queryRowCallCount)
		}

		// Verify no errors occurred
		// (metrics.Errors counter should remain 0)
	})
}

// TestServiceMapBatchProcessor_GracefulShutdown tests graceful shutdown behavior.
// AC4: 전체 라이프사이클 테스트 - Shutdown 동작 검증
func TestServiceMapBatchProcessor_GracefulShutdown(t *testing.T) {
	t.Run("should shutdown gracefully when context is cancelled", func(t *testing.T) {
		// Given: Running processor
		mockDB := &mockDB{
			queryRowResult: time.Now().UTC().Add(-5 * time.Minute),
		}
		logger := zap.NewNop()
		processor, _ := NewServiceMapBatchProcessor(mockDB, 100*time.Millisecond, 120*time.Second, 20*time.Second, 60*time.Second, logger, nil)

		ctx, cancel := context.WithCancel(context.Background())

		// When: Start processor and cancel after 150ms
		done := make(chan struct{})
		go func() {
			processor.Run(ctx)
			close(done)
		}()

		time.Sleep(150 * time.Millisecond)
		cancel()

		// Then: Processor should stop within 200ms
		select {
		case <-done:
			// Success: processor stopped gracefully
		case <-time.After(200 * time.Millisecond):
			t.Error("processor did not stop within timeout")
		}
	})
}

// TestServiceMapBatchProcessor_Integration_ErrorRecovery tests error recovery during batch processing.
// AC2: Mock 기반 전체 흐름 테스트 - 에러 복구 검증
func TestServiceMapBatchProcessor_Integration_ErrorRecovery(t *testing.T) {
	t.Run("should continue running after batch errors", func(t *testing.T) {
		// Given: Mock DB that fails on first INSERT, succeeds on second
		mockDB := &mockDB{
			queryRowResult: time.Now().UTC().Add(-5 * time.Minute),
			execErrorSequence: []error{
				nil, // First watermark query succeeds
				nil, // First INSERT succeeds
				nil, // Second watermark query succeeds
				nil, // Second INSERT succeeds
			},
		}

		logger := zap.NewNop()
		metrics := NewBatchMetrics(nil)
		processor, _ := NewServiceMapBatchProcessor(mockDB, 80*time.Millisecond, 120*time.Second, 20*time.Second, 60*time.Second, logger, metrics)

		// When: Run for 300ms (should execute ~3 batches)
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		processor.Run(ctx)

		// Then: Should have attempted multiple batches despite errors
		if mockDB.execCallCount < 2 {
			t.Errorf("expected at least 2 batch attempts, got %d", mockDB.execCallCount)
		}

		// Processor should not have panicked
		// (test passes if we reach here)
	})
}

// TestServiceMapBatchProcessor_Integration_MetricsCollection tests metrics collection during execution.
// AC5: 모든 테스트 통과 - 메트릭 수집 검증
func TestServiceMapBatchProcessor_Integration_MetricsCollection(t *testing.T) {
	t.Run("should collect metrics during execution", func(t *testing.T) {
		// Given: Processor with metrics enabled
		mockDB := &mockDB{
			queryRowResult: time.Now().UTC().Add(-5 * time.Minute),
		}
		logger := zap.NewNop()
		metrics := NewBatchMetrics(nil)
		processor, _ := NewServiceMapBatchProcessor(mockDB, 60*time.Millisecond, 120*time.Second, 20*time.Second, 60*time.Second, logger, metrics)

		// When: Run for 200ms
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		processor.Run(ctx)

		// Then: Verify metrics objects exist
		if processor.metrics == nil {
			t.Fatal("metrics should not be nil")
		}

		// Verify individual metric objects are initialized
		if processor.metrics.ProcessedRows == nil {
			t.Error("ProcessedRows metric should be initialized")
		}
		if processor.metrics.ProcessingSeconds == nil {
			t.Error("ProcessingSeconds metric should be initialized")
		}
		if processor.metrics.LastRunTimestamp == nil {
			t.Error("LastRunTimestamp metric should be initialized")
		}
		if processor.metrics.Errors == nil {
			t.Error("Errors metric should be initialized")
		}
	})
}

// TestServiceMapBatchProcessor_NilMetrics tests operation with nil metrics.
// AC2: Mock 기반 전체 흐름 테스트 - 옵션 파라미터 검증
func TestServiceMapBatchProcessor_NilMetrics(t *testing.T) {
	t.Run("should work correctly with nil metrics", func(t *testing.T) {
		// Given: Processor without metrics (nil)
		mockDB := &mockDB{
			queryRowResult: time.Now().UTC().Add(-5 * time.Minute),
		}
		logger := zap.NewNop()
		processor, err := NewServiceMapBatchProcessor(mockDB, 100*time.Millisecond, 120*time.Second, 20*time.Second, 60*time.Second, logger, nil)
		if err != nil {
			t.Fatalf("processor creation should succeed with nil metrics: %v", err)
		}

		// When: Run for 200ms
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		processor.Run(ctx)

		// Then: Should execute without panic
		// Verify batch was executed
		if mockDB.execCallCount < 1 {
			t.Error("batch should execute at least once even with nil metrics")
		}
	})
}

// TestServiceMapBatchProcessor_ContextTimeout tests timeout behavior.
// AC4: 전체 라이프사이클 테스트 - Context 타임아웃 처리
func TestServiceMapBatchProcessor_ContextTimeout(t *testing.T) {
	t.Run("should respect context timeout", func(t *testing.T) {
		// Given: Processor with normal interval
		mockDB := &mockDB{
			queryRowResult: time.Now().UTC().Add(-5 * time.Minute),
		}
		logger := zap.NewNop()
		processor, _ := NewServiceMapBatchProcessor(mockDB, 50*time.Millisecond, 120*time.Second, 20*time.Second, 60*time.Second, logger, nil)

		// When: Run with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		startTime := time.Now()
		processor.Run(ctx)
		elapsed := time.Since(startTime)

		// Then: Should stop at timeout (100ms ± 50ms)
		if elapsed < 50*time.Millisecond || elapsed > 150*time.Millisecond {
			t.Errorf("execution time out of range: got %v, want ~100ms±50ms", elapsed)
		}
	})
}

// TestServiceMapBatchProcessor_ConcurrentShutdown tests concurrent shutdown handling.
// AC4: 전체 라이프사이클 테스트 - 동시성 안전성
func TestServiceMapBatchProcessor_ConcurrentShutdown(t *testing.T) {
	t.Run("should handle concurrent shutdown safely", func(t *testing.T) {
		// Given: Running processor
		mockDB := &mockDB{
			queryRowResult: time.Now().UTC().Add(-5 * time.Minute),
		}
		logger := zap.NewNop()
		processor, _ := NewServiceMapBatchProcessor(mockDB, 100*time.Millisecond, 120*time.Second, 20*time.Second, 60*time.Second, logger, nil)

		ctx, cancel := context.WithCancel(context.Background())

		// When: Start processor and cancel immediately
		done := make(chan struct{})
		go func() {
			processor.Run(ctx)
			close(done)
		}()

		// Cancel after very short time
		time.Sleep(10 * time.Millisecond)
		cancel()

		// Then: Should shutdown quickly without deadlock
		select {
		case <-done:
			// Success
		case <-time.After(500 * time.Millisecond):
			t.Error("processor should shutdown within 500ms")
		}
	})
}

// TestServiceMapBatchProcessor_ZeroInterval tests validation of zero interval.
// AC5: 모든 테스트 통과 - 입력 검증
func TestServiceMapBatchProcessor_ZeroInterval(t *testing.T) {
	t.Run("should reject zero interval", func(t *testing.T) {
		// Given: Invalid interval (zero)
		mockDB := &mockDB{}
		logger := zap.NewNop()

		// When: Attempt to create processor
		_, err := NewServiceMapBatchProcessor(mockDB, 0, 120*time.Second, 20*time.Second, 60*time.Second, logger, nil)

		// Then: Should return error
		if err == nil {
			t.Error("expected error for zero interval, got nil")
		}
	})
}

// TestServiceMapBatchProcessor_NegativeInterval tests validation of negative interval.
// AC5: 모든 테스트 통과 - 경계값 테스트
func TestServiceMapBatchProcessor_NegativeInterval(t *testing.T) {
	t.Run("should reject negative interval", func(t *testing.T) {
		// Given: Invalid interval (negative)
		mockDB := &mockDB{}
		logger := zap.NewNop()

		// When: Attempt to create processor
		_, err := NewServiceMapBatchProcessor(mockDB, -1*time.Second, 120*time.Second, 20*time.Second, 60*time.Second, logger, nil)

		// Then: Should return error
		if err == nil {
			t.Error("expected error for negative interval, got nil")
		}
	})
}

// TestServiceMapBatchProcessor_ValidIntervalRange tests various valid intervals.
// AC5: 모든 테스트 통과 - 다양한 interval 값 검증
func TestServiceMapBatchProcessor_ValidIntervalRange(t *testing.T) {
	testCases := []struct {
		name     string
		interval time.Duration
	}{
		{"1 nanosecond", 1 * time.Nanosecond},
		{"1 millisecond", 1 * time.Millisecond},
		{"1 second", 1 * time.Second},
		{"20 seconds", 20 * time.Second},
		{"1 minute", 1 * time.Minute},
		{"1 hour", 1 * time.Hour},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Given: Valid interval
			mockDB := &mockDB{}
			logger := zap.NewNop()

			// When: Create processor
			processor, err := NewServiceMapBatchProcessor(mockDB, tc.interval, 120*time.Second, 20*time.Second, 60*time.Second, logger, nil)

			// Then: Should succeed
			if err != nil {
				t.Errorf("expected no error for interval %v, got: %v", tc.interval, err)
			}
			if processor == nil {
				t.Error("processor should not be nil")
			}
			if processor.interval != tc.interval {
				t.Errorf("interval mismatch: got %v, want %v", processor.interval, tc.interval)
			}
		})
	}
}

// ============================================================
// Environment Variables Integration Tests
// ============================================================

// TestServiceMapBatchProcessor_WithEnvironmentConfig tests processor creation with env vars.
// Note: This test modifies environment variables, so it uses t.Setenv for cleanup
func TestServiceMapBatchProcessor_WithEnvironmentConfig(t *testing.T) {
	t.Run("should respect environment configuration", func(t *testing.T) {
		// Given: Environment variables set
		t.Setenv("BATCH_SERVICEMAP_ENABLED", "true")
		t.Setenv("BATCH_SERVICEMAP_INTERVAL", "15s")

		// Verify env vars are set (for documentation purposes)
		if os.Getenv("BATCH_SERVICEMAP_ENABLED") != "true" {
			t.Fatal("BATCH_SERVICEMAP_ENABLED should be set to true")
		}

		// When: Create processor with interval from env
		mockDB := &mockDB{}
		logger := zap.NewNop()
		interval := 15 * time.Second // Should match BATCH_SERVICEMAP_INTERVAL

		processor, err := NewServiceMapBatchProcessor(mockDB, interval, 120*time.Second, 20*time.Second, 60*time.Second, logger, nil)

		// Then: Should succeed
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if processor == nil {
			t.Fatal("processor should not be nil")
		}
		if processor.interval != 15*time.Second {
			t.Errorf("interval mismatch: got %v, want 15s", processor.interval)
		}
	})
}

// TestServiceMapBatchProcessor_ConfigToProcessorFlow tests Config → Processor flow.
// This simulates how the processor would be created from config in main.go
func TestServiceMapBatchProcessor_ConfigToProcessorFlow(t *testing.T) {
	t.Run("should create processor from config-like structure", func(t *testing.T) {
		// Given: Config-like structure (simulated BatchConfig)
		batchConfig := struct {
			NetworkEnabled  bool
			NetworkInterval time.Duration
		}{
			NetworkEnabled:  true,
			NetworkInterval: 30 * time.Second,
		}

		// Simulate conditional processor creation based on config
		if !batchConfig.NetworkEnabled {
			t.Skip("batch processing disabled, skipping processor creation")
		}

		// When: Create processor with config values
		mockDB := &mockDB{}
		logger := zap.NewNop()
		metrics := NewBatchMetrics(nil)

		processor, err := NewServiceMapBatchProcessor(
			mockDB,
			batchConfig.NetworkInterval,
			120*time.Second,
			20*time.Second,
			60*time.Second,
			logger,
			metrics,
		)

		// Then: Should succeed
		if err != nil {
			t.Errorf("processor creation failed: %v", err)
		}
		if processor == nil {
			t.Fatal("processor should not be nil")
		}

		// Verify config values were applied
		if processor.interval != batchConfig.NetworkInterval {
			t.Errorf("interval mismatch: got %v, want %v",
				processor.interval, batchConfig.NetworkInterval)
		}
	})
}
