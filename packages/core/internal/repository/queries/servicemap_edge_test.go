package queries

import (
	"strings"
	"testing"
)

// TestEdgeDetailErrorDetection tests that all Edge Detail queries include OpenTelemetry span status check
// TASK-007~010: Edge Detail API 에러 감지 로직 검증
func TestEdgeDetailErrorDetection(t *testing.T) {
	tests := []struct {
		name          string
		buildQuery    func() string
		expectedChecks []string
	}{
		{
			name: "BuildQueryTopSlowRequest includes span status check",
			buildQuery: func() string {
				return BuildQueryTopSlowRequest()
			},
			expectedChecks: []string{
				"WHEN status_code = 2 THEN 1",  // OpenTelemetry span error check
				"STATUS_CODE_ERROR",             // Comment explaining the check
				"WHEN protocol = 'HTTP' AND status >= 400 THEN 1", // HTTP error check
				"WHEN protocol = 'gRPC' AND status != 0 THEN 1",   // gRPC error check
			},
		},
		{
			name: "BuildQueryRecentError includes span status check",
			buildQuery: func() string {
				return BuildQueryRecentError()
			},
			expectedChecks: []string{
				"WHEN status_code = 2 THEN 1",
				"STATUS_CODE_ERROR",
				"WHEN protocol = 'HTTP' AND status >= 400 THEN 1",
				"WHEN protocol = 'gRPC' AND status != 0 THEN 1",
			},
		},
		{
			name: "BuildQueryRequests includes span status check",
			buildQuery: func() string {
				return BuildQueryRequests()
			},
			expectedChecks: []string{
				"WHEN status_code = 2 THEN 1",
				"STATUS_CODE_ERROR",
				"WHEN protocol = 'HTTP' AND status >= 400 THEN 1",
				"WHEN protocol = 'gRPC' AND status != 0 THEN 1",
			},
		},
		{
			name: "BuildQueryTopSlowInternalToExternal includes span status check",
			buildQuery: func() string {
				return BuildQueryTopSlowInternalToExternal()
			},
			expectedChecks: []string{
				"WHEN status_code = 2 THEN 1",
				"STATUS_CODE_ERROR",
				"WHEN protocol = 'HTTP' AND status >= 400 THEN 1",
				"WHEN protocol = 'gRPC' AND status != 0 THEN 1",
			},
		},
		{
			name: "BuildQueryRecentErrorInternalToExternal includes span status check",
			buildQuery: func() string {
				return BuildQueryRecentErrorInternalToExternal()
			},
			expectedChecks: []string{
				"WHEN status_code = 2 THEN 1",
				"STATUS_CODE_ERROR",
				"WHEN protocol = 'HTTP' AND status >= 400 THEN 1",
				"WHEN protocol = 'gRPC' AND status != 0 THEN 1",
			},
		},
		{
			name: "BuildQueryRequestsInternalToExternal includes span status check",
			buildQuery: func() string {
				return BuildQueryRequestsInternalToExternal()
			},
			expectedChecks: []string{
				"WHEN status_code = 2 THEN 1",
				"STATUS_CODE_ERROR",
				"WHEN protocol = 'HTTP' AND status >= 400 THEN 1",
				"WHEN protocol = 'gRPC' AND status != 0 THEN 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := tt.buildQuery()

			// Verify all expected checks are present
			for _, check := range tt.expectedChecks {
				if !strings.Contains(query, check) {
					t.Errorf("%s: query does not contain expected check: %s", tt.name, check)
				}
			}

			// Verify span status check comes BEFORE protocol-specific checks
			spanCheckPos := strings.Index(query, "WHEN status_code = 2 THEN 1")
			httpCheckPos := strings.Index(query, "WHEN protocol = 'HTTP' AND status >= 400 THEN 1")

			if spanCheckPos == -1 {
				t.Errorf("%s: span status check not found in query", tt.name)
			}
			if httpCheckPos == -1 {
				t.Errorf("%s: HTTP status check not found in query", tt.name)
			}
			if spanCheckPos >= httpCheckPos {
				t.Errorf("%s: span status check (pos %d) should come BEFORE HTTP check (pos %d)", tt.name, spanCheckPos, httpCheckPos)
			}
		})
	}
}

// TestErrorDetectionLogic tests the error detection CASE statement logic
// TASK-007: Span Error Only (OOM, panic, exception)
func TestErrorDetectionLogic_SpanErrorOnly(t *testing.T) {
	query := BuildQueryTopSlowRequest()

	// Test Case A: Span Error Only (status_code=2, HTTP 200)
	// Expected: is_error = 1 (should be detected as error)
	t.Run("Span Error Only - OOM with HTTP 200", func(t *testing.T) {
		// Verify the query includes the span status check as the FIRST condition
		expectedPattern := "WHEN status_code = 2 THEN 1"
		if !strings.Contains(query, expectedPattern) {
			t.Errorf("Query should detect span status_code=2 as error, but pattern not found: %s", expectedPattern)
		}

		// Verify comment is present
		if !strings.Contains(query, "STATUS_CODE_ERROR") {
			t.Error("Query should include STATUS_CODE_ERROR comment for clarity")
		}
	})
}

