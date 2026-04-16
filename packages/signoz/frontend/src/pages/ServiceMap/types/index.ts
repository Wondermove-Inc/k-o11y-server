import { Edge, Node, EdgeProps as ReactFlowEdgeProps, NodeProps as ReactFlowNodeProps } from '@xyflow/react';

export interface Position {
  x: number;
  y: number;
}

export interface NodeMetrics {
  requestRate: number;   
  latencyP95: number; 
  errorRate: number; 
  totalRequests: number;
  totalErrors: number;  
}

export interface DeploymentInfo {
  replicas: number;
  runningPods: number;
  image: string; 
  uptime: string;
}

export interface NetworkNodeData {
  id: string;
  workloadName: string;
  namespace: string;
  cluster: string;
  issueCount: number;
  isExternal: number;
  type: string;
  status: string; 
}

export interface NetworkNode extends Node {
  id: string;
  type: 'networkNode';
  position: Position;
  data: NetworkNodeData & Record<string, unknown>;
  draggable: boolean;
  selectable: boolean;
}
export interface NetworkEdgeData {
  target: string;
  id: string;
  source: string;
  destination: string;
  srcRaw?: string;      // ✅ 원본 src 값 (trace 매칭용) - network_map_connections.src_raw
  destRaw?: string;     // ✅ 원본 dest 값 (trace 매칭용) - network_map_connections.dest_raw
  protocol: string;
  isError: boolean;
  isExternal: number;
}

export interface EdgeMetrics {
  requestCount: number;
  requestRate: number;
  latencyP95: number;
  errorRate: number;
  bandwidth: string;
}

export interface NetworkEdge extends Edge {
  id: string;
  source: string;
  target: string;
  type: 'networkEdge';
  data: NetworkEdgeData & Record<string, unknown>;
  markerEnd?: {
    type: 'arrow' | 'arrowclosed';
    width: number;
    height: number;
    color?: string;
  };
}

export interface NetworkTopologyData {
  nodes: NetworkNode[];
  edges: NetworkEdge[];
  timeRange: string;
}

export interface TimeRange {
  start: string;
  end: string;
  preset?: 'last_5m' | 'last_10m' | 'last_15m' | 'last_30m' | 'last_1h' | 'last_3h' | 'last_24h' | 'custom';
}

export interface TimeRangeOption {
  value: TimeRange['preset'];
  label: string;
  minutes: number; // 실제 시간 범위 (분 단위)
}

// 자동 새로고침 설정
export interface AutoRefreshConfig {
  enabled: boolean;
  intervalSeconds: number;
  intervalOptions: number[]; // [10, 30, 60, 300] = [10s, 30s, 1m, 5m]
}

// 새로고침 상태
export interface RefreshState {
  lastRefreshed: Date;
  isRefreshing: boolean;
  autoRefresh: AutoRefreshConfig;
}

// 필터 옵션 (Groundcover 스타일 - dependency-aware filtering)
export interface NetworkFilters {
  clusters: string[];
  namespaces: string[];
  protocols: string[];
  showErrors: boolean;
  nodeTypes: string[];
  connectionStatuses: string[]; // 연결 상태 필터 (Ok, Error) - Groundcover 스타일
  workloads: string[];       // 워크로드 필터
}


// 서비스 상세 정보
export interface ServiceDetails {
  service_name: string;
  namespace: string;
  cluster: string;
  service_type: string;
  endpoints: string[];
  ports: Array<{
    name: string;
    port: number;
    target_port: string;
    protocol: string;
  }>;
  labels: { [key: string]: string };
  annotations: { [key: string]: string };
  metrics: NodeMetrics;
  deployment: DeploymentInfo;
  recent_traces: Array<{
    trace_id: string;
    span_name: string;
    start_time: string;
    duration_ms: number;
    status: 'success' | 'error';
    service_name: string;
  }>;
}

// Top Peers 정보
export interface TopPeer {
  peer_service: string;
  peer_namespace: string;
  peer_cluster: string;
  connection_type: 'incoming' | 'outgoing';
  total_requests: number;
  error_rate_percent: number;
  avg_response_time_ms: number;
  last_seen: string;
}

// 레이아웃 타입 - ELK Layout만 사용
export type LayoutType = 'elk';

// 시각화 모드
export type VisualMode = 'basic' | 'advanced';

// ✅ MUST: NodeProps 타입을 모든 커스텀 노드에 적용
// React Flow 네이티브 NodeProps 확장
export interface CustomNetworkNodeProps extends ReactFlowNodeProps {
  data: NetworkNodeData & Record<string, unknown>;
  visualMode?: VisualMode;
}

