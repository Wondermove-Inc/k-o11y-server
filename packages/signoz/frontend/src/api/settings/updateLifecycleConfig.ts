import axios from 'api';
import { ErrorResponseHandlerV2 } from 'api/ErrorResponseHandlerV2';
import { AxiosError } from 'axios';
import { ErrorV2Resp, SuccessResponseV2 } from 'types/api';

export interface UpdateLifecycleConfigProps {
	hot_days?: number;
	warm_days?: number;
	glacier_enabled?: number;
	glacier_retention_days?: number;
}

interface UpdateLifecycleConfigResponse {
	message: string;
}

const updateLifecycleConfig = async (
	props: UpdateLifecycleConfigProps,
): Promise<SuccessResponseV2<UpdateLifecycleConfigResponse>> => {
	try {
		const response = await axios.put<UpdateLifecycleConfigResponse>(
			'/settings/lifecycle',
			props,
		);

		return {
			httpStatusCode: response.status,
			data: response.data,
		};
	} catch (error) {
		ErrorResponseHandlerV2(error as AxiosError<ErrorV2Resp>);
	}
};

export default updateLifecycleConfig;
