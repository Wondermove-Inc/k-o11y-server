package servicemap

import "time"

// ==================================================================================== //
// TopologyRequest 토폴로지 요청 (POST 방식으로 변경 - 멀티클러스터 대용량 필터 지원)
type TopologyRequest struct {
	StartTime string   `json:"startTime" binding:"required"` // 시작 시간
	EndTime   string   `json:"endTime" binding:"required"`   // 종료 시간
	Cluster   []string `json:"cluster"`                      // 클러스터 목록 (대용량 지원)
	Namespace []string `json:"namespace"`                    // 네임스페이스 목록 (대용량 지원)
	Protocol  []string `json:"protocol"`                     // 프로토콜 목록
	Status    []string `json:"status"`                       // 상태 목록 - Ok, Error
	Workload  []string `json:"workload"`                     // 워크로드 목록 (대용량 지원)
}

// TopologyResponse 메인 토폴로지 응답 (Figma 0001.png) - 필터 데이터 통합
type TopologyResponse struct {
	Nodes     []ServiceMapNode `json:"nodes"`
	Edges     []ServiceMapEdge `json:"edges"`
	TimeRange string           `json:"timeRange"`
}

// ServiceMapNode React Flow 라이브러리 호환 노드
type ServiceMapNode struct {
	ID           string `json:"id"`           // "namespace$$workloadName" 형식
	WorkloadName string `json:"workloadName"` // React Flow 필수 필드
	Namespace    string `json:"namespace"`
	Cluster      string `json:"cluster"`
	IssueCount   int    `json:"issueCount"` // React Flow 에러 카운트
	IsExternal   *uint8 `json:"isExternal"` // React Flow 필수 필드
	Type         string `json:"type"`
	Status       string `json:"status"`
}

// ServiceMapEdge React Flow 라이브러리 호환 엣지
type ServiceMapEdge struct {
	ID          string `json:"id"`          // "source$$workload##target$$workload##protocol" 형식
	Source      string `json:"source"`      // "namespace$$workloadName" 형식
	Destination string `json:"destination"` // "namespace$$workloadName" 형식
	SrcRaw      string `json:"srcRaw"`      // ✅ 원본 src 값 (trace 매칭용) - network_map_connections.src_raw
	DestRaw     string `json:"destRaw"`     // ✅ 원본 dest 값 (trace 매칭용) - network_map_connections.dest_raw
	Protocol    string `json:"protocol"`    // "gRPC", "HTTP" 등 대문자
	IsError     bool   `json:"isError"`     // React Flow 에러 상태
	IsExternal  *uint8 `json:"isExternal"`  // React Flow 외부 연결 표시
}

// ==================================================================================== //

// WorkloadDetailRequest 워크로드 상세 요청
type WorkloadDetailRequest struct {
	Cluster      string `json:"cluster" binding:"required"`      // 3-tier 식별자: 클러스터
	Namespace    string `json:"namespace" binding:"required"`    // 3-tier 식별자: 네임스페이스
	WorkloadName string `json:"workloadName" binding:"required"` // 3-tier 식별자: 서비스명
	// TimeRange    string `json:"timeRange" binding:"required"`    // 시간 범위
	StartTime string `json:"startTime" binding:"required"` // 시작 시간
	EndTime   string `json:"endTime" binding:"required"`   // 종료 시간
}

// WorkloadDetailResponse 워크로드 상세 응답 (Figma 0004.png)
type WorkloadDetailResponse struct {
	WorkloadName           string           `json:"workloadName"`
	Cluster                string           `json:"cluster"`
	Namespace              string           `json:"namespace"`
	Kind                   string           `json:"kind"`
	Replicas               int              `json:"replicas"`
	RunningPods            int              `json:"runningPods"`
	CpuMetricList          []WorkloadMetric `json:"cpuMetrics"`
	MemoryMetricList       []WorkloadMetric `json:"memoryMetrics"`
	NetworkIoMetricList    []WorkloadMetric `json:"networkIoMetrics"`
	NetworkErrorMetricList []WorkloadMetric `json:"networkErrorMetrics"`
}

// WorkloadMetric 워크로드 메트릭
type WorkloadMetric struct {
	QueryName string                `json:"queryName"`
	Labels    map[string]string     `json:"labels"`
	Values    []WorkloadMetricValue `json:"values"`
}

type WorkloadMetricValue struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

// ==================================================================================== //

