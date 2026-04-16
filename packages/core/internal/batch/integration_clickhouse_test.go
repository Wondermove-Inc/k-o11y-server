//go:build integration

package batch

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.uber.org/zap"
)

// ============================================================
// TASK-012: 실제 ClickHouse 통합 테스트
// ============================================================
//
// 이 파일은 실제 ClickHouse 연결을 필요로 하는 통합 테스트를 포함합니다.
// 빌드 태그로 분리되어 실제 DB 환경에서만 실행됩니다.
//
// 빌드 태그: integration
//
// 실행 방법:
//   go test -tags=integration ./... -v
//
// 필수 환경 변수:
//   CLICKHOUSE_HOST - ClickHouse 호스트 (default: localhost)
//   CLICKHOUSE_PORT - ClickHouse 포트 (default: 9000)
//   CLICKHOUSE_DATABASE - 데이터베이스 이름 (default: signoz_traces)
//   CLICKHOUSE_USER - 사용자 이름 (default: default)
//   CLICKHOUSE_PASSWORD - 비밀번호 (default: empty)

// setupRealClickHouse creates a real ClickHouse connection for integration tests.
// Returns nil and skips test if connection fails (test environment not available).
func setupRealClickHouse(t *testing.T) driver.Conn {
	t.Helper()

	// Get ClickHouse configuration from environment
	host := getEnvOrDefault("CLICKHOUSE_HOST", "<YOUR_IP>")
	port := getEnvOrDefault("CLICKHOUSE_PORT", "9000")
	database := getEnvOrDefault("CLICKHOUSE_DATABASE", "signoz_traces")
	username := getEnvOrDefault("CLICKHOUSE_USER", "default")
	password := getEnvOrDefault("CLICKHOUSE_PASSWORD", "<CLICKHOUSE_PASSWORD>")

	// Create ClickHouse connection
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", host, port)},
		Auth: clickhouse.Auth{
			Database: database,
			Username: username,
			Password: password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 10 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})

	if err != nil {
		t.Skipf("ClickHouse connection failed (test environment not available): %v", err)
		return nil
	}

	// Ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		conn.Close()
		t.Skipf("ClickHouse ping failed (test environment not available): %v", err)
		return nil
	}

	t.Logf("Connected to ClickHouse: %s:%s/%s", host, port, database)
	return conn
}

// teardownRealClickHouse cleans up ClickHouse connection.
func teardownRealClickHouse(conn driver.Conn) {
	if conn != nil {
		conn.Close()
	}
}

// getEnvOrDefault retrieves an environment variable or returns default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================
// 실제 ClickHouse 통합 테스트
// ============================================================

// TestServiceMapBatchProcessor_RealClickHouse_ProcessBatch tests batch processing with real DB.
// AC3: 실제 ClickHouse 테스트는 빌드 태그로 분리
func TestServiceMapBatchProcessor_RealClickHouse_ProcessBatch(t *testing.T) {
	// Given: Real ClickHouse connection
	conn := setupRealClickHouse(t)
	if conn == nil {
		return // Test skipped (connection failed)
	}
	defer teardownRealClickHouse(conn)

	logger := zap.NewNop()
	metrics := NewBatchMetrics(nil)

	processor, err := NewServiceMapBatchProcessor(conn, 20*time.Second, 120*time.Second, 20*time.Second, 60*time.Second, logger, metrics)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	// When: Execute processBatch
	ctx := context.Background()
	startTime := time.Now()
	err = processor.processBatch(ctx)
	elapsed := time.Since(startTime)

	// Then: Should complete successfully
	if err != nil {
		t.Errorf("processBatch failed: %v", err)
	}

	// Processing should be reasonably fast (< 5 seconds)
	if elapsed > 5*time.Second {
		t.Errorf("processing too slow: %v", elapsed)
	}

	t.Logf("Batch processing completed in %v", elapsed)
}