// TASK-008: HTTP Error Only (회귀 테스트)
func TestErrorDetectionLogic_HTTPErrorOnly(t *testing.T) {
	query := BuildQueryTopSlowRequest()

	// Test Case B: HTTP Error Only (status_code=1, HTTP 500)
	// Expected: is_error = 1 (should be detected as error)
	t.Run("HTTP Error Only - HTTP 500 with Span OK", func(t *testing.T) {
		// Verify the query still includes HTTP status check
		expectedPattern := "WHEN protocol = 'HTTP' AND status >= 400 THEN 1"
		if !strings.Contains(query, expectedPattern) {
			t.Errorf("Query should detect HTTP status >= 400 as error, but pattern not found: %s", expectedPattern)
		}

		// Verify gRPC, Redis, SQL checks are also present (regression test)
		protocolChecks := []string{
			"WHEN protocol = 'gRPC' AND status != 0 THEN 1",
			"WHEN protocol = 'Redis' AND status != 0 THEN 1",
			"WHEN protocol = 'SQL' AND status != 0 THEN 1",
		}

		for _, check := range protocolChecks {
			if !strings.Contains(query, check) {
				t.Errorf("Query should include protocol check: %s", check)
			}
		}
	})
}

// TASK-009: Both Errors (span error + HTTP error)
func TestErrorDetectionLogic_BothErrors(t *testing.T) {
	query := BuildQueryTopSlowRequest()

	// Test Case C: Both Errors (status_code=2, HTTP 500)
	// Expected: is_error = 1 (should be detected as error by first condition)
	t.Run("Both Errors - Span Error and HTTP 500", func(t *testing.T) {
		// Verify OR logic: both conditions present
		spanCheck := "WHEN status_code = 2 THEN 1"
		httpCheck := "WHEN protocol = 'HTTP' AND status >= 400 THEN 1"

		if !strings.Contains(query, spanCheck) {
			t.Error("Query should include span status check")
		}
		if !strings.Contains(query, httpCheck) {
			t.Error("Query should include HTTP status check")
		}

		// Verify priority: span check comes first
		spanPos := strings.Index(query, spanCheck)
		httpPos := strings.Index(query, httpCheck)

		if spanPos >= httpPos {
			t.Errorf("Span status check should have higher priority (come before HTTP check)")
		}
	})
}

// TASK-010: No Errors (정상 케이스)
func TestErrorDetectionLogic_NoErrors(t *testing.T) {
	query := BuildQueryTopSlowRequest()

	// Test Case D: No Errors (status_code=0 or 1, HTTP 200)
	// Expected: is_error = 0 (should NOT be detected as error)
	t.Run("No Errors - Normal Request", func(t *testing.T) {
		// Verify ELSE 0 is present (default to no error)
		expectedElse := "ELSE 0"
		if !strings.Contains(query, expectedElse) {
			t.Errorf("Query should default to is_error=0 for normal requests: %s", expectedElse)
		}

		// Verify COALESCE with default 0
		expectedCoalesce := "COALESCE"
		if !strings.Contains(query, expectedCoalesce) {
			t.Error("Query should use COALESCE to handle NULL values")
		}
	})
}

// TestAllQueriesConsistency verifies all 6 queries have identical error detection logic
func TestAllQueriesConsistency(t *testing.T) {
	queries := []struct {
		name  string
		query string
	}{
		{"TopSlowRequest", BuildQueryTopSlowRequest()},
		{"RecentError", BuildQueryRecentError()},
		{"Requests", BuildQueryRequests()},
		{"TopSlowInternalToExternal", BuildQueryTopSlowInternalToExternal()},
		{"RecentErrorInternalToExternal", BuildQueryRecentErrorInternalToExternal()},
		{"RequestsInternalToExternal", BuildQueryRequestsInternalToExternal()},
	}

	// Expected error detection pattern (consistent across all queries)
	expectedPattern := []string{
		"WHEN status_code = 2 THEN 1",  // Span error
		"STATUS_CODE_ERROR",
		"WHEN protocol = 'HTTP' AND status >= 400 THEN 1",
		"WHEN protocol = 'gRPC' AND status != 0 THEN 1",
		"WHEN protocol = 'Redis' AND status != 0 THEN 1",
		"WHEN protocol = 'SQL' AND status != 0 THEN 1",
		"ELSE 0",
	}

	for _, q := range queries {
		t.Run(q.name, func(t *testing.T) {
			for _, pattern := range expectedPattern {
				if !strings.Contains(q.query, pattern) {
					t.Errorf("%s: missing expected pattern: %s", q.name, pattern)
				}
			}
		})
	}
}
