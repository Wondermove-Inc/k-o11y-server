// NetworkMap 컴포넌트 export
export { default as NetworkTopology } from './components/NetworkTopology';
export { default as NetworkNodeComponent } from './components/NetworkNode';
export { default as NetworkEdgeComponent } from './components/NetworkEdge';
export { default as NetworkControls } from './components/NetworkControls';

// 타입 export
export type {
  NetworkNode as NetworkNodeData,
  NetworkEdge as NetworkEdgeData,
  NetworkTopologyData,
  NetworkFilters,
  NetworkTopologyProps,
  NetworkControlsProps,
  TimeRange,
  RefreshState,
  AutoRefreshConfig,
  TimeRangeOption,
  LayoutType,
  VisualMode,
  ServiceDetails,
  TopPeer,
  Position,
  NodeMetrics,
  DeploymentInfo,
} from './types';

// ✅ MUST: ReactFlowProvider로 래핑
// 에러 바운더리 및 접근성 준수를 위한 메인 NetworkMap 컴포넌트
import React, { useState, useMemo, ErrorInfo, Component, ReactNode } from 'react';
import { ReactFlowProvider } from '@xyflow/react';
import NetworkTopology from './components/NetworkTopology';
import { 
  NetworkFilters, 
  TimeRange, 
  VisualMode, 
  NetworkNodeData, 
  NetworkEdgeData,
  NetworkNode as NetworkNodeType,
  NetworkEdge as NetworkEdgeType,
  RefreshState
} from './types';

// ✅ MUST: 에러 바운더리 구현
interface ErrorBoundaryState {
  hasError: boolean;
  error?: Error;
}

class NetworkMapErrorBoundary extends Component<
  { children: ReactNode },
  ErrorBoundaryState
> {
  constructor(props: { children: ReactNode }) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('NetworkMap Error:', error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div 
          className="network-map-error" 
          role="alert"
          aria-live="assertive"
        >
          <h3>Network Map Error</h3>
          <p>Failed to render network topology. Please try refreshing the page.</p>
          <details>
            <summary>Error Details</summary>
            <pre>{this.state.error?.message}</pre>
          </details>
          <button
            onClick={() => this.setState({ hasError: false, error: undefined })}
            className="retry-button"
          >
            Retry
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}

interface NetworkMapProps {
  className?: string;
  onNodeClick?: (node: NetworkNodeData) => void;
  onEdgeClick?: (edge: NetworkEdgeData) => void;
  onNodesChange?: (nodes: NetworkNodeType[]) => void;
  onEdgesChange?: (edges: NetworkEdgeType[]) => void;
  initialTimeRange?: TimeRange;
  initialRefreshState?: RefreshState;
  initialFilters?: Partial<NetworkFilters>;
  initialVisualMode?: VisualMode;
}

const NetworkMap: React.FC<NetworkMapProps> = React.memo(({ 
  className, 
  onNodeClick,
  onEdgeClick,
  onNodesChange,
  onEdgesChange,
  initialTimeRange,
  initialRefreshState,
  initialFilters,
  initialVisualMode = 'basic'
}) => {
  // ✅ MUST: useMemo로 계산값 최적화 - 기본 상태값
  const defaultTimeRange = useMemo((): TimeRange => 
    initialTimeRange || {
      start: new Date(Date.now() - 5 * 60 * 1000).toISOString(), // 5분 전
      end: new Date().toISOString(),
      preset: 'last_5m'
    }, [initialTimeRange]);

  const defaultFilters = useMemo((): NetworkFilters => ({
    clusters: [],
    namespaces: [],
    protocols: [],
    showErrors: false,
    nodeTypes: [],
    connectionStatuses: ['Ok', 'Error'], // 초기에 모든 연결 상태 선택
    workloads: [],
    ...initialFilters
  }), [initialFilters]);

  const defaultRefreshState = useMemo((): RefreshState => 
    initialRefreshState || {
      lastRefreshed: new Date(),
      isRefreshing: false,
      autoRefresh: {
        enabled: false,
        intervalSeconds: 30,
        intervalOptions: [10, 30, 60, 300]
      }
    }, [initialRefreshState]);

  const [timeRange] = useState<TimeRange>(defaultTimeRange);
  const [filters] = useState<NetworkFilters>(defaultFilters);
  const [refreshState] = useState<RefreshState>(defaultRefreshState);
  const [visualMode] = useState<VisualMode>(initialVisualMode);

  return (
    <NetworkMapErrorBoundary>
      {/* ✅ MUST: ReactFlowProvider로 래핑 */}
      <ReactFlowProvider>
        <div 
          className={`network-map ${className || ''}`}
          role="application"
          aria-label="Network topology visualization"
        >
          <NetworkTopology
            timeRange={timeRange}
            filters={filters}
            refreshState={refreshState}
            visualMode={visualMode}
            onNodeClick={onNodeClick}
            onEdgeClick={onEdgeClick}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
          />
        </div>
      </ReactFlowProvider>
    </NetworkMapErrorBoundary>
  );
});

// ✅ MUST: React.memo() displayName 설정
NetworkMap.displayName = 'NetworkMap';

export default NetworkMap;