import axios from 'api';
import { AxiosError } from 'axios';

export interface TestS3ConnectionProps {
	auth_mode: string;
	bucket: string;
	region: string;
	access_key_id?: string;
	secret_access_key?: string;
}

export interface TestS3ConnectionResponse {
	success: boolean;
	message: string;
}

const testS3Connection = async (
	type: 'warm' | 'cold',
	props: TestS3ConnectionProps,
): Promise<TestS3ConnectionResponse> => {
	try {
		const response = await axios.post<TestS3ConnectionResponse>(
			`/settings/s3/${type}/test`,
			props,
		);
		return response.data;
	} catch (error) {
		const axiosError = error as AxiosError;
		return {
			success: false,
			message: axiosError.response?.data
				? String((axiosError.response.data as any).error || 'Connection failed')
				: 'Connection failed',
		};
	}
};

export default testS3Connection;
