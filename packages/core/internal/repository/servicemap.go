package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/domain/servicemap"
	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/infrastructure"
	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/repository/queries"
	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/utils"
)

type ServiceMapRepository interface {
	GetTopology(ctx context.Context, req *servicemap.TopologyRequest) (*servicemap.TopologyResponse, error)
	GetWorkloadDetails(ctx context.Context, req *servicemap.WorkloadDetailRequest) (*servicemap.WorkloadDetailResponse, error)
	GetWorkloadHover(ctx context.Context, req *servicemap.WorkloadHoverRequest) (*servicemap.WorkloadHoverResponse, error)
	// edge detail
	ExecuteEdgeQuery(ctx context.Context, queryType string, parsedParam servicemap.ParsedParam, startTimeFormatted, endTimeFormatted string) (*servicemap.EdgeTraceDetailResponse, error)
}

type ServiceMapRepositoryImpl struct{}

var serviceMapRepositoryInstance ServiceMapRepository = nil

func ServiceMapRepositoryInstance() ServiceMapRepository {
	if serviceMapRepositoryInstance == nil {
		serviceMapRepositoryInstance = &ServiceMapRepositoryImpl{}
	}
	return serviceMapRepositoryInstance
}

// GetTopology 메인 서비스맵 토폴로지 조회 (ClickHouse Native API 최적화)
func (r *ServiceMapRepositoryImpl) GetTopology(ctx context.Context, req *servicemap.TopologyRequest) (*servicemap.TopologyResponse, error) {
	// Context timeout 확인 (조기 탐지)
	if ctx.Err() != nil {
		log.Printf("[ERROR] Request context canceled: %v", ctx.Err())
		return nil, fmt.Errorf("request context canceled before query execution: %w", ctx.Err())
	}

	// 충분한 타임아웃으로 새로운 context 생성 (Context Canceled 방지)
	queryCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	// timeMinutes := calculateDurationMinutes(req.StartTime, req.EndTime)
	// log.Printf("GetTopology started: startTime=%s, endTime=%s, minutes=%d", req.StartTime, req.EndTime, timeMinutes)

	// ClickHouse 쿼리용 시간 형식 변환
	startTimeFormatted := utils.FormatTimeForClickHouse(req.StartTime)
	endTimeFormatted := utils.FormatTimeForClickHouse(req.EndTime)
	// log.Printf("ClickHouse formatted times: %s → %s", startTimeFormatted, endTimeFormatted)

	// 쿼리 생성
	customQuery := buildQueryNetMapConn(req)

	// 쿼리 파라미터 준비
	customQueryParams := buildQueryParamsNetMapConn(startTimeFormatted, endTimeFormatted, req.Cluster, req.Namespace, req.Workload, req.Protocol)

	// 쿼리 디버깅: 실제 실행되는 쿼리 로깅
	// log.Printf("🔍 DEBUG - Query: %s", customQuery)
	// log.Printf("🔍 DEBUG - Query Params: %+v", customQueryParams)

	// Native API 쿼리 실행 - CustomQuery
	customRows, err := infrastructure.QueryClickHouseWithContext(queryCtx, customQuery, customQueryParams...)
	if err != nil {
		log.Printf("[ERROR] Custom Query Failed: %v", err)
		return nil, fmt.Errorf("failed to query custom metrics: %w", err)
	}
	defer func() {
		if closeErr := customRows.Close(); closeErr != nil {
			log.Printf("[WARN] Failed to close ClickHouse rows: %v", closeErr)
		}
	}()

	// 2. 노드 맵으로 중복 제거
	nodes := make(map[string]*servicemap.ServiceMapNode)
	edges := []servicemap.ServiceMapEdge{}

	nodeCount := 0
	for customRows.Next() {
		// 새로운 쿼리 구조에 맞는 변수 선언
		var cluster, srcWorkload, destWorkload, srcNamespace, destNamespace string
		var srcRaw, destRaw string // ✅ 원본 값 (trace 매칭용)
		var isExternal *uint8
		var protocol string
		var totalErrors uint64

		// 스캔
		if err := customRows.Scan(&cluster, &srcWorkload, &destWorkload, &srcNamespace, &destNamespace,
			&isExternal, &protocol, &srcRaw, &destRaw, &totalErrors); err != nil {
			return nil, fmt.Errorf("failed to scan node row %d: %w", nodeCount+1, err)
		}

		// 노드 타입 결정
		nodeType := "workload"
		if isExternal != nil && *isExternal == 1 {
			nodeType = "external-service"
		}

		// 4. Source 노드 처리 (다중 Edge 에러 누적)
		// - Internal 타겟: Source 노드도 에러 상태 반영
		// - External 타겟: IssueCount만 누적, Status는 "Ok" 유지
		srcNodeID := fmt.Sprintf("%s$%s$%s", cluster, srcNamespace, srcWorkload)
		if existingNode, exists := nodes[srcNodeID]; exists {
			// 기존 노드: 에러 누적
			existingNode.IssueCount += int(totalErrors)

			// Internal → External 케이스: Source는 Ok 유지
			// Internal → Internal 케이스: Source도 Error 반영
			if isExternal == nil || *isExternal == 0 {
				// Internal target: Source도 에러 상태 업데이트
				existingNode.Status = determineNodeStatus(uint64(existingNode.IssueCount))
			}
			// External target: IssueCount만 업데이트, Status는 "Ok" 유지
		} else {
			// 새 노드 생성
			zero := uint8(0)
			status := "Ok"
			if isExternal == nil || *isExternal == 0 {
				// Internal target: 에러 반영
				status = determineNodeStatus(totalErrors)
			}

			nodes[srcNodeID] = &servicemap.ServiceMapNode{
				ID:           generateNodeID(true, cluster, srcNamespace, srcWorkload, isExternal),
				WorkloadName: srcWorkload,
				Namespace:    srcNamespace,
				Cluster:      cluster,
				IssueCount:   int(totalErrors),
				IsExternal:   &zero,
				Type:         nodeType,
				Status:       status,
			}
			nodeCount++
		}

		// 5. Destination 노드 처리 (다중 Edge 에러 누적)
		// Destination은 항상 에러 반영 (Internal/External 구분 없음)
		destNodeID := fmt.Sprintf("%s$%s$%s", cluster, destNamespace, destWorkload)
		if existingNode, exists := nodes[destNodeID]; exists {
			// 기존 노드: 에러 누적 및 상태 업데이트
			existingNode.IssueCount += int(totalErrors)
			existingNode.Status = determineNodeStatus(uint64(existingNode.IssueCount))
		} else {
			// 새 노드 생성
			nodes[destNodeID] = &servicemap.ServiceMapNode{
				ID:           generateNodeID(false, cluster, destNamespace, destWorkload, isExternal),
				WorkloadName: destWorkload,
				Namespace:    destNamespace,
				Cluster:      cluster,
				IssueCount:   int(totalErrors),
				IsExternal:   isExternal,
				Type:         nodeType,
				Status:       determineNodeStatus(totalErrors),
			}
			nodeCount++
		}

		// ServiceMapEdge 생성
		edge := servicemap.ServiceMapEdge{
			ID:          srcNamespace + "$$" + srcWorkload + "##" + destNamespace + "$$" + destWorkload + "##" + protocol,
			Source:      cluster + "$$" + srcNamespace + "$$" + srcWorkload,
			Destination: determineTargetID(destWorkload, cluster, destNamespace, isExternal),
			SrcRaw:      srcRaw,  // ✅ 원본 src 값 (trace 매칭용)
			DestRaw:     destRaw, // ✅ 원본 dest 값 (trace 매칭용)
			Protocol:    protocol,
			IsError:     totalErrors > 0,
			IsExternal:  isExternal,
		}
		edges = append(edges, edge)
	}

	// 노드 쿼리 에러 확인
	if err := customRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating node rows: %w", err)
	}
	// log.Printf("Successfully parsed %d nodes from ClickHouse", nodeCount)

	// 7. 노드 맵을 배열로 변환
	nodesArray := make([]servicemap.ServiceMapNode, 0, len(nodes))
	for _, node := range nodes {
		nodesArray = append(nodesArray, *node)
	}

	// 실제 데이터 기반 응답 생성
	response := &servicemap.TopologyResponse{
		Nodes:     nodesArray,
		Edges:     edges,
		TimeRange: formatTimeRange(req.StartTime, req.EndTime),
	}

	// log.Printf("GetTopology completed with real data: %d nodes, %d edges", len(nodes), len(edges))
	return response, nil
}

