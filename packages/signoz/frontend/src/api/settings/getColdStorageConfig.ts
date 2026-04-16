import axios from 'api';
import { AxiosError } from 'axios';
import { ErrorResponseHandler } from 'api/ErrorResponseHandler';

export interface ColdStorageConfig {
	signal_type: string;
	glacier_enabled: number;
	glacier_retention_days: number;
	backup_frequency_hours: number;
	min_delete_retention_days: number;
	updated_by: string;
	updated_at: string;
}

const getColdStorageConfig = async (): Promise<ColdStorageConfig | null> => {
	try {
		const response = await axios.get<ColdStorageConfig>(
			'/settings/cold-storage',
		);
		return response.data;
	} catch (error) {
		const axiosError = error as AxiosError;
		// Return null if cold storage is not configured (404 or empty response)
		if (axiosError.response?.status === 404) {
			return null;
		}
		ErrorResponseHandler(error as AxiosError);
		return null;
	}
};

export default getColdStorageConfig;