// WorkloadHoverRequest 워크로드 호버 정보 요청 (기존 TopPeersRequest 개선)
type WorkloadHoverRequest struct {
	Cluster      string `json:"cluster" binding:"required"`      // 3-tier 식별자: 클러스터
	Namespace    string `json:"namespace" binding:"required"`    // 3-tier 식별자: 네임스페이스
	WorkloadName string `json:"workloadName" binding:"required"` // 3-tier 식별자: 서비스명
	StartTime    string `json:"startTime" binding:"required"`    // 시작 시간
	EndTime      string `json:"endTime" binding:"required"`      // 종료 시간
}

// WorkloadHoverResponse 워크로드 호버 정보 응답 (RED + Top Peers 통합)
type WorkloadHoverResponse struct {
	WorkloadName string       `json:"workloadName"`
	Cluster      string       `json:"cluster"`
	Namespace    string       `json:"namespace"`
	NodeMetrics  *NodeMetrics `json:"metrics"`  // RED 메트릭 (RequestRate, ErrorRate, Latency)
	TopPeers     []PeerInfo   `json:"topPeers"` // 상위 연결 서비스
}

// NodeMetrics 노드 메트릭 (백엔드 표준 camelCase)
type NodeMetrics struct {
	RequestRate   float64 `json:"requestRate"` // req/s
	LatencyP95    float64 `json:"latencyP95"`  // ms
	ErrorRate     float64 `json:"errorRate"`   // %
	TotalRequests int64   `json:"totalRequests"`
	TotalErrors   int64   `json:"totalErrors"`
}

// PeerInfo Peer 정보
type PeerInfo struct {
	Rank          int     `json:"rank"`
	WorkloadName  string  `json:"workloadName"`
	Direction     string  `json:"direction"`
	RequestRate   float64 `json:"requestRate"` // req/s
	LatencyP95    float64 `json:"latencyP95"`  // ms
	ErrorRate     float64 `json:"errorRate"`   // %
	TotalRequests int64   `json:"totalRequests"`
}

// ==================================================================================== //

type EdgeTraceDetailRequest struct {
	EdgeId           string `json:"edgeId" binding:"required"`
	Source           string `json:"source" binding:"required"`
	Destination      string `json:"destination" binding:"required"`
	SourceRaw        string `json:"sourceRaw"`      // 원본 src 값 (trace 매칭용) - network_map_connections.src_raw
	DestinationRaw   string `json:"destinationRaw"` // 원본 dest 값 (trace 매칭용) - network_map_connections.dest_raw
	StartTime        string `json:"startTime" binding:"required"`
	EndTime          string `json:"endTime" binding:"required"`
	IsClientExternal int    `json:"isClientExternal"` // 0 = 내부, 1 = 외부
	IsServerExternal int    `json:"isServerExternal"` // 0 = 내부, 1 = 외부
}

type EdgeTraceDetailResponse struct {
	SrcWorkload     string           `json:"srcWorkload"`
	SrcNamespace    string           `json:"srcNamespace"`
	DestWorkload    string           `json:"destWorkload"`
	DestNamespace   string           `json:"destNamespace"`
	Protocol        string           `json:"protocol"`
	TopSlowRequests []TopSlowRequest `json:"topSlowRequests"`
	RecentErrors    []RecentError    `json:"recentErrors"`
	Requests        []Requests       `json:"requests"`
	Cursor          CursorMeta       `json:"cursor"`
}

type TopSlowRequest struct {
	Timestamp string  `json:"timestamp"`
	TraceId   string  `json:"traceId"`
	Path      string  `json:"path"`
	Method    string  `json:"method"`
	Status    int     `json:"status"`
	IsError   bool    `json:"isError"`
	Latency   float64 `json:"latency"`
}

type RecentError struct {
	Timestamp string  `json:"timestamp"`
	TraceId   string  `json:"traceId"`
	Path      string  `json:"path"`
	Method    string  `json:"method"`
	Status    int     `json:"status"`
	IsError   bool    `json:"isError"`
	Latency   float64 `json:"latency"`
}

type Requests struct {
	Timestamp  string  `json:"timestamp"`
	TraceId    string  `json:"traceId"`
	Connection string  `json:"connection"`
	Path       string  `json:"path"`
	Method     string  `json:"method"`
	Status     int     `json:"status"`
	IsError    bool    `json:"isError"`
	Latency    float64 `json:"latency"`
	Protocol   string  `json:"protocol"`
}
type CursorMeta struct {
	HasMore    bool   `json:"hasMore"`
	NextCursor string `json:"nextCursor"`
	PageSize   int    `json:"pageSize"`
	Total      int    `json:"total"` // 선택적
}

