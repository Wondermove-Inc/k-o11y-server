package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/domain/servicemap"
	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/repository"
	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/utils"

	"github.com/patrickmn/go-cache"
)

type ServiceMapService interface {
	GetTopology(req *servicemap.TopologyRequest) (*servicemap.TopologyResponse, error)
	GetWorkloadDetails(req *servicemap.WorkloadDetailRequest) (*servicemap.WorkloadDetailResponse, error)
	GetWorkloadHover(req *servicemap.WorkloadHoverRequest) (*servicemap.WorkloadHoverResponse, error) // 호버 API
	GetEdgeTraceDetails(req *servicemap.EdgeTraceDetailRequest) (*servicemap.EdgeTraceDetailResponse, error)
}

type ServiceMapServiceImpl struct {
	serviceMapRepository repository.ServiceMapRepository
	cache                *cache.Cache
	cacheMutex           sync.RWMutex
}

var serviceMapServiceInstance ServiceMapService = nil

func ServiceMapServiceInstance() ServiceMapService {
	if serviceMapServiceInstance == nil {
		// go-cache 인스턴스 생성 (API 명세서 캐싱 전략 기반)
		cacheInstance := cache.New(5*time.Minute, 10*time.Minute)

		serviceMapServiceInstance = &ServiceMapServiceImpl{
			serviceMapRepository: repository.ServiceMapRepositoryInstance(),
			cache:                cacheInstance,
		}
	}
	return serviceMapServiceInstance
}

// GetTopology 메인 서비스맵 토폴로지 조회 (통합 캐시 사용)
func (s *ServiceMapServiceImpl) GetTopology(req *servicemap.TopologyRequest) (*servicemap.TopologyResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 토폴로지별 캐시 키 생성 (StartTime/EndTime 기반)
	timeRange := fmt.Sprintf("%s_%s", req.StartTime, req.EndTime)
	cacheKey := fmt.Sprintf("topology_%s_%v_%v_%v_%v_%s",
		timeRange, req.Cluster, req.Namespace, req.Protocol, req.Status)

	// 캐시에서 조회 시도
	s.cacheMutex.RLock()
	if cached, found := s.cache.Get(cacheKey); found {
		s.cacheMutex.RUnlock()
		if data, ok := cached.(*servicemap.TopologyResponse); ok {
			// log.Printf("Cache hit for topology: %s to %s", req.StartTime, req.EndTime)
			return data, nil
		}
	}
	s.cacheMutex.RUnlock()

	// Repository에서 데이터 조회
	start := time.Now()
	response, err := s.serviceMapRepository.GetTopology(ctx, req)
	duration := time.Since(start)
	if err != nil {
		log.Printf("[ERROR] Failed to get topology - Cluster: %s, Namespace: %s, Error: %v", req.Cluster, req.Namespace, err)
		return nil, fmt.Errorf("failed to fetch topology data: %w", err)
	}
	if duration > 2*time.Second {
		log.Printf("[WARN] Slow topology query - Duration: %v, Cluster: %s",
			duration, req.Cluster)
	}

	// 노드별 RED 메트릭을 통합 캐시에서 조회하여 동기화
	// calculatedTimeRange := s.formatTimeRangeFromRequest(req.StartTime, req.EndTime)
	// s.enrichTopologyWithUnifiedCache(response, calculatedTimeRange)

	// 추가 비즈니스 로직 처리
	// s.enrichTopologyData(response)

	// 토폴로지 캐시에 저장 (30초 TTL - 호버 캐시와 동일)
	s.cacheMutex.Lock()
	s.cache.Set(cacheKey, response, 30*time.Second)
	s.cacheMutex.Unlock()

	// log.Printf("GetTopology completed: %d nodes, %d edges", len(response.Nodes), len(response.Edges))
	return response, nil
}

