import { NetworkNodeData, NetworkEdgeData, Position } from '../types';

export async function calculateElkMultipleHandlesLayout(
  nodes: NetworkNodeData[],
  edges: NetworkEdgeData[]
): Promise<{
  nodes: Array<NetworkNodeData & Position>;
  edges: NetworkEdgeData[];
}> {
  try {
    const layoutedNodes = calculateCustomLayout(nodes, edges);
    
    return {
      nodes: layoutedNodes,
      edges: edges,
    };
  } catch (error) {
    
    const fallbackNodes = nodes.map((node, index) => ({
      ...node,
      x: 50 + (index % 4) * 220,
      y: 100 + Math.floor(index / 4) * 120,
    }));

    return {
      nodes: fallbackNodes,
      edges: edges,
    };
  }
}

function calculateCustomLayout(
  nodes: NetworkNodeData[],
  edges: NetworkEdgeData[]
): Array<NetworkNodeData & Position> {
  
  const incomingTargets = new Set(edges.map(e => e.target));
  const layer0Nodes = nodes.filter(node => !incomingTargets.has(node.id));
  
  const layer0Groups = groupLayer0ByTargets(layer0Nodes, edges);
  
  const layoutedNodes: Array<NetworkNodeData & Position> = [];
  const processedNodes = new Set<string>();
  
  positionLayer0Nodes(layer0Groups, layoutedNodes);
  layer0Nodes.forEach(node => processedNodes.add(node.id));
  
  positionRemainingLayers(nodes, edges, layoutedNodes, processedNodes);
  
  
  return layoutedNodes;
}

function groupLayer0ByTargets(
  layer0Nodes: NetworkNodeData[],
  edges: NetworkEdgeData[]
): Map<string, NetworkNodeData[]> {
  
  const groups = new Map<string, NetworkNodeData[]>();
  
  layer0Nodes.forEach(node => {
    const targets = edges
      .filter(e => e.source === node.id)
      .map(e => e.target)
      .sort()
      .join(',');
    
    if (!groups.has(targets)) {
      groups.set(targets, []);
    }
    groups.get(targets)!.push(node);
  });
  
  return groups;
}

function positionLayer0Nodes(
  layer0Groups: Map<string, NetworkNodeData[]>,
  layoutedNodes: Array<NetworkNodeData & Position>
): void {
  
  const baseX = 50;
  const startY = 100;
  
  const totalNodes = Array.from(layer0Groups.values()).reduce((sum, nodes) => sum + nodes.length, 0);
  const nodeSpacing = Math.max(200, Math.min(300, 4000 / totalNodes));
  const groupSpacing = Math.max(50, nodeSpacing * 0.3);
  
  const optimizedOrder = optimizeLayer0Order(layer0Groups);
  
  let currentY = startY;
  
  optimizedOrder.forEach(({ targets, nodes }, groupIndex) => {
    
    nodes.forEach((node, index) => {
      const desiredY = currentY + (index * nodeSpacing);
      const adjustedY = avoidOverlapInLayer(baseX, desiredY, layoutedNodes);
      
      layoutedNodes.push({
        ...node,
        x: baseX,
        y: adjustedY,
      });
      
    });
    
    currentY += (nodes.length * nodeSpacing) + groupSpacing;
  });
}

function optimizeLayer0Order(
  layer0Groups: Map<string, NetworkNodeData[]>
): Array<{ targets: string, nodes: NetworkNodeData[] }> {
  
  const groups = Array.from(layer0Groups.entries()).map(([targets, nodes]) => 
    ({ targets, nodes })
  );
  
  if (groups.length <= 1) return groups;
  
  
  const ordered: Array<{ targets: string, nodes: NetworkNodeData[] }> = [];
  const remaining = [...groups];
  
  const targetToGroups = new Map<string, Array<{ targets: string, nodes: NetworkNodeData[] }>>();
  
  groups.forEach(group => {
    const targetList = group.targets.split(',').filter(t => t);
    targetList.forEach(target => {
      if (!targetToGroups.has(target)) {
        targetToGroups.set(target, []);
      }
      targetToGroups.get(target)!.push(group);
    });
  });
  
  
  const sortedTargets = Array.from(targetToGroups.entries())
    .sort((a, b) => b[1].length - a[1].length);
  
  const processedGroups = new Set<string>();
  
  sortedTargets.forEach(([target, groupsForTarget]) => {
    
    groupsForTarget.forEach(group => {
      if (!processedGroups.has(group.targets)) {
        ordered.push(group);
        processedGroups.add(group.targets);
      }
    });
  });
  
  remaining.forEach(group => {
    if (!processedGroups.has(group.targets)) {
      ordered.push(group);
    }
  });
  
  
  return ordered;
}

