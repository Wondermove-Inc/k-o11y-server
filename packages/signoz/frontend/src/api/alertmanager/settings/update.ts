import axios from 'api';
import { ErrorResponseHandlerV2 } from 'api/ErrorResponseHandlerV2';
import { AxiosError } from 'axios';
import { ErrorV2Resp, SuccessResponseV2 } from 'types/api';
import {
	AlertmanagerSettingsResponse,
	AlertmanagerSettingsUpdate,
} from 'types/api/alertmanager/settings';

const updateAlertmanagerSettings = async (
	payload: AlertmanagerSettingsUpdate,
): Promise<SuccessResponseV2<AlertmanagerSettingsResponse>> => {
	try {
		const response = await axios.put<{
			data: AlertmanagerSettingsResponse;
		}>('/alertmanager/settings', payload);

		return {
			httpStatusCode: response.status,
			data: response.data.data,
		};
	} catch (error) {
		ErrorResponseHandlerV2(error as AxiosError<ErrorV2Resp>);
	}
};

export default updateAlertmanagerSettings;
