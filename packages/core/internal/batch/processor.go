package batch

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.uber.org/zap"
)

// sqlFiles embeds all SQL files from the sql directory.
//
// This uses Go's embed package to include SQL files in the compiled binary,
// eliminating the need for external file dependencies at runtime.
//
// Embedded files:
//   - sql/network_insert.sql: INSERT query for network_map_connections table
//   - sql/watermark_update.sql: UPDATE query for watermark tracking
//
//go:embed sql/*.sql
var sqlFiles embed.FS

// NewServiceMapBatchProcessor creates and initializes a new ServiceMapBatchProcessor instance.
//
// This constructor function validates all required dependencies and returns a properly
// configured processor ready to execute batch operations.
//
// Parameters:
//   - db: ClickHouse database connection (must not be nil)
//   - interval: Batch execution interval (must be > 0)
//   - logger: Structured logger for operational logs (must not be nil)
//   - metrics: Prometheus metrics collector (can be nil if metrics not needed)
//
// Returns:
//   - *ServiceMapBatchProcessor: Initialized processor instance
//   - error: Validation error if any parameter is invalid
//
// Validation rules:
//   - db connection must be non-nil
//   - interval must be greater than 0
//   - logger must be non-nil
//   - metrics can be nil (optional)
//
// Usage example:
//
//	processor, err := batch.NewServiceMapBatchProcessor(
//	    clickhouseConn,
//	    20 * time.Second,
//	    zapLogger,
//	    batchMetrics,
//	)
//	if err != nil {
//	    log.Fatalf("Failed to create processor: %v", err)
//	}
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	go processor.Run(ctx)
func NewServiceMapBatchProcessor(
	db driver.Conn,
	interval time.Duration,
	insertTimeout time.Duration,
	safetyBuffer time.Duration,
	maxWindow time.Duration,
	logger *zap.Logger,
	metrics *BatchMetrics,
) (*ServiceMapBatchProcessor, error) {
	// Validate required parameters
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}
	if interval <= 0 {
		return nil, fmt.Errorf("interval must be greater than 0, got %v", interval)
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	// Default insertTimeout to 120s if not specified
	if insertTimeout <= 0 {
		insertTimeout = 120 * time.Second
	}

	// Default safetyBuffer to 20s if not specified
	if safetyBuffer <= 0 {
		safetyBuffer = 20 * time.Second
	}

	// Default maxWindow to 30s if not specified
	if maxWindow <= 0 {
		maxWindow = 30 * time.Second
	}

	return &ServiceMapBatchProcessor{
		db:            db,
		interval:      interval,
		insertTimeout: insertTimeout,
		safetyBuffer:  safetyBuffer,
		maxWindow:     maxWindow,
		logger:        logger,
		metrics:       metrics,
	}, nil
}