function positionRemainingLayers(
  nodes: NetworkNodeData[],
  edges: NetworkEdgeData[],
  layoutedNodes: Array<NetworkNodeData & Position>,
  processedNodes: Set<string>
): void {
  
  const layerSpacing = 600;
  let currentLayer = 1;
  
  while (processedNodes.size < nodes.length) {
    const initialProcessedCount = processedNodes.size;
    const currentLayerNodes: NetworkNodeData[] = [];
    
    nodes.forEach(node => {
      if (processedNodes.has(node.id)) return;
      
      const parentEdges = edges.filter(e => e.target === node.id);
      if (parentEdges.length === 0) return;
      
      const allParentsProcessed = parentEdges.every(edge => 
        processedNodes.has(edge.source)
      );
      
      if (allParentsProcessed) {
        currentLayerNodes.push(node);
      }
    });
    
    if (currentLayerNodes.length === 0) {
      break;
    }
    
    if (processedNodes.size === initialProcessedCount && currentLayerNodes.length === 0) {
      break;
    }
    
    
    const layerX = 50 + (currentLayer * layerSpacing);
    
    positionLayerNodes(currentLayerNodes, layerX, edges, layoutedNodes);
    
    currentLayerNodes.forEach(node => {
      processedNodes.add(node.id);
    });
    
    currentLayer++;
  }
  
  const unprocessedNodes = nodes.filter(n => !processedNodes.has(n.id));
  if (unprocessedNodes.length > 0) {
    const finalX = 50 + (currentLayer * layerSpacing);
    unprocessedNodes.forEach((node, index) => {
      layoutedNodes.push({
        ...node,
        x: finalX,
        y: 100 + (index * 200),
      });
    });
  }
  
}

function positionLayerNodes(
  layerNodes: NetworkNodeData[],
  layerX: number,
  edges: NetworkEdgeData[],
  layoutedNodes: Array<NetworkNodeData & Position>
): void {
  
  
  const parentGroups = new Map<string, NetworkNodeData[]>();
  
  layerNodes.forEach(node => {
    const parentIds = edges
      .filter(e => e.target === node.id)
      .map(e => e.source)
      .sort()
      .join(',');
    
    if (!parentGroups.has(parentIds)) {
      parentGroups.set(parentIds, []);
    }
    parentGroups.get(parentIds)!.push(node);
  });
  
  const sortedGroups = Array.from(parentGroups.entries()).sort((a, b) => {
    const aParentIds = a[0].split(',');
    const bParentIds = b[0].split(',');
    
    const aParentsY = aParentIds.map(id => layoutedNodes.find(n => n.id === id)?.y || 0);
    const bParentsY = bParentIds.map(id => layoutedNodes.find(n => n.id === id)?.y || 0);
    
    const aMinY = Math.min(...aParentsY);
    const bMinY = Math.min(...bParentsY);
    
    return aMinY - bMinY;
  });
  
  
  sortedGroups.forEach(([parentIds, groupNodes]) => {
    const parents = parentIds.split(',')
      .map(id => layoutedNodes.find(n => n.id === id))
      .filter(Boolean) as Array<NetworkNodeData & Position>;
    
    if (parents.length === 0) return;
    
    const parentYs = parents.map(p => p.y);
    const minParentY = Math.min(...parentYs);
    const maxParentY = Math.max(...parentYs);
    const centerY = (minParentY + maxParentY) / 2;
    
    
    if (groupNodes.length === 1) {
      const child = groupNodes[0];
      let childY;
      
      if (parents.length === 1) {
        childY = parents[0].y;
      } else {
        childY = parents.reduce((sum, p) => sum + p.y, 0) / parents.length;
      }
      
      const adjustedY = avoidOverlapInLayer(layerX, childY, layoutedNodes);
      
      layoutedNodes.push({
        ...child,
        x: layerX,
        y: adjustedY,
      });
      
    } else {
      const parent = parents[0];
      const spacing = 200;
      
      
      if (groupNodes.length === 2) {
        const y1 = parent.y - spacing/2;
        const y2 = parent.y + spacing/2;
        
        const adjustedY1 = avoidOverlapInLayer(layerX, y1, layoutedNodes);
        const adjustedY2 = avoidOverlapInLayer(layerX, y2, layoutedNodes);
        
        layoutedNodes.push({
          ...groupNodes[0],
          x: layerX,
          y: adjustedY1,
        });
        layoutedNodes.push({
          ...groupNodes[1], 
          x: layerX,
          y: adjustedY2,
        });
        
      } else {
        const firstY = parent.y;
        const adjustedFirstY = avoidOverlapInLayer(layerX, firstY, layoutedNodes);
        
        layoutedNodes.push({
          ...groupNodes[0],
          x: layerX,
          y: adjustedFirstY,
        });
        
        for (let i = 1; i < groupNodes.length; i++) {
          const offsetY = Math.ceil(i / 2) * spacing * (i % 2 === 1 ? 1 : -1);
          const childY = parent.y + offsetY;
          const adjustedChildY = avoidOverlapInLayer(layerX, childY, layoutedNodes);
          
          layoutedNodes.push({
            ...groupNodes[i],
            x: layerX,
            y: adjustedChildY,
          });
        }
      }
    }
  });
}

