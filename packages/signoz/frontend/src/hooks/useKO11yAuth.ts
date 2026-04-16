import { useEffect } from 'react';

import afterLogin from 'AppRoutes/utils';
import getLocalStorageApi from 'api/browser/localstorage/get';
import setLocalStorageApi from 'api/browser/localstorage/set';
import { LOCALSTORAGE } from 'constants/localStorage';

/**
 * IPC message types sent from Freelens host via electronAPI.
 */
interface KO11yAuthMessage {
	type: 'KO11Y_LOGIN' | 'KO11Y_LOGOUT' | 'KO11Y_TOKEN_REFRESH';
	token?: string;
}

/**
 * useKO11yAuth listens for K-O11y IPC auth events from the Freelens
 * Electron host. When a login event is received, it stores the K-O11y JWT
 * in localStorage so the Axios interceptor can send it as a Bearer token.
 *
 * This hook is safe to mount even when electronAPI is not available
 * (standard browser environment).
 */
function useKO11yAuth(): void {
	useEffect(() => {
		const electronAPI = (window as ElectronWindow).electronAPI;

		// Not running inside Electron/Freelens — nothing to do
		if (!electronAPI?.on) {
			return undefined;
		}

		const handleAuthMessage = (_event: unknown, message: KO11yAuthMessage): void => {
			switch (message.type) {
				case 'KO11Y_LOGIN':
				case 'KO11Y_TOKEN_REFRESH':
					if (message.token) {
						setLocalStorageApi(LOCALSTORAGE.AUTH_TOKEN, message.token);
						setLocalStorageApi(LOCALSTORAGE.IS_LOGGED_IN, 'true');

						// Only call afterLogin on initial login (not refresh)
						if (message.type === 'KO11Y_LOGIN') {
							afterLogin(message.token, '');
						}
					}
					break;

				case 'KO11Y_LOGOUT':
					setLocalStorageApi(LOCALSTORAGE.AUTH_TOKEN, '');
					setLocalStorageApi(LOCALSTORAGE.REFRESH_AUTH_TOKEN, '');
					setLocalStorageApi(LOCALSTORAGE.IS_LOGGED_IN, 'false');
					break;

				default:
					break;
			}
		};

		electronAPI.on('ko11y-auth', handleAuthMessage);

		// Request current auth state on mount (in case app loaded after login)
		if (electronAPI.send && !getLocalStorageApi(LOCALSTORAGE.AUTH_TOKEN)) {
			electronAPI.send('ko11y-auth-request', {});
		}

		return (): void => {
			if (electronAPI.removeListener) {
				electronAPI.removeListener('ko11y-auth', handleAuthMessage);
			}
		};
	}, []);
}

export default useKO11yAuth;
