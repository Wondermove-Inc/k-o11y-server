package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

var (
	clickHouseConn driver.Conn = nil
	database       string      = ""
)

// ClickHouseConfig 구조체 정의 (호환성 유지)
type ClickHouseConfig struct {
	Host       string
	Port       int
	Database   string
	Username   string
	Password   string
	Timeout    string
	MaxRetries int
}

// ConnectClickHouse ClickHouse 연결 (공식 문서 권장 방식)
func ConnectClickHouse(config ClickHouseConfig) error {
	if clickHouseConn != nil {
		fmt.Println("[ClickHouse] Already connected")
		return nil
	}

	// 공식 문서 권장: clickhouse.Options 사용
	options := &clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
		// 공식 권장: Connection Pool & Performance
		MaxOpenConns:    20,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
		// 공식 권장: 압축 최적화
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		Settings: clickhouse.Settings{
			"max_execution_time":            120, // 타임아웃 통일 (120초)
			"send_progress_in_http_headers": 1,
			"connect_timeout":               30,  // 연결 타임아웃
			"receive_timeout":               120, // 수신 타임아웃
			"send_timeout":                  120, // 전송 타임아웃
			"tcp_keep_alive_timeout":        600, // Keep-alive (10분)
		},
		DialTimeout: 30 * time.Second,
		// 연결 안정성 강화 (Tavily 검색 기반 권장사항)
		ConnOpenStrategy: clickhouse.ConnOpenInOrder, // 순차 연결 시도
		// 공식 권장: 대용량 데이터 처리 최적화
		BlockBufferSize: 10,
		Debug:           false,
	}

	var err error
	// 공식 문서 방식: Native API 연결
	clickHouseConn, err = clickhouse.Open(options)
	if err != nil {
		return fmt.Errorf("failed to connect ClickHouse: %w", err)
	}

	// 공식 방식: 연결 확인
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := clickHouseConn.Ping(ctx); err != nil {
		return fmt.Errorf("clickHouse ping failed: %w", err)
	}

	fmt.Println("[ClickHouse] ✅ Connected to ClickHouse with Native API")
	database = config.Database

	return nil
}

// QueryClickHouseWithContext Context를 전파하는 ClickHouse 쿼리 실행
func QueryClickHouseWithContext(ctx context.Context, query string, args ...interface{}) (driver.Rows, error) {
	if clickHouseConn == nil {
		return nil, fmt.Errorf("clickhouse connection not initialized")
	}

	// 연결 상태 재확인 (포트 포워딩 끊김 감지) - 별도 context 사용
	// 타임아웃을 5초에서 30초로 증가 (간헐적 네트워크 지연으로 인한 ping 실패 해결)
	// 기존: 5초 → 변경: 30초 (NetworkMap 쿼리 타임아웃 문제 해결을 위한 테스트)
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer pingCancel()

	if err := clickHouseConn.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("clickhouse connection lost (possibly port-forward disconnected): %w", err)
	}

	// 전달받은 context 사용하여 쿼리 실행 (Context 전파)
	rows, err := clickHouseConn.Query(ctx, query, args...)
	if err != nil {
		// Context 취소 에러 명확하게 처리 (Tavily 검색 기반)
		if ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("query canceled by context: %w", ctx.Err())
		}
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("query timeout: %w", ctx.Err())
		}
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return rows, nil
}

// QueryRowClickHouse 단일 행 쿼리 (타입 안전 방식)
func QueryRowClickHouse(query string, args ...interface{}) (interface{}, error) {
	if clickHouseConn == nil {
		return nil, fmt.Errorf("clickhouse connection not initialized")
	}

	// Context timeout 설정
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Query 방식으로 단일 결과 처리
	rows, err := clickHouseConn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("single query failed: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		// ClickHouse SELECT 1은 UInt8 타입 반환 - 타입 안전 처리
		var result uint8
		if scanErr := rows.Scan(&result); scanErr != nil {
			return nil, fmt.Errorf("scan failed: %w", scanErr)
		}
		return result, nil
	}

	return nil, fmt.Errorf("no rows returned")
}

// GetClickHouseConn returns the global ClickHouse connection instance.
//
// This function provides access to the established ClickHouse connection
// for use by other packages (e.g., batch processing).
//
// Returns:
//   - driver.Conn: The active ClickHouse connection, or nil if not connected
//
// Usage example:
//
//	conn := infrastructure.GetClickHouseConn()
//	if conn == nil {
//	    return fmt.Errorf("ClickHouse connection not available")
//	}
//	processor := batch.NewNetworkBatchProcessor(conn, interval, logger, metrics)
func GetClickHouseConn() driver.Conn {
	return clickHouseConn
}