function positionChildrenCorrectly(
  groupNodes: NetworkNodeData[],
  parents: Array<NetworkNodeData & Position>,
  parentIds: string,
  layerX: number,
  edges: NetworkEdgeData[],
  layoutedNodes: Array<NetworkNodeData & Position>
): void {
  
  if (groupNodes.length === 1) {
    const child = groupNodes[0];
    
    if (parents.length === 1) {
      const childY = parents[0].y;
      
      layoutedNodes.push({
        ...child,
        x: layerX,
        y: childY,
      });
    } else {
      const parentYs = parents.map(p => p.y);
      const minY = Math.min(...parentYs);
      const maxY = Math.max(...parentYs);
      const range = maxY - minY;
      const centerY = (minY + maxY) / 2;
      
      // 부모들이 너무 멀리 떨어져 있으면 경고
      if (range > 300) {
      }
      
      
      layoutedNodes.push({
        ...child,
        x: layerX,
        y: centerY,
      });
    }
  } else {
    if (parents.length === 1) {
      const parent = parents[0];
      const parentY = parent.y;
      const spacing = 120;
      
      
      if (groupNodes.length === 2) {
        layoutedNodes.push({
          ...groupNodes[0],
          x: layerX,
          y: parentY - spacing/2,
        });
        layoutedNodes.push({
          ...groupNodes[1],
          x: layerX,
          y: parentY + spacing/2,
        });
      } else {
        layoutedNodes.push({
          ...groupNodes[0],
          x: layerX,
          y: parentY,
        });
        
        for (let i = 1; i < groupNodes.length; i++) {
          const offsetY = Math.ceil(i / 2) * spacing * (i % 2 === 1 ? 1 : -1);
          layoutedNodes.push({
            ...groupNodes[i],
            x: layerX,
            y: parentY + offsetY,
          });
        }
      }
    } else {
      const minY = Math.min(...parents.map(p => p.y));
      const maxY = Math.max(...parents.map(p => p.y));
      const centerY = (minY + maxY) / 2;
      const range = Math.max(maxY - minY, (groupNodes.length - 1) * 120);
      
      
      groupNodes.forEach((node, index) => {
        const nodeY = centerY - (range / 2) + (index * range / Math.max(groupNodes.length - 1, 1));
        layoutedNodes.push({
          ...node,
          x: layerX,
          y: nodeY,
        });
      });
    }
  }
}

function checkMiddleWorkload(
  node: NetworkNodeData,
  parentIds: string,
  edges: NetworkEdgeData[]
): boolean {
  
  const parents = parentIds.split(',');
  const nodeTargets = edges.filter(e => e.source === node.id).map(e => e.target);
  
  return parents.some(parentId => {
    const parentTargets = edges.filter(e => e.source === parentId).map(e => e.target);
    return nodeTargets.some(target => parentTargets.includes(target));
  });
}

function avoidOverlapInLayer(
  targetX: number,
  desiredY: number,
  existingNodes: Array<NetworkNodeData & Position>,
  minSpacing: number = 200
): number {
  
  const sameXNodes = existingNodes.filter(node => node.x === targetX);
  
  if (sameXNodes.length === 0) {
    return desiredY;
  }
  
  sameXNodes.sort((a, b) => a.y - b.y);
  
  let adjustedY = desiredY;
  
  for (const existingNode of sameXNodes) {
    const distance = Math.abs(adjustedY - existingNode.y);
    
    if (distance < minSpacing) {
      if (adjustedY >= existingNode.y) {
        adjustedY = existingNode.y + minSpacing;
      } else {
        adjustedY = existingNode.y - minSpacing;
      }
      
    }
  }
  
  if (adjustedY !== desiredY) {
    const recheck = sameXNodes.some(node => Math.abs(adjustedY - node.y) < minSpacing);
    if (recheck) {
      const maxY = Math.max(...sameXNodes.map(n => n.y));
      adjustedY = maxY + minSpacing;
    }
  }
  
  return adjustedY;
}