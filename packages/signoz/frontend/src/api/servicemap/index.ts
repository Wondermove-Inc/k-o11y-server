import { AxiosError } from 'axios';
import { NetworkTopologyData, TimeRange, NetworkFilters, NetworkNode, NetworkEdge, NetworkNodeData, NetworkEdgeData, WorkloadDetailRequest, WorkloadDetailResponse, WorkloadHoverRequest, WorkloadHoverResponse, EdgeTraceDetailRequest, EdgeTraceDetailResponse } from '../../pages/ServiceMap/types';
import axios from 'axios';
import { ENVIRONMENT } from 'constants/env';

const transformToReactFlow = (backendData: any): NetworkTopologyData => {
  if (!backendData || !backendData.nodes || !backendData.edges) {
    return {
      nodes: [],
      edges: [],
      timeRange: backendData?.timeRange || ''
    };
  }

  const nodes: NetworkNode[] = backendData.nodes.map((nodeData: NetworkNodeData, index: number) => ({
    id: nodeData.id,
    type: 'networkNode',
    position: { x: index * 200, y: index * 100 },
    data: nodeData,
    draggable: true,
    selectable: true
  }));

  const edges: NetworkEdge[] = backendData.edges.map((edgeData: NetworkEdgeData) => ({
    id: edgeData.id,
    source: edgeData.source,
    target: edgeData.destination,
    type: 'networkEdge',
    data: edgeData,
    markerEnd: {
      type: 'arrowclosed' as const,
      width: 20,
      height: 20,
      color: edgeData.isError ? '#ef4444' : '#64748b'
    },
    style: {
      stroke: edgeData.isError ? '#ef4444' : '#64748b',
      strokeWidth: 2
    }
  }));


  return {
    nodes,
    edges,
    timeRange: backendData.timeRange || ''
  };
};

export const getTopologyData = async (timeRange: TimeRange, filters?: NetworkFilters): Promise<NetworkTopologyData | null> => {
  const requestData = {
    startTime: timeRange.start,
    endTime: timeRange.end,
    cluster: filters?.clusters || [],
    namespace: filters?.namespaces || [],
    protocol: filters?.protocols || [],
    status: filters?.connectionStatuses || [],
    workload: filters?.workloads || []
  };

  console.log('[API PARAM] getTopologyData : ', requestData);
  return await axios
    .post(`${ENVIRONMENT.ko11yURL}/api/v1/servicemap/topology`, requestData)
    .then((res) => {
      console.log('[API SUCCEED] getTopologyData : ', res);
      if (res?.data?.result) {
        const reactFlowData = transformToReactFlow(res.data.result);
        // 원본 백엔드 데이터도 함께 저장
        (reactFlowData as any).originalBackendData = res.data.result;
        return reactFlowData;
      }
      return null;
    })
    .catch((error: AxiosError) => {
      console.error('[API ERROR] getTopologyData : ', error);
      return null;
    });
};

export const getServiceDetails = async (serviceName: string, namespace: string, cluster: string) => {
  const requestData = {
    serviceName: serviceName,
    namespace: namespace,
    cluster: cluster,
    timeRange: '24h'
  };

  console.log('[API PARAM] getServiceDetails : ', requestData);
  return await axios
    .post(`${ENVIRONMENT.ko11yURL}/api/v1/servicemap/service/details`, requestData)
    .then((res) => {
      console.log('[API SUCCEED] getServiceDetails : ', res);
      if (res?.data?.result) return res.data.result;
    })
    .catch((error: AxiosError) =>
      console.error('[API ERROR] getServiceDetails : ', error),
    );
};

export const getConnectionDetails = async (connectionId: string) => {
  const requestData = {
    connectionId: connectionId,
    timeRange: '24h'
  };

  console.log('[API PARAM] getConnectionDetails : ', requestData);
  return await axios
    .post(`${ENVIRONMENT.ko11yURL}/api/v1/servicemap/connection/details`, requestData)
    .then((res) => {
      console.log('[API SUCCEED] getConnectionDetails : ', res);
      if (res?.data?.result) return res.data.result;
    })
    .catch((error: AxiosError) =>
      console.error('[API ERROR] getConnectionDetails : ', error),
    );
};

export const getWorkloadHoverInfo = async (requestData: WorkloadHoverRequest): Promise<WorkloadHoverResponse | null> => {

  console.log('[API PARAM] getWorkloadHoverInfo : ', requestData);
  return await axios
    .post(`${ENVIRONMENT.ko11yURL}/api/v1/servicemap/workload/hover-info`, requestData)
    .then((res) => {
      console.log('[API SUCCEED] getWorkloadHoverInfo : ', res);
      if (res?.data?.result) return res.data.result;
      return null;
    })
    .catch((error: AxiosError) => {
      console.error('[API ERROR] getWorkloadHoverInfo : ', error);
      return null;
    });
};

export const getWorkloadDetails = async (requestData: WorkloadDetailRequest): Promise<WorkloadDetailResponse | null> => {
  console.log('[API PARAM] getWorkloadDetails : ', requestData);
  return await axios
    .post(`${ENVIRONMENT.ko11yURL}/api/v1/servicemap/workload/details`, requestData)
    .then((res) => {
      console.log('[API SUCCEED] getWorkloadDetails : ', res);

      if (res?.data?.result) {
        return res.data.result;
      }

      return null;
    })
    .catch((error: AxiosError) => {
      console.error('[API ERROR] getWorkloadDetails : ', error);
      return null;
    });
};

export const getEdgeTraceDetails = async (requestData: EdgeTraceDetailRequest & { cursor?: string }): Promise<EdgeTraceDetailResponse | null> => {
  console.log('[API PARAM] getEdgeTraceDetails : ', requestData);
  return await axios
    .post(`${ENVIRONMENT.ko11yURL}/api/v1/servicemap/edge/trace/details`, requestData)
    .then((res) => {
      console.log('[API SUCCEED] getEdgeTraceDetails : ', res);

      if (res?.data?.result) {
        return res.data.result;
      }

      return null;
    })
    .catch((error: AxiosError) => {
      console.error('[API ERROR] getEdgeTraceDetails : ', error);
      return null;
    });
};