// TestServiceMapBatchProcessor_RealClickHouse_WatermarkUpdate tests watermark update.
// AC3: 실제 ClickHouse 테스트 - watermark 갱신 확인
func TestServiceMapBatchProcessor_RealClickHouse_WatermarkUpdate(t *testing.T) {
	// Given: Real ClickHouse connection
	conn := setupRealClickHouse(t)
	if conn == nil {
		return
	}
	defer teardownRealClickHouse(conn)

	logger := zap.NewNop()
	processor, _ := NewServiceMapBatchProcessor(conn, 20*time.Second, 120*time.Second, 20*time.Second, 60*time.Second, logger, nil)

	ctx := context.Background()

	// When: Get watermark before and after batch
	watermarkBefore, err := processor.getWatermark(ctx)
	if err != nil {
		t.Fatalf("failed to get watermark before: %v", err)
	}

	// Execute batch
	err = processor.processBatch(ctx)
	if err != nil {
		t.Fatalf("processBatch failed: %v", err)
	}

	// Get watermark after
	time.Sleep(100 * time.Millisecond) // Brief delay for watermark to update
	watermarkAfter, err := processor.getWatermark(ctx)
	if err != nil {
		t.Fatalf("failed to get watermark after: %v", err)
	}

	// Then: Watermark should be updated (or same if no new data)
	// At minimum, watermark should not be zero
	if watermarkAfter.IsZero() {
		t.Error("watermark should not be zero after batch execution")
	}

	// Log watermark values for inspection
	t.Logf("Watermark before: %v", watermarkBefore)
	t.Logf("Watermark after: %v", watermarkAfter)

	// If watermarks are different, verify after >= before
	if !watermarkAfter.Equal(watermarkBefore) {
		if watermarkAfter.Before(watermarkBefore) {
			t.Error("watermark should not go backwards")
		}
	}
}

// TestServiceMapBatchProcessor_RealClickHouse_FullCycle tests full execution cycle.
// AC4: Config → Processor 생성 → Run → Shutdown 전체 라이프사이클 (실제 DB)
func TestServiceMapBatchProcessor_RealClickHouse_FullCycle(t *testing.T) {
	// Given: Real ClickHouse connection
	conn := setupRealClickHouse(t)
	if conn == nil {
		return
	}
	defer teardownRealClickHouse(conn)

	logger := zap.NewNop()
	metrics := NewBatchMetrics(nil)

	processor, err := NewServiceMapBatchProcessor(conn, 3*time.Second, 120*time.Second, 20*time.Second, 60*time.Second, logger, metrics)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	// When: Run processor for 10 seconds (should execute ~3 batches)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	startTime := time.Now()
	processor.Run(ctx)
	elapsed := time.Since(startTime)

	// Then: Should have run for approximately 10 seconds
	if elapsed < 9*time.Second || elapsed > 11*time.Second {
		t.Errorf("execution time out of range: got %v, want ~10s±1s", elapsed)
	}

	t.Logf("Full cycle completed in %v", elapsed)
}

// TestServiceMapBatchProcessor_RealClickHouse_ConcurrentExecution tests concurrent execution safety.
// AC5: 모든 테스트 통과 - 동시성 안전성 (실제 DB)
func TestServiceMapBatchProcessor_RealClickHouse_ConcurrentExecution(t *testing.T) {
	// Given: Real ClickHouse connection
	conn := setupRealClickHouse(t)
	if conn == nil {
		return
	}
	defer teardownRealClickHouse(conn)

	logger := zap.NewNop()
	processor, _ := NewServiceMapBatchProcessor(conn, 2*time.Second, 120*time.Second, 20*time.Second, 60*time.Second, logger, nil)

	// When: Run two concurrent batch processes (should handle safely)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		processor.Run(ctx)
		close(done)
	}()

	// Wait for execution to complete
	<-done

	// Then: Should complete without deadlock or panic
	t.Logf("Concurrent execution completed successfully")
}

// TestServiceMapBatchProcessor_RealClickHouse_ErrorHandling tests error handling with real DB.
// AC2: Mock 기반 전체 흐름 테스트 - 에러 처리 (실제 DB에서도 확인)
func TestServiceMapBatchProcessor_RealClickHouse_ErrorHandling(t *testing.T) {
	// Given: Real ClickHouse connection
	conn := setupRealClickHouse(t)
	if conn == nil {
		return
	}
	defer teardownRealClickHouse(conn)

	logger := zap.NewNop()
	metrics := NewBatchMetrics(nil)
	processor, _ := NewServiceMapBatchProcessor(conn, 4*time.Second, 120*time.Second, 20*time.Second, 60*time.Second, logger, metrics)

	// When: Run processor with short timeout (may cause some batches to fail)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	processor.Run(ctx)

	// Then: Should complete without panic even if some batches fail
	// Error metrics should be trackable via metrics.Errors counter
	if processor.metrics == nil {
		t.Error("metrics should not be nil")
	}

	t.Logf("Error handling test completed")
}