// GetWorkloadDetails 워크로드 상세 정보 조회
func (r *ServiceMapRepositoryImpl) GetWorkloadDetails(ctx context.Context, req *servicemap.WorkloadDetailRequest) (*servicemap.WorkloadDetailResponse, error) {

	// Context timeout 확인 (조기 탐지)
	if ctx.Err() != nil {
		log.Printf("[ERROR] Request context canceled: %v", ctx.Err())
		return nil, fmt.Errorf("request context canceled before query execution: %w", ctx.Err())
	}

	// 충분한 타임아웃으로 새로운 context 생성 (Context Canceled 방지)
	queryCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	// ClickHouse 쿼리용 시간 형식 변환
	startTimeFormatted := utils.FormatTimeForClickHouse(req.StartTime)
	endTimeFormatted := utils.FormatTimeForClickHouse(req.EndTime)

	// ========================================
	// 단계 1: Kind 쿼리 (순차 실행 필수 - runningPods 쿼리에서 workloadType 필요)
	// ========================================
	kindQueryStart := time.Now()
	kindQuery := buildQueryWorkloadKind()
	kindParams := buildQueryParamsWorkloadKind(startTimeFormatted, endTimeFormatted, req.Cluster, req.Namespace, req.WorkloadName)

	rows, err := infrastructure.QueryClickHouseWithContext(queryCtx, kindQuery, kindParams...)
	if err != nil {
		log.Printf("[ERROR] Kind query failed: %v", err)
		return nil, fmt.Errorf("failed to query workload kind: %w", err)
	}
	defer rows.Close()

	var workloadType string
	if rows.Next() {
		if err := rows.Scan(&workloadType); err != nil {
			log.Printf("[ERROR] Kind scan failed: %v", err)
			return nil, fmt.Errorf("failed to scan workload kind: %w", err)
		}
	}
	log.Printf("[PERF] Kind query took: %v (result: %s)", time.Since(kindQueryStart), workloadType)

	// ========================================
	// 단계 2 & 3: RunningPods + 통합 메트릭 쿼리
	// Rollout은 두 쿼리를 병렬 실행하여 성능 개선 (2초 → 1초)
	// ========================================
	var metrics []string
	switch workloadType {
	case "Deployment":
		metrics = append(metrics, "kube_deployment_spec_replicas", "kube_deployment_status_replicas_available")
	case "StatefulSet":
		metrics = append(metrics, "kube_statefulset_replicas", "kube_statefulset_status_replicas")
	case "DaemonSet":
		metrics = append(metrics, "kube_daemonset_status_desired_number_scheduled", "kube_daemonset_status_current_number_scheduled")
	case "Rollout":
		metrics = append(metrics, "rollout_info")
	}

	var desiredPods int
	var availablePods int
	var (
		workloadCpuMetricList          []servicemap.WorkloadMetric
		workloadMemoryMetricList       []servicemap.WorkloadMetric
		workloadNetworkIoMetricList    []servicemap.WorkloadMetric
		workloadNetworkErrorMetricList []servicemap.WorkloadMetric
	)

	// 메트릭별 데이터 분류를 위한 맵
	cpuMetricsMap := make(map[string]*servicemap.WorkloadMetric)
	memoryMetricsMap := make(map[string]*servicemap.WorkloadMetric)
	networkIoMetricsMap := make(map[string]*servicemap.WorkloadMetric)
	networkErrorMetricsMap := make(map[string]*servicemap.WorkloadMetric)

	if workloadType == "Rollout" {
		// ========================================
		// Rollout: 두 쿼리를 병렬로 실행하여 성능 최적화
		// ========================================
		var wg sync.WaitGroup
		var runningPodsErr, metricsErr error
		rolloutStartTime := time.Now()

		// 병렬 실행 1: RunningPods 쿼리
		wg.Add(1)
		go func() {
			defer wg.Done()
			queryStart := time.Now()
			rolloutPodsQuery := buildQueryWorkloadRunningPodsForRollout()
			rolloutPodsParams := buildQueryParamsWorkloadRunningPodsForRollout(req.Cluster, req.Namespace, req.WorkloadName)

			runningPodsRows, err := infrastructure.QueryClickHouseWithContext(queryCtx, rolloutPodsQuery, rolloutPodsParams...)
			if err != nil {
				log.Printf("[ERROR] Rollout RunningPods query failed: %v", err)
				runningPodsErr = fmt.Errorf("failed to query rollout running pods: %w", err)
				return
			}
			defer runningPodsRows.Close()

			for runningPodsRows.Next() {
				var workloadName, namespace string
				var runningPods uint64
				if err := runningPodsRows.Scan(&workloadName, &namespace, &runningPods); err != nil {
					log.Printf("[ERROR] Rollout RunningPods scan failed: %v", err)
					runningPodsErr = fmt.Errorf("failed to scan rollout running pods: %w", err)
					return
				}
				desiredPods = int(runningPods)
				availablePods = int(runningPods)
			}
			log.Printf("[PERF] Rollout RunningPods query took: %v", time.Since(queryStart))
		}()

		// 병렬 실행 2: AllMetrics 쿼리
		wg.Add(1)
		go func() {
			defer wg.Done()
			queryStart := time.Now()
			unifiedQuery := buildQueryWorkloadAllMetrics(workloadType)
			unifiedParams := buildQueryWorkloadAllMetricsParamsForRollout(startTimeFormatted, endTimeFormatted, req.Cluster, req.Namespace, req.WorkloadName)

			metricsRows, err := infrastructure.QueryClickHouseWithContext(queryCtx, unifiedQuery, unifiedParams...)
			if err != nil {
				log.Printf("[ERROR] Rollout metrics query failed: %v", err)
				metricsErr = fmt.Errorf("rollout metrics query failed: %w", err)
				return
			}
			log.Printf("[PERF] Rollout AllMetrics query execution took: %v", time.Since(queryStart))
			defer metricsRows.Close()

			// 쿼리 결과 파싱
			parseStart := time.Now()
			rowCount := 0
			for metricsRows.Next() {
				var timestamp time.Time
				var workloadName, namespace string
				var direction, interfaceName sql.NullString
				var metricName string
				var metricValue float64

				if err := metricsRows.Scan(&timestamp, &workloadName, &namespace, &direction, &interfaceName, &metricName, &metricValue); err != nil {
					log.Printf("[ERROR] Rollout metrics scan failed: %v", err)
					metricsErr = fmt.Errorf("failed to scan rollout metrics: %w", err)
					return
				}
				rowCount++

				timestampStr := timestamp.Format("2006-01-02 15:04:05")
				parseMetricResult(metricName, timestampStr, metricValue, direction, interfaceName,
					cpuMetricsMap, memoryMetricsMap, networkIoMetricsMap, networkErrorMetricsMap)
			}
			log.Printf("[PERF] Rollout AllMetrics parsing took: %v (rows: %d)", time.Since(parseStart), rowCount)
		}()

		// 두 쿼리 완료 대기
		wg.Wait()
		log.Printf("[PERF] Rollout total parallel execution took: %v", time.Since(rolloutStartTime))

		// 에러 확인
		if runningPodsErr != nil {
			return nil, runningPodsErr
		}
		if metricsErr != nil {
			return nil, metricsErr
		}

		// 맵에서 리스트로 변환
		for _, v := range cpuMetricsMap {
			workloadCpuMetricList = append(workloadCpuMetricList, *v)
		}
		for _, v := range memoryMetricsMap {
			workloadMemoryMetricList = append(workloadMemoryMetricList, *v)
		}
		for _, v := range networkIoMetricsMap {
			workloadNetworkIoMetricList = append(workloadNetworkIoMetricList, *v)
		}
		for _, v := range networkErrorMetricsMap {
			workloadNetworkErrorMetricList = append(workloadNetworkErrorMetricList, *v)
		}
	} else {
		// ========================================
		// 그 외 워크로드: 기존 순차 실행 유지
		// ========================================
		podStatusQuery := buildQueryWorkloadRunningPods(workloadType)
		podStatusQueryParams := buildQueryParamsWorkloadRunningPods(metrics, startTimeFormatted, endTimeFormatted, req.WorkloadName, req.Namespace)

		runningPodsRows, err := infrastructure.QueryClickHouseWithContext(queryCtx, podStatusQuery, podStatusQueryParams...)
		if err != nil {
			log.Printf("[ERROR] RunningPods query failed: %v", err)
			return nil, fmt.Errorf("failed to query running pods: %w", err)
		}
		defer runningPodsRows.Close()

		for runningPodsRows.Next() {
			var deploymentName, namespace, metricName string
			var metricValue float64
			if err := runningPodsRows.Scan(&deploymentName, &namespace, &metricName, &metricValue); err != nil {
				log.Printf("[ERROR] RunningPods scan failed: %v", err)
				return nil, fmt.Errorf("failed to scan running pods: %w", err)
			}

			if deploymentName != req.WorkloadName || namespace != req.Namespace {
				log.Printf("[ERROR] Mismatched deployment name or namespace: %s/%s", deploymentName, namespace)
				return nil, fmt.Errorf("mismatched deployment name or namespace")
			}

			switch metricName {
			case "kube_deployment_spec_replicas", "kube_statefulset_replicas", "kube_daemonset_status_desired_number_scheduled":
				desiredPods = int(metricValue)
			case "kube_deployment_status_replicas_available", "kube_statefulset_status_replicas", "kube_daemonset_status_current_number_scheduled":
				availablePods = int(metricValue)
			default:
				log.Printf("[WARN] Unknown metric name: %s", metricName)
			}
		}

		// 통합 메트릭 쿼리 실행
		unifiedQuery := buildQueryWorkloadAllMetrics(workloadType)
		unifiedParams := buildQueryWorkloadAllMetricsParams(startTimeFormatted, endTimeFormatted, req.WorkloadName, req.Namespace)

		rows, err2 := infrastructure.QueryClickHouseWithContext(queryCtx, unifiedQuery, unifiedParams...)
		if err2 != nil {
			log.Printf("[ERROR] Unified metrics query failed: %v", err2)
			return nil, fmt.Errorf("unified metrics query failed: %w", err2)
		}
		defer rows.Close()

		// 쿼리 결과 파싱
		for rows.Next() {
			var timestamp time.Time
			var workloadName, namespace string
			var direction, interfaceName sql.NullString
			var metricName string
			var metricValue float64

			if err := rows.Scan(&timestamp, &workloadName, &namespace, &direction, &interfaceName, &metricName, &metricValue); err != nil {
				log.Printf("[ERROR] Row scan failed: %v", err)
				return nil, fmt.Errorf("row scan failed: %w", err)
			}

			timestampStr := timestamp.Format("2006-01-02 15:04:05")

			// 메트릭 타입별 분류
			switch metricName {
			case "k8s.pod.cpu.usage":
				if cpuMetricsMap["cpuUsage"] == nil {
					cpuMetricsMap["cpuUsage"] = &servicemap.WorkloadMetric{QueryName: "cpuUsage", Values: []servicemap.WorkloadMetricValue{}}
				}
				cpuMetricsMap["cpuUsage"].Values = append(cpuMetricsMap["cpuUsage"].Values, servicemap.WorkloadMetricValue{
					Timestamp: timestampStr,
					Value:     metricValue,
				})

			case "k8s.container.cpu_request":
				if cpuMetricsMap["cpuRequest"] == nil {
					cpuMetricsMap["cpuRequest"] = &servicemap.WorkloadMetric{QueryName: "cpuRequest", Values: []servicemap.WorkloadMetricValue{}}
				}
				cpuMetricsMap["cpuRequest"].Values = append(cpuMetricsMap["cpuRequest"].Values, servicemap.WorkloadMetricValue{
					Timestamp: timestampStr,
					Value:     metricValue,
				})

			case "k8s.container.cpu_limit":
				if cpuMetricsMap["cpuLimit"] == nil {
					cpuMetricsMap["cpuLimit"] = &servicemap.WorkloadMetric{QueryName: "cpuLimit", Values: []servicemap.WorkloadMetricValue{}}
				}
				cpuMetricsMap["cpuLimit"].Values = append(cpuMetricsMap["cpuLimit"].Values, servicemap.WorkloadMetricValue{
					Timestamp: timestampStr,
					Value:     metricValue,
				})

			case "k8s.pod.memory.usage":
				if memoryMetricsMap["memoryUsage"] == nil {
					memoryMetricsMap["memoryUsage"] = &servicemap.WorkloadMetric{QueryName: "memoryUsage", Values: []servicemap.WorkloadMetricValue{}}
				}
				memoryMetricsMap["memoryUsage"].Values = append(memoryMetricsMap["memoryUsage"].Values, servicemap.WorkloadMetricValue{
					Timestamp: timestampStr,
					Value:     metricValue,
				})

			case "k8s.container.memory_request":
				if memoryMetricsMap["memoryRequest"] == nil {
					memoryMetricsMap["memoryRequest"] = &servicemap.WorkloadMetric{QueryName: "memoryRequest", Values: []servicemap.WorkloadMetricValue{}}
				}
				memoryMetricsMap["memoryRequest"].Values = append(memoryMetricsMap["memoryRequest"].Values, servicemap.WorkloadMetricValue{
					Timestamp: timestampStr,
					Value:     metricValue,
				})

			case "k8s.container.memory_limit":
				if memoryMetricsMap["memoryLimit"] == nil {
					memoryMetricsMap["memoryLimit"] = &servicemap.WorkloadMetric{QueryName: "memoryLimit", Values: []servicemap.WorkloadMetricValue{}}
				}
				memoryMetricsMap["memoryLimit"].Values = append(memoryMetricsMap["memoryLimit"].Values, servicemap.WorkloadMetricValue{
					Timestamp: timestampStr,
					Value:     metricValue,
				})

			case "k8s.pod.network.io":
				key := fmt.Sprintf("%s_%s", direction.String, interfaceName.String)
				if networkIoMetricsMap[key] == nil {
					networkIoMetricsMap[key] = &servicemap.WorkloadMetric{
						QueryName: "networkIo",
						Labels: map[string]string{
							"direction": direction.String,
							"interface": interfaceName.String,
						},
						Values: []servicemap.WorkloadMetricValue{},
					}
				}
				networkIoMetricsMap[key].Values = append(networkIoMetricsMap[key].Values, servicemap.WorkloadMetricValue{
					Timestamp: timestampStr,
					Value:     metricValue,
				})

			case "k8s.pod.network.errors":
				key := fmt.Sprintf("%s_%s", direction.String, interfaceName.String)
				if networkErrorMetricsMap[key] == nil {
					networkErrorMetricsMap[key] = &servicemap.WorkloadMetric{
						QueryName: "networkError",
						Labels: map[string]string{
							"direction": direction.String,
							"interface": interfaceName.String,
						},
						Values: []servicemap.WorkloadMetricValue{},
					}
				}
				networkErrorMetricsMap[key].Values = append(networkErrorMetricsMap[key].Values, servicemap.WorkloadMetricValue{
					Timestamp: timestampStr,
					Value:     metricValue,
				})
			}
		}

		// 맵을 슬라이스로 변환
		for _, metric := range cpuMetricsMap {
			workloadCpuMetricList = append(workloadCpuMetricList, *metric)
		}
		for _, metric := range memoryMetricsMap {
			workloadMemoryMetricList = append(workloadMemoryMetricList, *metric)
		}
		for _, metric := range networkIoMetricsMap {
			workloadNetworkIoMetricList = append(workloadNetworkIoMetricList, *metric)
		}
		for _, metric := range networkErrorMetricsMap {
			workloadNetworkErrorMetricList = append(workloadNetworkErrorMetricList, *metric)
		}
	}

	// ========================================
	// 단계 4: 최종 응답 조립
	// ========================================
	result := servicemap.WorkloadDetailResponse{
		WorkloadName:           req.WorkloadName,
		Cluster:                req.Cluster,
		Kind:                   workloadType,
		Namespace:              req.Namespace,
		Replicas:               desiredPods,
		RunningPods:            availablePods,
		CpuMetricList:          workloadCpuMetricList,
		MemoryMetricList:       workloadMemoryMetricList,
		NetworkIoMetricList:    workloadNetworkIoMetricList,
		NetworkErrorMetricList: workloadNetworkErrorMetricList,
	}

	return &result, nil
}

