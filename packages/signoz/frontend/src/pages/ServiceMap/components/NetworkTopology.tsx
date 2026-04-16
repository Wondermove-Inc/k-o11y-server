import React, { useState, useCallback, useMemo, useRef, useEffect, startTransition } from 'react';
import {
  ReactFlow,
  Node,
  Edge,
  useNodesState,
  useEdgesState,
  addEdge,
  Connection,
  Controls,
  MiniMap,
  Background,
  BackgroundVariant,
  MarkerType,
  ReactFlowInstance,
  ConnectionLineType,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import NetworkNode from './NetworkNode';
import NetworkEdge from './NetworkEdge';
import NetworkControls from './NetworkControls';
import NetworkHeader from './NetworkHeader';
import NetworkEdgeDetailPanel from './NetworkEdgeDetailPanel';
import NetworkMapLoader from './NetworkMapLoader';
import MetricChart from './MetricChart';
import TraceDetailSidePanel from './TraceDetailSidePanel';
import { 
  NetworkTopologyProps, 
  NetworkNodeData,
  NetworkEdgeData,
  NetworkNode as NetworkNodeType,
  NetworkEdge as NetworkEdgeType,
  NetworkFilters,
  LayoutType,
  VisualMode,
  Position,
  RefreshState,
  TimeRange,
  AutoRefreshConfig,
  NetworkTopologyData
} from '../types';
import './NetworkTopology.css';
import { getTopologyData, getWorkloadDetails } from 'api/servicemap';
import { useElkMultipleHandles } from '../hooks/useElkMultipleHandles';
import { useTranslation } from 'react-i18next';
import { useIsDarkMode } from 'hooks/useDarkMode';

const capitalizeFirstLetter = (str?: string): string => {
  if (!str) return 'Unknown';
  return str.charAt(0).toUpperCase() + str.slice(1);
};

const nodeTypes = {
  networkNode: NetworkNode,
};

const edgeTypes = {
  networkEdge: NetworkEdge,
};

const defaultFitViewOptions = {
  padding: 0.2,
  maxZoom: 1.2,
  duration: 800,
};

const NetworkTopology: React.FC<NetworkTopologyProps> = React.memo(({
  timeRange,
  filters: initialFilters,
  refreshState,
  visualMode: initialVisualMode,
  onNodeClick,
  onEdgeClick,
  onNodesChange,
  onEdgesChange,
}) => {
  const { t } = useTranslation('network_map');
  const isDarkMode = useIsDarkMode();
  const [filters, setFilters] = useState<NetworkFilters>(initialFilters);
  
  useEffect(() => {
    setFilters(initialFilters);
  }, [initialFilters]);
  const layout: LayoutType = 'elk';
  const [visualMode, setVisualMode] = useState<VisualMode>(initialVisualMode);
  const [currentTimeRange, setCurrentTimeRange] = useState<TimeRange>(timeRange);
  const [currentRefreshState, setCurrentRefreshState] = useState<RefreshState>(refreshState);
  const [loading, setLoading] = useState(false);
  const [apiError, setApiError] = useState<string | null>(null);
  const [realData, setRealData] = useState<NetworkTopologyData | null>(null);
  const [selectedNodeData, setSelectedNodeData] = useState<NetworkNodeData | null>(null);
  const [detailPanelOpen, setDetailPanelOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [sidebarWidth, setSidebarWidth] = useState(240);
  const isResizingRef = useRef(false);
  const [hoveredEdge, setHoveredEdge] = useState<string | null>(null);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);
  const [selectedEdgeData, setSelectedEdgeData] = useState<NetworkEdgeData | null>(null);
  const [selectedSourceNodeData, setSelectedSourceNodeData] = useState<NetworkNodeData | null>(null);
  const [selectedTargetNodeData, setSelectedTargetNodeData] = useState<NetworkNodeData | null>(null);
  const [edgeDetailPanelOpen, setEdgeDetailPanelOpen] = useState(false);
  const [traceDetailPanelOpen, setTraceDetailPanelOpen] = useState(false);
  const [selectedTraceId, setSelectedTraceId] = useState<string | null>(null);
  const [workloadDetailsData, setWorkloadDetailsData] = useState<any>(null);
  const [workloadDetailsLoading, setWorkloadDetailsLoading] = useState(false);

  useEffect(() => {
    setFilters(initialFilters);
  }, [initialFilters]);

  useEffect(() => {
    setVisualMode(initialVisualMode);
  }, [initialVisualMode]);

  useEffect(() => {
    setCurrentTimeRange(timeRange);
  }, [timeRange]);

  useEffect(() => {
    setCurrentRefreshState(refreshState);
  }, [refreshState]);

  
    const [nodes, setNodes, onNodesChangeInternal] = useNodesState<NetworkNodeType>([]);
  const [edges, setEdges, onEdgesChangeInternal] = useEdgesState<NetworkEdgeType>([]);
  
  const reactFlowRef = useRef<ReactFlowInstance | null>(null);
  
  const isApiCallingRef = useRef<boolean>(false);
  const lastCallParamsRef = useRef<string>('');
  
  const headerPanelCloseRef = useRef<(() => void) | null>(null);

  const fetchTopologyData = useCallback(async (source: string = 'unknown') => {
    
    const currentParams = JSON.stringify({ 
      timeRange: currentTimeRange, 
      filters: filters 
    });
    
    if (isApiCallingRef.current) {
      return;
    }
    
    const isRefreshAction = source === 'auto-refresh' || source === 'manual-refresh';
    if (!isRefreshAction && lastCallParamsRef.current === currentParams) {
      return;
    }
    
    isApiCallingRef.current = true;
    lastCallParamsRef.current = currentParams;
    
    try {
      setLoading(true);
      setApiError(null);
      
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      const data = await getTopologyData(currentTimeRange, filters);
      setRealData(data);
      
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : 'API 호출 중 오류가 발생했습니다.';
      console.error('[fetchTopologyData] API 호출 실패:', source, errorMsg, error);
      setApiError(errorMsg);
      setRealData(null);
    } finally {
      setLoading(false);
      isApiCallingRef.current = false;
    }
  }, [currentTimeRange, filters]);

  useEffect(() => {
    const timeoutId = setTimeout(() => {
      fetchTopologyData('timerange-changed');
    }, 300);
    
    return () => {
      clearTimeout(timeoutId);
    };
  }, [currentTimeRange]);

  const filteredData = useMemo(() => {
    // 데이터가 없으면 빈 배열 사용
    let sourceData: NetworkTopologyData;
    if (!realData || !Array.isArray(realData.nodes) || !Array.isArray(realData.edges)) {
      console.log('[ServiceMap] No data available');
      sourceData = {
        nodes: [],
        edges: []
      } as NetworkTopologyData;
    } else {
      sourceData = realData;
    }

    let filteredEdges = sourceData.edges as any[];
    
    const parseEdgeId = (edgeId: string) => {
      const parts = edgeId.split('##');
      if (parts.length === 3) {
        const [sourcePart, destPart, protocol] = parts;
        const [sourceNs, sourceWorkload] = sourcePart.split('$$');
        const [destNs, destWorkload] = destPart.split('$$');
        
        return {
          sourceNamespace: sourceNs,
          sourceWorkload: sourceWorkload,
          destNamespace: destNs, 
          destWorkload: destWorkload,
          protocol: protocol,
          namespaces: [sourceNs, destNs].filter(Boolean),
          workloads: [sourceWorkload, destWorkload].filter(Boolean)
        };
      }
      return null;
    };

    const hasConnectionFilter = filters.connectionStatuses.length > 0 && 
                               !(filters.connectionStatuses.includes('Ok') && filters.connectionStatuses.includes('Error'));
    
    if (hasConnectionFilter) {
      filteredEdges = filteredEdges.filter((edge: any) => {
        const edgeStatus = edge.data.isError === true ? 'Error' : 'Ok';
        return filters.connectionStatuses.includes(edgeStatus);
      });
    }
    
    if (filters.protocols.length > 0) {
      filteredEdges = filteredEdges.filter((edge: any) => {
        const parsed = parseEdgeId(edge.data.id);
        return parsed && filters.protocols.includes(parsed.protocol);
      });
    }
    
    if (filters.namespaces.length > 0) {
      filteredEdges = filteredEdges.filter((edge: any) => {
        const parsed = parseEdgeId(edge.data.id);
        return parsed && parsed.namespaces.some(ns => filters.namespaces.includes(ns));
      });
    }
    
    if (filters.workloads.length > 0) {
      filteredEdges = filteredEdges.filter((edge: any) => {
        const parsed = parseEdgeId(edge.data.id);
        if (!parsed) return false;
        
        return filters.workloads.some(selectedWorkload => 
          parsed.sourceWorkload === selectedWorkload || parsed.destWorkload === selectedWorkload
        );
      });
    }
    
    if (filters.clusters.length > 0) {
      filteredEdges = filteredEdges.filter((edge: any) => {
        const sourceCluster = edge.data.source.split('$$')[0];
        const destCluster = edge.data.destination.split('$$')[0];
        const clusters = [sourceCluster, destCluster].filter(Boolean);
        return clusters.some(cluster => filters.clusters.includes(cluster));
      });
    }
    
    const connectedNodeIds = new Set<string>();
    filteredEdges.forEach((edge: any) => {
      connectedNodeIds.add(edge.data.source);
      connectedNodeIds.add(edge.data.destination);
    });
    
    const userAppliedEdgeFilter = filters.protocols.length > 0 || hasConnectionFilter || 
                                 filters.namespaces.length > 0 || filters.workloads.length > 0 || 
                                 filters.clusters.length > 0;
    if (userAppliedEdgeFilter && filteredEdges.length === 0) {
      return { nodes: [], edges: [] };
    }
    
    
    const filteredNodes: any[] = [];
    for (const node of sourceData.nodes) {
      const nodeData = node as any;
      
      if (userAppliedEdgeFilter) {
        if (connectedNodeIds.has(nodeData.data.id)) {
          filteredNodes.push(node);
        }
      } else {
        let includeNode = true;
        
        if (filters.showErrors) {
          includeNode = includeNode && (nodeData.data.issueCount > 0 || nodeData.data.status === 'Error');
        }
        
        if (includeNode) {
          filteredNodes.push(node);
        }
      }
    }
    
    const finalNodeIds = new Set(filteredNodes.map((n: any) => n.data.id));
    const finalFilteredEdges = filteredEdges.filter((edge: any) => 
      finalNodeIds.has(edge.data.source) && finalNodeIds.has(edge.data.destination)
    );
    
    const hasAnyFilter = filters.clusters.length > 0 || 
                        filters.namespaces.length > 0 || 
                        filters.protocols.length > 0 ||
                        filters.workloads.length > 0 ||
                        filters.nodeTypes.length > 0 ||
                        hasConnectionFilter ||
                        filters.showErrors;
    
    if (!hasAnyFilter) {
      return {
        nodes: sourceData.nodes as any[],
        edges: sourceData.edges as any[]
      };
    }
    
    return {
      nodes: filteredNodes,
      edges: finalFilteredEdges
    };
  }, [realData, filters]);

  const {
    layoutedNodes: elkLayoutedNodes,
    isCalculating: isELKCalculating,
    error: elkError,
  } = useElkMultipleHandles(
    filteredData.nodes as NetworkNodeData[],
    filteredData.edges as NetworkEdgeData[],
    {
      autoLayout: true,
      debounceMs: 200,
    }
  );

  useEffect(() => {
    if (elkLayoutedNodes.length > 0 && !isELKCalculating && reactFlowRef.current) {
      setTimeout(() => {
        reactFlowRef.current?.fitView({
          padding: 0.1,
          maxZoom: 1,
          duration: 600,
        });
      }, 100);
    }
  }, [elkLayoutedNodes.length, isELKCalculating]);

  const getNodePosition = useCallback((nodeData: NetworkNodeData, index: number): Position => {
    const elkLayoutedNode = elkLayoutedNodes.find(n => n.id === nodeData.id);
    if (elkLayoutedNode && elkLayoutedNode.x !== undefined && elkLayoutedNode.y !== undefined) {
      return {
        x: elkLayoutedNode.x,
        y: elkLayoutedNode.y,
      };
    }
    
    if (isELKCalculating) {
      return {
        x: 400 + (index * 50), // 중앙에서 약간씩 분산
        y: 300 + (index * 30), 
      };
    }
    
    const angle = (index * 2 * Math.PI) / filteredData.nodes.length;
    const radius = Math.max(200, filteredData.nodes.length * 15); // 노드 수에 따라 반지름 조정
    const centerX = 500;
    const centerY = 400;
    
    const position = {
      x: centerX + radius * Math.cos(angle),
      y: centerY + radius * Math.sin(angle),
    };
    
    return position;
  }, [elkLayoutedNodes, isELKCalculating, filteredData.nodes.length]);

  const reactFlowNodes = useMemo((): NetworkNodeType[] => {
    const nodes = filteredData.nodes.map((nodeData: any, index: number): NetworkNodeType => {
      // Groundcover 스타일: 해당 노드와 연결된 에러 엣지가 있는지 확인
      const hasErrorConnections = filteredData.edges.some((edge: any) => 
        (edge.source === nodeData.id || edge.destination === nodeData.id) && edge.status === 'Error'
      );

      const node: NetworkNodeType = {
        id: nodeData.id,
        type: 'networkNode' as const,
        position: getNodePosition(nodeData, index),
        data: { 
          ...nodeData, 
          visualMode,
          hasErrorConnections, // Groundcover 스타일 에러 표시를 위한 플래그 추가
          timeRange: currentTimeRange
        },
        draggable: true,
        selectable: true,
      };
      
      return node;
    });
    
    return nodes;
  }, [filteredData.nodes, filteredData.edges, layout, getNodePosition, visualMode, currentTimeRange]);

  const nodesWithOpacity = useMemo((): NetworkNodeType[] => {
    if (hoveredEdge && !hoveredNode) {
      const connectedNodeIds = new Set<string>();
      const hoveredEdgeData = (filteredData.edges as any[]).find((edge: any) => edge.id === hoveredEdge);
      if (hoveredEdgeData) {
        connectedNodeIds.add(hoveredEdgeData.source);
        connectedNodeIds.add(hoveredEdgeData.target);
      }

      return nodes.map(node => {
        const isConnected = connectedNodeIds.has(node.id);


        return {
          ...node,
          className: isConnected ? `${node.className || ''} connected-highlight`.trim() : node.className,
          style: {
            ...node.style,
            opacity: isConnected ? 1 : 0.3,
          },
        };
      });
    }

    if (hoveredNode && !hoveredEdge) {
      const connectedNodeIds = new Set<string>();
      const hoveredNodeNormalized = hoveredNode ? hoveredNode.replace(/\$\$/g, '$') : '';
      connectedNodeIds.add(hoveredNodeNormalized);
      
      for (const edge of filteredData.edges) {
        const edgeSourceNormalized = edge.source ? edge.source.replace(/\$\$/g, '$') : '';
        const edgeTargetNormalized = edge.target ? edge.target.replace(/\$\$/g, '$') : '';
        
        if (edgeSourceNormalized === hoveredNodeNormalized) {
          connectedNodeIds.add(edgeTargetNormalized);
        }
        if (edgeTargetNormalized === hoveredNodeNormalized) {
          connectedNodeIds.add(edgeSourceNormalized);
        }
      }

      return nodes.map(node => {
        const nodeIdNormalized = node.id.replace(/\$\$/g, '$');
        const isConnected = connectedNodeIds.has(nodeIdNormalized);
        const finalClassName = isConnected ? `${node.className || ''} connected-highlight`.trim() : node.className;

        return {
          ...node,
          className: finalClassName,
          style: {
            ...node.style,
            opacity: isConnected ? 1 : 0.3,
          },
        };
      });
    }

    return nodes;
  }, [nodes, hoveredEdge, hoveredNode, filteredData.edges]);

  const reactFlowEdges = useMemo((): NetworkEdgeType[] => {
    return filteredData.edges.map((edgeData: any, index: number): NetworkEdgeType => ({
      id: `${edgeData.id}_${index}`,
      source: edgeData.source,
      target: edgeData.target,
      type: 'networkEdge',
      data: edgeData.data,
      animated: false,
      selectable: true,
      deletable: false,
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: 12,
        height: 12,
        color: edgeData.data?.isError ? '#dc3545' : (isDarkMode ? '#6c757d' : '#b8b8b8'),
      },
      style: {
        strokeWidth: 2,
        stroke: edgeData.data?.isError ? '#dc3545' : (isDarkMode ? '#6c757d' : '#b8b8b8'),
        transition: 'all 0.1s ease-in-out',
      },
    }));
  }, [filteredData.edges, isDarkMode]);

  const edgesWithOpacity = useMemo((): NetworkEdgeType[] => {
    if (hoveredEdge && !hoveredNode) {
      
      return edges.map(edge => {
        const originalEdgeId = (edge.data as any)?.id || edge.id;
        const isHovered = originalEdgeId === hoveredEdge;
        const edgeData = filteredData.edges.find((e: any) => e.id === originalEdgeId);
        const strokeColor = isHovered ? (edgeData?.isError ? '#dc3545' : '#00C02A') : (edge.style?.stroke || '#6c757d');
        
        
        return {
          ...edge,
          style: {
            ...edge.style,
            opacity: isHovered ? 1 : 0.25,
            stroke: strokeColor,
          },
        };
      });
    }

    if (hoveredNode && !hoveredEdge) {
      const connectedEdgeIds = new Set<string>();
      
      for (const edge of filteredData.edges) {
        if (edge.source === hoveredNode || edge.target === hoveredNode) {
          connectedEdgeIds.add(edge.id);
        }
      }


      return edges.map(edge => {
        const originalEdgeId = (edge.data as any)?.id || edge.id;
        const isConnected = connectedEdgeIds.has(originalEdgeId);
        const strokeColor = isConnected ? (edge.data.isError ? '#dc3545' : '#00C02A') : (edge.style?.stroke || '#6c757d');
        
        
        return {
          ...edge,
          style: {
            ...edge.style,
            opacity: isConnected ? 1 : 0.25,
            stroke: strokeColor,
          },
        };
      });
    }

    return edges;
  }, [edges, hoveredEdge, hoveredNode, filteredData.edges]);

  const onConnect = useCallback(
    (params: Connection) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  );

  const fetchWorkloadDetails = useCallback(async (nodeData: NetworkNodeData) => {
    if (!nodeData.cluster || !nodeData.namespace || !nodeData.workloadName) {
      return null;
    }

    setWorkloadDetailsLoading(true);
    try {
      const requestData = {
        cluster: nodeData.cluster,
        namespace: nodeData.namespace,
        workloadName: nodeData.workloadName,
        startTime: currentTimeRange.start,
        endTime: currentTimeRange.end
      };

      const data = await getWorkloadDetails(requestData);

      // startTransition으로 논블로킹 상태 업데이트
      startTransition(() => {
        setWorkloadDetailsData(data);
        setWorkloadDetailsLoading(false);
      });

      return data;
    } catch (error) {
      console.error('Failed to fetch workload details:', error);
      startTransition(() => {
        setWorkloadDetailsData(null);
        setWorkloadDetailsLoading(false);
      });
      return null;
    }
  }, [currentTimeRange]);

  // Detail panel이 열려있을 때 timeRange 변경 시에만 자동 refetch
  // 노드 클릭 시에는 handleNodeClick에서 직접 호출하므로 여기서는 timeRange 변경만 감지
  useEffect(() => {
    if (detailPanelOpen && selectedNodeData && !selectedNodeData.isExternal) {
      fetchWorkloadDetails(selectedNodeData);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentTimeRange.start, currentTimeRange.end]);

  const handleNodeClick = useCallback(async (event: React.MouseEvent, node: Node) => {
    if (node.data) {
      const nodeData = node.data.data as unknown as NetworkNodeData;
      setSelectedNodeData(nodeData);
      setDetailPanelOpen(true);
      setEdgeDetailPanelOpen(false);
      setSelectedEdgeData(null);
      
      if (!nodeData.isExternal) {
        await fetchWorkloadDetails(nodeData);
      }
    }
    
    if (onNodeClick && node.data) {
      onNodeClick(node.data as unknown as NetworkNodeData);
    }
  }, [onNodeClick, fetchWorkloadDetails]);

  const handleNodeMouseEnter = useCallback((event: React.MouseEvent, node: Node) => {
    setHoveredNode(node.id);
  }, []);

  const handleNodeMouseLeave = useCallback(() => {
    setHoveredNode(null);
  }, []);

  const handleEdgeClick = useCallback((event: React.MouseEvent, edge: Edge) => {
    if (edge.data) {
      const edgeData = edge.data as unknown as NetworkEdgeData;
      
      const sourceNodeData = filteredData.nodes.find(node => node.id === edge.source);
      const targetNodeData = filteredData.nodes.find(node => node.id === edge.target);
      
      setSelectedEdgeData(edgeData);
      setSelectedSourceNodeData(sourceNodeData);
      setSelectedTargetNodeData(targetNodeData);
      setEdgeDetailPanelOpen(true);
      setDetailPanelOpen(false);
      setSelectedNodeData(null);
    }
    
    if (onEdgeClick && edge.data) {
      onEdgeClick(edge.data as unknown as NetworkEdgeData);
    }
  }, [onEdgeClick]);

  const handlePaneClick = useCallback(() => {
    setDetailPanelOpen(false);
    setSelectedNodeData(null);
    setEdgeDetailPanelOpen(false);
    setSelectedEdgeData(null);
    
    setNodes((nds) => nds.map((node) => ({ ...node, selected: false })));
    setEdges((eds) => eds.map((edge) => ({ ...edge, selected: false })));
    
    if (headerPanelCloseRef.current) {
      headerPanelCloseRef.current();
    }
  }, [setNodes, setEdges]);

  const handleEdgeMouseEnter = useCallback((event: React.MouseEvent, edge: Edge) => {
    const originalEdgeId = (edge.data as any)?.id || edge.id;
    setHoveredEdge(originalEdgeId);
  }, []);

  const handleEdgeMouseLeave = useCallback(() => {
    setHoveredEdge(null);
  }, []);

  const handleNodeDragStop = useCallback(() => {
    
    setNodes((nds) => nds.map((node) => ({ ...node, selected: false })));
    setEdges((eds) => eds.map((edge) => ({ ...edge, selected: false })));
    
  }, [setNodes, setEdges]);

  const handleNodesChange = useCallback((changes: any) => {
    onNodesChangeInternal(changes);
    
    if (onNodesChange) {
      onNodesChange(nodes);
    }
  }, [onNodesChangeInternal, onNodesChange, nodes]);

  const handleEdgesChange = useCallback((changes: any) => {
    onEdgesChangeInternal(changes);
    if (onEdgesChange) {
      onEdgesChange(edges);
    }
  }, [onEdgesChangeInternal, onEdgesChange, edges]);

  const handleTimeRangeChange = useCallback((newTimeRange: TimeRange) => {
    setCurrentTimeRange(newTimeRange);
  }, []);

  // 수동 새로고침 핸들러
  const handleManualRefresh = useCallback(async (source: string = 'manual-refresh') => {
    setCurrentRefreshState(prev => ({
      ...prev,
      isRefreshing: true,
      lastRefreshed: new Date()
    }));
    
    try {
      await fetchTopologyData(source);
    } finally {
      setCurrentRefreshState(prev => ({
        ...prev,
        isRefreshing: false
      }));
    }
  }, [fetchTopologyData]);

  const handleQuickRefresh = useCallback(async () => {
    setCurrentRefreshState(prev => ({
      ...prev,
      isRefreshing: true,
      lastRefreshed: new Date()
    }));
    
    try {
      const now = new Date();
      const fifteenMinutesAgo = new Date(now.getTime() - 15 * 60 * 1000);
      
      const quickTimeRange: TimeRange = {
        start: fifteenMinutesAgo.toISOString(),
        end: now.toISOString(),
        preset: 'last_15m'
      };
      
      const emptyFilters: NetworkFilters = {
        clusters: [],
        namespaces: [],
        workloads: [],
        protocols: [],
        connectionStatuses: [],
        showErrors: true,
        nodeTypes: []
      };
      
      setLoading(true);
      setApiError(null);
      
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      const data = await getTopologyData(quickTimeRange, emptyFilters);
      
      setRealData(data);
      
      const resetFilters: NetworkFilters = {
        clusters: [],
        namespaces: [],
        workloads: [],
        protocols: [],
        connectionStatuses: [],
        showErrors: false,
        nodeTypes: []
      };
      
      setFilters(resetFilters);
      
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
      setApiError(errorMessage);
    } finally {
      setLoading(false);
      setCurrentRefreshState(prev => ({
        ...prev,
        isRefreshing: false
      }));
    }
  }, []);

  const handleAutoRefreshChange = useCallback((newConfig: AutoRefreshConfig) => {
    setCurrentRefreshState(prev => ({
      ...prev,
      autoRefresh: newConfig
    }));
  }, []);

  // 자동 새로고침 useEffect
  useEffect(() => {
    let intervalId: NodeJS.Timeout | null = null;

    if (currentRefreshState.autoRefresh.enabled) {
      // 자동 새로고침 타이머 시작
      intervalId = setInterval(() => {
        // 자동 새로고침 타이머 실행
        handleManualRefresh('auto-refresh');
      }, currentRefreshState.autoRefresh.intervalSeconds * 1000);
    }

    return () => {
      if (intervalId) {
        clearInterval(intervalId);
      }
    };
  }, [currentRefreshState.autoRefresh.enabled, currentRefreshState.autoRefresh.intervalSeconds, handleManualRefresh]);

  useEffect(() => {
    const updateInterval = setInterval(() => {
      setCurrentRefreshState(prev => ({ ...prev }));
    }, 1000);
    return () => clearInterval(updateInterval);
  }, []);

  const handleCloseDetailPanel = useCallback(() => {
    setDetailPanelOpen(false);
    setSelectedNodeData(null);
  }, []);

  const handleCloseEdgeDetailPanel = useCallback(() => {
    setEdgeDetailPanelOpen(false);
    setSelectedEdgeData(null);
    setSelectedSourceNodeData(null);
    setSelectedTargetNodeData(null);
    setTraceDetailPanelOpen(false);
    setSelectedTraceId(null);
  }, []);

  const handleOpenTraceDetailPanel = useCallback((traceId: string) => {
    setSelectedTraceId(traceId);
    setTraceDetailPanelOpen(true);
  }, []);

  const handleCloseTraceDetailPanel = useCallback(() => {
    setTraceDetailPanelOpen(false);
    setSelectedTraceId(null);
  }, []);

  const handleToggleSidebar = useCallback(() => {
    setSidebarCollapsed(prev => !prev);
  }, []);

  /** 사이드바 리사이즈 드래그 핸들러 */
  const handleResizeMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    isResizingRef.current = true;
    const startX = e.clientX;
    const startWidth = sidebarWidth;

    const handleMouseMove = (moveEvent: MouseEvent): void => {
      if (!isResizingRef.current) return;
      const delta = moveEvent.clientX - startX;
      const newWidth = Math.min(480, Math.max(200, startWidth + delta));
      setSidebarWidth(newWidth);
    };

    const handleMouseUp = (): void => {
      isResizingRef.current = false;
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };

    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  }, [sidebarWidth]);


  // ✅ MUST: fitView() 호출로 초기 뷰 설정
  const onInit = useCallback((reactFlowInstance: ReactFlowInstance<NetworkNodeType, Edge>) => {
    reactFlowRef.current = reactFlowInstance as any;
    // 초기 뷰 설정
    setTimeout(() => {
      reactFlowInstance.fitView(defaultFitViewOptions);
    }, 100);
  }, []);

  // React Flow 노드/엣지 상태 업데이트
  useEffect(() => {
    setNodes(reactFlowNodes);
    setEdges(reactFlowEdges);
  }, [reactFlowNodes, reactFlowEdges, setNodes, setEdges]);

  // ✅ MUST: 1000+ 노드 처리 시 가상화 적용 (성능 최적화)
  const shouldUseVirtualization = useMemo(() => {
    return filteredData.nodes.length > 1000;
  }, [filteredData.nodes.length]);

  // 에러 상태 표시
  if (apiError) {
    return (
      <div className="network-topology error-state">
        <div className="error-message">
          <h3>😱 API 에러</h3>
          <p>{apiError}</p>
          <button onClick={() => fetchTopologyData('retry-button')} className="retry-button">
            다시 시도
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="network-topology">
      <div className="network-header">
        <NetworkHeader
          timeRange={currentTimeRange}
          refreshState={currentRefreshState}
          onTimeRangeChange={handleTimeRangeChange}
          onManualRefresh={handleManualRefresh}
          onAutoRefreshChange={handleAutoRefreshChange}
          onQuickRefresh={handleQuickRefresh}
          isLoading={loading}
          sidebarCollapsed={sidebarCollapsed}
          sidebarWidth={sidebarWidth}
          onToggleSidebar={handleToggleSidebar}
          onPanelClose={headerPanelCloseRef}
        />
      </div>

      {/* 메인 컨텐츠 영역 */}
      <div className="network-content">
        {/* 왼쪽 사이드바 - 리사이즈 가능 */}
        <div
          className={`network-sidebar ${sidebarCollapsed ? 'collapsed' : ''}`}
          style={!sidebarCollapsed ? { width: sidebarWidth, minWidth: sidebarWidth } : undefined}
        >
          <NetworkControls
            filters={filters}
            visualMode={visualMode}
            onFiltersChange={setFilters}
            onVisualModeChange={setVisualMode}
            onRefreshData={() => fetchTopologyData('controls-refresh')}
            isLoading={loading}
            filteredData={{
              nodes: filteredData.nodes as any[],
              edges: filteredData.edges as any[]
            }}
            originalData={{
              nodes: (realData as any)?.originalBackendData?.nodes || [],
              edges: (realData as any)?.originalBackendData?.edges || []
            }}
          />
          {!sidebarCollapsed && (
            <div
              className="sidebar-resize-handle"
              onMouseDown={handleResizeMouseDown}
              role="separator"
              aria-orientation="vertical"
              aria-label="Resize sidebar"
            />
          )}
        </div>

        <div className={`react-flow-container ${detailPanelOpen ? 'with-detail-panel' : ''}`}>
        <ReactFlow
          nodes={nodesWithOpacity}
          edges={edgesWithOpacity}
          onNodesChange={handleNodesChange}
          onEdgesChange={handleEdgesChange}
          onConnect={onConnect}
          onNodeClick={handleNodeClick}
          onNodeMouseEnter={handleNodeMouseEnter}
          onNodeMouseLeave={handleNodeMouseLeave}
          onNodeDragStop={handleNodeDragStop}
          onEdgeClick={handleEdgeClick}
          onEdgeMouseEnter={handleEdgeMouseEnter}
          onEdgeMouseLeave={handleEdgeMouseLeave}
          onPaneClick={handlePaneClick}
          // onInit={onInit}
          connectionLineType={ConnectionLineType.SmoothStep}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          fitView={false} // onInit에서 수동으로 처리
          attributionPosition="top-right"
          // ✅ MUST: 1000+ 노드 대응 가상화
          onlyRenderVisibleElements={shouldUseVirtualization}
          nodesDraggable={!shouldUseVirtualization}
          nodesConnectable={false}
          elementsSelectable={false} // 노드 선택 시 작은 패널 제거
          zoomOnScroll={true}
          zoomOnPinch={true}
          panOnScroll={false}
          panOnDrag={true}
          deleteKeyCode={null} // 삭제 방지
          // 엣지 잔상 방지 최적화 설정
          elevateEdgesOnSelect={false} // 엣지 선택 시 z-index 변경 방지
          disableKeyboardA11y={false} // 키보드 접근성 유지
        >
          <Controls showZoom showFitView showInteractive={false} />
          {nodes.length > 0 && (
            <MiniMap
              nodeStrokeColor={isDarkMode ? "rgba(238, 238, 238, 0.3)" : "rgba(0, 0, 0, 0.3)"}
              nodeColor={isDarkMode ? "#2d3748" : "#e0e0e0"}
              nodeBorderRadius={4}
              maskColor={isDarkMode ? "rgba(255,255,255,0.05)" : "rgb(105 108 109 / 7%)"}
              style={{
                height: 120,
                width: 200,
                bottom: 20,
                right: 20,
              }}
              pannable={true}
              zoomable={true}
            />
          )}
          <Background
            variant={BackgroundVariant.Lines}
            gap={20}
            size={0.3}
            color={isDarkMode ? "#1e242d" : "#f5f5f5"}
          />
        </ReactFlow>
      </div>

      {/* 네트워크 맵 로더 - Lottie 애니메이션 */}
      {(loading || isELKCalculating || (filteredData.nodes.length > 0 && elkLayoutedNodes.length === 0)) && (
        <NetworkMapLoader
          message={
            loading
              ? 'Loading network data...'
              : 'Preparing network topology...'
          }
          size="large"
        />
      )}
      </div>

      

      {detailPanelOpen && selectedNodeData && (


        <div className="detail-panel">
          <div className="detail-panel-header">
            <div className="detail-panel-title">
              <h3>{selectedNodeData.workloadName}</h3>
              <div className="detail-panel-subtitle">
                {selectedNodeData.namespace} • {selectedNodeData.cluster}
              </div>
            </div>
            <button 
              className="detail-panel-close" 
              onClick={handleCloseDetailPanel}
              aria-label="Close detail panel"
            >
              ✕
            </button>
          </div>

          <div className="detail-panel-content">
            {selectedNodeData.isExternal === 1 ? (
              <>
                <div className="detail-section">
                  <h4>External Service Info</h4>
                  <div className="external-info-grid">
                    <div className="external-info-item">
                      <span className="external-info-label">Service Name</span>
                      <span className="external-info-value">{selectedNodeData.workloadName}</span>
                    </div>
                    <div className="external-info-item">
                      <span className="external-info-label">Type</span>
                      <span className="external-info-value">External Service</span>
                    </div>
                    <div className="external-info-item">
                      <span className="external-info-label">Domain</span>
                      <span className="external-info-value">{selectedNodeData.workloadName}</span>
                    </div>
                    <div className="external-info-item">
                      <span className="external-info-label">Connection Status</span>
                      <span className={`external-info-value status-${selectedNodeData.status}`}>
                        {selectedNodeData.status === 'Ok' ? 'Connected' : 'Connection Issues'}
                      </span>
                    </div>
                  </div>
                </div>
              </>
            ) : (
              <>
                <div className="detail-section">
                  <h4>{t('overview')}</h4>
                  <div className="overview-grid">
                    <div className="overview-item">
                      <span className="overview-label">Status</span>
                      <span className={`overview-value status-${selectedNodeData.status}`}>
                        {selectedNodeData.status}
                      </span>
                    </div>
                    <div className="overview-item">
                        <span className="overview-label">Kind</span>
                        <span className="overview-value">{workloadDetailsData?.kind ? capitalizeFirstLetter(workloadDetailsData.kind) : "Unknown"}</span>
                    </div>
                    <div className="overview-item">
                      <span className="overview-label">Replicas</span>
                      <span className="overview-value">
                        {workloadDetailsData ? `${workloadDetailsData.runningPods}/${workloadDetailsData.replicas}` : '1/1'}
                      </span>
                    </div>
                  </div>
                  </div>
                  
                <div className="detail-section">
                  <h4>{t('performance_metrics')}</h4>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', marginTop: '16px' }}>
                    {workloadDetailsLoading ? (
                      <div style={{ padding: '20px', textAlign: 'center', color: '#a0aec0' }}>
                        {t('loading_workload_details')}
                      </div>
                    ) : workloadDetailsData ? (
                      <>
                        <MetricChart
                          metricData={workloadDetailsData.cpuMetrics}
                          title="CPU usage, request, limits"
                          metricType="cpu"
                          isDarkMode={isDarkMode}
                        />
                        <MetricChart
                          metricData={workloadDetailsData.memoryMetrics}
                          title="Memory usage, request, limits"
                          metricType="memory"
                          isDarkMode={isDarkMode}
                        />
                        <MetricChart
                          metricData={workloadDetailsData.networkIoMetrics}
                          title="Network IO"
                          metricType="networkIO"
                          isDarkMode={isDarkMode}
                        />
                        <MetricChart
                          metricData={workloadDetailsData.networkErrorMetrics}
                          title="Network errors count"
                          metricType="networkErrors"
                          isDarkMode={isDarkMode}
                        />
                      </>
                    ) : (
                      <div style={{ padding: '20px', textAlign: 'center', color: '#ff6b6b' }}>
                        {t('failed_to_load_workload_details')}
                      </div>
                    )}
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* 엣지 상세 패널 - 0009.png 스타일 */}
      {edgeDetailPanelOpen && selectedEdgeData && (
        <NetworkEdgeDetailPanel
          edgeData={selectedEdgeData}
          onClose={handleCloseEdgeDetailPanel}
          onTraceClick={handleOpenTraceDetailPanel}
          timeRange={currentTimeRange}
        />
      )}

      {traceDetailPanelOpen && selectedTraceId && (
        <>
          <div
            className="trace-detail-side-panel-backdrop"
            onClick={handleCloseTraceDetailPanel}
            aria-hidden="true"
          />
          <TraceDetailSidePanel
            traceId={selectedTraceId}
            onClose={handleCloseTraceDetailPanel}
          />
        </>
      )}
    </div>
  );
});

// ✅ MUST: React.memo() displayName 설정
NetworkTopology.displayName = 'NetworkTopology';

export default NetworkTopology;
