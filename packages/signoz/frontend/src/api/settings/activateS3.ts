import axios from 'api';
import { AxiosError } from 'axios';

export interface ActivateS3Response {
	status: string;
	job_id: string;
	message: string;
}

const activateS3 = async (mode: 'activate' | 'apply' = 'activate'): Promise<ActivateS3Response | null> => {
	try {
		const response = await axios.post<ActivateS3Response>(
			`/settings/s3/activate?mode=${mode}`,
		);
		return response.data;
	} catch (error) {
		const axiosError = error as AxiosError;
		throw axiosError;
	}
};

export default activateS3;