type ParsedParam struct {
	SrcCluster     string `json:"srcCluster"`
	SrcNamespace   string `json:"srcNamespace"`
	SrcWorkload    string `json:"srcWorkload"`
	SrcWorkloadRaw string `json:"srcWorkloadRaw"` // 원본 src 값 (trace 매칭용)
	DstCluster     string `json:"dstCluster"`
	DstNamespace   string `json:"dstNamespace"`
	DstWorkload    string `json:"dstWorkload"`
	DstWorkloadRaw string `json:"dstWorkloadRaw"` // 원본 dest 값 (trace 매칭용)
	Limit          int    `json:"limit"`
}

// ==================================================================================== //
// NodeMetric inbound 메트릭 정보
type NodeMetric struct {
	WorkloadName     string
	Namespace        string
	Cluster          string
	TotalRequests    uint64
	TotalErrors      uint64
	P95LatencyMs     float64
	ErrorRatePercent float64
}

// EdgeMetrics 엣지 메트릭
type EdgeMetrics struct {
	RequestCount int     `json:"requestCount"`
	RequestRate  float64 `json:"requestRate"`
	LatencyP95   int     `json:"latencyP95"`
	ErrorRate    float64 `json:"errorRate"`
	Bandwidth    string  `json:"bandwidth"`
}

// DeploymentInfo 배포 정보
type DeploymentInfo struct {
	Replicas    int    `json:"replicas"`
	RunningPods int    `json:"runningPods"`
	Image       string `json:"image"`
	Uptime      string `json:"uptime"`
}

// ServiceMapStats 서비스맵 통계 (Figma 상태 패널)
type ServiceMapStats struct {
	TotalNodes              int      `json:"totalNodes"`
	TotalEdges              int      `json:"totalEdges"`
	HealthyNodes            int      `json:"healthyNodes"`
	ErrorNodes              int      `json:"errorNodes"`
	WarningNodes            int      `json:"warningNodes"`
	Protocols               []string `json:"protocols"`
	TotalClusters           int      `json:"totalClusters"`
	CrossClusterConnections int      `json:"crossClusterConnections"`
}

// ResourceUtilization 리소스 사용률 (Figma 0004.png 패널)
type ResourceUtilization struct {
	PodCount          PodCount          `json:"podCount"`
	CPUUsage          ResourceUsage     `json:"cpuUsage"`
	MemoryUsage       ResourceUsage     `json:"memoryUsage"`
	DiskIO            DiskIOMetrics     `json:"diskIO"`
	BandwidthUsage    BandwidthMetrics  `json:"bandwidthUsage"`
	NetworkThroughput ThroughputMetrics `json:"networkThroughput"`
}

// PodCount 파드 수 정보
type PodCount struct {
	Running int `json:"running"`
	Total   int `json:"total"`
	Desired int `json:"desired"`
}

// ResourceUsage 리소스 사용량
type ResourceUsage struct {
	Current    string  `json:"current"`
	Limit      string  `json:"limit"`
	Percentage float64 `json:"percentage"`
	Trend      string  `json:"trend"`
}

// DiskIOMetrics 디스크 I/O 메트릭
type DiskIOMetrics struct {
	Read  string `json:"read"`
	Write string `json:"write"`
	IOPS  int    `json:"iops"`
}

// BandwidthMetrics 대역폭 메트릭
type BandwidthMetrics struct {
	Inbound  string `json:"inbound"`
	Outbound string `json:"outbound"`
	Total    string `json:"total"`
}

// ThroughputMetrics 처리량 메트릭
type ThroughputMetrics struct {
	Current string `json:"current"`
	Peak    string `json:"peak"`
	Average string `json:"average"`
}

// ServiceMapConnectionInfo 서비스맵 연결 정보
type ServiceMapConnectionInfo struct {
	TotalConnections    int            `json:"totalConnections"`
	InboundConnections  int            `json:"inboundConnections"`
	OutboundConnections int            `json:"outboundConnections"`
	Protocols           []string       `json:"protocols"`
	Ports               []int          `json:"ports"`
	Endpoints           []EndpointInfo `json:"endpoints"`
}

// EndpointInfo API 엔드포인트 정보
type EndpointInfo struct {
	Path         string `json:"path"`
	Method       string `json:"method"`
	RequestCount int64  `json:"requestCount"`
	LatencyP95   int    `json:"latencyP95"`
}

// HealthCheckStatus 헬스 체크 상태
type HealthCheckStatus struct {
	Readiness string `json:"readiness"`
	Liveness  string `json:"liveness"`
	Startup   string `json:"startup"`
}

