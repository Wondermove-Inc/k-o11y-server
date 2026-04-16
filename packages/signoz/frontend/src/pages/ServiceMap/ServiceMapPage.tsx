// ✅ MUST: React.memo() 적용으로 불필요한 리렌더링 방지
// ✅ MUST: useCallback, useMemo로 핸들러 함수 및 계산값 최적화
import React, { useCallback, useMemo, useState } from 'react';
import { ReactFlowProvider } from '@xyflow/react';
import NetworkTopology from './components/NetworkTopology';
import { 
  NetworkNode as NetworkNodeType,
  NetworkEdge as NetworkEdgeType,
  TimeRange,
  RefreshState,
  NetworkFilters,
} from './types';

const ServiceMapPage: React.FC = React.memo(() => {
  const [nodeCount, setNodeCount] = useState<number>(0);
  const [edgeCount, setEdgeCount] = useState<number>(0);


  const handleNodesChange = useCallback((nodes: NetworkNodeType[]) => {
    setNodeCount(nodes.length);
  }, []);

  const handleEdgesChange = useCallback((edges: NetworkEdgeType[]) => {
    setEdgeCount(edges.length);
  }, []);

  const initialTimeRange = useMemo((): TimeRange => ({
    start: new Date(Date.now() - 15 * 60 * 1000).toISOString(), // 15분 전
    end: new Date().toISOString(),   // 현재 시간
    preset: 'last_15m'
  }), []);

  const initialRefreshState = useMemo((): RefreshState => ({
    lastRefreshed: new Date(),
    isRefreshing: false,
    autoRefresh: {
      enabled: false,
      intervalSeconds: 30, // 30초
      intervalOptions: [10, 30, 60, 300]
    }
  }), []);

  const initialFilters = useMemo((): NetworkFilters => ({
    clusters: [],
    namespaces: [],
    protocols: [],
    connectionStatuses: ['Ok', 'Error'], // 기본적으로 모든 상태 체크
    workloads: [],
    showErrors: false,
    nodeTypes: []
  }), []);

  return (
    <div className="service-map-wrapper" style={{
      // 🎯 SigNoz 절대 위치 레이아웃 시스템
      position: 'relative', // 자식의 absolute positioning을 위한 positioning context
      flex: 1, // Signoz 레이아웃 내에서 남은 공간 채우기
      background: 'var(--bg-ink-400, #121317)', // SigNoz 다크 테마
      overflow: 'hidden',
      width: '100%',
      height: '100%'
    }}>

      {/* React Flow Context 제공 */}
      <ReactFlowProvider>
        {/* 메인 네트워크 토폴로지 */}
        <NetworkTopology
          timeRange={initialTimeRange}
          filters={initialFilters}
          refreshState={initialRefreshState}
          visualMode="basic"
          onNodesChange={handleNodesChange}
          onEdgesChange={handleEdgesChange}
        />
      </ReactFlowProvider>
    </div>
  );
});

ServiceMapPage.displayName = 'ServiceMapPage';

export default ServiceMapPage;