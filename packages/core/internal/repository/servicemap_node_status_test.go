package repository

import (
	"testing"

	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/domain/servicemap"
)

// TestDetermineNodeStatus tests the determineNodeStatus function
func TestDetermineNodeStatus(t *testing.T) {
	tests := []struct {
		name        string
		totalErrors uint64
		expected    string
	}{
		{
			name:        "No errors should return Ok",
			totalErrors: 0,
			expected:    "Ok",
		},
		{
			name:        "Single error should return Error",
			totalErrors: 1,
			expected:    "Error",
		},
		{
			name:        "Multiple errors should return Error",
			totalErrors: 87,
			expected:    "Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineNodeStatus(tt.totalErrors)
			if result != tt.expected {
				t.Errorf("determineNodeStatus(%d) = %s, want %s", tt.totalErrors, result, tt.expected)
			}
		})
	}
}

// TestNodeAccumulation_SingleEdge tests single edge scenarios (regression test)
// TASK-007: 단일 Edge 테스트 (회귀 테스트)
func TestNodeAccumulation_SingleEdge(t *testing.T) {
	tests := []struct {
		name              string
		srcWorkload       string
		destWorkload      string
		totalErrors       uint64
		isExternal        *uint8
		expectedSrcStatus string
		expectedDstStatus string
		expectedSrcCount  int
		expectedDstCount  int
	}{
		{
			name:              "Single internal edge with no errors",
			srcWorkload:       "service-a",
			destWorkload:      "service-b",
			totalErrors:       0,
			isExternal:        uint8Ptr(0),
			expectedSrcStatus: "Ok",
			expectedDstStatus: "Ok",
			expectedSrcCount:  0,
			expectedDstCount:  0,
		},
		{
			name:              "Single internal edge with errors",
			srcWorkload:       "service-a",
			destWorkload:      "service-b",
			totalErrors:       10,
			isExternal:        uint8Ptr(0),
			expectedSrcStatus: "Error",
			expectedDstStatus: "Error",
			expectedSrcCount:  10,
			expectedDstCount:  10,
		},
		{
			name:              "Single external edge with errors",
			srcWorkload:       "service-a",
			destWorkload:      "external-api",
			totalErrors:       5,
			isExternal:        uint8Ptr(1),
			expectedSrcStatus: "Ok",
			expectedDstStatus: "Error",
			expectedSrcCount:  5,
			expectedDstCount:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := make(map[string]*servicemap.ServiceMapNode)
			cluster := "test-cluster"
			namespace := "default"

			// Simulate first edge processing
			srcNodeID := cluster + "$" + namespace + "$" + tt.srcWorkload
			destNodeID := cluster + "$" + namespace + "$" + tt.destWorkload

			// Source node creation (first edge)
			zero := uint8(0)
			status := "Ok"
			if tt.isExternal == nil || *tt.isExternal == 0 {
				status = determineNodeStatus(tt.totalErrors)
			}

			nodes[srcNodeID] = &servicemap.ServiceMapNode{
				WorkloadName: tt.srcWorkload,
				Namespace:    namespace,
				Cluster:      cluster,
				IssueCount:   int(tt.totalErrors),
				IsExternal:   &zero,
				Status:       status,
			}

			// Destination node creation (first edge)
			nodes[destNodeID] = &servicemap.ServiceMapNode{
				WorkloadName: tt.destWorkload,
				Namespace:    namespace,
				Cluster:      cluster,
				IssueCount:   int(tt.totalErrors),
				IsExternal:   tt.isExternal,
				Status:       determineNodeStatus(tt.totalErrors),
			}

			// Verify results
			srcNode := nodes[srcNodeID]
			if srcNode.Status != tt.expectedSrcStatus {
				t.Errorf("Source node status = %s, want %s", srcNode.Status, tt.expectedSrcStatus)
			}
			if srcNode.IssueCount != tt.expectedSrcCount {
				t.Errorf("Source node IssueCount = %d, want %d", srcNode.IssueCount, tt.expectedSrcCount)
			}

			destNode := nodes[destNodeID]
			if destNode.Status != tt.expectedDstStatus {
				t.Errorf("Destination node status = %s, want %s", destNode.Status, tt.expectedDstStatus)
			}
			if destNode.IssueCount != tt.expectedDstCount {
				t.Errorf("Destination node IssueCount = %d, want %d", destNode.IssueCount, tt.expectedDstCount)
			}
		})
	}
}