// ServiceEvent 서비스 이벤트
type ServiceEvent struct {
	Time    time.Time `json:"time"`
	Type    string    `json:"type"`
	Reason  string    `json:"reason"`
	Message string    `json:"message"`
}

type PeerMetrics struct {
	RequestCount int     `json:"requestCount"`
	RequestRate  float64 `json:"requestRate"`
	LatencyP50   int     `json:"latencyP50"`
	LatencyP95   int     `json:"latencyP95"`
	LatencyP99   int     `json:"latencyP99"`
	ErrorRate    float64 `json:"errorRate"`
	ErrorCount   int     `json:"errorCount"`
	Bandwidth    string  `json:"bandwidth"`
	Throughput   string  `json:"throughput"`
}

// TrendInfo 트렌드 정보
type TrendInfo struct {
	RequestTrend string `json:"requestTrend"`
	LatencyTrend string `json:"latencyTrend"`
	ErrorTrend   string `json:"errorTrend"`
}

// ConnectionDetailResponse 연결 상세 응답 (Figma 0005.png)
type ConnectionDetailResponse struct {
	ConnectionID   string            `json:"connectionId"`
	Source         ServiceEndpoint   `json:"source"`
	Target         ServiceEndpoint   `json:"target"`
	Protocol       string            `json:"protocol"`
	ConnectionType string            `json:"connectionType"`
	Status         string            `json:"status"`
	Metrics        ConnectionMetrics `json:"metrics"`
	Timeline       []TimelineEntry   `json:"timeline"`
	RecentTraces   []TraceInfo       `json:"recentTraces"`
}

// ServiceEndpoint 서비스 엔드포인트
type ServiceEndpoint struct {
	ServiceName string `json:"serviceName"`
	Cluster     string `json:"cluster"`
	Namespace   string `json:"namespace"`
	DisplayName string `json:"displayName"`
}

// ConnectionMetrics 연결 메트릭
type ConnectionMetrics struct {
	RequestCount        int                   `json:"requestCount"`
	RequestRate         float64               `json:"requestRate"`
	ErrorCount          int                   `json:"errorCount"`
	ErrorRate           float64               `json:"errorRate"`
	LatencyStats        LatencyStats          `json:"latencyStats"`
	LatencyDistribution []LatencyDistribution `json:"latencyDistribution"`
	StatusCodes         map[string]int        `json:"statusCodes"`
	Bandwidth           BandwidthInfo         `json:"bandwidth"`
}

// LatencyStats 지연시간 통계
type LatencyStats struct {
	P50 int `json:"p50"`
	P90 int `json:"p90"`
	P95 int `json:"p95"`
	P99 int `json:"p99"`
	Max int `json:"max"`
}

