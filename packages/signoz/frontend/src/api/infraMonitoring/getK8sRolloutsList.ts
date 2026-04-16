import axios from 'api';
import { ErrorResponseHandler } from 'api/ErrorResponseHandler';
import { AxiosError } from 'axios';
import { ErrorResponse, SuccessResponse } from 'types/api';
import { BaseAutocompleteData } from 'types/api/queryBuilder/queryAutocompleteResponse';
import { TagFilter } from 'types/api/queryBuilder/queryBuilderData';

import { UnderscoreToDotMap } from '../utils';

export interface K8sRolloutsListPayload {
	filters: TagFilter;
	groupBy?: BaseAutocompleteData[];
	offset?: number;
	limit?: number;
	orderBy?: {
		columnName: string;
		order: 'asc' | 'desc';
	};
}

export interface K8sRolloutsData {
	rolloutName: string;
	cpuUsage: number;
	memoryUsage: number;
	cpuRequest: number;
	memoryRequest: number;
	cpuLimit: number;
	memoryLimit: number;
	restarts: number;
	meta: {
		k8s_cluster_name: string;
		k8s_rollout_name: string;
		k8s_namespace_name: string;
	};
}

export interface K8sRolloutsListResponse {
	status: string;
	data: {
		type: string;
		records: K8sRolloutsData[];
		groups: null;
		total: number;
		sentAnyHostMetricsData: boolean;
		isSendingK8SAgentMetrics: boolean;
	};
}

export const rolloutsMetaMap = [
	{ dot: 'k8s.cluster.name', under: 'k8s_cluster_name' },
	{ dot: 'k8s.rollout.name', under: 'k8s_rollout_name' },
	{ dot: 'k8s.namespace.name', under: 'k8s_namespace_name' },
] as const;

export function mapRolloutsMeta(
	raw: Record<string, unknown>,
): K8sRolloutsData['meta'] {
	const out: Record<string, unknown> = { ...raw };
	rolloutsMetaMap.forEach(({ dot, under }) => {
		if (dot in raw) {
			const v = raw[dot];
			out[under] = typeof v === 'string' ? v : raw[under];
		}
	});
	return out as K8sRolloutsData['meta'];
}

export const getK8sRolloutsList = async (
	props: K8sRolloutsListPayload,
	signal?: AbortSignal,
	headers?: Record<string, string>,
	dotMetricsEnabled = false,
): Promise<SuccessResponse<K8sRolloutsListResponse> | ErrorResponse> => {
	try {
		const requestProps =
			dotMetricsEnabled && Array.isArray(props.filters?.items)
				? {
						...props,
						filters: {
							...props.filters,
							items: props.filters.items.reduce<typeof props.filters.items>(
								(acc, item) => {
									if (item.value === undefined) return acc;
									if (
										item.key &&
										typeof item.key === 'object' &&
										'key' in item.key &&
										typeof item.key.key === 'string'
									) {
										const mappedKey = UnderscoreToDotMap[item.key.key] ?? item.key.key;
										acc.push({
											...item,
											key: { ...item.key, key: mappedKey },
										});
									} else {
										acc.push(item);
									}
									return acc;
								},
								[] as typeof props.filters.items,
							),
						},
				  }
				: props;

		const response = await axios.post('/rollouts/list', requestProps, {
			signal,
			headers,
		});
		const payload: K8sRolloutsListResponse = response.data;

		payload.data.records = payload.data.records.map((record) => ({
			...record,
			meta: mapRolloutsMeta(record.meta as Record<string, unknown>),
		}));

		return {
			statusCode: 200,
			error: null,
			message: 'Success',
			payload,
			params: requestProps,
		};
	} catch (error) {
		return ErrorResponseHandler(error as AxiosError);
	}
};
