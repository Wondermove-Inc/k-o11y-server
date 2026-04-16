import axios from 'api';
import { ErrorResponseHandlerV2 } from 'api/ErrorResponseHandlerV2';
import { AxiosError } from 'axios';
import { ErrorV2Resp, SuccessResponseV2 } from 'types/api';

export interface UpdateS3ConfigProps {
	auth_mode?: string;
	bucket?: string;
	region?: string;
	endpoint?: string;
	access_key_id?: string;
	secret_access_key?: string;
	s3_enabled?: number;
}

interface UpdateS3ConfigResponse {
	message: string;
}

const updateS3Config = async (
	type: 'warm' | 'cold',
	props: UpdateS3ConfigProps,
): Promise<SuccessResponseV2<UpdateS3ConfigResponse>> => {
	try {
		const response = await axios.put<UpdateS3ConfigResponse>(
			`/settings/s3/${type}`,
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

export default updateS3Config;
