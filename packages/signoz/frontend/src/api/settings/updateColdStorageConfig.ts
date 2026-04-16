import axios from 'api';
import { ErrorResponseHandlerV2 } from 'api/ErrorResponseHandlerV2';
import { AxiosError } from 'axios';
import { ErrorV2Resp, SuccessResponseV2 } from 'types/api';

export interface UpdateColdStorageConfigProps {
	glacier_enabled?: number;
	glacier_retention_days?: number;
	backup_frequency_hours?: number;
	min_delete_retention_days?: number;
}

interface UpdateColdStorageConfigResponse {
	message: string;
}

const updateColdStorageConfig = async (
	props: UpdateColdStorageConfigProps,
): Promise<SuccessResponseV2<UpdateColdStorageConfigResponse>> => {
	try {
		const response = await axios.put<UpdateColdStorageConfigResponse>(
			'/settings/cold-storage',
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

export default updateColdStorageConfig;