// ServiceMapBatchProcessor는 서비스맵 연결 데이터를 주기적으로 처리하는 배치 프로세서입니다.
//
// 이 프로세서는 ClickHouse의 network_raw 테이블에서 데이터를 읽어
// network_map_connections 테이블로 집계 및 변환하는 작업을 수행합니다.
//
// 주요 기능:
//   - 주기적 배치 실행 (기본 20초 간격)
//   - Context 기반 우아한 종료 (graceful shutdown)
//   - 메트릭 수집 및 모니터링
//   - 에러 처리 및 로깅
//
// 사용 예시:
//
//	processor := &ServiceMapBatchProcessor{
//	    db:       clickhouseConn,
//	    interval: 20 * time.Second,
//	    logger:   zapLogger,
//	    metrics:  batchMetrics,
//	}
//	ctx := context.Background()
//	processor.Run(ctx)
type ServiceMapBatchProcessor struct {
	// db는 ClickHouse 데이터베이스 연결입니다.
	// Native API (clickhouse-go/v2)를 사용하여 배치 쿼리를 실행합니다.
	db driver.Conn

	// interval은 배치 작업 실행 주기입니다.
	// 기본값은 20초이며, 환경 변수로 조정 가능합니다.
	interval time.Duration

	// insertTimeout은 INSERT 쿼리 실행 타임아웃입니다.
	// 환경 변수: BATCH_INSERT_TIMEOUT (기본값: 120s)
	insertTimeout time.Duration

	// safetyBuffer는 데이터 안정화를 위해 now()에서 빼는 시간입니다.
	// 늦게 도착하는 데이터를 포함하기 위한 버퍼입니다.
	// 환경 변수: BATCH_SAFETY_BUFFER (기본값: 20s)
	safetyBuffer time.Duration

	// maxWindow는 한 배치가 처리하는 최대 시간 범위입니다.
	// Snowball 효과를 방지하여 배치 처리 시간을 안정화합니다.
	// gap > maxWindow이면 catch-up 모드로 maxWindow 크기씩 순차 처리합니다.
	// 환경 변수: BATCH_MAX_WINDOW (기본값: 30s)
	maxWindow time.Duration

	// logger는 구조화된 로깅을 위한 Zap logger입니다.
	// 배치 실행 상태, 에러, 성능 지표 등을 기록합니다.
	logger *zap.Logger

	// metrics는 배치 프로세서의 성능 지표를 수집합니다.
	// 처리된 레코드 수, 실행 시간, 에러 횟수 등을 추적합니다.
	// BatchMetrics 구조체는 Prometheus 메트릭을 관리합니다.
	metrics *BatchMetrics
}

// Run은 배치 프로세서를 시작하고 주기적으로 배치 작업을 실행합니다.
//
// Completion-based scheduling을 사용하여 배치 완료 후 interval만큼 대기합니다.
// Catch-up 모드: watermark가 maxWindow 이상 뒤처지면 interval 대기 없이 즉시 다음 배치를 실행합니다.
//
// Parameters:
//   - ctx: 배치 프로세서의 생명주기를 제어하는 context
//     context가 취소되면 현재 실행 중인 배치 작업을 완료한 후 종료합니다.
//
// 동작 방식:
//  1. processBatch()를 실행합니다.
//  2. watermark와 현재 시간의 gap을 확인합니다.
//  3. gap > maxWindow이면 catch-up: 즉시 다음 배치를 실행합니다.
//  4. gap <= maxWindow이면 정상: interval만큼 대기 후 다음 배치를 실행합니다.
//  5. context가 취소되면 우아하게 종료합니다.
func (p *ServiceMapBatchProcessor) Run(ctx context.Context) {
	p.logger.Info("ServiceMapBatchProcessor started",
		zap.Duration("interval", p.interval),
		zap.Duration("safetyBuffer", p.safetyBuffer),
		zap.Duration("maxWindow", p.maxWindow))

	for {
		// Check context before processing
		select {
		case <-ctx.Done():
			p.logger.Info("ServiceMapBatchProcessor shutting down")
			return
		default:
		}

		// Execute batch processing
		start := time.Now()
		if err := p.processBatch(ctx); err != nil {
			p.logger.Error("batch processing failed", zap.Error(err))
			if p.metrics != nil {
				p.metrics.Errors.Inc()
			}
		} else if p.metrics != nil {
			p.metrics.ProcessingSeconds.Observe(time.Since(start).Seconds())
			p.metrics.LastRunTimestamp.SetToCurrentTime()
		}

		// Catch-up decision: if watermark is behind by more than maxWindow,
		// skip interval wait and process next batch immediately
		watermark, err := p.getWatermark(ctx)
		if err == nil {
			gap := time.Now().UTC().Add(-p.safetyBuffer).Sub(watermark)
			if gap > p.maxWindow {
				p.logger.Info("catch-up mode: skipping interval wait",
					zap.Duration("gap", gap),
					zap.Duration("maxWindow", p.maxWindow))
				continue
			}
		}

		// Normal mode: wait for interval before next batch
		select {
		case <-ctx.Done():
			p.logger.Info("ServiceMapBatchProcessor shutting down")
			return
		case <-time.After(p.interval):
		}
	}
}

