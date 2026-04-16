# NetworkBatchProcessor Integration Tests

This document describes the integration tests for the NetworkBatchProcessor and how to run them.

## Test Architecture

The integration tests are split into two files:

### 1. `integration_test.go` - Mock-based Tests (Always Run)

Mock-based integration tests that **do not require** a real ClickHouse connection. These tests:

- Use `mockDB` to simulate database behavior
- Test the complete processor lifecycle (Config → Processor → Run → Shutdown)
- Verify error handling, metrics collection, and graceful shutdown
- Run automatically in CI/CD pipelines without external dependencies

**Run command:**
```bash
go test ./... -v
```

**Test count:** 14 integration tests

### 2. `integration_clickhouse_test.go` - Real DB Tests (Build Tag Required)

Integration tests that require a **real ClickHouse connection**. These tests:

- Use `//go:build integration` build tag
- Connect to actual ClickHouse database
- Verify real batch processing with watermark updates
- Test long-running stability and persistence

**Run command:**
```bash
go test -tags=integration ./... -v
```

**Test count:** 9 integration tests with real ClickHouse

## Prerequisites

### For Mock-based Tests (No Prerequisites)

Mock-based tests run out-of-the-box with no setup required.

### For Real ClickHouse Tests

Set the following environment variables:

```bash
export CLICKHOUSE_HOST=<YOUR_IP>
export CLICKHOUSE_PORT=9000
export CLICKHOUSE_DATABASE=signoz_traces
export CLICKHOUSE_USER=default
export CLICKHOUSE_PASSWORD=<CLICKHOUSE_PASSWORD>
```

Or use default values (tests will auto-detect):
- Host: `<YOUR_IP>`
- Port: `9000`
- Database: `signoz_traces`
- User: `default`
- Password: `<CLICKHOUSE_PASSWORD>`

## Running Tests

### Run All Tests (Mock-based only)

```bash
cd internal/batch
go test -v
```

**Output:**
```
=== RUN   TestNetworkBatchProcessor_Lifecycle
--- PASS: TestNetworkBatchProcessor_Lifecycle (0.35s)
...
PASS
ok      github.com/.../batch    2.906s  coverage: 96.3%
```

### Run Integration Tests with Real ClickHouse

```bash
cd internal/batch
go test -tags=integration -v
```

**Output:**
```
=== RUN   TestNetworkBatchProcessor_RealClickHouse_ProcessBatch
--- PASS: TestNetworkBatchProcessor_RealClickHouse_ProcessBatch (0.42s)
...
```

### Run Specific Test

```bash
# Mock-based test
go test -v -run TestNetworkBatchProcessor_Lifecycle

# Real ClickHouse test
go test -tags=integration -v -run TestNetworkBatchProcessor_RealClickHouse_ProcessBatch
```

### Check Coverage

```bash
go test -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out
```

**Current coverage:** 96.3% of statements

### Skip Long-running Tests

```bash
go test -tags=integration -short -v
```

This skips tests marked with `if testing.Short()` (e.g., 30-second long-running tests).

## Test Cases

### Mock-based Integration Tests

| Test | Description | Duration |
|------|-------------|----------|
| `TestNetworkBatchProcessor_Lifecycle` | Full lifecycle: Config → Run → Shutdown | 0.35s |
| `TestNetworkBatchProcessor_DisabledConfig` | Behavior when batch disabled | <0.01s |
| `TestNetworkBatchProcessor_MultipleBatchCycles` | Multiple batch execution cycles | 0.50s |
| `TestNetworkBatchProcessor_GracefulShutdown` | Context cancellation handling | 0.15s |
| `TestNetworkBatchProcessor_Integration_ErrorRecovery` | Error recovery and continuation | 0.30s |
| `TestNetworkBatchProcessor_Integration_MetricsCollection` | Metrics collection validation | 0.20s |
| `TestNetworkBatchProcessor_NilMetrics` | Operation with nil metrics | 0.20s |
| `TestNetworkBatchProcessor_ContextTimeout` | Context timeout handling | 0.10s |
| `TestNetworkBatchProcessor_ConcurrentShutdown` | Concurrent shutdown safety | 0.01s |
| `TestNetworkBatchProcessor_ZeroInterval` | Zero interval validation | <0.01s |
| `TestNetworkBatchProcessor_NegativeInterval` | Negative interval validation | <0.01s |
| `TestNetworkBatchProcessor_ValidIntervalRange` | Valid interval ranges (6 sub-tests) | <0.01s |
| `TestNetworkBatchProcessor_WithEnvironmentConfig` | Environment variable support | <0.01s |
| `TestNetworkBatchProcessor_ConfigToProcessorFlow` | Config → Processor flow | <0.01s |