// TestNodeAccumulation_MultiEdge_Internal tests multiple internal edges
// TASK-008: 다중 Edge Internal 테스트
func TestNodeAccumulation_MultiEdge_Internal(t *testing.T) {
	nodes := make(map[string]*servicemap.ServiceMapNode)
	cluster := "test-cluster"
	namespace := "default"

	// Scenario: backoffice → client-demo-oom (87 errors)
	//           backoffice → payment-service (0 errors)

	edges := []struct {
		src         string
		dest        string
		totalErrors uint64
		isExternal  *uint8
	}{
		{"backoffice", "client-demo-oom", 87, uint8Ptr(0)},
		{"backoffice", "payment-service", 0, uint8Ptr(0)},
	}

	for _, edge := range edges {
		srcNodeID := cluster + "$" + namespace + "$" + edge.src
		destNodeID := cluster + "$" + namespace + "$" + edge.dest

		// Source node processing (with accumulation logic)
		if existingNode, exists := nodes[srcNodeID]; exists {
			// Existing node: accumulate errors
			existingNode.IssueCount += int(edge.totalErrors)

			// Internal target: update status
			if edge.isExternal == nil || *edge.isExternal == 0 {
				existingNode.Status = determineNodeStatus(uint64(existingNode.IssueCount))
			}
		} else {
			// New node creation
			zero := uint8(0)
			status := "Ok"
			if edge.isExternal == nil || *edge.isExternal == 0 {
				status = determineNodeStatus(edge.totalErrors)
			}

			nodes[srcNodeID] = &servicemap.ServiceMapNode{
				WorkloadName: edge.src,
				Namespace:    namespace,
				Cluster:      cluster,
				IssueCount:   int(edge.totalErrors),
				IsExternal:   &zero,
				Status:       status,
			}
		}

		// Destination node processing (with accumulation logic)
		if existingNode, exists := nodes[destNodeID]; exists {
			// Existing node: accumulate errors and update status
			existingNode.IssueCount += int(edge.totalErrors)
			existingNode.Status = determineNodeStatus(uint64(existingNode.IssueCount))
		} else {
			// New node creation
			nodes[destNodeID] = &servicemap.ServiceMapNode{
				WorkloadName: edge.dest,
				Namespace:    namespace,
				Cluster:      cluster,
				IssueCount:   int(edge.totalErrors),
				IsExternal:   edge.isExternal,
				Status:       determineNodeStatus(edge.totalErrors),
			}
		}
	}

	// Verify backoffice node
	backofficeNodeID := cluster + "$" + namespace + "$backoffice"
	backoffice := nodes[backofficeNodeID]
	if backoffice == nil {
		t.Fatal("backoffice node not found")
	}
	if backoffice.IssueCount != 87 {
		t.Errorf("backoffice IssueCount = %d, want 87", backoffice.IssueCount)
	}
	if backoffice.Status != "Error" {
		t.Errorf("backoffice Status = %s, want Error", backoffice.Status)
	}

	// Verify client-demo-oom node
	oomNodeID := cluster + "$" + namespace + "$client-demo-oom"
	oom := nodes[oomNodeID]
	if oom == nil {
		t.Fatal("client-demo-oom node not found")
	}
	if oom.IssueCount != 87 {
		t.Errorf("client-demo-oom IssueCount = %d, want 87", oom.IssueCount)
	}
	if oom.Status != "Error" {
		t.Errorf("client-demo-oom Status = %s, want Error", oom.Status)
	}

	// Verify payment-service node
	paymentNodeID := cluster + "$" + namespace + "$payment-service"
	payment := nodes[paymentNodeID]
	if payment == nil {
		t.Fatal("payment-service node not found")
	}
	if payment.IssueCount != 0 {
		t.Errorf("payment-service IssueCount = %d, want 0", payment.IssueCount)
	}
	if payment.Status != "Ok" {
		t.Errorf("payment-service Status = %s, want Ok", payment.Status)
	}
}