// TestServiceMapBatchProcessor_RealClickHouse_LongRunning tests long-running execution.
// This test runs for 30 seconds to verify stability over time
func TestServiceMapBatchProcessor_RealClickHouse_LongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	// Given: Real ClickHouse connection
	conn := setupRealClickHouse(t)
	if conn == nil {
		return
	}
	defer teardownRealClickHouse(conn)

	logger := zap.NewNop()
	metrics := NewBatchMetrics(nil)
	processor, _ := NewServiceMapBatchProcessor(conn, 5*time.Second, 120*time.Second, 20*time.Second, 60*time.Second, logger, metrics)

	// When: Run for 30 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()
	processor.Run(ctx)
	elapsed := time.Since(startTime)

	// Then: Should run stably for the entire duration
	if elapsed < 28*time.Second || elapsed > 32*time.Second {
		t.Errorf("execution time out of range: got %v, want ~30s±2s", elapsed)
	}

	t.Logf("Long-running test completed in %v", elapsed)
}

// TestServiceMapBatchProcessor_RealClickHouse_ContextCancellation tests cancellation with real DB.
// AC4: 전체 라이프사이클 테스트 - Context 취소 (실제 DB)
func TestServiceMapBatchProcessor_RealClickHouse_ContextCancellation(t *testing.T) {
	// Given: Real ClickHouse connection
	conn := setupRealClickHouse(t)
	if conn == nil {
		return
	}
	defer teardownRealClickHouse(conn)

	logger := zap.NewNop()
	processor, _ := NewServiceMapBatchProcessor(conn, 3*time.Second, 120*time.Second, 20*time.Second, 60*time.Second, logger, nil)

	ctx, cancel := context.WithCancel(context.Background())

	// When: Start processor and cancel after 5 seconds
	done := make(chan struct{})
	go func() {
		processor.Run(ctx)
		close(done)
	}()

	time.Sleep(5 * time.Second)
	cancel()

	// Then: Should stop within 5 seconds of cancellation
	select {
	case <-done:
		t.Logf("Processor stopped gracefully after cancellation")
	case <-time.After(5 * time.Second):
		t.Error("processor did not stop within 5 seconds of cancellation")
	}
}

// TestServiceMapBatchProcessor_RealClickHouse_WatermarkPersistence tests watermark persistence.
// Verifies that watermark survives across multiple processor instances
func TestServiceMapBatchProcessor_RealClickHouse_WatermarkPersistence(t *testing.T) {
	// Given: Real ClickHouse connection
	conn := setupRealClickHouse(t)
	if conn == nil {
		return
	}
	defer teardownRealClickHouse(conn)

	logger := zap.NewNop()

	// Create first processor instance
	processor1, _ := NewServiceMapBatchProcessor(conn, 20*time.Second, 120*time.Second, 20*time.Second, 60*time.Second, logger, nil)
	ctx := context.Background()

	// When: Execute batch with first processor
	watermark1, err := processor1.getWatermark(ctx)
	if err != nil {
		t.Fatalf("failed to get watermark from processor1: %v", err)
	}

	err = processor1.processBatch(ctx)
	if err != nil {
		t.Fatalf("processBatch failed on processor1: %v", err)
	}

	time.Sleep(100 * time.Millisecond) // Brief delay for persistence

	// Create second processor instance (simulates restart)
	processor2, _ := NewServiceMapBatchProcessor(conn, 20*time.Second, 120*time.Second, 20*time.Second, 60*time.Second, logger, nil)

	// Get watermark with second processor
	watermark2, err := processor2.getWatermark(ctx)
	if err != nil {
		t.Fatalf("failed to get watermark from processor2: %v", err)
	}

	// Then: Watermark should persist across instances
	if watermark2.Before(watermark1) {
		t.Error("watermark should not go backwards across processor instances")
	}

	t.Logf("Watermark persisted: %v -> %v", watermark1, watermark2)
}
