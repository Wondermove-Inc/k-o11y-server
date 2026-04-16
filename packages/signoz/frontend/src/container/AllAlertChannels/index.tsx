import './AllAlertChannels.styles.scss';

import { ExclamationCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { Input, InputNumber, Select, Switch, Tooltip, Typography } from 'antd';
import getAlertmanagerSettings from 'api/alertmanager/settings/get';
import updateAlertmanagerSettings from 'api/alertmanager/settings/update';
import getAll from 'api/channels/getAll';
import logEvent from 'api/common/logEvent';
import getOrgPreference from 'api/v1/org/preferences/name/get';
import updateOrgPreference from 'api/v1/org/preferences/name/update';
import Spinner from 'components/Spinner';
import { ORG_PREFERENCES } from 'constants/orgPreferences';
import ROUTES from 'constants/routes';
import useComponentPermission from 'hooks/useComponentPermission';
import { useNotifications } from 'hooks/useNotifications';
import history from 'lib/history';
import { isUndefined } from 'lodash-es';
import { useAppContext } from 'providers/App/App';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMutation, useQuery } from 'react-query';
import { SuccessResponseV2 } from 'types/api';
import {
	AlertmanagerSMTPAuthSettingsUpdate,
	AlertmanagerSettingsResponse,
	AlertmanagerSettingsUpdate,
} from 'types/api/alertmanager/settings';
import { Channels } from 'types/api/channels/getAll';
import APIError from 'types/api/error';

import AlertChannelsComponent from './AlertChannels';
import {
	DurationInput,
	DurationUnit,
	formatDuration,
	parseDuration,
} from './alertmanagerDuration';
import TextToolTip from '../../components/TextToolTip/TextToolTip';
import { Button, ButtonContainer, RightActionContainer } from './styles';

const { Paragraph } = Typography;
const { Option } = Select;