// GetWorkloadHover 워크로드 호버 정보 조회
func (r *ServiceMapRepositoryImpl) GetWorkloadHover(ctx context.Context, req *servicemap.WorkloadHoverRequest) (*servicemap.WorkloadHoverResponse, error) {
	// Context timeout 확인 (조기 탐지)
	if ctx.Err() != nil {
		log.Printf("[ERROR] Request context already canceled: %v", ctx.Err())
		return nil, fmt.Errorf("request context canceled before query execution: %w", ctx.Err())
	}

	// 충분한 타임아웃으로 새로운 context 생성 (Context Canceled 방지)
	queryCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	// ClickHouse 쿼리용 시간 형식 변환
	startTimeFormatted := utils.FormatTimeForClickHouse(req.StartTime)
	endTimeFormatted := utils.FormatTimeForClickHouse(req.EndTime)
	// log.Printf("ClickHouse formatted times: %s → %s", startTimeFormatted, endTimeFormatted)

	// ======= 1. Golden Signal Metrics
	goldenSignalQuery := buildQueryGoldenSignal(req)
	goldenSignalParams := buildQueryParamsGoldenSignal(startTimeFormatted, endTimeFormatted, req.Cluster, req.Namespace, req.WorkloadName)

	// 쿼리 디버깅: 실제 실행되는 쿼리 로깅
	// log.Printf("🔍 DEBUG - Query: %s", goldenSignalQuery)
	// log.Printf("🔍 DEBUG - Query Params: %+v", goldenSignalParams)

	// Native API 쿼리 실행 - CustomQuery
	gsRows, err := infrastructure.QueryClickHouseWithContext(queryCtx, goldenSignalQuery, goldenSignalParams...)
	if err != nil {
		log.Printf("[ERROR] Custom Query Failed: %v", err)
		return nil, fmt.Errorf("failed to query custom metrics: %w", err)
	}
	defer func() {
		if closeErr := gsRows.Close(); closeErr != nil {
			log.Printf("[WARN] failed to close node rows: %v", closeErr)
		}
	}()

	// 스캔
	var nodeMetrics *servicemap.NodeMetrics
	var cluster, destNamespace, destWorkload string
	var rateRps, errorRatePercent, durationAvgMs, durationP50Ms, durationP95Ms, durationP99Ms float64
	var totalRequests, totalErrors *uint64

	if gsRows.Next() {
		if err := gsRows.Scan(&cluster, &destNamespace, &destWorkload, &rateRps, &errorRatePercent, &durationAvgMs, &durationP50Ms, &durationP95Ms, &durationP99Ms, &totalRequests, &totalErrors); err != nil {
			log.Printf("[ERROR] Scan Failed: %v", err)
			return nil, fmt.Errorf("failed to scan node row: %w", err)
		}
		nodeMetrics = &servicemap.NodeMetrics{
			RequestRate:   rateRps,
			LatencyP95:    durationP95Ms,
			ErrorRate:     errorRatePercent,
			TotalRequests: int64(*totalRequests),
			TotalErrors:   int64(*totalErrors),
		}
	}

	// =======  2. TopPeers
	topPeersQuery := buildQueryTopPeers(req)
	topPeersParams := buildQueryParamsTopPeers(startTimeFormatted, endTimeFormatted, req.Cluster, req.Namespace, req.WorkloadName)

	// 쿼리 디버깅: 실제 실행되는 쿼리 로깅
	// log.Printf("🔍 DEBUG - Query: %s", topPeersQuery)
	// log.Printf("🔍 DEBUG - Query Params: %+v", topPeersParams)

	// Native API 쿼리 실행 - CustomQuery
	topPeersRows, err := infrastructure.QueryClickHouseWithContext(queryCtx, topPeersQuery, topPeersParams...)
	if err != nil {
		log.Printf("[ERROR] Custom Query Failed: %v", err)
		return nil, fmt.Errorf("failed to query custom metrics: %w", err)
	}
	defer func() {
		if closeErr := topPeersRows.Close(); closeErr != nil {
			log.Printf("[WARN] failed to close node rows: %v", closeErr)
		}
	}()

	// 스캔
	var topPeers []servicemap.PeerInfo
	var rank int = 1 // Rank 초기값 설정
	var srcNamespace, srcWorkload string

	for topPeersRows.Next() {
		if err := topPeersRows.Scan(&cluster, &destNamespace, &destWorkload, &srcNamespace, &srcWorkload,
			&rateRps, &errorRatePercent, &durationAvgMs, &durationP50Ms, &durationP95Ms, &durationP99Ms,
			&totalRequests, &totalErrors); err != nil {
			log.Printf("[ERROR] Scan Failed: %v", err)
			return nil, fmt.Errorf("failed to scan node row: %w", err)
		}
		// direction 결정
		if srcNamespace == req.Namespace && srcWorkload == req.WorkloadName {
			topPeers = append(topPeers, servicemap.PeerInfo{
				Rank:          rank, // 순차적으로 증가하는 Rank 할당
				WorkloadName:  destWorkload,
				Direction:     "outbound",
				RequestRate:   rateRps,
				LatencyP95:    durationP95Ms,
				ErrorRate:     errorRatePercent,
				TotalRequests: int64(*totalRequests),
			})
		} else {
			topPeers = append(topPeers, servicemap.PeerInfo{
				Rank:          rank, // 순차적으로 증가하는 Rank 할당
				WorkloadName:  srcWorkload,
				Direction:     "inbound",
				RequestRate:   rateRps,
				LatencyP95:    durationP95Ms,
				ErrorRate:     errorRatePercent,
				TotalRequests: int64(*totalRequests),
			})
		}

		rank++ // 다음 데이터를 위해 Rank 증가
	}

	// =======  데이터 반환
	response := &servicemap.WorkloadHoverResponse{
		WorkloadName: req.WorkloadName,
		Cluster:      req.Cluster,
		Namespace:    req.Namespace,
		NodeMetrics:  nodeMetrics,
		TopPeers:     topPeers,
	}

	// log.Printf("GetWorkloadHover completed for workload: %s", req.WorkloadName)
	return response, nil
}

// GetEdgeTraceDetails 엣지 디테일 정보 조회
// ExecuteEdgeQuery - Unified edge query execution (delegates to specific methods based on queryType)
func (r *ServiceMapRepositoryImpl) ExecuteEdgeQuery(
	ctx context.Context,
	queryType string,
	parsedParam servicemap.ParsedParam,
	startTimeFormatted, endTimeFormatted string,
) (*servicemap.EdgeTraceDetailResponse, error) {
	// Use unified GetEdgeDetails method
	return r.GetEdgeDetails(ctx, queryType, parsedParam, startTimeFormatted, endTimeFormatted)
}