// processBatch는 단일 배치 작업을 실행합니다.
//
// Upper Bound Injection + Bounded Window 방식으로 데이터 유실 없이 처리합니다.
// INSERT와 watermark update가 동일한 (watermark, upperBound] 범위를 사용하여
// now64(9) 평가 시점 차이로 인한 데이터 유실을 방지합니다.
//
// Bounded Window: upperBound = min(now()-safetyBuffer, watermark+maxWindow)
// 이를 통해 한 배치가 처리하는 데이터량을 제한하여 snowball 효과를 방지합니다.
//
// Parameters:
//   - ctx: 배치 작업의 타임아웃 및 취소를 제어하는 context
//
// Returns:
//   - error: 배치 처리 중 발생한 에러 (nil이면 성공)
func (p *ServiceMapBatchProcessor) processBatch(ctx context.Context) error {
	batchStartTime := time.Now()

	// Step 1: Get watermark (last processed timestamp)
	watermark, err := p.getWatermark(ctx)
	if err != nil {
		return fmt.Errorf("failed to get watermark: %w", err)
	}

	// Step 2: Calculate upper bound with bounded window
	now := time.Now().UTC()
	upperBound := now.Add(-p.safetyBuffer)

	// Bounded Window: cap at watermark + maxWindow to prevent snowball
	maxUpperBound := watermark.Add(p.maxWindow)
	if upperBound.After(maxUpperBound) {
		upperBound = maxUpperBound
		p.logger.Info("bounded window applied",
			zap.Time("watermark", watermark),
			zap.Time("upperBound", upperBound),
			zap.Duration("maxWindow", p.maxWindow))
	}

	// Skip if no data to process
	if !upperBound.After(watermark) {
		p.logger.Debug("no data to process",
			zap.Time("watermark", watermark),
			zap.Time("upperBound", upperBound))
		return nil
	}

	p.logger.Debug("starting batch processing",
		zap.Time("watermark", watermark),
		zap.Time("upperBound", upperBound))

	// Step 3: Load and inject SQL parameters
	insertSQL, err := loadSQL("network_insert.sql")
	if err != nil {
		return fmt.Errorf("failed to load insert SQL: %w", err)
	}

	watermarkStr := watermark.UTC().Format("2006-01-02 15:04:05.000000000")
	upperBoundStr := upperBound.Format("2006-01-02 15:04:05.000000000")

	// Inject both watermark and upper bound into INSERT SQL
	insertSQL = strings.Replace(insertSQL, "{{WATERMARK_TS}}", watermarkStr, 1)
	insertSQL = strings.Replace(insertSQL, "{{UPPER_BOUND_TS}}", upperBoundStr, 1)

	// Step 4: Execute INSERT query with timeout
	timeout := p.insertTimeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Handle nil DB connection (testing or initialization phase)
	if p.db == nil {
		p.logger.Debug("no database connection, skipping INSERT execution")
		return nil
	}

	if err := p.db.Exec(execCtx, insertSQL); err != nil {
		return fmt.Errorf("failed to execute INSERT: %w", err)
	}

	p.logger.Debug("INSERT query executed successfully")

	// Step 5: Update watermark to upperBound (only if INSERT succeeded)
	// Uses parameterized query because clickhouse-go/v2 native protocol
	// silently ignores INSERT...VALUES with SQL functions (toDateTime64, now64).
	updateQuery := "INSERT INTO signoz_traces.network_batch_watermark (id, last_processed_ts, updated_at) VALUES (?, ?, ?)"
	if err := p.db.Exec(execCtx, updateQuery, uint8(1), upperBound, time.Now()); err != nil {
		return fmt.Errorf("failed to update watermark: %w", err)
	}

	p.logger.Debug("watermark updated successfully",
		zap.String("newWatermark", upperBoundStr))

	// Step 6: Log completion
	processingDuration := time.Since(batchStartTime)
	p.logger.Info("batch processing completed",
		zap.Time("watermark", watermark),
		zap.Time("upperBound", upperBound),
		zap.Duration("processing_time", processingDuration))

	return nil
}