function AlertChannels(): JSX.Element {
	const { t } = useTranslation(['channels']);
	const { user } = useAppContext();
	const { notifications } = useNotifications();
	const [addNewChannelPermission] = useComponentPermission(
		['add_new_channel'],
		user.role,
	);
	const [alertBaseURL, setAlertBaseURL] = useState<string>('');
	const [alertBaseURLError, setAlertBaseURLError] = useState<string | null>(null);
	const [alertmanagerSettings, setAlertmanagerSettings] =
		useState<AlertmanagerSettingsResponse | null>(null);
	const [alertmanagerSettingsError, setAlertmanagerSettingsError] = useState<
		string | null
	>(null);
	const [routeInputs, setRouteInputs] = useState<{
		groupWait: DurationInput;
		groupInterval: DurationInput;
		repeatInterval: DurationInput;
	}>({
		groupWait: { value: 0, unit: 's' },
		groupInterval: { value: 0, unit: 'm' },
		repeatInterval: { value: 0, unit: 'h' },
	});
	const [smtpPassword, setSmtpPassword] = useState<string>('');
	const [smtpSecret, setSmtpSecret] = useState<string>('');

	const onToggleHandler = useCallback(() => {
		history.push(ROUTES.CHANNELS_NEW);
	}, []);

	const { isLoading, data, error } = useQuery<
		SuccessResponseV2<Channels[]>,
		APIError
	>(['getChannels'], {
		queryFn: () => getAll(),
	});

	const {
		data: alertBaseURLData,
		isLoading: isAlertBaseURLLoading,
		refetch: refetchAlertBaseURL,
	} = useQuery({
		queryKey: ['orgPreference', ORG_PREFERENCES.ALERT_BASE_URL],
		queryFn: () =>
			getOrgPreference({ name: ORG_PREFERENCES.ALERT_BASE_URL }),
		refetchOnWindowFocus: false,
	});

	const {
		data: alertmanagerSettingsData,
		isLoading: isAlertmanagerSettingsLoading,
		refetch: refetchAlertmanagerSettings,
	} = useQuery({
		queryKey: ['alertmanagerSettings'],
		queryFn: () => getAlertmanagerSettings(),
		refetchOnWindowFocus: false,
		enabled: addNewChannelPermission,
	});

	const storedAlertBaseURL = useMemo(() => {
		const value = alertBaseURLData?.data?.value;
		const defaultValue = alertBaseURLData?.data?.defaultValue;
		if (typeof value === 'string') {
			return value;
		}
		if (typeof defaultValue === 'string') {
			return defaultValue;
		}
		return '';
	}, [alertBaseURLData?.data?.defaultValue, alertBaseURLData?.data?.value]);

	useEffect(() => {
		if (storedAlertBaseURL) {
			setAlertBaseURL(storedAlertBaseURL);
		}
	}, [storedAlertBaseURL]);

	useEffect(() => {
		if (alertmanagerSettingsData?.data) {
			setAlertmanagerSettings(alertmanagerSettingsData.data);
			setRouteInputs({
				groupWait: parseDuration(alertmanagerSettingsData.data.route.group_wait),
				groupInterval: parseDuration(
					alertmanagerSettingsData.data.route.group_interval,
				),
				repeatInterval: parseDuration(
					alertmanagerSettingsData.data.route.repeat_interval,
				),
			});
			setSmtpPassword('');
			setSmtpSecret('');
		}
	}, [alertmanagerSettingsData?.data]);

	const validateAlertBaseURL = useCallback(
		(value: string): string | null => {
			const trimmed = value.trim();
			if (!trimmed) {
				return t('alert_base_url_invalid');
			}

			try {
				const parsed = new URL(trimmed);
				if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
					return t('alert_base_url_invalid');
				}
				if (parsed.search || parsed.hash) {
					return t('alert_base_url_invalid');
				}
				if (parsed.pathname && parsed.pathname !== '/') {
					return t('alert_base_url_invalid');
				}
			} catch {
				return t('alert_base_url_invalid');
			}

			return null;
		},
		[t],
	);

	const normalizeAlertBaseURL = useCallback((value: string): string => {
		return value.trim().replace(/\/$/, '');
	}, []);

	const previewURL = useMemo(() => {
		if (!alertBaseURL) {
			return '';
		}
		return `${normalizeAlertBaseURL(alertBaseURL)}/alerts`;
	}, [alertBaseURL, normalizeAlertBaseURL]);

	const updateRouteInput = useCallback(
		(
			field: 'groupWait' | 'groupInterval' | 'repeatInterval',
			next: DurationInput,
		) => {
			setRouteInputs((prev) => ({
				...prev,
				[field]: next,
			}));
			setAlertmanagerSettingsError(null);
		},
		[],
	);

	const updateAlertmanagerSMTPText = useCallback(
		(field: 'address' | 'from' | 'hello', value: string) => {
			setAlertmanagerSettings((prev) => {
				if (!prev) {
					return prev;
				}
				return {
					...prev,
					smtp: {
						...prev.smtp,
						[field]: value,
					},
				};
			});
			setAlertmanagerSettingsError(null);
		},
		[],
	);

	const updateAlertmanagerSMTPRequireTLS = useCallback((value: boolean) => {
		setAlertmanagerSettings((prev) => {
			if (!prev) {
				return prev;
			}
			return {
				...prev,
				smtp: {
					...prev.smtp,
					require_tls: value,
				},
			};
		});
		setAlertmanagerSettingsError(null);
	}, []);

	const updateAlertmanagerSMTPAuth = useCallback(
		(field: 'username' | 'identity', value: string) => {
			setAlertmanagerSettings((prev) => {
				if (!prev) {
					return prev;
				}
				return {
					...prev,
					smtp: {
						...prev.smtp,
						auth: {
							...prev.smtp.auth,
							[field]: value,
						},
					},
				};
			});
			setAlertmanagerSettingsError(null);
		},
		[],
	);

	const updateAlertmanagerSMTPTLSBoolean = useCallback(
		(field: 'enabled' | 'insecure_skip_verify', value: boolean) => {
			setAlertmanagerSettings((prev) => {
				if (!prev) {
					return prev;
				}
				return {
					...prev,
					smtp: {
						...prev.smtp,
						tls: {
							...prev.smtp.tls,
							[field]: value,
						},
					},
				};
			});
			setAlertmanagerSettingsError(null);
		},
		[],
	);

	const updateAlertmanagerSMTPTLSText = useCallback(
		(
			field: 'ca_file_path' | 'cert_file_path' | 'key_file_path',
			value: string,
		) => {
			setAlertmanagerSettings((prev) => {
				if (!prev) {
					return prev;
				}
				return {
					...prev,
					smtp: {
						...prev.smtp,
						tls: {
							...prev.smtp.tls,
							[field]: value,
						},
					},
				};
			});
			setAlertmanagerSettingsError(null);
		},
		[],
	);

	const validateAlertmanagerSettings = useCallback(
		(settings: AlertmanagerSettingsResponse | null): string | null => {
			if (!settings) {
				return t('alertmanager_settings_invalid');
			}

			if (
				routeInputs.groupWait.value <= 0 ||
				routeInputs.groupInterval.value <= 0 ||
				routeInputs.repeatInterval.value <= 0
			) {
				return t('alertmanager_settings_duration_invalid');
			}

			const address = settings.smtp.address.trim();
			try {
				const parsed = new URL(`smtp://${address}`);
				if (!parsed.hostname || !parsed.port) {
					return t('alertmanager_settings_address_invalid');
				}
			} catch {
				return t('alertmanager_settings_address_invalid');
			}

			const tls = settings.smtp.tls;
			const tlsPathsSet =
				Boolean(tls.ca_file_path) ||
				Boolean(tls.cert_file_path) ||
				Boolean(tls.key_file_path);

			if (!tls.enabled && tlsPathsSet) {
				return t('alertmanager_settings_tls_invalid');
			}

			if (
				tls.enabled &&
				(Boolean(tls.cert_file_path) !== Boolean(tls.key_file_path))
			) {
				return t('alertmanager_settings_tls_invalid');
			}

			return null;
		},
		[routeInputs, t],
	);

	const { mutate: saveAlertBaseURL, isLoading: isSavingAlertBaseURL } =
		useMutation(updateOrgPreference, {
			onSuccess: () => {
				notifications.success({
					message: t('alert_base_url_saved'),
				});
				refetchAlertBaseURL();
			},
			onError: () => {
				notifications.error({
					message: t('alert_base_url_save_failed'),
				});
			},
		});

	const onSaveAlertBaseURL = useCallback(() => {
		const errorMessage = validateAlertBaseURL(alertBaseURL);
		if (errorMessage) {
			setAlertBaseURLError(errorMessage);
			return;
		}

		const normalizedValue = normalizeAlertBaseURL(alertBaseURL);
		setAlertBaseURLError(null);
		saveAlertBaseURL({
			name: ORG_PREFERENCES.ALERT_BASE_URL,
			value: normalizedValue,
		});
	}, [
		alertBaseURL,
		normalizeAlertBaseURL,
		saveAlertBaseURL,
		validateAlertBaseURL,
	]);

	const {
		mutate: saveAlertmanagerSettings,
		isLoading: isSavingAlertmanagerSettings,
	} = useMutation(updateAlertmanagerSettings, {
		onSuccess: () => {
			notifications.success({
				message: t('alertmanager_settings_saved'),
			});
			refetchAlertmanagerSettings();
			setSmtpPassword('');
			setSmtpSecret('');
		},
		onError: () => {
			notifications.error({
				message: t('alertmanager_settings_save_failed'),
			});
		},
	});

	const isAlertmanagerSettingsDirty = useMemo(() => {
		if (!alertmanagerSettings || !alertmanagerSettingsData?.data) {
			return false;
		}

		const stored = alertmanagerSettingsData.data;
		const groupWait = formatDuration(routeInputs.groupWait);
		const groupInterval = formatDuration(routeInputs.groupInterval);
		const repeatInterval = formatDuration(routeInputs.repeatInterval);

		if (groupWait !== stored.route.group_wait) {
			return true;
		}
		if (groupInterval !== stored.route.group_interval) {
			return true;
		}
		if (repeatInterval !== stored.route.repeat_interval) {
			return true;
		}
		if (alertmanagerSettings.smtp.address !== stored.smtp.address) {
			return true;
		}
		if (alertmanagerSettings.smtp.from !== stored.smtp.from) {
			return true;
		}
		if (alertmanagerSettings.smtp.hello !== stored.smtp.hello) {
			return true;
		}
		if (
			alertmanagerSettings.smtp.require_tls !== stored.smtp.require_tls
		) {
			return true;
		}
		if (
			alertmanagerSettings.smtp.auth.username !== stored.smtp.auth.username
		) {
			return true;
		}
		if (
			alertmanagerSettings.smtp.auth.identity !== stored.smtp.auth.identity
		) {
			return true;
		}
		if (alertmanagerSettings.smtp.tls.enabled !== stored.smtp.tls.enabled) {
			return true;
		}
		if (
			alertmanagerSettings.smtp.tls.insecure_skip_verify !==
			stored.smtp.tls.insecure_skip_verify
		) {
			return true;
		}
		if (
			alertmanagerSettings.smtp.tls.ca_file_path !==
			stored.smtp.tls.ca_file_path
		) {
			return true;
		}
		if (
			alertmanagerSettings.smtp.tls.cert_file_path !==
			stored.smtp.tls.cert_file_path
		) {
			return true;
		}
		if (
			alertmanagerSettings.smtp.tls.key_file_path !==
			stored.smtp.tls.key_file_path
		) {
			return true;
		}
		if (smtpPassword.trim() || smtpSecret.trim()) {
			return true;
		}

		return false;
	}, [
		alertmanagerSettings,
		alertmanagerSettingsData?.data,
		routeInputs,
		smtpPassword,
		smtpSecret,
	]);

	const onSaveAlertmanagerSettings = useCallback(() => {
		if (!alertmanagerSettings) {
			return;
		}

		const errorMessage = validateAlertmanagerSettings(alertmanagerSettings);
		if (errorMessage) {
			setAlertmanagerSettingsError(errorMessage);
			return;
		}

		const authPayload: AlertmanagerSMTPAuthSettingsUpdate = {
			username: alertmanagerSettings.smtp.auth.username,
			identity: alertmanagerSettings.smtp.auth.identity,
		};

		const password = smtpPassword.trim();
		if (password) {
			authPayload.password = password;
		}

		const secret = smtpSecret.trim();
		if (secret) {
			authPayload.secret = secret;
		}

		const payload: AlertmanagerSettingsUpdate = {
			route: {
				group_wait: formatDuration(routeInputs.groupWait),
				group_interval: formatDuration(routeInputs.groupInterval),
				repeat_interval: formatDuration(routeInputs.repeatInterval),
			},
			smtp: {
				address: alertmanagerSettings.smtp.address,
				from: alertmanagerSettings.smtp.from,
				hello: alertmanagerSettings.smtp.hello,
				require_tls: alertmanagerSettings.smtp.require_tls,
				auth: authPayload,
				tls: {
					enabled: alertmanagerSettings.smtp.tls.enabled,
					insecure_skip_verify:
						alertmanagerSettings.smtp.tls.insecure_skip_verify,
					ca_file_path: alertmanagerSettings.smtp.tls.ca_file_path,
					cert_file_path: alertmanagerSettings.smtp.tls.cert_file_path,
					key_file_path: alertmanagerSettings.smtp.tls.key_file_path,
				},
			},
		};

		setAlertmanagerSettingsError(null);
		saveAlertmanagerSettings(payload);
	}, [
		alertmanagerSettings,
		routeInputs,
		saveAlertmanagerSettings,
		smtpPassword,
		smtpSecret,
		validateAlertmanagerSettings,
	]);

	useEffect(() => {
		if (!isUndefined(data?.data)) {
			logEvent('Alert Channel: Channel list page visited', {
				number: data?.data?.length,
			});
		}
	}, [data?.data]);

	if (error) {
		return <Typography>{error.getErrorMessage()}</Typography>;
	}

	if (isLoading || isUndefined(data?.data)) {
		return <Spinner tip={t('loading_channels_message')} height="90vh" />;
	}

	return (
		<div className="alert-channels-container">
			<ButtonContainer>
				<Paragraph ellipsis type="secondary">
					{t('sending_channels_note')}
				</Paragraph>
			</ButtonContainer>

			<div className="alert-base-url-section">
				<Typography.Text strong className="alert-section-title">
					{t('alert_base_url_label')}
				</Typography.Text>
				<Typography.Paragraph type="secondary">
					{t('alert_base_url_helper')}
				</Typography.Paragraph>
				<div className="alert-base-url-controls">
					<Input
						className="alert-base-url-input"
						placeholder={t('alert_base_url_placeholder')}
						value={alertBaseURL}
						disabled={!addNewChannelPermission}
						onChange={(event): void => {
							setAlertBaseURL(event.target.value);
							setAlertBaseURLError(null);
						}}
					/>
					<Tooltip
						title={
							!addNewChannelPermission
								? t('alert_base_url_admin_only')
								: undefined
						}
					>
						<Button
							loading={isSavingAlertBaseURL || isAlertBaseURLLoading}
							disabled={
								!addNewChannelPermission ||
								isAlertBaseURLLoading ||
								normalizeAlertBaseURL(alertBaseURL) ===
									normalizeAlertBaseURL(storedAlertBaseURL)
							}
							onClick={onSaveAlertBaseURL}
						>
							{t('alert_base_url_save')}
						</Button>
					</Tooltip>
				</div>
				{alertBaseURLError && (
					<Typography.Text type="danger">{alertBaseURLError}</Typography.Text>
				)}
				{previewURL && (
					<Typography.Text type="secondary">
						{t('alert_base_url_preview', { url: previewURL })}
					</Typography.Text>
				)}
			</div>

			{addNewChannelPermission && (
				<div className="alertmanager-settings-section">
					<Typography.Text strong className="alert-section-title">
						{t('alertmanager_settings_title')}
					</Typography.Text>
					<Typography.Paragraph type="secondary">
						{t('alertmanager_settings_helper')}
					</Typography.Paragraph>
					{isAlertmanagerSettingsLoading && (
						<Spinner
							height="auto"
							tip={t('alertmanager_settings_loading')}
						/>
					)}
					{!isAlertmanagerSettingsLoading && alertmanagerSettings && (
						<>
							<div className="alertmanager-settings-grid alertmanager-duration-grid">
								<div className="alertmanager-settings-field">
									<div className="alertmanager-label-with-tooltip">
										<Typography.Text>
											{t('alertmanager_route_group_wait')}
										</Typography.Text>
										<span className="alertmanager-help-icon">
											<TextToolTip
												text={t('alertmanager_route_group_wait_tooltip')}
												useFilledIcon={false}
												outlinedIcon={<ExclamationCircleOutlined />}
											/>
										</span>
									</div>
									<div className="alertmanager-duration-control">
										<InputNumber
											min={1}
											value={routeInputs.groupWait.value}
											onChange={(value): void =>
												updateRouteInput('groupWait', {
													value: typeof value === 'number' ? value : 0,
													unit: routeInputs.groupWait.unit,
												})
											}
										/>
										<Select
											value={routeInputs.groupWait.unit}
											onChange={(unit: DurationUnit): void =>
												updateRouteInput('groupWait', {
													value: routeInputs.groupWait.value,
													unit,
												})
											}
										>
											<Option value="s">s</Option>
											<Option value="m">m</Option>
											<Option value="h">h</Option>
										</Select>
									</div>
								</div>
								<div className="alertmanager-settings-field">
									<div className="alertmanager-label-with-tooltip">
										<Typography.Text>
											{t('alertmanager_route_group_interval')}
										</Typography.Text>
										<span className="alertmanager-help-icon">
											<TextToolTip
												text={t('alertmanager_route_group_interval_tooltip')}
												useFilledIcon={false}
												outlinedIcon={<ExclamationCircleOutlined />}
											/>
										</span>
									</div>
									<div className="alertmanager-duration-control">
										<InputNumber
											min={1}
											value={routeInputs.groupInterval.value}
											onChange={(value): void =>
												updateRouteInput('groupInterval', {
													value: typeof value === 'number' ? value : 0,
													unit: routeInputs.groupInterval.unit,
												})
											}
										/>
										<Select
											value={routeInputs.groupInterval.unit}
											onChange={(unit: DurationUnit): void =>
												updateRouteInput('groupInterval', {
													value: routeInputs.groupInterval.value,
													unit,
												})
											}
										>
											<Option value="s">s</Option>
											<Option value="m">m</Option>
											<Option value="h">h</Option>
										</Select>
									</div>
								</div>
								<div className="alertmanager-settings-field">
									<div className="alertmanager-label-with-tooltip">
										<Typography.Text>
											{t('alertmanager_route_repeat_interval')}
										</Typography.Text>
										<span className="alertmanager-help-icon">
											<TextToolTip
												text={t('alertmanager_route_repeat_interval_tooltip')}
												useFilledIcon={false}
												outlinedIcon={<ExclamationCircleOutlined />}
											/>
										</span>
									</div>
									<div className="alertmanager-duration-control">
										<InputNumber
											min={1}
											value={routeInputs.repeatInterval.value}
											onChange={(value): void =>
												updateRouteInput('repeatInterval', {
													value: typeof value === 'number' ? value : 0,
													unit: routeInputs.repeatInterval.unit,
												})
											}
										/>
										<Select
											value={routeInputs.repeatInterval.unit}
											onChange={(unit: DurationUnit): void =>
												updateRouteInput('repeatInterval', {
													value: routeInputs.repeatInterval.value,
													unit,
												})
											}
										>
											<Option value="s">s</Option>
											<Option value="m">m</Option>
											<Option value="h">h</Option>
										</Select>
									</div>
								</div>
							</div>

							<div className="alertmanager-subsection">
								<Typography.Text strong className="alert-subsection-title">
									{t('alertmanager_smtp_title')}
								</Typography.Text>
								<div className="alertmanager-settings-grid">
									<div className="alertmanager-settings-field">
										<Typography.Text>{t('alertmanager_smtp_address')}</Typography.Text>
										<Input
											value={alertmanagerSettings.smtp.address}
											onChange={(event): void =>
												updateAlertmanagerSMTPText('address', event.target.value)
											}
										/>
									</div>
									<div className="alertmanager-settings-field">
										<Typography.Text>{t('alertmanager_smtp_from')}</Typography.Text>
										<Input
											value={alertmanagerSettings.smtp.from}
											onChange={(event): void =>
												updateAlertmanagerSMTPText('from', event.target.value)
											}
										/>
									</div>
									<div className="alertmanager-settings-field">
										<Typography.Text>{t('alertmanager_smtp_hello')}</Typography.Text>
										<Input
											value={alertmanagerSettings.smtp.hello}
											onChange={(event): void =>
												updateAlertmanagerSMTPText('hello', event.target.value)
											}
										/>
									</div>
									<div className="alertmanager-settings-field alertmanager-settings-toggle">
										<Typography.Text>
											{t('alertmanager_smtp_require_tls')}
										</Typography.Text>
										<Switch
											checked={alertmanagerSettings.smtp.require_tls}
											onChange={(checked): void =>
												updateAlertmanagerSMTPRequireTLS(checked)
											}
										/>
									</div>
								</div>
							</div>

							<div className="alertmanager-subsection">
								<Typography.Text strong className="alert-subsection-title">
									{t('alertmanager_smtp_auth_title')}
								</Typography.Text>
								<div className="alertmanager-settings-grid">
									<div className="alertmanager-settings-field">
										<Typography.Text>
											{t('alertmanager_smtp_auth_username')}
										</Typography.Text>
										<Input
											value={alertmanagerSettings.smtp.auth.username}
											onChange={(event): void =>
												updateAlertmanagerSMTPAuth(
													'username',
													event.target.value,
												)
											}
										/>
									</div>
									<div className="alertmanager-settings-field">
										<Typography.Text>
											{t('alertmanager_smtp_auth_identity')}
										</Typography.Text>
										<Input
											value={alertmanagerSettings.smtp.auth.identity}
											onChange={(event): void =>
												updateAlertmanagerSMTPAuth(
													'identity',
													event.target.value,
												)
											}
										/>
									</div>
									<div className="alertmanager-settings-field">
										<Typography.Text>
											{t('alertmanager_smtp_auth_password')}
										</Typography.Text>
										<Input.Password
											value={smtpPassword}
											placeholder={
												alertmanagerSettings.smtp.auth.password_set
													? t('alertmanager_settings_password_set')
													: t('alertmanager_smtp_auth_password_placeholder')
											}
											onChange={(event): void => {
												setSmtpPassword(event.target.value);
												setAlertmanagerSettingsError(null);
											}}
										/>
									</div>
									<div className="alertmanager-settings-field">
										<Typography.Text>
											{t('alertmanager_smtp_auth_secret')}
										</Typography.Text>
										<Input.Password
											value={smtpSecret}
											placeholder={
												alertmanagerSettings.smtp.auth.secret_set
													? t('alertmanager_settings_secret_set')
													: t('alertmanager_smtp_auth_secret_placeholder')
											}
											onChange={(event): void => {
												setSmtpSecret(event.target.value);
												setAlertmanagerSettingsError(null);
											}}
										/>
									</div>
								</div>
							</div>

							<div className="alertmanager-subsection">
								<Typography.Text strong className="alert-subsection-title">
									{t('alertmanager_smtp_tls_title')}
								</Typography.Text>
								<div className="alertmanager-settings-grid alertmanager-settings-toggle-grid">
									<div className="alertmanager-settings-field alertmanager-settings-toggle">
										<Typography.Text>
											{t('alertmanager_smtp_tls_enabled')}
										</Typography.Text>
										<Switch
											checked={alertmanagerSettings.smtp.tls.enabled}
											onChange={(checked): void =>
												updateAlertmanagerSMTPTLSBoolean('enabled', checked)
											}
										/>
									</div>
									<div className="alertmanager-settings-field alertmanager-settings-toggle">
										<Typography.Text>
											{t('alertmanager_smtp_tls_insecure_skip')}
										</Typography.Text>
										<Switch
											checked={
												alertmanagerSettings.smtp.tls.insecure_skip_verify
											}
											onChange={(checked): void =>
												updateAlertmanagerSMTPTLSBoolean(
													'insecure_skip_verify',
													checked,
												)
											}
											disabled={!alertmanagerSettings.smtp.tls.enabled}
										/>
									</div>
								</div>
								<div className="alertmanager-settings-grid alertmanager-settings-grid-secondary">
									<div className="alertmanager-settings-field">
										<Typography.Text>
											{t('alertmanager_smtp_tls_ca_file')}
										</Typography.Text>
										<Input
											value={alertmanagerSettings.smtp.tls.ca_file_path}
											onChange={(event): void =>
												updateAlertmanagerSMTPTLSText(
													'ca_file_path',
													event.target.value,
												)
											}
											disabled={!alertmanagerSettings.smtp.tls.enabled}
										/>
									</div>
									<div className="alertmanager-settings-field">
										<Typography.Text>
											{t('alertmanager_smtp_tls_cert_file')}
										</Typography.Text>
										<Input
											value={alertmanagerSettings.smtp.tls.cert_file_path}
											onChange={(event): void =>
												updateAlertmanagerSMTPTLSText(
													'cert_file_path',
													event.target.value,
												)
											}
											disabled={!alertmanagerSettings.smtp.tls.enabled}
										/>
									</div>
									<div className="alertmanager-settings-field">
										<Typography.Text>
											{t('alertmanager_smtp_tls_key_file')}
										</Typography.Text>
										<Input
											value={alertmanagerSettings.smtp.tls.key_file_path}
											onChange={(event): void =>
												updateAlertmanagerSMTPTLSText(
													'key_file_path',
													event.target.value,
												)
											}
											disabled={!alertmanagerSettings.smtp.tls.enabled}
										/>
									</div>
								</div>
							</div>

							<div className="alertmanager-settings-actions">
								<Button
									onClick={onSaveAlertmanagerSettings}
									loading={
										isAlertmanagerSettingsLoading ||
										isSavingAlertmanagerSettings
									}
									disabled={
										!isAlertmanagerSettingsDirty ||
										isAlertmanagerSettingsLoading ||
										isSavingAlertmanagerSettings
									}
								>
									{t('alertmanager_settings_save')}
								</Button>
							</div>
							{alertmanagerSettingsError && (
								<Typography.Text type="danger">
									{alertmanagerSettingsError}
								</Typography.Text>
							)}
						</>
					)}
				</div>
			)}

			<div className="alert-channels-section">
				<div className="alert-channels-header">
					<div className="alert-channels-title">
						<Typography.Text strong className="alert-section-title">
							{t('alert_channels_title')}
						</Typography.Text>
						<Typography.Paragraph type="secondary">
							{t('sending_channels_note')}
						</Typography.Paragraph>
					</div>
					<RightActionContainer>
						<Tooltip
							title={
								!addNewChannelPermission
									? 'Ask an admin to create alert channel'
									: undefined
							}
						>
							<Button
								onClick={onToggleHandler}
								icon={<PlusOutlined />}
								disabled={!addNewChannelPermission}
							>
								{t('button_new_channel')}
							</Button>
						</Tooltip>
					</RightActionContainer>
				</div>
				<AlertChannelsComponent allChannels={data?.data || []} />
			</div>
		</div>
	);
}

export default AlertChannels;
