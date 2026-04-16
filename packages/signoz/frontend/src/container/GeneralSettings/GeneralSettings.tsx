/* eslint-disable sonarjs/no-duplicate-string */
import { InfoCircleOutlined, LoadingOutlined } from '@ant-design/icons';
import {
	Button,
	Card,
	Col,
	Divider,
	Input,
	InputNumber,
	Modal,
	Row,
	Select,
	Spin,
	Switch,
	Tooltip,
	Typography,
} from 'antd';
import getLifecycleConfigApi, {
	DataLifecycleConfig,
} from 'api/settings/getLifecycleConfig';
import setRetentionApi from 'api/settings/setRetention';
import setRetentionApiV2 from 'api/settings/setRetentionV2';
import updateLifecycleConfigApi from 'api/settings/updateLifecycleConfig';
import getS3ConfigApi, { S3Config } from 'api/settings/getS3Config';
import updateS3ConfigApi from 'api/settings/updateS3Config';
import testS3ConnectionApi from 'api/settings/testS3Connection';
import getS3StatusApi, { S3Status } from 'api/settings/getS3Status';
import activateS3Api from 'api/settings/activateS3';
import TextToolTip from 'components/TextToolTip';
import GeneralSettingsCloud from 'container/GeneralSettingsCloud';
import useComponentPermission from 'hooks/useComponentPermission';
import { useGetTenantLicense } from 'hooks/useGetTenantLicense';
import { useNotifications } from 'hooks/useNotifications';
import { StatusCodes } from 'http-status-codes';
import find from 'lodash-es/find';
import { useAppContext } from 'providers/App/App';
import { Fragment, useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { UseQueryResult } from 'react-query';
import { useInterval } from 'react-use';
import {
	ErrorResponse,
	ErrorResponseV2,
	SuccessResponse,
	SuccessResponseV2,
} from 'types/api';
import {
	IDiskType,
	PayloadProps as GetDisksPayload,
} from 'types/api/disks/getDisks';
import APIError from 'types/api/error';
import { TTTLType } from 'types/api/settings/common';
import {
	PayloadPropsLogs as GetRetentionPeriodLogsPayload,
	PayloadPropsMetrics as GetRetentionPeriodMetricsPayload,
	PayloadPropsTraces as GetRetentionPeriodTracesPayload,
} from 'types/api/settings/getRetention';

import Retention from './Retention';
import StatusMessage from './StatusMessage';
import { ActionItemsContainer, ErrorText, ErrorTextContainer } from './styles';

type NumberOrNull = number | null;

function GeneralSettings({
	metricsTtlValuesPayload,
	tracesTtlValuesPayload,
	logsTtlValuesPayload,
	getAvailableDiskPayload,
	metricsTtlValuesRefetch,
	tracesTtlValuesRefetch,
	logsTtlValuesRefetch,
}: GeneralSettingsProps): JSX.Element {
	const { t } = useTranslation(['generalSettings']);
	const [modalMetrics, setModalMetrics] = useState<boolean>(false);
	const [modalTraces, setModalTraces] = useState<boolean>(false);
	const [modalLogs, setModalLogs] = useState<boolean>(false);

	const [postApiLoadingMetrics, setPostApiLoadingMetrics] = useState<boolean>(
		false,
	);
	const [postApiLoadingTraces, setPostApiLoadingTraces] = useState<boolean>(
		false,
	);
	const [postApiLoadingLogs, setPostApiLoadingLogs] = useState<boolean>(false);

	const [availableDisks] = useState<IDiskType[]>(getAvailableDiskPayload);

	const [metricsCurrentTTLValues, setMetricsCurrentTTLValues] = useState(
		metricsTtlValuesPayload,
	);
	const [tracesCurrentTTLValues, setTracesCurrentTTLValues] = useState(
		tracesTtlValuesPayload,
	);

	const [logsCurrentTTLValues, setLogsCurrentTTLValues] = useState(
		logsTtlValuesPayload,
	);

	const { user } = useAppContext();

	const [setRetentionPermission] = useComponentPermission(
		['set_retention_period'],
		user.role,
	);

	const [
		metricsTotalRetentionPeriod,
		setMetricsTotalRetentionPeriod,
	] = useState<NumberOrNull>(null);
	const [
		metricsS3RetentionPeriod,
		setMetricsS3RetentionPeriod,
	] = useState<NumberOrNull>(null);
	const [
		tracesTotalRetentionPeriod,
		setTracesTotalRetentionPeriod,
	] = useState<NumberOrNull>(null);
	const [
		tracesS3RetentionPeriod,
		setTracesS3RetentionPeriod,
	] = useState<NumberOrNull>(null);

	const [
		logsTotalRetentionPeriod,
		setLogsTotalRetentionPeriod,
	] = useState<NumberOrNull>(null);
	const [
		logsS3RetentionPeriod,
		setLogsS3RetentionPeriod,
	] = useState<NumberOrNull>(null);

	useEffect(() => {
		if (metricsCurrentTTLValues) {
			setMetricsTotalRetentionPeriod(
				metricsCurrentTTLValues.metrics_ttl_duration_hrs,
			);
			setMetricsS3RetentionPeriod(
				metricsCurrentTTLValues.metrics_move_ttl_duration_hrs
					? metricsCurrentTTLValues.metrics_move_ttl_duration_hrs
					: null,
			);
		}
	}, [metricsCurrentTTLValues]);

	useEffect(() => {
		if (tracesCurrentTTLValues) {
			setTracesTotalRetentionPeriod(
				tracesCurrentTTLValues.traces_ttl_duration_hrs,
			);
			setTracesS3RetentionPeriod(
				tracesCurrentTTLValues.traces_move_ttl_duration_hrs
					? tracesCurrentTTLValues.traces_move_ttl_duration_hrs
					: null,
			);
		}
	}, [tracesCurrentTTLValues]);

	useEffect(() => {
		if (logsCurrentTTLValues) {
			setLogsTotalRetentionPeriod(logsCurrentTTLValues.default_ttl_days * 24);
			setLogsS3RetentionPeriod(
				logsCurrentTTLValues.cold_storage_ttl_days
					? logsCurrentTTLValues.cold_storage_ttl_days * 24
					: null,
			);
		}
	}, [logsCurrentTTLValues]);

	useInterval(
		async (): Promise<void> => {
			if (metricsTtlValuesPayload.status === 'pending') {
				metricsTtlValuesRefetch();
			}
		},
		metricsTtlValuesPayload.status === 'pending' ? 1000 : null,
	);

	useInterval(
		async (): Promise<void> => {
			if (tracesTtlValuesPayload.status === 'pending') {
				tracesTtlValuesRefetch();
			}
		},
		tracesTtlValuesPayload.status === 'pending' ? 1000 : null,
	);

	useInterval(
		async (): Promise<void> => {
			if (logsTtlValuesPayload.status === 'pending') {
				logsTtlValuesRefetch();
			}
		},
		logsTtlValuesPayload.status === 'pending' ? 1000 : null,
	);

	const { notifications } = useNotifications();

	const onModalToggleHandler = (type: TTTLType): void => {
		if (type === 'metrics') setModalMetrics((modal) => !modal);
		if (type === 'traces') setModalTraces((modal) => !modal);
		if (type === 'logs') setModalLogs((modal) => !modal);
	};
	const onPostApiLoadingHandler = (type: TTTLType): void => {
		if (type === 'metrics') setPostApiLoadingMetrics((modal) => !modal);
		if (type === 'traces') setPostApiLoadingTraces((modal) => !modal);
		if (type === 'logs') setPostApiLoadingLogs((modal) => !modal);
	};

	const onClickSaveHandler = useCallback(
		(type: TTTLType) => {
			if (!setRetentionPermission) {
				notifications.error({
					message: `Sorry you don't have permission to make these changes`,
				});
				return;
			}
			onModalToggleHandler(type);
		},
		[setRetentionPermission, notifications],
	);

	const s3Enabled = useMemo(
		() =>
			!!find(
				availableDisks,
				(disks: IDiskType) =>
					disks?.type === 's3' || disks?.type === 'ObjectStorage',
			),
		[availableDisks],
	);

	// Cold Archive (Glacier IR) state
	const [coldStorageConfig, setColdStorageConfig] = useState<DataLifecycleConfig | null>(null);
	const [coldGlacierEnabled, setColdGlacierEnabled] = useState<boolean>(false);
	const [coldRetentionDays, setColdRetentionDays] = useState<number>(90);
	const [coldBackupFreqDays, setColdBackupFreqDays] = useState<number>(1);
	const [coldArchiveLoading, setColdArchiveLoading] = useState<boolean>(false);
	const [coldArchiveModal, setColdArchiveModal] = useState<boolean>(false);

	// S3 Activation state
	const [s3Status, setS3Status] = useState<S3Status | null>(null);
	const [s3Activating, setS3Activating] = useState<boolean>(false);
	const [s3ActivateModal, setS3ActivateModal] = useState<boolean>(false);

	// Warm S3 Settings state
	const [warmS3Config, setWarmS3Config] = useState<S3Config | null>(null);
	const [warmAuthMode, setWarmAuthMode] = useState<string>('static');
	const [warmBucket, setWarmBucket] = useState<string>('');
	const [warmRegion, setWarmRegion] = useState<string>('ap-northeast-2');
	const [warmAccessKey, setWarmAccessKey] = useState<string>('');
	const [warmSecretKey, setWarmSecretKey] = useState<string>('');
	const [warmSaving, setWarmSaving] = useState<boolean>(false);
	const [warmTesting, setWarmTesting] = useState<boolean>(false);
	const [warmTestResult, setWarmTestResult] = useState<{ success: boolean; message: string } | null>(null);

	// Cold S3 Settings state
	const [coldS3Config, setColdS3Config] = useState<S3Config | null>(null);
	const [coldUseSameAsWarm, setColdUseSameAsWarm] = useState<boolean>(true);
	const [coldAuthMode, setColdAuthMode] = useState<string>('static');
	const [coldBucket, setColdBucket] = useState<string>('');
	const [coldRegionS3, setColdRegionS3] = useState<string>('ap-northeast-2');
	const [coldAccessKey, setColdAccessKey] = useState<string>('');
	const [coldSecretKey, setColdSecretKey] = useState<string>('');
	const [coldS3Saving, setColdS3Saving] = useState<boolean>(false);
	const [coldS3Testing, setColdS3Testing] = useState<boolean>(false);
	const [coldS3TestResult, setColdS3TestResult] = useState<{ success: boolean; message: string } | null>(null);

	// Fetch lifecycle config on mount
	useEffect(() => {
		const fetchLifecycleConfig = async (): Promise<void> => {
			try {
				const config = await getLifecycleConfigApi();
				if (config) {
					setColdStorageConfig(config);
					setColdGlacierEnabled(config.glacier_enabled === 1);
					setColdRetentionDays(config.glacier_retention_days);
					setColdBackupFreqDays(Math.round(config.backup_frequency_hours / 24) || 1);

					// Hot/Warm을 lifecycle config에서 읽기 (single source of truth)
					if (config.hot_days > 0 || config.warm_days > 0) {
						const hotHrs = config.hot_days * 24;
						const totalHrs = (config.hot_days + config.warm_days) * 24;
						setMetricsTotalRetentionPeriod(totalHrs);
						setMetricsS3RetentionPeriod(hotHrs);
						setTracesTotalRetentionPeriod(totalHrs);
						setTracesS3RetentionPeriod(hotHrs);
						setLogsTotalRetentionPeriod(totalHrs);
						setLogsS3RetentionPeriod(hotHrs);
					}
				}
			} catch {
				// Lifecycle config not configured - keep defaults
			}
		};
		fetchLifecycleConfig();
	}, []);

	// Fetch Warm/Cold S3 configs on mount
	useEffect(() => {
		const fetchS3Configs = async (): Promise<void> => {
			try {
				const warm = await getS3ConfigApi('warm');
				if (warm) {
					setWarmS3Config(warm);
					setWarmAuthMode(warm.auth_mode || 'static');
					setWarmBucket(warm.bucket || '');
					setWarmRegion(warm.region || 'ap-northeast-2');
					setWarmAccessKey(warm.access_key_id || '');
					setWarmSecretKey(warm.secret_access_key || '');
				}
				const cold = await getS3ConfigApi('cold');
				if (cold && cold.bucket && cold.bucket !== warm?.bucket) {
					setColdUseSameAsWarm(false);
					setColdS3Config(cold);
					setColdAuthMode(cold.auth_mode || 'static');
					setColdBucket(cold.bucket || '');
					setColdRegionS3(cold.region || 'ap-northeast-2');
					setColdAccessKey(cold.access_key_id || '');
					setColdSecretKey(cold.secret_access_key || '');
				}
			} catch {
				// S3 not configured
			}
		};
		fetchS3Configs();
	}, []);

	// Fetch S3 activation status on mount + poll when activating
	useEffect(() => {
		const fetchStatus = async (): Promise<void> => {
			const status = await getS3StatusApi();
			if (status) setS3Status(status);
		};
		fetchStatus();
	}, []);

	useEffect(() => {
		if (!s3Activating) return;
		const interval = setInterval(async () => {
			const status = await getS3StatusApi();
			if (status) {
				setS3Status(status);
				if (status.activation_status === 'success' || status.activation_status === 'failed') {
					setS3Activating(false);
					if (status.activation_status === 'success') {
						notifications.success({ message: 'S3 storage activated successfully' });
						window.location.reload();
					} else {
						notifications.error({ message: status.job_message || 'S3 activation failed' });
					}
				}
			}
		}, 5000);
		return (): void => clearInterval(interval);
	}, [s3Activating, notifications]);

	// S3 Activate handler
	const onActivateS3 = async (mode: 'activate' | 'apply' = 'activate'): Promise<void> => {
		try {
			setS3Activating(true);
			await activateS3Api(mode);
		} catch (error) {
			setS3Activating(false);
			notifications.error({ message: 'Failed to start S3 activation' });
		}
	};

	// Warm S3 Save/Test
	const onWarmS3Save = async (): Promise<void> => {
		try {
			setWarmSaving(true);
			await updateS3ConfigApi('warm', {
				auth_mode: warmAuthMode,
				bucket: warmBucket,
				region: warmRegion,
				access_key_id: warmAccessKey !== '****' ? warmAccessKey : undefined,
				secret_access_key: warmSecretKey !== '****' ? warmSecretKey : undefined,
				s3_enabled: 1,
			});
			const config = await getS3ConfigApi('warm');
			if (config) {
				setWarmS3Config(config);
				setWarmAccessKey(config.access_key_id);
				setWarmSecretKey(config.secret_access_key);
			}
			// S3 status 갱신 (Activate 배너 표시를 위해)
			// ReplacingMergeTree FINAL 조회 타이밍을 위해 잠시 대기 후 조회
			await new Promise((resolve) => { setTimeout(resolve, 1000); });
			const status = await getS3StatusApi();
			if (status) setS3Status(status);
			notifications.success({ message: 'Warm S3 settings saved' });
		} catch (error) {
			notifications.error({
				message: 'Error',
				description: error instanceof Error ? error.message : 'Failed to save Warm S3 settings',
			});
		} finally {
			setWarmSaving(false);
		}
	};

	const onWarmS3Test = async (): Promise<void> => {
		try {
			setWarmTesting(true);
			setWarmTestResult(null);
			const result = await testS3ConnectionApi('warm', {
				auth_mode: warmAuthMode,
				bucket: warmBucket,
				region: warmRegion,
				access_key_id: warmAccessKey !== '****' ? warmAccessKey : undefined,
				secret_access_key: warmSecretKey !== '****' ? warmSecretKey : undefined,
			});
			setWarmTestResult(result);
		} catch {
			setWarmTestResult({ success: false, message: 'Test failed' });
		} finally {
			setWarmTesting(false);
		}
	};

	// Cold S3 Save/Test
	const onColdS3Save = async (): Promise<void> => {
		try {
			setColdS3Saving(true);
			if (coldUseSameAsWarm) {
				// Copy warm settings to cold
				await updateS3ConfigApi('cold', {
					auth_mode: warmAuthMode,
					bucket: warmBucket,
					region: warmRegion,
					access_key_id: warmAccessKey !== '****' ? warmAccessKey : undefined,
					secret_access_key: warmSecretKey !== '****' ? warmSecretKey : undefined,
					s3_enabled: 1,
				});
			} else {
				await updateS3ConfigApi('cold', {
					auth_mode: coldAuthMode,
					bucket: coldBucket,
					region: coldRegionS3,
					access_key_id: coldAccessKey !== '****' ? coldAccessKey : undefined,
					secret_access_key: coldSecretKey !== '****' ? coldSecretKey : undefined,
					s3_enabled: 1,
				});
			}
			notifications.success({ message: 'Cold S3 settings saved' });
		} catch (error) {
			notifications.error({
				message: 'Error',
				description: error instanceof Error ? error.message : 'Failed to save Cold S3 settings',
			});
		} finally {
			setColdS3Saving(false);
		}
	};

	const onColdS3Test = async (): Promise<void> => {
		try {
			setColdS3Testing(true);
			setColdS3TestResult(null);

			// Separate bucket mode: require all fields
			if (!coldUseSameAsWarm) {
				if (!coldBucket) {
					setColdS3TestResult({ success: false, message: 'Cold Bucket is required' });
					return;
				}
				if (coldAuthMode === 'static' && (!coldAccessKey || !coldSecretKey)) {
					setColdS3TestResult({ success: false, message: 'Access Key and Secret Key are required' });
					return;
				}
			}

			const bucket = coldUseSameAsWarm ? warmBucket : coldBucket;
			const region = coldUseSameAsWarm ? warmRegion : coldRegionS3;
			const authMode = coldUseSameAsWarm ? warmAuthMode : coldAuthMode;
			const ak = coldUseSameAsWarm ? warmAccessKey : coldAccessKey;
			const sk = coldUseSameAsWarm ? warmSecretKey : coldSecretKey;
			const result = await testS3ConnectionApi('cold', {
				auth_mode: authMode,
				bucket,
				region,
				access_key_id: ak !== '****' ? ak : undefined,
				secret_access_key: sk !== '****' ? sk : undefined,
			});
			setColdS3TestResult(result);
		} catch {
			setColdS3TestResult({ success: false, message: 'Test failed' });
		} finally {
			setColdS3Testing(false);
		}
	};

	const isColdArchiveSaveDisabled = useMemo(() => {
		if (!coldStorageConfig) return false; // Allow initial save
		return (
			coldGlacierEnabled === (coldStorageConfig.glacier_enabled === 1) &&
			coldRetentionDays === coldStorageConfig.glacier_retention_days
		);
	}, [coldStorageConfig, coldGlacierEnabled, coldRetentionDays]);

	const onColdArchiveSave = async (): Promise<void> => {
		try {
			setColdArchiveLoading(true);

			// Backend handles glacier toggle TTL changes synchronously
			// No need to call setTTL from frontend (avoids goroutine race condition)
			await updateLifecycleConfigApi({
				glacier_enabled: coldGlacierEnabled ? 1 : 0,
				glacier_retention_days: coldRetentionDays,
			});

			// Refresh config after save
			const config = await getLifecycleConfigApi();
			if (config) {
				setColdStorageConfig(config);
			}
			notifications.success({
				message: 'Cold archive settings saved successfully',
			});
		} catch (error) {
			notifications.error({
				message: 'Error',
				description:
					error instanceof Error ? error.message : 'Failed to save cold archive settings',
			});
		} finally {
			setColdArchiveLoading(false);
			setColdArchiveModal(false);
		}
	};

	const [
		isMetricsSaveDisabled,
		isTracesSaveDisabled,
		isLogsSaveDisabled,
		errorText,
	] = useMemo((): [
		boolean,
		boolean,
		boolean,
		string,
		// eslint-disable-next-line sonarjs/cognitive-complexity
	] => {
		// Various methods to return dynamic error message text.
		const messages = {
			compareError: (name: string | number): string =>
				t('retention_comparison_error', { name }),
			nullValueError: (name: string | number): string =>
				t('retention_null_value_error', { name }),
		};

		// Defaults to button not disabled and empty error message text.
		let isMetricsSaveDisabled = false;
		let isTracesSaveDisabled = false;
		let isLogsSaveDisabled = false;
		let errorText = '';

		if (s3Enabled) {
			if (
				(metricsTotalRetentionPeriod || metricsS3RetentionPeriod) &&
				Number(metricsTotalRetentionPeriod) <= Number(metricsS3RetentionPeriod)
			) {
				isMetricsSaveDisabled = true;
				errorText = messages.compareError('metrics');
			} else if (
				(tracesTotalRetentionPeriod || tracesS3RetentionPeriod) &&
				Number(tracesTotalRetentionPeriod) <= Number(tracesS3RetentionPeriod)
			) {
				isTracesSaveDisabled = true;
				errorText = messages.compareError('traces');
			} else if (
				(logsTotalRetentionPeriod || logsS3RetentionPeriod) &&
				Number(logsTotalRetentionPeriod) <= Number(logsS3RetentionPeriod)
			) {
				isLogsSaveDisabled = true;
				errorText = messages.compareError('logs');
			}
		}

		if (
			!metricsTotalRetentionPeriod ||
			!tracesTotalRetentionPeriod ||
			!logsTotalRetentionPeriod
		) {
			isMetricsSaveDisabled = true;
			isTracesSaveDisabled = true;
			isLogsSaveDisabled = true;
			if (
				!metricsTotalRetentionPeriod &&
				!tracesTotalRetentionPeriod &&
				!logsTotalRetentionPeriod
			) {
				errorText = messages.nullValueError('metrics, traces and logs');
			} else if (!metricsTotalRetentionPeriod) {
				errorText = messages.nullValueError('metrics');
			} else if (!tracesTotalRetentionPeriod) {
				errorText = messages.nullValueError('traces');
			} else if (!logsTotalRetentionPeriod) {
				errorText = messages.nullValueError('logs');
			}
		}
		if (
			metricsCurrentTTLValues?.metrics_ttl_duration_hrs ===
				metricsTotalRetentionPeriod &&
			metricsCurrentTTLValues.metrics_move_ttl_duration_hrs ===
				metricsS3RetentionPeriod
		)
			isMetricsSaveDisabled = true;

		if (
			tracesCurrentTTLValues.traces_ttl_duration_hrs ===
				tracesTotalRetentionPeriod &&
			tracesCurrentTTLValues.traces_move_ttl_duration_hrs ===
				tracesS3RetentionPeriod
		)
			isTracesSaveDisabled = true;

		if (
			logsCurrentTTLValues.default_ttl_days * 24 === logsTotalRetentionPeriod &&
			logsCurrentTTLValues.cold_storage_ttl_days &&
			logsCurrentTTLValues.cold_storage_ttl_days * 24 === logsS3RetentionPeriod
		)
			isLogsSaveDisabled = true;

		return [
			isMetricsSaveDisabled,
			isTracesSaveDisabled,
			isLogsSaveDisabled,
			errorText,
		];
	}, [
		logsCurrentTTLValues.cold_storage_ttl_days,
		logsCurrentTTLValues.default_ttl_days,
		logsS3RetentionPeriod,
		logsTotalRetentionPeriod,
		metricsCurrentTTLValues.metrics_move_ttl_duration_hrs,
		metricsCurrentTTLValues?.metrics_ttl_duration_hrs,
		metricsS3RetentionPeriod,
		metricsTotalRetentionPeriod,
		s3Enabled,
		t,
		tracesCurrentTTLValues.traces_move_ttl_duration_hrs,
		tracesCurrentTTLValues.traces_ttl_duration_hrs,
		tracesS3RetentionPeriod,
		tracesTotalRetentionPeriod,
	]);

	// eslint-disable-next-line sonarjs/cognitive-complexity
	const onOkHandler = async (type: TTTLType): Promise<void> => {
		let apiCallTotalRetention;
		let apiCallS3Retention;

		switch (type) {
			case 'metrics': {
				apiCallTotalRetention = metricsTotalRetentionPeriod;
				apiCallS3Retention = metricsS3RetentionPeriod;
				break;
			}
			case 'traces': {
				apiCallTotalRetention = tracesTotalRetentionPeriod;
				apiCallS3Retention = tracesS3RetentionPeriod;
				break;
			}
			case 'logs': {
				apiCallTotalRetention = logsTotalRetentionPeriod;
				apiCallS3Retention = logsS3RetentionPeriod;
				break;
			}
			default: {
				break;
			}
		}
		try {
			onPostApiLoadingHandler(type);
			let hasSetTTLFailed = false;

			try {
				if (type === 'logs') {
					// Only send S3 values if user has specified a duration
					const s3RetentionDays =
						apiCallS3Retention && apiCallS3Retention > 0
							? apiCallS3Retention / 24
							: 0;

					await setRetentionApiV2({
						type,
						defaultTTLDays: apiCallTotalRetention ? apiCallTotalRetention / 24 : -1, // convert Hours to days
						coldStorageVolume: s3RetentionDays > 0 ? 's3' : '',
						coldStorageDurationDays: s3RetentionDays,
						ttlConditions: [],
					});
				} else {
					await setRetentionApi({
						type,
						totalDuration: `${apiCallTotalRetention || -1}h`,
						coldStorage: s3Enabled ? 's3' : null,
						toColdDuration: `${apiCallS3Retention || -1}h`,
					});
				}
			} catch (error) {
				hasSetTTLFailed = true;
				if ((error as APIError).getHttpStatusCode() === StatusCodes.CONFLICT) {
					notifications.error({
						message: 'Error',
						description: t('retention_request_race_condition'),
						placement: 'topRight',
					});
				} else {
					notifications.error({
						message: 'Error',
						description: (error as APIError).getErrorMessage(),
						placement: 'topRight',
					});
				}
			}

			// Sync hot/warm days to lifecycle config after successful TTL change
			if (!hasSetTTLFailed && s3Enabled) {
				const hotDays = apiCallS3Retention ? Math.round(apiCallS3Retention / 24) : 0;
				const warmDays = apiCallTotalRetention && apiCallS3Retention
					? Math.round((apiCallTotalRetention - apiCallS3Retention) / 24) : 0;
				try {
					await updateLifecycleConfigApi({ hot_days: hotDays, warm_days: warmDays });
				} catch {
					// Non-fatal: lifecycle config sync failed
				}
			}

			if (type === 'metrics') {
				metricsTtlValuesRefetch();

				if (!hasSetTTLFailed)
					// Updates the currentTTL Values in order to avoid pushing the same values.
					setMetricsCurrentTTLValues({
						metrics_ttl_duration_hrs: metricsTotalRetentionPeriod || -1,
						metrics_move_ttl_duration_hrs: metricsS3RetentionPeriod || -1,
						status: '',
					});
			} else if (type === 'traces') {
				tracesTtlValuesRefetch();

				if (!hasSetTTLFailed)
					// Updates the currentTTL Values in order to avoid pushing the same values.
					setTracesCurrentTTLValues({
						traces_ttl_duration_hrs: tracesTotalRetentionPeriod || -1,
						traces_move_ttl_duration_hrs: tracesS3RetentionPeriod || -1,
						status: '',
					});
			} else if (type === 'logs') {
				logsTtlValuesRefetch();
				if (!hasSetTTLFailed)
					// Updates the currentTTL Values in order to avoid pushing the same values.
					setLogsCurrentTTLValues((prev) => ({
						...prev,
						cold_storage_ttl_days: logsS3RetentionPeriod
							? logsS3RetentionPeriod / 24
							: -1,
						default_ttl_days: logsTotalRetentionPeriod
							? logsTotalRetentionPeriod / 24 // convert Hours to days
							: -1,
					}));
			}
		} catch (error) {
			notifications.error({
				message: 'Error',
				description: t('retention_failed_message'),
				placement: 'topRight',
			});
		}

		onPostApiLoadingHandler(type);
		onModalToggleHandler(type);
	};

	const { isCloudUser: isCloudUserVal } = useGetTenantLicense();

	// --- Derived hot/warm days from existing hour-based states ---
	const metricsHotDays = metricsS3RetentionPeriod ? Math.round(metricsS3RetentionPeriod / 24) : null;
	const metricsWarmDays =
		metricsTotalRetentionPeriod && metricsS3RetentionPeriod
			? Math.round((metricsTotalRetentionPeriod - metricsS3RetentionPeriod) / 24)
			: null;
	const metricsTotalDays = metricsTotalRetentionPeriod
		? Math.round(metricsTotalRetentionPeriod / 24)
		: null;

	const tracesHotDays = tracesS3RetentionPeriod ? Math.round(tracesS3RetentionPeriod / 24) : null;
	const tracesWarmDays =
		tracesTotalRetentionPeriod && tracesS3RetentionPeriod
			? Math.round((tracesTotalRetentionPeriod - tracesS3RetentionPeriod) / 24)
			: null;
	const tracesTotalDays = tracesTotalRetentionPeriod
		? Math.round(tracesTotalRetentionPeriod / 24)
		: null;

	const logsHotDays = logsS3RetentionPeriod ? Math.round(logsS3RetentionPeriod / 24) : null;
	const logsWarmDays =
		logsTotalRetentionPeriod && logsS3RetentionPeriod
			? Math.round((logsTotalRetentionPeriod - logsS3RetentionPeriod) / 24)
			: null;
	const logsTotalDays = logsTotalRetentionPeriod
		? Math.round(logsTotalRetentionPeriod / 24)
		: null;

	// --- Handlers for hot/warm day changes ---
	const onHotDaysChange = useCallback(
		(type: TTTLType, days: number | null): void => {
			if (days === null || days < 1) return;
			const hotHours = days * 24;
			if (type === 'metrics') {
				setMetricsS3RetentionPeriod(hotHours);
				setMetricsTotalRetentionPeriod(hotHours + (metricsWarmDays || 0) * 24);
			} else if (type === 'traces') {
				setTracesS3RetentionPeriod(hotHours);
				setTracesTotalRetentionPeriod(hotHours + (tracesWarmDays || 0) * 24);
			} else if (type === 'logs') {
				setLogsS3RetentionPeriod(hotHours);
				setLogsTotalRetentionPeriod(hotHours + (logsWarmDays || 0) * 24);
			}
		},
		[metricsWarmDays, tracesWarmDays, logsWarmDays],
	);

	const onWarmDaysChange = useCallback(
		(type: TTTLType, days: number | null): void => {
			if (days === null || days < 1) return;
			const warmHours = days * 24;
			if (type === 'metrics') {
				setMetricsTotalRetentionPeriod((metricsHotDays || 0) * 24 + warmHours);
			} else if (type === 'traces') {
				setTracesTotalRetentionPeriod((tracesHotDays || 0) * 24 + warmHours);
			} else if (type === 'logs') {
				setLogsTotalRetentionPeriod((logsHotDays || 0) * 24 + warmHours);
			}
		},
		[metricsHotDays, tracesHotDays, logsHotDays],
	);

	const onRetentionDaysChange = useCallback(
		(type: TTTLType, days: number | null): void => {
			if (days === null || days < 1) return;
			const hours = days * 24;
			if (type === 'metrics') setMetricsTotalRetentionPeriod(hours);
			else if (type === 'traces') setTracesTotalRetentionPeriod(hours);
			else if (type === 'logs') setLogsTotalRetentionPeriod(hours);
		},
		[],
	);

	// --- Per-signal config for unified table ---
	const signalRows = [
		{
			name: 'Metrics',
			type: 'metrics' as TTTLType,
			hotDays: metricsHotDays,
			warmDays: metricsWarmDays,
			totalDays: metricsTotalDays,
			isSaveDisabled:
				metricsTtlValuesPayload.status === 'pending' || isMetricsSaveDisabled,
			isPending: metricsTtlValuesPayload.status === 'pending',
			isLoading: postApiLoadingMetrics,
			modal: modalMetrics,
			statusComponent: (
				<StatusMessage
					total_retention={metricsTtlValuesPayload.expected_metrics_ttl_duration_hrs}
					status={metricsTtlValuesPayload.status}
					s3_retention={metricsTtlValuesPayload.expected_metrics_move_ttl_duration_hrs}
				/>
			),
		},
		{
			name: 'Traces',
			type: 'traces' as TTTLType,
			hotDays: tracesHotDays,
			warmDays: tracesWarmDays,
			totalDays: tracesTotalDays,
			isSaveDisabled:
				tracesTtlValuesPayload.status === 'pending' || isTracesSaveDisabled,
			isPending: tracesTtlValuesPayload.status === 'pending',
			isLoading: postApiLoadingTraces,
			modal: modalTraces,
			statusComponent: (
				<StatusMessage
					total_retention={tracesTtlValuesPayload.expected_traces_ttl_duration_hrs}
					status={tracesTtlValuesPayload.status}
					s3_retention={tracesTtlValuesPayload.expected_traces_move_ttl_duration_hrs}
				/>
			),
		},
		{
			name: 'Logs',
			type: 'logs' as TTTLType,
			hotDays: logsHotDays,
			warmDays: logsWarmDays,
			totalDays: logsTotalDays,
			isSaveDisabled:
				logsTtlValuesPayload.status === 'pending' || isLogsSaveDisabled,
			isPending: logsTtlValuesPayload.status === 'pending',
			isLoading: postApiLoadingLogs,
			modal: modalLogs,
			statusComponent: (
				<StatusMessage
					total_retention={logsTtlValuesPayload.expected_logs_ttl_duration_hrs}
					status={logsTtlValuesPayload.status}
					s3_retention={logsTtlValuesPayload.expected_logs_move_ttl_duration_hrs}
				/>
			),
		},
	];

	// Timeline bar widths (proportional)
	const coldIsUnlimited = coldGlacierEnabled && coldRetentionDays === 0;
	const coldDaysForTimeline = coldGlacierEnabled ? (coldRetentionDays || 0) : 0;
	// For unlimited cold, reserve 30% of bar for cold; otherwise include cold days in max
	const hotWarmMax = Math.max(metricsTotalDays || 0, tracesTotalDays || 0, logsTotalDays || 0, 1);
	const timelineBarMax = coldIsUnlimited
		? hotWarmMax // cold gets fixed 30%, hot+warm share 70%
		: Math.max(
			(metricsTotalDays || 0) + coldDaysForTimeline,
			(tracesTotalDays || 0) + coldDaysForTimeline,
			(logsTotalDays || 0) + coldDaysForTimeline,
			1,
		);

	const infoIconStyle = { color: 'rgba(255,255,255,0.45)', marginLeft: 4 };
	const headerStyle: React.CSSProperties = {
		fontWeight: 600,
		fontSize: '13px',
		color: 'rgba(255,255,255,0.65)',
		paddingBottom: 8,
	};

	return (
		<>
			{Element}
			<Col xs={24} md={22} xl={20} xxl={18} style={{ margin: 'auto' }}>
				<ErrorTextContainer>
					{errorText && <ErrorText>{errorText}</ErrorText>}
				</ErrorTextContainer>

				<Card>
					<Typography.Title style={{ margin: 0 }} level={3}>
						Data Lifecycle
					</Typography.Title>
					<Typography.Text type="secondary" style={{ display: 'block', marginTop: 4 }}>
						{s3Enabled
							? 'Data flows through storage tiers: Hot → Warm (S3) → Delete, with optional Cold (Glacier IR) archival.'
							: 'Configure how long each signal type retains data before deletion.'}
					</Typography.Text>

					<Divider style={{ margin: '1rem 0' }} />

					{/* ========== Storage Tier Table ========== */}
					{s3Enabled ? (
						<>
							{/* Header */}
							<Row gutter={16} style={{ marginBottom: 4 }}>
								<Col span={3} style={headerStyle}>Signal</Col>
								<Col span={4} style={headerStyle}>
									Hot{' '}
									<Tooltip title="Duration data stays on fast local storage before moving to S3.">
										<InfoCircleOutlined style={infoIconStyle} />
									</Tooltip>
								</Col>
								<Col span={4} style={headerStyle}>
									Warm (S3){' '}
									<Tooltip title="Duration data stays on S3 after leaving Hot storage. Data is recompressed with ZSTD(3) for cost efficiency.">
										<InfoCircleOutlined style={infoIconStyle} />
									</Tooltip>
								</Col>
								<Col span={3} style={headerStyle}>
									Total{' '}
									<Tooltip title="Total retention = Hot + Warm. Data is permanently deleted after this period.">
										<InfoCircleOutlined style={infoIconStyle} />
									</Tooltip>
								</Col>
								<Col span={6} style={headerStyle}>Timeline</Col>
								<Col span={4} style={headerStyle} />
							</Row>

							<Divider style={{ margin: '0 0 12px 0', opacity: 0.3 }} />

							{/* Signal rows */}
							{signalRows.map((signal) => (
								<Fragment key={signal.name}>
									<Row gutter={16} align="middle" style={{ marginBottom: 16 }}>
										<Col span={3}>
											<Typography.Text strong>{signal.name}</Typography.Text>
										</Col>
										<Col span={4}>
											<InputNumber
												min={1}
												value={signal.hotDays}
												onChange={(val): void => onHotDaysChange(signal.type, val)}
												style={{ width: 130 }}
												addonAfter="days"
											/>
										</Col>
										<Col span={4}>
											<InputNumber
												min={1}
												value={signal.warmDays}
												onChange={(val): void => onWarmDaysChange(signal.type, val)}
												style={{ width: 130 }}
												addonAfter="days"
											/>
										</Col>
										<Col span={3}>
											<Typography.Text type="secondary" strong>
												{signal.totalDays ? `${signal.totalDays} days` : '-'}
											</Typography.Text>
										</Col>
										<Col span={6}>
											{/* Timeline bar */}
											<div style={{ display: 'flex', height: 20, borderRadius: 4, overflow: 'hidden', background: 'rgba(255,255,255,0.05)' }}>
												<Tooltip title={`Hot: ${signal.hotDays || 0} days`}>
													<div
														style={{
															width: coldIsUnlimited
																? `${((signal.hotDays || 0) / hotWarmMax) * 70}%`
																: `${((signal.hotDays || 0) / timelineBarMax) * 100}%`,
															background: '#ff6b6b',
															minWidth: signal.hotDays ? 4 : 0,
															transition: 'width 0.3s',
														}}
													/>
												</Tooltip>
												<Tooltip title={`Warm: ${signal.warmDays || 0} days (S3)`}>
													<div
														style={{
															width: coldIsUnlimited
																? `${((signal.warmDays || 0) / hotWarmMax) * 70}%`
																: `${((signal.warmDays || 0) / timelineBarMax) * 100}%`,
															background: '#ffd93d',
															minWidth: signal.warmDays ? 4 : 0,
															transition: 'width 0.3s',
														}}
													/>
												</Tooltip>
												{coldGlacierEnabled && (
													<Tooltip title={`Cold: ${coldRetentionDays === 0 ? 'Unlimited' : `${coldRetentionDays} days`} (Glacier IR)`}>
														<div
															style={{
																width: coldIsUnlimited
																	? '30%'
																	: `${(coldDaysForTimeline / timelineBarMax) * 100}%`,
																background: coldIsUnlimited
																	? 'linear-gradient(90deg, #6bcaff, transparent)'
																	: '#6bcaff',
																minWidth: 4,
																transition: 'width 0.3s',
															}}
														/>
													</Tooltip>
												)}
											</div>
										</Col>
										<Col span={4} style={{ textAlign: 'right', display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: 8 }}>
											{signal.statusComponent}
											{!isCloudUserVal && (
												<Button
													type="primary"
													size="small"
													onClick={(): void => onClickSaveHandler(signal.type)}
													disabled={signal.isSaveDisabled}
													loading={signal.isPending}
												>
													Save
												</Button>
											)}
										</Col>
									</Row>

									{/* Confirmation modal */}
									<Modal
										title={t('retention_confirmation')}
										focusTriggerAfterClose
										forceRender
										destroyOnClose
										closable
										onCancel={(): void => onModalToggleHandler(signal.type)}
										onOk={(): Promise<void> => onOkHandler(signal.type)}
										centered
										open={signal.modal}
										confirmLoading={signal.isLoading}
									>
										<Typography>
											{t('retention_confirmation_description', {
												name: signal.name.toLowerCase(),
											})}
										</Typography>
									</Modal>
								</Fragment>
							))}

							{/* Timeline legend */}
							<Row style={{ marginTop: 8, marginBottom: 8 }}>
								<Col offset={3}>
									<span style={{ display: 'inline-flex', alignItems: 'center', marginRight: 16, fontSize: 12 }}>
										<span style={{ display: 'inline-block', width: 12, height: 12, borderRadius: 2, background: '#ff6b6b', marginRight: 4 }} />
										Hot
									</span>
									<span style={{ display: 'inline-flex', alignItems: 'center', marginRight: 16, fontSize: 12 }}>
										<span style={{ display: 'inline-block', width: 12, height: 12, borderRadius: 2, background: '#ffd93d', marginRight: 4 }} />
										Warm (S3)
									</span>
									{coldGlacierEnabled && (
										<span style={{ display: 'inline-flex', alignItems: 'center', fontSize: 12 }}>
											<span style={{ display: 'inline-block', width: 12, height: 12, borderRadius: 2, background: '#6bcaff', marginRight: 4 }} />
											Cold (Glacier IR)
										</span>
									)}
								</Col>
							</Row>
						</>
					) : (
						/* Non-S3: simple retention per signal */
						<>
							<Row gutter={16} style={{ marginBottom: 4 }}>
								<Col span={6} style={headerStyle}>Signal</Col>
								<Col span={8} style={headerStyle}>
									Retention Period{' '}
									<Tooltip title="How long data is kept before permanent deletion.">
										<InfoCircleOutlined style={infoIconStyle} />
									</Tooltip>
								</Col>
								<Col span={6} style={headerStyle} />
								<Col span={4} style={headerStyle} />
							</Row>
							<Divider style={{ margin: '0 0 12px 0', opacity: 0.3 }} />
							{signalRows.map((signal) => (
								<Row key={signal.name} gutter={16} align="middle" style={{ marginBottom: 16 }}>
									<Col span={6}>
										<Typography.Text strong>{signal.name}</Typography.Text>
									</Col>
									<Col span={8}>
										<InputNumber
											min={1}
											value={signal.totalDays}
											onChange={(val): void => onRetentionDaysChange(signal.type, val)}
											style={{ width: 120 }}
											addonAfter="days"
										/>
									</Col>
									<Col span={10} style={{ textAlign: 'right', display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: 8 }}>
										{signal.statusComponent}
										{!isCloudUserVal && (
											<Button
												type="primary"
												size="small"
												onClick={(): void => onClickSaveHandler(signal.type)}
												disabled={signal.isSaveDisabled}
												loading={signal.isPending}
											>
												Save
											</Button>
										)}
									</Col>
									<Modal
										title={t('retention_confirmation')}
										focusTriggerAfterClose
										forceRender
										destroyOnClose
										closable
										onCancel={(): void => onModalToggleHandler(signal.type)}
										onOk={(): Promise<void> => onOkHandler(signal.type)}
										centered
										open={signal.modal}
										confirmLoading={signal.isLoading}
									>
										<Typography>
											{t('retention_confirmation_description', {
												name: signal.name.toLowerCase(),
											})}
										</Typography>
									</Modal>
								</Row>
							))}
						</>
					)}

					{/* ========== S3 Activation Banner ========== */}
					{s3Status && s3Status.activation_status === 'running' && (
						<>
							<Divider style={{ margin: '1rem 0' }} />
							<div style={{ background: '#1a1a2e', border: '1px solid #e2b93b', borderRadius: 8, padding: '16px 20px', textAlign: 'center' }}>
								<Spin indicator={<LoadingOutlined style={{ fontSize: 20 }} spin />} />
								<Typography.Text style={{ marginLeft: 12 }}>
									Activating S3 Storage... Applying ClickHouse configuration and restarting
								</Typography.Text>
							</div>
						</>
					)}

					{s3Status && s3Status.activation_status === 'failed' && (
						<>
							<Divider style={{ margin: '1rem 0' }} />
							<div style={{ background: '#2a1a1a', border: '1px solid #ff4d4f', borderRadius: 8, padding: '16px 20px' }}>
								<Typography.Text type="danger">
									S3 activation failed: {s3Status.job_message || 'Unknown error'}
								</Typography.Text>
								<Button size="small" onClick={(): void => setS3ActivateModal(true)} style={{ marginLeft: 12 }}>
									Retry
								</Button>
							</div>
						</>
					)}

					{/* S3 미설정 + 미활성화 시: Enable 안내 + Activate 버튼 */}
					{!s3Enabled && s3Status && s3Status.s3_configured && !s3Status.s3_active && s3Status.activation_status === 'idle' && (
						<>
							<Divider style={{ margin: '1rem 0' }} />
							<div style={{ background: '#1a1a2e', border: '1px solid #1890ff', borderRadius: 8, padding: '16px 20px', textAlign: 'center' }}>
								<Typography.Text>
									S3 configuration saved. Click the button below to activate.
								</Typography.Text>
								<br />
								<Button type="primary" onClick={(): void => setS3ActivateModal(true)} loading={s3Activating} style={{ marginTop: 12 }}>
									Activate S3 Storage
								</Button>
								<Modal
									title="Confirm S3 Activation"
									open={s3ActivateModal}
									onOk={async (): Promise<void> => { setS3ActivateModal(false); await onActivateS3('activate'); }}
									onCancel={(): void => setS3ActivateModal(false)}
									okText="Activate"
									cancelText="Cancel"
									centered
								>
									<Typography.Text>
										Activating S3 Storage will restart the ClickHouse server.
										Data queries may be temporarily unavailable during the restart.
									</Typography.Text>
									<br /><br />
									<Typography.Text type="secondary">
										Estimated time: 1~3 minutes
									</Typography.Text>
									<br /><br />
									<Typography.Text type="warning">
										Do you want to proceed?
									</Typography.Text>
								</Modal>
							</div>
						</>
					)}

					{/* ========== S3 미설정 + 미구성: Enable S3 Storage 폼 ========== */}
					{!s3Enabled && s3Status && !s3Status.s3_configured && s3Status.activation_status === 'idle' && (
						<>
							<Divider style={{ margin: '1rem 0' }} />
							<Typography.Title style={{ margin: 0 }} level={4}>
								S3 Storage Tiering{' '}
								<Tooltip title="Enable S3 to add Warm/Cold storage tiers for cost-effective data retention.">
									<InfoCircleOutlined style={infoIconStyle} />
								</Tooltip>
							</Typography.Title>
							<Typography.Text type="secondary" style={{ display: 'block', marginTop: 4 }}>
								Enable S3 storage to move older data from local disk to S3, reducing storage costs.
							</Typography.Text>
							<Row gutter={[24, 16]} style={{ marginTop: 12 }}>
								<Col span={6}>
									<Typography.Text type="secondary" style={{ fontSize: 12 }}>Auth Mode</Typography.Text>
									<Select value={warmAuthMode} onChange={(v): void => setWarmAuthMode(v)} style={{ width: '100%', marginTop: 4 }}>
										<Select.Option value="static">Static Key</Select.Option>
										<Select.Option value="iam">IAM Role</Select.Option>
									</Select>
								</Col>
								<Col span={9}>
									<Typography.Text type="secondary" style={{ fontSize: 12 }}>Bucket</Typography.Text>
									<Input value={warmBucket} onChange={(e): void => setWarmBucket(e.target.value)} placeholder="my-s3-bucket" style={{ marginTop: 4 }} />
								</Col>
								<Col span={9}>
									<Typography.Text type="secondary" style={{ fontSize: 12 }}>Region</Typography.Text>
									<Input value={warmRegion} onChange={(e): void => setWarmRegion(e.target.value)} placeholder="ap-northeast-2" style={{ marginTop: 4 }} />
								</Col>
							</Row>
							{warmAuthMode === 'static' && (
								<Row gutter={[24, 16]} style={{ marginTop: 8 }}>
									<Col span={12}>
										<Typography.Text type="secondary" style={{ fontSize: 12 }}>Access Key ID</Typography.Text>
										<Input value={warmAccessKey} onChange={(e): void => setWarmAccessKey(e.target.value)} placeholder="AKIA..." style={{ marginTop: 4 }} />
									</Col>
									<Col span={12}>
										<Typography.Text type="secondary" style={{ fontSize: 12 }}>Secret Access Key</Typography.Text>
										<Input.Password value={warmSecretKey} onChange={(e): void => setWarmSecretKey(e.target.value)} style={{ marginTop: 4 }} />
									</Col>
								</Row>
							)}
							<Row justify="end" style={{ marginTop: 12 }}>
								<Col>
									<Button onClick={onWarmS3Test} loading={warmTesting} disabled={!warmBucket}>Test Connection</Button>
									{warmTestResult && (
										<Typography.Text type={warmTestResult.success ? 'success' : 'danger'} style={{ marginLeft: 6, fontSize: 13 }}>
											{warmTestResult.success ? '✓' : '✗'}
										</Typography.Text>
									)}
									<Button type="primary" onClick={onWarmS3Save} loading={warmSaving} disabled={!warmBucket || !warmTestResult || !warmTestResult.success} style={{ marginLeft: 8 }}>Save</Button>
								</Col>
							</Row>
						</>
					)}

					{/* ========== Warm S3 Storage (S3 디스크 있을 때만 표시) ========== */}
					{s3Enabled && (
					<>
					<Divider style={{ margin: '1rem 0' }} />
					<Typography.Title style={{ margin: 0 }} level={4}>
						Warm S3 Storage{' '}
						<Tooltip title="S3 credentials for ClickHouse Warm tier (TTL MOVE). Data moves from Hot (local) to this S3 bucket.">
							<InfoCircleOutlined style={infoIconStyle} />
						</Tooltip>
					</Typography.Title>
					<Row gutter={[24, 16]} style={{ marginTop: 12 }}>
						<Col span={6}>
							<Typography.Text type="secondary" style={{ fontSize: 12 }}>Auth Mode</Typography.Text>
							<Select value={warmAuthMode} onChange={(v): void => setWarmAuthMode(v)} style={{ width: '100%', marginTop: 4 }}>
								<Select.Option value="static">Static Key</Select.Option>
								<Select.Option value="iam">IAM Role</Select.Option>
							</Select>
						</Col>
						<Col span={9}>
							<Typography.Text type="secondary" style={{ fontSize: 12 }}>Bucket</Typography.Text>
							<Input value={warmBucket} onChange={(e): void => setWarmBucket(e.target.value)} placeholder="warm-bucket" style={{ marginTop: 4 }} />
						</Col>
						<Col span={9}>
							<Typography.Text type="secondary" style={{ fontSize: 12 }}>Region</Typography.Text>
							<Input value={warmRegion} onChange={(e): void => setWarmRegion(e.target.value)} placeholder="ap-northeast-2" style={{ marginTop: 4 }} />
						</Col>
					</Row>
					{warmAuthMode === 'static' && (
						<Row gutter={[24, 16]} style={{ marginTop: 8 }}>
							<Col span={12}>
								<Typography.Text type="secondary" style={{ fontSize: 12 }}>Access Key ID</Typography.Text>
								<Input value={warmAccessKey} onChange={(e): void => setWarmAccessKey(e.target.value)} placeholder="AKIA..." style={{ marginTop: 4 }} />
							</Col>
							<Col span={12}>
								<Typography.Text type="secondary" style={{ fontSize: 12 }}>Secret Access Key</Typography.Text>
								<Input.Password value={warmSecretKey} onChange={(e): void => setWarmSecretKey(e.target.value)} style={{ marginTop: 4 }} />
							</Col>
						</Row>
					)}
					<Row justify="end" style={{ marginTop: 12 }}>
						<Col>
							<Button onClick={onWarmS3Test} loading={warmTesting} disabled={!warmBucket}>Test Connection</Button>
							{warmTestResult && (
								<Typography.Text type={warmTestResult.success ? 'success' : 'danger'} style={{ marginLeft: 6, fontSize: 13 }}>
									{warmTestResult.success ? '✓' : '✗'}
								</Typography.Text>
							)}
							<Button type="primary" onClick={onWarmS3Save} loading={warmSaving} disabled={!warmBucket || !warmTestResult || !warmTestResult.success} style={{ marginLeft: 8 }}>Save</Button>
						</Col>
					</Row>

					</>
					)}

					{/* ========== Cold Archive Section ========== */}
					{s3Enabled && (
						<>
							<Divider style={{ margin: '1rem 0' }} />
							<Typography.Title style={{ margin: 0 }} level={4}>
								Cold Archive (Glacier IR){' '}
								<Tooltip title="Long-term archival to AWS S3 Glacier Instant Retrieval. Data is backed up via clickhouse-backup cron before deletion.">
									<InfoCircleOutlined style={infoIconStyle} />
								</Tooltip>
							</Typography.Title>
							<Row align="middle" style={{ marginTop: 12 }}>
								<Col>
									<Switch
										checked={coldGlacierEnabled}
										onChange={(checked): void => setColdGlacierEnabled(checked)}
										checkedChildren="ON"
										unCheckedChildren="OFF"
									/>
									<Typography.Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
										Enable Cold Archive
									</Typography.Text>
								</Col>
							</Row>

							{coldGlacierEnabled && (
								<>
									{/* Cold S3 Storage */}
									<Row align="middle" style={{ marginTop: 8 }}>
										<Col>
											<Switch
												checked={!coldUseSameAsWarm}
												onChange={(checked): void => setColdUseSameAsWarm(!checked)}
												checkedChildren="ON"
												unCheckedChildren="OFF"
											/>
											<Typography.Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
												Use separate S3 bucket for Cold backup
											</Typography.Text>
										</Col>
									</Row>

									{!coldUseSameAsWarm && (
										<>
											<Row gutter={[24, 16]} style={{ marginTop: 8 }}>
												<Col span={6}>
													<Typography.Text type="secondary" style={{ fontSize: 12 }}>Auth Mode</Typography.Text>
													<Select value={coldAuthMode} onChange={(v): void => setColdAuthMode(v)} style={{ width: '100%', marginTop: 4 }}>
														<Select.Option value="static">Static Key</Select.Option>
														<Select.Option value="iam">IAM Role</Select.Option>
													</Select>
												</Col>
												<Col span={9}>
													<Typography.Text type="secondary" style={{ fontSize: 12 }}>Bucket</Typography.Text>
													<Input value={coldBucket} onChange={(e): void => setColdBucket(e.target.value)} placeholder="cold-archive-bucket" style={{ marginTop: 4 }} />
												</Col>
												<Col span={9}>
													<Typography.Text type="secondary" style={{ fontSize: 12 }}>Region</Typography.Text>
													<Input value={coldRegionS3} onChange={(e): void => setColdRegionS3(e.target.value)} placeholder="ap-northeast-2" style={{ marginTop: 4 }} />
												</Col>
											</Row>
											{coldAuthMode === 'static' && (
												<Row gutter={[24, 16]} style={{ marginTop: 8 }}>
													<Col span={12}>
														<Typography.Text type="secondary" style={{ fontSize: 12 }}>Access Key ID</Typography.Text>
														<Input value={coldAccessKey} onChange={(e): void => setColdAccessKey(e.target.value)} placeholder="AKIA..." style={{ marginTop: 4 }} />
													</Col>
													<Col span={12}>
														<Typography.Text type="secondary" style={{ fontSize: 12 }}>Secret Access Key</Typography.Text>
														<Input.Password value={coldSecretKey} onChange={(e): void => setColdSecretKey(e.target.value)} style={{ marginTop: 4 }} />
													</Col>
												</Row>
											)}
											<Row justify="end" style={{ marginTop: 12 }}>
												<Col>
													<Button onClick={onColdS3Test} loading={coldS3Testing} disabled={!coldBucket}>Test Connection</Button>
													{coldS3TestResult && (
														<Typography.Text type={coldS3TestResult.success ? 'success' : 'danger'} style={{ marginLeft: 6, fontSize: 13 }}>
															{coldS3TestResult.success ? '✓' : '✗'}
														</Typography.Text>
													)}
													<Button type="primary" onClick={onColdS3Save} loading={coldS3Saving} disabled={!coldBucket || !coldS3TestResult || !coldS3TestResult.success} style={{ marginLeft: 8 }}>Save</Button>
												</Col>
											</Row>
										</>
									)}

									<Divider style={{ margin: '1rem 0', opacity: 0.3 }} />

							<Row gutter={[24, 16]} style={{ marginTop: 16 }}>
								<Col span={12}>
									<div style={{ marginBottom: 8 }}>
										<Typography.Text type="secondary" style={{ fontSize: 12 }}>
											Retention{' '}
											<Tooltip title="How long archived data is retained in Glacier IR. Set 0 for unlimited retention.">
												<InfoCircleOutlined style={infoIconStyle} />
											</Tooltip>
										</Typography.Text>
									</div>
									<InputNumber
										min={0}
										value={coldRetentionDays}
										onChange={(val): void => setColdRetentionDays(val ?? 0)}
										disabled={!coldGlacierEnabled}
										style={{ width: 130 }}
										addonAfter="days"
										placeholder="0=∞"
									/>
									{coldRetentionDays === 0 && (
										<Typography.Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
											Unlimited
										</Typography.Text>
									)}
								</Col>
								<Col span={12}>
									<div style={{ marginBottom: 8 }}>
										<Typography.Text type="secondary" style={{ fontSize: 12 }}>
											Backup Frequency{' '}
											<Tooltip title="Backup runs daily via cron. This value is configured at install time and cannot be changed from the UI.">
												<InfoCircleOutlined style={infoIconStyle} />
											</Tooltip>
										</Typography.Text>
									</div>
									<InputNumber
										value={coldBackupFreqDays}
										disabled
										style={{ width: 130 }}
										addonAfter="days"
									/>
									<Typography.Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
										(daily)
									</Typography.Text>
								</Col>
							</Row>

							{!isCloudUserVal && (
								<div style={{ marginTop: 16, textAlign: 'right' }}>
									<Button
										type="primary"
										onClick={(): void => setColdArchiveModal(true)}
										disabled={isColdArchiveSaveDisabled || (!coldUseSameAsWarm && (!coldS3TestResult || !coldS3TestResult.success))}
									>
										Save Cold Archive
									</Button>
									<Modal
										title="Confirm Cold Archive Changes"
										focusTriggerAfterClose
										forceRender
										destroyOnClose
										closable
										onCancel={(): void => setColdArchiveModal(false)}
										onOk={onColdArchiveSave}
										centered
										open={coldArchiveModal}
										confirmLoading={coldArchiveLoading}
									>
										<Typography>
											Are you sure you want to update cold archive settings? This
											will affect the Glacier IR backup schedule.
										</Typography>
									</Modal>
								</div>
							)}
								</>
							)}
						</>
					)}
				</Card>

				{/* S3 Storage Settings cards removed — Warm S3 is in Data Lifecycle card, Cold S3 is in Cold Archive card */}

				{isCloudUserVal && <GeneralSettingsCloud />}
			</Col>
		</>
	);
}

interface GeneralSettingsProps {
	getAvailableDiskPayload: GetDisksPayload;
	metricsTtlValuesPayload: GetRetentionPeriodMetricsPayload;
	tracesTtlValuesPayload: GetRetentionPeriodTracesPayload;
	logsTtlValuesPayload: GetRetentionPeriodLogsPayload;
	metricsTtlValuesRefetch: UseQueryResult<
		ErrorResponse | SuccessResponse<GetRetentionPeriodMetricsPayload>
	>['refetch'];
	tracesTtlValuesRefetch: UseQueryResult<
		ErrorResponse | SuccessResponse<GetRetentionPeriodTracesPayload>
	>['refetch'];
	logsTtlValuesRefetch: UseQueryResult<
		ErrorResponseV2 | SuccessResponseV2<GetRetentionPeriodLogsPayload>
	>['refetch'];
}

export default GeneralSettings;