// =================================== Helper functions ========================================

// formatTimeRange StartTime/EndTime으로 시간 범위 문자열 생성
func formatTimeRange(startTime, endTime string) string {
	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return "30m" // 기본값
	}
	end, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		return "30m" // 기본값
	}

	duration := end.Sub(start)
	minutes := int(duration.Minutes())

	if minutes <= 0 {
		return "1m"
	} else if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	} else if minutes < 1440 {
		hours := minutes / 60
		return fmt.Sprintf("%dh", hours)
	} else {
		days := minutes / 1440
		return fmt.Sprintf("%dd", days)
	}
}

// generateNodeID 노드 ID 생성 (외부/내부 서비스 통일된 규칙)
func generateNodeID(isSrc bool, cluster, namespace, workloadName string, isExternal *uint8) string {
	if isSrc == true && isExternal != nil {
		return cluster + "$$" + namespace + "$$" + workloadName
	}

	if isExternal != nil && *isExternal == 1 {
		return "external$$external$$" + workloadName
	}
	return cluster + "$$" + namespace + "$$" + workloadName
}

// determineTargetID 타겟 서비스의 ID를 결정 (외부/내부 서비스 구분)
func determineTargetID(dest, cluster, namespace string, isExternal *uint8) string {
	if isExternal != nil && *isExternal == 1 {
		// 외부 서비스는 external$$external$$서비스명 형태
		return "external$$external$$" + dest
	}
	// 내부 서비스는 클러스터$$네임스페이스$$서비스명 형태
	return cluster + "$$" + namespace + "$$" + dest
}

// determineNodeStatus 에러 개수 기반 노드 상태 결정 (단순화)
func determineNodeStatus(totalErrors uint64) string {
	if totalErrors > 0 {
		return "Error"
	}
	return "Ok"
}

// buildStatusFilterForCombinedNodes 서브쿼리 결과용 Status 필터링
func buildStatusFilterForCombinedNodes(statusList []string) string {
	// Status 필터가 없거나 빈 경우
	if len(statusList) == 0 {
		return ""
	}
	// ["Ok", "Error"] 둘 다 포함인 경우 (전체 조회)
	if containsAll(statusList, []string{"Ok", "Error"}) {
		return ""
	}
	// "Ok"만 포함인 경우 (에러가 없는 연결만)
	if containsOnly(statusList, "Ok") {
		return "\n  HAVING total_errors = 0"
	}
	// "Error"만 포함인 경우 (에러가 있는 연결만)
	if containsOnly(statusList, "Error") {
		return "\n  HAVING total_errors > 0"
	}
	// 기본값: 필터링 안함
	return ""
}

// containsOnly 슬라이스가 특정 값 하나만 포함하는지 확인
func containsOnly(slice []string, target string) bool {
	return len(slice) == 1 && slice[0] == target
}

// containsAll 슬라이스가 모든 대상 값을 포함하는지 확인
func containsAll(slice []string, targets []string) bool {
	if len(slice) != len(targets) {
		return false
	}

	targetMap := make(map[string]bool)
	for _, target := range targets {
		targetMap[target] = true
	}

	for _, item := range slice {
		if !targetMap[item] {
			return false
		}
	}

	return len(slice) == len(targets)
}

// flash
func buildQueryNetMapConn(req *servicemap.TopologyRequest) string {
	query := `
	SELECT
	    COALESCE(NULLIF(k8s_cluster_name, ''), 'unknown') as cluster,
		src as src_workload,
		dest as dest_workload,
		COALESCE(NULLIF(src_namespace, ''), 'unknown') as src_namespace,
		COALESCE(NULLIF(dest_namespace, ''), 'unknown') as dest_namespace,
		is_external as is_external,
		protocol,
		COALESCE(any(src_raw), '') as src_raw,
		COALESCE(any(dest_raw), '') as dest_raw,
		--sum(total_count) as total_requests,
		sum(error_count) as total_errors
		--round(avgIf(duration_p95, duration_p95 > 0), 2) as p95_latency_ms,
		--round(total_errors * 100.0 / nullif(total_requests, 0), 2) as error_rate_percent
	FROM signoz_traces.network_map_connections
	WHERE timestamp >= toDateTime(?)
		AND timestamp <= toDateTime(?)`

	// 필터 조건 추가
	if len(req.Cluster) > 0 {
		query += "\n  AND k8s_cluster_name IN (" + buildInClause(len(req.Cluster)) + ")"
	}
	if len(req.Workload) > 0 {
		query += "\n  AND (src IN (" + buildInClause(len(req.Workload)) + ") OR dest IN (" + buildInClause(len(req.Workload)) + "))"
	}
	if len(req.Namespace) > 0 {
		query += "\n  AND (src_namespace IN (" + buildInClause(len(req.Namespace)) + ") OR dest_namespace IN (" + buildInClause(len(req.Namespace)) + "))"
	}
	if len(req.Protocol) > 0 {
		query += "\n  AND protocol IN (" + buildInClause(len(req.Protocol)) + ")"
	}

	query += `
	GROUP BY src_workload, dest_workload, protocol, cluster, src_namespace, dest_namespace, is_external
	`
	// Status 필터링을 외부 WHERE절에 추가
	query += buildStatusFilterForCombinedNodes(req.Status)

	// query += `
	// ORDER BY total_requests DESC`

	return query
}

// flash
func buildQueryParamsNetMapConn(startTime, endTime string, cluster, namespace, workload, protocol []string) []interface{} {
	var params []interface{}

	// 기본 시간 파라미터
	params = append(params, startTime, endTime)

	if len(cluster) > 0 {
		for _, cluster := range cluster {
			params = append(params, cluster)
		}
	}
	for _, workload := range workload {
		params = append(params, workload) // src
	}
	for _, workload := range workload {
		params = append(params, workload) // dest
	}
	if len(namespace) > 0 {
		for _, namespace := range namespace {
			params = append(params, namespace)
			params = append(params, namespace)
		}
	}

	if len(protocol) > 0 {
		for _, protocol := range protocol {
			params = append(params, protocol)
		}
	}
	return params
}

// buildInClause IN 절용 플레이스홀더 생성
func buildInClause(count int) string {
	if count == 0 {
		return ""
	}

	placeholders := make([]string, count)
	for i := 0; i < count; i++ {
		placeholders[i] = "?"
	}
	return strings.Join(placeholders, ",")
}

// flash
// [2026-01-27] Rollout 타입 추가 - Argo Rollouts 서비스맵 지원
// [2026-01-27] KSM 기반 쿼리로 변경 - Beyla가 Rollout을 Deployment로 잘못 인식하는 문제 해결
// [2026-01-28] 성능 최적화 - time_series_v4에서 fingerprint 먼저 확보 후 samples_v4 필터링
// Dictionary(pod_workload_map_dict)와 동일한 로직 사용
func buildQueryWorkloadKind() string {
	query := `
		WITH pod_info_fingerprints AS (
			SELECT
				fingerprint,
				JSONExtractString(labels, 'k8s.cluster.name') AS cluster_name,
				JSONExtractString(labels, 'namespace') AS namespace,
				JSONExtractString(labels, 'created_by_name') AS created_by_name,
				JSONExtractString(labels, 'created_by_kind') AS created_by_kind
			FROM signoz_metrics.time_series_v4
			WHERE metric_name = 'kube_pod_info'
				AND JSONExtractString(labels, 'k8s.cluster.name') = ?
				AND JSONExtractString(labels, 'namespace') = ?
		),
		active_pod_info AS (
			SELECT DISTINCT
				pf.cluster_name,
				pf.namespace,
				pf.created_by_name,
				pf.created_by_kind
			FROM signoz_metrics.samples_v4 s
			INNER JOIN pod_info_fingerprints pf ON s.fingerprint = pf.fingerprint
			WHERE s.metric_name = 'kube_pod_info'
				AND s.value = 1
				AND toDateTime64(s.unix_milli / 1000, 9) >= now() - INTERVAL 10 MINUTE
		)
		SELECT
			if(
				pod_info.created_by_kind = 'ReplicaSet' AND rd.workload_type != '',
				rd.workload_type,
				pod_info.created_by_kind
			) AS workload_type
		FROM active_pod_info AS pod_info
		LEFT JOIN signoz_traces.replicaset_deployment_map AS rd FINAL
			ON rd.cluster_name = pod_info.cluster_name
			AND rd.namespace = pod_info.namespace
			AND rd.replicaset_name = pod_info.created_by_name
		WHERE
			-- ReplicaSet인 경우: deployment_name(=워크로드명) 매칭
			(pod_info.created_by_kind = 'ReplicaSet' AND rd.deployment_name = ?)
			-- ReplicaSet이 아닌 경우: created_by_name 직접 매칭 (DaemonSet, StatefulSet 등)
			OR (pod_info.created_by_kind != 'ReplicaSet' AND pod_info.created_by_name = ?)
		LIMIT 1
	`
	return query
}

// flash
// [2026-01-27] KSM 기반 쿼리에 맞게 파라미터 수정
func buildQueryParamsWorkloadKind(startTime, endTime, cluster, namespace, workload string) []interface{} {
	var params []interface{}

	// KSM 쿼리 파라미터: cluster, namespace, workload(ReplicaSet용), workload(직접 매칭용)
	params = append(params, cluster)
	params = append(params, namespace)
	params = append(params, workload) // ReplicaSet → deployment_name 매칭
	params = append(params, workload) // 직접 워크로드 이름 매칭 (DaemonSet, StatefulSet 등)
	return params
}

func buildQueryWorkloadRunningPods(workloadType string) string {
	workloadKey := getWorkloadKey(workloadType) // "deployment", "statefulset", "daemonset"

	query := fmt.Sprintf(`
          SELECT attrs['%s'] as workload_name,
                 attrs['namespace'] as namespace,
                 metric_name,
                 max(value) as metric_value
           FROM signoz_metrics.samples_v4 s
           JOIN signoz_metrics.time_series_v4 ts ON s.fingerprint = ts.fingerprint
          WHERE metric_name IN (?)
            AND fromUnixTimestamp64Milli(unix_milli) >= ?
            AND fromUnixTimestamp64Milli(unix_milli) <= ?
            AND attrs['%s'] = ?
            AND attrs['namespace'] = ?
          GROUP BY workload_name, namespace, metric_name
      `, workloadKey, workloadKey)

	return query
}

func getWorkloadKey(workloadType string) string {
	switch workloadType {
	case "Deployment":
		return "deployment"
	case "StatefulSet":
		return "statefulset"
	case "DaemonSet":
		return "daemonset"
	case "Rollout":
		return "rollout"
	default:
		return "deployment"
	}
}