// GetServiceDetails 서비스 상세 정보 조회 (30초 TTL 캐싱)
func (s *ServiceMapServiceImpl) GetWorkloadDetails(req *servicemap.WorkloadDetailRequest) (*servicemap.WorkloadDetailResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// 입력 검증
	if req.WorkloadName == "" {
		return nil, fmt.Errorf("workload name is required")
	}

	// 캐시 키 생성 (3-tier 식별자 기반)
	cacheKey := fmt.Sprintf("workload_details_%s_%s_%s_%s",
		req.Cluster, req.Namespace, req.WorkloadName, req.StartTime, req.EndTime)

	// 캐시에서 조회 시도
	s.cacheMutex.RLock()
	if cached, found := s.cache.Get(cacheKey); found {
		s.cacheMutex.RUnlock()
		if data, ok := cached.(*servicemap.WorkloadDetailResponse); ok {
			// log.Printf("Cache hit for workload details: %s", req.WorkloadName)
			return data, nil
		}
	}
	s.cacheMutex.RUnlock()

	// Repository에서 데이터 조회
	start := time.Now()
	response, err := s.serviceMapRepository.GetWorkloadDetails(ctx, req)
	duration := time.Since(start)
	if err != nil {
		log.Printf("[ERROR] Failed to get workload details - Cluster: %s, Namespace: %s, Workload: %s, Error: %v", req.Cluster, req.Namespace, req.WorkloadName, err)

		return nil, fmt.Errorf("failed to fetch workload details: %w", err)
	}
	if duration > 2*time.Second {
		log.Printf("[WARN] Slow workload details query - Duration: %v, Cluster: %s",
			duration, req.Cluster)
	}

	// 캐시에 저장 (30초 TTL)
	s.cacheMutex.Lock()
	s.cache.Set(cacheKey, response, 30*time.Second)
	s.cacheMutex.Unlock()

	// log.Printf("GetWorkloadDetails completed for workload: %s", req.WorkloadName)
	return response, nil
}

// GetWorkloadHover 워크로드 호버 정보 조회 (RED + Top Peers 통합)
func (s *ServiceMapServiceImpl) GetWorkloadHover(req *servicemap.WorkloadHoverRequest) (*servicemap.WorkloadHoverResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 토폴로지별 캐시 키 생성 (StartTime/EndTime 기반)
	timeRange := fmt.Sprintf("%s_%s", req.StartTime, req.EndTime)
	cacheKey := fmt.Sprintf("workload_hover_%s_%v_%v_%v_%v_%s",
		timeRange, req.Cluster, req.Namespace, req.WorkloadName)

	// 캐시에서 조회 시도
	s.cacheMutex.RLock()
	if cached, found := s.cache.Get(cacheKey); found {
		s.cacheMutex.RUnlock()
		if data, ok := cached.(*servicemap.WorkloadHoverResponse); ok {
			// log.Printf("Cache hit for workload hover: %s to %s", req.StartTime, req.EndTime)
			return data, nil
		}
	}
	s.cacheMutex.RUnlock()
	// 입력 검증
	if req.WorkloadName == "" {
		return nil, fmt.Errorf("service name is required")
	}

	start := time.Now()
	response, err := s.serviceMapRepository.GetWorkloadHover(ctx, req)
	duration := time.Since(start)
	if err != nil {
		log.Printf("[ERROR] Failed to get workload hover - Cluster: %s, Namespace: %s, Workload: %s, Error: %v",
			req.Cluster, req.Namespace, req.WorkloadName, err)
		return nil, fmt.Errorf("failed to fetch workload hover: %w", err)
	}
	if duration > 2*time.Second {
		log.Printf("[WARN] Slow workload hover query - Duration: %v, WorkloadName: %s",
			duration, req.WorkloadName)
	}
	// 통합 캐시에서 워크로드 데이터 조회
	// workloadData, cacheHit, err := s.getServiceData(ctx, req)
	// if err != nil {
	// 	log.Printf("Failed to get service data: %v", err)
	// 	return nil, fmt.Errorf("failed to fetch service hover data: %w", err)
	// }

	// 호버 응답 구성
	// response := &servicemap.WorkloadHoverResponse{
	// 	WorkloadName: req.WorkloadName,
	// 	Cluster:      req.Cluster,
	// 	Namespace:    req.Namespace,
	// 	NodeMetrics:  servicemap.NodeMetrics{},
	// 	TopPeers:     []servicemap.PeerInfo{},
	// }

	// log.Printf("GetWorkloadHover completed for workload: %s", req.WorkloadName)
	return response, nil
}

