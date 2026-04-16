/**
 * ELK Multiple Handles React Hook - React Flow 공식 예제 기반  
 * Groundcover 스타일 전문적인 네트워크 레이아웃을 위한 훅
 */
import { useState, useCallback, useEffect, useRef } from 'react';
import { NetworkNodeData, NetworkEdgeData, Position } from '../types';
import { calculateElkMultipleHandlesLayout } from '../utils/elkLayoutAdvanced';

export interface UseElkMultipleHandlesOptions {
  autoLayout?: boolean; // 데이터 변경 시 자동 레이아웃
  debounceMs?: number;  // 디바운스 시간 (기본값: 300ms)
}

export interface UseElkMultipleHandlesReturn {
  layoutedNodes: Array<NetworkNodeData & Position>;
  isCalculating: boolean;
  calculateLayout: () => Promise<void>;
  resetLayout: () => void;
  error: string | null;
}

/**
 * ELK Multiple Handles Layout Hook
 */
export function useElkMultipleHandles(
  nodes: NetworkNodeData[],
  edges: NetworkEdgeData[],
  options: UseElkMultipleHandlesOptions = {}
): UseElkMultipleHandlesReturn {
  const { autoLayout = false, debounceMs = 300 } = options;
  
  // 상태 관리
  const [layoutedNodes, setLayoutedNodes] = useState<Array<NetworkNodeData & Position>>([]);
  const [isCalculating, setIsCalculating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // 중복 계산 방지
  const calculationRef = useRef<AbortController | null>(null);
  const debounceRef = useRef<NodeJS.Timeout | null>(null);
  const lastInputHashRef = useRef<string>('');
  
  /**
   * 입력 데이터 해시 계산 (중복 방지)
   */
  const getInputHash = useCallback(() => {
    return JSON.stringify({
      nodeIds: nodes.map(n => n.id).sort(),
      edgeIds: edges.map(e => e.id).sort(),
      nodeCount: nodes.length,
      edgeCount: edges.length,
    });
  }, [nodes, edges]);
  
  /**
   * ELK Multiple Handles Layout 계산
   */
  const calculateLayout = useCallback(async () => {
    // 빈 데이터 체크
    if (nodes.length === 0) {
      setLayoutedNodes([]);
      setError(null);
      return;
    }

    // 중복 계산 방지
    const currentHash = getInputHash();
    if (currentHash === lastInputHashRef.current && layoutedNodes.length > 0) {
      return;
    }

    // 이전 계산 취소
    if (calculationRef.current) {
      calculationRef.current.abort();
    }

    const abortController = new AbortController();
    calculationRef.current = abortController;

    try {
      setIsCalculating(true);
      setError(null);
      // 취소 체크
      if (abortController.signal.aborted) return;

      // ELK Multiple Handles Layout 계산
      const layoutResult = await calculateElkMultipleHandlesLayout(nodes, edges);

      // 취소 체크
      if (abortController.signal.aborted) return;

      // 상태 업데이트
      setLayoutedNodes(layoutResult.nodes);
      lastInputHashRef.current = currentHash;
      
    } catch (err) {
      if (err instanceof Error && err.name !== 'AbortError') {
        console.error('[useElkMultipleHandles] 레이아웃 계산 실패:', err);
        setError(err.message);
        
        // Fallback: Groundcover 스타일 기본 레이아웃
        const fallbackNodes = nodes.map((node, index) => ({
          ...node,
          x: (index % 4) * 220 + 50,
          y: Math.floor(index / 4) * 120 + 50,
        }));
        setLayoutedNodes(fallbackNodes);
      }
    } finally {
      setIsCalculating(false);
      calculationRef.current = null;
    }
  }, [nodes, edges, getInputHash, layoutedNodes.length]);

  /**
   * 디바운스된 계산 함수
   */
  const debouncedCalculateLayout = useCallback(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }
    
    debounceRef.current = setTimeout(() => {
      calculateLayout();
    }, debounceMs);
  }, [calculateLayout, debounceMs]);

  /**
   * 레이아웃 리셋
   */
  const resetLayout = useCallback(() => {
    // 진행 중인 계산들 취소
    if (calculationRef.current) {
      calculationRef.current.abort();
      calculationRef.current = null;
    }
    
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
      debounceRef.current = null;
    }

    // 상태 초기화
    setLayoutedNodes([]);
    setError(null);
    setIsCalculating(false);
    lastInputHashRef.current = '';
    
  }, []);

  /**
   * 자동 레이아웃 효과
   */
  useEffect(() => {
    if (autoLayout && nodes.length > 0) {
      debouncedCalculateLayout();
    }
    
    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
    };
  }, [autoLayout, debouncedCalculateLayout, nodes.length]);

  /**
   * 컴포넌트 언마운트 정리
   */
  useEffect(() => {
    return () => {
      if (calculationRef.current) {
        calculationRef.current.abort();
      }
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
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