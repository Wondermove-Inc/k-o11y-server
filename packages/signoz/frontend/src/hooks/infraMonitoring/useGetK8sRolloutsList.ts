import {
	getK8sRolloutsList,
	K8sRolloutsListPayload,
	K8sRolloutsListResponse,
} from 'api/infraMonitoring/getK8sRolloutsList';
import { REACT_QUERY_KEY } from 'constants/reactQueryKeys';
import { useMemo } from 'react';
import { useQuery, UseQueryOptions, UseQueryResult } from 'react-query';
import { ErrorResponse, SuccessResponse } from 'types/api';

type UseGetK8sRolloutsList = (
	requestData: K8sRolloutsListPayload,
	options?: UseQueryOptions<
		SuccessResponse<K8sRolloutsListResponse> | ErrorResponse,
		Error
	>,
	headers?: Record<string, string>,
	dotMetricsEnabled?: boolean,
) => UseQueryResult<
	SuccessResponse<K8sRolloutsListResponse> | ErrorResponse,
	Error
>;

export const useGetK8sRolloutsList: UseGetK8sRolloutsList = (
	requestData,
	options,
	headers,
	dotMetricsEnabled,
) => {
	const queryKey = useMemo(() => {
		if (options?.queryKey && Array.isArray(options.queryKey)) {
			return [...options.queryKey];
		}

		if (options?.queryKey && typeof options.queryKey === 'string') {
			return options.queryKey;
		}

		return [REACT_QUERY_KEY.GET_ROLLOUT_LIST, requestData];
	}, [options?.queryKey, requestData]);

	return useQuery<
		SuccessResponse<K8sRolloutsListResponse> | ErrorResponse,
		Error
	>({
		queryFn: ({ signal }) =>
			getK8sRolloutsList(requestData, signal, headers, dotMetricsEnabled),
		...options,
		queryKey,
	});
};
