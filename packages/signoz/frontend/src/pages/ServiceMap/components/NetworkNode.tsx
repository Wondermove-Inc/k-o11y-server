// ✅ MUST: React.memo() 적용으로 불필요한 리렌더링 방지
// ✅ MUST: NodeProps 타입을 모든 커스텀 노드에 적용
// ✅ MUST: CSS 클래스 사용 (CSS-in-JS 대신)
import React, { memo, useCallback, useMemo, useState, useEffect } from 'react';
import { createPortal } from 'react-dom';
import { Handle, Position, useReactFlow } from '@xyflow/react';
import { CustomNetworkNodeProps, NetworkNodeData, WorkloadHoverResponse } from '../types';
import { circleCheckGreen, circleWarningRed, world } from '../../../assets/ServiceMapIcons';
import { getWorkloadHoverInfo } from 'api/servicemap';
import './NetworkNode.css';

const NetworkNode: React.FC<CustomNetworkNodeProps> = memo(({ data, selected, id }) => {
  const nodeData = data.data as NetworkNodeData;
  const formatLatency = (latency: number): string => {
    if (latency >= 1000) {
      return `${(latency / 1000).toFixed(1)} s`;
    } else if (latency >= 1) {
      return `${latency.toFixed(1)} ms`;
    } else {
      return `${(latency * 1000).toFixed(1)} μs`;
    }
  };
  const formatRequestRate = (rate: number): string => {
    if (rate === 0) return '0 /s';
    if (rate < 0.1) return '< 0.1/s';
    if (rate < 1) return `${rate.toFixed(2)}/s`;
    if (rate >= 1000) return `${(rate / 1000).toFixed(1)}K/s`;
    return `${rate.toFixed(1)}/s`;
  };
  const [isHovered, setIsHovered] = useState(false);
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 });
  const [hideTimeout, setHideTimeout] = useState<NodeJS.Timeout | null>(null);
  const [hoverData, setHoverData] = useState<WorkloadHoverResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  
  const { getViewport } = useReactFlow();
  
  const visualMode = (data as any).visualMode || 'basic';
  
  const handleClick = useCallback(() => {
  }, [nodeData.id]);

  const handleMouseEnter = useCallback(async (e: React.MouseEvent) => {
    if (hideTimeout) {
      clearTimeout(hideTimeout);
      setHideTimeout(null);
    }
    
    const nodeElement = e.currentTarget as HTMLElement;
    const nodeRect = nodeElement.getBoundingClientRect();
    
    const tooltipX = nodeRect.right + 10;
    const tooltipY = nodeRect.top;
    
    setMousePosition({ x: tooltipX, y: tooltipY });
    setIsHovered(true);
    
    // API 호출해서 hover 데이터 가져오기
    if (nodeData.cluster && nodeData.namespace && nodeData.workloadName) {
      setIsLoading(true);
      try {
        const timeRange = (data as any).timeRange;
        const startTime = timeRange?.start || new Date(Date.now() - 15 * 60 * 1000).toISOString();
        const endTime = timeRange?.end || new Date().toISOString();
        
        const hoverInfo = await getWorkloadHoverInfo({
          cluster: nodeData.cluster,
          namespace: nodeData.namespace,
          workloadName: nodeData.workloadName,
          startTime,
          endTime
        });
        setHoverData(hoverInfo);
      } catch (error) {
        console.error('Failed to fetch hover data:', error);
        setHoverData(null);
      } finally {
        setIsLoading(false);
      }
    }
  }, [hideTimeout, getViewport, nodeData.workloadName, nodeData.cluster, nodeData.namespace, id, (data as any).timeRange?.start, (data as any).timeRange?.end]);

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    // 마우스 이동해도 위치 고정 - 노드 기준 위치 유지
    if (isHovered) {
      const nodeElement = e.currentTarget as HTMLElement;
      const nodeRect = nodeElement.getBoundingClientRect();
      
      // 동일한 화면 좌표 계산 로직
      const tooltipX = nodeRect.right + 10; // 노드 오른쪽 끝에서 10px 더 (20px에서 줄임)
      const tooltipY = nodeRect.top;        // 노드 상단과 같은 높이
      
      setMousePosition({ x: tooltipX, y: tooltipY });
    }
  }, [isHovered]);

  const handleMouseLeave = useCallback(() => {
    // 약간의 지연을 두고 툴팁 숨기기
    const timeout = setTimeout(() => {
      setIsHovered(false);
    }, 150); // 150ms 지연
    setHideTimeout(timeout);
  }, []);

  const nodeColor = useMemo(() => {
    return '#e9ecef'; 
  }, []);

  const statusIconSrc = useMemo(() => {
    if (nodeData.isExternal) return world;
    
    const status = nodeData.status?.toLowerCase();
    if (status === 'error' || (nodeData as any).hasErrorConnections) {
      return circleWarningRed; 
    }
    
    if (status === 'ok') {
      return circleCheckGreen; 
    }
    
    return circleCheckGreen;
  }, [nodeData.status, (nodeData as any).hasErrorConnections, nodeData.isExternal]);

  const isExternalService = useMemo(() => nodeData.isExternal, [nodeData.isExternal]);

  // ✅ MUST: useMemo로 계산값 최적화 - CSS 클래스명 (선택 상태 포함)
  const nodeClassName = useMemo(() => {
    const baseClass = 'network-node';
    const modifiers = [
      nodeData.isExternal ? 'external' : 'internal',
      nodeData.type,
      nodeData.status,
      visualMode === 'advanced' ? 'advanced-mode' : 'basic-mode',
      selected ? 'selected' : null, // 선택 상태 클래스 추가
    ].filter(Boolean);
    
    const className = [baseClass, ...modifiers].join(' ');
    
    // 선택된 노드의 클래스 확인
    
    return className;
  }, [nodeData.isExternal, nodeData.type, nodeData.status, visualMode, selected, nodeData.workloadName]);

  // ✅ NEVER: 렌더 함수 내 새 객체/배열 생성 - 스타일 객체 최적화
  const nodeStyle = useMemo(() => ({
    background: 'linear-gradient(to top, rgba(255, 255, 255, 0.01), rgba(255, 255, 255, 0.08))',
    // border는 CSS 클래스에서 처리하도록 제거
  }), []);

  const tooltipPosition = useMemo(() => {
    const tooltipWidth = 240;
    const tooltipHeight = 400;
    
    let finalLeft, finalTop;
    
    // 오른쪽에 툴팁을 배치할 공간이 있는지 확인
    // mousePosition.x는 이미 nodeRect.right + 10 값
    const rightSpaceNeeded = mousePosition.x + tooltipWidth;
    const hasRightSpace = rightSpaceNeeded <= window.innerWidth - 20;
    
    if (hasRightSpace) {
      // 오른쪽에 배치 (원래 계산된 위치 사용)
      finalLeft = mousePosition.x;
    } else {
      // 왼쪽에 배치 - nodeRect.right에서 역산하여 nodeRect.left 계산
      const nodeRight = mousePosition.x - 10; // mousePosition.x = nodeRect.right + 10이므로
      const nodeWidth = 190; //
      const nodeLeft = nodeRight - nodeWidth;
      
      finalLeft = nodeLeft - tooltipWidth - 10;
      finalLeft = Math.max(20, finalLeft); // 화면 왼쪽 끝을 넘지 않도록
    }
    
    // Y 위치는 원래 계산된 위치 사용 (nodeRect.top)
    finalTop = mousePosition.y;
    
    // 상하 경계 조정
    const maxTop = window.innerHeight - tooltipHeight - 20;
    finalTop = Math.min(Math.max(20, finalTop), maxTop);
    
    
    return { left: finalLeft, top: finalTop };
  }, [mousePosition.x, mousePosition.y, nodeData.workloadName]);
  
  return (
    <div 
      className={nodeClassName}
      onClick={handleClick}
      onMouseEnter={handleMouseEnter}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      style={nodeStyle}
      role="button"
      tabIndex={0}
      aria-label={`Network node: ${nodeData.workloadName} in ${nodeData.namespace}`}
    >
      {/* ✅ MUST: 필수 속성 id, type, position 포함 - React Flow 연결 핸들 (Groundcover 스타일: 좌우 연결) */}
      <Handle
        type="target"
        position={Position.Left}
        className="node-handle node-handle-target"
        isConnectable={false}
        id={`${id}-target`}
      />
      <Handle
        type="source"
        position={Position.Right}
        className="node-handle node-handle-source"
        isConnectable={false}
        id={`${id}-source`}
      />

      {/* 새로운 2층 노드 구조 */}
      <div className="node-content">
        {/* 워크로드 이름 (메인 텍스트) */}
        <div className="workload-name" title={nodeData.workloadName || 'No workload name'}>
          {nodeData.workloadName || 'Unknown Workload'}
        </div>

        {/* 네임스페이스 박스 (작은 회색 박스) */}
        <div className="namespace-box" title={`Namespace: ${nodeData.namespace || 'unknown'}`}>
          {nodeData.namespace || 'unknown'}
        </div>

        {/* 상태 아이콘 (우측 상단) */}
        <div className={`status-icon-container ${
          isExternalService ? 'external' : 
          nodeData.status?.toLowerCase() === 'error' || (nodeData as any).hasErrorConnections ? 'error' : 'healthy'
        }`}>
          {statusIconSrc ? (
            <img 
              src={statusIconSrc} 
              alt={
                isExternalService 
                  ? 'External Service'
                  : nodeData.status?.toLowerCase() === 'error' 
                  ? 'Error' 
                  : 'Healthy'
              }
              className="status-icon"
              title={
                isExternalService
                  ? 'External Service'
                  : nodeData.status?.toLowerCase() === 'error' 
                  ? `Error - Issues: ${nodeData.issueCount}` 
                  : nodeData.status?.toLowerCase() === 'ok'
                  ? 'OK - Healthy'
                  : 'Healthy'
              }
            />
          ) : null}
        </div>
      </div>

      {/* 상세 모드 - 구분선과 RED 메트릭 */}
      {visualMode === 'advanced' && (
        <div className="advanced-metrics-container">
          {/* 회색 구분선 */}
          <div className="metrics-divider"></div>
        </div>
      )}

      {/* 기본 모드 호버 툴팁 - Portal 방식으로 document.body에 직접 렌더링 */}
      {visualMode === 'basic' && isHovered && createPortal(
        <div 
          className="basic-hover-tooltip basic-hover-tooltip-portal"
          style={{
            position: 'fixed',
            left: `${tooltipPosition.left}px`,
            top: `${tooltipPosition.top}px`,
            transform: 'none',
            zIndex: 999999,
            pointerEvents: 'none'
          }}
        >
          <div className="basic-metrics-section">
            <div className="basic-metric-row">
              <span className="basic-metric-label">Req/s</span>
              <span className="basic-metric-value">
                {isLoading ? '...' : (hoverData?.metrics?.requestRate !== undefined ? formatRequestRate(hoverData.metrics.requestRate) : 'N/A')}
              </span>
            </div>
            <div className="basic-metric-row">
              <span className="basic-metric-label">Latency p95</span>
              <span className="basic-metric-value">
                {isLoading ? '...' : (hoverData?.metrics?.latencyP95 !== undefined ? formatLatency(hoverData.metrics.latencyP95) : 'N/A')}
              </span>
            </div>
            <div className="basic-metric-row">
              <span className="basic-metric-label">Error Rate</span>
              <span className="basic-metric-value">
                {isLoading ? '...' : (hoverData?.metrics?.errorRate !== undefined ? `${hoverData.metrics.errorRate.toFixed(1)}%` : 'N/A')}
              </span>
            </div>
          </div>
          
          <div className="basic-connections-section">
            <div className="basic-section-title">상대 노드 Top5</div>
            <div className="basic-connections-table">
              <div className="basic-table-header">
                <span className="basic-header-rank">Rank</span>
                <span className="basic-header-name">Name</span>
                <span className="basic-header-reqs">Reqs</span>
                <span className="basic-header-p95">p95</span>
              </div>
              {isLoading ? (
                <div className="basic-table-row">
                  <span className="basic-rank-number">...</span>
                  <span className="basic-service-name">Loading...</span>
                  <span className="basic-req-value">...</span>
                  <span className="basic-p95-value">...</span>
                </div>
              ) : (
                hoverData?.topPeers?.slice(0, 5).map((peer, index) => (
                  <div key={`${peer.rank}-${peer.workloadName}-${peer.direction}`} className="basic-table-row">
                    <span className="basic-rank-number">{String(peer.rank).padStart(2, '0')}</span>
                    <span className="basic-service-name">{peer.workloadName}</span>
                    <span className="basic-req-value">{peer.totalRequests?.toLocaleString() || 'N/A'}</span>
                    <span className="basic-p95-value">{peer.latencyP95 !== undefined ? formatLatency(peer.latencyP95) : 'N/A'}</span>
                  </div>
                )) || (
                  <div className="basic-table-row">
                    <span className="basic-rank-number">--</span>
                    <span className="basic-service-name">No data available</span>
                    <span className="basic-req-value">--</span>
                    <span className="basic-p95-value">--</span>
                  </div>
                )
              )}
            </div>
          </div>
        </div>,
        document.body // Portal로 document.body에 직접 렌더링
      )}

      {visualMode === 'advanced' && isHovered && (
        <div className="top-connections-tooltip">
          <div className="tooltip-header">
            <h4>상대 노드 Top5</h4>
          </div>
          <div className="connections-list">
            {/* 컬럼 헤더 */}
            <div className="connection-header">
              <span className="header-rank">Rank</span>
              <span className="header-name">Name</span>
              <span className="header-metric">Reqs</span>
              <span className="header-metric">p95</span>
            </div>
            {isLoading ? (
              <div className="connection-item">
                <span className="rank">...</span>
                <span className="service-name">Loading...</span>
                <span className="conn-metric">...</span>
                <span className="conn-metric">...</span>
              </div>
            ) : (
              hoverData?.topPeers?.slice(0, 5).map((peer, index) => (
                <div key={`${peer.rank}-${peer.workloadName}-${peer.direction}`} className="connection-item">
                  <span className="rank">{String(peer.rank).padStart(2, '0')}</span>
                  <span className="service-name">{peer.workloadName}</span>
                  <span className="conn-metric">{peer.totalRequests?.toLocaleString() || 'N/A'}</span>
                  <span className="conn-metric">{peer.latencyP95 ? `${peer.latencyP95.toFixed(0)}ms` : 'N/A'}</span>
                </div>
              )) || (
                <div className="connection-item">
                  <span className="rank">--</span>
                  <span className="service-name">No data available</span>
                  <span className="conn-metric">--</span>
                  <span className="conn-metric">--</span>
                </div>
              )
            )}
          </div>
        </div>
      )}
    </div>
  );
});

// ✅ MUST: React.memo() displayName 설정
NetworkNode.displayName = 'NetworkNode';

export default NetworkNode;