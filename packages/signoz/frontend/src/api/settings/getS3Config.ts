import axios from 'api';
import { AxiosError } from 'axios';
import { ErrorResponseHandler } from 'api/ErrorResponseHandler';

export interface S3Config {
	config_id: string;
	auth_mode: string;
	bucket: string;
	region: string;
	endpoint: string;
	access_key_id: string;
	secret_access_key: string;
	s3_enabled: number;
	connection_tested: number;
	connection_tested_at: string;
	updated_by: string;
	updated_at: string;
	version: number;
}

const getS3Config = async (type: 'warm' | 'cold' = 'warm'): Promise<S3Config | null> => {
	try {
		const response = await axios.get<S3Config>(`/settings/s3/${type}`);
		if ((response.data as any).status === 'not_configured') {
			return null;
		}
		return response.data;
	} catch (error) {
		const axiosError = error as AxiosError;
		if (axiosError.response?.status === 404) {
			return null;
		}
		ErrorResponseHandler(error as AxiosError);
		return null;
	}
};

export default getS3Config;
