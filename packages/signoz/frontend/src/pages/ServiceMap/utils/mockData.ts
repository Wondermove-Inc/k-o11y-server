// 백엔드 DTO 구조에 맞춘 Mock 데이터
import { NetworkNodeData, NetworkEdgeData } from '../types';

export const mockNetworkData = {
  nodes: [
    {
      id: "<YOUR_CLUSTER>$demo$emailservice",
      workloadName: "emailservice",
      namespace: "demo",
      cluster: "<YOUR_CLUSTER>",
      issueCount: 0,
      isExternal: 0,
      type: "workload",
      status: "Ok"
    },
    {
      id: "<YOUR_CLUSTER>$demo$checkoutservice",
      workloadName: "checkoutservice",
      namespace: "demo",
      cluster: "<YOUR_CLUSTER>",
      issueCount: 0,
      isExternal: 0,
      type: "workload",
      status: "Ok"
    },
    {
      id: "<YOUR_CLUSTER>$demo$cartservice",
      workloadName: "cartservice",
      namespace: "demo",
      cluster: "<YOUR_CLUSTER>",
      issueCount: 0,
      isExternal: 0,
      type: "workload",
      status: "Ok"
    },
    {
      id: "<YOUR_CLUSTER>$demo$frontend",
      workloadName: "frontend",
      namespace: "demo",
      cluster: "<YOUR_CLUSTER>",
      issueCount: 0,
      isExternal: 0,
      type: "workload",
      status: "Ok"
    },
    {
      id: "<YOUR_CLUSTER>$demo$recommendationservice",
      workloadName: "recommendationservice",
      namespace: "demo",
      cluster: "<YOUR_CLUSTER>",
      issueCount: 0,
      isExternal: 0,
      type: "workload",
      status: "Ok"
    }
  ] as NetworkNodeData[],

  edges: [
    {
      id: "demo$checkoutservice##demo$cartservice##gRPC",
      source: "<YOUR_CLUSTER>$demo$checkoutservice",
      destination: "<YOUR_CLUSTER>$demo$cartservice",
      protocol: "gRPC",
      isError: false,
      isExternal: 0
    },
    {
      id: "demo$frontend##demo$checkoutservice##gRPC",
      source: "<YOUR_CLUSTER>$demo$frontend",
      destination: "<YOUR_CLUSTER>$demo$checkoutservice",
      protocol: "gRPC",
      isError: false,
      isExternal: 0
    },
    {
      id: "demo$checkoutservice##demo$emailservice##gRPC",
      source: "<YOUR_CLUSTER>$demo$checkoutservice",
      destination: "<YOUR_CLUSTER>$demo$emailservice",
      protocol: "gRPC",
      isError: false,
      isExternal: 0
    },
    {
      id: "demo$frontend##demo$recommendationservice##gRPC",
      source: "<YOUR_CLUSTER>$demo$frontend",
      destination: "<YOUR_CLUSTER>$demo$recommendationservice",
      protocol: "gRPC",
      isError: false,
      isExternal: 0
    }
  ] as NetworkEdgeData[]
};