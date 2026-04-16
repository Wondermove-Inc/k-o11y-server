/**
 * ELK Layout React Hook - React Flow 공식 예제 기반
 * Groundcover 스타일 네트워크 레이아웃을 위한 커스텀 훅
 */
import { useState, useCallback, useEffect, useRef } from 'react';
import { NetworkNodeData, NetworkEdgeData, Position } from '../types';
import { getLayoutedElements, elkLayoutPresets } from '../utils/elkLayout';

export interface UseELKLayoutOptions {
  preset?: keyof typeof elkLayoutPresets;
  customOptions?: Record<string, string>;
  autoLayout?: boolean; // 데이터 변경 시 자동 레이아웃
}

export interface UseELKLayoutReturn {
  layoutedNodes: Array<NetworkNodeData & Position>;
  isCalculating: boolean;
  calculateLayout: () => Promise<void>;
  resetLayout: () => void;
  error: string | null;
}

/**
 * ELK Layout Hook
 */
export function useELKLayout(
  nodes: NetworkNodeData[],
  edges: NetworkEdgeData[],
  options: UseELKLayoutOptions = {}
): UseELKLayoutReturn {
  const { preset = 'groundcover', customOptions = {}, autoLayout = false } = options;
  
  // 상태 관리
  const [layoutedNodes, setLayoutedNodes] = useState<Array<NetworkNodeData & Position>>([]);
  const [isCalculating, setIsCalculating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // 중복 계산 방지를 위한 ref
  const calculationRef = useRef<AbortController | null>(null);
  const lastInputHashRef = useRef<string>('');
  
  /**
   * 입력 데이터의 해시값 계산 (중복 계산 방지)
   */
  const getInputHash = useCallback(() => {
    return JSON.stringify({
      nodeIds: nodes.map(n => n.id).sort(),
      edgeIds: edges.map(e => e.id).sort(),
      preset,
      customOptions,
    });
  }, [nodes, edges, preset, customOptions]);
  
  /**
   * ELK Layout 계산 실행
   */
  const calculateLayout = useCallback(async () => {
    // 빈 데이터인 경우 스킵
    if (nodes.length === 0) {
      setLayoutedNodes([]);
      return;
    }
    
    // 중복 계산 방지 체크
    const currentInputHash = getInputHash();
    if (currentInputHash === lastInputHashRef.current && layoutedNodes.length > 0) {
      console.log('[useELKLayout] 중복 계산 방지 - 스킵');
      return;
    }
    
    // 이전 계산 취소
    if (calculationRef.current) {
      calculationRef.current.abort();
    }
    
    // 새로운 AbortController 생성
    const abortController = new AbortController();
    calculationRef.current = abortController;
    
    try {
      setIsCalculating(true);
      setError(null);
      
      console.log('[useELKLayout] 레이아웃 계산 시작:', {
        노드수: nodes.length,
        엣지수: edges.length,
        프리셋: preset,
      });
      
      // ELK 레이아웃 옵션 결합
      const elkOptions = {
        ...elkLayoutPresets[preset],
        ...customOptions,
      };
      
      // 계산이 취소되었는지 확인
      if (abortController.signal.aborted) {
        return;
      }
      
      // ELK Layout 계산
      const layoutResult = await getLayoutedElements(nodes, edges, elkOptions);
      
      // 계산이 취소되었는지 확인
      if (abortController.signal.aborted) {
        return;
      }
      
      console.log('[useELKLayout] 레이아웃 계산 완료:', {
        변환된_노드수: layoutResult.nodes.length,
      });
      
      // 상태 업데이트
      setLayoutedNodes(layoutResult.nodes);
      lastInputHashRef.current = currentInputHash;
      
    } catch (err) {
      // AbortError가 아닌 경우에만 에러 처리
      if (err instanceof Error && err.name !== 'AbortError') {
        console.error('[useELKLayout] 레이아웃 계산 실패:', err);
        setError(err.message);
        
        // Fallback: 원본 노드에 기본 위치 할당
        const fallbackNodes = nodes.map((node, index) => ({
          ...node,
          x: (index % 5) * 200,
          y: Math.floor(index / 5) * 120,
        }));
        setLayoutedNodes(fallbackNodes);
      }
    } finally {
      setIsCalculating(false);
      calculationRef.current = null;
    }
  }, [nodes, edges, preset, customOptions, getInputHash, layoutedNodes.length]);
  
  /**
   * 레이아웃 리셋
   */
  const resetLayout = useCallback(() => {
    // 진행 중인 계산 취소
    if (calculationRef.current) {
      calculationRef.current.abort();
      calculationRef.current = null;
    }
    
    setLayoutedNodes([]);
    setError(null);
    setIsCalculating(false);
    lastInputHashRef.current = '';
    
    console.log('[useELKLayout] 레이아웃 리셋 완료');
  }, []);
  
  /**
   * 자동 레이아웃 효과
   */
  useEffect(() => {
    if (autoLayout && nodes.length > 0) {
      console.log('[useELKLayout] 자동 레이아웃 트리거');
      calculateLayout();
    }
  }, [autoLayout, calculateLayout, nodes.length]);
  
  /**
   * 컴포넌트 언마운트 시 정리
   */
  useEffect(() => {
    return () => {
      if (calculationRef.current) {
        calculationRef.current.abort();
      }
    };
  }, []);
  
  return {
    layoutedNodes,
    isCalculating,
    calculateLayout,
    resetLayout,
    error,
  };
}