package batch

import (
	"github.com/prometheus/client_golang/prometheus"
)

// BatchMetrics는 서비스맵 배치 프로세서의 Prometheus 메트릭을 관리하는 구조체입니다.
//
// 이 구조체는 배치 처리의 성능 지표를 추적하고 Prometheus를 통해 노출합니다.
// 모든 메트릭은 `o11y_servicemap_batch_` 접두사를 사용하여 네임스페이스를 구분합니다.
//
// 메트릭 종류:
//   - ProcessedRows: 처리된 총 레코드 수 (Counter)
//   - ProcessingSeconds: 배치 처리 시간 분포 (Histogram)
//   - Errors: 발생한 총 에러 수 (Counter)
//   - LastRunTimestamp: 마지막 배치 실행 시간 (Gauge)
//
// 사용 예시:
//
//	registry := prometheus.NewRegistry()
//	metrics := NewBatchMetrics(registry)
//
//	// 배치 처리 시작
//	start := time.Now()
//	rowCount, err := processBatch()
//	duration := time.Since(start).Seconds()
//
//	// 메트릭 기록
//	if err != nil {
//	    metrics.Errors.Inc()
//	} else {
//	    metrics.ProcessedRows.Add(float64(rowCount))
//	    metrics.ProcessingSeconds.Observe(duration)
//	    metrics.LastRunTimestamp.SetToCurrentTime()
//	}
type BatchMetrics struct {
	// ProcessedRows는 배치 처리에서 성공적으로 처리된 총 레코드 수를 추적합니다.
	// Counter 타입으로 단조 증가하며, 배치 실행마다 처리된 행 수만큼 증가합니다.
	//
	// 메트릭 이름: o11y_servicemap_batch_processed_rows_total
	// 타입: Counter
	// 레이블: cluster_name (선택적)
	ProcessedRows prometheus.Counter

	// ProcessingSeconds는 배치 처리 시간의 분포를 추적합니다.
	// Histogram 타입으로 처리 시간을 버킷별로 집계하여 P50, P95, P99 등의 지표를 계산할 수 있습니다.
	//
	// 메트릭 이름: o11y_servicemap_batch_processing_seconds
	// 타입: Histogram
	// 버킷: [0.1, 0.5, 1, 2, 5, 10]초
	// 레이블: cluster_name (선택적)
	//
	// 버킷 설명:
	//   - 0.1초: 매우 빠른 처리 (일반적으로 작은 데이터셋)
	//   - 0.5초: 빠른 처리
	//   - 1초: 정상 처리
	//   - 2초: 느린 처리
	//   - 5초: 매우 느린 처리
	//   - 10초: 비정상적으로 느린 처리 (조사 필요)
	ProcessingSeconds prometheus.Histogram

	// Errors는 배치 처리 중 발생한 총 에러 수를 추적합니다.
	// Counter 타입으로 에러 발생 시마다 증가합니다.
	//
	// 메트릭 이름: o11y_servicemap_batch_errors_total
	// 타입: Counter
	// 레이블: cluster_name (선택적)
	//
	// 에러 종류:
	//   - ClickHouse 연결 실패
	//   - 쿼리 실행 실패
	//   - 데이터 변환 실패
	//   - Watermark 업데이트 실패
	Errors prometheus.Counter

	// LastRunTimestamp는 배치가 마지막으로 성공적으로 실행된 시간을 추적합니다.
	// Gauge 타입으로 Unix timestamp (초 단위)를 저장합니다.
	//
	// 메트릭 이름: o11y_servicemap_batch_last_run_timestamp
	// 타입: Gauge
	// 레이블: cluster_name (선택적)
	//
	// 사용 용도:
	//   - 배치가 정상적으로 실행 중인지 모니터링
	//   - 알람 설정 (예: 60초 이상 업데이트 없으면 알람)
	//   - 마지막 실행 시간 기록
	LastRunTimestamp prometheus.Gauge
}