// GetConnectionDetails 연결 상세 정보 조회
func (s *ServiceMapServiceImpl) GetEdgeTraceDetails(req *servicemap.EdgeTraceDetailRequest) (*servicemap.EdgeTraceDetailResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// 입력 검증
	if req.EdgeId == "" {
		return nil, fmt.Errorf("edge ID is required")
	}

	// check context timeout
	if err := checkContextTimeout(ctx); err != nil {
		return nil, err
	}
	// create context
	queryCtx, cancel := createContext(ctx)
	defer cancel()

	// get formatted time
	startTimeFormatted, endTimeFormatted := getTimeFormatted(req)

	// get parsed parameters
	parsedParam := getParsedParam(req)

	// 조회
	start := time.Now()

	// queryType 결정
	queryType, err := determineQueryType(req.IsClientExternal, req.IsServerExternal)
	if err != nil {
		return nil, err
	}

	response, err := s.serviceMapRepository.ExecuteEdgeQuery(queryCtx, queryType, parsedParam, startTimeFormatted, endTimeFormatted)

	// response, err = s.serviceMapRepository.GetEdgeTraceDetails(ctx, req)
	duration := time.Since(start)
	if err != nil {
		log.Printf("[ERROR] Failed to get edge trace details - EdgeID: %s, Error: %v", req.EdgeId, err)
		return nil, fmt.Errorf("failed to fetch edge trace details: %w", err)
	}
	if duration > 2*time.Second {
		log.Printf("[WARN] Slow edge trace details query - Duration: %v, EdgeID: %s",
			duration, req.EdgeId)
	}

	// 추가 비즈니스 로직 처리
	// s.enrichConnectionDetails(response)

	// log.Printf("GetConnectionDetails completed for connection: %s", req.EdgeId)
	return response, nil
}

// ----------------------- Common -----------------------
func checkContextTimeout(ctx context.Context) error {
	// Context timeout 확인 (조기 탐지)
	if ctx.Err() != nil {
		log.Printf("[ERROR] Request context already canceled: %v", ctx.Err())
		return fmt.Errorf("request context canceled before query execution: %w", ctx.Err())
	}
	return nil
}

func createContext(ctx context.Context) (context.Context, context.CancelFunc) {
	queryCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
	return queryCtx, cancel
}

func getTimeFormatted(req *servicemap.EdgeTraceDetailRequest) (string, string) {
	startTimeFormatted := utils.FormatTimeForClickHouse(req.StartTime)
	endTimeFormatted := utils.FormatTimeForClickHouse(req.EndTime)

	return startTimeFormatted, endTimeFormatted
}

func getParsedParam(req *servicemap.EdgeTraceDetailRequest) servicemap.ParsedParam {
	srcCluster, srcNamespace, srcWorkload := utils.ParseEdge(req.Source)
	dstCluster, dstNamespace, dstWorkload := utils.ParseEdge(req.Destination)
	limit := 10 // TODO: limit 값 정의

	return servicemap.ParsedParam{
		SrcCluster:     srcCluster,
		SrcNamespace:   srcNamespace,
		SrcWorkload:    srcWorkload,
		SrcWorkloadRaw: req.SourceRaw, // ✅ 원본 src 값 (trace 매칭용)
		DstCluster:     dstCluster,
		DstNamespace:   dstNamespace,
		DstWorkload:    dstWorkload,
		DstWorkloadRaw: req.DestinationRaw, // ✅ 원본 dest 값 (trace 매칭용)
		Limit:          limit,
	}
}

// determineQueryType determines the query type based on edge external flags
func determineQueryType(isClientExternal, isServerExternal int) (string, error) {
	switch {
	case isClientExternal == 0 && isServerExternal == 0:
		return "InternalToInternal", nil
	case isClientExternal == 0 && isServerExternal == 1:
		return "InternalToExternal", nil
	case isClientExternal == 1 && isServerExternal == 0:
		return "", fmt.Errorf("external to internal edge is not supported yet")
	default:
		return "", fmt.Errorf("unexpected edge type: isClientExternal=%d, isServerExternal=%d", isClientExternal, isServerExternal)
	}
}