func buildQueryParamsWorkloadRunningPods(metrics []string, startTime, endTime, deployment, namespace string) []interface{} {
	var params []interface{}

	params = append(params, metrics)
	params = append(params, startTime)
	params = append(params, endTime)
	params = append(params, deployment)
	params = append(params, namespace)
	return params
}

// Rollout용 Running Pods 쿼리 (kube_pod_info + replicaset_deployment_map join 방식)
// Argo Rollouts는 rollout_info 메트릭이 없으므로 KSM의 kube_pod_info를 사용하여 Pod 수 계산
// 최적화: time_series_v4에서 fingerprint 먼저 확보 → samples_v4는 fingerprint + 시간으로만 필터링
func buildQueryWorkloadRunningPodsForRollout() string {
	query := `
		WITH rollout_replicasets AS (
			SELECT replicaset_name
			FROM signoz_traces.replicaset_deployment_map FINAL
			WHERE cluster_name = ?
				AND namespace = ?
				AND deployment_name = ?
				AND workload_type = 'Rollout'
		),
		target_fingerprints AS (
			SELECT fingerprint, JSONExtractString(labels, 'pod') AS pod_name
			FROM signoz_metrics.time_series_v4
			WHERE metric_name = 'kube_pod_info'
				AND JSONExtractString(labels, 'k8s.cluster.name') = ?
				AND JSONExtractString(labels, 'namespace') = ?
				AND JSONExtractString(labels, 'created_by_name') IN (SELECT replicaset_name FROM rollout_replicasets)
		)
		SELECT
			? AS workload_name,
			? AS namespace,
			count(DISTINCT tf.pod_name) AS running_pods
		FROM signoz_metrics.samples_v4 s
		INNER JOIN target_fingerprints tf ON s.fingerprint = tf.fingerprint
		WHERE s.metric_name = 'kube_pod_info'
			AND s.value = 1
			AND toDateTime64(s.unix_milli / 1000, 9) >= now() - INTERVAL 10 MINUTE
	`
	return query
}

func buildQueryParamsWorkloadRunningPodsForRollout(clusterName, namespace, workloadName string) []interface{} {
	var params []interface{}
	// CTE: rollout_replicasets 파라미터
	params = append(params, clusterName)  // cluster_name
	params = append(params, namespace)    // namespace
	params = append(params, workloadName) // deployment_name
	// CTE: target_fingerprints 파라미터
	params = append(params, clusterName) // k8s.cluster.name
	params = append(params, namespace)   // namespace
	// SELECT 파라미터
	params = append(params, workloadName) // workload_name (AS)
	params = append(params, namespace)    // namespace (AS)
	return params
}

// 3-1. CPU (서브쿼리 방식 - InfraMonitoring과 동일)
func buildQueryWorkloadCPUUsage(workloadType string) string {
	workloadKey := getWorkloadKey(workloadType) // "deployment", "statefulset", "daemonset"
	labelWorkloadType := "k8s." + workloadKey + ".name"

	query := fmt.Sprintf(`
		WITH filtered_fingerprints AS (
			SELECT DISTINCT fingerprint,
				JSONExtractString(labels, '%s') as workload_name,
				JSONExtractString(labels, 'k8s.namespace.name') as namespace
			FROM signoz_metrics.time_series_v4
			WHERE JSONExtractString(labels, '%s') = ?
				AND JSONExtractString(labels, 'k8s.namespace.name') = ?
		)
		SELECT
			toStartOfMinute(fromUnixTimestamp64Milli(s.unix_milli)) as timestamp,
			f.workload_name,
			f.namespace,
			s.metric_name,
			CASE
				WHEN s.metric_name = 'k8s.pod.cpu.usage' THEN avg(s.value)
				ELSE any(s.value)
			END as metric_value
		FROM signoz_metrics.samples_v4 s
		INNER JOIN filtered_fingerprints f ON s.fingerprint = f.fingerprint
		WHERE s.metric_name IN ('k8s.pod.cpu.usage', 'k8s.container.cpu_request', 'k8s.container.cpu_limit')
			AND fromUnixTimestamp64Milli(s.unix_milli) >= toDateTime(?)
			AND fromUnixTimestamp64Milli(s.unix_milli) <= toDateTime(?)
		GROUP BY
			timestamp,
			f.workload_name,
			f.namespace,
			s.metric_name
		ORDER BY timestamp ASC, s.metric_name
	`, labelWorkloadType, labelWorkloadType)

	return query
}

// 3-1. CPU
func buildQueryWorkloadCPUUsageParams(startTime, endTime, deployment, namespace string) []interface{} {
	var params []interface{}

	// CTE 파라미터 (deployment, namespace)
	params = append(params, deployment)
	params = append(params, namespace)
	// WHERE 파라미터 (startTime, endTime)
	params = append(params, startTime)
	params = append(params, endTime)
	return params
}

// 3-2. Memory
func buildQueryWorkloadMemoryUsage(workloadType string) string {
	workloadKey := getWorkloadKey(workloadType) // "deployment", "statefulset", "daemonset"
	labelWorkloadType := "k8s." + workloadKey + ".name"

	query := fmt.Sprintf(`
		WITH filtered_fingerprints AS (
			SELECT DISTINCT fingerprint,
				JSONExtractString(labels, '%s') as workload_name,
				JSONExtractString(labels, 'k8s.namespace.name') as namespace
			FROM signoz_metrics.time_series_v4
			WHERE JSONExtractString(labels, '%s') = ?
				AND JSONExtractString(labels, 'k8s.namespace.name') = ?
		)
		SELECT
			toStartOfMinute(fromUnixTimestamp64Milli(s.unix_milli)) as timestamp,
			f.workload_name,
			f.namespace,
			s.metric_name,
			CASE
				WHEN s.metric_name = 'k8s.pod.memory.usage' THEN avg(s.value)
				ELSE any(s.value)
			END as metric_value
		FROM signoz_metrics.samples_v4 s
		INNER JOIN filtered_fingerprints f ON s.fingerprint = f.fingerprint
		WHERE s.metric_name IN ('k8s.pod.memory.usage', 'k8s.container.memory_request', 'k8s.container.memory_limit')
			AND fromUnixTimestamp64Milli(s.unix_milli) >= toDateTime(?)
			AND fromUnixTimestamp64Milli(s.unix_milli) <= toDateTime(?)
		GROUP BY
			timestamp,
			f.workload_name,
			f.namespace,
			s.metric_name
		ORDER BY timestamp ASC, s.metric_name
	`, labelWorkloadType, labelWorkloadType)

	return query
}

// 3-2. 메모리
func buildQueryWorkloadMemoryUsageParams(startTime, endTime, deployment, namespace string) []interface{} {
	var params []interface{}

	// CTE 파라미터 (deployment, namespace)
	params = append(params, deployment)
	params = append(params, namespace)
	// WHERE 파라미터 (startTime, endTime)
	params = append(params, startTime)
	params = append(params, endTime)
	return params
}

// 3-3. Network I/O
func buildQueryWorkloadNetworkIo(workloadType string) string {
	workloadKey := getWorkloadKey(workloadType) // "deployment", "statefulset", "daemonset"
	labelWorkloadType := "k8s." + workloadKey + ".name"

	query := fmt.Sprintf(`
		WITH filtered_fingerprints AS (
			SELECT DISTINCT fingerprint,
				JSONExtractString(labels, '%s') as workload_name,
				JSONExtractString(labels, 'k8s.namespace.name') as namespace,
				JSONExtractString(labels, 'direction') as direction,
				JSONExtractString(labels, 'interface') as interface_name
			FROM signoz_metrics.time_series_v4
			WHERE JSONExtractString(labels, '%s') = ?
				AND JSONExtractString(labels, 'k8s.namespace.name') = ?
		)
		SELECT
			toStartOfMinute(fromUnixTimestamp64Milli(s.unix_milli)) as timestamp,
			f.workload_name,
			f.namespace,
			f.direction,
			f.interface_name,
			s.metric_name,
			CASE
				WHEN (max(s.unix_milli) - min(s.unix_milli)) > 0
					THEN (max(s.value) - min(s.value)) / (max(s.unix_milli) - min(s.unix_milli)) * 1000
				ELSE 0.0
			END as metric_value
		FROM signoz_metrics.samples_v4 s
		INNER JOIN filtered_fingerprints f ON s.fingerprint = f.fingerprint
		WHERE s.metric_name = 'k8s.pod.network.io'
			AND fromUnixTimestamp64Milli(s.unix_milli) >= toDateTime(?)
			AND fromUnixTimestamp64Milli(s.unix_milli) <= toDateTime(?)
		GROUP BY
			timestamp,
			f.workload_name,
			f.namespace,
			f.direction,
			f.interface_name,
			s.metric_name
		ORDER BY timestamp ASC, direction, interface_name
	`, labelWorkloadType, labelWorkloadType)
	return query
}

// 3-3. Network I/O
func buildQueryWorkloadNetworkIoParams(startTime, endTime, deployment, namespace string) []interface{} {
	var params []interface{}

	// CTE 파라미터 (deployment, namespace)
	params = append(params, deployment)
	params = append(params, namespace)
	// WHERE 파라미터 (startTime, endTime)
	params = append(params, startTime)
	params = append(params, endTime)
	return params
}

// 3-4. Network Error Count
func buildQueryWorkloadNetworkError(workloadType string) string {
	workloadKey := getWorkloadKey(workloadType) // "deployment", "statefulset", "daemonset"
	labelWorkloadType := "k8s." + workloadKey + ".name"

	query := fmt.Sprintf(`
		WITH filtered_fingerprints AS (
			SELECT DISTINCT fingerprint,
				JSONExtractString(labels, '%s') as workload_name,
				JSONExtractString(labels, 'k8s.namespace.name') as namespace,
				JSONExtractString(labels, 'direction') as direction,
				JSONExtractString(labels, 'interface') as interface_name
			FROM signoz_metrics.time_series_v4
			WHERE JSONExtractString(labels, '%s') = ?
				AND JSONExtractString(labels, 'k8s.namespace.name') = ?
		)
		SELECT
			toStartOfMinute(fromUnixTimestamp64Milli(s.unix_milli)) as timestamp,
			f.workload_name,
			f.namespace,
			f.direction,
			f.interface_name,
			s.metric_name,
			CASE
				WHEN (max(s.unix_milli) - min(s.unix_milli)) > 0
					THEN (max(s.value) - min(s.value)) / (max(s.unix_milli) - min(s.unix_milli)) * 1000
				ELSE 0.0
			END as metric_value
		FROM signoz_metrics.samples_v4 s
		INNER JOIN filtered_fingerprints f ON s.fingerprint = f.fingerprint
		WHERE s.metric_name = 'k8s.pod.network.errors'
			AND fromUnixTimestamp64Milli(s.unix_milli) >= toDateTime(?)
			AND fromUnixTimestamp64Milli(s.unix_milli) <= toDateTime(?)
		GROUP BY
			timestamp,
			f.workload_name,
			f.namespace,
			f.direction,
			f.interface_name,
			s.metric_name
		ORDER BY timestamp ASC, direction, interface_name
		`, labelWorkloadType, labelWorkloadType)
	return query
}