// getWatermark retrieves the last processed timestamp from the watermark table.
//
// This method queries the network_batch_watermark table to get the last successfully
// processed timestamp. If no watermark exists (first run or table empty), it returns
// a default value of current time - 1 hour.
//
// Parameters:
//   - ctx: Context for query timeout and cancellation control
//
// Returns:
//   - time.Time: Last processed timestamp or default value (now - 1 hour)
//   - error: Error if query fails (returns default value on sql.ErrNoRows)
//
// Query executed:
//
//	SELECT last_processed_ts FROM signoz_traces.network_batch_watermark FINAL
//
// Error handling:
//   - sql.ErrNoRows: Returns default watermark (now - 1 hour) with no error
//   - DB connection error: Returns error immediately
//   - Query timeout: Returns error with context info
//
// Example usage:
//
//	watermark, err := p.getWatermark(ctx)
//	if err != nil {
//	    return fmt.Errorf("failed to get watermark: %w", err)
//	}
//	p.logger.Info("processing from watermark", zap.Time("watermark", watermark))
func (p *ServiceMapBatchProcessor) getWatermark(ctx context.Context) (time.Time, error) {
	// Context with timeout (maximum 5 seconds for watermark query)
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Default watermark: current time - 15 minutes
	defaultWatermark := time.Now().UTC().Add(-15 * time.Minute)

	// Handle nil DB connection (testing or initialization phase)
	if p.db == nil {
		p.logger.Debug("no database connection, using default watermark",
			zap.Time("watermark", defaultWatermark))
		return defaultWatermark, nil
	}

	// Query watermark table
	query := "SELECT last_processed_ts FROM signoz_traces.network_batch_watermark FINAL"

	var watermark time.Time
	row := p.db.QueryRow(queryCtx, query)
	if err := row.Scan(&watermark); err != nil {
		// Check if no rows exist (first run or empty table)
		if err.Error() == "sql: no rows in result set" {
			p.logger.Debug("no watermark found, using default",
				zap.Time("watermark", defaultWatermark))
			return defaultWatermark, nil
		}

		// Other errors should be returned
		return time.Time{}, fmt.Errorf("failed to get watermark: %w", err)
	}

	// Log retrieved watermark at DEBUG level
	p.logger.Debug("watermark retrieved from database",
		zap.Time("watermark", watermark))

	return watermark, nil
}

// loadSQL loads an embedded SQL file and returns its content as a string.
//
// This function reads SQL files that are embedded in the binary using Go's embed package.
// The files are located in the sql/ subdirectory relative to this package.
//
// Parameters:
//   - filename: Name of the SQL file to load (e.g., "network_insert.sql")
//     The filename should NOT include the "sql/" prefix.
//
// Returns:
//   - string: Content of the SQL file
//   - error: Error if the file cannot be read, wrapped with context
//
// Example:
//
//	insertSQL, err := loadSQL("network_insert.sql")
//	if err != nil {
//	    return fmt.Errorf("failed to load insert SQL: %w", err)
//	}
//
// Error Cases:
//   - File not found: Returns error with "file not found" message
//   - Read failure: Returns error with "failed to read" message
//   - All errors are wrapped using fmt.Errorf with %w for error chain preservation
func loadSQL(filename string) (string, error) {
	// Read the embedded SQL file
	content, err := sqlFiles.ReadFile("sql/" + filename)
	if err != nil {
		return "", fmt.Errorf("failed to load SQL file %s: %w", filename, err)
	}

	// Convert bytes to string and return
	return string(content), nil
}
