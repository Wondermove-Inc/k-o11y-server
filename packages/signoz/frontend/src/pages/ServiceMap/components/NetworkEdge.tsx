import React, { memo, useMemo } from 'react';
import { getBezierPath, EdgeLabelRenderer, Position } from '@xyflow/react';
import { CustomNetworkEdgeProps, NetworkEdgeData } from '../types';
import { useIsDarkMode } from 'hooks/useDarkMode';

const NetworkEdge: React.FC<CustomNetworkEdgeProps> = memo(({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
  selected,
  style,
  onLabelMouseEnter,
  onLabelMouseLeave
}) => {
  const isDarkMode = useIsDarkMode();
  // 데이터 구조 단순화
  const edgeData = data;
  const [edgePath, labelX, labelY] = useMemo(() => {
    return getBezierPath({
      sourceX,
      sourceY,
      targetX,
      targetY,
      sourcePosition,
      targetPosition,
    });
  }, [sourceX, sourceY, targetX, targetY, sourcePosition, targetPosition]);

  const edgeColor = useMemo(() => {
    if (edgeData.isError) return '#dc3545';
    return isDarkMode ? '#6c757d' : '#b8b8b8';
  }, [edgeData.isError, isDarkMode]);

  const edgeWidth = useMemo(() => {
    const baseWidth = 1;
    const maxWidth = 1.4;
    const requestCount = (edgeData as any).metrics?.requestCount || 0;
    
    if (requestCount === 0) return baseWidth;
    
    const logScale = Math.log10(Math.max(1, requestCount)) / 3;
    return Math.min(maxWidth, baseWidth + (maxWidth - baseWidth) * logScale);
  }, [(edgeData as any).metrics?.requestCount]);

  const edgeStyle = useMemo(() => {
    let strokeColor;
    if (edgeData.isError) {
      strokeColor = '#dc3545';
    } else if (style?.stroke === '#00C02A') {
      strokeColor = '#00C02A';
    } else if (selected) {
      strokeColor = '#00C02A';
    } else if (style?.stroke && style.stroke !== edgeColor) {
      strokeColor = style.stroke;
    } else {
      strokeColor = edgeColor;
    }
    
    const baseStyle = {
      stroke: strokeColor,
      strokeWidth: edgeWidth,
      fill: 'none',
    };

    return baseStyle;
  }, [id, edgeColor, edgeWidth, selected, edgeData.isError, style?.opacity, style?.stroke]);

  const protocolDisplay = useMemo(() => {
    return edgeData.protocol;
  }, [edgeData.protocol]);


  const labelStyle = useMemo(() => {
    let backgroundColor, borderColor, textColor;

    if (edgeData.isError) {
      backgroundColor = isDarkMode ? 'rgba(220, 53, 69, 1)' : '#ffffff';
      borderColor = '#dc3545';
      textColor = isDarkMode ? '#ffffff' : '#dc3545';
    } else if (selected || style?.stroke === '#00C02A') {
      backgroundColor = isDarkMode ? 'rgba(40, 167, 69, 1)' : '#ffffff';
      borderColor = '#00C02A';
      textColor = isDarkMode ? '#ffffff' : '#00C02A';
    } else {
      backgroundColor = isDarkMode ? 'rgba(30, 36, 45, 1)' : '#ffffff';
      borderColor = isDarkMode ? edgeColor : 'rgba(0, 0, 0, 0.2)';
      textColor = isDarkMode ? '#ffffff' : '#1f1f1f';
    }

    return {
      position: 'absolute' as const,
      transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
      fontSize: 11,
      fontWeight: 600,
      background: backgroundColor,
      padding: '3px 8px',
      borderRadius: '20px',
      border: `2px solid ${borderColor}`,
      color: textColor,
      pointerEvents: 'all' as const,
      display: 'flex',
      flexDirection: 'column' as const,
      alignItems: 'center' as const,
      gap: '2px',
      minWidth: '40px',
      textAlign: 'center' as const,
      opacity: style?.opacity ?? 1,
    };
  }, [labelX, labelY, edgeColor, edgeData.isError, selected, style?.opacity, style?.stroke, isDarkMode]);

  const edgeClassName = useMemo(() => {
    const classes = ['react-flow__edge-path'];
    
    if (edgeData.isError) {
      classes.push('error');
    }
    
    if (selected) {
      classes.push('selected');
    }
    
    return classes.join(' ');
  }, [edgeData.isError, selected]);

  const arrowPath = useMemo(() => {
    const distance = Math.sqrt((targetX - sourceX) ** 2 + (targetY - sourceY) ** 2);
    const defaultOffset = Math.min(distance / 2, 120);
    
    // React Flow getBezierPath와 동일한 제어점 계산
    let [sourceControlX, sourceControlY] = [sourceX, sourceY];
    let [targetControlX, targetControlY] = [targetX, targetY];
    
    switch (sourcePosition) {
      case Position.Left:
        sourceControlX = sourceX - defaultOffset;
        break;
      case Position.Right:
        sourceControlX = sourceX + defaultOffset;
        break;
      case Position.Top:
        sourceControlY = sourceY - defaultOffset;
        break;
      case Position.Bottom:
        sourceControlY = sourceY + defaultOffset;
        break;
    }
    
    switch (targetPosition) {
      case Position.Left:
        targetControlX = targetX - defaultOffset;
        break;
      case Position.Right:
        targetControlX = targetX + defaultOffset;
        break;
      case Position.Top:
        targetControlY = targetY - defaultOffset;
        break;
      case Position.Bottom:
        targetControlY = targetY + defaultOffset;
        break;
    }
    
    const sampleT1 = 0.99;
    const sampleT2 = 1.0;
    
    const getBezierPoint = (t: number) => {
      const t1 = 1 - t;
      const t1_2 = t1 * t1;
      const t1_3 = t1_2 * t1;
      const t_2 = t * t;
      const t_3 = t_2 * t;
      
      const x = t1_3 * sourceX + 
                3 * t1_2 * t * sourceControlX + 
                3 * t1 * t_2 * targetControlX + 
                t_3 * targetX;
                
      const y = t1_3 * sourceY + 
                3 * t1_2 * t * sourceControlY + 
                3 * t1 * t_2 * targetControlY + 
                t_3 * targetY;
                
      return [x, y];
    };
    
    const [x1, y1] = getBezierPoint(sampleT1);
    const [x2, y2] = getBezierPoint(sampleT2);
    
    const tangentX = x2 - x1;
    const tangentY = y2 - y1;
    const tangentLength = Math.sqrt(tangentX * tangentX + tangentY * tangentY);
    
    let ux, uy;
    
    if (tangentLength > 0.001) {
      ux = tangentX / tangentLength;
      uy = tangentY / tangentLength;
    } else {
      const dx = targetX - sourceX;
      const dy = targetY - sourceY;
      const dist = Math.sqrt(dx * dx + dy * dy);
      
      if (dist < 0.001) return '';
      
      ux = dx / dist;
      uy = dy / dist;
    }
    
    const arrowLength = 8;
    const arrowWidth = 4;
    
    const offset = arrowLength * 0.3;
    
    const arrowTipX = targetX + ux * (arrowLength - offset);
    const arrowTipY = targetY + uy * (arrowLength - offset);
    
    const arrowBaseX = targetX - ux * offset;
    const arrowBaseY = targetY - uy * offset;
    
    const perpX = -uy;
    const perpY = ux;
    
    const leftX = arrowBaseX + perpX * arrowWidth;
    const leftY = arrowBaseY + perpY * arrowWidth;
    const rightX = arrowBaseX - perpX * arrowWidth;
    const rightY = arrowBaseY - perpY * arrowWidth;
    
    return `M ${leftX} ${leftY} L ${arrowTipX} ${arrowTipY} L ${rightX} ${rightY} Z`;
  }, [sourceX, sourceY, targetX, targetY, sourcePosition, targetPosition]);

  const arrowColor = useMemo(() => {
    if (edgeData.isError) {
      return '#dc3545'; 
    } else if (style?.stroke === '#00C02A') {
      return '#00C02A';
    } else if (selected) {
      return '#00C02A';
    } else if (style?.stroke && style.stroke !== edgeColor) {
      return style.stroke;
    } else {
      return edgeColor; 
    }
  }, [edgeData.isError, selected, style?.stroke, edgeColor]);

  const edgeContainerClass = useMemo(() => {
    const classes = [];
    
    if (edgeData.isError) {
      classes.push('error');
    }
    
    if (selected) {
      classes.push('selected');
    }
    
    return classes.join(' ');
  }, [edgeData.isError, selected]);

  return (
    <g className={edgeContainerClass}>
      <path
        id={id}
        style={edgeStyle}
        className={edgeClassName}
        d={edgePath}
      />
      <path
        d={arrowPath}
        fill={arrowColor}
        stroke="none"
      />
      
      <EdgeLabelRenderer>
        <div
          style={labelStyle}
          className="custom-edge-label"
          role="tooltip"
          aria-label={`Connection: ${edgeData.protocol} from ${edgeData.source} to ${edgeData.destination}`}
          onMouseEnter={() => onLabelMouseEnter?.(edgeData.id)}
          onMouseLeave={() => onLabelMouseLeave?.()}
        >
          <div className="protocol-display" title={`Protocol: ${edgeData.protocol}`}>
            {protocolDisplay}
          </div>
        </div>
      </EdgeLabelRenderer>
    </g>
  );
});

NetworkEdge.displayName = 'NetworkEdge';

export default NetworkEdge; 