// NewBatchMetrics는 BatchMetrics를 생성하고 Prometheus 레지스트리에 등록합니다.
//
// 이 함수는 cluster_name 레이블 없이 메트릭을 생성합니다.
// 단일 클러스터 환경이나 클러스터 구분이 필요 없는 경우 사용합니다.
//
// Parameters:
//   - registry: Prometheus 메트릭을 등록할 레지스트리
//     일반적으로 prometheus.DefaultRegisterer 또는 사용자 정의 레지스트리를 사용합니다.
//
// Returns:
//   - *BatchMetrics: 초기화된 BatchMetrics 구조체
//     모든 메트릭이 생성되고 레지스트리에 등록된 상태입니다.
//
// 사용 예시:
//
//	registry := prometheus.NewRegistry()
//	metrics := NewBatchMetrics(registry)
//
//	// HTTP 엔드포인트에서 메트릭 노출
//	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
func NewBatchMetrics(registry prometheus.Registerer) *BatchMetrics {
	return NewBatchMetricsWithCluster(registry, "")
}

// NewBatchMetricsWithCluster는 cluster_name 레이블과 함께 BatchMetrics를 생성합니다.
//
// 멀티 클러스터 환경에서 각 클러스터의 배치 성능을 개별적으로 추적할 때 사용합니다.
// clusterName이 빈 문자열이면 레이블 없이 메트릭을 생성합니다.
//
// Parameters:
//   - registry: Prometheus 메트릭을 등록할 레지스트리
//   - clusterName: 클러스터 이름 (빈 문자열이면 레이블 미사용)
//
// Returns:
//   - *BatchMetrics: 초기화된 BatchMetrics 구조체
//
// 사용 예시:
//
//	registry := prometheus.NewRegistry()
//	metrics := NewBatchMetricsWithCluster(registry, "production-cluster-1")
//
//	// 배치 처리 시 자동으로 cluster_name="production-cluster-1" 레이블이 붙음
//	metrics.ProcessedRows.Add(100)
func NewBatchMetricsWithCluster(registry prometheus.Registerer, clusterName string) *BatchMetrics {
	// 레이블 설정: cluster_name이 제공되면 레이블 추가
	var labels prometheus.Labels
	if clusterName != "" {
		labels = prometheus.Labels{"cluster_name": clusterName}
	}

	// ProcessedRows Counter 생성
	processedRows := prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "o11y_servicemap_batch_processed_rows_total",
		Help:        "Total number of rows processed by the network batch processor",
		ConstLabels: labels,
	})

	// ProcessingSeconds Histogram 생성
	// 버킷: [0.1, 0.5, 1, 2, 5, 10]초
	processingSeconds := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:        "o11y_servicemap_batch_processing_seconds",
		Help:        "Histogram of batch processing time in seconds",
		Buckets:     []float64{0.1, 0.5, 1, 2, 5, 10},
		ConstLabels: labels,
	})

	// Errors Counter 생성
	errors := prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "o11y_servicemap_batch_errors_total",
		Help:        "Total number of errors occurred during batch processing",
		ConstLabels: labels,
	})

	// LastRunTimestamp Gauge 생성
	lastRunTimestamp := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "o11y_servicemap_batch_last_run_timestamp",
		Help:        "Unix timestamp of the last successful batch run",
		ConstLabels: labels,
	})

	// Prometheus 레지스트리에 메트릭 등록
	// 레지스트리가 nil이 아닌 경우에만 등록합니다.
	// 이미 등록된 메트릭이 있으면 무시됩니다 (패닉 방지).
	if registry != nil {
		registry.MustRegister(processedRows)
		registry.MustRegister(processingSeconds)
		registry.MustRegister(errors)
		registry.MustRegister(lastRunTimestamp)
	}

	return &BatchMetrics{
		ProcessedRows:     processedRows,
		ProcessingSeconds: processingSeconds,
		Errors:            errors,
		LastRunTimestamp:  lastRunTimestamp,
	}
}