// 3-4. Network Error Count
func buildQueryWorkloadNetworkErrorParams(startTime, endTime, deployment, namespace string) []interface{} {
	var params []interface{}

	// CTE 파라미터 (deployment, namespace)
	params = append(params, deployment)
	params = append(params, namespace)
	// WHERE 파라미터 (startTime, endTime)
	params = append(params, startTime)
	params = append(params, endTime)
	return params
}

// 통합 메트릭 쿼리 (CPU + Memory + NetworkIO + NetworkError)
func buildQueryWorkloadAllMetrics(workloadType string) string {
	// Rollout은 k8s.rollout.name 라벨이 없으므로 별도 쿼리 사용
	if workloadType == "Rollout" {
		return buildQueryWorkloadAllMetricsForRollout()
	}

	workloadKey := getWorkloadKey(workloadType)
	labelWorkloadType := "k8s." + workloadKey + ".name"

	query := fmt.Sprintf(`
		WITH filtered_fingerprints AS (
			SELECT DISTINCT fingerprint,
				JSONExtractString(labels, '%s') as workload_name,
				JSONExtractString(labels, 'k8s.namespace.name') as namespace,
				JSONExtractString(labels, 'direction') as direction,
				JSONExtractString(labels, 'interface') as interface_name
			FROM signoz_metrics.time_series_v4
			WHERE metric_name IN (
				'k8s.pod.cpu.usage', 'k8s.container.cpu_request', 'k8s.container.cpu_limit',
				'k8s.pod.memory.usage', 'k8s.container.memory_request', 'k8s.container.memory_limit',
				'k8s.pod.network.io', 'k8s.pod.network.errors'
			)
				AND JSONExtractString(labels, '%s') = ?
				AND JSONExtractString(labels, 'k8s.namespace.name') = ?
		)
		SELECT
			toStartOfMinute(fromUnixTimestamp64Milli(s.unix_milli)) as timestamp,
			f.workload_name,
			f.namespace,
			f.direction,
			f.interface_name,
			s.metric_name,
			CASE
				WHEN s.metric_name IN ('k8s.pod.cpu.usage', 'k8s.pod.memory.usage') THEN avg(s.value)
				WHEN s.metric_name IN ('k8s.pod.network.io', 'k8s.pod.network.errors') THEN
					CASE
						WHEN (max(s.unix_milli) - min(s.unix_milli)) > 0
							THEN (max(s.value) - min(s.value)) / (max(s.unix_milli) - min(s.unix_milli)) * 1000
						ELSE 0.0
					END
				ELSE any(s.value)
			END as metric_value
		FROM signoz_metrics.samples_v4 s
		INNER JOIN filtered_fingerprints f ON s.fingerprint = f.fingerprint
		WHERE s.metric_name IN (
			'k8s.pod.cpu.usage', 'k8s.container.cpu_request', 'k8s.container.cpu_limit',
			'k8s.pod.memory.usage', 'k8s.container.memory_request', 'k8s.container.memory_limit',
			'k8s.pod.network.io', 'k8s.pod.network.errors'
		)
			AND fromUnixTimestamp64Milli(s.unix_milli) >= toDateTime(?)
			AND fromUnixTimestamp64Milli(s.unix_milli) <= toDateTime(?)
		GROUP BY
			timestamp,
			f.workload_name,
			f.namespace,
			f.direction,
			f.interface_name,
			s.metric_name
		ORDER BY timestamp ASC, s.metric_name
	`, labelWorkloadType, labelWorkloadType)

	return query
}

// Rollout용 통합 메트릭 쿼리 (kube_pod_info + replicaset_deployment_map join 방식)
// OTel k8sattributes가 Argo Rollouts CRD를 인식하지 못하므로 KSM 기반 조회 사용
// 최적화: time_series_v4에서 fingerprint 먼저 확보 → samples_v4는 fingerprint 기반 필터링
func buildQueryWorkloadAllMetricsForRollout() string {
	query := `
		WITH rollout_replicasets AS (
			SELECT replicaset_name
			FROM signoz_traces.replicaset_deployment_map FINAL
			WHERE cluster_name = ?
				AND namespace = ?
				AND deployment_name = ?
				AND workload_type = 'Rollout'
		),
		pod_info_fingerprints AS (
			SELECT fingerprint, JSONExtractString(labels, 'pod') AS pod_name
			FROM signoz_metrics.time_series_v4
			WHERE metric_name = 'kube_pod_info'
				AND JSONExtractString(labels, 'k8s.cluster.name') = ?
				AND JSONExtractString(labels, 'namespace') = ?
				AND JSONExtractString(labels, 'created_by_name') IN (SELECT replicaset_name FROM rollout_replicasets)
		),
		active_pods AS (
			SELECT DISTINCT pf.pod_name
			FROM signoz_metrics.samples_v4 s
			INNER JOIN pod_info_fingerprints pf ON s.fingerprint = pf.fingerprint
			WHERE s.metric_name = 'kube_pod_info'
				AND s.value = 1
				AND toDateTime64(s.unix_milli / 1000, 9) >= now() - INTERVAL 30 MINUTE
		),
		metric_fingerprints AS (
			SELECT
				fingerprint,
				? AS workload_name,
				JSONExtractString(labels, 'k8s.namespace.name') as namespace,
				JSONExtractString(labels, 'direction') as direction,
				JSONExtractString(labels, 'interface') as interface_name
			FROM signoz_metrics.time_series_v4
			WHERE metric_name IN (
				'k8s.pod.cpu.usage', 'k8s.container.cpu_request', 'k8s.container.cpu_limit',
				'k8s.pod.memory.usage', 'k8s.container.memory_request', 'k8s.container.memory_limit',
				'k8s.pod.network.io', 'k8s.pod.network.errors'
			)
				AND JSONExtractString(labels, 'k8s.pod.name') IN (SELECT pod_name FROM active_pods)
		)
		SELECT
			toStartOfMinute(fromUnixTimestamp64Milli(s.unix_milli)) as timestamp,
			mf.workload_name,
			mf.namespace,
			mf.direction,
			mf.interface_name,
			s.metric_name,
			CASE
				WHEN s.metric_name IN ('k8s.pod.cpu.usage', 'k8s.pod.memory.usage') THEN avg(s.value)
				WHEN s.metric_name IN ('k8s.pod.network.io', 'k8s.pod.network.errors') THEN
					CASE
						WHEN (max(s.unix_milli) - min(s.unix_milli)) > 0
							THEN (max(s.value) - min(s.value)) / (max(s.unix_milli) - min(s.unix_milli)) * 1000
						ELSE 0.0
					END
				ELSE any(s.value)
			END as metric_value
		FROM signoz_metrics.samples_v4 s
		INNER JOIN metric_fingerprints mf ON s.fingerprint = mf.fingerprint
		WHERE s.metric_name IN (
			'k8s.pod.cpu.usage', 'k8s.container.cpu_request', 'k8s.container.cpu_limit',
			'k8s.pod.memory.usage', 'k8s.container.memory_request', 'k8s.container.memory_limit',
			'k8s.pod.network.io', 'k8s.pod.network.errors'
		)
			AND fromUnixTimestamp64Milli(s.unix_milli) >= toDateTime(?)
			AND fromUnixTimestamp64Milli(s.unix_milli) <= toDateTime(?)
		GROUP BY
			timestamp,
			mf.workload_name,
			mf.namespace,
			mf.direction,
			mf.interface_name,
			s.metric_name
		ORDER BY timestamp ASC, s.metric_name
	`
	return query
}

func buildQueryWorkloadAllMetricsParams(startTime, endTime, deployment, namespace string) []interface{} {
	var params []interface{}

	// CTE 파라미터 (deployment, namespace)
	params = append(params, deployment)
	params = append(params, namespace)
	// WHERE 파라미터 (startTime, endTime)
	params = append(params, startTime)
	params = append(params, endTime)
	return params
}

// Rollout용 파라미터 함수 (kube_pod_info join 쿼리용)
// 최적화된 쿼리 구조: rollout_replicasets → pod_info_fingerprints → active_pods → metric_fingerprints
func buildQueryWorkloadAllMetricsParamsForRollout(startTime, endTime, clusterName, namespace, workloadName string) []interface{} {
	var params []interface{}

	// CTE 1: rollout_replicasets 파라미터
	params = append(params, clusterName)  // cluster_name
	params = append(params, namespace)    // namespace
	params = append(params, workloadName) // deployment_name

	// CTE 2: pod_info_fingerprints 파라미터
	params = append(params, clusterName) // k8s.cluster.name
	params = append(params, namespace)   // namespace

	// CTE 4: metric_fingerprints 파라미터
	params = append(params, workloadName) // AS workload_name

	// WHERE 파라미터 (startTime, endTime)
	params = append(params, startTime)
	params = append(params, endTime)
	return params
}

