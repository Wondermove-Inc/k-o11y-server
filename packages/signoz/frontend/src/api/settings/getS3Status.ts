import axios from 'api';
import { AxiosError } from 'axios';

export interface S3Status {
	s3_configured: boolean;
	s3_active: boolean;
	pending_restart: boolean;
	activation_status: 'idle' | 'running' | 'success' | 'failed';
	job_name?: string;
	job_message?: string;
}

const getS3Status = async (): Promise<S3Status | null> => {
	try {
		const response = await axios.get<S3Status>('/settings/s3/status');
		return response.data;
	} catch (error) {
		const axiosError = error as AxiosError;
		if (axiosError.response?.status === 404) {
			return null;
		}
		return null;
	}
};

export default getS3Status;
