package batch

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// mockDB is a simple mock implementation of driver.Conn for testing purposes.
//
// This mock allows us to test code paths that interact with ClickHouse
// without requiring an actual database connection during unit tests.
//
// Supported operations:
//   - Exec: Returns configurable error or nil
//   - QueryRow: Returns configurable watermark timestamp or error
type mockDB struct {
	// execError is returned by Exec() if not nil
	execError error

	// execErrorSequence is a sequence of errors for multiple Exec calls
	// If set, execError is ignored and errors are returned in sequence
	execErrorSequence []error

	// queryRowResult is the timestamp returned by QueryRow()
	queryRowResult time.Time

	// queryRowError is returned by QueryRow().Scan() if not nil
	queryRowError error

	// execCallCount tracks how many times Exec was called
	execCallCount int

	// queryRowCallCount tracks how many times QueryRow was called
	queryRowCallCount int
}

// Exec implements driver.Conn.Exec for testing
func (m *mockDB) Exec(ctx context.Context, query string, args ...interface{}) error {
	// Check context cancellation
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Use error sequence if provided
	if len(m.execErrorSequence) > 0 {
		// Get error for current call index
		index := m.execCallCount
		m.execCallCount++

		if index < len(m.execErrorSequence) {
			return m.execErrorSequence[index]
		}

		// If sequence is exhausted, return last error
		return m.execErrorSequence[len(m.execErrorSequence)-1]
	}

	// Otherwise, use single execError
	m.execCallCount++
	return m.execError
}

// QueryRow implements driver.Conn.QueryRow for testing
func (m *mockDB) QueryRow(ctx context.Context, query string, args ...interface{}) driver.Row {
	m.queryRowCallCount++

	// Return mockRow with configured result/error
	return &mockRow{
		result: m.queryRowResult,
		err:    m.queryRowError,
	}
}

// mockRow implements driver.Row for testing
type mockRow struct {
	result time.Time
	err    error
}

// Scan implements driver.Row.Scan for testing
func (r *mockRow) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}

	// Scan result into destination
	if len(dest) > 0 {
		if t, ok := dest[0].(*time.Time); ok {
			*t = r.result
			return nil
		}
	}

	return fmt.Errorf("scan destination type mismatch")
}

// Err implements driver.Row.Err for testing
func (r *mockRow) Err() error {
	return r.err
}

// ScanStruct implements driver.Row.ScanStruct for testing
func (r *mockRow) ScanStruct(dest interface{}) error {
	// Simple implementation for testing
	return r.err
}

// ============================================================
// Unused methods (required by driver.Conn interface)
// These are not used in our tests but must be implemented
// ============================================================

func (m *mockDB) Contributors() []string                        { return nil }
func (m *mockDB) ServerVersion() (*driver.ServerVersion, error) { return nil, nil }
func (m *mockDB) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return nil
}
func (m *mockDB) Query(ctx context.Context, query string, args ...interface{}) (driver.Rows, error) {
	return nil, nil
}
func (m *mockDB) PrepareBatch(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error) {
	return nil, nil
}
func (m *mockDB) AsyncInsert(ctx context.Context, query string, wait bool, args ...interface{}) error {
	return nil
}
func (m *mockDB) Ping(ctx context.Context) error { return nil }
func (m *mockDB) Stats() driver.Stats            { return driver.Stats{} }
func (m *mockDB) Close() error                   { return nil }
func (m *mockDB) SyncInsert(ctx context.Context, query string, args ...interface{}) error {
	return nil
}