// React Flow 네이티브 EdgeProps 확장  
export interface CustomNetworkEdgeProps extends ReactFlowEdgeProps {
  data: NetworkEdgeData & Record<string, unknown>;
  onLabelMouseEnter?: (edgeId: string) => void;
  onLabelMouseLeave?: () => void;
}

// 컴포넌트 Props 타입들 - React Flow 가이드라인 준수
export interface NetworkTopologyProps {
  timeRange: TimeRange;
  filters: NetworkFilters;
  visualMode: VisualMode;
  refreshState: RefreshState;
  onNodeClick?: (nodeData: NetworkNodeData) => void;
  onEdgeClick?: (edgeData: NetworkEdgeData) => void;
  onNodesChange?: (nodes: NetworkNode[]) => void;
  onEdgesChange?: (edges: NetworkEdge[]) => void;
  onTimeRangeChange?: (timeRange: TimeRange) => void;
  onRefreshStateChange?: (refreshState: RefreshState) => void;
}

export interface NetworkControlsProps {
  filters: NetworkFilters;
  visualMode: VisualMode;
  onFiltersChange: (filters: NetworkFilters) => void;
  onVisualModeChange: (mode: VisualMode) => void;
  onRefreshData: () => void; // API 호출을 위한 새로운 prop
  isLoading: boolean;
  // 필터 카운트 계산을 위한 실제 데이터
  filteredData: {
    nodes: NetworkNodeData[];
    edges: NetworkEdgeData[];
  };
  // 카운트 계산을 위한 전체 원본 데이터 (필터 상태와 무관)
  originalData: {
    nodes: NetworkNodeData[];
    edges: NetworkEdgeData[];
  };
}

export interface NodeMetrics {
  requestRate: number; 
  errorRate: number;
  latencyP95: number;
}

export interface PeerInfo {
  rank: number;
  workloadName: string;
  direction: string;
  requestRate: number;
  latencyP95: number;
  errorRate: number;
  totalRequests: number;
}

export interface WorkloadHoverRequest {
  cluster: string;
  namespace: string;
  workloadName: string;
  startTime: string;
  endTime: string;
}

export interface WorkloadHoverResponse {
  workloadName: string;
  cluster: string;
  namespace: string;
  metrics: NodeMetrics | null;
  topPeers: PeerInfo[] | null;
}

export interface WorkloadDetailRequest {
  cluster: string;
  namespace: string;
  workloadName: string;
  startTime: string;
  endTime: string;
}

export interface WorkloadMetricValue {
  timestamp: string;
  value: number;
}

export interface WorkloadMetric {
  queryName: string;
  labels: { [key: string]: string };
  values: WorkloadMetricValue[];
}

export interface WorkloadDetailResponse {
  workloadName: string;
  cluster: string;
  namespace: string;
  kind: string;
  replicas: number;
  runningPods: number;
  cpuMetrics: WorkloadMetric[];
  memoryMetrics: WorkloadMetric[];
  networkIoMetrics: WorkloadMetric[];
  networkErrorMetrics: WorkloadMetric[];
}

export interface EdgeTraceDetailRequest {
  edgeId: string;
  source: string;
  destination: string;
  sourceRaw?: string;       // ✅ 원본 src 값 (trace 매칭용) - network_map_connections.src_raw
  destinationRaw?: string;  // ✅ 원본 dest 값 (trace 매칭용) - network_map_connections.dest_raw
  startTime: string;
  endTime: string;
  isClientExternal?: number;  // 0 = 내부, 1 = 외부
  isServerExternal?: number;  // 0 = 내부, 1 = 외부
}

export interface EdgeTraceDetailResponse {
  srcWorkload: string;
  srcNamespace: string;
  destWorkload: string;
  destNamespace: string;
  protocol: string;
  topSlowRequests: TopSlowRequest[];
  recentErrors: RecentError[];
  requests: Requests[];
  cursor: CursorMeta;
}

export interface TopSlowRequest {
  timestamp: string;
  traceId: string;
  path: string;
  method: string;
  status: number;
  isError: boolean;
  latency: number;
}

export interface RecentError {
  timestamp: string;
  traceId: string;
  path: string;
  method: string;
  status: number;
  isError: boolean;
  latency: number;
}

export interface Requests {
  timestamp: string;
  traceId: string;
  connection: string;
  path: string;
  method: string;
  status: number;
  isError: boolean;
  latency: number;
  protocol: string;
}

export interface CursorMeta {
  hasMore: boolean;
  nextCursor: string;
  pageSize: number;
  total: number;
}