// parseMetricResult 메트릭 결과를 파싱하여 해당 맵에 추가
func parseMetricResult(
	metricName string,
	timestampStr string,
	metricValue float64,
	direction sql.NullString,
	interfaceName sql.NullString,
	cpuMetricsMap map[string]*servicemap.WorkloadMetric,
	memoryMetricsMap map[string]*servicemap.WorkloadMetric,
	networkIoMetricsMap map[string]*servicemap.WorkloadMetric,
	networkErrorMetricsMap map[string]*servicemap.WorkloadMetric,
) {
	switch metricName {
	case "k8s.pod.cpu.usage":
		if cpuMetricsMap["cpuUsage"] == nil {
			cpuMetricsMap["cpuUsage"] = &servicemap.WorkloadMetric{QueryName: "cpuUsage", Values: []servicemap.WorkloadMetricValue{}}
		}
		cpuMetricsMap["cpuUsage"].Values = append(cpuMetricsMap["cpuUsage"].Values, servicemap.WorkloadMetricValue{
			Timestamp: timestampStr,
			Value:     metricValue,
		})

	case "k8s.container.cpu_request":
		if cpuMetricsMap["cpuRequest"] == nil {
			cpuMetricsMap["cpuRequest"] = &servicemap.WorkloadMetric{QueryName: "cpuRequest", Values: []servicemap.WorkloadMetricValue{}}
		}
		cpuMetricsMap["cpuRequest"].Values = append(cpuMetricsMap["cpuRequest"].Values, servicemap.WorkloadMetricValue{
			Timestamp: timestampStr,
			Value:     metricValue,
		})

	case "k8s.container.cpu_limit":
		if cpuMetricsMap["cpuLimit"] == nil {
			cpuMetricsMap["cpuLimit"] = &servicemap.WorkloadMetric{QueryName: "cpuLimit", Values: []servicemap.WorkloadMetricValue{}}
		}
		cpuMetricsMap["cpuLimit"].Values = append(cpuMetricsMap["cpuLimit"].Values, servicemap.WorkloadMetricValue{
			Timestamp: timestampStr,
			Value:     metricValue,
		})

	case "k8s.pod.memory.usage":
		if memoryMetricsMap["memoryUsage"] == nil {
			memoryMetricsMap["memoryUsage"] = &servicemap.WorkloadMetric{QueryName: "memoryUsage", Values: []servicemap.WorkloadMetricValue{}}
		}
		memoryMetricsMap["memoryUsage"].Values = append(memoryMetricsMap["memoryUsage"].Values, servicemap.WorkloadMetricValue{
			Timestamp: timestampStr,
			Value:     metricValue,
		})

	case "k8s.container.memory_request":
		if memoryMetricsMap["memoryRequest"] == nil {
			memoryMetricsMap["memoryRequest"] = &servicemap.WorkloadMetric{QueryName: "memoryRequest", Values: []servicemap.WorkloadMetricValue{}}
		}
		memoryMetricsMap["memoryRequest"].Values = append(memoryMetricsMap["memoryRequest"].Values, servicemap.WorkloadMetricValue{
			Timestamp: timestampStr,
			Value:     metricValue,
		})

	case "k8s.container.memory_limit":
		if memoryMetricsMap["memoryLimit"] == nil {
			memoryMetricsMap["memoryLimit"] = &servicemap.WorkloadMetric{QueryName: "memoryLimit", Values: []servicemap.WorkloadMetricValue{}}
		}
		memoryMetricsMap["memoryLimit"].Values = append(memoryMetricsMap["memoryLimit"].Values, servicemap.WorkloadMetricValue{
			Timestamp: timestampStr,
			Value:     metricValue,
		})

	case "k8s.pod.network.io":
		key := fmt.Sprintf("%s_%s", direction.String, interfaceName.String)
		if networkIoMetricsMap[key] == nil {
			networkIoMetricsMap[key] = &servicemap.WorkloadMetric{
				QueryName: "networkIo",
				Labels: map[string]string{
					"direction": direction.String,
					"interface": interfaceName.String,
				},
				Values: []servicemap.WorkloadMetricValue{},
			}
		}
		networkIoMetricsMap[key].Values = append(networkIoMetricsMap[key].Values, servicemap.WorkloadMetricValue{
			Timestamp: timestampStr,
			Value:     metricValue,
		})

	case "k8s.pod.network.errors":
		key := fmt.Sprintf("%s_%s", direction.String, interfaceName.String)
		if networkErrorMetricsMap[key] == nil {
			networkErrorMetricsMap[key] = &servicemap.WorkloadMetric{
				QueryName: "networkError",
				Labels: map[string]string{
					"direction": direction.String,
					"interface": interfaceName.String,
				},
				Values: []servicemap.WorkloadMetricValue{},
			}
		}
		networkErrorMetricsMap[key].Values = append(networkErrorMetricsMap[key].Values, servicemap.WorkloadMetricValue{
			Timestamp: timestampStr,
			Value:     metricValue,
		})
	}
}

// Golden Signal 쿼리
func buildQueryGoldenSignal(req *servicemap.WorkloadHoverRequest) string {
	query := `
		SELECT 
			COALESCE(NULLIF(k8s_cluster_name, ''), 'unknown') as cluster,
			COALESCE(NULLIF(dest_namespace, ''), 'unknown') as dest_namespace,
			dest as dest_workload,

			-- Rate: 초당 요청 수 (RPS - 0 나눗셈 방지)
			CASE
				WHEN (toUnixTimestamp(max(timestamp)) - toUnixTimestamp(min(timestamp))) > 0
				THEN sum(total_count) / (toUnixTimestamp(max(timestamp)) - toUnixTimestamp(min(timestamp)))
				ELSE 0.0
			END as rate_rps,
			
			-- Error: 에러율 (% - 0 나눗셈 방지)
			CASE
				WHEN sum(total_count) > 0
				THEN (sum(error_count) * 100.0) / sum(total_count)
				ELSE 0.0
			END as error_rate_percent,
			
			-- Duration: 응답 시간 메트릭들 - 0 나눗셈 방지
			CASE
				WHEN sum(duration_count) > 0
				THEN sum(duration_sum) / sum(duration_count)
				ELSE 0.0
			END as duration_avg_ms,
			
			-- 백분위수는 이미 계산되어 있지만, 정확한 집계를 위해서는 quantile 함수 사용
			quantile(0.5)(duration_p50) as duration_p50_ms,
			quantile(0.95)(duration_p95) as duration_p95_ms,
			quantile(0.99)(duration_p99) as duration_p99_ms,
			
			-- 추가 메트릭
			sum(total_count) as total_requests,
			sum(error_count) as total_errors
		FROM signoz_traces.network_map_connections 
		WHERE 1=1
		  AND timestamp >= toDateTime(?)
		  AND timestamp <= toDateTime(?)
		  AND k8s_cluster_name = ?
		  AND dest_namespace = ?
		  AND dest = ?
		GROUP BY dest_workload, cluster, dest_namespace
		HAVING total_requests > 0
		`

	return query
}

// Golden Signal Query Param
func buildQueryParamsGoldenSignal(startTime, endTime string, cluster, namespace, workload string) []interface{} {
	var params []interface{}

	params = append(params, startTime, endTime)
	params = append(params, cluster)
	params = append(params, namespace)
	params = append(params, workload)

	return params
}

// Top Peer 쿼리
func buildQueryTopPeers(req *servicemap.WorkloadHoverRequest) string {
	query := `
		SELECT 
			COALESCE(NULLIF(k8s_cluster_name, ''), 'unknown') as cluster,
			COALESCE(NULLIF(dest_namespace, ''), 'unknown') as dest_namespace,
			dest as dest_workload,
			COALESCE(NULLIF(src_namespace, ''), 'unknown') as src_namespace,
			src as src_workload,
			
			-- Rate: 초당 요청 수 (RPS - 0 나눗셈 방지)
			CASE
				WHEN (toUnixTimestamp(max(timestamp)) - toUnixTimestamp(min(timestamp))) > 0
				THEN sum(total_count) / (toUnixTimestamp(max(timestamp)) - toUnixTimestamp(min(timestamp)))
				ELSE 0.0
			END as rate_rps,
			
			-- Error: 에러율 (% - 0 나눗셈 방지)
			CASE
				WHEN sum(total_count) > 0
				THEN (sum(error_count) * 100.0) / sum(total_count)
				ELSE 0.0
			END as error_rate_percent,
			
			-- Duration: 응답 시간 메트릭들 - 0 나눗셈 방지
			CASE
				WHEN sum(duration_count) > 0
				THEN sum(duration_sum) / sum(duration_count)
				ELSE 0.0
			END as duration_avg_ms,
			
			-- 백분위수는 이미 계산되어 있지만, 정확한 집계를 위해서는 quantile 함수 사용
			quantile(0.5)(duration_p50) as duration_p50_ms,
			quantile(0.95)(duration_p95) as duration_p95_ms,
			quantile(0.99)(duration_p99) as duration_p99_ms,
			
			-- 추가 메트릭
			sum(total_count) as total_requests,
			sum(error_count) as total_errors
		FROM signoz_traces.network_map_connections 
		WHERE 1=1
		  AND timestamp >= toDateTime(?)
		  AND timestamp <= toDateTime(?)
		  AND k8s_cluster_name = ?
		  AND (src_namespace = ? OR dest_namespace = ?)
		  AND (src = ? OR dest = ?)
		GROUP BY dest_workload, cluster, dest_namespace, src_workload, src_namespace
		HAVING total_requests > 0
		ORDER BY total_requests desc
		LIMIT 5
	`
	return query
}

func buildQueryParamsTopPeers(startTime, endTime, cluster, namespace, workload string) []interface{} {
	var params []interface{}

	params = append(params, startTime, endTime, cluster, namespace, namespace, workload, workload)

	return params
}

// =================================== 4. ExecuteEdgeQuery Helper functions ========================================

// ==================================================================================== //
// 엣지 디테일 쿼리 통합 메서드
// ==================================================================================== //

// GetEdgeDetails queryType에 따라 엣지 상세 정보 조회 (통합 구현)
// 모든 엣지 타입을 처리하는 통합 구현:
// - InternalToInternal: 내부 서비스 → 내부 서비스
// - InternalToExternal: 내부 서비스 → 외부 서비스
// - ExternalToInternal: 외부 서비스 → 내부 서비스 (미지원)
//
// 3가지 쿼리를 순차 실행:
// 1. TopSlowRequest: 가장 느린 요청 Top N
// 2. RecentError: 최근 에러 발생 요청
// 3. Requests: 시간 범위 내 모든 요청
func (r *ServiceMapRepositoryImpl) GetEdgeDetails(
	ctx context.Context,
	queryType string,
	parsedParam servicemap.ParsedParam,
	startTimeFormatted, endTimeFormatted string,
) (*servicemap.EdgeTraceDetailResponse, error) {

	// 1. TopSlowRequest
	topSlowQuery, topSlowParams := selectTopSlowRequestQuery(queryType, startTimeFormatted, endTimeFormatted, parsedParam)
	topSlowRequests, err := r.executeTopSlowRequestQuery(ctx, topSlowQuery, topSlowParams, parsedParam)
	if err != nil {
		return nil, err
	}

	// 2. RecentError
	recentErrorQuery, recentErrorParams := selectRecentErrorQuery(queryType, startTimeFormatted, endTimeFormatted, parsedParam)
	recentErrors, err := r.executeRecentErrorQuery(ctx, recentErrorQuery, recentErrorParams, parsedParam)
	if err != nil {
		return nil, err
	}

	// 3. Requests
	requestsQuery, requestsParams := selectRequestsQuery(queryType, startTimeFormatted, endTimeFormatted, parsedParam)
	requests, protocol, err := r.executeRequestsQuery(ctx, requestsQuery, requestsParams, parsedParam)
	if err != nil {
		return nil, err
	}

	// 4. Build response
	response := &servicemap.EdgeTraceDetailResponse{
		SrcWorkload:     parsedParam.SrcWorkload,
		SrcNamespace:    parsedParam.SrcNamespace,
		DestWorkload:    parsedParam.DstWorkload,
		DestNamespace:   parsedParam.DstNamespace,
		Protocol:        protocol,
		TopSlowRequests: topSlowRequests,
		RecentErrors:    recentErrors,
		Requests:        requests,
		Cursor:          servicemap.CursorMeta{},
	}

	return response, nil
}