### Real ClickHouse Integration Tests

| Test | Description | Duration |
|------|-------------|----------|
| `TestNetworkBatchProcessor_RealClickHouse_ProcessBatch` | Real batch processing | ~0.4s |
| `TestNetworkBatchProcessor_RealClickHouse_WatermarkUpdate` | Watermark persistence | ~0.5s |
| `TestNetworkBatchProcessor_RealClickHouse_FullCycle` | Full lifecycle with real DB | ~10s |
| `TestNetworkBatchProcessor_RealClickHouse_ConcurrentExecution` | Concurrent safety | ~8s |
| `TestNetworkBatchProcessor_RealClickHouse_ErrorHandling` | Error handling with real DB | ~6s |
| `TestNetworkBatchProcessor_RealClickHouse_LongRunning` | Long-running stability (skipped in `-short`) | ~30s |
| `TestNetworkBatchProcessor_RealClickHouse_ContextCancellation` | Context cancellation | ~5s |
| `TestNetworkBatchProcessor_RealClickHouse_WatermarkPersistence` | Watermark persistence across instances | ~0.5s |

## Test Environment

### Local Development

```bash
# Run mock tests (fast)
go test -v

# Run with real ClickHouse (requires DB)
go test -tags=integration -v
```

### CI/CD Pipeline

**Recommended approach:**

1. **Unit Tests + Mock Integration Tests** (Always run)
   ```yaml
   - name: Run Tests
     run: go test ./... -v -coverprofile=coverage.out
   ```

2. **Real ClickHouse Tests** (Optional, only when ClickHouse available)
   ```yaml
   - name: Integration Tests
     if: env.CLICKHOUSE_AVAILABLE == 'true'
     run: go test -tags=integration ./... -v
   ```

## Troubleshooting

### Mock Tests Failing

If mock-based tests fail, it indicates a problem with the processor logic itself (not database-related).

**Common issues:**
- Timer/ticker timing issues (adjust test timeouts if needed)
- Test name conflicts (ensure unique test names)

### Real ClickHouse Tests Skipped

If integration tests are skipped with message:
```
ClickHouse connection failed (test environment not available)
```

**Solutions:**
1. Verify ClickHouse is running: `nc -zv <YOUR_IP> 9000`
2. Check environment variables
3. Verify network connectivity
4. Check ClickHouse credentials

### Tests Timeout

If tests timeout:
```bash
# Increase timeout (default: 10m)
go test -timeout 30m -tags=integration -v
```

### Tests Fail with "context deadline exceeded"

This usually means ClickHouse is too slow or unreachable.

**Solutions:**
1. Check ClickHouse performance
2. Reduce test duration (not recommended)
3. Skip long-running tests: `go test -short`

## Test Metrics

### Coverage

Current test coverage: **96.3%** of statements

**Coverage breakdown:**
- `NewNetworkBatchProcessor`: 100%
- `Run`: 100%
- `processBatch`: 88.0%
- `getWatermark`: 100%
- `loadSQL`: 100%

### Test Execution Time

**Mock-based tests:** ~2.9s (14 tests)
**Real ClickHouse tests:** ~30-60s (9 tests, depending on DB performance)

## Best Practices

1. **Always run mock tests** before committing code
2. **Run real ClickHouse tests** before merging to main branch
3. **Use `-short` flag** for quick feedback during development
4. **Check coverage** regularly to maintain >80% threshold
5. **Review test logs** for performance regressions

## Related Documentation

- [TASK-012 Requirements](../../docs/tasks/TASK-012.md)
- [NetworkBatchProcessor Documentation](./README.md)
- [Testing Best Practices](../../docs/TESTING.md)
