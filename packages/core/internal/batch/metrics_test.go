package batch

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestBatchMetrics_NewBatchMetrics는 BatchMetrics 생성자 테스트입니다.
func TestBatchMetrics_NewBatchMetrics(t *testing.T) {
	t.Run("should create BatchMetrics with all metrics registered", func(t *testing.T) {
		// Given: 새로운 레지스트리
		registry := prometheus.NewRegistry()

		// When: BatchMetrics 생성
		metrics := NewBatchMetrics(registry)

		// Then: 모든 메트릭이 nil이 아니어야 함
		if metrics == nil {
			t.Fatal("Expected metrics to be non-nil")
		}
		if metrics.ProcessedRows == nil {
			t.Error("Expected ProcessedRows counter to be non-nil")
		}
		if metrics.ProcessingSeconds == nil {
			t.Error("Expected ProcessingSeconds histogram to be non-nil")
		}
		if metrics.Errors == nil {
			t.Error("Expected Errors counter to be non-nil")
		}
		if metrics.LastRunTimestamp == nil {
			t.Error("Expected LastRunTimestamp gauge to be non-nil")
		}
	})

	t.Run("should register metrics with correct names", func(t *testing.T) {
		// Given: 새로운 레지스트리
		registry := prometheus.NewRegistry()

		// When: BatchMetrics 생성
		_ = NewBatchMetrics(registry)

		// Then: 레지스트리에서 메트릭을 조회할 수 있어야 함
		metricFamilies, err := registry.Gather()
		if err != nil {
			t.Fatalf("Failed to gather metrics: %v", err)
		}

		// 메트릭 이름 확인
		expectedNames := map[string]bool{
			"o11y_servicemap_batch_processed_rows_total": false,
			"o11y_servicemap_batch_processing_seconds":   false,
			"o11y_servicemap_batch_errors_total":         false,
			"o11y_servicemap_batch_last_run_timestamp":   false,
		}

		for _, mf := range metricFamilies {
			if _, exists := expectedNames[mf.GetName()]; exists {
				expectedNames[mf.GetName()] = true
			}
		}

		for name, found := range expectedNames {
			if !found {
				t.Errorf("Expected metric %s to be registered", name)
			}
		}
	})
}

// TestBatchMetrics_ProcessedRows는 ProcessedRows Counter 테스트입니다.
func TestBatchMetrics_ProcessedRows(t *testing.T) {
	t.Run("should increment ProcessedRows counter", func(t *testing.T) {
		// Given: BatchMetrics 생성
		registry := prometheus.NewRegistry()
		metrics := NewBatchMetrics(registry)

		// When: 처리된 행 수 증가
		metrics.ProcessedRows.Add(100)
		metrics.ProcessedRows.Add(50)

		// Then: 카운터 값이 150이어야 함
		value := testutil.ToFloat64(metrics.ProcessedRows)
		expected := 150.0
		if value != expected {
			t.Errorf("Expected ProcessedRows to be %v, got %v", expected, value)
		}
	})
}

// TestBatchMetrics_ProcessingSeconds는 ProcessingSeconds Histogram 테스트입니다.
func TestBatchMetrics_ProcessingSeconds(t *testing.T) {
	t.Run("should observe ProcessingSeconds histogram", func(t *testing.T) {
		// Given: BatchMetrics 생성
		registry := prometheus.NewRegistry()
		metrics := NewBatchMetrics(registry)

		// When: 처리 시간 기록
		metrics.ProcessingSeconds.Observe(0.5) // 0.5초
		metrics.ProcessingSeconds.Observe(1.2) // 1.2초
		metrics.ProcessingSeconds.Observe(2.8) // 2.8초

		// Then: 히스토그램 메트릭이 정상적으로 수집되어야 함
		count := testutil.CollectAndCount(metrics.ProcessingSeconds)
		expected := 1 // Histogram은 1개의 메트릭으로 수집됨
		if count != expected {
			t.Errorf("Expected histogram metric count to be %v, got %v", expected, count)
		}
	})

	t.Run("should have correct histogram buckets", func(t *testing.T) {
		// Given: BatchMetrics 생성
		registry := prometheus.NewRegistry()
		metrics := NewBatchMetrics(registry)

		// When: 다양한 처리 시간 기록
		metrics.ProcessingSeconds.Observe(0.05) // < 0.1
		metrics.ProcessingSeconds.Observe(0.3)  // 0.1 ~ 0.5
		metrics.ProcessingSeconds.Observe(0.8)  // 0.5 ~ 1
		metrics.ProcessingSeconds.Observe(1.5)  // 1 ~ 2
		metrics.ProcessingSeconds.Observe(3.0)  // 2 ~ 5
		metrics.ProcessingSeconds.Observe(7.0)  // 5 ~ 10

		// Then: 히스토그램 메트릭이 정상적으로 수집되어야 함
		count := testutil.CollectAndCount(metrics.ProcessingSeconds)
		expected := 1 // Histogram은 1개의 메트릭으로 수집됨
		if count != expected {
			t.Errorf("Expected histogram metric count to be %v, got %v", expected, count)
		}
	})
}