// TestNodeAccumulation_MultiEdge_External tests multiple edges with external services
// TASK-009: 다중 Edge External 테스트
func TestNodeAccumulation_MultiEdge_External(t *testing.T) {
	nodes := make(map[string]*servicemap.ServiceMapNode)
	cluster := "test-cluster"
	namespace := "external-secrets"

	// Scenario: external-secrets → sts.amazonaws.com (0 errors)
	//           external-secrets → secretsmanager.amazonaws.com (2 errors)

	edges := []struct {
		src         string
		srcNS       string
		dest        string
		destNS      string
		totalErrors uint64
		isExternal  *uint8
	}{
		{"external-secrets", "external-secrets", "sts.us-east-1.amazonaws.com", "unknown", 0, uint8Ptr(1)},
		{"external-secrets", "external-secrets", "secretsmanager.us-east-1.amazonaws.com", "unknown", 2, uint8Ptr(1)},
	}

	for _, edge := range edges {
		srcNodeID := cluster + "$" + edge.srcNS + "$" + edge.src
		destNodeID := cluster + "$" + edge.destNS + "$" + edge.dest

		// Source node processing (with accumulation logic)
		if existingNode, exists := nodes[srcNodeID]; exists {
			// Existing node: accumulate errors
			existingNode.IssueCount += int(edge.totalErrors)

			// External target: Status remains "Ok" (important!)
			if edge.isExternal == nil || *edge.isExternal == 0 {
				existingNode.Status = determineNodeStatus(uint64(existingNode.IssueCount))
			}
		} else {
			// New node creation
			zero := uint8(0)
			status := "Ok"
			if edge.isExternal == nil || *edge.isExternal == 0 {
				status = determineNodeStatus(edge.totalErrors)
			}

			nodes[srcNodeID] = &servicemap.ServiceMapNode{
				WorkloadName: edge.src,
				Namespace:    edge.srcNS,
				Cluster:      cluster,
				IssueCount:   int(edge.totalErrors),
				IsExternal:   &zero,
				Status:       status,
			}
		}

		// Destination node processing (with accumulation logic)
		if existingNode, exists := nodes[destNodeID]; exists {
			// Existing node: accumulate errors and update status
			existingNode.IssueCount += int(edge.totalErrors)
			existingNode.Status = determineNodeStatus(uint64(existingNode.IssueCount))
		} else {
			// New node creation
			nodes[destNodeID] = &servicemap.ServiceMapNode{
				WorkloadName: edge.dest,
				Namespace:    edge.destNS,
				Cluster:      cluster,
				IssueCount:   int(edge.totalErrors),
				IsExternal:   edge.isExternal,
				Status:       determineNodeStatus(edge.totalErrors),
			}
		}
	}

	// Verify external-secrets node (CRITICAL TEST)
	secretsNodeID := cluster + "$" + namespace + "$external-secrets"
	secrets := nodes[secretsNodeID]
	if secrets == nil {
		t.Fatal("external-secrets node not found")
	}
	if secrets.IssueCount != 2 {
		t.Errorf("external-secrets IssueCount = %d, want 2 (참고용)", secrets.IssueCount)
	}
	// CRITICAL: Status must be "Ok" for external targets
	if secrets.Status != "Ok" {
		t.Errorf("external-secrets Status = %s, want Ok (External 타겟이므로 Ok 유지)", secrets.Status)
	}

	// Verify sts.amazonaws.com node
	stsNodeID := cluster + "$unknown$sts.us-east-1.amazonaws.com"
	sts := nodes[stsNodeID]
	if sts == nil {
		t.Fatal("sts.amazonaws.com node not found")
	}
	if sts.IssueCount != 0 {
		t.Errorf("sts.amazonaws.com IssueCount = %d, want 0", sts.IssueCount)
	}
	if sts.Status != "Ok" {
		t.Errorf("sts.amazonaws.com Status = %s, want Ok", sts.Status)
	}

	// Verify secretsmanager.amazonaws.com node
	smNodeID := cluster + "$unknown$secretsmanager.us-east-1.amazonaws.com"
	sm := nodes[smNodeID]
	if sm == nil {
		t.Fatal("secretsmanager.amazonaws.com node not found")
	}
	if sm.IssueCount != 2 {
		t.Errorf("secretsmanager.amazonaws.com IssueCount = %d, want 2", sm.IssueCount)
	}
	if sm.Status != "Error" {
		t.Errorf("secretsmanager.amazonaws.com Status = %s, want Error", sm.Status)
	}
}