// LatencyDistribution 지연시간 분포
type LatencyDistribution struct {
	Bucket     string  `json:"bucket"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// BandwidthInfo 대역폭 정보
type BandwidthInfo struct {
	Total    string `json:"total"`
	Inbound  string `json:"inbound"`
	Outbound string `json:"outbound"`
}

// TimelineEntry 타임라인 엔트리
type TimelineEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	RequestRate float64   `json:"requestRate"`
	ErrorRate   float64   `json:"errorRate"`
	LatencyP95  int       `json:"latencyP95"`
}

// TraceRequest 트레이스 요청
type TraceRequest struct {
	Cluster     string `json:"cluster" binding:"required"`     // 3-tier 식별자: 클러스터
	Namespace   string `json:"namespace" binding:"required"`   // 3-tier 식별자: 네임스페이스
	ServiceName string `json:"serviceName" binding:"required"` // 3-tier 식별자: 서비스명
	TimeRange   string `json:"timeRange" binding:"required"`   // 시간 범위
	Limit       int    `json:"limit,omitempty"`                // 결과 개수 제한
	Status      []int  `json:"status,omitempty"`               // HTTP 상태 코드 필터
	MinLatency  int    `json:"minLatency,omitempty"`           // 최소 지연시간
	MaxLatency  int    `json:"maxLatency,omitempty"`           // 최대 지연시간
	Operation   string `json:"operation,omitempty"`            // 오퍼레이션 필터
}

// TraceResponse 트레이스 응답 (Figma 0009.png)
type TraceResponse struct {
	ServiceName string       `json:"serviceName"`
	TimeRange   string       `json:"timeRange"`
	TotalCount  int          `json:"totalCount"`
	Summary     TraceSummary `json:"summary"`
	Traces      []TraceInfo  `json:"traces"`
	Filters     TraceFilters `json:"filters"`
}

// TraceSummary 트레이스 요약
type TraceSummary struct {
	AvgLatency float64  `json:"avgLatency"`
	ErrorRate  float64  `json:"errorRate"`
	Throughput float64  `json:"throughput"`
	Protocols  []string `json:"protocols"`
}

// TraceInfo 트레이스 정보 (Figma 0009.png)
type TraceInfo struct {
	Time       string            `json:"time"`
	TraceID    string            `json:"traceId"`
	Workload   string            `json:"workload"`
	PathMethod string            `json:"pathMethod"`
	Operation  string            `json:"operation"`
	Status     int               `json:"status"`
	StatusText string            `json:"statusText"`
	Latency    int               `json:"latency"`
	Protocol   string            `json:"protocol"`
	Tags       map[string]string `json:"tags"`
	Spans      int               `json:"spans"`
	Errors     int               `json:"errors"`
}

// TraceFilters 트레이스 필터
type TraceFilters struct {
	AvailableOperations []string     `json:"availableOperations"`
	AvailableStatuses   []int        `json:"availableStatuses"`
	LatencyRange        LatencyRange `json:"latencyRange"`
}

// LatencyRange 지연시간 범위
type LatencyRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// FiltersResponse 필터 응답
type FiltersResponse struct {
	HealthStatus []FilterOption `json:"healthStatus"`
	Protocols    []FilterOption `json:"protocols"`
	Clusters     []FilterOption `json:"clusters"`
	Namespaces   []FilterOption `json:"namespaces"`
	Workloads    []FilterOption `json:"workloads"`
	ServiceTypes []FilterOption `json:"serviceTypes"`
}

// FilterOption 필터 옵션
type FilterOption struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	Count       int    `json:"count"`
	Enabled     bool   `json:"enabled"`
}

// RealtimeMetricsRequest와 RealtimeMetricsResponse 제거됨 - NodeMetrics로 통합

// ServiceMetrics 서비스 메트릭 (확장 가능한 구조체 - 향후 사용 가능)
type ServiceMetrics struct {
	ServiceName string          `json:"serviceName"`
	Cluster     string          `json:"cluster"`
	Namespace   string          `json:"namespace"`
	Metrics     RealtimeMetrics `json:"metrics"`
	Alerts      []AlertInfo     `json:"alerts"`
}

// RealtimeMetrics 실시간 메트릭 (NodeMetrics와 유사하지만 더 상세한 정보)
type RealtimeMetrics struct {
	RequestRate       float64 `json:"requestRate"`
	LatencyP95        int     `json:"latencyP95"`
	ErrorRate         float64 `json:"errorRate"`
	CPUUsage          float64 `json:"cpuUsage"`
	MemoryUsage       float64 `json:"memoryUsage"`
	ActiveConnections int     `json:"activeConnections"`
}

// AlertInfo 알림 정보 (향후 알림 기능에 활용 가능)
type AlertInfo struct {
	Level     string  `json:"level"`
	Message   string  `json:"message"`
	Threshold float64 `json:"threshold"`
	Current   float64 `json:"current"`
}

// GlobalStats 전역 통계 (향후 대시보드 기능에 활용 가능)
type GlobalStats struct {
	TotalRequests  int64   `json:"totalRequests"`
	TotalErrors    int64   `json:"totalErrors"`
	AvgLatency     float64 `json:"avgLatency"`
	ActiveServices int     `json:"activeServices"`
}

// ServiceIdentifier 서비스 식별자 (캐싱 키 생성용)
type ServiceIdentifier struct {
	Cluster     string `json:"cluster"`
	Namespace   string `json:"namespace"`
	ServiceName string `json:"serviceName"`
	TimeRange   string `json:"timeRange"`
}

// ServiceData 통합 서비스 데이터 (캐싱 전략용)
type ServiceData struct {
	ServiceInfo     ServiceIdentifier `json:"serviceInfo"`
	REDMetrics      NodeMetrics       `json:"redMetrics"`      // RED 메트릭
	TopPeers        []PeerInfo        `json:"topPeers"`        // Top Peers 정보
	PeersByProtocol map[string]int    `json:"peersByProtocol"` // 프로토콜별 분포
	PeersByCluster  map[string]int    `json:"peersByCluster"`  // 클러스터별 분포
	DataTimestamp   time.Time         `json:"dataTimestamp"`   // 데이터 수집 시점
	CacheTTL        time.Duration     `json:"cacheTtl"`        // 캐시 TTL
}
