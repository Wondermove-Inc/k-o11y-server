import { useAppContext } from 'providers/App/App';
import { LicensePlatform } from 'types/api/licensesV3/getActive';

export const useGetTenantLicense = (): {
	isCloudUser: boolean;
	isEnterpriseSelfHostedUser: boolean;
	isCommunityUser: boolean;
	isCommunityEnterpriseUser: boolean;
} => {
	const { activeLicense, activeLicenseFetchError } = useAppContext();

	const responsePayload = {
		isCloudUser: activeLicense?.platform === LicensePlatform.CLOUD || false,
		isEnterpriseSelfHostedUser:
			activeLicense?.platform === LicensePlatform.SELF_HOSTED || false,
		isCommunityUser: false,
		isCommunityEnterpriseUser: false,
	};

	const statusCode =
		activeLicenseFetchError &&
		typeof activeLicenseFetchError.getHttpStatusCode === 'function'
			? activeLicenseFetchError.getHttpStatusCode()
			: null;

	if (statusCode === 404) {
		responsePayload.isCommunityEnterpriseUser = true;
	}

	if (statusCode === 501 || (activeLicenseFetchError && statusCode === null)) {
		responsePayload.isCommunityUser = true;
	}

	return responsePayload;
};