// ==================================================================================== //
// 엣지 쿼리 헬퍼 함수 - 쿼리 선택
// ==================================================================================== //

// selectTopSlowRequestQuery queryType에 따라 TopSlowRequest 쿼리와 파라미터 반환
func selectTopSlowRequestQuery(queryType string, startTime, endTime string, params servicemap.ParsedParam) (string, []interface{}) {
	switch queryType {
	case "InternalToInternal":
		query := queries.BuildQueryTopSlowRequest()
		// ✅ dest_raw 값 사용 (server_addr와 매칭), 빈 값이면 기존 DstWorkload로 폴백
		dstWorkloadRaw := params.DstWorkloadRaw
		if dstWorkloadRaw == "" {
			dstWorkloadRaw = params.DstWorkload
		}
		queryParams := queries.BuildQueryParamsTopSlowRequest(
			startTime, endTime,
			params.SrcCluster, params.SrcNamespace, params.SrcWorkload,
			params.DstCluster, params.DstNamespace, dstWorkloadRaw,
			params.Limit,
		)
		return query, queryParams
	case "InternalToExternal":
		query := queries.BuildQueryTopSlowInternalToExternal()
		queryParams := queries.BuildQueryParamsTopSlowInternalToExternal(
			startTime, endTime,
			params.SrcCluster, params.SrcNamespace, params.SrcWorkload,
			params.DstCluster, params.DstNamespace, params.DstWorkload,
			params.Limit,
		)
		return query, queryParams
	case "ExternalToInternal":
		// TODO: implement when needed
		return "", nil
	default:
		return "", nil
	}
}

// selectRecentErrorQuery queryType에 따라 RecentError 쿼리와 파라미터 반환
func selectRecentErrorQuery(queryType string, startTime, endTime string, params servicemap.ParsedParam) (string, []interface{}) {
	switch queryType {
	case "InternalToInternal":
		query := queries.BuildQueryRecentError()
		// ✅ dest_raw 값 사용 (server_addr와 매칭), 빈 값이면 기존 DstWorkload로 폴백
		dstWorkloadRaw := params.DstWorkloadRaw
		if dstWorkloadRaw == "" {
			dstWorkloadRaw = params.DstWorkload
		}
		queryParams := queries.BuildQueryParamsRecentError(
			startTime, endTime,
			params.SrcCluster, params.SrcNamespace, params.SrcWorkload,
			params.DstCluster, params.DstNamespace, dstWorkloadRaw,
			params.Limit,
		)
		return query, queryParams
	case "InternalToExternal":
		query := queries.BuildQueryRecentErrorInternalToExternal()
		queryParams := queries.BuildQueryParamsRecentErrorInternalToExternal(
			startTime, endTime,
			params.SrcCluster, params.SrcNamespace, params.SrcWorkload,
			params.DstCluster, params.DstNamespace, params.DstWorkload,
			params.Limit,
		)
		return query, queryParams
	case "ExternalToInternal":
		// TODO: implement when needed
		return "", nil
	default:
		return "", nil
	}
}

// selectRequestsQuery queryType에 따라 Requests 쿼리와 파라미터 반환
func selectRequestsQuery(queryType string, startTime, endTime string, params servicemap.ParsedParam) (string, []interface{}) {
	switch queryType {
	case "InternalToInternal":
		query := queries.BuildQueryRequests()
		// ✅ dest_raw 값 사용 (server_addr와 매칭), 빈 값이면 기존 DstWorkload로 폴백
		dstWorkloadRaw := params.DstWorkloadRaw
		if dstWorkloadRaw == "" {
			dstWorkloadRaw = params.DstWorkload
		}
		queryParams := queries.BuildQueryParamsRequests(
			startTime, endTime,
			params.SrcCluster, params.SrcNamespace, params.SrcWorkload,
			params.DstCluster, params.DstNamespace, dstWorkloadRaw,
		)
		return query, queryParams
	case "InternalToExternal":
		query := queries.BuildQueryRequestsInternalToExternal()
		queryParams := queries.BuildQueryParamsRequestsInternalToExternal(
			startTime, endTime,
			params.SrcCluster, params.SrcNamespace, params.SrcWorkload,
			params.DstCluster, params.DstNamespace, params.DstWorkload,
		)
		return query, queryParams
	case "ExternalToInternal":
		// TODO: implement when needed
		return "", nil
	default:
		return "", nil
	}
}

// ==================================================================================== //
// 엣지 쿼리 헬퍼 함수 - 쿼리 실행
// ==================================================================================== //

// executeTopSlowRequestQuery TopSlowRequest 쿼리 실행 및 결과 반환
func (r *ServiceMapRepositoryImpl) executeTopSlowRequestQuery(
	ctx context.Context,
	query string,
	params []interface{},
	parsedParam servicemap.ParsedParam,
) ([]servicemap.TopSlowRequest, error) {
	// Query execution
	rows, err := infrastructure.QueryClickHouseWithContext(ctx, query, params...)
	if err != nil {
		log.Printf("[ERROR] TopSlowRequest Query Failed: %v", err)
		return nil, fmt.Errorf("failed to query top slow requests: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			log.Printf("[WARN] failed to close rows: %v", closeErr)
		}
	}()

	// Scan results
	var timestamp time.Time
	var protocol, traceId, method, path string
	var isError, status uint16
	var latencyMs float64
	var clusterName, destNamespace, srcWorkloadScan, dstWorkloadScan string

	results := []servicemap.TopSlowRequest{}
	for rows.Next() {
		if err := rows.Scan(&protocol, &status, &isError, &timestamp, &traceId, &method,
			&path, &latencyMs, &srcWorkloadScan, &dstWorkloadScan, &clusterName,
			&parsedParam.SrcNamespace, &destNamespace); err != nil {
			log.Printf("[ERROR] Scan Failed: %v", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, servicemap.TopSlowRequest{
			Timestamp: timestamp.Format(time.RFC3339),
			TraceId:   traceId,
			Path:      path,
			Method:    method,
			Status:    int(status),
			IsError:   isError == 1,
			Latency:   latencyMs,
		})
	}
	return results, nil
}

// executeRecentErrorQuery RecentError 쿼리 실행 및 결과 반환
func (r *ServiceMapRepositoryImpl) executeRecentErrorQuery(
	ctx context.Context,
	query string,
	params []interface{},
	parsedParam servicemap.ParsedParam,
) ([]servicemap.RecentError, error) {
	// Query execution
	rows, err := infrastructure.QueryClickHouseWithContext(ctx, query, params...)
	if err != nil {
		log.Printf("[ERROR] RecentError Query Failed: %v", err)
		return nil, fmt.Errorf("failed to query recent errors: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			log.Printf("[WARN] failed to close rows: %v", closeErr)
		}
	}()

	// Scan results
	var timestamp time.Time
	var protocol, traceId, method, path string
	var isError, status uint16
	var latencyMs float64
	var clusterName, destNamespace, srcWorkloadScan, dstWorkloadScan string

	results := []servicemap.RecentError{}
	for rows.Next() {
		if err := rows.Scan(&protocol, &status, &isError, &timestamp, &traceId, &method,
			&path, &latencyMs, &srcWorkloadScan, &dstWorkloadScan, &clusterName,
			&parsedParam.SrcNamespace, &destNamespace); err != nil {
			log.Printf("[ERROR] Scan Failed: %v", err)
			return nil, fmt.Errorf("failed to scan recent error row: %w", err)
		}
		results = append(results, servicemap.RecentError{
			Timestamp: timestamp.Format(time.RFC3339),
			TraceId:   traceId,
			Path:      path,
			Method:    method,
			Status:    int(status),
			IsError:   isError == 1,
			Latency:   latencyMs,
		})
	}
	return results, nil
}

// executeRequestsQuery Requests 쿼리 실행 및 결과와 프로토콜 반환
func (r *ServiceMapRepositoryImpl) executeRequestsQuery(
	ctx context.Context,
	query string,
	params []interface{},
	parsedParam servicemap.ParsedParam,
) ([]servicemap.Requests, string, error) {
	// Query execution
	rows, err := infrastructure.QueryClickHouseWithContext(ctx, query, params...)
	if err != nil {
		log.Printf("[ERROR] Requests Query Failed: %v", err)
		return nil, "", fmt.Errorf("failed to query requests: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			log.Printf("[WARN] failed to close rows: %v", closeErr)
		}
	}()

	// Scan results
	var timestamp time.Time
	var protocol, traceId, method, path string
	var isError, status uint16
	var latencyMs float64
	var clusterName, destNamespace, srcWorkloadScan, dstWorkloadScan string

	results := []servicemap.Requests{}
	for rows.Next() {
		if err := rows.Scan(&protocol, &status, &isError, &timestamp, &traceId, &method,
			&path, &latencyMs, &srcWorkloadScan, &dstWorkloadScan, &clusterName,
			&parsedParam.SrcNamespace, &destNamespace); err != nil {
			log.Printf("[ERROR] Scan Failed: %v", err)
			return nil, "", fmt.Errorf("failed to scan request row: %w", err)
		}
		results = append(results, servicemap.Requests{
			Timestamp:  timestamp.Format(time.RFC3339),
			TraceId:    traceId,
			Connection: fmt.Sprintf("%s -> %s", parsedParam.SrcWorkload, parsedParam.DstWorkload),
			Path:       path,
			Method:     method,
			Status:     int(status),
			IsError:    isError == 1,
			Latency:    latencyMs,
			Protocol:   protocol,
		})
	}
	return results, protocol, nil
}
