/**
 * ELK.js Layout Utility - React Flow 공식 예제 기반
 * Groundcover 스타일 네트워크 시각화를 위한 ELK Layout 엔진
 */
import ELK from 'elkjs/lib/elk.bundled.js';
import { NetworkNodeData, NetworkEdgeData, Position } from '../types';

// ELK 인스턴스 생성
const elk = new ELK();

// ELK Layout 옵션 (React Flow 공식 예제 기반)
const defaultLayoutOptions = {
  'elk.algorithm': 'layered',
  'elk.layered.spacing.nodeNodeBetweenLayers': '100', // 레이어 간 간격
  'elk.layered.spacing.nodeNodeWithinLayer': '50',   // 동일 레이어 노드 간격
  'elk.spacing.nodeNode': '80',                      // 일반 노드 간격
  'elk.direction': 'RIGHT',                          // 레이아웃 방향
  'elk.portConstraints': 'FIXED_ORDER',              // 포트 제약 (엣지 교차 최소화)
  'elk.layered.layering.strategy': 'LONGEST_PATH',   // 레이어링 전략
  'elk.layered.crossingMinimization.strategy': 'LAYER_SWEEP', // 교차 최소화 전략
  'elk.layered.nodePlacement.strategy': 'BRANDES_KOEPF',      // 노드 배치 전략
};

/**
 * ELK Graph 구조로 변환
 */
function transformToElkGraph(
  nodes: NetworkNodeData[], 
  edges: NetworkEdgeData[], 
  options: Record<string, string> = {}
) {
  // 네임스페이스별 클러스터링 고려
  const namespaceGroups = nodes.reduce((acc, node) => {
    if (!acc[node.namespace]) {
      acc[node.namespace] = [];
    }
    acc[node.namespace].push(node);
    return acc;
  }, {} as Record<string, NetworkNodeData[]>);

  // 다중 네임스페이스인 경우 클러스터링 적용
  const shouldCluster = Object.keys(namespaceGroups).length > 1;

  if (shouldCluster) {
    // 클러스터링 구조로 ELK Graph 생성
    const children = Object.entries(namespaceGroups).map(([namespace, nsNodes]) => ({
      id: `cluster_${namespace}`,
      layoutOptions: {
        'elk.algorithm': 'layered',
        'elk.direction': 'DOWN',
        'elk.spacing.nodeNode': '30',
        'elk.padding': '[top=20,left=20,bottom=20,right=20]',
      },
      children: nsNodes.map(node => ({
        id: node.id,
        width: 180,
        height: 80,
      })),
      edges: edges
        .filter(edge => {
          const sourceNode = nsNodes.find(n => n.id === edge.source);
          const targetNode = nsNodes.find(n => n.id === edge.target);
          return sourceNode && targetNode;
        })
        .map(edge => ({
          id: edge.id,
          sources: [edge.source],
          targets: [edge.target],
        })),
    }));

    return {
      id: 'root',
      layoutOptions: { ...defaultLayoutOptions, ...options },
      children,
      edges: edges
        .filter(edge => {
          const sourceNamespace = nodes.find(n => n.id === edge.source)?.namespace;
          const targetNamespace = nodes.find(n => n.id === edge.target)?.namespace;
          return sourceNamespace !== targetNamespace;
        })
        .map(edge => ({
          id: edge.id,
          sources: [`cluster_${nodes.find(n => n.id === edge.source)?.namespace}`],
          targets: [`cluster_${nodes.find(n => n.id === edge.target)?.namespace}`],
        })),
    };
  } else {
    // 단일 네임스페이스 - 평면 레이아웃
    return {
      id: 'root',
      layoutOptions: { ...defaultLayoutOptions, ...options },
      children: nodes.map(node => ({
        id: node.id,
        width: 180,
        height: 80,
      })),
      edges: edges.map(edge => ({
        id: edge.id,
        sources: [edge.source],
        targets: [edge.target],
      })),
    };
  }
}

/**
 * ELK 결과를 React Flow 노드로 변환
 */
