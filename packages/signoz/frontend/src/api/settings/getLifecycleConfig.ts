import axios from 'api';
import { AxiosError } from 'axios';
import { ErrorResponseHandler } from 'api/ErrorResponseHandler';

export interface DataLifecycleConfig {
	signal_type: string;
	hot_days: number;
	warm_days: number;
	glacier_enabled: number;
	glacier_retention_days: number;
	backup_frequency_hours: number;
	last_backup_status: string;
	last_backup_at: string;
	last_backup_error: string;
	updated_by: string;
	updated_at: string;
	version: number;
}

const getLifecycleConfig = async (): Promise<DataLifecycleConfig | null> => {
	try {
		const response = await axios.get<DataLifecycleConfig>(
			'/settings/lifecycle',
		);
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

export default getLifecycleConfig;