// TestNodeAccumulation_MixedScenario tests mixed internal and external edges
func TestNodeAccumulation_MixedScenario(t *testing.T) {
	nodes := make(map[string]*servicemap.ServiceMapNode)
	cluster := "test-cluster"
	namespace := "default"

	// Scenario: api-gateway → auth-service (5 errors, internal)
	//           api-gateway → redis.external.com (10 errors, external)

	edges := []struct {
		src         string
		dest        string
		totalErrors uint64
		isExternal  *uint8
	}{
		{"api-gateway", "auth-service", 5, uint8Ptr(0)},
		{"api-gateway", "redis.external.com", 10, uint8Ptr(1)},
	}

	for _, edge := range edges {
		srcNodeID := cluster + "$" + namespace + "$" + edge.src
		destNodeID := cluster + "$" + namespace + "$" + edge.dest

		// Source node processing
		if existingNode, exists := nodes[srcNodeID]; exists {
			existingNode.IssueCount += int(edge.totalErrors)

			if edge.isExternal == nil || *edge.isExternal == 0 {
				existingNode.Status = determineNodeStatus(uint64(existingNode.IssueCount))
			}
		} else {
			zero := uint8(0)
			status := "Ok"
			if edge.isExternal == nil || *edge.isExternal == 0 {
				status = determineNodeStatus(edge.totalErrors)
			}

			nodes[srcNodeID] = &servicemap.ServiceMapNode{
				WorkloadName: edge.src,
				Namespace:    namespace,
				Cluster:      cluster,
				IssueCount:   int(edge.totalErrors),
				IsExternal:   &zero,
				Status:       status,
			}
		}

		// Destination node processing
		if existingNode, exists := nodes[destNodeID]; exists {
			existingNode.IssueCount += int(edge.totalErrors)
			existingNode.Status = determineNodeStatus(uint64(existingNode.IssueCount))
		} else {
			nodes[destNodeID] = &servicemap.ServiceMapNode{
				WorkloadName: edge.dest,
				Namespace:    namespace,
				Cluster:      cluster,
				IssueCount:   int(edge.totalErrors),
				IsExternal:   edge.isExternal,
				Status:       determineNodeStatus(edge.totalErrors),
			}
		}
	}

	// Verify api-gateway node
	gatewayNodeID := cluster + "$" + namespace + "$api-gateway"
	gateway := nodes[gatewayNodeID]
	if gateway == nil {
		t.Fatal("api-gateway node not found")
	}
	if gateway.IssueCount != 15 {
		t.Errorf("api-gateway IssueCount = %d, want 15 (5 internal + 10 external)", gateway.IssueCount)
	}
	// Status should be "Error" because of the 5 internal errors
	if gateway.Status != "Error" {
		t.Errorf("api-gateway Status = %s, want Error (Internal 에러 5건으로 인해)", gateway.Status)
	}

	// Verify auth-service node
	authNodeID := cluster + "$" + namespace + "$auth-service"
	auth := nodes[authNodeID]
	if auth == nil {
		t.Fatal("auth-service node not found")
	}
	if auth.IssueCount != 5 {
		t.Errorf("auth-service IssueCount = %d, want 5", auth.IssueCount)
	}
	if auth.Status != "Error" {
		t.Errorf("auth-service Status = %s, want Error", auth.Status)
	}

	// Verify redis.external.com node
	redisNodeID := cluster + "$" + namespace + "$redis.external.com"
	redis := nodes[redisNodeID]
	if redis == nil {
		t.Fatal("redis.external.com node not found")
	}
	if redis.IssueCount != 10 {
		t.Errorf("redis.external.com IssueCount = %d, want 10", redis.IssueCount)
	}
	if redis.Status != "Error" {
		t.Errorf("redis.external.com Status = %s, want Error", redis.Status)
	}
}

// Helper function
func uint8Ptr(v uint8) *uint8 {
	return &v
}