function transformFromElkGraph(
  layoutedGraph: any, 
  originalNodes: NetworkNodeData[]
): Array<NetworkNodeData & Position> {
  const resultNodes: Array<NetworkNodeData & Position> = [];

  function extractNodes(elkNode: any, offsetX = 0, offsetY = 0) {
    if (elkNode.children) {
      elkNode.children.forEach((child: any) => {
        if (child.children) {
          // 클러스터인 경우 재귀적으로 처리
          extractNodes(child, offsetX + (child.x || 0), offsetY + (child.y || 0));
        } else {
          // 실제 노드인 경우
          const originalNode = originalNodes.find(n => n.id === child.id);
          if (originalNode && child.x !== undefined && child.y !== undefined) {
            resultNodes.push({
              ...originalNode,
              x: child.x + offsetX,
              y: child.y + offsetY,
            });
          }
        }
      });
    }
  }

  extractNodes(layoutedGraph);
  return resultNodes;
}

/**
 * ELK Layout 계산 메인 함수
 */
export async function getLayoutedElements(
  nodes: NetworkNodeData[],
  edges: NetworkEdgeData[],
  options: Record<string, string> = {}
): Promise<{
  nodes: Array<NetworkNodeData & Position>;
  edges: NetworkEdgeData[];
}> {
  try {
    console.log('[ELK Layout] 레이아웃 계산 시작:', { 노드수: nodes.length, 엣지수: edges.length });
    
    // ELK Graph 구조로 변환
    const elkGraph = transformToElkGraph(nodes, edges, options);
    console.log('[ELK Layout] ELK Graph 생성 완료:', elkGraph);
    
    // ELK.js로 레이아웃 계산
    const layoutedGraph = await elk.layout(elkGraph);
    console.log('[ELK Layout] 레이아웃 계산 완료:', layoutedGraph);
    
    // React Flow 노드로 변환
    const layoutedNodes = transformFromElkGraph(layoutedGraph, nodes);
    console.log('[ELK Layout] 노드 변환 완료:', { 변환된_노드수: layoutedNodes.length });
    
    return {
      nodes: layoutedNodes,
      edges: edges,
    };
  } catch (error) {
    console.error('[ELK Layout] 레이아웃 계산 실패:', error);
    
    // 실패 시 기본 그리드 레이아웃으로 fallback
    const fallbackNodes = nodes.map((node, index) => ({
      ...node,
      x: (index % 4) * 200,
      y: Math.floor(index / 4) * 100,
    }));
    
    return {
      nodes: fallbackNodes,
      edges: edges,
    };
  }
}

/**
 * 사전 정의된 ELK Layout 옵션들
 */
export const elkLayoutPresets = {
  // Groundcover 스타일 - 계층형 레이아웃
  groundcover: {
    'elk.algorithm': 'layered',
    'elk.direction': 'RIGHT',
    'elk.layered.spacing.nodeNodeBetweenLayers': '120',
    'elk.layered.spacing.nodeNodeWithinLayer': '60',
    'elk.spacing.nodeNode': '80',
    'elk.layered.layering.strategy': 'LONGEST_PATH',
    'elk.layered.crossingMinimization.strategy': 'LAYER_SWEEP',
    'elk.layered.nodePlacement.strategy': 'BRANDES_KOEPF',
    'elk.portConstraints': 'FIXED_ORDER',
  },
  
  // 수직 트리 레이아웃
  vertical: {
    'elk.algorithm': 'layered',
    'elk.direction': 'DOWN',
    'elk.layered.spacing.nodeNodeBetweenLayers': '80',
    'elk.layered.spacing.nodeNodeWithinLayer': '50',
    'elk.spacing.nodeNode': '60',
    'elk.layered.layering.strategy': 'LONGEST_PATH',
  },
  
  // 포스 기반 레이아웃
  force: {
    'elk.algorithm': 'force',
    'elk.force.iterations': '300',
    'elk.force.repulsion': '-500',
    'elk.force.attraction': '0.1',
    'elk.spacing.nodeNode': '100',
  },
  
  // 방사형 레이아웃  
  radial: {
    'elk.algorithm': 'radial',
    'elk.radial.radius': '200',
    'elk.spacing.nodeNode': '80',
  },
};