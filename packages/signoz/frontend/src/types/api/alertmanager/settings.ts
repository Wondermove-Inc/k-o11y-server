export interface AlertmanagerRouteSettings {
	group_wait: string;
	group_interval: string;
	repeat_interval: string;
}

export interface AlertmanagerSMTPAuthSettingsResponse {
	username: string;
	identity: string;
	password_set: boolean;
	secret_set: boolean;
}

export interface AlertmanagerSMTPTLSSettings {
	enabled: boolean;
	insecure_skip_verify: boolean;
	ca_file_path: string;
	cert_file_path: string;
	key_file_path: string;
}

export interface AlertmanagerSMTPSettingsResponse {
	address: string;
	from: string;
	hello: string;
	require_tls: boolean;
	auth: AlertmanagerSMTPAuthSettingsResponse;
	tls: AlertmanagerSMTPTLSSettings;
}

export interface AlertmanagerSettingsResponse {
	route: AlertmanagerRouteSettings;
	smtp: AlertmanagerSMTPSettingsResponse;
}

export interface AlertmanagerRouteSettingsUpdate {
	group_wait?: string;
	group_interval?: string;
	repeat_interval?: string;
}

export interface AlertmanagerSMTPAuthSettingsUpdate {
	username?: string;
	password?: string;
	secret?: string;
	identity?: string;
	clear_password?: boolean;
	clear_secret?: boolean;
}

export interface AlertmanagerSMTPTLSSettingsUpdate {
	enabled?: boolean;
	insecure_skip_verify?: boolean;
	ca_file_path?: string;
	cert_file_path?: string;
	key_file_path?: string;
}

export interface AlertmanagerSMTPSettingsUpdate {
	address?: string;
	from?: string;
	hello?: string;
	require_tls?: boolean;
	auth?: AlertmanagerSMTPAuthSettingsUpdate;
	tls?: AlertmanagerSMTPTLSSettingsUpdate;
}

export interface AlertmanagerSettingsUpdate {
	route?: AlertmanagerRouteSettingsUpdate;
	smtp?: AlertmanagerSMTPSettingsUpdate;
}