// TestBatchMetrics_Errors는 Errors Counter 테스트입니다.
func TestBatchMetrics_Errors(t *testing.T) {
	t.Run("should increment Errors counter", func(t *testing.T) {
		// Given: BatchMetrics 생성
		registry := prometheus.NewRegistry()
		metrics := NewBatchMetrics(registry)

		// When: 에러 발생
		metrics.Errors.Inc()
		metrics.Errors.Inc()
		metrics.Errors.Inc()

		// Then: 카운터 값이 3이어야 함
		value := testutil.ToFloat64(metrics.Errors)
		expected := 3.0
		if value != expected {
			t.Errorf("Expected Errors to be %v, got %v", expected, value)
		}
	})
}

// TestBatchMetrics_LastRunTimestamp는 LastRunTimestamp Gauge 테스트입니다.
func TestBatchMetrics_LastRunTimestamp(t *testing.T) {
	t.Run("should set LastRunTimestamp gauge", func(t *testing.T) {
		// Given: BatchMetrics 생성
		registry := prometheus.NewRegistry()
		metrics := NewBatchMetrics(registry)

		// When: 마지막 실행 시간 설정
		timestamp := 1706745600.0 // 2024-02-01 00:00:00 UTC
		metrics.LastRunTimestamp.Set(timestamp)

		// Then: 게이지 값이 설정한 값과 같아야 함
		value := testutil.ToFloat64(metrics.LastRunTimestamp)
		if value != timestamp {
			t.Errorf("Expected LastRunTimestamp to be %v, got %v", timestamp, value)
		}
	})

	t.Run("should update LastRunTimestamp gauge", func(t *testing.T) {
		// Given: BatchMetrics 생성 및 초기 시간 설정
		registry := prometheus.NewRegistry()
		metrics := NewBatchMetrics(registry)
		initialTime := 1706745600.0
		metrics.LastRunTimestamp.Set(initialTime)

		// When: 새로운 시간으로 업데이트
		newTime := 1706745620.0 // 20초 후
		metrics.LastRunTimestamp.Set(newTime)

		// Then: 게이지 값이 새로운 값으로 업데이트되어야 함
		value := testutil.ToFloat64(metrics.LastRunTimestamp)
		if value != newTime {
			t.Errorf("Expected LastRunTimestamp to be %v, got %v", newTime, value)
		}
	})
}

// TestBatchMetrics_ClusterLabel는 cluster_name 레이블 테스트입니다.
func TestBatchMetrics_ClusterLabel(t *testing.T) {
	t.Run("should support cluster_name label when provided", func(t *testing.T) {
		// Given: 클러스터 이름과 함께 BatchMetrics 생성
		registry := prometheus.NewRegistry()
		clusterName := "test-cluster"
		metrics := NewBatchMetricsWithCluster(registry, clusterName)

		// When: 메트릭 값 설정
		metrics.ProcessedRows.Add(100)

		// Then: 메트릭이 정상적으로 증가해야 함
		value := testutil.ToFloat64(metrics.ProcessedRows)
		expected := 100.0
		if value != expected {
			t.Errorf("Expected ProcessedRows to be %v, got %v", expected, value)
		}
	})
}
